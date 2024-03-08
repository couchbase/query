//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package test

import (
	"fmt"
	"github.com/couchbase/query/datastore"
	"io/ioutil"
	"path/filepath"
	"testing"
)

const _NAMESPACE = "default"

func start() *MockServer {
	return Start("dir:", "json", _NAMESPACE)
}

func TestSyntaxErr(t *testing.T) {
	qc := start()

	rr := Run(qc, true, "this is a bad query", nil, nil, _NAMESPACE)
	if rr.Err == nil || len(rr.Results) != 0 {
		t.Errorf("expected err")
	}
	rr = Run(qc, true, "", nil, nil, _NAMESPACE) // empty string query
	if rr.Err == nil || len(rr.Results) != 0 {
		t.Errorf("expected err")
	}
}

func TestRoleStatements(t *testing.T) {
	qc := start()

	pete := datastore.User{Name: "Peter Peterson", Id: "pete", Domain: "local",
		Roles: []datastore.Role{datastore.Role{Name: "cluster_admin"}, datastore.Role{Name: "bucket_admin", Target: "contacts"}}}
	sam := datastore.User{Name: "Sam Samson", Id: "sam", Domain: "local",
		Roles: []datastore.Role{datastore.Role{Name: "replication_admin"}, datastore.Role{Name: "bucket_admin",
			Target: "products"}}}

	ds := qc.dstore
	ds.PutUserInfo(&pete)
	ds.PutUserInfo(&sam)

	rr := Run(qc, true, "GRANT bucket_admin ON products TO pete, sam", nil, nil, _NAMESPACE)
	if rr.Err != nil {
		t.Fatalf("Unable to run GRANT: %s", rr.Err.Error())
	}
	if len(rr.Results) != 0 {
		t.Fatalf("Expected no return, got %v", rr.Results)
	}

	users, err := ds.GetUserInfoAll()
	if err != nil {
		t.Fatalf("Could not get user info after running GRANT ROLE: %s", err.Error())
	}

	expectedAfterGrant := []datastore.User{
		datastore.User{
			Name:   "Peter Peterson",
			Id:     "pete",
			Domain: "local",
			Roles: []datastore.Role{
				datastore.Role{Name: "cluster_admin"},
				datastore.Role{Name: "bucket_admin", Target: "contacts"},
				datastore.Role{Name: "bucket_admin", Target: "products"},
			},
		},
		datastore.User{
			Name:   "Sam Samson",
			Id:     "sam",
			Domain: "local",
			Roles: []datastore.Role{
				datastore.Role{Name: "replication_admin"},
				datastore.Role{Name: "bucket_admin", Target: "products"},
			},
		},
	}
	compareUserLists(&expectedAfterGrant, &users, t)

	rr = Run(qc, true, "REVOKE bucket_admin ON products FROM pete, sam", nil, nil, _NAMESPACE)
	if rr.Err != nil {
		t.Fatalf("Unable to run REVOKE: %s", rr.Err.Error())
	}
	if len(rr.Results) != 0 {
		t.Fatalf("Expected no return, got %v", rr.Results)
	}

	users, err = ds.GetUserInfoAll()
	if err != nil {
		t.Fatalf("Could not get user info after running REVOKE: %s", err.Error())
	}

	expectedAfterRevoke := []datastore.User{
		datastore.User{
			Name:   "Peter Peterson",
			Id:     "pete",
			Domain: "local",
			Roles: []datastore.Role{
				datastore.Role{Name: "cluster_admin"},
				datastore.Role{Name: "bucket_admin", Target: "contacts"},
			},
		},
		datastore.User{
			Name:   "Sam Samson",
			Id:     "sam",
			Domain: "local",
			Roles: []datastore.Role{
				datastore.Role{Name: "replication_admin"},
			},
		},
	}
	compareUserLists(&expectedAfterRevoke, &users, t)
}

func compareUserLists(expected *[]datastore.User, result *[]datastore.User, t *testing.T) {
	if len(*expected) != len(*result) {
		t.Errorf("Expected length %d, got length %d", len(*expected), len(*result))
	}

	for _, expectedUser := range *expected {
		foundUser := false
		var matchResultUser datastore.User
		for _, resultUser := range *result {
			if resultUser.Id == expectedUser.Id {
				foundUser = true
				matchResultUser = resultUser
				break
			}
		}
		if foundUser {
			if expectedUser.Name != matchResultUser.Name {
				t.Errorf("Expected user name %s, got %s", expectedUser.Name, matchResultUser.Name)
			}
			compareRoleLists(&expectedUser.Roles, &matchResultUser.Roles, t)
		} else {
			t.Errorf("Unable to find expected user id %s", expectedUser.Id)
		}
	}
}

func compareRoleLists(expected *[]datastore.Role, result *[]datastore.Role, t *testing.T) {
	if len(*expected) != len(*result) {
		t.Errorf("Mismatching length of role lists. Expected %v, got %v.", *expected, *result)
		return
	}
	for _, expRole := range *expected {
		found := false
		for _, resultRole := range *result {
			if expRole == resultRole {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Result list does not contain expected role. Expected role %v, result list %v.", expRole, *result)
		}
	}
}

func TestSimpleSelect(t *testing.T) {
	qc := start()

	rr := Run(qc, true, "select 1 + 1", nil, nil, _NAMESPACE)
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	if len(rr.Results) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	rr = Run(qc, true, "select * from system:keyspaces", nil, nil, _NAMESPACE)
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	if len(rr.Results) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	rr = Run(qc, true, "select * from default:orders", nil, nil, _NAMESPACE)
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	if len(rr.Results) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	fileInfos, _ := ioutil.ReadDir("json/default/orders")
	if len(rr.Results) != len(fileInfos) {
		fmt.Printf("num results : %#v, fileInfos: %#v\n", len(rr.Results), len(fileInfos))
		t.Errorf("expected # of results to match directory listing")
	}
}

func TestAllCaseFiles(t *testing.T) {
	qc := start()
	matches, err := filepath.Glob("json/default/cases/case_*.json")
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}
	for _, m := range matches {
		FtestCaseFile(m, qc, _NAMESPACE)
	}

	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, err := FtestCaseFile(m, qc, _NAMESPACE)
		if err != nil {
			t.Errorf("Error received : %s \n", err)
			return
		}
		if stmt != "" {
			t.Logf(" %v\n", stmt)
		}
		fmt.Print("\nQuery matched: ", m, "\n\n")
	}

}
