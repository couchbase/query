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
	"regexp"
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

const (
	// Models
	GPT4o_2024_05_13 = "gpt-4o-2024-05-13"
)

var _NLCONTEXT_REGEX *regexp.Regexp

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
	bucketOrScopeIdentifier := `[^\x60,\.\[\]]+`
	bucketscopecollection := fmt.Sprintf("(%s|\\`%s\\`)", bucketOrScopeIdentifier,
		bucketOrScopeIdentifier)

	coll1 := bucketscopecollection
	coll2 := fmt.Sprintf("%s\\,%s", bucketscopecollection, bucketscopecollection)
	coll3 := fmt.Sprintf("%s\\,%s\\,%s", bucketscopecollection, bucketscopecollection,
		bucketscopecollection)
	coll4 := fmt.Sprintf("%s\\,%s\\,%s\\,%s", bucketscopecollection, bucketscopecollection,
		bucketscopecollection, bucketscopecollection)

	collList := fmt.Sprintf("\\[(%s|%s|%s|%s)\\]", coll1, coll2, coll3, coll4)
	optcollList := fmt.Sprintf("(\\.%s)?", collList)

	namespace := bucketscopecollection
	optnamespace := fmt.Sprintf("(%s\\:)?", namespace)

	pattern := fmt.Sprintf("^%s%s\\.%s%s$", optnamespace, bucketscopecollection,
		bucketscopecollection, optcollList)
	_NLCONTEXT_REGEX = regexp.MustCompile(pattern)

	nlreqThrottler = naturalReqThrottler{
		gate:       make(chan bool, maxconcurrency),
		maxwaiters: maxWaiters,
	}

	for i := 0; i < maxconcurrency; i++ {
		nlreqThrottler.nlgate() <- true
	}
}

// Accept
// bucket.scope
// bucket.scope.[<upto_4_collections>]
// namespace:bucket.scope
// namespace:bucket.scope.[<upto_4_collections>]
func ValidateNaturalContext(naturalContext string) errors.Error {

	if !_NLCONTEXT_REGEX.MatchString(naturalContext) {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT)
	}
	return nil
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
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_AUTH, nlCred)
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

func ParseNaturalLanguageContext(naturalLanguageContext string) (namespace, bucket, scope string,
	inferKeyspaceNames []string) {
	matches := _NLCONTEXT_REGEX.FindStringSubmatch(naturalLanguageContext)

	if len(matches) > 0 {
		namespace = matches[2]
		bucket = matches[3]
		scope = matches[4]

		addUnique := func(slice []string, strs ...string) []string {
			uniqueMap := make(map[string]bool)
			// Add new strings if they are not already in the slice
			for _, s := range strs {
				if !uniqueMap[s] && s != "" {
					slice = append(slice, s)
					uniqueMap[s] = true
				}
			}

			return slice
		}
		if matches[13] != "" {
			coll1name := matches[13]
			coll2name := matches[14]
			coll3name := matches[15]
			coll4name := matches[16]
			inferKeyspaceNames = addUnique(inferKeyspaceNames, coll1name, coll2name, coll3name, coll4name)
		} else if matches[10] != "" {
			coll1name := matches[10]
			coll2name := matches[11]
			coll3name := matches[12]
			inferKeyspaceNames = addUnique(inferKeyspaceNames, coll1name, coll2name, coll3name)
		} else if matches[8] != "" {
			coll1name := matches[8]
			coll2name := matches[9]
			inferKeyspaceNames = addUnique(inferKeyspaceNames, coll1name, coll2name)
		} else if matches[7] != "" {
			coll1name := matches[7]
			inferKeyspaceNames = addUnique(inferKeyspaceNames, coll1name)
		}
	}

	return
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

func newPrompt(collSchema map[string]map[string]string, collnames []string, bucket, scope string,
	naturalPrompt string) (*prompt, errors.Error) {
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
	if len(collSchema) == 0 {

		var collnamesString string
		b := strings.Builder{}
		end := len(collnames) - 1
		for _, c := range collnames[:end] {
			b.WriteString("\"")
			b.WriteString(c)
			b.WriteString("\"")
		}
		b.WriteString("\"")
		b.WriteString(collnames[end])
		b.WriteString("\"")

		collnamesString = b.String()

		userMessageBuf.WriteString("Your minimum task is to CREATE a COLLECTION. Consider the below Information")
		userMessageBuf.WriteString(" to determine whether this task can be accomplished:\n\nInformation:")
		userMessageBuf.WriteString("\n\n- Prompt: \"")
		userMessageBuf.WriteString(naturalPrompt)
		userMessageBuf.WriteString("\"")
		userMessageBuf.WriteString("\n\n- Bucket: ")
		userMessageBuf.WriteString(bucket)
		userMessageBuf.WriteString("\n\n- Scope: ")
		userMessageBuf.WriteString(scope)
		userMessageBuf.WriteString("\n\n- Taken collection names: ")
		userMessageBuf.WriteString(collnamesString)
		userMessageBuf.WriteString("\n\nHaving considered the above Information, follow these steps:")
		userMessageBuf.WriteString("\n\n1. Ask yourself: is the user prompt relevant to the task at hand? If not,")
		userMessageBuf.WriteString(" first write \"#ERR:\" and politely explain why not.")
		userMessageBuf.WriteString("\n\n2. If the task can't be accomplished, say \"#ERR:\" and politely explain why not.")
		userMessageBuf.WriteString("\n\n3. Output valid SQL++ only and with no explanation.")

		userMessage = userMessageBuf.String()
	} else if len(collSchema) == 1 {
		for coll, schema := range collSchema {
			binSchemaData, err := json.Marshal(schema)
			if err != nil {
				return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
			}
			schemaData := string(binSchemaData)

			userMessageBuf.WriteString("Information:\n\nCollection: ")
			userMessageBuf.WriteString(coll)
			userMessageBuf.WriteString("\n\nCollection's schema: ")
			userMessageBuf.WriteString(schemaData)
			userMessageBuf.WriteString("\n\nPrompt: \"")
			userMessageBuf.WriteString(naturalPrompt)
			userMessageBuf.WriteString("\"\n\nThe query context is set.")
			userMessageBuf.WriteString(" \n\nBased on the above Information, write valid SQL++ only and with no explanation.")
			userMessageBuf.WriteString(" For retrieval, use aliases.")
			userMessageBuf.WriteString("\n\nIf you're sure the Prompt can't be used to generate a query,")
			userMessageBuf.WriteString(" first say \"#ERR:\" and then explain why not.")

			userMessage = userMessageBuf.String()
			break
		}

	} else if len(collSchema) > 1 {
		binSchemaData, err := json.Marshal(collSchema)
		if err != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
		}
		schemaData := string(binSchemaData)

		userMessageBuf.WriteString("Information:\n\nPrompt: \"")
		userMessageBuf.WriteString(naturalPrompt)
		userMessageBuf.WriteString("\"\n\nAvailable collections and their schemas:\n\n")
		userMessageBuf.WriteString(schemaData)
		userMessageBuf.WriteString("\"\n\nThe query context is set.")
		userMessageBuf.WriteString("\n\nBased on the above Information, write valid SQL++ only and with no explanation.")
		userMessageBuf.WriteString(" For retrieval, use aliases.")
		userMessageBuf.WriteString("\n\nIf you're sure the Prompt can't be used to generate a query, first say")
		userMessageBuf.WriteString("\"#ERR:\" and then explain why not.")

		userMessage = userMessageBuf.String()
	}

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

	if content[:4] == "#ERR" {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, content[6:])
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

func inferSchema(schema map[string]string, namespace, bucket, scope, coll string,
	context NaturalContext) (map[string]string, errors.Error) {

	keyspace, err := datastore.GetKeyspace(namespace, bucket, scope, coll)
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

func inferSchemaForKeyspaces(keyspaceSchemas map[string]map[string]string, namespace, bucket, scope string, keyspaces []string,
	context NaturalContext) (map[string]map[string]string, errors.Error) {

	for _, keyspace := range keyspaces {
		schema := map[string]string{}
		schema, err := inferSchema(schema, namespace, bucket, scope, keyspace, context)
		if err != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_INFER, keyspace, err)
		}
		keyspaceSchemas[keyspace] = schema
	}

	return keyspaceSchemas, nil
}

