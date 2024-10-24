//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package natural

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	// APIs
	CP_URL = "https://api.cloud.couchbase.com"
	// for dev:
	// CP_URL = "https://api.dev.nonprod-project-avengers.com"
	_SESSIONS_API = CP_URL + "/sessions"
)

func getCompletionsApi(orgid string) string {

	return fmt.Sprintf("%v/v2/organizations/%v/integrations/iq/openai/chat/completions", CP_URL, orgid)
}

const _CACHE_LIMIT = 65536
const MAX_KEYSPACES = 4

const (
	// Models
	GPT4o_2024_05_13 = "gpt-4o-2024-05-13"
)

const (
	maxconcurrency = 4
	maxWaiters     = 16
	WaitTimeout    = 20 * time.Second
)

type naturalReqThrottler struct {
	gate       chan bool
	waiters    int32
	maxwaiters int32
}

var nlreqThrottler naturalReqThrottler

var cacheEntryCreation sync.Mutex

func (this *naturalReqThrottler) getWaiter() errors.Error {
	if atomic.AddInt32(&this.waiters, 1) >= this.maxwaiters {
		atomic.AddInt32(&this.waiters, -1)
		return errors.NewNaturalLanguageRequestError(errors.E_NL_TOO_MANY_WAITERS)
	}
	return nil
}

func (this *naturalReqThrottler) releaseWaiter() {
	atomic.AddInt32(&this.waiters, -1)
}

func (this *naturalReqThrottler) nlgate() chan bool {
	return this.gate
}

func init() {

	nlreqThrottler = naturalReqThrottler{
		gate:       make(chan bool, maxconcurrency),
		maxwaiters: maxWaiters,
	}

	for i := 0; i < maxconcurrency; i++ {
		nlreqThrottler.nlgate() <- true
	}
}

type NaturalContext interface {
	datastore.Context
	datastore.QueryContext
}

var jwtCache *util.GenCache

func init() {
	jwtCache = util.NewGenCache(_CACHE_LIMIT)
}

type jwtCacheEntry struct {
	sync.RWMutex
	token     string
	expiresAt time.Time
}

type jwtResponse struct {
	JWT  string `json:"jwt"`
	User struct {
		ExpiresAt string `json:"expiresAt"`
	} `json:"user"`
}

func getJWTFromSessionsApi(nlCred string, refresh bool) (string, errors.Error) {
	var entry *jwtCacheEntry
	if !refresh {
		cacheEntry := jwtCache.Get(nlCred, nil)
		if cacheEntry != nil {
			entry = cacheEntry.(*jwtCacheEntry)
			entry.RLock()
			if time.Now().Before(entry.expiresAt) {
				jwt := entry.token
				entry.RUnlock()
				return jwt, nil
			}
			entry.RUnlock()

			entry.Lock()
			defer entry.Unlock()

			// Someone else has refreshed the entry already
			if time.Now().Before(entry.expiresAt) {
				return entry.token, nil
			}
		} else {
			cacheEntryCreation.Lock()
			defer cacheEntryCreation.Unlock()

			// already created before us
			cacheEntry := jwtCache.Get(nlCred, nil)
			if cacheEntry != nil {
				entry = cacheEntry.(*jwtCacheEntry)
				entry.RLock()
				if time.Now().Before(entry.expiresAt) {
					jwt := entry.token
					entry.RUnlock()
					return jwt, nil
				}

				entry.RUnlock()

				entry.Lock()
				defer entry.Unlock()

				// Someone else has refreshed the entry already
				if time.Now().Before(entry.expiresAt) {
					return entry.token, nil
				}
			} else {
				entry = &jwtCacheEntry{
					expiresAt: time.Time{},
				}
			}
		}
	} else {
		entry = &jwtCacheEntry{
			expiresAt: time.Time{},
		}
		cacheEntryCreation.Lock()
		defer cacheEntryCreation.Unlock()
	}

	reqJwt, err := http.NewRequest("POST", _SESSIONS_API, nil)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CREATE_SESSIONS_REQ, _SESSIONS_API, err)
	}

	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(nlCred))
	reqJwt.Header.Set("Authorization", "Basic "+encodedCredentials)
	client := http.Client{}

	resp, err := client.Do(reqJwt)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SEND_SESSIONS_REQ, _SESSIONS_API, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_AUTH)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_RESP_READ, _SESSIONS_API, err)
	}

	var result jwtResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_RESP_UNMARSHAL, _SESSIONS_API, err)
	}

	expTime, err := time.Parse(time.RFC3339Nano, result.User.ExpiresAt)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_PARSE_EXPIRE_TIME,
			result.User.ExpiresAt, err)
	}

	entry.token = "Bearer " + result.JWT
	entry.expiresAt = expTime

	jwtCache.Add(entry, nlCred, nil)

	return entry.token, nil
}

