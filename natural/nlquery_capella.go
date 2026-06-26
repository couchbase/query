//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// This file holds the Capella (iQ) natural-language path: the query engine
// generates statements via the Capella iQ service. All package-level symbols
// are capella-prefixed per the natural package naming convention; shared
// machinery (chat lifecycle, prompt/ChatEntry types, output parsing, throttler)
// lives in nlquery.go and is called unprefixed. The direct ai_gateway path is
// in nlquery_direct.go. Routing between the two is decided by IsCapellaPath.

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
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func init() {
	expression.GetModelProvidersFunc = GetCapellaModelProviders
}

const (
	// APIs
	capellaCPURL = "https://api.cloud.couchbase.com"
	// for dev:
	// capellaCPURL = "https://api.dev.nonprod-project-avengers.com"
	capellaSessionsAPI = capellaCPURL + "/sessions"
)

func getCapellaCompletionsApi(orgid string) string {
	return fmt.Sprintf("%v/v2/organizations/%v/integrations/iq/openai/chat/completions", capellaCPURL, orgid)
}

func getCapellaModelProvidersApi(orgid string, enabledOnly bool) string {
	if enabledOnly {
		return fmt.Sprintf("%v/v2/organizations/%v/iq/modelProviders?enabled=true", capellaCPURL, orgid)
	}
	return fmt.Sprintf("%v/v2/organizations/%v/iq/modelProviders", capellaCPURL, orgid)
}

const capellaCacheLimit = 65536

const capellaCompletionsReqBackoffInit = 1 * time.Second
const capellaCompletionsReqRetry = 5

const capellaClientTimeout = 2 * time.Minute

var capellaClient = &http.Client{
	Timeout: capellaClientTimeout,
}

const (
	// Models
	capellaGPT4o20240513         = "gpt-4o-2024-05-13"
	capellaBedrockClaudeSonnet45 = "us.anthropic.claude-sonnet-4-5-20250929-v1:0"
)

const (
	// Providers
	capellaProviderOpenAI  = "openai"
	capellaProviderBedrock = "bedrock"
)

var capellaDefaultProviderModels = map[string]string{
	capellaProviderOpenAI:  capellaGPT4o20240513,
	capellaProviderBedrock: capellaBedrockClaudeSonnet45,
}

func capellaResolveModel(provider string, availableModels []string) (string, errors.Error) {
	if model := capellaDefaultProviderModels[provider]; model != "" {
		return model, nil
	}
	if len(availableModels) > 0 {
		return availableModels[0], nil
	}
	return "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_DEFAULT_MODEL_FOR_VENDOR, provider)
}

type capellaModelProvider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Models  []string `json:"models"`
	Enabled bool     `json:"enabled"`
}

// errCauseFromBody reads the response body and returns a cause error populated
// with the backend message. If the body is valid JSON it is formatted as a map;
// otherwise the raw text is used. Returns nil when the body is empty.

func capellaV2ErrCauseFromBody(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var errRes map[string]interface{}
	if json.Unmarshal(body, &errRes) == nil {
		return fmt.Errorf("%v", errRes)
	}
	return fmt.Errorf("%s", body)
}

func capellaMakeV2Request(method, url string, payload []byte, jwt string) (*http.Response, errors.Error) {
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
	resp, e := capellaClient.Do(req)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_V2_SEND_REQ, url, e)
	}
	return resp, nil
}

// doV2Request makes an authenticated request to a v2 API endpoint. On a 401 it
// refreshes the JWT and retries with exponential backoff. Returns the response so
// the caller can read and close the body; connection-level errors are the only
// non-nil errors returned.

