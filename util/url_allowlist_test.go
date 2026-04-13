//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"testing"
)

func TestValidateURLInAllowlist(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		serverAllowlist map[string]any
		allowlist       map[string]any
		wantErr         bool
	}{
		// --- cbRestrictedPaths: blocked regardless of either allowlist ---
		{
			name: "/diag/eval blocked even when all_access true",
			url:  "http://example.com/diag/eval",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess: true,
			},
			wantErr: true,
		},
		{
			name: "/diag/eval subpath blocked even when all_access true",
			url:  "http://example.com/diag/eval/something",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess: true,
			},
			wantErr: true,
		},
		{
			name: "/diag/eval blocked despite being in allowed_urls",
			url:  "http://example.com/diag/eval",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/diag/eval"},
			},
			wantErr: true,
		},

		// --- server allowlist checked first ---
		{
			name: "server allowlist permits - allowlist empty",
			url:  "http://example.com/api",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			allowlist: map[string]any{},
			wantErr:   false,
		},
		{
			name: "server allowlist denies but allowlist permits",
			url:  "http://example.com/api",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess: false,
			},
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: false,
		},
		{
			name: "both allowlists deny",
			url:  "http://example.com/api",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess: false,
			},
			allowlist: map[string]any{
				AllowlistKeyAllAccess: false,
			},
			wantErr: true,
		},
		{
			name: "server all_access true permits without checking allowlist",
			url:  "http://anything.example.com/any/path",
			serverAllowlist: map[string]any{
				AllowlistKeyAllAccess: true,
			},
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://different.com"},
			},
			wantErr: false,
		},

		// --- single allowlist cases (serverAllowlist nil, using allowlist only) ---
		{
			name:            "both nil allowed",
			url:             "http://example.com",
			serverAllowlist: nil,
			allowlist:       nil,
			wantErr:         false,
		},
		{
			name:      "empty allowlist allowed",
			url:       "http://example.com",
			allowlist: map[string]any{},
			wantErr:   false,
		},
		{
			name:      "missing all_access key",
			url:       "http://example.com",
			allowlist: map[string]any{"allowed_urls": []any{"http://example.com"}},
			wantErr:   true,
		},
		{
			name:      "all_access true permits any url",
			url:       "http://anything.example.com/any/path",
			allowlist: map[string]any{AllowlistKeyAllAccess: true},
			wantErr:   false,
		},
		{
			name: "exact url match",
			url:  "http://example.com/api",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: false,
		},
		{
			name: "path prefix match",
			url:  "http://example.com/api/v1/users",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: false,
		},
		{
			name: "path prefix does not match partial segment",
			url:  "http://example.com/apikeys",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: true,
		},
		{
			name: "wildcard host single label",
			url:  "https://api.example.com/v1",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"https://*.example.com/v1"},
			},
			wantErr: false,
		},
		{
			name: "wildcard host base domain denied",
			url:  "https://example.com/v1",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"https://*.example.com/v1"},
			},
			wantErr: true,
		},
		{
			name: "wildcard host multiple labels denied",
			url:  "https://deep.api.example.com/v1",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"https://*.example.com/v1"},
			},
			wantErr: true,
		},
		{
			name: "disallowed url blocks match",
			url:  "http://example.com/api/secret",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:      false,
				AllowlistKeyAllowedURLs:    []any{"http://example.com/api"},
				AllowlistKeyDisallowedURLs: []any{"http://example.com/api/secret"},
			},
			wantErr: true,
		},
		{
			name: "disallowed wildcard blocks subdomain",
			url:  "http://evil.example.com/api",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:      false,
				AllowlistKeyAllowedURLs:    []any{"http://*.example.com/api"},
				AllowlistKeyDisallowedURLs: []any{"http://evil.example.com"},
			},
			wantErr: true,
		},
		{
			name: "scheme mismatch denied",
			url:  "https://example.com/api",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: true,
		},
		{
			name: "port mismatch denied",
			url:  "http://example.com:9090/api",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com:8080/api"},
			},
			wantErr: true,
		},
		{
			name: "port match allowed",
			url:  "http://example.com:8080/api",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com:8080/api"},
			},
			wantErr: false,
		},
		{
			name: "unsupported scheme rejected",
			url:  "ftp://example.com/file",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"ftp://example.com/file"},
			},
			wantErr: true,
		},
		{
			name: "slice of strings allowed_urls",
			url:  "http://example.com/path",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []string{"http://example.com/path"},
			},
			wantErr: false,
		},
		{
			name: "empty path pattern matches all paths",
			url:  "http://example.com/any/deep/path",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com"},
			},
			wantErr: false,
		},
		{
			name: "non-restricted path on same host is allowed",
			url:  "http://example.com/api",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: false,
		},

		// --- path traversal via ".." segments must be denied ---
		{
			name: "dotdot traversal escapes allowed prefix",
			url:  "http://127.0.0.1:8091/test/../v1",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://127.0.0.1:8091/test"},
			},
			wantErr: true,
		},
		{
			name: "dotdot traversal to restricted path is blocked",
			url:  "http://example.com/safe/../diag/eval",
			allowlist: map[string]any{
				AllowlistKeyAllAccess: true,
			},
			wantErr: true,
		},
		{
			name: "dotdot that stays within allowed prefix is permitted",
			url:  "http://example.com/api/v1/../v2",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://example.com/api"},
			},
			wantErr: false,
		},
		{
			name: "dotdot in pattern URL is normalized before matching",
			url:  "http://127.0.0.1:8091/v1",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://127.0.0.1:8091/test/../v1"},
			},
			wantErr: false, // pattern normalizes to /v1, exact match
		},
		{
			name: "dotdot in pattern does not grant access to un-normalized prefix",
			url:  "http://127.0.0.1:8091/test/sub",
			allowlist: map[string]any{
				AllowlistKeyAllAccess:   false,
				AllowlistKeyAllowedURLs: []any{"http://127.0.0.1:8091/test/../v1"},
			},
			wantErr: true, // pattern normalizes to /v1, /test/sub does not match
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateURLInAllowlist(tc.url, tc.serverAllowlist, tc.allowlist)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateURLInAllowlist(%q) error=%v, wantErr=%v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestWildcardHostMatch(t *testing.T) {
	tests := []struct {
		input   string
		pattern string
		want    bool
	}{
		{"example.com", "example.com", true},
		{"Example.COM", "example.com", true},
		{"api.example.com", "*.example.com", true},
		{"www.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"deep.api.example.com", "*.example.com", false},
		{"other.com", "*.example.com", false},
		{"api.example.com", "api.example.com", true},
		{"www.example.com", "api.example.com", false},
	}

	for _, tc := range tests {
		got := wildcardHostMatch(tc.input, tc.pattern)
		if got != tc.want {
			t.Errorf("wildcardHostMatch(%q, %q) = %v, want %v", tc.input, tc.pattern, got, tc.want)
		}
	}
}

func TestPathPrefixMatch(t *testing.T) {
	tests := []struct {
		input  string
		prefix string
		want   bool
	}{
		{"/api/v1", "/api", true},
		{"/api", "/api", true},
		{"/apikeys", "/api", false},
		{"/api/", "/api", true},
		{"/other", "/api", false},
		{"/anything", "", true},
		{"/anything", "/", true},
		{"/api/v1/users", "/api/v1", true},
	}

	for _, tc := range tests {
		got := PathPrefixMatch(tc.input, tc.prefix)
		if got != tc.want {
			t.Errorf("pathPrefixMatch(%q, %q) = %v, want %v", tc.input, tc.prefix, got, tc.want)
		}
	}
}