func getCollNames(namespace, bucket, scope string) ([]string, errors.Error) {
	if namespace == "" {
		namespace = "default"
	}
	ns, err := datastore.GetDatastore().NamespaceById(namespace)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_COLLNAMES,
			namespace+":"+"`"+bucket+"`.`"+scope+"`", err)
	}

	b, err := ns.BucketByName(bucket)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_COLLNAMES,
			namespace+":"+"`"+bucket+"`.`"+scope+"`", err)
	}

	s, err := b.ScopeByName(scope)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_COLLNAMES,
			namespace+":"+"`"+bucket+"`.`"+scope+"`", err)
	}

	collnames, err := s.KeyspaceNames()
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_COLLNAMES,
			namespace+":"+"`"+bucket+"`.`"+scope+"`", err)
	}

	return collnames, nil
}

func ProcessRequest(nlCred, nlOrgId, nlquery string, namespace, bucket, scope string, inferKeyspaceNames []string,
	context NaturalContext, record func(execution.Phases, time.Duration)) (algebra.Statement, string, string, errors.Error) {

	if err := nlreqThrottler.getWaiter(); err != nil {
		return nil, "", "", err
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
		return nil, "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_TIMEOUT)
	}

	getJwt := util.Now()
	jwt, err := getJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return nil, "", "", err
	}

	collSchema := make(map[string]map[string]string, len(inferKeyspaceNames))
	var collnames []string

	inferschema := util.Now()
	collSchema, err = inferSchemaForKeyspaces(collSchema, namespace, bucket, scope, inferKeyspaceNames,
		context)

	if len(collSchema) == 0 {
		collnames, err = getCollNames(namespace, bucket, scope)
	}
	record(execution.INFERSCHEMA, util.Since(inferschema))

	if err != nil {
		return nil, "", "", err
	}

	prompt, err := newPrompt(collSchema, collnames, bucket, scope, nlquery)
	if err != nil {
		return nil, "", "", err
	}

	chatcompletionreq := util.Now()
	sqlstmt, err := doChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return nil, "", "", err
	}

	queryContext := bucket + "." + scope
	if namespace != "" {
		queryContext = namespace + ":" + queryContext
	}

	var parseErr error
	parse := util.Now()
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(sqlstmt, namespace, queryContext)
	record(execution.NLPARSE, util.Since(parse))
	if parseErr != nil {
		return nil, "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_PARSE_GENERATED_STMT, sqlstmt, parseErr)
	}

	return nlAlgebraStmt, sqlstmt, queryContext, nil
}
