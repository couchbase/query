//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package natural

import (
	"encoding/json"
	"testing"

	"github.com/couchbase/query/errors"
)

// With no model in the request, Query sends an empty model so Capella iQ applies its own current
// default for the vendor, and sends no temperature with it.
func TestNoModelSendsEmptyModelAndNoTemperature(t *testing.T) {
	for _, vendor := range []string{VENDOR_OPENAI, VENDOR_BEDROCK} {
		sqlPrompt, err := newSQLPrompt(map[string]interface{}{}, "show few documents", false, vendor, "")
		if err != nil {
			t.Fatalf("%s: newSQLPrompt: %v", vendor, err)
		}
		udfPrompt, err := newJSUDFPrompt(map[string]interface{}{}, "add 2 numbers", vendor, "")
		if err != nil {
			t.Fatalf("%s: newJSUDFPrompt: %v", vendor, err)
		}
		for _, p := range []*prompt{sqlPrompt, udfPrompt} {
			payload, e := json.Marshal(p)
			if e != nil {
				t.Fatalf("%s: marshal: %v", vendor, e)
			}
			var sent struct {
				Vendor             string                 `json:"vendor"`
				CompletionSettings map[string]interface{} `json:"completionSettings"`
			}
			if e = json.Unmarshal(payload, &sent); e != nil {
				t.Fatalf("%s: unmarshal: %v", vendor, e)
			}
			if sent.Vendor != vendor {
				t.Errorf("vendor sent = %q, want %q", sent.Vendor, vendor)
			}
			if model, ok := sent.CompletionSettings["model"]; !ok || model != "" {
				t.Errorf("%s: model sent = %v (present %v), want empty", vendor, model, ok)
			}
			if temp, ok := sent.CompletionSettings["temperature"]; ok {
				t.Errorf("%s: temperature %v sent with an empty model", vendor, temp)
			}
		}
	}
}

func TestPickVendorAndModel(t *testing.T) {
	providers := []modelProvider{
		{ID: "bedrock", Models: []string{"us.anthropic.claude-sonnet-4-5-20250929-v1:0"}, Enabled: true},
		{ID: "openai", Models: []string{"gpt-3.5-turbo", "GPT-4o"}, Enabled: true},
		{ID: "gemini", Enabled: false},
	}
	noOpenAI := []modelProvider{
		{ID: "openai", Enabled: false},
		{ID: "bedrock", Enabled: true},
	}
	cases := []struct {
		name       string
		vendor     string
		model      string
		providers  []modelProvider
		wantCode   errors.ErrorCode
		wantVendor string
		wantModel  string
	}{
		{"model without vendor", "", "gpt-4o", providers, errors.E_NL_MODEL_WITHOUT_VENDOR, "", ""},
		{"no providers", "openai", "", nil, errors.E_NL_NO_VENDORS_AVAILABLE, "", ""},
		{"unknown vendor", "mistral", "", providers, errors.E_NL_VENDOR_NOT_SUPPORTED, "", ""},
		{"disabled vendor", "gemini", "", providers, errors.E_NL_VENDOR_NOT_ENABLED, "", ""},
		{"none enabled", "", "", []modelProvider{{ID: "openai"}}, errors.E_NL_NO_VENDORS_ENABLED, "", ""},
		{"neither given: openai preferred, no model", "", "", providers, 0, "openai", ""},
		{"neither given: first enabled when openai is disabled", "", "", noOpenAI, 0, "bedrock", ""},
		{"vendor only: no model", "Bedrock", "", providers, 0, "bedrock", ""},
		{"listed model", "openai", "gpt-3.5-turbo", providers, 0, "openai", "gpt-3.5-turbo"},
		{"case-insensitive match uses listed spelling", "openai", "gpt-4o", providers, 0, "openai", "GPT-4o"},
		{"unlisted model passes through", "openai", "gpt-5.4-2026-03-05", providers, 0, "openai",
			"gpt-5.4-2026-03-05"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vendor, model, err := pickVendorAndModel(c.vendor, c.model, c.providers)
			if c.wantCode != 0 {
				if err == nil || err.Code() != c.wantCode {
					t.Fatalf("expected error code %v, got %v", c.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if vendor != c.wantVendor || model != c.wantModel {
				t.Fatalf("got (%q, %q), want (%q, %q)", vendor, model, c.wantVendor, c.wantModel)
			}
		})
	}
}

func TestGetTemperatureForModel(t *testing.T) {
	cases := []struct {
		vendor, model string
		want          float64
	}{
		{VENDOR_OPENAI, "", 0},
		{VENDOR_OPENAI, "gpt-4o", 0},
		{VENDOR_OPENAI, "gpt-5", 1},
		{VENDOR_OPENAI, "gpt-5.1", 1},
		{VENDOR_OPENAI, "GPT-5.1", 1},
		{VENDOR_BEDROCK, "", 0},
		{VENDOR_BEDROCK, "gpt-5.1", 0},
	}
	for _, c := range cases {
		if got := getTemperatureForModel(c.vendor, c.model); got != c.want {
			t.Errorf("getTemperatureForModel(%q, %q) = %v, want %v", c.vendor, c.model, got, c.want)
		}
	}
}
