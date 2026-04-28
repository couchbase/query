//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"strings"

	json "github.com/couchbase/go_json"
)

// BuildObjectFromDottedPaths constructs a nested object from a slice of dotted notation strings.
// Each string represents a path to a field in the nested object structure.
// The value for each path is the dotted notation string with backticks removed.
//
// The function properly handles backticks around field names, which can contain spaces
// or be empty. Backticks are removed from the leaf values.
//
// Example input:
//
//	[]string{"x.y.z", "x.y1.z1", "x.y1.z2", "x.y2", "`x`.`y3`.`z`", "x1"}
//
// Example output (as a nested map):
//
//	map[string]interface{}{
//	  "x": map[string]interface{}{
//	    "y": map[string]interface{}{"z": "x.y.z"},
//	    "y1": map[string]interface{}{"z1": "x.y1.z1", "z2": "x.y1.z2"},
//	    "y2": "x.y2",
//	    "y3": map[string]interface{}{"z": "x.y3.z"},
//	  },
//	  "x1": "x1",
//	}
func BuildObjectFromDottedPaths(paths []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, path := range paths {
		segments := parseDottedPath(path)
		if len(segments) == 0 {
			continue
		}

		// Navigate/create the nested structure
		current := result
		lastSegment := segments[len(segments)-1]

		for i := 0; i < len(segments)-1; i++ {
			segment := segments[i]
			if _, exists := current[segment]; !exists {
				current[segment] = make(map[string]interface{})
			}

			// Assert that the existing value is a map
			if nestedMap, ok := current[segment].(map[string]interface{}); ok {
				current = nestedMap
			} else {
				// If there's already a leaf value at this path, we can't continue nesting
				// Skip this path or handle accordingly
				current = nil
				break
			}
		}

		// Set the value at the final segment (remove backticks)
		if current != nil {
			// Join segments without backticks to create the lookup key
			pathWithoutBackticks := strings.Join(segments, ".")
			current[lastSegment] = pathWithoutBackticks
		}
	}

	return result
}

// parseDottedPath parses a dotted notation string into field segments,
// properly handling backticks around field names.
//
// Examples:
//
//	"x.y.z"        -> ["x", "y", "z"]
//	"`x`.`y`.`z`"  -> ["x", "y", "z"]
//	"x.`y with space`.z" -> ["x", "y with space", "z"]
//	"`x`.``.`z`"   -> ["x", "", "z"]
func parseDottedPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var segments []string
	var currentSegment strings.Builder
	inBacktick := false
	escapeNext := false

	for i := 0; i < len(path); i++ {
		c := path[i]

		if escapeNext {
			// Handle escaped characters (e.g., \`)
			currentSegment.WriteByte(c)
			escapeNext = false
			continue
		}

		if c == '\\' && inBacktick {
			// Next character is escaped within backticks
			escapeNext = true
			continue
		}

		if c == '`' {
			if inBacktick {
				// Closing backtick
				if currentSegment.Len() > 0 || (i > 0 && path[i-1] == '`') {
					segments = append(segments, currentSegment.String())
					currentSegment.Reset()
				}
				inBacktick = false
			} else {
				// Opening backtick
				if currentSegment.Len() > 0 {
					// Non-backtick content before a backtick
					segments = append(segments, currentSegment.String())
					currentSegment.Reset()
				}
				inBacktick = true
			}
		} else if c == '.' && !inBacktick {
			// Dot outside of backticks is a separator
			if currentSegment.Len() > 0 {
				segments = append(segments, currentSegment.String())
				currentSegment.Reset()
			}
		} else {
			currentSegment.WriteByte(c)
		}
	}

	// Handle any remaining content
	if inBacktick {
		// Unclosed backtick - treat the whole remaining content as a segment
		segments = append(segments, currentSegment.String())
	} else if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}

	return segments
}

// ObjectPopulate populates an object template structure with values from a values map.
// The template is a nested map structure where leaf string values act as keys
// to look up in the values map.
//
// Example:
//
//	template := map[string]interface{}{
//	  "x": map[string]interface{}{
//	    "y": map[string]interface{}{"z": "x.y.z"},
//	    "y1": map[string]interface{}{"z1": "x.y1.z1", "z2": "x.y1.z2"},
//	    "y2": "x.y2",
//	    "y3": map[string]interface{}{"z": "x.y3.z"},
//	  },
//	  "x1": "x1",
//	}
//
//	values := map[string]interface{}{
//	  "x.y.z": "abc",
//	  "x.y1.z1": 1,
//	  "x.y1.z2": map[string]interface{}{"a": 1},
//	  "x.y2": true,
//	  "x.y3.z": 1.2,
//	  "x1": 50,
//	}
//
//	result := ObjectPopulate(template, values)
//	// result == map[string]interface{}{
//	//   "x": map[string]interface{}{
//	//     "y": map[string]interface{}{"z": "abc"},
//	//     "y1": map[string]interface{}{"z1": 1, "z2": map[string]interface{}{"a": 1}},
//	//     "y2": true,
//	//     "y3": map[string]interface{}{"z": 1.2},
//	//   },
//	//   "x1": 50,
//	// }
func ObjectPopulate(template, values map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	objectPopulateRecursive(template, values, result)
	return result
}

// objectPopulateRecursive recursively populates the template structure
func objectPopulateRecursive(template, values, result map[string]interface{}) {
	for key, value := range template {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			// This is a nested map, recurse into it
			nestedResult := make(map[string]interface{})
			objectPopulateRecursive(nestedMap, values, nestedResult)
			// Only add the nested map if it's not empty (or if we want to include empty maps)
			result[key] = nestedResult
		} else if lookupKey, ok := value.(string); ok {
			// This is a leaf value, use it as a dotted path to lookup in values map
			// First try direct lookup (for flat maps)
			if actualValue, exists := values[lookupKey]; exists {
				result[key] = actualValue
			} else {
				// If direct lookup fails, try nested lookup using dotted path
				if nestedValue := GetNestedValue(values, lookupKey); nestedValue != nil {
					result[key] = nestedValue
				}
				// If neither lookup succeeds, the key is removed entirely
			}
		}
		// Non-string leaf values are removed entirely
	}
}

// GetNestedValue extracts a value from a nested map using a dotted path.
// For example, GetNestedValue(map, "x.y.z") will return map["x"]["y"]["z"]
// Returns nil if any part of the path doesn't exist or if a non-map value is encountered
// before reaching the final segment.
func GetNestedValue(m map[string]interface{}, path string) interface{} {
	segments := parseDottedPath(path)
	if len(segments) == 0 {
		return nil
	}

	current := m
	for i := 0; i < len(segments)-1; i++ {
		segment := segments[i]
		if current == nil {
			return nil
		}
		nestedMap, ok := current[segment].(map[string]interface{})
		if !ok {
			return nil
		}
		current = nestedMap
	}

	if current == nil {
		return nil
	}
	return current[segments[len(segments)-1]]
}

func ObjectToValue(row, resultObject map[string]interface{}) (Value, error) {
	if row == nil {
		return NULL_VALUE, nil
	}

	// If resultObject is present, use it as a template and populate with row data
	var finalRow map[string]interface{}
	if resultObject != nil {
		finalRow = ObjectPopulate(resultObject, row)
	} else {
		finalRow = row
	}

	// Marshal row to JSON
	jsonData, err := json.Marshal(finalRow)
	if err != nil {
		return nil, err
	}

	// Create annotated value from parsed JSON
	return NewAnnotatedValue(NewParsedValue(jsonData, true)), nil
}
