//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Tests for gateway.go - the provider registry, the shared HTTP transport and
// the DoChatCompletion orchestration.
//
// Strategy:
//   - testContext is a self-contained no-op implementation of the gateway's
//     Context plus expression.Context and expression.CurlContext (the same
//     approach as expression's cred_handler tests), carrying a configurable
//     cluster allowlist so the transport's gate can be driven both ways.
//   - The transport (doProviderRequest) and the orchestrator (DoChatCompletion)
//     are exercised end-to-end against httptest servers; the retry backoff is
//     shrunk via the _COMPLETIONS_REQ_BACKOFF_INIT var so tests run in
//     milliseconds.
//   - The credstore path is covered only up to credential resolution failure;
//     resolving real credentials requires cbauth and is covered by
//     expression's own tests.

package ai_gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// ─── Fake execution context ───────────────────────────────────────────────────

// testContext is a no-op implementation of every context interface the gateway
// touches: the gateway's own Context (datastore.Context + datastore.QueryContext),
// expression.Context (asserted by the transport) and expression.CurlContext
// (queried for the cluster allowlist). Tests only set the fields they care
// about.
type testContext struct {
	allowlist map[string]interface{}
	cred      *cbauth.Credential
	credErr   error
}

// Compile-time checks that the stub satisfies everything the gateway needs.
var _ Context = (*testContext)(nil)
var _ expression.Context = (*testContext)(nil)
var _ expression.CurlContext = (*testContext)(nil)

// allowAllContext returns a context whose cluster allowlist permits every URL.
func allowAllContext() *testContext {
	return &testContext{allowlist: map[string]interface{}{util.AllowlistKeyAllAccess: true}}
}

// expression.Context
func (c *testContext) Now() time.Time                 { return time.Time{} }
func (c *testContext) GetTimeout() time.Duration      { return 0 }
func (c *testContext) Credentials() *auth.Credentials { return auth.NewCredentials() }
func (c *testContext) Credential() cbauth.Creds       { return nil }
func (c *testContext) ExternalCredential(_ string) (*cbauth.Credential, error) {
	return c.cred, c.credErr
}
func (c *testContext) DatastoreVersion() string                                { return "" }
func (c *testContext) NewQueryContext(_ string, _ bool) interface{}            { return nil }
func (c *testContext) AdminContext() (interface{}, error)                      { return nil, nil }
func (c *testContext) QueryContext() string                                    { return "" }
func (c *testContext) QueryContextParts() []string                             { return nil }
func (c *testContext) GetTxContext() interface{}                               { return nil }
func (c *testContext) SetTxContext(_ interface{})                              {}
func (c *testContext) Readonly() bool                                          { return false }
func (c *testContext) SetAdvisor()                                             {}
func (c *testContext) IncRecursionCount(_ int) int                             { return 0 }
func (c *testContext) RecursionCount() int                                     { return 0 }
func (c *testContext) StoreValue(_ string, _ interface{})                      {}
func (c *testContext) RetrieveValue(_ string) interface{}                      { return nil }
func (c *testContext) ReleaseValue(_ string)                                   {}
func (c *testContext) SetTracked(_ bool)                                       {}
func (c *testContext) IsTracked() bool                                         { return false }
func (c *testContext) RecordJsCU(_ time.Duration, _ uint64)                    {}
func (c *testContext) SetPreserveProjectionOrder(_ bool) bool                  { return false }
func (c *testContext) IsAdmin() bool                                           { return false }
func (c *testContext) IsPrepared() bool                                        { return false }
func (c *testContext) SanitizeStatement(_ string) (string, value.Value, error) { return "", nil, nil }
func (c *testContext) Parse(_ string) (interface{}, error)                     { return nil, nil }
func (c *testContext) Infer(_ value.Value, _ value.Value) (value.Value, error) { return nil, nil }
func (c *testContext) InferKeyspace(_ interface{}, _ value.Value) (value.Value, error) {
	return nil, nil
}
func (c *testContext) EvaluateStatement(_ string, _ map[string]value.Value, _ value.Values,
	_, _, _ bool, _ string) (value.Value, uint64, error) {
	return nil, 0, nil
}
func (c *testContext) OpenStatement(_ string, _ map[string]value.Value, _ value.Values,
	_, _, _ bool, _ string) (functions.Handle, error) {
	return nil, nil
}