// Prompt
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionSettings struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
	// Optional fields
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Seed        int     `json:"seed,omitempty"`
}

type prompt struct {
	InitMessages       []message          `json:"initMessages"`
	CompletionSettings completionSettings `json:"completionSettings"`
	Messages           []message          `json:"messages"`
}

func newPrompt(keyspaceInfo map[string]interface{}, naturalPrompt string) (*prompt, errors.Error) {
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a Couchbase Capella expert. Your task is to create valid queries to retrieve" +
					" or create data based on the provided Information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		},
		CompletionSettings: completionSettings{
			Model:       GPT4o_2024_05_13,
			Temperature: 0,
			Seed:        1,
			Stream:      false,
		},
	}

	var userMessage string
	var userMessageBuf strings.Builder

	binKeyspacesInfo, err := json.Marshal(keyspaceInfo)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
	}
	userMessageBuf.WriteString("Information about keyspaces:\n\n")
	userMessageBuf.WriteString(string(binKeyspacesInfo))
	userMessageBuf.WriteString("\n\nPrompt: \"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\"\n\nBased on the above Information, write valid SQL++ only and with no explanation." +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters." +
		"\n\nReturn only a single SQL++ statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a query, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")
	userMessage = userMessageBuf.String()
	rv.Messages = []message{
		message{
			Role:    "user",
			Content: userMessage,
		},
	}

	return rv, nil
}

func doChatCompletionsReq(prompt *prompt, nlOrganizationId string, jwt string, nlCred string) (string, errors.Error) {
	type ResultMessage struct {
		Content string `json:"content"`
	}

	type Choice struct {
		Message ResultMessage `json:"message"`
	}
	type ChatCompletionResponse struct {
		Choices []Choice `json:"choices"`
	}

	chatCompletionsUrl := getCompletionsApi(nlOrganizationId)

	client := http.Client{}
	payload, perr := json.Marshal(prompt)
	if perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, perr)
	}
	chatReq, perr := http.NewRequest("POST", chatCompletionsUrl, bytes.NewBuffer(payload))
	if perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CREATE_CHATCOMPLETIONS_REQ, chatCompletionsUrl)
	}

	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", jwt)
	chatRes, perr := client.Do(chatReq)
	if perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SEND_CHATCOMPLETIONS_REQ, chatCompletionsUrl, perr)
	}

	if statusCode := chatRes.StatusCode; statusCode != http.StatusOK {

		if statusCode == http.StatusNotFound {
			chatRes.Body.Close()
			return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ORG_NOT_FOUND, nlOrganizationId)
		} else if statusCode == http.StatusUnauthorized {

			// JWT refreshed by an external client
			// unauthorized, try refreshing jwt

			// possible ways a request was unauthorized
			// 1. user doesn't access to the organization
			// 2. JWT refreshed by an external client

			//  no way to know which is the cause, so we'll retry until we give up
			chatRes.Body.Close()
			backoff := 1 * time.Second
			maxRetries := 5
			for retries := 0; retries < maxRetries; retries++ {
				time.Sleep(backoff)

				var err errors.Error
				jwt, err = getJWTFromSessionsApi(nlCred, true)
				if err != nil {
					return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED,
						chatRes.StatusCode, err)
				}

				chatReq, perr = http.NewRequest("POST", chatCompletionsUrl, bytes.NewBuffer(payload))
				if perr != nil {
					return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CREATE_CHATCOMPLETIONS_REQ,
						chatCompletionsUrl)
				}

				chatReq.Header.Set("Content-Type", "application/json")
				chatReq.Header.Set("Authorization", jwt)
				chatRes, perr = client.Do(chatReq)
				if perr != nil {
					return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SEND_CHATCOMPLETIONS_REQ,
						chatCompletionsUrl, perr)
				}

				if chatRes.StatusCode == http.StatusOK {
					break
				}

				chatRes.Body.Close()
				backoff *= 2
			}
			if chatRes.StatusCode == http.StatusUnauthorized {
				return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ORG_UNAUTH)
			} else if chatRes.StatusCode != http.StatusOK {
				return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED, chatRes.StatusCode)
			}
		} else {
			return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED, chatRes.StatusCode)
		}
	}

	defer chatRes.Body.Close()
	reader := bufio.NewReader(chatRes.Body)
	var llmResp bytes.Buffer
	for {
		b, perr := reader.ReadByte()
		if perr != nil {
			if perr != io.EOF {
				return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_READ_RESP_STREAM,
					chatCompletionsUrl, perr)
			}
			break
		}
		llmResp.WriteByte(b)
	}
	out := llmResp.Bytes()
	chatComplRes := ChatCompletionResponse{}
	perr = json.Unmarshal(out, &chatComplRes)
	if perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, perr)
	}

	content := chatComplRes.Choices[0].Message.Content

	if n := strings.Index(content, "#ERR"); n != -1 {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
			fmt.Errorf("%s", strings.TrimRight(content[n+6:], "\n `")))
	}

	sqlstmt := strings.TrimPrefix(content, "```sql\n")
	sqlstmt = strings.TrimSuffix(sqlstmt, "\n```")
	if end := len(sqlstmt) - 1; sqlstmt[end] == ';' {
		sqlstmt = sqlstmt[:end]
	}

	sqlstmt = strings.TrimSpace(sqlstmt)
	return sqlstmt, nil
}