func capellaDoV2Request(method, url string, payload []byte, nlCred, jwt string) (*http.Response, errors.Error) {

	resp, err := capellaMakeV2Request(method, url, payload, jwt)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// JWT may have been refreshed by an external client; retry with backoff.
	resp.Body.Close()
	backoff := capellaCompletionsReqBackoffInit
	for retries := 0; retries < capellaCompletionsReqRetry; retries++ {
		time.Sleep(backoff)
		var nlErr errors.Error
		jwt, nlErr = getCapellaJWTFromSessionsApi(nlCred, true)
		if nlErr != nil {
			return nil, nlErr
		}
		resp, err = capellaMakeV2Request(method, url, payload, jwt)
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

func GetCapellaModelProviders(nlCred, nlOrganizationId string, enabledOnly bool) (interface{}, errors.Error) {
	jwt, err := getCapellaJWTFromSessionsApi(nlCred, false)
	if err != nil {
		return nil, err
	}
	providers, err := getCapellaModelProviders(nlOrganizationId, nlCred, jwt, enabledOnly)
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

func getCapellaModelProviders(nlOrganizationId, nlCred, jwt string, enabledOnly bool) ([]capellaModelProvider, errors.Error) {
	url := getCapellaModelProvidersApi(nlOrganizationId, enabledOnly)
	resp, err := capellaDoV2Request("GET", url, nil, nlCred, jwt)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_PROVIDERS_REQ_FAILED, resp.StatusCode, capellaV2ErrCauseFromBody(body))
	}
	var providers []capellaModelProvider
	if e := json.NewDecoder(resp.Body).Decode(&providers); e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_PROVIDERS_RESP_UNMARSHAL, url, e)
	}
	return providers, nil
}

// capellaResolveProviderAndModel determines the provider and model to use for the request.
// It validates provider availability via the modelProviders API and enforces:
//   - model without provider → error
//   - unknown/disabled provider → error
//   - no providers enabled → error
//   - no provider specified → prefer openai; fall back to the only enabled provider
//   - no model specified → use the default model if defined, else use the first model listed by the API for the provider

func capellaResolveProviderAndModel(nlProvider, nlModel, nlOrganizationId, nlCred, jwt string) (provider, model string, err errors.Error) {
	if nlModel != "" && nlProvider == "" {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_WITHOUT_VENDOR)
	}

	allProviders, err := getCapellaModelProviders(nlOrganizationId, nlCred, jwt, false)
	if err != nil {
		return "", "", err
	}

	if len(allProviders) == 0 {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_VENDORS_AVAILABLE)
	}

	if nlProvider != "" {
		for i := range allProviders {
			// IDs are lowercase per the api.
			if allProviders[i].ID == strings.ToLower(nlProvider) {
				if allProviders[i].Enabled {
					provider = allProviders[i].ID
					if nlModel != "" {
						model = strings.ToLower(nlModel)
					} else {
						var resolveErr errors.Error
						model, resolveErr = capellaResolveModel(provider, allProviders[i].Models)
						if resolveErr != nil {
							return "", "", resolveErr
						}
					}
					return provider, model, nil
				} else {
					return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_ENABLED, nlProvider)
				}
			}
		}
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_SUPPORTED, nlProvider)
	}
	// No provider specified: prefer openai, else use the first available provider.
	var fallbackprovider, fallbackmodel string
	var modelErr errors.Error
	for i := range allProviders {
		if allProviders[i].ID == capellaProviderOpenAI && allProviders[i].Enabled {
			model, modelErr = capellaResolveModel(capellaProviderOpenAI, allProviders[i].Models)
			if modelErr != nil {
				return "", "", modelErr
			}
			return capellaProviderOpenAI, model, nil
		}

		if fallbackprovider == "" && allProviders[i].Enabled {
			fallbackprovider = allProviders[i].ID
			fallbackmodel, modelErr = capellaResolveModel(fallbackprovider, allProviders[i].Models)
			if modelErr != nil {
				return "", "", modelErr
			}
		}
	}

	if fallbackprovider == "" {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_VENDORS_ENABLED)
	}

	return fallbackprovider, fallbackmodel, nil
}

var capellaCacheEntryCreation sync.Mutex

var capellaJWTCache *util.GenCache

func init() {
	capellaJWTCache = util.NewGenCache(capellaCacheLimit)
}

type capellaJWTCacheEntry struct {
	sync.RWMutex
	token     string
	expiresAt time.Time
}

