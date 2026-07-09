// Copyright 2026-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in
// that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.

// Tests for cred_handler.go.
//
// Strategy:
//   - isUrlAllowedForCred and other internal helpers are tested directly
//     (same package).
//   - HandleCred is exercised end-to-end using a lightweight mockCredContext
//     that embeds queryContextImpl and overrides ExternalCredential.
//   - Credential types whose handlers call cbauth.GetTLSConfig (HTTP and
//     Couchbase) are covered only for the error paths that return before that
//     call, keeping the suite free of cbauth daemon dependencies.
//   - Transport types (awsSigV4Transport, azureSASTransport,
//     azureSharedKeyTransport) are tested in isolation via a captureTransport
//     that records the outgoing request and returns a canned 200 OK.

package expression

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/cbauth/cbauthimpl"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
	"github.com/youmark/pkcs8"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// testContextStub is a self-contained no-op implementation of expression.Context
// used as a base for test mock types. All methods satisfy the interface with
// zero/nil returns so tests only need to override the methods they care about.
type testContextStub struct{}

func (*testContextStub) Now() time.Time                 { return time.Time{} }
func (*testContextStub) GetTimeout() time.Duration      { return 0 }
func (*testContextStub) Credentials() *auth.Credentials { return auth.NewCredentials() }
func (*testContextStub) Credential() cbauth.Creds       { return nil }
func (*testContextStub) ExternalCredential(_ string) (*cbauth.Credential, error) {
	return nil, nil
}
func (*testContextStub) DatastoreVersion() string                                { return "" }
func (*testContextStub) NewQueryContext(_ string, _ bool) interface{}            { return nil }
func (*testContextStub) AdminContext() (interface{}, error)                      { return nil, nil }
func (*testContextStub) QueryContext() string                                    { return "" }
func (*testContextStub) QueryContextParts() []string                             { return nil }
func (*testContextStub) GetTxContext() interface{}                               { return nil }
func (*testContextStub) SetTxContext(_ interface{})                              {}
func (*testContextStub) Readonly() bool                                          { return false }
func (*testContextStub) SetAdvisor()                                             {}
func (*testContextStub) IncRecursionCount(_ int) int                             { return 0 }
func (*testContextStub) RecursionCount() int                                     { return 0 }
func (*testContextStub) StoreValue(_ string, _ interface{})                      {}
func (*testContextStub) RetrieveValue(_ string) interface{}                      { return nil }
func (*testContextStub) ReleaseValue(_ string)                                   {}
func (*testContextStub) SetTracked(_ bool)                                       {}
func (*testContextStub) IsTracked() bool                                         { return false }
func (*testContextStub) RecordJsCU(_ time.Duration, _ uint64)                    {}
func (*testContextStub) SetPreserveProjectionOrder(_ bool) bool                  { return false }
func (*testContextStub) IsAdmin() bool                                           { return false }
func (*testContextStub) IsPrepared() bool                                        { return false }
func (*testContextStub) SanitizeStatement(_ string) (string, value.Value, error) { return "", nil, nil }
func (*testContextStub) Parse(_ string) (interface{}, error)                     { return nil, nil }
func (*testContextStub) Infer(_ value.Value, _ value.Value) (value.Value, error) { return nil, nil }
func (*testContextStub) InferKeyspace(_ interface{}, _ value.Value) (value.Value, error) {
	return nil, nil
}
func (*testContextStub) EvaluateStatement(_ string, _ map[string]value.Value, _ value.Values, _, _ bool, _ bool, _ string) (value.Value, uint64, error) {
	return nil, 0, nil
}
func (*testContextStub) OpenStatement(_ string, _ map[string]value.Value, _ value.Values, _, _ bool, _ bool, _ string) (functions.Handle, error) {
	return nil, nil
}

// logging.Log no-ops
func (*testContextStub) Loga(_ logging.Level, _ func() string)            {}
func (*testContextStub) Debuga(_ func() string)                           {}
func (*testContextStub) Tracea(_ func() string)                           {}
func (*testContextStub) Infoa(_ func() string)                            {}
func (*testContextStub) Warna(_ func() string)                            {}
func (*testContextStub) Errora(_ func() string)                           {}
func (*testContextStub) Severea(_ func() string)                          {}
func (*testContextStub) Fatala(_ func() string)                           {}
func (*testContextStub) Logf(_ logging.Level, _ string, _ ...interface{}) {}
func (*testContextStub) Debugf(_ string, _ ...interface{})                {}
func (*testContextStub) Tracef(_ string, _ ...interface{})                {}
func (*testContextStub) Infof(_ string, _ ...interface{})                 {}
func (*testContextStub) Warnf(_ string, _ ...interface{})                 {}
func (*testContextStub) Errorf(_ string, _ ...interface{})                {}
func (*testContextStub) Severef(_ string, _ ...interface{})               {}
func (*testContextStub) Fatalf(_ string, _ ...interface{})                {}

// mockCredContext embeds testContextStub and overrides ExternalCredential so
// tests can inject any *cbauthimpl.Credential value (or a controlled error)
// without a live cbauth service.
type mockCredContext struct {
	testContextStub
	cred *cbauthimpl.Credential
	err  error
}

// ExternalCredential satisfies the Context interface.
// cbauth.Credential is a type alias for cbauthimpl.Credential, so the return
// type matches the interface declaration.
func (m *mockCredContext) ExternalCredential(_ string) (*cbauthimpl.Credential, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cred, nil
}

// allAccessCred returns a *cbauthimpl.Credential with URLWhitelist.AllAccess=true
// and all payload fields nil (callers set the desired payload).
func allAccessCred(id string) *cbauthimpl.Credential {
	return &cbauthimpl.Credential{
		ID: id,
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{AllAccess: true},
			},
		},
	}
}

// mustParseURL is a test helper that panics on bad input.
func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("mustParseURL(%q): %v", raw, err)
	}
	return u
}

// captureTransport records the last round-tripped request and returns a
// canned 200 OK.  This lets tests inspect headers that transports inject.
type captureTransport struct {
	last *http.Request
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.last = req.Clone(req.Context())
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
	}, nil
}

// noopTransport is a stateless RoundTripper used in concurrency tests where
// multiple goroutines share a transport and we do not need to inspect requests.
type noopTransport struct{}

func (n *noopTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
	}, nil
}

// roundTripFunc is an adapter that turns a plain function into an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// init stubs getCbAuthTLSConfig so HTTP and Couchbase handler happy-path tests
// run without a live cbauth daemon.  A zero TLSConfig (nil CipherSuites) is
// equivalent to the OS default, which is correct for unit tests.
func init() {
	getCbAuthTLSConfig = func() (cbauth.TLSConfig, error) {
		return cbauth.TLSConfig{}, nil
	}
}

// mockCurlContext extends mockCredContext and also satisfies CurlContext.
type mockCurlContext struct {
	mockCredContext
}

func (m *mockCurlContext) GetAllowlist() map[string]interface{}      { return nil }
func (m *mockCurlContext) UrlCredentials(_ string) *auth.Credentials { return nil }
func (m *mockCurlContext) DatastoreURL() string                      { return "" }
func (m *mockCurlContext) LoadX509KeyPair(cert, key string, passphrase []byte) (interface{}, error) {
	return tls.Certificate{}, nil
}

// ─── isUrlAllowedForCred ──────────────────────────────────────────────────────

func TestIsUrlAllowedForCred_DiagEvalBlocked(t *testing.T) {
	cred := allAccessCred("x")
	u := mustParseURL(t, "http://localhost/diag/eval")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error for /diag/eval path, got nil")
	}
	if !strings.Contains(err.Error(), "access restricted") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIsUrlAllowedForCred_DiagEvalSubpathBlocked(t *testing.T) {
	cred := allAccessCred("x")
	u := mustParseURL(t, "http://localhost/diag/eval/something")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error for /diag/eval/something path, got nil")
	}
}

func TestIsUrlAllowedForCred_AllAccess(t *testing.T) {
	cred := allAccessCred("x")
	u := mustParseURL(t, "http://example.com/some/api/resource")
	if err := isUrlAllowedForCred(u, cred); err != nil {
		t.Fatalf("expected no error with AllAccess=true, got: %v", err)
	}
}

