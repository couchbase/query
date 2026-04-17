//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package natural

import (
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
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/primitives/couchbase"
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
const _COMPLETIONS_REQ_BACKOFF_INIT = 1 * time.Second
const _COMPLETIONS_REQ_RETRY = 5

var _CHAT_LIMIT int

const (
	// Models
	GPT4o_2024_05_13 = "gpt-4o-2024-05-13"
)

const (
	maxconcurrency       = 4
	maxWaiters           = 16
	waitTimeout          = 20 * time.Second
	maxCorrectionRetries = 4
)

var naturalchatHistory *util.GenCache

type naturalReqThrottler struct {
	gate       chan bool
	waiters    int32
	maxwaiters int32
}

type naturalOutput int

const (
	SQL naturalOutput = iota
	JSUDF
	FTSSQL
	UNDEFINED_NATURAL_OUTPUT
)

func NewNaturalOutput(s string) naturalOutput {
	switch strings.ToUpper(s) {
	case "SQL":
		return SQL
	case "JSUDF":
		return JSUDF
	case "FTSSQL":
		return FTSSQL
	default:
		return UNDEFINED_NATURAL_OUTPUT
	}
}

func (n naturalOutput) String() string {
	var s string
	switch n {
	case SQL:
		s = "SQL"
	case JSUDF:
		s = "JSUDF"
	case FTSSQL:
		s = "FTSSQL"
	default:
		s = "UNDEFINED_NATURAL_OUTPUT"
	}
	return s
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

	_CHAT_LIMIT = util.NumCPU() * 2

	naturalchatHistory = util.NewGenCache(_CHAT_LIMIT)
}

type ChatEntry struct {
	Id        string
	prompt    *prompt
	Keyspaces []*algebra.Path
	Removed   bool
	User      string
	Paused    bool
	Summary   string
	sync.Mutex
}

func IsChatCacheFull() bool {
	return naturalchatHistory.Size() >= _CHAT_LIMIT
}

func AddConversation(ce *ChatEntry, id string) {
	naturalchatHistory.Add(ce, id, nil)
}

func GetConversation(id string) interface{} {
	return naturalchatHistory.Get(id, nil)
}

func DeleteConversation(id string) {
	naturalchatHistory.Delete(id, nil)
}

func ForEachConversation(nonBlocking func(chatId string, entry *ChatEntry) bool, blocking func() bool) {
	dummyF := func(chatId string, entry interface{}) bool {
		ce := entry.(*ChatEntry)
		return nonBlocking(chatId, ce)
	}
	naturalchatHistory.ForEach(dummyF, blocking)
}

func CountCoversations() int {
	return naturalchatHistory.Size()
}

func FormatChatEntry(ce *ChatEntry) map[string]interface{} {
	item := map[string]interface{}{}

	if ceId := ce.Id; ceId != "" {
		item["chatId"] = ceId
	}
	if cekeyspaces := ce.Keyspaces; len(cekeyspaces) > 0 {
		keyspaces := make([]interface{}, len(ce.Keyspaces))
		for i, p := range ce.Keyspaces {
			keyspaces[i] = p.ProtectedString()
		}
		item["keyspaces"] = keyspaces
	}
	if pmpt := ce.prompt; pmpt != nil {
		item["prompt"] = value.NewMarshalledValue(pmpt)
	}
	if user := ce.User; user != "" {
		item["user"] = user
	}
	if summary := ce.Summary; summary != "" {
		item["summary"] = summary
	}
	return item
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
	Size               int                `json:"size"`
}

const _INIT_SIZE = 250
const _MAX_PROMPT_SIZE = util.MiB

func newSQLPrompt(keyspaceInfo map[string]interface{}, naturalPrompt, summary string, forfts bool) (*prompt, errors.Error) {
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
		Size: _INIT_SIZE,
	}

	var userMessage string
	var userMessageBuf strings.Builder

	binKeyspacesInfo, err := json.Marshal(keyspaceInfo)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
	}
	if summary != "" {
		userMessageBuf.WriteString("Summary of the conversation so far: " + summary + "\n\n")
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
		"\nMake use of RAW keyword when you require a non-object result, for example when comparing a field with a subquery's result set.")
	if forfts {
		userMessageBuf.WriteString("\n\nAlways add the USE Clause in the query to use the FTS index." +
			"\n\nFor example, SELECT a.*, ap.* FROM `travel-sample`.`inventory`.`airline` AS a USE INDEX " +
			"(USING FTS) JOIN `travel-sample`.`inventory`.`airport` AS ap USE INDEX (USING FTS)" +
			" ON a.country = ap.country WHERE a.country = \"United Kingdom\"" +
			"\n\nIn other words, always use USE INDEX (USING FTS) in the query.")
	}
	userMessageBuf.WriteString("\n\nReturn only a single SQL++ statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a query, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")
	rv.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	rv.Messages = []message{
		message{
			Role:    "user",
			Content: userMessage,
		},
	}

	return rv, nil
}

