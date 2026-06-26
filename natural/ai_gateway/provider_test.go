//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Tests for provider.go - every vendor's codec and policy hooks.
//
// Strategy:
//   - Body builders and response parsers are pure Request/bytes transforms and
//     are tested directly; built bodies are unmarshalled into maps so tests can
//     assert both values and the presence/absence of optional keys.
//   - Bedrock's SDK path is tested through the bedrockConverseAPI interface
//     with a fake client; no AWS access is required. Complete() is covered for
//     the validation paths that return before any network or credential use.
//   - Moderate() paths that perform HTTP are covered in gateway_test.go, which
//     owns the fake context and httptest plumbing; the early-return paths are
//     covered here.

package ai_gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go"

	"github.com/couchbase/query/errors"
)

// unmarshalBody parses a built request body into a generic map so tests can
// assert the presence or absence of optional keys.
func unmarshalBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("built body is not valid JSON: %v\n%s", err, body)
	}
	return m
}

// ─── OpenAI codec ─────────────────────────────────────────────────────────────

func TestBuildOpenAIBody_MessageOrderAndFields(t *testing.T) {
	req := &Request{
		Model:        "gpt-4o-2024-05-13",
		InitMessages: []Message{{Role: "system", Content: "you are an expert"}},
		Messages: []Message{
			{Role: "user", Content: "list hotels"},
			{Role: "assistant", Content: "SELECT 1"},
		},
		Temperature: 0,
		Seed:        1,
	}
	body, err := buildOpenAIBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out openAIChatRequest
	if e := json.Unmarshal(body, &out); e != nil {
		t.Fatalf("unmarshal: %v", e)
	}
	if out.Model != req.Model {
		t.Fatalf("model: got %q", out.Model)
	}
	// InitMessages must precede Messages in the flattened list.
	if len(out.Messages) != 3 || out.Messages[0].Role != "system" ||
		out.Messages[1].Content != "list hotels" || out.Messages[2].Role != "assistant" {
		t.Fatalf("messages not flattened in order: %+v", out.Messages)
	}
	if out.Stream {
		t.Fatal("stream must be false")
	}

	m := unmarshalBody(t, body)
	// Temperature 0 is meaningful for OpenAI (deterministic) and must be sent.
	if _, ok := m["temperature"]; !ok {
		t.Fatal("temperature must be present even when 0")
	}
	// MaxTokens unset means "no engine-side cap": the key must be absent.
	if _, ok := m["max_tokens"]; ok {
		t.Fatal("max_tokens must be omitted when 0")
	}
	if _, ok := m["seed"]; !ok {
		t.Fatal("seed must be present when non-zero")
	}
}

func TestBuildOpenAIBody_MaxTokensSent(t *testing.T) {
	body, err := buildOpenAIBody(&Request{Model: "m", MaxTokens: 512})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := unmarshalBody(t, body)
	if mt, ok := m["max_tokens"].(float64); !ok || int(mt) != 512 {
		t.Fatalf("expected max_tokens 512, got %v", m["max_tokens"])
	}
}