func TestIsUrlAllowedForCred_AllowedURL(t *testing.T) {
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllAccess:   false,
					AllowedURLs: []string{"http://example.com/api"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api/resource")
	if err := isUrlAllowedForCred(u, cred); err != nil {
		t.Fatalf("expected no error for URL under allowed prefix: %v", err)
	}
}

func TestIsUrlAllowedForCred_DisallowedURL_TakesPrecedence(t *testing.T) {
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllAccess:      false,
					AllowedURLs:    []string{"http://example.com/api"},
					DisallowedURLs: []string{"http://example.com/api/secret"},
				},
			},
		},
	}
	// URL matches both allowed prefix and disallowed prefix; disallowed wins.
	u := mustParseURL(t, "http://example.com/api/secret/data")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error for disallowed URL, got nil")
	}
	if !strings.Contains(err.Error(), "explicitly disallowed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIsUrlAllowedForCred_NotInAllowedList(t *testing.T) {
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllAccess:   false,
					AllowedURLs: []string{"http://example.com/allowed"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/other")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error for URL not in AllowedURLs")
	}
	if !strings.Contains(err.Error(), "not in the allowed list") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIsUrlAllowedForCred_EmptyLists_Denied(t *testing.T) {
	// AllAccess=false with no lists → the allowed-URL branch is never entered,
	// so the function falls through to the final error.
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{AllAccess: false},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/anything")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error when no lists are configured and AllAccess is false")
	}
}