// datastore.Context extras
func (c *testContext) GetScanCap() int64             { return 0 }
func (c *testContext) MaxParallelism() int           { return 1 }
func (c *testContext) Fatal(_ errors.Error)          {}
func (c *testContext) Error(_ errors.Error)          {}
func (c *testContext) Warning(_ errors.Error)        {}
func (c *testContext) GetErrors() []errors.Error     { return nil }
func (c *testContext) GetReqDeadline() time.Time     { return time.Time{} }
func (c *testContext) TenantCtx() tenant.Context     { return nil }
func (c *testContext) SetFirstCreds(_ string)        {}
func (c *testContext) FirstCreds() (string, bool)    { return "", true }
func (c *testContext) RecordFtsRU(_ tenant.Unit)     {}
func (c *testContext) RecordGsiRU(_ tenant.Unit)     {}
func (c *testContext) RecordKvRU(_ tenant.Unit)      {}
func (c *testContext) RecordKvWU(_ tenant.Unit)      {}
func (c *testContext) ScanReportWait() time.Duration { return 0 }
func (c *testContext) SkipKey(_ string) bool         { return false }
func (c *testContext) GetActiveEncryptionKey(_ encryption.KeyDataType) (*encryption.EaRKey, errors.Error) {
	return nil, nil
}

// datastore.QueryContext extras
func (c *testContext) UseReplica() bool                           { return false }
func (c *testContext) Datastore() datastore.Datastore             { return nil }
func (c *testContext) TxDataVal() value.Value                     { return nil }
func (c *testContext) DurabilityLevel() datastore.DurabilityLevel { return datastore.DL_NONE }
func (c *testContext) KvTimeout() time.Duration                   { return 0 }
func (c *testContext) PreserveExpiry() bool                       { return false }
func (c *testContext) IsActive() bool                             { return true }
func (c *testContext) RequestId() string                          { return "" }
func (c *testContext) ErrorLimit() int                            { return 0 }
func (c *testContext) ErrorCount() int                            { return 0 }
func (c *testContext) DurationStyle() util.DurationStyle          { return util.DEFAULT }
func (c *testContext) FormatDuration(_ time.Duration) string      { return "" }
func (c *testContext) UserAgent() string                          { return "" }
func (c *testContext) Users() string                              { return "" }
func (c *testContext) RemoteAddr() string                         { return "" }

// expression.CurlContext
func (c *testContext) GetAllowlist() map[string]interface{}      { return c.allowlist }
func (c *testContext) UrlCredentials(_ string) *auth.Credentials { return nil }
func (c *testContext) DatastoreURL() string                      { return "" }
func (c *testContext) LoadX509KeyPair(_, _ string, _ []byte) (interface{}, error) {
	return nil, nil
}

// logging.Log no-ops
func (c *testContext) Loga(_ logging.Level, _ func() string)            {}
func (c *testContext) Debuga(_ func() string)                           {}
func (c *testContext) Tracea(_ func() string)                           {}
func (c *testContext) Infoa(_ func() string)                            {}
func (c *testContext) Warna(_ func() string)                            {}
func (c *testContext) Errora(_ func() string)                           {}
func (c *testContext) Severea(_ func() string)                          {}
func (c *testContext) Fatala(_ func() string)                           {}
func (c *testContext) Logf(_ logging.Level, _ string, _ ...interface{}) {}
func (c *testContext) Debugf(_ string, _ ...interface{})                {}
func (c *testContext) Tracef(_ string, _ ...interface{})                {}
func (c *testContext) Infof(_ string, _ ...interface{})                 {}
func (c *testContext) Warnf(_ string, _ ...interface{})                 {}
func (c *testContext) Errorf(_ string, _ ...interface{})                {}
func (c *testContext) Severef(_ string, _ ...interface{})               {}
func (c *testContext) Fatalf(_ string, _ ...interface{})                {}