func TestParseOpenAIResponse_HappyPath(t *testing.T) {
	body := []byte(`{
		"choices": [{"message": {"content": "SELECT 1"}}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
	}`)
	resp, err := parseOpenAIResponse(http.StatusOK, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "SELECT 1" {
		t.Fatalf("content: got %q", resp.Content)
	}
	if resp.Usage != (TokenUsage{Prompt: 10, Completion: 5, Total: 15}) {
		t.Fatalf("usage: got %+v", resp.Usage)
	}
}

func TestParseOpenAIResponse_NonOKStatus(t *testing.T) {
	_, err := parseOpenAIResponse(http.StatusUnauthorized, []byte(`{"error":{"message":"bad key"}}`))
	assertCode(t, err, errors.E_NL_CHATCOMPLETIONS_REQ_FAILED)
}

func TestParseOpenAIResponse_MalformedJSON(t *testing.T) {
	_, err := parseOpenAIResponse(http.StatusOK, []byte(`{not json`))
	assertCode(t, err, errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL)
}

func TestParseOpenAIResponse_ErrorField(t *testing.T) {
	_, err := parseOpenAIResponse(http.StatusOK, []byte(`{"error":{"message":"overloaded"}}`))
	assertCode(t, err, errors.E_NL_ERR_CHATCOMPLETIONS_RESP)
}

func TestParseOpenAIResponse_NoChoices(t *testing.T) {
	_, err := parseOpenAIResponse(http.StatusOK, []byte(`{"choices":[]}`))
	assertCode(t, err, errors.E_NL_ERR_CHATCOMPLETIONS_RESP)
}

// ─── OpenAI provider policy ───────────────────────────────────────────────────

func TestOpenAIChatEndpoint(t *testing.T) {
	p := &openAIProvider{}
	if got := p.ChatEndpoint(&Config{}); got != _OPENAI_CHAT_ENDPOINT {
		t.Fatalf("default endpoint: got %q", got)
	}
	if got := p.ChatEndpoint(&Config{Endpoint: "https://proxy/v1/chat/completions"}); got != "https://proxy/v1/chat/completions" {
		t.Fatalf("override endpoint: got %q", got)
	}
}

func TestOpenAIAuthorize_BearerHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", _OPENAI_CHAT_ENDPOINT, nil)
	if err := (&openAIProvider{}).Authorize(req, &Config{APIKey: "sk-123"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer sk-123" {
		t.Fatalf("Authorization: got %q", got)
	}
}

// With no key (permitted on a custom endpoint) no Authorization header may be
// sent: servers treat a bare "Bearer " as malformed rather than anonymous.
func TestOpenAIAuthorize_NoKeyNoHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://proxy/v1/chat/completions", nil)
	if err := (&openAIProvider{}).Authorize(req, &Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := req.Header["Authorization"]; ok {
		t.Fatalf("Authorization header must be absent, got %q", req.Header.Get("Authorization"))
	}
}

func TestModerationsEndpoint(t *testing.T) {
	p := &openAIProvider{}
	cases := []struct {
		chatEndpoint string
		want         string
	}{
		// Default chat endpoint: use the default moderations endpoint.
		{"", _OPENAI_MODERATIONS_ENDPOINT},
		// Standard layout on a custom host: moderate against the same host.
		{"https://proxy.example.com/v1/chat/completions", "https://proxy.example.com/v1/moderations"},
		// Non-standard layout: no moderations URL can be derived. Returning ""
		// (and skipping moderation) prevents user content and the caller's key
		// from being silently posted to the default api.openai.com host.
		{"https://proxy.example.com/custom-chat", ""},
	}
	for _, c := range cases {
		if got := p.moderationsEndpoint(c.chatEndpoint); got != c.want {
			t.Fatalf("moderationsEndpoint(%q): got %q, want %q", c.chatEndpoint, got, c.want)
		}
	}
}

func TestModerate_NoUserContent_NoRequest(t *testing.T) {
	// Only system/assistant turns: Moderate must return nil without any HTTP
	// activity (a nil context would panic if the transport were reached).
	req := &Request{Messages: []Message{
		{Role: "assistant", Content: "SELECT 1"},
		{Role: "user", Content: ""},
	}}
	if err := (&openAIProvider{}).Moderate(req, &Config{}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModerate_UnderivableEndpoint_Skipped(t *testing.T) {
	// A custom endpoint that does not follow the standard layout means no
	// moderations URL: Moderate must skip (nil) without any HTTP activity.
	req := &Request{Messages: []Message{{Role: "user", Content: "list hotels"}}}
	cfg := &Config{Endpoint: "https://proxy.example.com/custom-chat"}
	if err := (&openAIProvider{}).Moderate(req, cfg, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── Gemini codec ─────────────────────────────────────────────────────────────

func TestBuildGeminiBody_RolesAndSystemInstruction(t *testing.T) {
	req := &Request{
		InitMessages: []Message{{Role: "system", Content: "preamble"}},
		Messages: []Message{
			{Role: "system", Content: "embedded system turn"},
			{Role: "user", Content: "list hotels"},
			{Role: "assistant", Content: "SELECT 1"},
		},
	}
	body, err := buildGeminiBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out geminiRequest
	if e := json.Unmarshal(body, &out); e != nil {
		t.Fatalf("unmarshal: %v", e)
	}

	// Both the init message and the embedded system turn must land in
	// systemInstruction, not contents.
	if out.SystemInstruction == nil || len(out.SystemInstruction.Parts) != 2 ||
		out.SystemInstruction.Parts[0].Text != "preamble" ||
		out.SystemInstruction.Parts[1].Text != "embedded system turn" {
		t.Fatalf("systemInstruction: got %+v", out.SystemInstruction)
	}
	if len(out.Contents) != 2 {
		t.Fatalf("contents: got %+v", out.Contents)
	}
	if out.Contents[0].Role != "user" {
		t.Fatalf("user role: got %q", out.Contents[0].Role)
	}
	// Gemini names the assistant role "model".
	if out.Contents[1].Role != "model" || out.Contents[1].Parts[0].Text != "SELECT 1" {
		t.Fatalf("assistant turn: got %+v", out.Contents[1])
	}
}

// Temperature 0 must be omitted entirely so the model runs at the provider
// default, matching Capella's Google path.
func TestBuildGeminiBody_ZeroTemperatureOmitted(t *testing.T) {
	body, err := buildGeminiBody(&Request{
		Messages:    []Message{{Role: "user", Content: "q"}},
		Temperature: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := unmarshalBody(t, body)["generationConfig"]; ok {
		t.Fatal("generationConfig must be absent when temperature and max tokens are unset")
	}
}

func TestBuildGeminiBody_GenerationConfigWhenSet(t *testing.T) {
	body, err := buildGeminiBody(&Request{
		Messages:    []Message{{Role: "user", Content: "q"}},
		Temperature: 0.7,
		MaxTokens:   256,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out geminiRequest
	if e := json.Unmarshal(body, &out); e != nil {
		t.Fatalf("unmarshal: %v", e)
	}
	if out.GenerationConfig == nil || out.GenerationConfig.Temperature == nil ||
		*out.GenerationConfig.Temperature != 0.7 {
		t.Fatalf("temperature: got %+v", out.GenerationConfig)
	}
	if out.GenerationConfig.MaxOutputTokens == nil || *out.GenerationConfig.MaxOutputTokens != 256 {
		t.Fatalf("maxOutputTokens: got %+v", out.GenerationConfig)
	}
}

func TestParseGeminiResponse_ConcatenatesParts(t *testing.T) {
	body := []byte(`{
		"candidates": [{"content": {"parts": [{"text": "SELECT "}, {"text": "1"}], "role": "model"}}],
		"usageMetadata": {"promptTokenCount": 7, "candidatesTokenCount": 3, "totalTokenCount": 10}
	}`)
	resp, err := parseGeminiResponse(http.StatusOK, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "SELECT 1" {
		t.Fatalf("content: got %q", resp.Content)
	}
	if resp.Usage != (TokenUsage{Prompt: 7, Completion: 3, Total: 10}) {
		t.Fatalf("usage: got %+v", resp.Usage)
	}
}

func TestParseGeminiResponse_Errors(t *testing.T) {
	_, err := parseGeminiResponse(http.StatusForbidden, []byte(`{"error":{"message":"denied"}}`))
	assertCode(t, err, errors.E_NL_CHATCOMPLETIONS_REQ_FAILED)

	_, err = parseGeminiResponse(http.StatusOK, []byte(`{not json`))
	assertCode(t, err, errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL)

	_, err = parseGeminiResponse(http.StatusOK, []byte(`{"error":{"message":"quota"}}`))
	assertCode(t, err, errors.E_NL_ERR_CHATCOMPLETIONS_RESP)

	_, err = parseGeminiResponse(http.StatusOK, []byte(`{"candidates":[]}`))
	assertCode(t, err, errors.E_NL_ERR_CHATCOMPLETIONS_RESP)
}

// ─── Gemini provider policy ───────────────────────────────────────────────────

func TestGeminiChatEndpoint(t *testing.T) {
	p := &geminiProvider{}
	got := p.ChatEndpoint(&Config{Model: "gemini-2.5-pro"})
	want := _GEMINI_API_BASE + "gemini-2.5-pro:generateContent"
	if got != want {
		t.Fatalf("default endpoint: got %q, want %q", got, want)
	}
	if got := p.ChatEndpoint(&Config{Endpoint: "https://proxy/gen"}); got != "https://proxy/gen" {
		t.Fatalf("override endpoint: got %q", got)
	}
}

func TestGeminiAuthorize_KeyHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", _GEMINI_API_BASE+"gemini-2.5-pro:generateContent", nil)
	if err := (&geminiProvider{}).Authorize(req, &Config{APIKey: "g-key"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("x-goog-api-key"); got != "g-key" {
		t.Fatalf("x-goog-api-key header: got %q", got)
	}
	if _, ok := req.URL.Query()["key"]; ok {
		t.Fatalf("key must not appear as query param, got URL %q", req.URL.String())
	}
}

// With no key (permitted on a custom endpoint) no header may be set.
func TestGeminiAuthorize_NoKeyNoHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://proxy/generateContent", nil)
	if err := (&geminiProvider{}).Authorize(req, &Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("x-goog-api-key"); got != "" {
		t.Fatalf("x-goog-api-key header must be absent, got %q", got)
	}
}

// ─── SLM provider policy ──────────────────────────────────────────────────────

func TestSLMChatEndpoint_Passthrough(t *testing.T) {
	if got := (&slmProvider{}).ChatEndpoint(&Config{Endpoint: "http://selfhosted/v1/chat/completions"}); got != "http://selfhosted/v1/chat/completions" {
		t.Fatalf("endpoint: got %q", got)
	}
}

func TestSLMBuildChatRequest_RequiresEndpoint(t *testing.T) {
	_, err := (&slmProvider{}).BuildChatRequest(&Request{Messages: []Message{{Role: "user", Content: "q"}}}, &Config{})
	assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
}

func TestSLMAuthorize_BearerHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://selfhosted/v1/chat/completions", nil)
	if err := (&slmProvider{}).Authorize(req, &Config{APIKey: "token"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer token" {
		t.Fatalf("Authorization: got %q", got)
	}
}

// Self-hosted servers often run without auth: with no key configured, no
// Authorization header may be sent.
func TestSLMAuthorize_NoKeyNoHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://selfhosted/v1/chat/completions", nil)
	if err := (&slmProvider{}).Authorize(req, &Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := req.Header["Authorization"]; ok {
		t.Fatalf("Authorization header must be absent, got %q", req.Header.Get("Authorization"))
	}
}

// ─── Bedrock: Converse input mapping ──────────────────────────────────────────

func TestBuildBedrockConverseInput_SystemPromotionAndRoles(t *testing.T) {
	req := &Request{
		Model:        Claude45InferenceProfileID,
		InitMessages: []Message{{Role: "system", Content: "preamble"}},
		Messages: []Message{
			{Role: "system", Content: "embedded system turn"},
			{Role: "user", Content: "list hotels"},
			{Role: "assistant", Content: "SELECT 1"},
		},
	}
	input, err := buildBedrockConverseInput(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *input.ModelId != Claude45InferenceProfileID {
		t.Fatalf("model id: got %q", *input.ModelId)
	}
	// Converse rejects system entries in Messages: both the init message and
	// the embedded system turn must be promoted to System.
	if len(input.System) != 2 {
		t.Fatalf("system prompts: got %d, want 2", len(input.System))
	}
	if len(input.Messages) != 2 ||
		input.Messages[0].Role != types.ConversationRoleUser ||
		input.Messages[1].Role != types.ConversationRoleAssistant {
		t.Fatalf("messages: got %+v", input.Messages)
	}
}

func TestBuildBedrockConverseInput_NoValidMessages(t *testing.T) {
	_, err := buildBedrockConverseInput(&Request{
		Model:    "m",
		Messages: []Message{{Role: "tool", Content: "ignored"}},
	})
	assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
}

// Zero MaxTokens and zero Temperature mean "provider defaults": no
// InferenceConfig at all (same convention as the Gemini builder).
func TestBuildBedrockConverseInput_InferenceConfig(t *testing.T) {
	input, err := buildBedrockConverseInput(&Request{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "q"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.InferenceConfig != nil {
		t.Fatalf("expected nil InferenceConfig, got %+v", input.InferenceConfig)
	}

	input, err = buildBedrockConverseInput(&Request{
		Model:       "m",
		Messages:    []Message{{Role: "user", Content: "q"}},
		MaxTokens:   1024,
		Temperature: 0.5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.InferenceConfig == nil || *input.InferenceConfig.MaxTokens != 1024 ||
		*input.InferenceConfig.Temperature != 0.5 {
		t.Fatalf("inference config: got %+v", input.InferenceConfig)
	}
}

func TestBuildBedrockConverseInput_MaxTokensOutOfRange(t *testing.T) {
	_, err := buildBedrockConverseInput(&Request{
		Model:     "m",
		Messages:  []Message{{Role: "user", Content: "q"}},
		MaxTokens: -1,
	})
	assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
}

// ─── Bedrock: response extraction and error mapping ───────────────────────────

func TestExtractBedrockResponseText(t *testing.T) {
	if got := extractBedrockResponseText(nil); got != "" {
		t.Fatalf("nil output: got %q", got)
	}
	if got := extractBedrockResponseText(&bedrockruntime.ConverseOutput{}); got != "" {
		t.Fatalf("empty output: got %q", got)
	}
	out := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{Value: types.Message{
			Role:    types.ConversationRoleAssistant,
			Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: "SELECT 1"}},
		}},
	}
	if got := extractBedrockResponseText(out); got != "SELECT 1" {
		t.Fatalf("text output: got %q", got)
	}
}

func TestIsRetriableError(t *testing.T) {
	if isRetriableError(nil) {
		t.Fatal("nil must not be retriable")
	}
	if isRetriableError(fmt.Errorf("plain error")) {
		t.Fatal("non-API error must not be retriable")
	}
	for _, code := range []string{"ThrottlingException", "ServiceUnavailableException",
		"InternalServerException", "ModelTimeoutException"} {
		if !isRetriableError(&smithy.GenericAPIError{Code: code}) {
			t.Fatalf("%s must be retriable", code)
		}
	}
	if isRetriableError(&smithy.GenericAPIError{Code: "ValidationException"}) {
		t.Fatal("ValidationException must not be retriable")
	}
}

func TestMapBedrockError(t *testing.T) {
	cases := []struct {
		apiCode string
		want    errors.ErrorCode
	}{
		{"ThrottlingException", errors.E_NL_CHATCOMPLETIONS_REQ_FAILED},
		{"AccessDeniedException", errors.E_NL_CRED_RESOLUTION_FAILED},
		{"InvalidInputException", errors.E_NL_INVALID_NATURAL_CONFIG},
		{"ResourceNotFoundException", errors.E_NL_INVALID_NATURAL_CONFIG},
		{"ValidationException", errors.E_NL_INVALID_NATURAL_CONFIG},
	}
	for _, c := range cases {
		err := mapBedrockError(&smithy.GenericAPIError{Code: c.apiCode, Message: "boom"})
		assertCode(t, err, c.want)
	}
	// Anything unrecognized maps to the generic completion-response error.
	assertCode(t, mapBedrockError(fmt.Errorf("dial tcp: timeout")), errors.E_NL_ERR_CHATCOMPLETIONS_RESP)
}

// ─── Bedrock: client retry via the bedrockConverseAPI seam ────────────────────

// fakeConverseAPI returns the queued errors in order (nil meaning success with
// the canned output) and counts calls.
type fakeConverseAPI struct {
	calls int
	errs  []error
	out   *bedrockruntime.ConverseOutput
}

func (f *fakeConverseAPI) Converse(_ context.Context, _ *bedrockruntime.ConverseInput,
	_ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	i := f.calls
	f.calls++
	if i < len(f.errs) && f.errs[i] != nil {
		return nil, f.errs[i]
	}
	return f.out, nil
}

func TestBedrockClient_RetriesOnceOnTransientError(t *testing.T) {
	fake := &fakeConverseAPI{
		errs: []error{&smithy.GenericAPIError{Code: "ThrottlingException"}, nil},
		out:  &bedrockruntime.ConverseOutput{},
	}
	c := &bedrockClient{client: fake}
	out, err := c.Converse(context.Background(), &bedrockruntime.ConverseInput{})
	if err != nil || out == nil {
		t.Fatalf("expected retried success, got out=%v err=%v", out, err)
	}
	if fake.calls != 2 {
		t.Fatalf("expected 2 calls (original + one retry), got %d", fake.calls)
	}
}

func TestBedrockClient_NoRetryOnPermanentError(t *testing.T) {
	fake := &fakeConverseAPI{
		errs: []error{&smithy.GenericAPIError{Code: "ValidationException"}},
	}
	c := &bedrockClient{client: fake}
	if _, err := c.Converse(context.Background(), &bedrockruntime.ConverseInput{}); err == nil {
		t.Fatal("expected error to surface")
	}
	if fake.calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", fake.calls)
	}
}

// ─── Bedrock: Complete validation paths ───────────────────────────────────────

func TestBedrockComplete_RequiresModel(t *testing.T) {
	p := &bedrockProvider{}
	_, err := p.Complete(nil, &Request{Messages: []Message{{Role: "user", Content: "q"}}}, &Config{})
	assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
}

func TestBedrockComplete_RequiresMessages(t *testing.T) {
	p := &bedrockProvider{}
	_, err := p.Complete(nil, &Request{Model: "m"}, &Config{})
	assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
}

func TestBedrockComplete_RequiresExecutionContext(t *testing.T) {
	p := &bedrockProvider{}
	_, err := p.Complete(nil, &Request{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "q"}},
	}, &Config{})
	assertCode(t, err, errors.E_NL_CRED_RESOLUTION_FAILED)
}

// ─── Provider defaults / registry-facing surface ──────────────────────────────

func TestProviderIDsAndDefaults(t *testing.T) {
	cases := []struct {
		p            Provider
		id           string
		defaultModel string
		ambient      bool
	}{
		{&openAIProvider{}, ProviderOpenAI, GPT4o_2024_05_13, false},
		{&bedrockProvider{}, ProviderBedrock, Claude45InferenceProfileID, true},
		{&geminiProvider{}, ProviderGemini, Gemini25Pro, false},
		{&slmProvider{}, ProviderSLM, SLMDefaultModel, false},
	}
	for _, c := range cases {
		if c.p.ID() != c.id {
			t.Fatalf("ID: got %q, want %q", c.p.ID(), c.id)
		}
		if c.p.DefaultModel() != c.defaultModel {
			t.Fatalf("%s: default model got %q, want %q", c.id, c.p.DefaultModel(), c.defaultModel)
		}
		if c.p.AllowsAmbientAuth() != c.ambient {
			t.Fatalf("%s: AllowsAmbientAuth got %v, want %v", c.id, c.p.AllowsAmbientAuth(), c.ambient)
		}
	}
}
