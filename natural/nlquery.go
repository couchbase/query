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
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func init() {
	expression.GetModelProvidersFunc = GetModelProviders
}

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

func getModelProvidersApi(orgid string, enabledOnly bool) string {
	if enabledOnly {
		return fmt.Sprintf("%v/v2/organizations/%v/iq/modelProviders?enabled=true", CP_URL, orgid)
	}
	return fmt.Sprintf("%v/v2/organizations/%v/iq/modelProviders", CP_URL, orgid)
}

const _CACHE_LIMIT = 65536
const MAX_KEYSPACES = 4
const _COMPLETIONS_REQ_BACKOFF_INIT = 1 * time.Second
const _COMPLETIONS_REQ_RETRY = 5

const naturalClientTimeout = 2 * time.Minute

var naturalClient = &http.Client{
	Timeout: naturalClientTimeout,
}

const (
	// Models
	GPT4o_2024_05_13          = "gpt-4o-2024-05-13"
	BEDROCK_CLAUDE_SONNET_4_5 = "us.anthropic.claude-sonnet-4-5-20250929-v1:0"
)

const (
	// Vendors
	VENDOR_OPENAI  = "openai"
	VENDOR_BEDROCK = "bedrock"
)

var defaultVendorModels = map[string]string{
	VENDOR_OPENAI:  GPT4o_2024_05_13,
	VENDOR_BEDROCK: BEDROCK_CLAUDE_SONNET_4_5,
}

func resolveModel(vendor string, availableModels []string) (string, errors.Error) {
	if model := defaultVendorModels[vendor]; model != "" {
		return model, nil
	}
	if len(availableModels) > 0 {
		return availableModels[0], nil
	}
	return "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_DEFAULT_MODEL_FOR_VENDOR, vendor)
}

type modelProvider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Models  []string `json:"models"`
	Enabled bool     `json:"enabled"`
}

// errCauseFromBody reads the response body and returns a cause error populated
// with the backend message. If the body is valid JSON it is formatted as a map;
// otherwise the raw text is used. Returns nil when the body is empty.
func v2errCauseFromBody(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var errRes map[string]interface{}
	if json.Unmarshal(body, &errRes) == nil {
		return fmt.Errorf("%v", errRes)
	}
	return fmt.Errorf("%s", body)
}

func makeV2Request(method, url string, payload []byte, jwt string) (*http.Response, errors.Error) {
	var body io.Reader
	if len(payload) > 0 {
		body = bytes.NewBuffer(payload)
	}
	req, e := http.NewRequest(method, url, body)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_V2_CREATE_REQ, url, e)
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", jwt)
	resp, e := naturalClient.Do(req)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_V2_SEND_REQ, url, e)
	}
	return resp, nil
}

// doV2Request makes an authenticated request to a v2 API endpoint. On a 401 it
// refreshes the JWT and retries with exponential backoff. Returns the response so
// the caller can read and close the body; connection-level errors are the only
// non-nil errors returned.
func doV2Request(method, url string, payload []byte, nlCred, jwt string) (*http.Response, errors.Error) {

	resp, err := makeV2Request(method, url, payload, jwt)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// JWT may have been refreshed by an external client; retry with backoff.
	resp.Body.Close()
	backoff := _COMPLETIONS_REQ_BACKOFF_INIT
	for retries := 0; retries < _COMPLETIONS_REQ_RETRY; retries++ {
		time.Sleep(backoff)
		var nlErr errors.Error
		jwt, nlErr = getJWTFromSessionsApi(nlCred, true)
		if nlErr != nil {
			return nil, nlErr
		}
		resp, err = makeV2Request(method, url, payload, jwt)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		resp.Body.Close()
		backoff *= 2
	}
	return resp, nil
}

// GetModelProviders fetches the list of enabled model providers for the given
// organization. It handles JWT acquisition and returns the result as a value
// ready for use in query results.
func GetModelProviders(nlCred, nlOrganizationId string, enabledOnly bool) (interface{}, errors.Error) {
	jwt, err := getJWTFromSessionsApi(nlCred, false)
	if err != nil {
		return nil, err
	}
	providers, err := getModelProviders(nlOrganizationId, nlCred, jwt, enabledOnly)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(providers))
	for i, p := range providers {
		models := make([]interface{}, len(p.Models))
		for j, m := range p.Models {
			models[j] = m
		}
		result[i] = map[string]interface{}{
			"id":      p.ID,
			"name":    p.Name,
			"models":  models,
			"enabled": p.Enabled,
		}
	}
	return result, nil
}

