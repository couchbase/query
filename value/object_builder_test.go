//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"fmt"
	"testing"
)

func TestBuildObjectFromDottedPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]any
	}{
		{
			name:  "basic nested paths",
			input: []string{"x.y.z", "x.y1.z1", "x.y1.z2", "x.y2", "`x`.`y3`.`z`", "x1"},
			expected: map[string]any{
				"x": map[string]any{
					"y":  map[string]any{"z": "x.y.z"},
					"y1": map[string]any{"z1": "x.y1.z1", "z2": "x.y1.z2"},
					"y2": "x.y2",
					"y3": map[string]any{"z": "x.y3.z"},
				},
				"x1": "x1",
			},
		},
		{
			name:  "single path",
			input: []string{"a.b.c"},
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "a.b.c",
					},
				},
			},
		},
		{
			name:  "path with spaces in backticks",
			input: []string{"x.`y with space`.z"},
			expected: map[string]any{
				"x": map[string]any{
					"y with space": map[string]any{
						"z": "x.y with space.z",
					},
				},
			},
		},
		{
			name:  "empty backtick field",
			input: []string{"x.``.y"},
			expected: map[string]any{
				"x": map[string]any{
					"": map[string]any{
						"y": "x..y",
					},
				},
			},
		},
		{
			name:  "mixed backticks and regular",
			input: []string{"`a`.b.c", "d.`e`.f"},
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "a.b.c",
					},
				},
				"d": map[string]any{
					"e": map[string]any{
						"f": "d.e.f",
					},
				},
			},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]any{},
		},
		{
			name:  "single field",
			input: []string{"x"},
			expected: map[string]any{
				"x": "x",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildObjectFromDottedPaths(tt.input)
			if !mapsEqual(result, tt.expected) {
				t.Errorf("BuildObjectFromDottedPaths(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDottedPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple path", "x.y.z", []string{"x", "y", "z"}},
		{"fully backtick quoted", "`x`.`y`.`z`", []string{"x", "y", "z"}},
		{"mixed quotes", "x.`y`.z", []string{"x", "y", "z"}},
		{"spaces in backticks", "x.`y with space`.z", []string{"x", "y with space", "z"}},
		{"empty backtick", "x.``.y", []string{"x", "", "y"}},
		{"single field", "x", []string{"x"}},
		{"empty string", "", []string{}},
		{"backtick with space only", "x.` `.z", []string{"x", " ", "z"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDottedPath(tt.input)
			if !slicesEqual(result, tt.expected) {
				t.Errorf("parseDottedPath(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function to compare maps for testing
func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !interfaceEqual(av, bv) {
			return false
		}
	}
	return true
}

// Helper function to compare slices for testing
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function to compare arbitrary interface values for testing
func interfaceEqual(a, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case nil:
		return b == nil
	case map[string]any:
		bv, ok := b.(map[string]any)
		return ok && mapsEqual(av, bv)
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}

func TestObjectPopulate(t *testing.T) {
	template := map[string]any{
		"x": map[string]any{
			"y":  map[string]any{"z": "x.y.z"},
			"y1": map[string]any{"z1": "x.y1.z1", "z2": "x.y1.z2"},
			"y2": "x.y2",
			"y3": map[string]any{"z": "x.y3.z"},
		},
		"x1": "x1",
	}

	values := map[string]any{
		"x.y.z":   "abc",
		"x.y1.z1": 1,
		"x.y1.z2": map[string]any{"a": 1},
		"x.y2":    true,
		"x.y3.z":  1.2,
		"x1":      50,
	}

	expected := map[string]any{
		"x": map[string]any{
			"y": map[string]any{
				"z": "abc",
			},
			"y1": map[string]any{
				"z1": 1,
				"z2": map[string]any{"a": 1},
			},
			"y2": true,
			"y3": map[string]any{
				"z": 1.2,
			},
		},
		"x1": 50,
	}

	result := ObjectPopulate(template, values)
	if !mapsEqual(result, expected) {
		t.Errorf("ObjectPopulate() = %v, want %v", result, expected)
	}
}

func TestObjectPopulateMissingValues(t *testing.T) {
	template := map[string]any{
		"x": map[string]any{
			"y":  "x.y",
			"y1": "x.y1",
		},
	}

	values := map[string]any{
		"x.y": "present",
		// x.y1 is missing
	}

	expected := map[string]any{
		"x": map[string]any{
			"y": "present",
			// y1 is removed when value is missing
		},
	}

	result := ObjectPopulate(template, values)
	if !mapsEqual(result, expected) {
		t.Errorf("ObjectPopulate with missing values = %v, want %v", result, expected)
	}
}

func TestObjectPopulateEmpty(t *testing.T) {
	result := ObjectPopulate(map[string]any{}, map[string]any{})
	if !mapsEqual(result, map[string]any{}) {
		t.Errorf("ObjectPopulate with empty maps = %v, want empty map", result)
	}
}

func TestObjectPopulateNestedMaps(t *testing.T) {
	template := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "a.b.c",
			},
		},
	}

	values := map[string]any{
		"a.b.c": "deep_value",
	}

	expected := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep_value",
			},
		},
	}

	result := ObjectPopulate(template, values)
	if !mapsEqual(result, expected) {
		t.Errorf("ObjectPopulate with deeply nested maps = %v, want %v", result, expected)
	}
}

func TestGetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		path     string
		expected any
	}{
		{
			name: "simple nested path",
			input: map[string]any{
				"x": map[string]any{
					"y": map[string]any{
						"z": "value",
					},
				},
			},
			path:     "x.y.z",
			expected: "value",
		},
		{
			name: "path with backticks",
			input: map[string]any{
				"x": map[string]any{
					"y with space": map[string]any{
						"z": "value",
					},
				},
			},
			path:     "x.`y with space`.z",
			expected: "value",
		},
		{
			name: "nonexistent path",
			input: map[string]any{
				"x": map[string]any{
					"y": map[string]any{
						"a": "value",
					},
				},
			},
			path:     "x.y.z",
			expected: nil,
		},
		{
			name: "intermediate non-map value",
			input: map[string]any{
				"x": "string_value",
			},
			path:     "x.y.z",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNestedValue(tt.input, tt.path)
			if !interfaceEqual(result, tt.expected) {
				t.Errorf("GetNestedValue(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestObjectPopulateNestedValues(t *testing.T) {
	template := map[string]any{
		"x": map[string]any{
			"y": map[string]any{
				"z": "x.y.z",
			},
			"y1": map[string]any{
				"z1": "x.y1.z1",
				"z2": "x.y1.z2",
			},
		},
	}

	values := map[string]any{
		"x": map[string]any{
			"y": map[string]any{
				"z": "nested_value",
			},
			"y1": map[string]any{
				"z1": 1,
				"z2": map[string]any{"a": 1},
			},
		},
	}

	expected := map[string]any{
		"x": map[string]any{
			"y": map[string]any{
				"z": "nested_value",
			},
			"y1": map[string]any{
				"z1": 1,
				"z2": map[string]any{"a": 1},
			},
		},
	}

	result := ObjectPopulate(template, values)
	if !mapsEqual(result, expected) {
		t.Errorf("ObjectPopulate with nested values = %v, want %v", result, expected)
	}
}

// A null column value in a fixed-schema source (e.g. an Iceberg table) must be
// kept as a present null (-> N1QL NULL), distinct from a genuinely absent path
// (-> N1QL MISSING). This applies whether the null value is top-level or nested.
func TestObjectPopulateNullVsMissing(t *testing.T) {
	template := map[string]any{
		"country": "country",
		"address": map[string]any{
			"city": "address.city",
			"zip":  "address.zip",
		},
	}

	// country: present, null. address.city: present, null.
	// address.zip: absent from the values structure entirely.
	values := map[string]any{
		"country": nil,
		"address": map[string]any{
			"city": nil,
		},
	}

	expected := map[string]any{
		"country": nil, // present-but-null -> kept -> NULL
		"address": map[string]any{
			"city": nil, // present-but-null nested -> kept -> NULL
			// zip is absent from values -> dropped -> MISSING
		},
	}

	result := ObjectPopulate(template, values)
	if !mapsEqual(result, expected) {
		t.Errorf("ObjectPopulate null-vs-missing = %v, want %v", result, expected)
	}

	// Verify GetNestedValueExists distinguishes present-null from absent.
	if v, ok := GetNestedValueExists(values, "address.city"); !ok || v != nil {
		t.Errorf("GetNestedValueExists(address.city) = (%v, %v), want (nil, true)", v, ok)
	}
	if v, ok := GetNestedValueExists(values, "address.zip"); ok || v != nil {
		t.Errorf("GetNestedValueExists(address.zip) = (%v, %v), want (nil, false)", v, ok)
	}
}