func newJSUDFPrompt(keyspaceInfo map[string]interface{}, naturalPrompt, summary string) (*prompt, errors.Error) {
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a couchbase Capella expert. Your task is to write valid Javascript user defined functions" +
					" based on the provided information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		}, CompletionSettings: completionSettings{
			Model:       GPT4o_2024_05_13,
			Temperature: 0,
			Seed:        1,
			Stream:      false,
		},
		Size: _INIT_SIZE,
	}

	var userMessage string
	var userMessageBuf strings.Builder

	binKeyspacesInfo, err := json.Marshal(keyspaceInfo)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
	}
	if summary != "" {
		userMessageBuf.WriteString("Summary of the conversation so far: " + summary + "\n\n")
	}
	userMessageBuf.WriteString("Information about keyspaces:\n\n")
	userMessageBuf.WriteString(string(binKeyspacesInfo))
	userMessageBuf.WriteString("\n\nPrompt: \"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\"\n\nBased on the above Information, write a valid Javascript User Defined Function with" +
		" no explanation that implements the request in the Prompt." +
		"\n\nComment the code liberally to explain what each piece does and why it's written that way." +
		"\nAlways comment using multiline comment syntax,i.e /* ... */ as you have construct a SQL++ managed user defined function" +
		" in a single line." +
		"\n\nPrompt may request for something trivial such as string manipulation, mathematical calculations, etc." +
		" Which are not related to the information provided but must always instruct to generate a function." +
		" \n\n Your task is to return CREATE FUNCTION statement that follows construct for SQL++ managed user defined function." +
		"As Capella does not currently support a way to create or manage an external library" +
		"\n\nExamples:" +
		"\n\nExample1) shows an example for a request that doesn't use the information provided, example prompt:" +
		"add 2 numbers. Statement to create a function for the request would be: CREATE FUNCTION add(a,b) LANGUAGE JAVASCRIPT AS" +
		" 'function add(a,b) { return(a+b);}'" +
		"\n\nExample2) shows an example for a request that uses the information provided, example prompt:" +
		"select airlines given country as an argument. Statement to create a function for the request would be: CREATE FUNCTION" +
		" selectAirline(country) LANGUAGE JAVASCRIPT AS 'function selectAirline(country)" +
		" {var q = SELECT name as airline_name, callsign as airline_callsign FROM `travel-sample`.`inventory`.`airline` " +
		"WHERE country = $country; var res = []; for (const doc of q) { var airline = {}; airline.name = doc.airline_name;" +
		"airline.callsign = doc.airline_callsign; res.push(airline);} return res;}" +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters." +
		"\n\nReturn only a single CREATE FUNCTION statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a function, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")
	rv.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	rv.Messages = []message{
		message{
			Role:    "user",
			Content: userMessage,
		},
	}
	return rv, nil
}

