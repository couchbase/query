//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Tests for config.go.
//
// Strategy:
//   - ParseConfig is driven with hand-built value.Values; no server needed.
//   - ResolveProviderAndModel is tested against the real provider registry,
//     which the package init() populates - no mocking required.

package ai_gateway

import (
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

// assertCode fails the test unless err carries the expected error code.
func assertCode(t *testing.T, err errors.Error, want errors.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	if err.Code() != want {
		t.Fatalf("expected error code %v, got %v (%v)", want, err.Code(), err)
	}
}

// ─── ParseConfig ──────────────────────────────────────────────────────────────

func TestParseConfig_NilValue(t *testing.T) {
	_, err := ParseConfig(nil)
	assertCode(t, err, errors.E_NL_MISSING_NL_PARAM)
}

func TestParseConfig_NullValue(t *testing.T) {
	_, err := ParseConfig(value.NULL_VALUE)
	assertCode(t, err, errors.E_NL_MISSING_NL_PARAM)
}

func TestParseConfig_MissingValue(t *testing.T) {
	_, err := ParseConfig(value.MISSING_VALUE)
	assertCode(t, err, errors.E_NL_MISSING_NL_PARAM)
}

func TestParseConfig_NotAnObject(t *testing.T) {
	_, err := ParseConfig(value.NewValue("openai"))
	assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
}

func TestParseConfig_AllFields(t *testing.T) {
	cfg, err := ParseConfig(value.NewValue(map[string]interface{}{
		"provider":           "bedrock",
		"model":              "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
		"cred_id":            "my-cred",
		"api_key":            "sk-123",
		"endpoint":           "https://example.com/v1/chat/completions",
		"region":             "us-west-2",
		"output_token_limit": 4096,
		"moderation":         false,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != "bedrock" || cfg.Model != "us.anthropic.claude-sonnet-4-5-20250929-v1:0" ||
		cfg.CredId != "my-cred" || cfg.APIKey != "sk-123" ||
		cfg.Endpoint != "https://example.com/v1/chat/completions" ||
		cfg.Region != "us-west-2" || cfg.OutputTokenLimit != 4096 ||
		cfg.Moderation == nil || *cfg.Moderation {
		t.Fatalf("fields not extracted as expected: %+v", cfg)
	}
}

// The moderation opt-out is a pointer so an unset value (moderation runs by
// default) is distinguishable from an explicit false (moderation skipped).
func TestParseConfig_Moderation(t *testing.T) {
	// Absent: the pointer stays nil so moderation runs by default.
	cfg, err := ParseConfig(value.NewValue(map[string]interface{}{"provider": "openai"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Moderation != nil {
		t.Fatalf("absent moderation must leave a nil pointer, got %v", *cfg.Moderation)
	}

	// Explicit true / false are each captured distinctly from unset.
	for _, want := range []bool{true, false} {
		cfg, err := ParseConfig(value.NewValue(map[string]interface{}{"moderation": want}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Moderation == nil || *cfg.Moderation != want {
			t.Fatalf("moderation %v: got %v", want, cfg.Moderation)
		}
	}

	// A wrong-typed value is a caller error rather than a silent fallback: a
	// string where a boolean is expected would otherwise leave moderation
	// silently on, the opposite of the caller's apparent intent.
	if _, err := ParseConfig(value.NewValue(map[string]interface{}{"moderation": "yes"})); err == nil {
		t.Fatalf("wrong-typed moderation must return an error")
	}
}

// A present-but-wrong-typed field is a caller error, distinct from an absent
// field (which falls back to its default). Each field is validated for type.
func TestParseConfig_WrongTypedFieldsError(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		value interface{}
	}{
		{"provider not string", "provider", 123},
		{"model not string", "model", true},
		{"cred_id not string", "cred_id", 1},
		{"api_key not string", "api_key", 1},
		{"endpoint not string", "endpoint", 1},
		{"region not string", "region", 1},
		{"output_token_limit not number", "output_token_limit", "many"},
		{"output_token_limit not integer", "output_token_limit", 1.5},
		{"moderation not boolean", "moderation", "yes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseConfig(value.NewValue(map[string]interface{}{tc.field: tc.value})); err == nil {
				t.Fatalf("wrong-typed %q must return an error", tc.field)
			}
		})
	}
}

// An explicit JSON null is treated as "not provided" and falls back to the
// default, not as a type error.
func TestParseConfig_NullFieldsTreatedAsAbsent(t *testing.T) {
	cfg, err := ParseConfig(value.NewValue(map[string]interface{}{
		"provider":           nil,
		"output_token_limit": nil,
		"moderation":         nil,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != "" || cfg.OutputTokenLimit != 0 || cfg.Moderation != nil {
		t.Fatalf("null fields should fall back to defaults: %+v", cfg)
	}
}

func TestParseConfig_EmptyObject(t *testing.T) {
	cfg, err := ParseConfig(value.NewValue(map[string]interface{}{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *cfg != (Config{}) {
		t.Fatalf("expected zero-valued config, got: %+v", cfg)
	}
}

// ─── ResolveProviderAndModel ──────────────────────────────────────────────────

func TestResolveProviderAndModel_DefaultsToOpenAI(t *testing.T) {
	cfg := &Config{APIKey: "sk-123"}
	if err := cfg.ResolveProviderAndModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != ProviderOpenAI {
		t.Fatalf("expected default provider %q, got %q", ProviderOpenAI, cfg.Provider)
	}
	if cfg.Model != GPT4o_2024_05_13 {
		t.Fatalf("expected default model %q, got %q", GPT4o_2024_05_13, cfg.Model)
	}
}

func TestResolveProviderAndModel_ProviderLowercased(t *testing.T) {
	cfg := &Config{Provider: "OpenAI", APIKey: "sk-123"}
	if err := cfg.ResolveProviderAndModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != ProviderOpenAI {
		t.Fatalf("expected provider lowercased to %q, got %q", ProviderOpenAI, cfg.Provider)
	}
}

// Model identifiers must be passed to the provider verbatim: self-hosted
// served-model names and Bedrock ARNs are case-sensitive, so resolution must
// never normalize their case.
func TestResolveProviderAndModel_ModelCasePreserved(t *testing.T) {
	for _, m := range []string{
		"GPT-4o-2024-05-13",
		"org/Model-24K-16bit",
		"arn:aws:bedrock:us-east-1:123456789012:inference-profile/US.Anthropic.claude",
	} {
		cfg := &Config{Provider: ProviderOpenAI, Model: m, APIKey: "sk-123"}
		if err := cfg.ResolveProviderAndModel(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Model != m {
			t.Fatalf("model %q was altered to %q", m, cfg.Model)
		}
	}
}

func TestResolveProviderAndModel_UnknownProvider(t *testing.T) {
	cfg := &Config{Provider: "nosuchvendor", APIKey: "sk-123"}
	assertCode(t, cfg.ResolveProviderAndModel(), errors.E_NL_VENDOR_NOT_SUPPORTED)
}

func TestResolveProviderAndModel_DefaultModelPerProvider(t *testing.T) {
	cases := []struct {
		provider string
		endpoint string // slm has no built-in host, so its config needs one
		want     string
	}{
		{ProviderOpenAI, "", GPT4o_2024_05_13},
		{ProviderBedrock, "", Claude45InferenceProfileID},
		{ProviderGemini, "", Gemini25Pro},
		{ProviderSLM, "http://localhost:8000/v1/chat/completions", SLMDefaultModel},
	}
	for _, c := range cases {
		cfg := &Config{Provider: c.provider, APIKey: "key", Endpoint: c.endpoint}
		if err := cfg.ResolveProviderAndModel(); err != nil {
			t.Fatalf("%s: unexpected error: %v", c.provider, err)
		}
		if cfg.Model != c.want {
			t.Fatalf("%s: expected default model %q, got %q", c.provider, c.want, cfg.Model)
		}
	}
}

// Credential policy: on a provider's built-in host a credential is required
// unless the provider authenticates ambiently (Bedrock via the AWS default
// chain); on a caller-supplied endpoint the endpoint's owner decides the auth
// policy, so a credential is optional for every provider.
func TestResolveProviderAndModel_CredentialPolicy(t *testing.T) {
	cases := []struct {
		provider     string
		endpoint     string
		allowsNoCred bool
	}{
		{ProviderOpenAI, "", false},
		{ProviderGemini, "", false},
		{ProviderBedrock, "", true},
		{ProviderOpenAI, "https://proxy.example.com/v1/chat/completions", true},
		{ProviderGemini, "https://proxy.example.com/generateContent", true},
		{ProviderSLM, "http://localhost:8000/v1/chat/completions", true},
	}
	for _, c := range cases {
		cfg := &Config{Provider: c.provider, Endpoint: c.endpoint}
		err := cfg.ResolveProviderAndModel()
		if c.allowsNoCred {
			if err != nil {
				t.Fatalf("%s (endpoint %q): expected no-credential config to be accepted, got: %v",
					c.provider, c.endpoint, err)
			}
		} else {
			assertCode(t, err, errors.E_NL_INVALID_NATURAL_CONFIG)
		}
	}
}

// slm has no built-in host: a config without an endpoint must fail resolution
// even when a credential is supplied.
func TestResolveProviderAndModel_SLMRequiresEndpoint(t *testing.T) {
	for _, cfg := range []*Config{
		{Provider: ProviderSLM},
		{Provider: ProviderSLM, APIKey: "key"},
		{Provider: ProviderSLM, CredId: "my-cred"},
	} {
		assertCode(t, cfg.ResolveProviderAndModel(), errors.E_NL_INVALID_NATURAL_CONFIG)
	}
}

func TestResolveProviderAndModel_CredIdAloneSuffices(t *testing.T) {
	cfg := &Config{Provider: ProviderOpenAI, CredId: "my-cred"}
	if err := cfg.ResolveProviderAndModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
