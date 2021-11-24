//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