func getModelProviders(nlOrganizationId, nlCred, jwt string, enabledOnly bool) ([]modelProvider, errors.Error) {
	url := getModelProvidersApi(nlOrganizationId, enabledOnly)
	resp, err := doV2Request("GET", url, nil, nlCred, jwt)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_PROVIDERS_REQ_FAILED, resp.StatusCode, v2errCauseFromBody(body))
	}
	var providers []modelProvider
	if e := json.NewDecoder(resp.Body).Decode(&providers); e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_PROVIDERS_RESP_UNMARSHAL, url, e)
	}
	return providers, nil
}

// resolveVendorAndModel determines the vendor and model to use for the request.
// It validates vendor availability via the modelProviders API and enforces:
//   - model without vendor → error
//   - unknown/disabled vendor → error
//   - no vendors enabled → error
//   - no vendor specified → prefer openai; fall back to the only enabled vendor
//   - no model specified → use the default model if defined, else use the first model listed by the API for the vendor
func resolveVendorAndModel(nlVendor, nlModel, nlOrganizationId, nlCred, jwt string) (vendor, model string, err errors.Error) {
	if nlModel != "" && nlVendor == "" {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_WITHOUT_VENDOR)
	}

	allProviders, err := getModelProviders(nlOrganizationId, nlCred, jwt, false)
	if err != nil {
		return "", "", err
	}

	if len(allProviders) == 0 {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_VENDORS_AVAILABLE)
	}

	if nlVendor != "" {
		for i := range allProviders {
			// IDs are lowercase per the api.
			if allProviders[i].ID == strings.ToLower(nlVendor) {
				if allProviders[i].Enabled {
					vendor = allProviders[i].ID
					if nlModel != "" {
						model = strings.ToLower(nlModel)
					} else {
						var resolveErr errors.Error
						model, resolveErr = resolveModel(vendor, allProviders[i].Models)
						if resolveErr != nil {
							return "", "", resolveErr
						}
					}
					return vendor, model, nil
				} else {
					return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_ENABLED, nlVendor)
				}
			}
		}
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_SUPPORTED, nlVendor)
	}
	// No vendor specified: prefer openai, else use the first available vendor.
	var fallbackvendor, fallbackmodel string
	var modelErr errors.Error
	for i := range allProviders {
		if allProviders[i].ID == VENDOR_OPENAI && allProviders[i].Enabled {
			model, modelErr = resolveModel(VENDOR_OPENAI, allProviders[i].Models)
			if modelErr != nil {
				return "", "", modelErr
			}
			return VENDOR_OPENAI, model, nil
		}

		if fallbackvendor == "" && allProviders[i].Enabled {
			fallbackvendor = allProviders[i].ID
			fallbackmodel, modelErr = resolveModel(fallbackvendor, allProviders[i].Models)
			if modelErr != nil {
				return "", "", modelErr
			}
		}
	}

	if fallbackvendor == "" {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_VENDORS_ENABLED)
	}

	return fallbackvendor, fallbackmodel, nil
}

const (
	maxconcurrency       = 4
	maxWaiters           = 16
	waitTimeout          = 20 * time.Second
	maxCorrectionRetries = 4
)

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

	resp, err := naturalClient.Do(reqJwt)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SEND_SESSIONS_REQ, _SESSIONS_API, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_AUTH)
	}

	var result jwtResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
	Vendor             string             `json:"vendor"`
	Messages           []message          `json:"messages"`
}

func newSQLPrompt(keyspaceInfo map[string]interface{}, naturalPrompt string, forfts bool,
	vendor string, model string) (*prompt, errors.Error) {
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a Couchbase Capella expert. Your task is to create valid queries to retrieve" +
					" or create data based on the provided Information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		},
		Vendor: vendor,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: getTemperatureForModel(vendor, model),
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
		"\n\nQuote aliases with grave accent characters.")
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
	userMessage = userMessageBuf.String()
	rv.Messages = []message{
		message{
			Role:    "user",
			Content: userMessage,
		},
	}

	return rv, nil
}

