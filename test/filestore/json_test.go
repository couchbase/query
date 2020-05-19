//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package test

import (
	"fmt"
	"github.com/couchbase/query/datastore"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func start() *MockServer {
	return Start("dir:", "json", "json")
}

func TestSyntaxErr(t *testing.T) {
	qc := start()

	r, _, err := Run(qc, true, "this is a bad query", nil, nil, "json")
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
	r, _, err = Run(qc, true, "", nil, nil, "json") // empty string query
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
}

func TestRoleStatements(t *testing.T) {
	qc := start()

	pete := datastore.User{Name: "Peter Peterson", Id: "pete", Domain: "local",
		Roles: []datastore.Role{datastore.Role{Name: "cluster_admin"}, datastore.Role{Name: "bucket_admin", Target: "contacts"}}}
	sam := datastore.User{Name: "Sam Samson", Id: "sam", Domain: "local",
		Roles: []datastore.Role{datastore.Role{Name: "replication_admin"}, datastore.Role{Name: "bucket_admin", Target: "products"}}}

	ds := qc.dstore
	ds.PutUserInfo(&pete)
	ds.PutUserInfo(&sam)

	r, _, err := Run(qc, true, "GRANT bucket_admin ON products TO pete, sam", nil, nil, "json")
	if err != nil {
		t.Fatalf("Unable to run GRANT: %s", err.Error())
	}
	if len(r) != 0 {
		t.Fatalf("Expected no return, got %v", r)
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

	r, _, err = Run(qc, true, "REVOKE bucket_admin ON products FROM pete, sam", nil, nil, "json")
	if err != nil {
		t.Fatalf("Unable to run REVOKE: %s", err.Error())
	}
	if len(r) != 0 {
		t.Fatalf("Expected no return, got %v", r)
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

	r, _, err := Run(qc, true, "select 1 + 1", nil, nil, "json")
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	r, _, err = Run(qc, true, "select * from system:keyspaces", nil, nil, "json")
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	r, _, err = Run(qc, true, "select * from default:orders", nil, nil, "json")
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	fileInfos, _ := ioutil.ReadDir("json/default/orders")
	if len(r) != len(fileInfos) {
		fmt.Printf("num results : %#v, fileInfos: %#v\n", len(r), len(fileInfos))
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
		FtestCaseFile(m, qc, "json")
	}

	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, err := FtestCaseFile(m, qc, "json")
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
