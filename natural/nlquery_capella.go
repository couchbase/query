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
	"github.com/couchbase/query/natural/ai_gateway"
	"github.com/couchbase/query/parser/n1ql"
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
	// Providers
	capellaProviderOpenAI  = "openai"
	capellaProviderBedrock = "bedrock"
)

// CapellaMissingCreds returns the Capella credentials missing from a request, as
// E_NL_MISSING_NL_PARAM arguments: "natural_cred", then "natural_orgid". A
// request is routed to the Capella path when either is set (IsCapellaPath), so a
// missing one is a half-configured request; naming it beats the confusing iQ
// failure (e.g. a modelProviders 404) it otherwise causes. Shared with the REST
// "natural" parameter check so the names and their order cannot drift.
func CapellaMissingCreds(nlCred, nlOrgId string) []string {
	var missing []string
	if nlCred == "" {
		missing = append(missing, "\"natural_cred\"")
	}
	if nlOrgId == "" {
		missing = append(missing, "\"natural_orgid\"")
	}
	return missing
}

// CapellaCheckCreds enforces that a Capella request carries both natural_cred
// and natural_orgid (see CapellaMissingCreds).
func CapellaCheckCreds(nlCred, nlOrgId string) errors.Error {
	if missing := CapellaMissingCreds(nlCred, nlOrgId); len(missing) > 0 {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_NL_PARAM, strings.Join(missing, ","))
	}
	return nil
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
	var errRes map[string]any
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