// shrinkBackoff makes transient-failure retries near-instant for the duration
// of a test.
func shrinkBackoff(t *testing.T) {
	t.Helper()
	old := _COMPLETIONS_REQ_BACKOFF_INIT
	_COMPLETIONS_REQ_BACKOFF_INIT = time.Millisecond
	t.Cleanup(func() { _COMPLETIONS_REQ_BACKOFF_INIT = old })
}

// ─── Provider registry ────────────────────────────────────────────────────────

func TestProviderFor_RegisteredProviders(t *testing.T) {
	for _, id := range []string{ProviderOpenAI, ProviderBedrock, ProviderGemini, ProviderSLM} {
		p, err := providerFor(id)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", id, err)
		}
		if p.ID() != id {
			t.Fatalf("%s: registry returned provider %q", id, p.ID())
		}
	}
}

func TestProviderFor_Unknown(t *testing.T) {
	_, err := providerFor("nosuchvendor")
	assertCode(t, err, errors.E_NL_VENDOR_NOT_SUPPORTED)
}

// ─── llmErrCause ──────────────────────────────────────────────────────────────

func TestLLMErrCause(t *testing.T) {
	if got := llmErrCause(nil); got != nil {
		t.Fatalf("empty body: got %v", got)
	}
	if got := llmErrCause([]byte(`{"error":{"message":"boom"}}`)); got == nil ||
		!strings.Contains(got.Error(), "boom") {
		t.Fatalf("JSON body: got %v", got)
	}
	if got := llmErrCause([]byte("plain text failure")); got == nil ||
		got.Error() != "plain text failure" {
		t.Fatalf("text body: got %v", got)
	}
}

// ─── doProviderRequest: allowlist gate ────────────────────────────────────────

func TestDoProviderRequest_EmptyAllowlistDenied(t *testing.T) {
	ctx := &testContext{} // no allowlist configured
	_, err := doProviderRequest(context.Background(), "http://127.0.0.1:1/v1/chat/completions", []byte("{}"),
		&openAIProvider{}, &Config{APIKey: "k"}, ctx)
	assertCode(t, err, errors.E_NL_URL_NOT_ALLOWED)
}

func TestDoProviderRequest_URLNotInAllowlistDenied(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request must not reach the server when the allowlist denies it")
	}))
	defer ts.Close()

	ctx := &testContext{allowlist: map[string]interface{}{
		util.AllowlistKeyAllAccess:   false,
		util.AllowlistKeyAllowedURLs: []interface{}{"https://api.openai.com"},
	}}
	_, err := doProviderRequest(context.Background(), ts.URL+"/v1/chat/completions", []byte("{}"),
		&openAIProvider{}, &Config{APIKey: "k"}, ctx)
	assertCode(t, err, errors.E_NL_URL_NOT_ALLOWED)
}

// ─── doProviderRequest: auth and headers ──────────────────────────────────────

// fakeStaticHeaderProvider reuses the OpenAI provider but declares a static
// header, so the test can assert the transport applies StaticHeaders.
type fakeStaticHeaderProvider struct{ openAIProvider }

func (*fakeStaticHeaderProvider) StaticHeaders() map[string]string {
	return map[string]string{"X-Static-Test": "present"}
}

func TestDoProviderRequest_InlineAuthAndHeaders(t *testing.T) {
	var gotAuth, gotContentType, gotStatic string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotStatic = r.Header.Get("X-Static-Test")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := doProviderRequest(context.Background(), ts.URL, []byte("{}"),
		&fakeStaticHeaderProvider{}, &Config{APIKey: "sk-test"}, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer sk-test" {
		t.Fatalf("Authorization: got %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type: got %q", gotContentType)
	}
	if gotStatic != "present" {
		t.Fatalf("static header: got %q", gotStatic)
	}
}

