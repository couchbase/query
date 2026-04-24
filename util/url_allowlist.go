//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"fmt"
	"net/url"
	"strings"
)

// AllowlistKeys are the recognised keys in the allowlist map.
const (
	AllowlistKeyAllAccess      = "all_access"
	AllowlistKeyAllowedURLs    = "allowed_urls"
	AllowlistKeyDisallowedURLs = "disallowed_urls"
)

// cbRestrictedPaths lists URL paths that Couchbase unconditionally blocks,
// regardless of the allowlist configuration.
var cbRestrictedPaths = []string{
	"/diag/eval",
}

// ValidateURLInAllowlist reports whether urlStr is permitted by either
// serverAllowlist or allowlist, checking serverAllowlist first.
//
// Allowlist map keys (same for both maps):
//
//	"all_access"       bool  – if true every URL is permitted; no further checks.
//	"allowed_urls"     []any – patterns a URL must match (strings or *url.URL).
//	"disallowed_urls"  []any – patterns that explicitly block a URL (strings or *url.URL).
//
// URL pattern matching rules (applied to both lists):
//   - Scheme and host must match exactly, unless the host pattern begins with "*.".
//   - A host pattern "*.example.com" matches any single-label subdomain:
//     "api.example.com", "www.example.com" – but NOT "example.com" itself,
//     and NOT "deep.api.example.com" (only one wildcard label is supported).
//   - Path matching uses prefix semantics: a pattern path "/api" permits
//     "/api", "/api/v1", "/api/v1/users" but not "/apikeys".
//   - User-info in the pattern is only checked when it is non-empty; an empty
//     pattern user-info matches any (or no) credentials in the input URL.
//
// Evaluation order:
//  1. URL fails to parse or uses an unsupported scheme → denied.
//  2. URL path matches cbRestrictedPaths → denied (overrides all_access in both maps).
//  3. Both maps are nil or empty → allowed (no policy configured).
//  4. serverAllowlist permits the URL → allowed.
//  5. allowlist permits the URL → allowed.
//  6. Neither map permits → denied.
func ValidateURLInAllowlist(urlStr string, serverAllowlist, allowlist map[string]any) error {
	inputURL, err := parseAndValidateURL(urlStr)
	if err != nil {
		return err
	}

	// cbRestrictedPaths are blocked regardless of any allowlist configuration.
	for _, restricted := range cbRestrictedPaths {
		if pathPrefixMatch(inputURL.EscapedPath(), restricted) {
			return fmt.Errorf("access restricted - %v", urlStr)
		}
	}

	var lastErr error
	var checked bool
	for _, al := range []map[string]any{serverAllowlist, allowlist} {
		if len(al) == 0 {
			continue
		}
		checked = true
		if err := checkAllowlist(inputURL, urlStr, al); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if !checked {
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("URL %q is not in the allowed list", urlStr)
}

// checkAllowlist tests inputURL against a single allowlist map.
// Returns nil if the URL is permitted, or a non-nil error if denied or the map
// is unusable (missing 'all_access', etc.). Callers must not pass nil or empty maps.
func checkAllowlist(inputURL *url.URL, urlStr string, allowlist map[string]any) error {
	if len(allowlist) == 0 {
		return fmt.Errorf("allowed list is empty")
	}

	allAccess, ok := allowlist[AllowlistKeyAllAccess]
	if !ok {
		return fmt.Errorf("'all_access' field missing from allowed list")
	}
	allAccessBool, ok := allAccess.(bool)
	if !ok {
		return fmt.Errorf("'all_access' must be a boolean value")
	}
	if allAccessBool {
		return nil
	}

	if raw, ok := allowlist[AllowlistKeyDisallowedURLs]; ok {
		patterns, err := toURLPatterns(raw)
		if err != nil {
			return fmt.Errorf("invalid disallowed_urls: %v", err)
		}
		if urlMatchesAny(inputURL, patterns) {
			return fmt.Errorf("URL %q is explicitly disallowed", urlStr)
		}
	}

	if raw, ok := allowlist[AllowlistKeyAllowedURLs]; ok {
		patterns, err := toURLPatterns(raw)
		if err != nil {
			return fmt.Errorf("invalid allowed_urls: %v", err)
		}
		if urlMatchesAny(inputURL, patterns) {
			return nil
		}
	}

	return fmt.Errorf("URL %q is not in the allowed list", urlStr)
}

// parseAndValidateURL parses urlStr and verifies it has a supported scheme and a host.
func parseAndValidateURL(urlStr string) (*url.URL, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %v", urlStr, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q in URL %q (only http/https allowed)", u.Scheme, urlStr)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host in URL %q", urlStr)
	}
	// Normalize path to prevent traversal via ".." or "." segments.
	// JoinPath() with no arguments cleans the path via the URL machinery,
	// correctly handling both Path and RawPath (percent-encoding).
	return u.JoinPath(), nil
}