func GetCapellaModelProviders(nlCred, nlOrganizationId string, enabledOnly bool) (any, errors.Error) {
	jwt, err := getCapellaJWTFromSessionsApi(nlCred, false)
	if err != nil {
		return nil, err
	}
	providers, err := getCapellaModelProviders(nlOrganizationId, nlCred, jwt, enabledOnly)
	if err != nil {
		return nil, err
	}
	result := make([]any, len(providers))
	for i, p := range providers {
		models := make([]any, len(p.Models))
		for j, m := range p.Models {
			models[j] = m
		}
		result[i] = map[string]any{
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

// capellaResolveProviderAndModel determines the provider and model to use for
// the request, validating the provider against the modelProviders API.

func capellaResolveProviderAndModel(nlProvider, nlModel, nlOrganizationId, nlCred, jwt string) (provider, model string, err errors.Error) {
	key := capellaModelProvidersCacheKey(nlOrganizationId, nlCred)
	allProviders := capellaCachedModelProviders(key, time.Now())
	if allProviders == nil {
		allProviders, err = getCapellaModelProviders(nlOrganizationId, nlCred, jwt, false)
		if err != nil {
			return "", "", err
		}
		capellaCacheModelProviders(key, allProviders, time.Now())
	}

	return capellaPickProviderAndModel(nlProvider, nlModel, allProviders)
}

// Every request and chat turn resolves its vendor and model against the
// modelProviders list, which changes only when an admin enables, disables or
// adds a vendor. The list is reused for capellaModelProvidersTTL instead of being
// fetched for each one, so a vendor change takes up to that long to be seen. It is
// cached per organization and credential, so a credential never sees a list it
// did not fetch itself. The explicit listing (GetCapellaModelProviders) is not
// cached and always shows the current list.
const capellaModelProvidersTTL = 5 * time.Minute

var capellaModelProvidersCache = util.NewGenCache(capellaCacheLimit)

type capellaModelProvidersCacheEntry struct {
	providers []capellaModelProvider
	fetchedAt time.Time
}

func capellaModelProvidersCacheKey(nlOrganizationId, nlCred string) string {
	return nlOrganizationId + "\x00" + nlCred
}

// capellaCachedModelProviders returns the list cached under key if it was fetched
// less than capellaModelProvidersTTL before now, else nil. The returned slice is
// shared and must not be modified.
func capellaCachedModelProviders(key string, now time.Time) []capellaModelProvider {
	if e, ok := capellaModelProvidersCache.Get(key, nil).(*capellaModelProvidersCacheEntry); ok &&
		now.Sub(e.fetchedAt) < capellaModelProvidersTTL {
		return e.providers
	}
	return nil
}

// capellaCacheModelProviders caches a fetched list. An empty list is not cached:
// it fails every request (E_NL_NO_VENDORS_AVAILABLE), so the first vendor an
// admin adds is picked up by the next request.
func capellaCacheModelProviders(key string, providers []capellaModelProvider, now time.Time) {
	if len(providers) == 0 {
		return
	}
	capellaModelProvidersCache.Add(&capellaModelProvidersCacheEntry{providers: providers, fetchedAt: now}, key, nil)
}

// capellaPickProviderAndModel picks the provider and model from the providers
// listed by the modelProviders API:
//   - no providers listed → error
//   - provider given: unknown/disabled → error
//   - no provider given: prefer openai, else the first enabled provider; none
//     enabled → error
//
// Query does not choose a model. An empty model is sent as is and iQ applies its
// own current default for the vendor, so no default goes stale in Query when a
// vendor retires a model. A given model is matched case-insensitively against the
// provider's listed models only to use the listed spelling; a model that is not
// listed is still sent as given, since iQ forwards models it does not list, and
// a retired one is rejected by the vendor (model_not_found → E_NL_MODEL_NOT_AVAILABLE).

func capellaPickProviderAndModel(nlProvider, nlModel string, allProviders []capellaModelProvider) (
	provider, model string, err errors.Error) {

	if nlModel != "" && nlProvider == "" {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_WITHOUT_VENDOR)
	}

	if len(allProviders) == 0 {
		return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_VENDORS_AVAILABLE)
	}

	var picked *capellaModelProvider
	if nlProvider != "" {
		for i := range allProviders {
			// IDs are lowercase per the api.
			if allProviders[i].ID == strings.ToLower(nlProvider) {
				if !allProviders[i].Enabled {
					return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_ENABLED, nlProvider)
				}
				picked = &allProviders[i]
				break
			}
		}
		if picked == nil {
			return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_SUPPORTED, nlProvider)
		}
	} else {
		// No provider specified: prefer openai, else use the first enabled provider.
		for i := range allProviders {
			if !allProviders[i].Enabled {
				continue
			}
			if allProviders[i].ID == capellaProviderOpenAI {
				picked = &allProviders[i]
				break
			}
			if picked == nil {
				picked = &allProviders[i]
			}
		}
		if picked == nil {
			return "", "", errors.NewNaturalLanguageRequestError(errors.E_NL_NO_VENDORS_ENABLED)
		}
	}

	model = nlModel
	for _, m := range picked.Models {
		if strings.EqualFold(m, nlModel) {
			model = m
			break
		}
	}
	return picked.ID, model, nil
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

