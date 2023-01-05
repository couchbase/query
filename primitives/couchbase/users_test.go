//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/couchbase/cbauth"
)

func TestGetRolesAll(t *testing.T) {
	t.Skip("Skip this test, as it needs a live connection")

	client, err := ConnectWithAuth("http://Administrator:password@localhost:8091", cbauth.NewAuthHandler(nil), USER_AGENT)
	if err != nil {
		t.Fatalf("Unable to connect: %v", err)
	}
	roles, err := client.GetRolesAll()
	if err != nil {
		t.Fatalf("Unable to get roles: %v", err)
	}

	cases := make(map[string]RoleDescription, 2)
	cases["admin"] = RoleDescription{Role: "admin", Name: "Full Admin",
		Desc: "Can manage all cluster features (including security). This user can access the web console. This user can read and write all data.",
		Ce:   true}
	cases["query_select"] = RoleDescription{Role: "query_select", BucketName: "*", Name: "Query Select",
		Desc: "Can execute a SELECT statement on a given bucket to retrieve data. This user can access the web console and can read data, but not write it."}
	for roleName, expectedValue := range cases {
		foundThisRole := false
		for _, foundValue := range roles {
			if foundValue.Role == roleName {
				foundThisRole = true
				if expectedValue == foundValue {
					break // OK for this role
				}
				t.Fatalf("Unexpected value for role %s. Expected %+v, got %+v", roleName, expectedValue, foundValue)
			}
		}
		if !foundThisRole {
			t.Fatalf("Could not find role %s", roleName)
		}
	}
}

func TestUserUnmarshal(t *testing.T) {
	text := `[{"id":"ivanivanov","name":"Ivan Ivanov","roles":[{"role":"cluster_admin"},{"bucket_name":"default","role":"bucket_admin"}]},
			{"id":"petrpetrov","name":"Petr Petrov","roles":[{"role":"replication_admin"}]}]`
	users := make([]User, 0)

	err := json.Unmarshal([]byte(text), &users)
	if err != nil {
		t.Fatalf("Unable to unmarshal: %v", err)
	}

	expected := []User{
		User{Id: "ivanivanov", Name: "Ivan Ivanov", Roles: []Role{
			Role{Role: "cluster_admin"},
			Role{Role: "bucket_admin", BucketName: "default"}}},
		User{Id: "petrpetrov", Name: "Petr Petrov", Roles: []Role{
			Role{Role: "replication_admin"}}},
	}
	if !reflect.DeepEqual(users, expected) {
		t.Fatalf("Unexpected unmarshalled result. Expected %v, got %v.", expected, users)
	}

	ivanRoles := rolesToParamFormat(users[0].Roles)
	ivanRolesExpected := "cluster_admin,bucket_admin[default]"
	if ivanRolesExpected != ivanRoles {
		t.Errorf("Unexpected param for Ivan. Expected %v, got %v.", ivanRolesExpected, ivanRoles)
	}
	petrRoles := rolesToParamFormat(users[1].Roles)
	petrRolesExpected := "replication_admin"
	if petrRolesExpected != petrRoles {
		t.Errorf("Unexpected param for Petr. Expected %v, got %v.", petrRolesExpected, petrRoles)
	}

}
