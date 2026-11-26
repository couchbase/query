//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"reflect"
	"strings"
	"testing"
)

// MB-75645: the include/exclude and remap values cbbackupmgr sends on the global restore call
func TestParseKeyspacePathsAndRemaps(t *testing.T) {
	paths := parseKeyspacePaths("QUERY_METADATA,b1._default._default, b2.s1 ,,a.b.c.d,b\\.x.s")
	expected := [][]string{{"QUERY_METADATA"}, {"b1", "_default", "_default"}, {"b2", "s1"}, {"b.x", "s"}}
	if !reflect.DeepEqual(paths, expected) {
		t.Errorf("paths: got %v, expected %v", paths, expected)
	}
	if parseKeyspacePaths("") != nil {
		t.Errorf("expected no paths for an empty value")
	}

	remaps := parseKeyspaceRemaps("b1._default._default:b1.s2.c1,b1:b2,bad,a:b:c,b3:,:b4")
	expectedRemaps := []keyspaceRemap{
		{from: []string{"b1", "_default", "_default"}, to: []string{"b1", "s2", "c1"}},
		{from: []string{"b1"}, to: []string{"b2"}},
	}
	if !reflect.DeepEqual(remaps, expectedRemaps) {
		t.Errorf("remaps: got %v, expected %v", remaps, expectedRemaps)
	}
}

func TestKeyspaceRefParts(t *testing.T) {
	for ref, expected := range map[string][]string{
		"default:b1._default._default": {"b1", "_default", "_default"},
		"default:b1.s2.c1":             {"b1", "s2", "c1"},
		"default:b1":                   {"b1", "_default", "_default"},
	} {
		if parts := keyspaceRefParts(ref); !reflect.DeepEqual(parts, expected) {
			t.Errorf("%s: got %v, expected %v", ref, parts, expected)
		}
	}
}

func TestSavedPlanWarning(t *testing.T) {
	refs := func(r ...string) []interface{} {
		rv := make([]interface{}, len(r))
		for i := range r {
			rv[i] = r[i]
		}
		return rv
	}
	for _, tc := range []struct {
		desc            string
		refs            []interface{}
		remap, inc, exc string
		contains        string // "" for no warning
	}{
		{"collection remap", refs("default:b1._default._default"), "b1._default._default:b1.s2.c1", "", "",
			"references keyspace b1._default._default, mapped to b1.s2.c1 by the remap of this restore; " +
				"the plan still uses b1._default._default and has to be prepared again to use b1.s2.c1"},
		{"bucket remap", refs("default:b1.s1.c1"), "b1:b2", "", "", "mapped to b2.s1.c1"},
		{"bucket only reference", refs("default:b1"), "b1:b2", "", "", "mapped to b2._default._default"},
		{"scope remap", refs("default:b1.s1.c1"), "b1.s1:b1.s9", "", "", "mapped to b1.s9.c1"},
		{"most specific remap, bucket first", refs("default:b1.s1.c1"), "b1:b2,b1.s1:b1.s9", "", "",
			"mapped to b1.s9.c1"},
		{"most specific remap, bucket last", refs("default:b1.s1.c1"), "b1.s1:b1.s9,b1:b2", "", "",
			"mapped to b1.s9.c1"},
		{"bucket remap for another scope", refs("default:b1.s2.c1"), "b1.s1:b1.s9,b1:b2", "", "",
			"mapped to b2.s2.c1"},
		{"remap of another collection", refs("default:b1.s1.c1"), "b1.s1.c2:b1.s1.c3", "", "", ""},
		{"remap of another bucket", refs("default:b10.s1.c1"), "b1:b2", "", "", ""},
		{"second reference mapped", refs("default:b3.s.c", "default:b1.s.c"), "b1:b2", "", "",
			"references keyspace b1.s.c"},
		{"included", refs("default:b1._default._default"), "", "QUERY_METADATA,b1._default._default", "", ""},
		{"included bucket", refs("default:b1.s1.c1"), "", "QUERY_METADATA,b1", "", ""},
		{"not included", refs("default:b1.s1.c1"), "", "QUERY_METADATA,b1._default._default", "",
			"not included by the filter of this restore; the plan still uses b1.s1.c1, whose data this restore " +
				"does not restore"},
		{"excluded", refs("default:b1.s1.c1"), "", "", "b1.s1", "excluded by the filter of this restore"},
		{"not excluded", refs("default:b1.s1.c1"), "", "", "b1.s2", ""},
		{"nothing given", refs("default:b1.s1.c1"), "", "", "", ""},
	} {
		msg := savedPlanWarning("p1", tc.refs, parseKeyspaceRemaps(tc.remap), parseKeyspacePaths(tc.inc),
			parseKeyspacePaths(tc.exc))
		if tc.contains == "" {
			if msg != "" {
				t.Errorf("%s: expected no warning, got %q", tc.desc, msg)
			}
		} else if !strings.Contains(msg, tc.contains) || !strings.Contains(msg, "saved plan p1") {
			t.Errorf("%s: got %q, expected it to contain %q", tc.desc, msg, tc.contains)
		}
	}
}

// MB-75645: include/exclude of a direct call to the QUERY_METADATA backup/restore match saved plans down to the
// collection
func TestSavedPlanFilter(t *testing.T) {
	for _, tc := range []struct {
		ref, include, exclude string
		expected              bool
	}{
		{"default:b1._default._default", "_default._default", "", true},
		{"default:b1._default._default", "_default", "", true},
		{"default:b1", "_default._default", "", true},
		{"default:b1.s2.c1", "_default._default", "", false},
		{"default:b1.s2.c1", "s2.c1", "", true},
		{"default:b1.s2.c2", "s2.c1", "", false},
		{"default:b1.s2.c1", "", "s2", false},
		{"default:b1.s2.c1", "", "s2.c2", true},
		{"default:b1.s2.c1", "", "", true},
	} {
		include, err := newFilter(tc.include)
		if err != nil {
			t.Fatalf("include %q: %v", tc.include, err)
		}
		exclude, err := newFilter(tc.exclude)
		if err != nil {
			t.Fatalf("exclude %q: %v", tc.exclude, err)
		}
		if got := filterEval(savedPlanFilterPath(tc.ref), include, exclude, true); got != tc.expected {
			t.Errorf("%s, include %q, exclude %q: got %v, expected %v", tc.ref, tc.include, tc.exclude, got,
				tc.expected)
		}
	}
}