func newSQLIterativePrompt(chat *prompt, naturalPrompt string, forfts bool) *prompt {
	var userMessage string
	var userMessageBuf strings.Builder

	userMessageBuf.WriteString("Your goal is to iterate on the previouly generated query by modifying it's code,")
	userMessageBuf.WriteString(" based on this prompt:")
	userMessageBuf.WriteString("\"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\".")
	userMessageBuf.WriteString("Respond only with code and no explanation." +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters.")
	if forfts {
		userMessageBuf.WriteString("\n\nAlways add the USE Clause in the query to use the FTS index." +
			"\n\nFor example, SELECT a.*, ap.* FROM `travel-sample`.`inventory`.`airline` AS a USE INDEX " +
			"(USING FTS) JOIN `travel-sample`.`inventory`.`airport` AS ap USE INDEX (USING FTS)" +
			" ON a.country = ap.country WHERE a.country = \"United Kingdom\"" +
			"\n\nIn other words, always use USE INDEX (USING FTS) in the query.")
	}
	userMessageBuf.WriteString("\n\nIf the previous message was a CREATE FUNCTION statement you don't have to repeat a CREATE FUNCTION." +
		"\nReturn only a single SQL++ statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a query, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")

	chat.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	chat.Messages = append(chat.Messages, message{
		Content: userMessage,
		Role:    "user",
	})

	return chat
}

func newJSUDFIterativePrompt(chat *prompt, naturalPrompt string) *prompt {
	var userMessage string
	var userMessageBuf strings.Builder

	userMessageBuf.WriteString("Your goal is to iterate on the previouly generated query by modifying it's code,")
	userMessageBuf.WriteString(" based on this prompt:")
	userMessageBuf.WriteString("\"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\".")
	userMessageBuf.WriteString("\"\n\nBased on the above Information, write a valid Javascript User Defined Function with" +
		" no explanation that implements the request in the Prompt." +
		"\n\nComment the code liberally to explain what each piece does and why it's written that way." +
		"\nAlways comment using multiline comment syntax,i.e /* ... */ as you have construct a SQL++ managed user defined function" +
		" in a single line." +
		"\n\nPrompt may request for something trivial such as string manipulation, mathematical calculations, etc." +
		" Which are not related to the information provided but must always instruct to generate a function." +
		" \n\n Your task is to return CREATE FUNCTION statement that follows construct for SQL++ managed user defined function." +
		"As Capella does not currently support a way to create or manage an external library" +
		"\n\nExamples:" +
		"\n\nExample1) shows an example for a request that doesn't use the information provided, example prompt:" +
		"add 2 numbers. Statement to create a function for the request would be: CREATE FUNCTION add(a,b) LANGUAGE JAVASCRIPT AS" +
		" 'function add(a,b) { return(a+b);}'" +
		"\n\nExample2) shows an example for a request that uses the information provided, example prompt:" +
		"select airlines given country as an argument. Statement to create a function for the request would be: CREATE FUNCTION" +
		" selectAirline(country) LANGUAGE JAVASCRIPT AS 'function selectAirline(country)" +
		" {var q = SELECT name as airline_name, callsign as airline_callsign FROM `travel-sample`.`inventory`.`airline` " +
		"WHERE country = $country; var res = []; for (const doc of q) { var airline = {}; airline.name = doc.airline_name;" +
		"airline.callsign = doc.airline_callsign; res.push(airline);} return res;}" +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters." +
		"\n\nIf the previous message was not a CREATE FUNCTION statement, use the previous messages to for a CREATE FUNCTION statement." +
		"\nReturn only a single CREATE FUNCTION statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a function, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")

	chat.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	chat.Messages = append(chat.Messages, message{
		Content: userMessage,
		Role:    "user",
	})

	return chat
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

			// possible ways a request was unauthorized
			// 1. user doesn't have access to the organization
			// 2. JWT was refreshed by an external client

			//  no way to know which one is the cause, so we'll retry until we give up
			chatRes.Body.Close()
			backoff := _COMPLETIONS_REQ_BACKOFF_INIT
			for retries := 0; retries < _COMPLETIONS_REQ_RETRY; retries++ {
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
	decoder := json.NewDecoder(chatRes.Body)
	chatComplRes := ChatCompletionResponse{}
	if perr = decoder.Decode(&chatComplRes); perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, perr)
	}

	return chatComplRes.Choices[0].Message.Content, nil
}

func isErrorResponse(content string) bool {
	if n := strings.Index(content, "#ERR"); n != -1 {
		return true
	}
	return false
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

func throttleRequest() errors.Error {
	if err := nlreqThrottler.getWaiter(); err != nil {
		return err
	}
	defer nlreqThrottler.releaseWaiter()

	select {
	case <-nlreqThrottler.nlgate():
		defer func() {
			nlreqThrottler.nlgate() <- true
		}()
		return nil
	case <-time.After(waitTimeout):
		return errors.NewNaturalLanguageRequestError(errors.E_NL_TIMEOUT)
	}
}

func ProcessRequest(nlCred, nlOrgId, nlquery string, elems []*algebra.Path, nloutputOpt naturalOutput,
	explain, advise bool,
	context NaturalContext, record func(execution.Phases, time.Duration)) (string, algebra.Statement, errors.Error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", nil, err
	}

	getJwt := util.Now()
	jwt, err := getJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return "", nil, err
	}

	keyspaceInfo := make(map[string]interface{}, len(elems))
	inferschema := util.Now()
	keyspaceInfo, err = keyspacesInfoForPrompt(keyspaceInfo, elems, context)
	record(execution.INFERSCHEMA, util.Since(inferschema))
	if err != nil {
		return "", nil, err
	}

	var prompt *prompt
	switch nloutputOpt {
	case SQL:
		prompt, err = newSQLPrompt(keyspaceInfo, nlquery, "", false)
	case JSUDF:
		prompt, err = newJSUDFPrompt(keyspaceInfo, nlquery, "")
	case FTSSQL:
		prompt, err = newSQLPrompt(keyspaceInfo, nlquery, "", true)
	default:
		err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
	}
	if err != nil {
		return "", nil, err
	}

	chatcompletionreq := util.Now()
	content, err := doChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", nil, err
	}
	if isErrorResponse(content) {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, content)
	}

	parse := util.Now()
	stmt, err := getStatement(content, nloutputOpt)
	if err != nil {
		return "", nil, err
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	var parseErr error
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))
	if parseErr != nil {
		retrytime := util.Now()
		prompt = buildRetryPrompt(prompt, content, parseErr.Error())
		orgContent := content
		for i := 1; i < maxCorrectionRetries; i++ {
			content, stmt, nlAlgebraStmt, parseErr = retryRequest(nlCred, nlOrgId, prompt, record, nloutputOpt, explain, advise)
			if parseErr == nil {
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			} else {
				prompt = buildRetryPrompt(prompt, content, parseErr.Error())
			}
		}
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			fmt.Sprintf("failed to parse generated statement- %v", orgContent), parseErr)
	}

	return stmt, nlAlgebraStmt, nil
}