func collectSchemaForPromptFromInfer(schema map[string]string, inferSchema value.Value) map[string]string {
	if v, ok := inferSchema.Index(0); ok {
		if prop, ok := v.Field("properties"); ok {
			schemaFieldNames := []string{}
			schemaFieldNames = prop.FieldNames(schemaFieldNames)
			for _, fieldname := range schemaFieldNames {
				if fieldname == "~meta" {
					continue
				}

				if fieldSpecific, ok := prop.Field(fieldname); ok {
					if typeinfo, ok := fieldSpecific.Field("type"); ok {
						if typeinfo.Type() == value.ARRAY {
							var typestring strings.Builder
							var typestr string
							if typestrslice, ok := typeinfo.Actual().([]interface{}); ok {
								if typestr, ok = typestrslice[0].(string); ok {
									typestring.WriteString(typestr)
									typestring.WriteRune(' ')
								}
								for _, s := range typestrslice[1:] {
									if typestr, ok = s.(string); ok {
										typestring.WriteString("or ")
										typestring.WriteString(typestr)
									}
								}
							}
							schema[fieldname] = typestring.String()
						} else {
							schema[fieldname] = typeinfo.String()
						}
					}
				}
			}
		}
	}

	return schema
}

func inferSchema(schema map[string]string, p *algebra.Path, context NaturalContext) (map[string]string, errors.Error) {

	keyspace, err := datastore.GetKeyspace(p.Parts()...)
	if err != nil {
		return nil, err
	}

	conn := datastore.NewValueConnection(context)
	infer, err := context.Datastore().Inferencer(datastore.INF_DEFAULT)
	if err != nil {
		return nil, err
	}
	infer.InferKeyspace(context, keyspace, nil, conn)

	inferSchema, ok := <-conn.ValueChannel()

	if inferSchema != nil && ok {
		schema = collectSchemaForPromptFromInfer(schema, inferSchema)
	}

	return schema, nil
}

func keyspacesInfoForPrompt(keyspaceInfo map[string]interface{}, elems []*algebra.Path,
	context NaturalContext) (map[string]interface{}, errors.Error) {

	var err errors.Error
	for _, p := range elems {
		schema := map[string]string{}
		schema, err = inferSchema(schema, p, context)

		if err != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_INFER, p.ProtectedString(), err)
		}
		info := map[string]interface{}{}
		info["schema"] = schema
		fullpath := p.ProtectedString()
		info["fullpath"] = fullpath[strings.Index(fullpath, ":"):]

		keyspaceInfo[p.Keyspace()] = info
	}

	return keyspaceInfo, nil
}

func ProcessRequest(nlCred, nlOrgId, nlquery string, elems []*algebra.Path,
	context NaturalContext, record func(execution.Phases, time.Duration)) (algebra.Statement, string, errors.Error) {

	if err := nlreqThrottler.getWaiter(); err != nil {
		return nil, "", err
	}
	defer nlreqThrottler.releaseWaiter()

	waitTime := util.Now()
	select {
	case <-nlreqThrottler.nlgate():
		record(execution.NLWAIT, util.Since(waitTime))
		defer func() {
			nlreqThrottler.nlgate() <- true
		}()
	case <-time.After(WaitTimeout):
		return nil, "", errors.NewNaturalLanguageRequestError(errors.E_NL_TIMEOUT)
	}

	getJwt := util.Now()
	jwt, err := getJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return nil, "", err
	}

	keyspaceInfo := make(map[string]interface{}, len(elems))
	inferschema := util.Now()
	keyspaceInfo, err = keyspacesInfoForPrompt(keyspaceInfo, elems, context)
	if err != nil {
		return nil, "", err
	}
	record(execution.INFERSCHEMA, util.Since(inferschema))

	prompt, err := newPrompt(keyspaceInfo, nlquery)
	if err != nil {
		return nil, "", err
	}

	chatcompletionreq := util.Now()
	sqlstmt, err := doChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return nil, "", err
	}

	var parseErr error
	parse := util.Now()
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(sqlstmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))
	if parseErr != nil {
		return nil, "", errors.NewNaturalLanguageRequestError(errors.E_NL_PARSE_GENERATED_STMT, sqlstmt, parseErr)
	}

	return nlAlgebraStmt, sqlstmt, nil
}