type capellaJWTResponse struct {
	JWT  string `json:"jwt"`
	User struct {
		ExpiresAt string `json:"expiresAt"`
	} `json:"user"`
}

func getCapellaJWTFromSessionsApi(nlCred string, refresh bool) (string, errors.Error) {
	var entry *capellaJWTCacheEntry
	if !refresh {
		cacheEntry := capellaJWTCache.Get(nlCred, nil)
		if cacheEntry != nil {
			entry = cacheEntry.(*capellaJWTCacheEntry)
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
			capellaCacheEntryCreation.Lock()
			defer capellaCacheEntryCreation.Unlock()

			// already created before us
			cacheEntry := capellaJWTCache.Get(nlCred, nil)
			if cacheEntry != nil {
				entry = cacheEntry.(*capellaJWTCacheEntry)
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
				entry = &capellaJWTCacheEntry{
					expiresAt: time.Time{},
				}
			}
		}
	} else {
		entry = &capellaJWTCacheEntry{
			expiresAt: time.Time{},
		}
		capellaCacheEntryCreation.Lock()
		defer capellaCacheEntryCreation.Unlock()
	}

	reqJwt, err := http.NewRequest("POST", capellaSessionsAPI, nil)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CREATE_SESSIONS_REQ, capellaSessionsAPI, err)
	}

	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(nlCred))
	reqJwt.Header.Set("Authorization", "Basic "+encodedCredentials)

	resp, err := capellaClient.Do(reqJwt)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SEND_SESSIONS_REQ, capellaSessionsAPI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_AUTH)
	}

	var result capellaJWTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_RESP_UNMARSHAL, capellaSessionsAPI, err)
	}

	expTime, err := time.Parse(time.RFC3339Nano, result.User.ExpiresAt)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_SESSIONS_PARSE_EXPIRE_TIME,
			result.User.ExpiresAt, err)
	}

	entry.token = "Bearer " + result.JWT
	entry.expiresAt = expTime

	capellaJWTCache.Add(entry, nlCred, nil)

	return entry.token, nil
}

// Prompt

func newCapellaSQLPrompt(keyspaceInfo map[string]interface{}, naturalPrompt, summary, hint string, forfts bool,
	provider string, model string) (*prompt, errors.Error) {
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a Couchbase Capella expert. Your task is to create valid queries to retrieve" +
					" or create data based on the provided Information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		},
		Vendor: provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: capellaGetTemperatureForModel(provider, model),
			Seed:        1,
			Stream:      false,
		},
		Size: _INIT_SIZE,
	}

	if err := appendSQLUserMessage(rv, keyspaceInfo, naturalPrompt, summary, hint, forfts); err != nil {
		return nil, err
	}

	return rv, nil
}