func buildRetryPrompt(pmt *prompt, assistantContent string, reason string) *prompt {
	assistantmessage := message{
		Role:    "assistant",
		Content: assistantContent,
	}
	pmt.Messages = append(pmt.Messages, assistantmessage)

	var parseErrorMessage strings.Builder
	parseErrorMessage.WriteString("The previous response errored out with: ")
	parseErrorMessage.WriteString(reason)
	parseErrorMessage.WriteString(".\nCan you correct the previous response?")

	pmt.Messages = append(pmt.Messages, message{
		Role:    "user",
		Content: parseErrorMessage.String(),
	})

	return pmt
}

func retryRequest(nlCred, nlOrgId string, prompt *prompt,
	record func(execution.Phases, time.Duration), nloutputOpt naturalOutput,
	explain, advise bool) (string, string, algebra.Statement, error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", "", nil, err
	}

	getJwt := util.Now()
	jwt, err := getJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return "", "", nil, err
	}

	chatcompletionreq := util.Now()
	content, err := doChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", "", nil, err
	}
	if isErrorResponse(content) {
		return "", "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, content)
	}

	parse := util.Now()
	stmt, err := getStatement(content, nloutputOpt)
	if err != nil {
		return "", "", nil, err
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	var parseErr error
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))

	return content, stmt, nlAlgebraStmt, parseErr
}

func ProcessConversationalRequest(nlCred, nlOrgId, nlquery string, chatId string, nloutputOpt naturalOutput,
	explain, advise bool,
	user string,
	context NaturalContext, record func(execution.Phases, time.Duration)) (string, algebra.Statement, errors.Error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", nil, err
	}

	getJwt := util.Now()
	jwt, err := getJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return "", nil, err
	}

	var ce *ChatEntry
	rv := naturalchatHistory.Get(chatId, nil)
	if rv != nil {
		ce = rv.(*ChatEntry)
	} else {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}

	ce.Lock()
	defer ce.Unlock()
	if ce.Removed {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL,
			fmt.Sprintf("conversation with \"natural_chatid\":%s was deleted", chatId))
	}
	if ce.Paused {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL,
			fmt.Sprintf("conversation with \"natural_chatid\":%s was paused", chatId))
	}

	if ce.User != user {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_WRONG_USER)
	}
	var prompt *prompt
	if ce.prompt == nil {
		keyspaceInfo := make(map[string]interface{}, len(ce.Keyspaces))
		inferschema := util.Now()
		keyspaceInfo, err = keyspacesInfoForPrompt(keyspaceInfo, ce.Keyspaces, context)
		record(execution.INFERSCHEMA, util.Since(inferschema))
		if err != nil {
			return "", nil, err
		}

		switch nloutputOpt {
		case SQL:
			prompt, err = newSQLPrompt(keyspaceInfo, nlquery, ce.Summary, false)
		case JSUDF:
			prompt, err = newJSUDFPrompt(keyspaceInfo, nlquery, ce.Summary)
		case FTSSQL:
			prompt, err = newSQLPrompt(keyspaceInfo, nlquery, ce.Summary, true)
		default:
			err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
		}
		ce.Summary = ""
		if err != nil {
			return "", nil, err
		}
	} else {
		switch nloutputOpt {
		case SQL:
			prompt = newSQLIterativePrompt(ce.prompt, nlquery, false)
		case JSUDF:
			prompt = newJSUDFIterativePrompt(ce.prompt, nlquery)
		case FTSSQL:
			prompt = newSQLIterativePrompt(ce.prompt, nlquery, true)
		default:
			err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
		}
		if err != nil {
			return "", nil, err
		}
	}

	if prompt.Size >= _MAX_PROMPT_SIZE {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PROMPT_TOO_LARGE,
			logging.HumanReadableSize(int64(prompt.Size), false), logging.HumanReadableSize(_MAX_PROMPT_SIZE, false))
	}

	chatcompletionreq := util.Now()
	content, err := doChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	completeConversationPrompt(content, ce, prompt)
	if err != nil {
		return "", nil, err
	}
	if isErrorResponse(content) {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, content)
	}

	parse := util.Now()
	stmt, err := getStatement(content, nloutputOpt)
	if err != nil {
		return "", nil, err
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	var parseErr error
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))
	if parseErr != nil {
		retrytime := util.Now()
		prompt = buildRetryPrompt(prompt, content, parseErr.Error())
		orgContent := content
		for i := 1; i < maxCorrectionRetries; i++ {
			content, stmt, nlAlgebraStmt, parseErr = retryRequest(nlCred, nlOrgId, prompt, record, nloutputOpt, explain, advise)
			if parseErr == nil {
				completeConversationPrompt(content, ce, prompt)
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			} else {
				prompt = buildRetryPrompt(prompt, content, parseErr.Error())
			}
		}
		completeConversationPrompt(content, ce, prompt)
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			fmt.Sprintf("failed to parse generated statement- %v", orgContent), parseErr)
	}

	return stmt, nlAlgebraStmt, nil
}

