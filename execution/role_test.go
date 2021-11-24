/*
Copyright 2017-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package execution

import (
	"testing"

	"github.com/couchbase/query/datastore"
)

func TestValidateRoles(t *testing.T) {
	validRoles := []datastore.Role{
		datastore.Role{Name: "admin"},
		datastore.Role{Name: "query_select", Target: "*"},
	}

	// No such role
	candidates := []datastore.Role{datastore.Role{Name: "foo"}}
	err := validateRoles(candidates, validRoles)
	if err == nil {
		t.Fatalf("Expected failure to validate, but passed: %+v", candidates)
	}

	// Param for unparameterized.
	candidates = []datastore.Role{datastore.Role{Name: "admin", Target: "bucket1"}}
	err = validateRoles(candidates, validRoles)
	if err == nil {
		t.Fatalf("Expected failure to validate, but passed: %+v", candidates)
	}

	// No param for parameterized.
	candidates = []datastore.Role{datastore.Role{Name: "query_select"}}
	err = validateRoles(candidates, validRoles)
	if err == nil {
		t.Fatalf("Expected failure to validate, but passed: %+v", candidates)
	}

	// Works fine.
	candidates = []datastore.Role{datastore.Role{Name: "query_select", Target: "bucket2"}, datastore.Role{Name: "admin"}}
	err = validateRoles(candidates, validRoles)
	if err != nil {
		t.Fatalf("Expected to validate, but failed: %v", err)
	}
}