func newCapellaJSUDFPrompt(keyspaceInfo map[string]interface{}, naturalPrompt, summary, hint string, provider string, model string) (*prompt, errors.Error) {
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a couchbase Capella expert. Your task is to write valid Javascript user defined functions" +
					" based on the provided information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		},
		Vendor: provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: capellaGetTemperatureForModel(provider, model),
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
	userMessageBuf.WriteString("\"")
	if hint != "" {
		userMessageBuf.WriteString("\n\nHint: \"")
		userMessageBuf.WriteString(hint)
		userMessageBuf.WriteString("\"")
	}
	userMessageBuf.WriteString("\n\nBased on the above Information, write a valid Javascript User Defined Function with" +
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

func newCapellaSQLIterativePrompt(chat *prompt, naturalPrompt string, hint string, forfts bool, provider, model string) *prompt {
	if provider != "" {
		chat.Vendor = provider
	}
	if model != "" {
		chat.CompletionSettings.Model = model
		chat.CompletionSettings.Temperature = capellaGetTemperatureForModel(provider, model)
	}

	return appendSQLIterativeUserMessage(chat, naturalPrompt, hint, forfts)
}

func newCapellaJSUDFIterativePrompt(chat *prompt, naturalPrompt string, hint string, provider, model string) *prompt {
	var userMessage string
	var userMessageBuf strings.Builder

	if provider != "" {
		chat.Vendor = provider
	}
	if model != "" {
		chat.CompletionSettings.Model = model
		chat.CompletionSettings.Temperature = capellaGetTemperatureForModel(provider, model)
	}
	userMessageBuf.WriteString("Your goal is to iterate on the previouly generated query by modifying it's code,")
	userMessageBuf.WriteString(" based on this prompt:")
	userMessageBuf.WriteString("\"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\".")
	if hint != "" {
		userMessageBuf.WriteString("\n\nHint: \"")
		userMessageBuf.WriteString(hint)
		userMessageBuf.WriteString("\"")
	}
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

func doCapellaChatCompletionsReq(prompt *prompt, nlOrganizationId string, jwt string, nlCred string) (string, errors.Error) {
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

	url := getCapellaCompletionsApi(nlOrganizationId)

	payload, perr := json.Marshal(prompt)
	if perr != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, perr)
	}

	resp, err := capellaDoV2Request("POST", url, payload, nlCred, jwt)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		cause := capellaV2ErrCauseFromBody(body)
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

	return chatComplRes.Choices[0].Message.Content, nil
}

func capellaGetTemperatureForModel(provider, model string) float64 {
	switch provider {
	case capellaProviderOpenAI:
		if strings.HasPrefix(model, "gpt-5") {
			return 1
		}
		return 0
	case capellaProviderBedrock:
		return 0
	default:
		return 0
	}
}

func ProcessCapellaRequest(nlCred, nlOrgId, nlProvider, nlModel, nlquery, nlHint string, elems []*algebra.Path, nloutputOpt naturalOutput,
	explain, advise bool,
	context NaturalContext, record func(execution.Phases, time.Duration)) (string, algebra.Statement, errors.Error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", nil, err
	}

	getJwt := util.Now()
	jwt, err := getCapellaJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return "", nil, err
	}

	provider, model, err := capellaResolveProviderAndModel(nlProvider, nlModel, nlOrgId, nlCred, jwt)
	if err != nil {
		return "", nil, err
	}

	keyspaceInfo := make(map[string]interface{}, len(elems))
	inferschema := util.Now()
	keyspaceInfo, _, err = keyspacesInfoForPrompt(keyspaceInfo, elems, context, false)
	record(execution.INFERSCHEMA, util.Since(inferschema))
	if err != nil {
		return "", nil, err
	}

	var prompt *prompt
	switch nloutputOpt {
	case SQL:
		prompt, err = newCapellaSQLPrompt(keyspaceInfo, nlquery, "", nlHint, false, provider, model)
	case JSUDF:
		prompt, err = newCapellaJSUDFPrompt(keyspaceInfo, nlquery, "", nlHint, provider, model)
	case FTSSQL:
		prompt, err = newCapellaSQLPrompt(keyspaceInfo, nlquery, "", nlHint, true, provider, model)

	default:
		err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
	}
	if err != nil {
		return "", nil, err
	}

	chatcompletionreq := util.Now()
	content, err := doCapellaChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", nil, err
	}
	if err := CheckAndReturnErrorResponse(content); err != nil {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, err)
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
		prompt = capellaBuildRetryPrompt(prompt, content, parseErr.Error())
		var retryErr error
		for i := 0; i < maxCorrectionRetries; i++ {
			var fatalErr errors.Error
			content, stmt, nlAlgebraStmt, fatalErr, retryErr = capellaRetryRequest(nlCred, nlOrgId, prompt, record, nloutputOpt, explain, advise)
			if fatalErr != nil {
				// Request-level failure (throttle, JWT/auth, gateway/transport error,
				// model refusal): not a correctable statement, so surface it immediately
				// instead of feeding it back as correction feedback and re-sending.
				return "", nil, fatalErr
			}
			if retryErr == nil {
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			} else {
				prompt = capellaBuildRetryPrompt(prompt, content, retryErr.Error())
			}
		}
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			content, retryErr)
	}

	return stmt, nlAlgebraStmt, nil
}