func completeConversationPrompt(content string, ce *ChatEntry, prompt *prompt) {
	if content != "" {
		assistantmessage := message{
			Role:    "assistant",
			Content: content,
		}
		prompt.Messages = append(prompt.Messages, assistantmessage)

		ce.prompt = prompt
		naturalchatHistory.Add(ce, ce.Id, nil)
	}
}

func ProcessBeginChat(naturalcontext, datastorecreds string, keyspaces []*algebra.Path) (string, errors.Error) {

	if IsChatCacheFull() {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_CACHE_FULL)
	}

	chatId, err := util.UUIDV4()
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, err.Error())
	}

	ce := &ChatEntry{
		Id:        chatId,
		Keyspaces: keyspaces,
		User:      datastorecreds,
	}
	AddConversation(ce, chatId)
	return chatId, nil
}

func ProcessEndChat(chatId, datastorecreds string) errors.Error {

	rv := GetConversation(chatId)
	if rv == nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}
	ce, ok := rv.(*ChatEntry)
	if !ok {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, "failed to cast cache entry")
	}
	ce.Lock()
	if ce.User != datastorecreds {
		ce.Unlock()
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_WRONG_USER)
	}
	DeleteConversation(chatId)
	ce.Removed = true
	ce.Unlock()
	return nil
}

const CHAT_DOC_TTL_DURATION = 7 * 24 * time.Hour
const summarizeThreshold = 1024 * 10
const summarizeMessageLen = 8

const (
	maxRetry = 6
	interval = 100 * time.Millisecond
)