func TestIsUrlAllowedForCred_NilURLWhitelist_Denied(t *testing.T) {
	// A nil URLWhitelist means no guardrails were configured for the credential.
	// URLWhitelist should always be set; a nil value is treated as "deny".
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: nil, // intentionally absent
			},
		},
	}
	u := mustParseURL(t, "http://example.com/anything")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("nil URLWhitelist should deny access, got nil error")
	}
	if !strings.Contains(err.Error(), "no URL guardrails found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIsUrlAllowedForCred_TrailingSlashInRequest(t *testing.T) {
	// AllowedURL "http://example.com/api" should match a request whose path
	// starts with "/api/" — the trailing slash must not block the match.
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllowedURLs: []string{"http://example.com/api"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api/")
	if err := isUrlAllowedForCred(u, cred); err != nil {
		t.Fatalf("trailing slash should not block a URL under the allowed prefix: %v", err)
	}
}

func TestIsUrlAllowedForCred_PortMismatch_Denied(t *testing.T) {
	// Host comparison includes port: "example.com:8080" ≠ "example.com".
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllowedURLs: []string{"http://example.com:8080/api"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api")
	if err := isUrlAllowedForCred(u, cred); err == nil {
		t.Fatal("expected denial when request port does not match allowed URL port")
	}
}

func TestIsUrlAllowedForCred_SchemeMismatch_Denied(t *testing.T) {
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllowedURLs: []string{"https://example.com/api"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api")
	if err := isUrlAllowedForCred(u, cred); err == nil {
		t.Fatal("expected denial when request scheme does not match allowed URL scheme")
	}
}

func TestIsUrlAllowedForCred_PathPrefixNoFalseMatch(t *testing.T) {
	// "/apiextra" must NOT be matched by an allowedURL of "/api".
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllowedURLs: []string{"http://example.com/api"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/apiextra/resource")
	if err := isUrlAllowedForCred(u, cred); err == nil {
		t.Fatal("expected denial: /apiextra should not match prefix /api")
	}
}

func TestIsUrlAllowedForCred_MalformedAllowedURL_ReturnsError(t *testing.T) {
	// A malformed entry in AllowedURLs (invalid percent-encoding breaks url.Parse)
	// must cause isUrlAllowedForCred to return an error rather than silently
	// skip the entry or allow the request.
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					AllowedURLs: []string{"http://example.com/%gg-not-valid"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error for malformed entry in AllowedURLs")
	}
	if !strings.Contains(err.Error(), "invalid allowed_urls") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIsUrlAllowedForCred_MalformedDisallowedURL_ReturnsError(t *testing.T) {
	// A malformed entry in DisallowedURLs (invalid percent-encoding) must
	// surface as an error; the disallowed list is evaluated before the allowed.
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					DisallowedURLs: []string{"http://example.com/%gg-bad"},
					AllowedURLs:    []string{"http://example.com/api"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error for malformed entry in DisallowedURLs")
	}
	if !strings.Contains(err.Error(), "invalid disallowed_urls") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIsUrlAllowedForCred_DisallowedEvaluatedBeforeAllowed_WhenBothMalformed(t *testing.T) {
	// Confirm evaluation order: DisallowedURLs is parsed first.
	// If DisallowedURLs has a malformed entry the error is returned without
	// ever attempting to parse AllowedURLs.
	cred := &cbauthimpl.Credential{
		ID: "x",
		Meta: cbauthimpl.CredentialMeta{
			Guardrails: cbauthimpl.CredentialGuardrails{
				URLWhitelist: &cbauthimpl.URLWhitelist{
					DisallowedURLs: []string{"http://example.com/%gg-disallowed"},
					AllowedURLs:    []string{"http://example.com/%gg-allowed"},
				},
			},
		},
	}
	u := mustParseURL(t, "http://example.com/api")
	err := isUrlAllowedForCred(u, cred)
	if err == nil {
		t.Fatal("expected error")
	}
	// The error must reference the disallowed list, not the allowed list.
	if !strings.Contains(err.Error(), "invalid disallowed_urls") {
		t.Errorf("expected error to reference the disallowed URL entry, got: %v", err)
	}
}

// ─── HandleCred — fetch / guardrail error cases ───────────────────────────────

func TestHandleCred_CredentialNotFound(t *testing.T) {
	ctx := &mockCredContext{cred: nil} // ExternalCredential returns (nil, nil)
	u := mustParseURL(t, "http://example.com/api")
	_, _, err := HandleCred(u, "missing-id", ctx)
	if err == nil {
		t.Fatal("expected error when credential is nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_ContextReturnsError(t *testing.T) {
	ctx := &mockCredContext{err: fmt.Errorf("store unavailable")}
	u := mustParseURL(t, "http://example.com/api")
	_, _, err := HandleCred(u, "my-cred", ctx)
	if err == nil {
		t.Fatal("expected error when context returns error")
	}
	if !strings.Contains(err.Error(), "store unavailable") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_DiagEvalBlocked(t *testing.T) {
	cred := allAccessCred("x")
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://localhost/diag/eval")
	_, _, err := HandleCred(u, "x", ctx)
	if err == nil {
		t.Fatal("expected error for /diag/eval")
	}
	if !strings.Contains(err.Error(), "access restricted") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── AWS credential ───────────────────────────────────────────────────────────

func TestHandleCred_AWS_HappyPath(t *testing.T) {
	cred := allAccessCred("aws-cred")
	cred.AWS = &cbauthimpl.AWSPayload{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region:          "us-east-1",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://s3.amazonaws.com/bucket")

	client, header, err := HandleCred(u, "aws-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(header) != 0 {
		t.Errorf("expected empty header for AWS, got: %v", header)
	}
	if _, ok := client.Transport.(*awsSigV4Transport); !ok {
		t.Errorf("expected *awsSigV4Transport, got %T", client.Transport)
	}
}

func TestHandleCred_AWS_SigV4Fields(t *testing.T) {
	cred := allAccessCred("aws-cred")
	cred.AWS = &cbauthimpl.AWSPayload{
		AccessKeyID:     "AKID",
		SecretAccessKey: "SECRET",
		SessionToken:    "TOKEN",
		Region:          "eu-west-1",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://s3.amazonaws.com/bucket")

	client, _, err := HandleCred(u, "aws-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr := client.Transport.(*awsSigV4Transport)
	if tr.sessionToken != "TOKEN" {
		t.Errorf("sessionToken not propagated: %q", tr.sessionToken)
	}
	if tr.region != "eu-west-1" {
		t.Errorf("region not propagated: %q", tr.region)
	}
}

func TestHandleCred_AWS_EndpointOverride(t *testing.T) {
	cred := allAccessCred("aws-cred")
	cred.AWS = &cbauthimpl.AWSPayload{
		AccessKeyID:     "AKID",
		SecretAccessKey: "SECRET",
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://s3.amazonaws.com/bucket/key")

	client, _, err := HandleCred(u, "aws-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr := client.Transport.(*awsSigV4Transport)
	if tr.endpoint == nil {
		t.Fatal("endpoint should be set on transport")
	}
	if tr.endpoint.Host != "localhost:9000" {
		t.Errorf("endpoint host = %q, want %q", tr.endpoint.Host, "localhost:9000")
	}

	// Verify RoundTrip rewrites the URL to the endpoint.
	var capturedHost string
	tr.base = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedHost = r.URL.Host
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket/key", nil)
	client.Transport.(*awsSigV4Transport).RoundTrip(req) //nolint:errcheck
	if capturedHost != "localhost:9000" {
		t.Errorf("RoundTrip sent request to %q, want %q", capturedHost, "localhost:9000")
	}
}

func TestHandleCred_AWS_InvalidEndpoint(t *testing.T) {
	cred := allAccessCred("aws-cred")
	cred.AWS = &cbauthimpl.AWSPayload{
		AccessKeyID:     "AKID",
		SecretAccessKey: "SECRET",
		Region:          "us-east-1",
		Endpoint:        "://bad url",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://s3.amazonaws.com/bucket")

	_, _, err := HandleCred(u, "aws-cred", ctx)
	if err == nil {
		t.Fatal("expected error for invalid AWS endpoint")
	}
	if !strings.Contains(err.Error(), "invalid endpoint") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── Azure Shared Key credential ──────────────────────────────────────────────

func TestHandleCred_AzureShared_HappyPath(t *testing.T) {
	cred := allAccessCred("az-shared")
	cred.AzureShared = &cbauthimpl.AzureSharedPayload{
		AccountName: "myaccount",
		AccountKey:  base64.StdEncoding.EncodeToString([]byte("some-account-key-bytes!!")),
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://myaccount.blob.core.windows.net/container")

	client, header, err := HandleCred(u, "az-shared", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(header) != 0 {
		t.Errorf("expected empty header for AzureShared, got: %v", header)
	}
	if _, ok := client.Transport.(*azureSharedKeyTransport); !ok {
		t.Errorf("expected *azureSharedKeyTransport, got %T", client.Transport)
	}
}

func TestHandleCred_AzureShared_InvalidBase64Key(t *testing.T) {
	cred := allAccessCred("az-shared")
	cred.AzureShared = &cbauthimpl.AzureSharedPayload{
		AccountName: "myaccount",
		AccountKey:  "not!!valid$$base64",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://myaccount.blob.core.windows.net/container")

	_, _, err := HandleCred(u, "az-shared", ctx)
	if err == nil {
		t.Fatal("expected error for invalid base64 accountKey")
	}
	if !strings.Contains(err.Error(), "invalid accountKey base64") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_AzureShared_EndpointOverride(t *testing.T) {
	cred := allAccessCred("az-shared")
	cred.AzureShared = &cbauthimpl.AzureSharedPayload{
		AccountName: "devstoreaccount1",
		AccountKey:  base64.StdEncoding.EncodeToString([]byte("some-account-key-bytes!!")),
		Endpoint:    "http://127.0.0.1:10000",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://devstoreaccount1.blob.core.windows.net/container/blob")

	client, _, err := HandleCred(u, "az-shared", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr := client.Transport.(*azureSharedKeyTransport)
	if tr.endpoint == nil {
		t.Fatal("endpoint should be set on transport")
	}
	if tr.endpoint.Host != "127.0.0.1:10000" {
		t.Errorf("endpoint host = %q, want %q", tr.endpoint.Host, "127.0.0.1:10000")
	}

	// Verify RoundTrip rewrites the URL to the endpoint.
	var capturedHost string
	tr.base = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedHost = r.URL.Host
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	req, _ := http.NewRequest("GET", "http://devstoreaccount1.blob.core.windows.net/container/blob", nil)
	client.Transport.(*azureSharedKeyTransport).RoundTrip(req) //nolint:errcheck
	if capturedHost != "127.0.0.1:10000" {
		t.Errorf("RoundTrip sent request to %q, want %q", capturedHost, "127.0.0.1:10000")
	}
}

func TestHandleCred_AzureShared_InvalidEndpoint(t *testing.T) {
	cred := allAccessCred("az-shared")
	cred.AzureShared = &cbauthimpl.AzureSharedPayload{
		AccountName: "myaccount",
		AccountKey:  base64.StdEncoding.EncodeToString([]byte("some-account-key-bytes!!")),
		Endpoint:    "://bad url",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://myaccount.blob.core.windows.net/container")

	_, _, err := HandleCred(u, "az-shared", ctx)
	if err == nil {
		t.Fatal("expected error for invalid AzureShared endpoint")
	}
	if !strings.Contains(err.Error(), "invalid endpoint") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── Azure SAS credential ─────────────────────────────────────────────────────

// newTestCertAndKey generates a self-signed RSA certificate for use in tests.
// Returns PEM-encoded cert+key bundle and the raw key.
// newTestCertKeyPEMs generates a self-signed RSA certificate and returns the
// cert PEM and key PEM as separate strings, matching the credstore layout.
func newTestCertKeyPEMs(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	var cert, key strings.Builder
	pem.Encode(&cert, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})                           //nolint:errcheck
	pem.Encode(&key, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}) //nolint:errcheck
	return cert.String(), key.String()
}

// newTestEncryptedKeyPEMs generates a self-signed certificate and returns the
// cert PEM and a PKCS#8-encrypted key PEM protected by passphrase.
func newTestEncryptedKeyPEMs(t *testing.T, passphrase []byte) (certPEM, keyPEM string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-encrypted"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	encKeyDER, err := pkcs8.MarshalPrivateKey(k, passphrase, nil)
	if err != nil {
		t.Fatalf("pkcs8.MarshalPrivateKey: %v", err)
	}
	var cert, key strings.Builder
	pem.Encode(&cert, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})            //nolint:errcheck
	pem.Encode(&key, &pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: encKeyDER}) //nolint:errcheck
	return cert.String(), key.String()
}

func newTestCertAndKey(t *testing.T) (pemBundle string, key *rsa.PrivateKey) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	var buf strings.Builder
	pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})                            //nolint:errcheck
	pem.Encode(&buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}) //nolint:errcheck
	return buf.String(), k
}

// ─── parseAzureCertAndKey ─────────────────────────────────────────────────────

func TestParseAzureCertAndKey_Valid(t *testing.T) {
	pemBundle, wantKey := newTestCertAndKey(t)

	cert, key, err := parseAzureCertAndKey(pemBundle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert == nil {
		t.Fatal("cert is nil")
	}
	if key.N.Cmp(wantKey.N) != 0 {
		t.Error("returned key does not match the generated key")
	}
}

func TestParseAzureCertAndKey_NoCertificate(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	var buf strings.Builder
	pem.Encode(&buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}) //nolint:errcheck

	_, _, err := parseAzureCertAndKey(buf.String())
	if err == nil || !strings.Contains(err.Error(), "no certificate") {
		t.Errorf("expected 'no certificate' error, got: %v", err)
	}
}

func TestParseAzureCertAndKey_NoPrivateKey(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	var buf strings.Builder
	pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}) //nolint:errcheck

	_, _, err := parseAzureCertAndKey(buf.String())
	if err == nil || !strings.Contains(err.Error(), "no private key") {
		t.Errorf("expected 'no private key' error, got: %v", err)
	}
}

func TestParseAzureCertAndKey_InvalidCertBlock(t *testing.T) {
	var buf strings.Builder
	pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: []byte("not a certificate")}) //nolint:errcheck
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	pem.Encode(&buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}) //nolint:errcheck

	_, _, err := parseAzureCertAndKey(buf.String())
	if err == nil || !strings.Contains(err.Error(), "failed to parse certificate") {
		t.Errorf("expected certificate parse error, got: %v", err)
	}
}

// ─── buildAzureADClientAssertion ─────────────────────────────────────────────

func TestBuildAzureADClientAssertion_ValidJWTStructure(t *testing.T) {
	pemBundle, _ := newTestCertAndKey(t)
	cert, key, err := parseAzureCertAndKey(pemBundle)
	if err != nil {
		t.Fatalf("parseAzureCertAndKey: %v", err)
	}

	jwt, err := buildAzureADClientAssertion("my-client-id", "https://login.microsoftonline.com/tenant/oauth2/v2.0/token", cert, key)
	if err != nil {
		t.Fatalf("buildAzureADClientAssertion: %v", err)
	}

	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d: %q", len(parts), jwt)
	}
	// Each part must be valid base64url.
	for i, p := range parts {
		if _, err := base64.RawURLEncoding.DecodeString(p); err != nil {
			t.Errorf("part %d is not valid base64url: %v", i, err)
		}
	}
	// Header must contain "RS256" and "x5t".
	header, _ := base64.RawURLEncoding.DecodeString(parts[0])
	if !strings.Contains(string(header), "RS256") {
		t.Errorf("header missing RS256: %s", header)
	}
	if !strings.Contains(string(header), "x5t") {
		t.Errorf("header missing x5t: %s", header)
	}
}

// ─── applyAzureADPayload — dispatch ──────────────────────────────────────────

// Verifies that when Certificate is set, the error path goes through the cert
// branch (parseAzureCertAndKey), not the client-secret branch.
func TestHandleCred_AzureAD_CertBranch_InvalidPEM(t *testing.T) {
	cred := allAccessCred("az-ad")
	cred.AzureAD = &cbauthimpl.AzureADPayload{
		TenantID:    "tenant",
		ClientID:    "client",
		Certificate: "not valid PEM",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "https://storage.azure.com/container")

	_, _, err := HandleCred(u, "az-ad", ctx)
	if err == nil {
		t.Fatal("expected error for invalid certificate PEM")
	}
	// Error must come from parseAzureCertAndKey, not client-secret path.
	if !strings.Contains(err.Error(), "no certificate") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── Azure SAS credential ─────────────────────────────────────────────────────

func TestHandleCred_AzureSAS_HappyPath(t *testing.T) {
	cred := allAccessCred("az-sas")
	cred.AzureSAS = &cbauthimpl.AzureSASPayload{
		AccountName:           "myaccount",
		SharedAccessSignature: "sv=2021-06-08&ss=b&sp=r&sig=abc",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://myaccount.blob.core.windows.net/container")

	client, header, err := HandleCred(u, "az-sas", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(header) != 0 {
		t.Errorf("expected empty header for AzureSAS, got: %v", header)
	}
	if _, ok := client.Transport.(*azureSASTransport); !ok {
		t.Errorf("expected *azureSASTransport, got %T", client.Transport)
	}
}

func TestHandleCred_AzureSAS_EndpointOverride(t *testing.T) {
	cred := allAccessCred("az-sas")
	cred.AzureSAS = &cbauthimpl.AzureSASPayload{
		AccountName:           "devstoreaccount1",
		SharedAccessSignature: "sv=2021-06-08&sig=abc",
		Endpoint:              "http://127.0.0.1:10000",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://devstoreaccount1.blob.core.windows.net/container/blob")

	client, _, err := HandleCred(u, "az-sas", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr := client.Transport.(*azureSASTransport)
	if tr.endpoint == nil {
		t.Fatal("endpoint should be set on transport")
	}
	if tr.endpoint.Host != "127.0.0.1:10000" {
		t.Errorf("endpoint host = %q, want %q", tr.endpoint.Host, "127.0.0.1:10000")
	}

	// Verify RoundTrip rewrites the URL to the endpoint.
	var capturedHost string
	tr.base = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedHost = r.URL.Host
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	req, _ := http.NewRequest("GET", "http://devstoreaccount1.blob.core.windows.net/container/blob", nil)
	client.Transport.(*azureSASTransport).RoundTrip(req) //nolint:errcheck
	if capturedHost != "127.0.0.1:10000" {
		t.Errorf("RoundTrip sent request to %q, want %q", capturedHost, "127.0.0.1:10000")
	}
}

func TestHandleCred_AzureSAS_InvalidEndpoint(t *testing.T) {
	cred := allAccessCred("az-sas")
	cred.AzureSAS = &cbauthimpl.AzureSASPayload{
		AccountName:           "myaccount",
		SharedAccessSignature: "sig=abc",
		Endpoint:              "://bad url",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://myaccount.blob.core.windows.net/container")

	_, _, err := HandleCred(u, "az-sas", ctx)
	if err == nil {
		t.Fatal("expected error for invalid AzureSAS endpoint")
	}
	if !strings.Contains(err.Error(), "invalid endpoint") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_AzureSAS_DiagEvalEndpoint(t *testing.T) {
	cred := allAccessCred("az-sas")
	cred.AzureSAS = &cbauthimpl.AzureSASPayload{
		AccountName:           "myaccount",
		SharedAccessSignature: "sig=abc",
		Endpoint:              "http://localhost:8091/diag/eval",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://myaccount.blob.core.windows.net/container")

	_, _, err := HandleCred(u, "az-sas", ctx)
	if err == nil {
		t.Fatal("expected error for /diag/eval endpoint")
	}
	if !strings.Contains(err.Error(), "restricted") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── GCP credential ───────────────────────────────────────────────────────────

func TestHandleCred_GCP_HMACMode(t *testing.T) {
	cred := allAccessCred("gcp-hmac")
	cred.GCP = &cbauthimpl.GCPPayload{
		AccessKeyID:     "GOOG1MYACCESSKEY",
		SecretAccessKey: "MySecretAccessKey",
		Region:          "us-central1",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://storage.googleapis.com/bucket")

	client, header, err := HandleCred(u, "gcp-hmac", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(header) != 0 {
		t.Errorf("expected empty header for GCP HMAC, got: %v", header)
	}
	if _, ok := client.Transport.(*awsSigV4Transport); !ok {
		t.Errorf("expected *awsSigV4Transport for GCP HMAC, got %T", client.Transport)
	}
}

func TestHandleCred_GCP_HMACMode_EndpointOverride(t *testing.T) {
	cred := allAccessCred("gcp-hmac")
	cred.GCP = &cbauthimpl.GCPPayload{
		AccessKeyID:     "GOOG1MYACCESSKEY",
		SecretAccessKey: "MySecretAccessKey",
		Region:          "us-central1",
		Endpoint:        "http://localhost:4443",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://storage.googleapis.com/bucket/obj")

	client, _, err := HandleCred(u, "gcp-hmac", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr := client.Transport.(*awsSigV4Transport)
	if tr.endpoint == nil {
		t.Fatal("endpoint should be set on GCP HMAC transport")
	}
	if tr.endpoint.Host != "localhost:4443" {
		t.Errorf("endpoint host = %q, want %q", tr.endpoint.Host, "localhost:4443")
	}

	// Verify RoundTrip rewrites the URL to the endpoint.
	var capturedHost string
	tr.base = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedHost = r.URL.Host
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	req, _ := http.NewRequest("GET", "http://storage.googleapis.com/bucket/obj", nil)
	client.Transport.(*awsSigV4Transport).RoundTrip(req) //nolint:errcheck
	if capturedHost != "localhost:4443" {
		t.Errorf("RoundTrip sent request to %q, want %q", capturedHost, "localhost:4443")
	}
}

func TestHandleCred_GCP_EmptyPayload_Error(t *testing.T) {
	cred := allAccessCred("gcp-empty")
	cred.GCP = &cbauthimpl.GCPPayload{} // neither jsonCredentials nor accessKeyId
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://storage.googleapis.com/bucket")

	_, _, err := HandleCred(u, "gcp-empty", ctx)
	if err == nil {
		t.Fatal("expected error for empty GCP payload")
	}
	if !strings.Contains(err.Error(), "either jsonCredentials or accessKeyId") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_GCP_HMACMode_DiagEvalEndpoint(t *testing.T) {
	cred := allAccessCred("gcp-hmac")
	cred.GCP = &cbauthimpl.GCPPayload{
		AccessKeyID:     "GOOG1MYACCESSKEY",
		SecretAccessKey: "MySecretAccessKey",
		Region:          "us-central1",
		Endpoint:        "http://localhost:8091/diag/eval",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://storage.googleapis.com/bucket")

	_, _, err := HandleCred(u, "gcp-hmac", ctx)
	if err == nil {
		t.Fatal("expected error for /diag/eval endpoint")
	}
	if !strings.Contains(err.Error(), "restricted") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_GCP_ServiceAccount_DiagEvalTokenURI(t *testing.T) {
	// A GCP service-account JSON with a malicious token_uri pointing to
	// /diag/eval must be blocked before any HTTP request is made.
	maliciousJSON := `{
		"private_key": "ignored",
		"client_email": "sa@project.iam.gserviceaccount.com",
		"token_uri": "http://localhost:8091/diag/eval"
	}`
	cred := allAccessCred("gcp-sa")
	cred.GCP = &cbauthimpl.GCPPayload{
		JSONCredentials: maliciousJSON,
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "https://storage.googleapis.com/bucket")

	_, _, err := HandleCred(u, "gcp-sa", ctx)
	if err == nil {
		t.Fatal("expected error for /diag/eval token_uri, got nil")
	}
	if !strings.Contains(err.Error(), "restricted") {
		t.Errorf("expected 'restricted' in error, got: %v", err)
	}
}

// ─── HTTP credential — early error cases (before cbauth.GetTLSConfig) ─────────

func TestHandleCred_HTTP_CertWithoutPrivateKey(t *testing.T) {
	cred := allAccessCred("http-cred")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme:  "basic",
		Username:    "user",
		Certificate: "some-cert-pem",
		// PrivateKey intentionally absent
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-cred", ctx)
	if err == nil {
		t.Fatal("expected error for certificate without privateKey")
	}
	if !strings.Contains(err.Error(), "both certificate and privateKey are required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_HTTP_PrivateKeyWithoutCert(t *testing.T) {
	cred := allAccessCred("http-cred")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "basic",
		Username:   "user",
		PrivateKey: "some-private-key-pem",
		// Certificate intentionally absent
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-cred", ctx)
	if err == nil {
		t.Fatal("expected error for privateKey without certificate")
	}
	if !strings.Contains(err.Error(), "both certificate and privateKey are required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_HTTP_InvalidRootCertificate(t *testing.T) {
	cred := allAccessCred("http-cred")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme:      "basic",
		Username:        "user",
		RootCertificate: "this is not valid PEM data",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-cred", ctx)
	if err == nil {
		t.Fatal("expected error for invalid rootCertificate PEM")
	}
	if !strings.Contains(err.Error(), "invalid rootCertificate") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestHandleCred_HTTP_MTLS_InvalidPEM verifies that invalid PEM content in
// Certificate or PrivateKey produces an error (no CurlContext required).
func TestHandleCred_HTTP_MTLS_InvalidPEM(t *testing.T) {
	cred := allAccessCred("http-cred")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme:  "mtls",
		Certificate: "not-valid-pem",
		PrivateKey:  "not-valid-pem",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-cred", ctx)
	if err == nil {
		t.Fatal("expected error for invalid mTLS PEM content")
	}
}

// ─── HTTP credential — happy paths (unlocked by getCbAuthTLSConfig stub) ──────

func TestHandleCred_HTTP_Basic_HappyPath(t *testing.T) {
	cred := allAccessCred("http-basic")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "basic",
		Username:   "alice",
		Password:   "s3cr3t",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-basic", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:s3cr3t"))
	if got := header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestHandleCred_HTTP_Basic_MissingUsername(t *testing.T) {
	cred := allAccessCred("http-basic")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "basic",
		// Username deliberately absent
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-basic", ctx)
	if err == nil {
		t.Fatal("expected error for basic auth without username")
	}
	if !strings.Contains(err.Error(), "username required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_HTTP_Bearer_Authorization_HappyPath(t *testing.T) {
	cred := allAccessCred("http-bearer")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "bearer",
		Token:      "my-jwt-token",
		HeaderName: "Authorization",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-bearer", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := header.Get("Authorization"); got != "Bearer my-jwt-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-jwt-token")
	}
}

func TestHandleCred_HTTP_Bearer_CustomHeader_HappyPath(t *testing.T) {
	cred := allAccessCred("http-bearer")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "bearer",
		Token:      "my-api-key",
		HeaderName: "X-API-Key",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-bearer", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := header.Get("X-API-Key"); got != "my-api-key" {
		t.Errorf("X-API-Key = %q, want %q", got, "my-api-key")
	}
	if auth := header.Get("Authorization"); auth != "" {
		t.Errorf("Authorization should be absent for custom headerName, got: %q", auth)
	}
}

func TestHandleCred_HTTP_Bearer_AuthorizationCaseInsensitive(t *testing.T) {
	cred := allAccessCred("http-bearer")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "bearer",
		Token:      "my-jwt-token",
		HeaderName: "authorization", // lowercase must still get the Bearer scheme
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-bearer", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := header.Get("Authorization"); got != "Bearer my-jwt-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-jwt-token")
	}
}

func TestHandleCred_HTTP_Bearer_MissingToken(t *testing.T) {
	cred := allAccessCred("http-bearer")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "bearer",
		HeaderName: "Authorization",
		// Token absent
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-bearer", ctx)
	if err == nil {
		t.Fatal("expected error for bearer auth without token")
	}
	if !strings.Contains(err.Error(), "token required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_HTTP_Bearer_MissingHeaderName_DefaultsToAuthorization(t *testing.T) {
	cred := allAccessCred("http-bearer")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "bearer",
		Token:      "my-token",
		// HeaderName absent — defaults to the Authorization header (RFC 6750).
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-bearer", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := header.Get("Authorization"); got != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-token")
	}
}

func TestHandleCred_HTTP_UnsupportedAuthScheme(t *testing.T) {
	cred := allAccessCred("http-cred")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme: "digest", // not a supported scheme
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-cred", ctx)
	if err == nil {
		t.Fatal("expected error for unsupported authScheme")
	}
	if !strings.Contains(err.Error(), "unsupported authScheme") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleCred_HTTP_MTLS_HappyPath(t *testing.T) {
	certPEM, keyPEM := newTestCertKeyPEMs(t)
	cred := allAccessCred("http-mtls")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme:  "mtls",
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-mtls", ctx)
	if err != nil {
		t.Fatalf("unexpected error for mTLS happy path: %v", err)
	}
	// mTLS has no Authorization header — auth happens at the TLS handshake.
	if auth := header.Get("Authorization"); auth != "" {
		t.Errorf("mTLS should produce no Authorization header, got: %q", auth)
	}
}

func TestHandleCred_HTTP_MTLS_EncryptedKey(t *testing.T) {
	passphrase := []byte("test-passphrase-123")
	certPEM, keyPEM := newTestEncryptedKeyPEMs(t, passphrase)
	cred := allAccessCred("http-mtls-enc")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme:  "mtls",
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		Passphrase:  string(passphrase),
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "http-mtls-enc", ctx)
	if err != nil {
		t.Fatalf("unexpected error for mTLS with encrypted key: %v", err)
	}
	if auth := header.Get("Authorization"); auth != "" {
		t.Errorf("mTLS should produce no Authorization header, got: %q", auth)
	}
}

func TestHandleCred_HTTP_MTLS_EncryptedKey_WrongPassphrase(t *testing.T) {
	certPEM, keyPEM := newTestEncryptedKeyPEMs(t, []byte("correct-passphrase"))
	cred := allAccessCred("http-mtls-enc")
	cred.HTTP = &cbauthimpl.HTTPPayload{
		AuthScheme:  "mtls",
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		Passphrase:  "wrong-passphrase",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "http-mtls-enc", ctx)
	if err == nil {
		t.Fatal("expected error for wrong passphrase")
	}
}

// ─── Couchbase credential — early error cases (before cbauth.GetTLSConfig) ────

// Note: the Couchbase cert-without-privateKey error path is not tested here
// because applyCouchbasePayload calls cbauth.GetTLSConfig() before it reaches
// the cert/key validation, and cbauth is not initialised in unit tests.

func TestHandleCred_Couchbase_InvalidRootCertificate(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType:  "full",
		RootCertificate: "not valid PEM",
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "cb-cred", ctx)
	if err == nil {
		t.Fatal("expected error for invalid Couchbase rootCertificate")
	}
	if !strings.Contains(err.Error(), "invalid rootCertificate") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── Couchbase credential — happy paths (unlocked by getCbAuthTLSConfig stub) ─

func TestHandleCred_Couchbase_Basic_HappyPath(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "full",
		Username:       "admin",
		Password:       "password",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "cb-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error for Couchbase basic auth: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:password"))
	if got := header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestHandleCred_Couchbase_MTLS_HappyPath(t *testing.T) {
	certPEM, keyPEM := newTestCertKeyPEMs(t)
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "full",
		Certificate:    certPEM,
		PrivateKey:     keyPEM,
	}
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, header, err := HandleCred(u, "cb-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error for Couchbase mTLS: %v", err)
	}
	// mTLS: no Authorization header; auth is at TLS handshake level.
	if auth := header.Get("Authorization"); auth != "" {
		t.Errorf("Couchbase mTLS should produce no Authorization header, got: %q", auth)
	}
}

func TestHandleCred_Couchbase_CertWithoutPrivateKey(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "full",
		Certificate:    "some-cert",
		// PrivateKey absent
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "cb-cred", ctx)
	if err == nil {
		t.Fatal("expected error for Couchbase certificate without privateKey")
	}
	if !strings.Contains(err.Error(), "both certificate and privateKey are required") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── Couchbase credential — EncryptionType branches ──────────────────────────

// "none": no TLS configuration changes; Authorization header is still set.
func TestHandleCred_Couchbase_EncryptionNone_BasicAuth(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "none",
		Username:       "alice",
		Password:       "secret",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "http://example.com/api")

	client, header, err := HandleCred(u, "cb-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error for Couchbase encryptionType=none: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	if got := header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
	// InsecureSkipVerify must NOT be set for "none" — plain HTTP URL means TLS
	// is not negotiated at all; modifying TLS config would be misleading.
	tr := client.Transport.(*http.Transport)
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("encryptionType=none should not set InsecureSkipVerify")
	}
}

// "half": TLS transport but peer certificate is NOT verified.
func TestHandleCred_Couchbase_EncryptionHalf_SetsInsecureSkipVerify(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "half",
		Username:       "bob",
		Password:       "pass",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "https://example.com/api")

	client, header, err := HandleCred(u, "cb-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error for Couchbase encryptionType=half: %v", err)
	}
	tr := client.Transport.(*http.Transport)
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("encryptionType=half must set InsecureSkipVerify=true")
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("bob:pass"))
	if got := header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

// "full": peer certificate IS verified (InsecureSkipVerify remains false).
func TestHandleCred_Couchbase_EncryptionFull_DoesNotSkipVerify(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "full",
		Username:       "carol",
		Password:       "pw",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "https://example.com/api")

	client, _, err := HandleCred(u, "cb-cred", ctx)
	if err != nil {
		t.Fatalf("unexpected error for Couchbase encryptionType=full: %v", err)
	}
	tr := client.Transport.(*http.Transport)
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("encryptionType=full must not set InsecureSkipVerify")
	}
}

// Unknown EncryptionType must be rejected.
func TestHandleCred_Couchbase_EncryptionUnknown_ReturnsError(t *testing.T) {
	cred := allAccessCred("cb-cred")
	cred.Couchbase = &cbauthimpl.CouchbasePayload{
		EncryptionType: "bogus",
		Username:       "user",
	}
	ctx := &mockCurlContext{mockCredContext: mockCredContext{cred: cred}}
	u := mustParseURL(t, "https://example.com/api")

	_, _, err := HandleCred(u, "cb-cred", ctx)
	if err == nil {
		t.Fatal("expected error for unknown Couchbase encryptionType")
	}
	if !strings.Contains(err.Error(), "unsupported encryptionType") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── No recognized payload ────────────────────────────────────────────────────

func TestHandleCred_NoPayload(t *testing.T) {
	cred := allAccessCred("empty-cred")
	// All payload fields remain nil.
	ctx := &mockCredContext{cred: cred}
	u := mustParseURL(t, "http://example.com/api")

	_, _, err := HandleCred(u, "empty-cred", ctx)
	if err == nil {
		t.Fatal("expected error for credential with no recognized payload")
	}
	if !strings.Contains(err.Error(), "no recognized payload type") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── Pure helper: awsServiceFromHost ─────────────────────────────────────────

func TestAWSServiceFromHost(t *testing.T) {
	cases := []struct {
		host    string
		service string
	}{
		{"bedrock-runtime.us-east-1.amazonaws.com", "bedrock-runtime"},
		{"s3.amazonaws.com", "s3"},
		{"s3.amazonaws.com:443", "s3"},
		{"singlepart", "singlepart"},
		// empty host: SplitN("", ".", 2) returns [""], parts[0] is "",
		// but awsServiceFromHost falls back to "execute-api"
		{"", "execute-api"},
	}
	for _, tc := range cases {
		got := awsServiceFromHost(tc.host)
		if got != tc.service {
			t.Errorf("awsServiceFromHost(%q) = %q, want %q", tc.host, got, tc.service)
		}
	}
}

// ─── Pure helper: awsCanonicalURI ─────────────────────────────────────────────

func TestAWSCanonicalURI(t *testing.T) {
	cases := []struct {
		rawURL string
		want   string
	}{
		{"http://example.com", "/"}, // empty path → "/"
		{"http://example.com/foo/bar", "/foo/bar"},
		{"http://example.com/foo%20bar", "/foo%20bar"},
	}
	for _, tc := range cases {
		u, _ := url.Parse(tc.rawURL)
		if got := awsCanonicalURI(u); got != tc.want {
			t.Errorf("awsCanonicalURI(%q) = %q, want %q", tc.rawURL, got, tc.want)
		}
	}
}

// ─── Pure helper: awsCanonicalQueryString ─────────────────────────────────────

func TestAWSCanonicalQueryString_NoParams(t *testing.T) {
	u, _ := url.Parse("http://example.com/path")
	if got := awsCanonicalQueryString(u); got != "" {
		t.Errorf("expected empty canonical query string, got %q", got)
	}
}

func TestAWSCanonicalQueryString_Sorted(t *testing.T) {
	u, _ := url.Parse("http://example.com/path?z=last&a=first&m=middle")
	got := awsCanonicalQueryString(u)
	// Keys must appear in lexicographic order.
	if !strings.HasPrefix(got, "a=first&m=middle&z=last") {
		t.Errorf("canonical query string not sorted: %q", got)
	}
}

// ─── Pure helper: hmacSHA256 / sha256Hex ──────────────────────────────────────

func TestHmacSHA256_Deterministic(t *testing.T) {
	key := []byte("Jefe")
	data := []byte("what do ya want for nothing?")
	r1 := hmacSHA256(key, data)
	r2 := hmacSHA256(key, data)
	if len(r1) != 32 {
		t.Fatalf("expected 32-byte HMAC-SHA256, got %d", len(r1))
	}
	for i := range r1 {
		if r1[i] != r2[i] {
			t.Fatal("hmacSHA256 is non-deterministic")
		}
	}
}

func TestSHA256HexBytes_EmptyInput(t *testing.T) {
	// The SHA-256 digest of the empty byte slice is a well-known constant.
	const wantEmptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := sha256HexBytes([]byte{}); got != wantEmptyHash {
		t.Errorf("sha256HexBytes(empty) = %q, want %q", got, wantEmptyHash)
	}
}

func TestSHA256HexString_MatchesBytes(t *testing.T) {
	s := "hello world"
	if sha256HexString(s) != sha256HexBytes([]byte(s)) {
		t.Error("sha256HexString and sha256HexBytes disagree for the same input")
	}
}

// ─── awsSigV4Transport.RoundTrip ──────────────────────────────────────────────

func TestAWSSigV4Transport_RoundTrip_Headers(t *testing.T) {
	capture := &captureTransport{}
	transport := &awsSigV4Transport{
		base:            capture,
		accessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		region:          "us-east-1",
	}
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/my-bucket/my-key", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	authHdr := capture.last.Header.Get("Authorization")
	if !strings.HasPrefix(authHdr, "AWS4-HMAC-SHA256 ") {
		t.Errorf("Authorization header missing AWS4-HMAC-SHA256 prefix: %q", authHdr)
	}
	if capture.last.Header.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date header not set")
	}
	if capture.last.Header.Get("X-Amz-Content-Sha256") == "" {
		t.Error("X-Amz-Content-Sha256 header not set")
	}
}

func TestAWSSigV4Transport_RoundTrip_SessionToken(t *testing.T) {
	capture := &captureTransport{}
	transport := &awsSigV4Transport{
		base:            capture,
		accessKeyID:     "AKID",
		secretAccessKey: "SECRET",
		sessionToken:    "my-session-token",
		region:          "us-east-1",
	}
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	if got := capture.last.Header.Get("X-Amz-Security-Token"); got != "my-session-token" {
		t.Errorf("X-Amz-Security-Token = %q, want %q", got, "my-session-token")
	}
}

func TestAWSSigV4Transport_RoundTrip_NoSessionToken(t *testing.T) {
	capture := &captureTransport{}
	transport := &awsSigV4Transport{
		base:            capture,
		accessKeyID:     "AKID",
		secretAccessKey: "SECRET",
		region:          "us-east-1",
		// sessionToken deliberately empty
	}
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	if capture.last.Header.Get("X-Amz-Security-Token") != "" {
		t.Error("X-Amz-Security-Token should not be set when sessionToken is empty")
	}
}

func TestAWSSigV4Transport_ServiceFieldOverridesHostInference(t *testing.T) {
	capture := &captureTransport{}
	transport := &awsSigV4Transport{
		base:            capture,
		accessKeyID:     "AKID",
		secretAccessKey: "SECRET",
		region:          "us-east-1",
		service:         "s3",
		endpoint:        mustParseURL(t, "http://localhost:9000"),
	}
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket/key", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	// The Authorization header's credential scope must use "s3" as the service,
	// not "localhost" which would be inferred from the endpoint host.
	authHdr := capture.last.Header.Get("Authorization")
	if !strings.Contains(authHdr, "/s3/aws4_request") {
		t.Errorf("credential scope should use 's3' service, got: %q", authHdr)
	}
	// Request was sent to the endpoint host.
	if capture.last.URL.Host != "localhost:9000" {
		t.Errorf("request host = %q, want %q", capture.last.URL.Host, "localhost:9000")
	}
}

// ─── azureSASTransport.RoundTrip ──────────────────────────────────────────────

func TestAzureSASTransport_RoundTrip_ParamsAppended(t *testing.T) {
	capture := &captureTransport{}
	transport := &azureSASTransport{
		base: capture,
		sas:  "sv=2021-06-08&ss=b&sp=r&sig=abc123",
	}
	req, _ := http.NewRequest("GET", "http://myaccount.blob.core.windows.net/container/blob", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	q := capture.last.URL.Query()
	if q.Get("sv") != "2021-06-08" {
		t.Errorf("sv param = %q, want %q", q.Get("sv"), "2021-06-08")
	}
	if q.Get("sig") != "abc123" {
		t.Errorf("sig param = %q, want %q", q.Get("sig"), "abc123")
	}
}

func TestAzureSASTransport_RoundTrip_MergesExistingParams(t *testing.T) {
	capture := &captureTransport{}
	transport := &azureSASTransport{
		base: capture,
		sas:  "sig=fromSAS",
	}
	// The original request already carries a query param.
	req, _ := http.NewRequest("GET", "http://myaccount.blob.core.windows.net/file?existing=yes", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	q := capture.last.URL.Query()
	if q.Get("existing") != "yes" {
		t.Errorf("existing param should be preserved, got: %v", q)
	}
	if q.Get("sig") != "fromSAS" {
		t.Errorf("SAS sig param not merged, got: %v", q)
	}
}

// ─── azureSharedKeyTransport.RoundTrip ────────────────────────────────────────

func TestAzureSharedKeyTransport_RoundTrip_Headers(t *testing.T) {
	// Use a 32-byte key so Base64 decoding in the real handler won't fail.
	keyBytes := []byte("0123456789012345678901234567890X") // 32 bytes
	capture := &captureTransport{}
	transport := &azureSharedKeyTransport{
		base:        capture,
		accountName: "myaccount",
		accountKey:  keyBytes,
	}
	req, _ := http.NewRequest("GET", "http://myaccount.blob.core.windows.net/container/blob", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	authHdr := capture.last.Header.Get("Authorization")
	if !strings.HasPrefix(authHdr, "SharedKey myaccount:") {
		t.Errorf("Authorization header malformed: %q", authHdr)
	}
	if capture.last.Header.Get("x-ms-date") == "" {
		t.Error("x-ms-date header not set")
	}
	if got := capture.last.Header.Get("x-ms-version"); got != "2024-11-04" {
		t.Errorf("x-ms-version = %q, want %q", got, "2024-11-04")
	}
}

// ─── Concurrency ──────────────────────────────────────────────────────────────

// TestAWSSigV4Transport_Concurrent verifies that awsSigV4Transport is safe to
// share across goroutines.  The transport is stateless after construction:
// every RoundTrip clones the incoming request and only reads struct fields.
// Run with -race to surface any data races not caught by functional assertions.
func TestAWSSigV4Transport_Concurrent(t *testing.T) {
	const goroutines = 20
	transport := &awsSigV4Transport{
		base:            &noopTransport{},
		accessKeyID:     "AKID",
		secretAccessKey: "SECRET",
		region:          "us-east-1",
	}
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket/key", nil)
			if _, err := transport.RoundTrip(req); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent RoundTrip error: %v", err)
	}
}

// ─── azureSharedKeyTransport concurrency ─────────────────────────────────────

func TestAzureSharedKeyTransport_Concurrent(t *testing.T) {
	const goroutines = 20
	keyBytes := []byte("0123456789012345678901234567890X")
	transport := &azureSharedKeyTransport{
		base:        &noopTransport{},
		accountName: "myaccount",
		accountKey:  keyBytes,
	}
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "http://myaccount.blob.core.windows.net/container/blob", nil)
			if _, err := transport.RoundTrip(req); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent RoundTrip error: %v", err)
	}
}

// ─── azureSASTransport concurrency ───────────────────────────────────────────

func TestAzureSASTransport_Concurrent(t *testing.T) {
	const goroutines = 20
	transport := &azureSASTransport{
		base: &noopTransport{},
		sas:  "sv=2021-06-08&ss=b&sp=r&sig=abc",
	}
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "http://myaccount.blob.core.windows.net/container/blob", nil)
			if _, err := transport.RoundTrip(req); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent RoundTrip error: %v", err)
	}
}

// ─── Azure Shared Key signature correctness ──────────────────────────────────

func TestAzureSharedKeySignature_KnownInput(t *testing.T) {
	// Verify the signature function with a deterministic input. We build a
	// request with known headers and compare the HMAC-SHA256 output against
	// a value computed by hand.
	key := []byte("0123456789ABCDEF0123456789ABCDEF") // 32 bytes
	accountName := "devaccount"

	req, _ := http.NewRequest("GET", "http://devaccount.blob.core.windows.net/mycontainer/myblob", nil)
	req.Header.Set("x-ms-date", "Thu, 01 Jan 2026 00:00:00 GMT")
	req.Header.Set("x-ms-version", "2024-11-04")

	sig, err := azureSharedKeySignature(key, accountName, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reproduce the expected signature:
	// Method = GET, all standard headers absent (empty), x-ms- headers sorted,
	// canonical resource = /devaccount/mycontainer/myblob
	stringToSign := strings.Join([]string{
		"GET", // method
		"",    // Content-Encoding
		"",    // Content-Language
		"",    // Content-Length (absent)
		"",    // Content-MD5
		"",    // Content-Type
		"",    // Date
		"",    // If-Modified-Since
		"",    // If-Match
		"",    // If-None-Match
		"",    // If-Unmodified-Since
		"",    // Range
		"x-ms-date:Thu, 01 Jan 2026 00:00:00 GMT\nx-ms-version:2024-11-04",
		"/devaccount/mycontainer/myblob",
	}, "\n")

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if sig != want {
		t.Errorf("signature mismatch:\n  got:  %s\n  want: %s", sig, want)
	}
}

func TestAzureSharedKeySignature_WithQueryParams(t *testing.T) {
	key := []byte("testkey1234567890testkey12345678")
	accountName := "myaccount"

	req, _ := http.NewRequest("GET", "http://myaccount.blob.core.windows.net/container/blob?comp=metadata&timeout=30", nil)
	req.Header.Set("x-ms-date", "Mon, 01 Jan 2024 00:00:00 GMT")
	req.Header.Set("x-ms-version", "2024-11-04")

	sig, err := azureSharedKeySignature(key, accountName, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Query params must appear in the canonical resource, sorted and lowercased.
	stringToSign := strings.Join([]string{
		"GET", "", "", "", "", "", "", "", "", "", "", "",
		"x-ms-date:Mon, 01 Jan 2024 00:00:00 GMT\nx-ms-version:2024-11-04",
		"/myaccount/container/blob\ncomp:metadata\ntimeout:30",
	}, "\n")

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if sig != want {
		t.Errorf("signature mismatch:\n  got:  %s\n  want: %s", sig, want)
	}
}

func TestAzureSharedKeySignature_ContentLengthZero_OmittedFromSignature(t *testing.T) {
	key := []byte("testkey1234567890testkey12345678")
	accountName := "myaccount"

	req, _ := http.NewRequest("PUT", "http://myaccount.blob.core.windows.net/c/b", nil)
	req.Header.Set("Content-Length", "0")
	req.Header.Set("x-ms-date", "Mon, 01 Jan 2024 00:00:00 GMT")
	req.Header.Set("x-ms-version", "2024-11-04")

	sig, err := azureSharedKeySignature(key, accountName, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Content-Length "0" must be treated as empty in the string to sign.
	stringToSign := strings.Join([]string{
		"PUT", "", "", "", "", "", "", "", "", "", "", "",
		"x-ms-date:Mon, 01 Jan 2024 00:00:00 GMT\nx-ms-version:2024-11-04",
		"/myaccount/c/b",
	}, "\n")

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if sig != want {
		t.Errorf("Content-Length=0 not omitted; signature mismatch:\n  got:  %s\n  want: %s", sig, want)
	}
}

// ─── AWS SigV4 with request body ─────────────────────────────────────────────

func TestAWSSigV4Transport_RoundTrip_WithBody(t *testing.T) {
	capture := &captureTransport{}
	transport := &awsSigV4Transport{
		base:            capture,
		accessKeyID:     "AKID",
		secretAccessKey: "SECRET",
		region:          "us-east-1",
	}
	body := "hello world"
	req, _ := http.NewRequest("POST", "http://s3.amazonaws.com/bucket/key", strings.NewReader(body))
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// X-Amz-Content-Sha256 must be the SHA-256 of the body, not of empty.
	wantHash := sha256HexBytes([]byte(body))
	if got := capture.last.Header.Get("X-Amz-Content-Sha256"); got != wantHash {
		t.Errorf("X-Amz-Content-Sha256 = %q, want %q", got, wantHash)
	}

	// The body must be preserved on the forwarded request.
	if capture.last.Body == nil {
		t.Fatal("forwarded request body is nil")
	}
	forwarded, _ := io.ReadAll(capture.last.Body)
	if string(forwarded) != body {
		t.Errorf("forwarded body = %q, want %q", string(forwarded), body)
	}

	// Authorization must contain the method.
	authHdr := capture.last.Header.Get("Authorization")
	if !strings.HasPrefix(authHdr, "AWS4-HMAC-SHA256 ") {
		t.Errorf("Authorization header missing prefix: %q", authHdr)
	}
}

func TestAWSSigV4Transport_RoundTrip_EmptyBody_Hash(t *testing.T) {
	capture := &captureTransport{}
	transport := &awsSigV4Transport{
		base:            capture,
		accessKeyID:     "AKID",
		secretAccessKey: "SECRET",
		region:          "us-east-1",
	}
	req, _ := http.NewRequest("GET", "http://s3.amazonaws.com/bucket/key", nil)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	// SHA-256 of empty body is the well-known constant.
	const emptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := capture.last.Header.Get("X-Amz-Content-Sha256"); got != emptyHash {
		t.Errorf("X-Amz-Content-Sha256 = %q, want %q", got, emptyHash)
	}
}

// ─── Token fetch happy paths (httptest) ──────────────────────────────────────

func TestFetchAzureADToken_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST with the expected form fields.
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q", r.FormValue("grant_type"))
		}
		if r.FormValue("client_id") != "my-client" {
			t.Errorf("client_id = %q", r.FormValue("client_id"))
		}
		if r.FormValue("client_secret") != "my-secret" {
			t.Errorf("client_secret = %q", r.FormValue("client_secret"))
		}
		if r.FormValue("scope") != "https://storage.azure.com/.default" {
			t.Errorf("scope = %q", r.FormValue("scope"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-ad-token-12345",
		})
	}))
	defer srv.Close()

	token, err := fetchAzureADToken("my-tenant", "my-client", "my-secret", srv.URL, "https://storage.azure.com/.default", _tokenFetchTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-ad-token-12345" {
		t.Errorf("token = %q, want %q", token, "test-ad-token-12345")
	}
}

func TestFetchAzureADToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal failure"))
	}))
	defer srv.Close()

	_, err := fetchAzureADToken("tenant", "client", "secret", srv.URL, "scope", _tokenFetchTimeout)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchAzureADToken_TokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_client",
			"error_description": "bad credentials",
		})
	}))
	defer srv.Close()

	_, err := fetchAzureADToken("tenant", "client", "wrong", srv.URL, "scope", _tokenFetchTimeout)
	if err == nil {
		t.Fatal("expected error for token error response")
	}
	if !strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchAzureADTokenWithCert_HappyPath(t *testing.T) {
	pemBundle, _ := newTestCertAndKey(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.FormValue("client_assertion_type") != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
			t.Errorf("client_assertion_type = %q", r.FormValue("client_assertion_type"))
		}
		assertion := r.FormValue("client_assertion")
		parts := strings.Split(assertion, ".")
		if len(parts) != 3 {
			t.Errorf("client_assertion is not a 3-part JWT: %q", assertion)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "cert-token-99",
		})
	}))
	defer srv.Close()

	token, err := fetchAzureADTokenWithCert("my-tenant", "my-client", pemBundle, srv.URL, "https://storage.azure.com/.default", _tokenFetchTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "cert-token-99" {
		t.Errorf("token = %q, want %q", token, "cert-token-99")
	}
}

func TestFetchAzureManagedToken_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Must be a GET with Metadata header.
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Metadata") != "true" {
			t.Errorf("Metadata header = %q, want %q", r.Header.Get("Metadata"), "true")
		}

		q := r.URL.Query()
		if q.Get("api-version") != "2018-02-01" {
			t.Errorf("api-version = %q", q.Get("api-version"))
		}
		if q.Get("resource") != "https://storage.azure.com" {
			t.Errorf("resource = %q", q.Get("resource"))
		}
		if q.Get("client_id") != "my-managed-id" {
			t.Errorf("client_id = %q", q.Get("client_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "managed-token-42",
		})
	}))
	defer srv.Close()

	token, err := fetchAzureManagedToken("my-managed-id", srv.URL+"/metadata/identity/oauth2/token", "https://storage.azure.com", _tokenFetchTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "managed-token-42" {
		t.Errorf("token = %q, want %q", token, "managed-token-42")
	}
}

func TestFetchAzureManagedToken_NoManagedIdentityID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// client_id should NOT be present when managedIdentityID is empty.
		if q.Get("client_id") != "" {
			t.Errorf("client_id should be absent, got %q", q.Get("client_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "system-assigned-token",
		})
	}))
	defer srv.Close()

	token, err := fetchAzureManagedToken("", srv.URL+"/metadata/identity/oauth2/token", "https://storage.azure.com", _tokenFetchTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "system-assigned-token" {
		t.Errorf("token = %q, want %q", token, "system-assigned-token")
	}
}

func TestFetchAzureManagedToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("not on Azure"))
	}))
	defer srv.Close()

	_, err := fetchAzureManagedToken("", srv.URL+"/metadata/identity/oauth2/token", "https://storage.azure.com", _tokenFetchTimeout)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("unexpected error: %v", err)
	}
}

// newTestPKCS8Key generates an RSA key and returns it as a PKCS#8 PEM string.
func newTestPKCS8Key(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey: %v", err)
	}
	var buf strings.Builder
	pem.Encode(&buf, &pem.Block{Type: "PRIVATE KEY", Bytes: der}) //nolint:errcheck
	return k, buf.String()
}

func TestFetchGCPServiceAccountToken_HappyPath(t *testing.T) {
	key, keyPEM := newTestPKCS8Key(t)
	_ = key // the server doesn't verify the JWT signature in this test

	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.FormValue("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Errorf("grant_type = %q", r.FormValue("grant_type"))
		}
		assertion := r.FormValue("assertion")
		parts := strings.Split(assertion, ".")
		if len(parts) != 3 {
			t.Errorf("assertion is not a 3-part JWT: %q", assertion)
		}
		// Verify the claims contain the expected audience (token_uri).
		claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			t.Fatalf("decode claims: %v", err)
		}
		if !strings.Contains(string(claimsJSON), srvURL) {
			t.Errorf("JWT aud should contain the token URI, got claims: %s", claimsJSON)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "gcp-token-777",
		})
	}))
	defer srv.Close()
	srvURL = srv.URL

	sa := gcpServiceAccountKey{
		PrivateKey:  keyPEM,
		ClientEmail: "test@project.iam.gserviceaccount.com",
		TokenURI:    srv.URL,
	}
	token, err := fetchGCPServiceAccountToken(sa, _tokenFetchTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "gcp-token-777" {
		t.Errorf("token = %q, want %q", token, "gcp-token-777")
	}
}

func TestFetchGCPServiceAccountToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad jwt"))
	}))
	defer srv.Close()

	key, keyPEM := newTestPKCS8Key(t)
	_ = key
	sa := gcpServiceAccountKey{
		PrivateKey:  keyPEM,
		ClientEmail: "test@project.iam.gserviceaccount.com",
		TokenURI:    srv.URL,
	}
	_, err := fetchGCPServiceAccountToken(sa, _tokenFetchTimeout)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchGCPServiceAccountToken_DefaultTokenURI(t *testing.T) {
	// Verify that an empty TokenURI defaults to the Google endpoint.
	// We can't actually call it, but we can verify the JWT's aud claim.
	key, keyPEM := newTestPKCS8Key(t)
	_ = key
	sa := gcpServiceAccountKey{
		PrivateKey:  keyPEM,
		ClientEmail: "test@project.iam.gserviceaccount.com",
		TokenURI:    "", // should default to https://oauth2.googleapis.com/token
	}
	// This will fail with a network or HTTP error, but the error message
	// should come from the token exchange step, not a parse/key issue.
	_, err := fetchGCPServiceAccountToken(sa, _tokenFetchTimeout)
	if err == nil {
		t.Fatal("expected error when calling real Google endpoint")
	}
	// The error must originate from the HTTP call or response check, not key parsing.
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("unexpected error (expected token-related error on real endpoint): %v", err)
	}
}

