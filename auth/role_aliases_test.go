//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package auth

import (
	"encoding/json"
	"testing"
)

func TestConvertRolesToAliases(t *testing.T) {
	var user map[string]interface{}

	input := `{ "domain":"local", "id":"reviewowner", "name":"OwnerOfreview",
		    "roles":[{"bucket_name":"customer","role":"query_select"},
		             {"bucket_name":"customer","role":"query_insert"}, {"bucket_name":"review","role":"bucket_full_access"}]}`
	err := json.Unmarshal([]byte(input), &user)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}
	ConvertRolesToAliases(user)
	bytes, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Unable to unmarshal")
	}
	strRes := string(bytes)
	exp := `{"domain":"local","id":"reviewowner","name":"OwnerOfreview","roles":[{"bucket_name":"customer","role":"select"},` +
		`{"bucket_name":"customer","role":"insert"},{"bucket_name":"review","role":"bucket_full_access"}]}`
	if strRes != exp {
		t.Fatalf("Result %q, expected %q", strRes, exp)
	}
}

func TestNormalizeRoleNames(t *testing.T) {
	val := []string{"this", "THAT", "select", "QUERY_", "insert_"}
	exp := []string{"this", "that", "query_select", "query_", "insert_"}

	ret := NormalizeRoleNames(val)
	if len(ret) != len(exp) {
		t.Fatalf("Expected %v, got %v", exp, ret)
	}
	for i := range exp {
		if ret[i] != exp[i] {
			t.Fatalf("Expected %v, got %v", exp, ret)
		}
	}
}
