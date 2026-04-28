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
		expected map[string]interface{}
	}{
		{
			name:  "basic nested paths",
			input: []string{"x.y.z", "x.y1.z1", "x.y1.z2", "x.y2", "`x`.`y3`.`z`", "x1"},
			expected: map[string]interface{}{
				"x": map[string]interface{}{
					"y":  map[string]interface{}{"z": "x.y.z"},
					"y1": map[string]interface{}{"z1": "x.y1.z1", "z2": "x.y1.z2"},
					"y2": "x.y2",
					"y3": map[string]interface{}{"z": "x.y3.z"},
				},
				"x1": "x1",
			},
		},
		{
			name:  "single path",
			input: []string{"a.b.c"},
			expected: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "a.b.c",
					},
				},
			},
		},
		{
			name:  "path with spaces in backticks",
			input: []string{"x.`y with space`.z"},
			expected: map[string]interface{}{
				"x": map[string]interface{}{
					"y with space": map[string]interface{}{
						"z": "x.y with space.z",
					},
				},
			},
		},
		{
			name:  "empty backtick field",
			input: []string{"x.``.y"},
			expected: map[string]interface{}{
				"x": map[string]interface{}{
					"": map[string]interface{}{
						"y": "x..y",
					},
				},
			},
		},
		{
			name:  "mixed backticks and regular",
			input: []string{"`a`.b.c", "d.`e`.f"},
			expected: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "a.b.c",
					},
				},
				"d": map[string]interface{}{
					"e": map[string]interface{}{
						"f": "d.e.f",
					},
				},
			},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]interface{}{},
		},
		{
			name:  "single field",
			input: []string{"x"},
			expected: map[string]interface{}{
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
func mapsEqual(a, b map[string]interface{}) bool {
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
func interfaceEqual(a, b interface{}) bool {
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
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		return ok && mapsEqual(av, bv)
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}

func TestObjectPopulate(t *testing.T) {
	template := map[string]interface{}{
		"x": map[string]interface{}{
			"y":  map[string]interface{}{"z": "x.y.z"},
			"y1": map[string]interface{}{"z1": "x.y1.z1", "z2": "x.y1.z2"},
			"y2": "x.y2",
			"y3": map[string]interface{}{"z": "x.y3.z"},
		},
		"x1": "x1",
	}

	values := map[string]interface{}{
		"x.y.z":   "abc",
		"x.y1.z1": 1,
		"x.y1.z2": map[string]interface{}{"a": 1},
		"x.y2":    true,
		"x.y3.z":  1.2,
		"x1":      50,
	}

	expected := map[string]interface{}{
		"x": map[string]interface{}{
			"y": map[string]interface{}{
				"z": "abc",
			},
			"y1": map[string]interface{}{
				"z1": 1,
				"z2": map[string]interface{}{"a": 1},
			},
			"y2": true,
			"y3": map[string]interface{}{
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
	template := map[string]interface{}{
		"x": map[string]interface{}{
			"y":  "x.y",
			"y1": "x.y1",
		},
	}

	values := map[string]interface{}{
		"x.y": "present",
		// x.y1 is missing
	}

	expected := map[string]interface{}{
		"x": map[string]interface{}{
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
	result := ObjectPopulate(map[string]interface{}{}, map[string]interface{}{})
	if !mapsEqual(result, map[string]interface{}{}) {
		t.Errorf("ObjectPopulate with empty maps = %v, want empty map", result)
	}
}

func TestObjectPopulateNestedMaps(t *testing.T) {
	template := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "a.b.c",
			},
		},
	}

	values := map[string]interface{}{
		"a.b.c": "deep_value",
	}

	expected := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
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
		input    map[string]interface{}
		path     string
		expected interface{}
	}{
		{
			name: "simple nested path",
			input: map[string]interface{}{
				"x": map[string]interface{}{
					"y": map[string]interface{}{
						"z": "value",
					},
				},
			},
			path:     "x.y.z",
			expected: "value",
		},
		{
			name: "path with backticks",
			input: map[string]interface{}{
				"x": map[string]interface{}{
					"y with space": map[string]interface{}{
						"z": "value",
					},
				},
			},
			path:     "x.`y with space`.z",
			expected: "value",
		},
		{
			name: "nonexistent path",
			input: map[string]interface{}{
				"x": map[string]interface{}{
					"y": map[string]interface{}{
						"a": "value",
					},
				},
			},
			path:     "x.y.z",
			expected: nil,
		},
		{
			name: "intermediate non-map value",
			input: map[string]interface{}{
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
	template := map[string]interface{}{
		"x": map[string]interface{}{
			"y": map[string]interface{}{
				"z": "x.y.z",
			},
			"y1": map[string]interface{}{
				"z1": "x.y1.z1",
				"z2": "x.y1.z2",
			},
		},
	}

	values := map[string]interface{}{
		"x": map[string]interface{}{
			"y": map[string]interface{}{
				"z": "nested_value",
			},
			"y1": map[string]interface{}{
				"z1": 1,
				"z2": map[string]interface{}{"a": 1},
			},
		},
	}

	expected := map[string]interface{}{
		"x": map[string]interface{}{
			"y": map[string]interface{}{
				"z": "nested_value",
			},
			"y1": map[string]interface{}{
				"z1": 1,
				"z2": map[string]interface{}{"a": 1},
			},
		},
	}

	result := ObjectPopulate(template, values)
	if !mapsEqual(result, expected) {
		t.Errorf("ObjectPopulate with nested values = %v, want %v", result, expected)
	}
}