func capellaBuildRetryPrompt(pmt *prompt, assistantContent string, reason string) *prompt {
	assistantmessage := message{
		Role:    "assistant",
		Content: assistantContent,
	}
	pmt.Messages = append(pmt.Messages, assistantmessage)

	var parseErrorMessage strings.Builder
	parseErrorMessage.WriteString("The previous response errored out with: ")
	parseErrorMessage.WriteString(reason)
	parseErrorMessage.WriteString(".\nCan you correct the previous response?")
	pmt.Size += parseErrorMessage.Len()

	pmt.Messages = append(pmt.Messages, message{
		Role:    "user",
		Content: parseErrorMessage.String(),
	})

	return pmt
}

// capellaRetryRequest runs one correction round: it re-sends the prompt, extracts
// the statement and parses it. It separates the two failure modes so the caller
// can react correctly:
//   - fatalErr is a request-level failure (throttle, JWT/auth, a gateway or
//     transport error from the completion call, or a model refusal via #ERR). It
//     is not correctable by feeding it back to the model, so the caller must
//     surface it immediately.
//   - parseErr is a correctable failure: the model produced a statement that did
//     not parse (or an empty response), so the caller can append it as feedback
//     and retry.
//
// At most one of fatalErr / parseErr is non-nil.
func capellaRetryRequest(nlCred, nlOrgId string, prompt *prompt,
	record func(execution.Phases, time.Duration), nloutputOpt naturalOutput,
	explain, advise bool) (content, stmt string, nlAlgebraStmt algebra.Statement,
	fatalErr errors.Error, parseErr error) {

	waitTime := util.Now()
	if err := throttleRequest(); err != nil {
		record(execution.NLWAIT, util.Since(waitTime))
		return "", "", nil, err, nil
	}
	record(execution.NLWAIT, util.Since(waitTime))

	getJwt := util.Now()
	jwt, err := getCapellaJWTFromSessionsApi(nlCred, false)
	record(execution.GETJWT, util.Since(getJwt))
	if err != nil {
		return "", "", nil, err, nil
	}

	chatcompletionreq := util.Now()
	content, err = doCapellaChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", "", nil, err, nil
	}
	if cerr := CheckAndReturnErrorResponse(content); cerr != nil {
		return content, "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, cerr), nil
	}

	parse := util.Now()
	stmt, serr := getStatement(content, nloutputOpt)
	if serr != nil {
		return content, "", nil, nil, serr
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))

	return content, stmt, nlAlgebraStmt, nil, parseErr
}