func newCapellaSQLPrompt(keyspaceInfo map[string]any, naturalPrompt, summary, hint string, forfts bool,
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

func newCapellaJSUDFPrompt(keyspaceInfo map[string]any, naturalPrompt, summary, hint string, provider string, model string) (*prompt, errors.Error) {
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

	if err := appendJSUDFUserMessage(rv, keyspaceInfo, naturalPrompt, summary, hint); err != nil {
		return nil, err
	}

	return rv, nil
}

// capellaChatTurnVendorModel returns the vendor and model a chat turn resolves
// from. A turn naming neither continues with the chat's own, as PAUSE CHAT does;
// otherwise vendor resolution would fall back to the default (openai) while the
// chat's model was kept. A partial pair is not mixed with the stored one. Only a
// pair a Capella turn stored (Vendor set) is reused: after a direct turn the
// stored model belongs to a natural_config provider, so the turn starts fresh.
// The pair comes from the chat's prompt or, once the prompt was replaced by a
// summary, from the pair kept with it. Caller holds the entry lock.
func capellaChatTurnVendorModel(ce *ChatEntry, nlProvider, nlModel string) (string, string) {
	if nlProvider != "" || nlModel != "" {
		return nlProvider, nlModel
	}
	if p := ce.prompt; p != nil {
		if p.Vendor != "" {
			return p.Vendor, p.CompletionSettings.Model
		}
	} else if ce.vendor != "" {
		return ce.vendor, ce.model
	}
	return "", ""
}

// capellaSetVendorModel applies a turn's resolved vendor and model to the chat's
// prompt. They are set together, even when the model is empty (iQ then applies
// its default), so one vendor's model is never sent to another vendor; the
// temperature follows the model (none is sent with an empty model). The direct
// path's Provider is cleared: the whole prompt is sent to iQ, which must not
// receive a natural_config provider left by an earlier direct turn.
func capellaSetVendorModel(chat *prompt, provider, model string) {
	chat.Provider = ""
	chat.Vendor = provider
	chat.CompletionSettings.Model = model
	chat.CompletionSettings.Temperature = capellaGetTemperatureForModel(provider, model)
}

// capellaSaveVendorModel returns a function that puts chat's current vendor and
// model (with the provider and temperature that go with them) back. An iterative
// turn applies its pair to the chat's own prompt before the request is sent, so a
// turn whose model the vendor rejects (E_NL_MODEL_NOT_AVAILABLE) restores the
// previous pair; otherwise capellaChatTurnVendorModel would reuse the rejected
// pair for every later turn naming neither, and each would fail the same way.
func capellaSaveVendorModel(chat *prompt) func() {
	provider, vendor, model, temperature := chat.Provider, chat.Vendor,
		chat.CompletionSettings.Model, chat.CompletionSettings.Temperature
	return func() {
		chat.Provider, chat.Vendor = provider, vendor
		chat.CompletionSettings.Model, chat.CompletionSettings.Temperature = model, temperature
	}
}

func newCapellaSQLIterativePrompt(chat *prompt, naturalPrompt string, hint string, forfts bool, provider, model string) *prompt {
	capellaSetVendorModel(chat, provider, model)
	return appendSQLIterativeUserMessage(chat, naturalPrompt, hint, forfts)
}

func newCapellaJSUDFIterativePrompt(chat *prompt, naturalPrompt string, hint string, provider, model string) *prompt {
	capellaSetVendorModel(chat, provider, model)
	chat = appendJSUDFIterativeUserMessage(chat, naturalPrompt, hint)
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
		Error   map[string]any `json:"error"`
		Choices []Choice       `json:"choices"`
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
		// The vendor rejected the model (e.g. retired). Capella reports this with
		// HTTP 200 and the vendor's error object in the body. With no model in the
		// request, iQ applied its default for the vendor, and the error says the
		// default model is unavailable.
		if code, _ := chatComplRes.Error["code"].(string); code == "model_not_found" {
			return "", capellaModelNotAvailableError(prompt.CompletionSettings.Model, prompt.Vendor,
				chatComplRes.Error)
		}
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, chatComplRes.Error)
	}
	if len(chatComplRes.Choices) == 0 {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, fmt.Errorf("no message in response"))
	}

	return chatComplRes.Choices[0].Message.Content, nil
}

// capellaModelNotAvailableError reports that the vendor rejected model
// (E_NL_MODEL_NOT_AVAILABLE), with the vendor's error object as the cause. An
// empty model means the request named none and the vendor rejected the default
// iQ applied; the message says so rather than naming an empty model.
func capellaModelNotAvailableError(model, vendor string, cause map[string]any) errors.Error {
	what := "The default model"
	if model != "" {
		what = "Model \"" + model + "\""
	}
	return errors.NewNaturalLanguageRequestError(errors.E_NL_MODEL_NOT_AVAILABLE, what, vendor, cause)
}

func capellaGetTemperatureForModel(provider, model string) float64 {
	switch provider {
	case capellaProviderOpenAI:
		return ai_gateway.TemperatureForModel(model)
	case capellaProviderBedrock:
		return 0
	default:
		return 0
	}
}

