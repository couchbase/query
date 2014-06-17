//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package n1ql

import (
	"encoding/json"
	"strings"
)

// Unmarshal a double quoted string. s must begin and end with double
// quotes.
func UnmarshaDoubleQuoted(s string) (t string, e error) {
	if !strings.ContainsRune(s, '\\') {
		return s[1 : len(s)-1], nil
	}

	var rv string
	e = json.Unmarshal([]byte(s), &rv)
	if e == nil {
		t = rv
	}

	return t, e
}

// Unmarshal a single-quoted string. s must begin and end with single
// quotes.
func UnmarshalSingleQuoted(s string) (t string, e error) {
	s = s[1 : len(s)-1]
	s = strings.Replace(s, "''", "'", -1)  // '' escapes '
	s = strings.Replace(s, "\\'", "'", -1) // \' escapes '

	if !strings.ContainsRune(s, '\\') {
		return s, nil
	}

	var rv string
	e = json.Unmarshal([]byte(`"`+s+`"`), &rv)
	if e == nil {
		t = rv
	}

	return t, e
}