func TestDoProviderRequest_CredResolutionFailure(t *testing.T) {
	// The stub's ExternalCredential returns (nil, nil): the credential does not
	// exist, so the transport must fail before any request is attempted.
	ctx := allowAllContext()
	_, err := doProviderRequest(context.Background(), "http://127.0.0.1:1/v1/chat/completions", []byte("{}"),
		&openAIProvider{}, &Config{CredId: "no-such-cred"}, ctx)
	assertCode(t, err, errors.E_NL_CRED_RESOLUTION_FAILED)
}

// ─── doProviderRequest: retry policy ──────────────────────────────────────────

func TestDoProviderRequest_RetriesTransientThenSucceeds(t *testing.T) {
	shrinkBackoff(t)
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := doProviderRequest(context.Background(), ts.URL, []byte("{}"),
		&openAIProvider{}, &Config{APIKey: "k"}, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts: got %d, want 3", got)
	}
}

func TestDoProviderRequest_ExhaustsRetriesAndReturnsLastResponse(t *testing.T) {
	shrinkBackoff(t)
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"still overloaded"}`))
	}))
	defer ts.Close()

	resp, err := doProviderRequest(context.Background(), ts.URL, []byte("{}"),
		&openAIProvider{}, &Config{APIKey: "k"}, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	// The final attempt's response is handed back intact so the caller can
	// surface the provider's error detail.
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != _COMPLETIONS_REQ_RETRY {
		t.Fatalf("attempts: got %d, want %d", got, _COMPLETIONS_REQ_RETRY)
	}
}

func TestDoProviderRequest_NonTransientNotRetried(t *testing.T) {
	shrinkBackoff(t)
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	resp, err := doProviderRequest(context.Background(), ts.URL, []byte("{}"),
		&openAIProvider{}, &Config{APIKey: "k"}, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts: got %d, want 1", got)
	}
}

// ─── DoChatCompletion orchestration ───────────────────────────────────────────

const testChatResponse = `{
	"choices": [{"message": {"content": "SELECT 1"}}],
	"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
}`

// openAITestServer stands up an httptest server implementing the standard
// OpenAI URL layout with an unflagging moderations endpoint, returning the
// server plus counters/captures for both endpoints.
func openAITestServer(t *testing.T, flag bool) (ts *httptest.Server,
	chatCalls, modCalls *int32, modInput *[]string) {
	t.Helper()
	var chat, mod int32
	var inputs []string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chat, 1)
		w.Write([]byte(testChatResponse))
	})
	mux.HandleFunc("/v1/moderations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&mod, 1)
		var body struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("moderation body: %v", err)
		}
		inputs = body.Input
		results := make([]map[string]interface{}, len(body.Input))
		for i := range results {
			results[i] = map[string]interface{}{"flagged": flag}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	})
	ts = httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, &chat, &mod, &inputs
}

func TestDoChatCompletion_OpenAIWithModeration(t *testing.T) {
	ts, chatCalls, modCalls, modInput := openAITestServer(t, false)

	cfg := &Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o-2024-05-13",
		APIKey:   "k",
		Endpoint: ts.URL + "/v1/chat/completions",
	}
	req := &Request{
		Model:        cfg.Model,
		InitMessages: []Message{{Role: "system", Content: "you are an expert"}},
		Messages: []Message{
			{Role: "user", Content: "first question"},
			{Role: "assistant", Content: "SELECT 0"},
			{Role: "user", Content: "second question"},
		},
	}

	resp, err := DoChatCompletion(req, cfg, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "SELECT 1" {
		t.Fatalf("content: got %q", resp.Content)
	}
	if resp.Usage != (TokenUsage{Prompt: 10, Completion: 5, Total: 15}) {
		t.Fatalf("usage: got %+v", resp.Usage)
	}

	// All user turns must be screened in a single moderation round trip, with
	// system/assistant turns excluded.
	if got := atomic.LoadInt32(modCalls); got != 1 {
		t.Fatalf("moderation calls: got %d, want 1", got)
	}
	if len(*modInput) != 2 || (*modInput)[0] != "first question" || (*modInput)[1] != "second question" {
		t.Fatalf("moderation input: got %v", *modInput)
	}
	if got := atomic.LoadInt32(chatCalls); got != 1 {
		t.Fatalf("chat calls: got %d, want 1", got)
	}
}

func TestDoChatCompletion_ModerationFlaggedBlocksChat(t *testing.T) {
	ts, chatCalls, _, _ := openAITestServer(t, true)

	cfg := &Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o-2024-05-13",
		APIKey:   "k",
		Endpoint: ts.URL + "/v1/chat/completions",
	}
	req := &Request{
		Model:    cfg.Model,
		Messages: []Message{{Role: "user", Content: "something disallowed"}},
	}

	_, err := DoChatCompletion(req, cfg, allowAllContext())
	assertCode(t, err, errors.E_NL_MODERATION_FLAGGED)
	if got := atomic.LoadInt32(chatCalls); got != 0 {
		t.Fatalf("chat endpoint must not be called after a flag, got %d calls", got)
	}
}

func TestDoChatCompletion_ModerationSkippedOnNonStandardEndpoint(t *testing.T) {
	var chatCalls, otherCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/custom-chat", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chatCalls, 1)
		w.Write([]byte(testChatResponse))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&otherCalls, 1)
		w.WriteHeader(http.StatusNotFound)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cfg := &Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o-2024-05-13",
		APIKey:   "k",
		Endpoint: ts.URL + "/custom-chat",
	}
	req := &Request{
		Model:    cfg.Model,
		Messages: []Message{{Role: "user", Content: "a question"}},
	}

	resp, err := DoChatCompletion(req, cfg, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "SELECT 1" {
		t.Fatalf("content: got %q", resp.Content)
	}
	// No moderations URL is derivable from the custom layout: the request must
	// go straight to the chat endpoint, with no other host contacted.
	if got := atomic.LoadInt32(&chatCalls); got != 1 {
		t.Fatalf("chat calls: got %d, want 1", got)
	}
	if got := atomic.LoadInt32(&otherCalls); got != 0 {
		t.Fatalf("unexpected non-chat requests: %d", got)
	}
}

// "moderation":false opts out for every provider/endpoint: the moderations
// endpoint must not be contacted even when it exists, and the chat proceeds even
// for content the server would otherwise flag.
func TestDoChatCompletion_ModerationDisabledSkipsModeration(t *testing.T) {
	ts, chatCalls, modCalls, _ := openAITestServer(t, true) // flag=true: would block if run

	off := false
	cfg := &Config{
		Provider:   ProviderOpenAI,
		Model:      "gpt-4o-2024-05-13",
		APIKey:     "k",
		Endpoint:   ts.URL + "/v1/chat/completions",
		Moderation: &off,
	}
	req := &Request{
		Model:    cfg.Model,
		Messages: []Message{{Role: "user", Content: "something otherwise disallowed"}},
	}

	resp, err := DoChatCompletion(req, cfg, allowAllContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "SELECT 1" {
		t.Fatalf("content: got %q", resp.Content)
	}
	if got := atomic.LoadInt32(modCalls); got != 0 {
		t.Fatalf("moderation must be skipped when disabled, got %d calls", got)
	}
	if got := atomic.LoadInt32(chatCalls); got != 1 {
		t.Fatalf("chat calls: got %d, want 1", got)
	}
}

// An explicit "moderation":true behaves like the unset default: moderation runs.
func TestDoChatCompletion_ModerationEnabledExplicitly(t *testing.T) {
	ts, _, modCalls, _ := openAITestServer(t, false)

	on := true
	cfg := &Config{
		Provider:   ProviderOpenAI,
		Model:      "gpt-4o-2024-05-13",
		APIKey:     "k",
		Endpoint:   ts.URL + "/v1/chat/completions",
		Moderation: &on,
	}
	req := &Request{Model: cfg.Model, Messages: []Message{{Role: "user", Content: "a question"}}}

	if _, err := DoChatCompletion(req, cfg, allowAllContext()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(modCalls); got != 1 {
		t.Fatalf("moderation must run when explicitly enabled, got %d calls", got)
	}
}

// Moderate maps the moderations endpoint's status precisely: only 404/405/501
// (the endpoint has no moderation route) advise opting out via
// E_NL_MODERATION_UNAVAILABLE; every other failure keeps its real error so a
// transient or auth problem is never mistaken for "endpoint lacks moderation".
func TestModerate_StatusMapping(t *testing.T) {
	shrinkBackoff(t)
	cases := []struct {
		name   string
		status int
		want   errors.ErrorCode
	}{
		{"not_found_no_route", http.StatusNotFound, errors.E_NL_MODERATION_UNAVAILABLE},
		{"method_not_allowed_no_route", http.StatusMethodNotAllowed, errors.E_NL_MODERATION_UNAVAILABLE},
		{"not_implemented_no_route", http.StatusNotImplemented, errors.E_NL_MODERATION_UNAVAILABLE},
		{"service_unavailable_real_failure", http.StatusServiceUnavailable, errors.E_NL_CHATCOMPLETIONS_REQ_FAILED},
		{"unauthorized_real_failure", http.StatusUnauthorized, errors.E_NL_CHATCOMPLETIONS_REQ_FAILED},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/v1/moderations", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
			})
			ts := httptest.NewServer(mux)
			defer ts.Close()

			cfg := &Config{
				Provider: ProviderOpenAI,
				APIKey:   "k",
				Endpoint: ts.URL + "/v1/chat/completions",
			}
			req := &Request{Messages: []Message{{Role: "user", Content: "list hotels"}}}
			assertCode(t, (&openAIProvider{}).Moderate(req, cfg, allowAllContext()), c.want)
		})
	}
}

func TestDoChatCompletion_UnknownProvider(t *testing.T) {
	_, err := DoChatCompletion(&Request{}, &Config{Provider: "nosuchvendor"}, allowAllContext())
	assertCode(t, err, errors.E_NL_VENDOR_NOT_SUPPORTED)
}

// ─── DirectCompleter dispatch ─────────────────────────────────────────────────

// fakeDirectProvider records whether the orchestrator dispatched to Complete
// (the SDK path) instead of the HTTP path.
type fakeDirectProvider struct {
	called bool
}

func (p *fakeDirectProvider) ID() string           { return "testdirect" }
func (p *fakeDirectProvider) DefaultModel() string { return "test-model" }
func (p *fakeDirectProvider) ChatEndpoint(*Config) string {
	return ""
}
func (p *fakeDirectProvider) BuildChatRequest(*Request, *Config) ([]byte, errors.Error) {
	return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
		"HTTP path must not be used")
}
func (p *fakeDirectProvider) ParseChatResponse(int, []byte) (*Response, errors.Error) {
	return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
		"HTTP path must not be used")
}
func (p *fakeDirectProvider) Authorize(*http.Request, *Config) errors.Error { return nil }
func (p *fakeDirectProvider) StaticHeaders() map[string]string              { return nil }
func (p *fakeDirectProvider) AllowsAmbientAuth() bool                       { return true }
func (p *fakeDirectProvider) Moderate(*Request, *Config, Context) errors.Error {
	return nil
}
func (p *fakeDirectProvider) Complete(_ Context, _ *Request, _ *Config) (*Response, errors.Error) {
	p.called = true
	return &Response{Content: "direct"}, nil
}

// A provider implementing DirectCompleter must own its whole round-trip: the
// orchestrator dispatches to Complete before (and instead of) the HTTP path.
func TestDoChatCompletion_DirectCompleterBypassesHTTP(t *testing.T) {
	fake := &fakeDirectProvider{}
	registerProvider(fake)

	t.Cleanup(func() { delete(providers, fake.ID()) })

	resp, err := DoChatCompletion(&Request{}, &Config{Provider: "testdirect"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fake.called {
		t.Fatal("Complete was not dispatched")
	}
	if resp.Content != "direct" {
		t.Fatalf("content: got %q", resp.Content)
	}
}