func ProcessCapellaConversationalRequest(nlCred, nlOrgId, nlProvider, nlModel, nlquery, nlHint string, chatId string,
	nloutputOpt naturalOutput, explain, advise bool,
	users []string,
	context NaturalContext, record func(execution.Phases, time.Duration)) (string, algebra.Statement, errors.Error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", nil, err
	}

	getJwt := util.Now()
	jwt, err := getCapellaJWTFromSessionsApi(nlCred, false)
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

	if err := ce.CheckUser(users); err != nil {
		return "", nil, err
	}

	provider, model, err := capellaResolveProviderAndModel(nlProvider, nlModel, nlOrgId, nlCred, jwt)
	if err != nil {
		return "", nil, err
	}

	var prompt *prompt
	if ce.prompt == nil {
		keyspaceInfo := make(map[string]interface{}, len(ce.Keyspaces))
		inferschema := util.Now()
		keyspaceInfo, _, err = keyspacesInfoForPrompt(keyspaceInfo, ce.Keyspaces, context, false)
		record(execution.INFERSCHEMA, util.Since(inferschema))
		if err != nil {
			return "", nil, err
		}

		switch nloutputOpt {
		case SQL:
			prompt, err = newCapellaSQLPrompt(keyspaceInfo, nlquery, ce.Summary, nlHint, false, provider, model)
		case JSUDF:
			prompt, err = newCapellaJSUDFPrompt(keyspaceInfo, nlquery, ce.Summary, nlHint, provider, model)
		case FTSSQL:
			prompt, err = newCapellaSQLPrompt(keyspaceInfo, nlquery, ce.Summary, nlHint, true, provider, model)
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
			prompt = newCapellaSQLIterativePrompt(ce.prompt, nlquery, nlHint, false, provider, model)
		case JSUDF:
			prompt = newCapellaJSUDFIterativePrompt(ce.prompt, nlquery, nlHint, provider, model)
		case FTSSQL:
			prompt = newCapellaSQLIterativePrompt(ce.prompt, nlquery, nlHint, true, provider, model)
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
	content, err := doCapellaChatCompletionsReq(prompt, nlOrgId, jwt, nlCred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", nil, err
	}
	if err := CheckAndReturnErrorResponse(content); err != nil {
		completeConversationPromptLocked(content, ce, prompt)
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, err)
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
		prompt = capellaBuildRetryPrompt(prompt, content, parseErr.Error())
		var retryErr error
		for i := 0; i < maxCorrectionRetries; i++ {
			var fatalErr errors.Error
			content, stmt, nlAlgebraStmt, fatalErr, retryErr = capellaRetryRequest(nlCred, nlOrgId, prompt, record, nloutputOpt, explain, advise)
			if fatalErr != nil {
				// Request-level failure (throttle, JWT/auth, gateway/transport error,
				// model refusal): not a correctable statement, so surface it immediately
				// instead of feeding it back as correction feedback and re-sending.
				completeConversationPromptLocked(content, ce, prompt)
				return "", nil, fatalErr
			}
			if retryErr == nil {
				completeConversationPromptLocked(content, ce, prompt)
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			} else if i < maxCorrectionRetries-1 {
				prompt = capellaBuildRetryPrompt(prompt, content, retryErr.Error())
			}
		}
		completeConversationPromptLocked(content, ce, prompt)
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			content, retryErr)
	}

	completeConversationPromptLocked(content, ce, prompt)
	return stmt, nlAlgebraStmt, err
}

// caller should have already acquired lock on ce

func ProcessCapellaPauseChat(chatId, requestId string,
	datastorecreds []string,
	nlOrgId, nlCred string,
	summarize value.Tristate, nlprovider, nlmodel string,
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
	if err := ce.CheckUser(datastorecreds); err != nil {
		return err
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

		getJwt := util.Now()
		jwt, err := getCapellaJWTFromSessionsApi(nlCred, false)
		record(execution.GETJWT, util.Since(getJwt))
		if err != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, "failed to get JWT", err)
		}
		provider, model, err := capellaResolveProviderAndModel(nlprovider, nlmodel, nlOrgId, nlCred, jwt)
		if err != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, "failed to resolve vendor and model", err)
		}

		err = capellaSummarizePrompt(ce, nlOrgId, nlCred, provider, model, jwt, record)
		if err != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, err)
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
				logging.Errorf("%s Error inserting into QUERY_METADATA bucket: %v (key %s)", _CHAT_LOG_PREFIX, errs, key)
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED,
					fmt.Sprintf("err inserting the chat document: %v", errs))
			}
		} else {
			break
		}
	}
	ce.stopInactivityTimer()
	DeleteConversation(chatId)
	ce.Paused = true
	logging.Infof("%s Chat with id %s paused", _CHAT_LOG_PREFIX, chatId)
	return nil
}

func capellaSummarizePrompt(ce *ChatEntry, nlorgid, nlcred, provider, model, jwt string, record func(execution.Phases, time.Duration)) errors.Error {
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
		Vendor: provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: capellaGetTemperatureForModel(provider, model),
			Seed:        1,
			Stream:      false,
		},
		Size: len(promptBuf.String()),
	}

	chatcompletions := util.Now()
	content, err := doCapellaChatCompletionsReq(pmt, nlorgid, jwt, nlcred)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletions))
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, "chat completions request failed", err)
	}
	ce.Summary = content
	ce.prompt = nil
	return nil
}