func ProcessPauseChat(chatId, requestId, datastorecreds string,
	nlOrgId, nlCred string,
	summarize value.Tristate,
	record func(execution.Phases, time.Duration)) errors.Error {
	if chatId == "" {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_CHAT_ID)
	}

	rv := GetConversation(chatId)
	if rv == nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}
	ce, ok := rv.(*ChatEntry)
	if !ok {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, "failed to cast cache entry")
	}
	ce.Lock()
	defer ce.Unlock()
	if ce.User != datastorecreds {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_WRONG_USER)
	}

	shouldSummarize := ce.prompt != nil &&
		(summarize == value.TRUE ||
			(summarize == value.NONE &&
				(ce.prompt.Size >= summarizeThreshold || len(ce.prompt.Messages) >= summarizeMessageLen)))
	if shouldSummarize {
		missingnlparams := []string{}
		if nlOrgId == "" {
			missingnlparams = append(missingnlparams, "\"natural_orgid\"")
		}
		if nlCred == "" {
			missingnlparams = append(missingnlparams, "\"natural_cred\"")
		}
		if len(missingnlparams) > 0 {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_NL_PARAM, strings.Join(missingnlparams, ","))
		}
		err := summarizePrompt(ce, nlOrgId, nlCred, record)
		if err != nil {
			return err
		}
	}

	hasquerymetadata, err := hasQueryMetadataForNLChat(true, requestId, "Natural Language chat PAUSE", true)
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED,
			fmt.Sprintf("failed to get query metadata: %v", err))
	} else if !hasquerymetadata {
		return errors.NewMissingQueryMetadataError("PAUSE CHAT")
	}

	store := datastore.GetDatastore()
	if store == nil {
		err := errors.NewNoDatastoreError()
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED, "failed to get datastore", err)
	}

	queryMetadata, err := store.GetQueryMetadata()
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED, "failed to get query metadata: %v", err)
	}

	dpairs := make([]value.Pair, 1)
	queryContext := datastore.GetDurableQueryContextFor(queryMetadata)

	marshalledchat, merr := ce.MarshalJSON()
	if merr != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED, "failed to marshal chat entry", merr)
	}
	key := fmt.Sprintf("%s%s", CHAT_DOC_PREFIX, chatId)
	dpairs[0].Name = key
	dpairs[0].Value = value.NewValue(map[string]interface{}{"chat": base64.StdEncoding.EncodeToString(marshalledchat)})
	ttltime := time.Now().Add(CHAT_DOC_TTL_DURATION)
	opt := value.NewValue(map[string]interface{}{})
	opt.SetField("expiration", ttltime.Unix())
	dpairs[0].Options = opt
	insertInterval := interval
	for i := 0; i < maxRetry; i++ {
		_, _, errs := queryMetadata.Insert(dpairs, queryContext, false)
		if len(errs) > 0 {
			if couchbase.CanRetryWithRefresh(errs[0]) {
				time.Sleep(insertInterval)
				insertInterval *= 2
			} else {
				logging.Errorf("Error inserting into QUERY_METADATA bucket: %v (key %s)", errs, key)
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED,
					fmt.Sprintf("err inserting the chat document: %v", errs))
			}
		} else {
			break
		}
	}
	DeleteConversation(chatId)
	return nil
}

func summarizePrompt(ce *ChatEntry, nlorgid, nlcred string, record func(execution.Phases, time.Duration)) errors.Error {
	if ce.prompt == nil || len(ce.prompt.Messages) <= 1 {
		return nil
	}

	var promptBuf strings.Builder
	promptBuf.WriteString("The following is a conversation history between a user and an assistant. " +
		"The conversation history is being summarized to save space but important information might be lost in the process. " +
		"Summarize the conversation while keeping important details that can be useful for the continuation of the conversation. " +
		"Preserve all important details related to the assistant's sql++ suggestions :" +
		"Fields used in SELECT, WHERE, JOIN, GROUP BY, and ORDER BY clauses\n" +
		"Any predicates, filters, conditions, and their values\n" +
		"Join relationships, including keys and join types\n" +
		"Aggregations, functions, and computed expressions\n" +
		"Relevant bucket, scope, and collection names\n" +
		"Capture the user's intent and any constraints or preferences expressed\n" +
		"Retain important assumptions or clarifications made by the assistant\n" +
		"Trim redundant information\n\n")
	for _, msg := range ce.prompt.Messages {
		promptBuf.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	promptBuf.WriteString("Summarize the above conversation history as precisely as possible.\n\n")

	pmt := &prompt{
		InitMessages: []message{
			message{
				Role:    "system",
				Content: "You are a helpful assistant for summarizing conversation history.",
			},
		},
		Messages: []message{
			message{
				Role:    "user",
				Content: promptBuf.String(),
			},
		},
		CompletionSettings: completionSettings{
			Model:       GPT4o_2024_05_13,
			Temperature: 0,
			Seed:        1,
			Stream:      false,
		},
		Size: len(promptBuf.String()),
	}

	getJwt := util.Now()
	jwt, err := getJWTFromSessionsApi(nlcred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, err)
	}

	chatcompletions := util.Now()
	content, err := doChatCompletionsReq(pmt, nlorgid, jwt, nlcred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletions))
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, err)
	}
	ce.Summary = content
	ce.prompt = nil
	return nil
}

const _BATCH_SIZE = 64

var _STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(_BATCH_SIZE)

