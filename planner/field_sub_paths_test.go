//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"reflect"
	"sort"
	"testing"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

func mustParseExpr(t *testing.T, s string) expression.Expression {
	t.Helper()
	e, err := parser.Parse(s)
	if err != nil {
		t.Fatalf("parser.Parse(%q): %v", s, err)
	}
	return e
}

// collectSubPaths runs collectFieldSubPaths over every given expression string
// (as build_select_from.go's external-scan block does over this.node.Expressions())
// and returns the resulting tree flattened into the same sorted dotted-path
// format that ends up in SetEarlyProjection.
func collectSubPaths(t *testing.T, alias string, exprStrs ...string) []string {
	t.Helper()
	roots := make(map[string]*pathNode)
	for _, s := range exprStrs {
		collectFieldSubPaths(mustParseExpr(t, s), alias, roots)
	}
	var got []string
	for name, node := range roots {
		got = append(got, flattenPathNode(name, node)...)
	}
	sort.Strings(got)
	return got
}

func TestCollectFieldSubPaths(t *testing.T) {
	tests := []struct {
		name  string
		exprs []string
		want  []string
	}{
		{
			name:  "single sub-path access",
			exprs: []string{"d.attrs.color"},
			want:  []string{"attrs.color"},
		},
		{
			name:  "two distinct sub-paths on the same field",
			exprs: []string{"d.attrs.color", "d.attrs.size"},
			want:  []string{"attrs.color", "attrs.size"},
		},
		{
			name:  "bare field access needs the whole field",
			exprs: []string{"d.attrs"},
			want:  []string{"attrs"},
		},
		{
			name:  "mixed bare and narrow falls back to whole field",
			exprs: []string{"d.attrs.color", "d.attrs"},
			want:  []string{"attrs"},
		},
		{
			name:  "deeper than one level narrows to the deepest common point",
			exprs: []string{"d.attrs.geo.lat"},
			want:  []string{"attrs.geo.lat"},
		},
		{
			name:  "two distinct paths at different depths under the same field",
			exprs: []string{"d.attrs.geo.lat", "d.attrs.color"},
			want:  []string{"attrs.color", "attrs.geo.lat"},
		},
		{
			name:  "a bare reference partway down still forces that subtree whole",
			exprs: []string{"d.attrs.geo.lat", "d.attrs.geo"},
			want:  []string{"attrs.geo"},
		},
		{
			name:  "unrelated bare field alongside a narrowed one",
			exprs: []string{"d.attrs.color", "d.id"},
			want:  []string{"attrs.color", "id"},
		},
		{
			name:  "sub-path inside a comparison predicate",
			exprs: []string{"d.attrs.color = 'red'"},
			want:  []string{"attrs.color"},
		},
		{
			// Concat isn't a *Field itself, so it doesn't match at the top of
			// collectFieldSubPaths -- but recursing into its Children() finds
			// each operand's own *Field chain independently, so both sides of a
			// concatenation still get narrowed. Two different top-level fields
			// here ("b" and "f"); see "two distinct sub-paths on the same field"
			// above for the same-top-level-field case.
			name:  "concatenation of two sub-paths narrows both independently",
			exprs: []string{"d.b.c || d.f.q"},
			want:  []string{"b.c", "f.q"},
		},
		{
			name:  "different alias is ignored",
			exprs: []string{"other.attrs.color"},
			want:  nil,
		},
		{
			name:  "same field narrowed then widened then narrowed again stays whole",
			exprs: []string{"d.attrs.color", "d.attrs", "d.attrs.size"},
			want:  []string{"attrs"},
		},
		{
			name:  "backtick-quoted names parse the same as unquoted",
			exprs: []string{"d.`attrs`.`color`"},
			want:  []string{"attrs.color"},
		},
		{
			name:  "array index falls back to whole field, not a bogus 'attrs[0]' key",
			exprs: []string{"d.attrs[0].color"},
			want:  []string{"attrs"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := collectSubPaths(t, "d", tc.exprs...)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("collectSubPaths() = %v, want %v", got, tc.want)
			}
		})
	}
}