func newJSUDFPrompt(keyspaceInfo map[string]interface{}, naturalPrompt string, vendor string, model string) (*prompt, errors.Error) {
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a couchbase Capella expert. Your task is to write valid Javascript user defined functions" +
					" based on the provided information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		},
		Vendor: vendor,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: getTemperatureForModel(vendor, model),
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
	type APIError struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	}
	type ChatCompletionResponse struct {
		Error   map[string]interface{} `json:"error"`
		Choices []Choice               `json:"choices"`
	}

	url := getCompletionsApi(nlOrganizationId)

	payload, perr := json.Marshal(prompt)
	if perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, perr)
	}

	resp, err := doV2Request("POST", url, payload, nlCred, jwt)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		cause := v2errCauseFromBody(body)
		switch resp.StatusCode {
		case http.StatusNotFound:
			return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ORG_NOT_FOUND, nlOrganizationId, cause)
		case http.StatusUnauthorized:
			return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ORG_UNAUTH, cause)
		default:
			return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED, resp.StatusCode, cause)
		}
	}

	var chatComplRes ChatCompletionResponse
	if perr = json.NewDecoder(resp.Body).Decode(&chatComplRes); perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, perr)
	}
	if chatComplRes.Error != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, chatComplRes.Error)
	}
	if len(chatComplRes.Choices) == 0 {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, fmt.Errorf("no message in response"))
	}
	content := chatComplRes.Choices[0].Message.Content
	if n := strings.Index(content, "#ERR"); n != -1 {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
			fmt.Errorf("%s", strings.TrimRight(content[n+6:], "\n `")))
	}
	return content, nil
}

func getTemperatureForModel(vendor, model string) float64 {
	switch vendor {
	case VENDOR_OPENAI:
		if strings.HasPrefix(model, "gpt-5") {
			return 1
		}
		return 0
	case VENDOR_BEDROCK:
		return 0
	default:
		return 0
	}
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
	priv := auth.NewPrivileges()

	var ds datastore.Datastore
	if context != nil {
		ds = context.Datastore()
		if ds == nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, fmt.Errorf("no datastore"))
		}
	} else {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, fmt.Errorf("no context"))
	}
	for _, p := range elems {
		ps := p.SimpleString()
		if p.IsSystem() || (strings.Contains(ps, ":") && algebra.IsSystemName(ps)) {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT,
				fmt.Errorf("system keyspace is not allowed: %s", ps))
		}
		priv.List = priv.List[:0]
		priv.Add(ps, auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)
		err = ds.Authorize(priv, context.Credentials())
		if err != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, err)
		}
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

func ProcessRequest(nlCred, nlOrgId, nlVendor, nlModel, nlquery string, elems []*algebra.Path, nloutputOpt naturalOutput,
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

	vendor, model, err := resolveVendorAndModel(nlVendor, nlModel, nlOrgId, nlCred, jwt)
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
		prompt, err = newSQLPrompt(keyspaceInfo, nlquery, false, vendor, model)
	case JSUDF:
		prompt, err = newJSUDFPrompt(keyspaceInfo, nlquery, vendor, model)
	case FTSSQL:
		prompt, err = newSQLPrompt(keyspaceInfo, nlquery, true, vendor, model)
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
		var retryErr error
		for i := 1; i < maxCorrectionRetries; i++ {
			content, stmt, nlAlgebraStmt, retryErr = retryRequest(nlCred, nlOrgId, prompt, record, nloutputOpt, explain, advise)
			if retryErr == nil {
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			} else {
				prompt = buildRetryPrompt(prompt, content, retryErr.Error())
			}
		}
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			content, retryErr)
	}

	return stmt, nlAlgebraStmt, nil
}

func buildRetryPrompt(pmt *prompt, assistantContent string, reason string) *prompt {
	assitantmessage := message{
		Role:    "assistant",
		Content: assistantContent,
	}
	pmt.Messages = append(pmt.Messages, assitantmessage)

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