func ProcessResumeChat(chatId, requestId, datastorecreds string) errors.Error {
	if chatId == "" {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_CHAT_ID)
	}

	if IsChatCacheFull() {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_CACHE_FULL))
	}

	hasquerymetadata, err := hasQueryMetadataForNLChat(false, requestId, "", false)
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			fmt.Sprintf("failed to get query metadata: %v", err))
	} else if !hasquerymetadata {
		return errors.NewMissingQueryMetadataError("RESUME CHAT")
	}

	store := datastore.GetDatastore()
	if store == nil {
		err := errors.NewNoDatastoreError()
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "failed to get datastore", err)
	}

	queryMetadata, err := store.GetQueryMetadata()
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "failed to get query metadata", err)
	}

	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)
	key := fmt.Sprintf("%s%s", CHAT_DOC_PREFIX, chatId)

	queryContext := datastore.GetDurableQueryContextFor(queryMetadata)
	ce := &ChatEntry{}
	var chatdoc value.AnnotatedValue
	var ok bool
	claimed := false

	claimInterval := interval
	for claimFetch := 0; claimFetch < maxRetry; claimFetch++ {

		errs := queryMetadata.Fetch([]string{key}, fetchMap, queryContext, nil, nil, false)
		if errs != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
				fmt.Sprintf("errs in fetching the chat document: %v", errs))
		}

		if chatdoc, ok = fetchMap[key]; !ok || chatdoc == nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
				fmt.Sprintf("chat with id:%s is not found in QUERY_METADATA", chatId))
		}

		val := chatdoc.GetValue()
		if vt := val.Type(); vt != value.OBJECT {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
				fmt.Sprintf("value type for chat document: %s expected object type %s", val, vt))
		}

		claimer, ok := val.Field("claimer")
		if ok && claimer.ToString() != distributed.RemoteAccess().WhoAmI() {
			claimtime, ok := val.Field("claim_time")
			if !ok {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
					"\"claim_time\" field is not found in the chat document")
			}
			if ct := claimtime.Type(); ct != value.STRING {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
					fmt.Sprintf("unexpected value type for \"claim_time\" field in the chat document: %s expected string", ct), err)
			}
			ct, perr := time.Parse(util.DEFAULT_FORMAT, claimtime.ToString())
			if perr != nil {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
					"failed to parse claim_time field in the chat document", perr)
			}
			if time.Since(ct) < 2*time.Minute {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
					fmt.Sprintf("chat is currently claimed by %s", claimer.ToString()))
			}
			// orphaned claim, can be claimed
		}

		b, err := GetChatDataFromObjectValue(val)
		if err != nil {
			return err
		}

		uerr := ce.UnmarshalJSON(b)
		if uerr != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "unmarshalling decoded chat failed", uerr)
		}

		if ce.User != datastorecreds {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_WRONG_USER)
		}

		udpairs := make([]value.Pair, 1)
		udpairs[0].Name = key
		chatdoc.SetField("claimer", value.NewValue(distributed.RemoteAccess().WhoAmI()))
		chatdoc.SetField("claim_time", value.NewValue(time.Now().Format(util.DEFAULT_FORMAT)))
		udpairs[0].Value = chatdoc

		retryClaim := false
		claimUpdateInterval := interval
		for claimUpdate := 0; claimUpdate < maxRetry; claimUpdate++ {
			_, _, errs = queryMetadata.Update(udpairs, queryContext, false)
			if len(errs) > 0 {
				if couchbase.CanRetryWithRefresh(errs[0]) {
					time.Sleep(claimUpdateInterval)
					claimUpdateInterval *= 2
				} else if errs[0].HasCause(errors.E_CAS_MISMATCH) || errs[0].ContainsText("SYNC_WRITE_IN_PROGRESS") {
					// some else tried to resume concurrently
					chatdoc.Recycle()
					chatdoc = nil
					fetchMap[key] = nil
					ce.Reset()
					retryClaim = true
					break
				} else {
					logging.Errorf("Chat claim failed: error updating QUERY_METADATA bucket (key %s): %v", key, errs)
					return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
						fmt.Sprintf("err updating the chat document: %v", errs))
				}
			} else {
				claimed = true
				break
			}
		}

		if retryClaim {
			claimInterval *= 2
			time.Sleep(claimInterval)
			continue
		}

		if claimed {
			logging.Infof("Chat claimed successfully for chat id: %s", chatId)
			break
		}
	}

	if !claimed {
		logging.Errorf("Chat claim failed after %d retries: failed to update the chat document for chat id: %s", maxRetry, chatId)
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			fmt.Sprintf("failed to claim chat document for chat id: %s after retries: %d", chatId, maxRetry))
	}

	dpairs := make([]value.Pair, 1)
	dpairs[0].Name = key
	completeClaimInterval := interval
	claimcompleted := true
	for claimComplete := 0; claimComplete < maxRetry; claimComplete++ {
		claimcompleted = false
		_, _, errs := queryMetadata.Delete(dpairs, queryContext, false)
		if len(errs) > 0 {
			if couchbase.CanRetryWithRefresh(errs[0]) {
				time.Sleep(completeClaimInterval)
				completeClaimInterval *= 2
			} else {
				logging.Errorf("Chat claim completion failed: error deleting from QUERY_METADATA bucket (key %s): %v", key, errs)
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
					fmt.Sprintf("err deleting the chat document: %v", errs))
			}
		} else {
			logging.Infof("Chat claim completed for chat id: %s", chatId)
			claimcompleted = true
			break
		}
	}

	if !claimcompleted {
		logging.Errorf("Chat claim completion failed after %d retries:"+
			" error in deleting the chat document for chat id: %s", maxRetry, chatId)
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			fmt.Sprintf("failed to complete the claim for chat document for chat id: %s after retries: %d", chatId, maxRetry))
	}

	ce.Id = chatId
	AddConversation(ce, ce.Id)
	return nil
}