// toURLPatterns converts raw (string, *url.URL, or []any of either) into a slice
// of *url.URL patterns.  Items that cannot be parsed are returned as an error.
// Every returned URL has its path cleaned via JoinPath() to remove ".." / "." segments.
func toURLPatterns(raw any) ([]*url.URL, error) {
	switch v := raw.(type) {
	case []*url.URL:
		out := make([]*url.URL, len(v))
		for i, u := range v {
			out[i] = u.JoinPath()
		}
		return out, nil
	case *url.URL:
		return []*url.URL{v.JoinPath()}, nil
	case string:
		u, err := url.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("cannot parse pattern %q: %v", v, err)
		}
		return []*url.URL{u.JoinPath()}, nil
	case []any:
		out := make([]*url.URL, 0, len(v))
		for i, item := range v {
			switch iv := item.(type) {
			case *url.URL:
				out = append(out, iv.JoinPath())
			case string:
				u, err := url.Parse(iv)
				if err != nil {
					return nil, fmt.Errorf("cannot parse pattern[%d] %q: %v", i, iv, err)
				}
				out = append(out, u.JoinPath())
			default:
				return nil, fmt.Errorf("pattern[%d] has unsupported type %T", i, item)
			}
		}
		return out, nil
	case []string:
		out := make([]*url.URL, 0, len(v))
		for i, s := range v {
			u, err := url.Parse(s)
			if err != nil {
				return nil, fmt.Errorf("cannot parse pattern[%d] %q: %v", i, s, err)
			}
			out = append(out, u.JoinPath())
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported pattern list type %T", raw)
	}
}

// urlMatchesAny returns true if inputURL matches any of the given patterns.
func urlMatchesAny(inputURL *url.URL, patterns []*url.URL) bool {
	for _, p := range patterns {
		if urlMatchesPattern(inputURL, p) {
			return true
		}
	}
	return false
}

// urlMatchesPattern checks whether inputURL satisfies pattern p.
//
// Matching rules:
//   - Scheme must be identical.
//   - Host is matched with wildcardHostMatch (supports "*.suffix" prefix wildcard).
//   - User-info in p is only checked when non-empty.
//   - Path is checked with pathPrefixMatch.
func urlMatchesPattern(inputURL, p *url.URL) bool {
	if inputURL.Scheme != p.Scheme {
		return false
	}

	if !wildcardHostMatch(inputURL.Hostname(), p.Hostname()) {
		return false
	}

	// Port: if the pattern specifies a port the input must carry the same port.
	if pPort := p.Port(); pPort != "" {
		if inputURL.Port() != pPort {
			return false
		}
	}

	// User-info: only enforced when the pattern carries credentials.
	if pUser := p.User.String(); pUser != "" {
		if inputURL.User.String() != pUser {
			return false
		}
	}

	return pathPrefixMatch(inputURL.EscapedPath(), p.EscapedPath())
}

// wildcardHostMatch reports whether inputHost matches patternHost.
//
// If patternHost starts with "*." it is treated as a single-label prefix wildcard:
//   - "*.example.com" matches "api.example.com" and "www.example.com".
//   - It does NOT match "example.com" (the base domain itself).
//   - It does NOT match "deep.api.example.com" (multiple extra labels).
//
// Otherwise an exact case-insensitive comparison is performed.
func wildcardHostMatch(inputHost, patternHost string) bool {
	patternHost = strings.ToLower(patternHost)
	inputHost = strings.ToLower(inputHost)

	if !strings.HasPrefix(patternHost, "*.") {
		return inputHost == patternHost
	}

	// patternHost is of the form "*.suffix"
	suffix := patternHost[1:] // includes the leading "."

	if !strings.HasSuffix(inputHost, suffix) {
		return false
	}

	// The label before the suffix must be a single label (no dots).
	label := inputHost[:len(inputHost)-len(suffix)]
	return label != "" && !strings.Contains(label, ".")
}

// pathPrefixMatch reports whether inputPath starts with prefixPath using
// path-segment semantics (avoids "/test" incorrectly matching "/testsuite").
//
// Rules:
//   - If prefixPath is empty or "/", every inputPath is considered a match.
//   - Exact match always qualifies.
//   - A partial prefix match is valid only when the next character in inputPath
//     is "/" (i.e., the prefix ends on a segment boundary).
func pathPrefixMatch(inputPath, prefixPath string) bool {
	if prefixPath == "" || prefixPath == "/" {
		return true
	}

	if !strings.HasPrefix(inputPath, prefixPath) {
		return false
	}

	// Exact match.
	if len(inputPath) == len(prefixPath) {
		return true
	}

	// Prefix must end at a segment boundary.
	// If prefixPath already ends with "/" the HasPrefix check is sufficient.
	if strings.HasSuffix(prefixPath, "/") {
		return true
	}

	// Otherwise the next character must be "/".
	return inputPath[len(prefixPath)] == '/'
}