// ─── UnwrapTransport tests ───────────────────────────────────────────────────

func TestUnwrapTransport_PlainHTTPTransport(t *testing.T) {
	inner := &http.Transport{}
	got := UnwrapTransport(inner)
	if got != inner {
		t.Fatal("expected same *http.Transport back")
	}
}

func TestUnwrapTransport_AWSWrapper(t *testing.T) {
	inner := &http.Transport{}
	wrapped := &awsSigV4Transport{base: inner}
	got := UnwrapTransport(wrapped)
	if got != inner {
		t.Fatal("expected inner *http.Transport from awsSigV4Transport")
	}
}

func TestUnwrapTransport_AzureSharedWrapper(t *testing.T) {
	inner := &http.Transport{}
	wrapped := &azureSharedKeyTransport{base: inner}
	got := UnwrapTransport(wrapped)
	if got != inner {
		t.Fatal("expected inner *http.Transport from azureSharedKeyTransport")
	}
}

func TestUnwrapTransport_AzureSASWrapper(t *testing.T) {
	inner := &http.Transport{}
	wrapped := &azureSASTransport{base: inner}
	got := UnwrapTransport(wrapped)
	if got != inner {
		t.Fatal("expected inner *http.Transport from azureSASTransport")
	}
}

func TestUnwrapTransport_UnknownRoundTripper_ReturnsNil(t *testing.T) {
	got := UnwrapTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, nil
	}))
	if got != nil {
		t.Fatal("expected nil for unknown RoundTripper type")
	}
}

func TestUnwrapTransport_Nil_ReturnsNil(t *testing.T) {
	got := UnwrapTransport(nil)
	if got != nil {
		t.Fatal("expected nil for nil input")
	}
}