func getStatement(content string, nloutputOpt naturalOutput) (string, errors.Error) {
	if content == "" {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT, "empty response")
	}
	switch nloutputOpt {
	case SQL, FTSSQL:
		return getSQLContent(content), nil
	case JSUDF:
		return getJsContent(content), nil
	default:
		return "", errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
	}
}

func getSQLContent(content string) string {
	sqlstmt := strings.TrimPrefix(content, "```sql\n")
	sqlstmt = strings.TrimSuffix(sqlstmt, "\n```")
	if end := len(sqlstmt) - 1; sqlstmt[end] == ';' {
		sqlstmt = sqlstmt[:end]
	}
	sqlstmt = strings.TrimSpace(sqlstmt)
	return sqlstmt
}

func getJsContent(content string) string {
	return strings.TrimSpace(
		strings.TrimSuffix(
			strings.TrimPrefix(content, "```javascript"), "\n```"))
}

const CHAT_DOC_PREFIX = "aichat::"

func (ce *ChatEntry) MarshalJSON() ([]byte, error) {
	rv := map[string]interface{}{}
	if user := ce.User; user != "" {
		rv["user"] = user
	}
	keyspaces := make([]string, len(ce.Keyspaces))
	for i, k := range ce.Keyspaces {
		keyspaces[i] = k.ProtectedString()
	}
	rv["keyspaces"] = keyspaces
	if pmt := ce.prompt; pmt != nil {
		rv["prompt"] = pmt
	}
	if summ := ce.Summary; summ != "" {
		rv["summary"] = summ
	}
	return json.Marshal(rv)
}

func (ce *ChatEntry) UnmarshalJSON(body []byte) error {
	var unmarshalledStruct struct {
		Keyspaces []string `json:"keyspaces"`
		Prompt    *prompt  `json:"prompt"`
		User      string   `json:"user"`
		Summary   string   `json:"summary"`
	}

	err := json.Unmarshal(body, &unmarshalledStruct)
	if err != nil {
		return err
	}

	if user := unmarshalledStruct.User; user != "" {
		ce.User = user
	}
	if keyspaces := unmarshalledStruct.Keyspaces; keyspaces != nil {
		keyspacelist := strings.Join(keyspaces, ",")
		elems, err := algebra.ParseAndValidatePathList(keyspacelist, "default", "")
		if err != nil {
			return fmt.Errorf("error validating keyspaces: %s", err)
		}
		ce.Keyspaces = elems
	}
	if prompt := unmarshalledStruct.Prompt; prompt != nil {
		ce.prompt = prompt
	}
	if summary := unmarshalledStruct.Summary; summary != "" {
		ce.Summary = summary
	}
	return nil
}

func (ce *ChatEntry) Reset() {
	ce.User = ""
	ce.Keyspaces = nil
	ce.prompt = nil
	ce.Id = ""
	ce.Summary = ""
	ce.Removed = false
	ce.Paused = false
}

func GetChatDataFromObjectValue(val value.Value) ([]byte, errors.Error) {
	encodedchat, ok := val.Field("chat")
	if !ok {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
			"\"chat\" field is not found in the chat document")
	}

	if et := encodedchat.Type(); et != value.STRING {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
			fmt.Sprintf("value type for \"chat\" field in the chat document: %s expected string", et))
	}

	b, derr := base64.StdEncoding.DecodeString(encodedchat.ToString())
	if derr != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "chat decoding failed", derr)
	}
	return b, nil
}