func ProcessCapellaRequest(nlCred, nlOrgId, nlProvider, nlModel, nlquery, nlHint string, elems []*algebra.Path, nloutputOpt naturalOutput,
	explain, advise, useKnowledge bool,
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

	keyspaceInfo := make(map[string]any, len(elems))
	inferschema := util.Now()
	keyspaceInfo, _, err = keyspacesInfoForPrompt(keyspaceInfo, elems, context, false, useKnowledge, nlquery)
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
	if parseErr == nil {
		parseErr = validateGeneratedStatement(nlAlgebraStmt, context)
	}
	if parseErr != nil {
		retrytime := util.Now()
		prompt = capellaBuildRetryPrompt(prompt, content, parseErr, true)
		var retryErr error
		for i := 0; i < maxCorrectionRetries; i++ {
			var fatalErr errors.Error
			content, stmt, nlAlgebraStmt, fatalErr, retryErr = capellaRetryRequest(nlCred, nlOrgId, prompt, context, record, nloutputOpt, explain, advise)
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
				prompt = capellaBuildRetryPrompt(prompt, content, retryErr, false)
			}
		}
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			maxCorrectionRetries, content, retryErr)
	}

	return stmt, nlAlgebraStmt, nil
}

// includeInstructions folds _SQLPP_TASK_INSTRUCTIONS into the feedback message.
// Callers pass true only for the first correction round of a request: the rules
// then stay in the message history for every subsequent round without needing to
// be resent.
func capellaBuildRetryPrompt(pmt *prompt, assistantContent string, retryErr error, includeInstructions bool) *prompt {
	return buildRetryPrompt(pmt, assistantContent, correctionFeedback(retryErr, includeInstructions))
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
func capellaRetryRequest(nlCred, nlOrgId string, prompt *prompt, context NaturalContext,
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
	if parseErr == nil {
		parseErr = validateGeneratedStatement(nlAlgebraStmt, context)
	}

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

	nlProvider, nlModel = capellaChatTurnVendorModel(ce, nlProvider, nlModel)
	provider, model, err := capellaResolveProviderAndModel(nlProvider, nlModel, nlOrgId, nlCred, jwt)
	if err != nil {
		return "", nil, err
	}

	var prompt *prompt
	// Set when the turn iterates on the chat's prompt: a first turn builds a new
	// prompt, which is stored only on success, so it has nothing to restore.
	var restoreVendorModel func()
	if ce.prompt == nil {
		keyspaceInfo := make(map[string]any, len(ce.Keyspaces))
		inferschema := util.Now()
		keyspaceInfo, _, err = keyspacesInfoForPrompt(keyspaceInfo, ce.Keyspaces, context, false, ce.useKnowledge, nlquery)
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
		if err != nil {
			return "", nil, err
		}
	} else {
		restoreVendorModel = capellaSaveVendorModel(ce.prompt)
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
		if err.Code() == errors.E_NL_MODEL_NOT_AVAILABLE && restoreVendorModel != nil {
			restoreVendorModel()
		}
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
	if parseErr == nil {
		parseErr = validateGeneratedStatement(nlAlgebraStmt, context)
	}
	if parseErr != nil {
		retrytime := util.Now()
		prompt = capellaBuildRetryPrompt(prompt, content, parseErr, true)
		var retryErr error
		for i := 0; i < maxCorrectionRetries; i++ {
			var fatalErr errors.Error
			content, stmt, nlAlgebraStmt, fatalErr, retryErr = capellaRetryRequest(nlCred, nlOrgId, prompt, context, record, nloutputOpt, explain, advise)
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
				prompt = capellaBuildRetryPrompt(prompt, content, retryErr, false)
			}
		}
		completeConversationPromptLocked(content, ce, prompt)
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			maxCorrectionRetries, content, retryErr)
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
		// Summarization needs both Capella credentials. The request was routed here
		// because at least one is set, so a missing one is a half-configured request
		// and is reported whether or not summarize was given explicitly.
		if err := CapellaCheckCreds(nlCred, nlOrgId); err != nil {
			return err
		}

		// With neither given, summarize with the vendor and model the conversation
		// was using.
		nlprovider, nlmodel = capellaChatTurnVendorModel(ce, nlprovider, nlmodel)

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

	queryContext := datastore.GetDurableQueryContextFor(queryMetadata)
	return persistPausedChat(chatId, ce, queryMetadata, queryContext)
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
	ce.summarized(content)
	return nil
}
