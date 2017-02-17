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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/couchbase/query/auth"
	"github.com/dustin/go-jsonpointer"
)

func start() *MockServer {
	return Start("dir:.", "json")
}

func TestSyntaxErr(t *testing.T) {
	qc := start()

	r, _, err := Run(qc, true, "this is a bad query")
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
	r, _, err = Run(qc, true, "") // empty string query
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
}

func TestRoleStatements(t *testing.T) {
	qc := start()

	pete := auth.User{Name: "Peter Peterson", Id: "pete",
		Roles: []auth.Role{auth.Role{Name: "cluster_admin"}, auth.Role{Name: "bucket_admin", Keyspace: "customer"}}}
	sam := auth.User{Name: "Sam Samson", Id: "sam",
		Roles: []auth.Role{auth.Role{Name: "replication_admin"}, auth.Role{Name: "bucket_admin", Keyspace: "orders"}}}

	ds := qc.dstore
	ds.PutUserInfo(&pete)
	ds.PutUserInfo(&sam)

	r, _, err := Run(qc, true, "GRANT ROLE cluster_admin, bucket_admin(product) TO pete, sam")
	if err != nil {
		t.Fatalf("Unable to run GRANT ROLE: %s", err.Error())
	}
	if len(r) != 0 {
		t.Fatalf("Expected no return, got %v", r)
	}

	users, err := ds.GetUserInfoAll()
	if err != nil {
		t.Fatalf("Could not get user info after running GRANT ROLE: %s", err.Error())
	}

	expectedAfterGrant := []auth.User{
		auth.User{
			Name: "Peter Peterson",
			Id:   "pete",
			Roles: []auth.Role{
				auth.Role{Name: "cluster_admin"},
				auth.Role{Name: "bucket_admin", Keyspace: "customer"},
				auth.Role{Name: "bucket_admin", Keyspace: "product"},
			},
		},
		auth.User{
			Name: "Sam Samson",
			Id:   "sam",
			Roles: []auth.Role{
				auth.Role{Name: "replication_admin"},
				auth.Role{Name: "cluster_admin"},
				auth.Role{Name: "bucket_admin", Keyspace: "orders"},
				auth.Role{Name: "bucket_admin", Keyspace: "product"},
			},
		},
	}
	compareUserLists(&expectedAfterGrant, &users, t)

	r, _, err = Run(qc, true, "REVOKE ROLE cluster_admin, bucket_admin(product) FROM pete, sam")
	if err != nil {
		t.Fatalf("Unable to run REVOKE ROLE: %s", err.Error())
	}
	if len(r) != 0 {
		t.Fatalf("Expected no return, got %v", r)
	}

	users, err = ds.GetUserInfoAll()
	if err != nil {
		t.Fatalf("Could not get user info after running GRANT ROLE: %s", err.Error())
	}

	expectedAfterRevoke := []auth.User{
		auth.User{
			Name: "Peter Peterson",
			Id:   "pete",
			Roles: []auth.Role{
				auth.Role{Name: "bucket_admin", Keyspace: "customer"},
			},
		},
		auth.User{
			Name: "Sam Samson",
			Id:   "sam",
			Roles: []auth.Role{
				auth.Role{Name: "replication_admin"},
				auth.Role{Name: "bucket_admin", Keyspace: "orders"},
			},
		},
	}
	compareUserLists(&expectedAfterRevoke, &users, t)
}

func compareUserLists(expected *[]auth.User, result *[]auth.User, t *testing.T) {
	if len(*expected) != len(*result) {
		t.Errorf("Expected length %d, got length %d", len(*expected), len(*result))
	}

	for _, expectedUser := range *expected {
		foundUser := false
		var matchResultUser auth.User
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

func compareRoleLists(expected *[]auth.Role, result *[]auth.Role, t *testing.T) {
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

	r, _, err := Run(qc, true, "select 1 + 1")
	if err != nil || len(r) == 0 {
		t.Errorf("did not expect err %s", err.Error())
	}

	r, _, err = Run(qc, true, "select * from system:keyspaces")
	if err != nil || len(r) == 0 {
		t.Errorf("did not expect err %s", err.Error())
	}

	r, _, err = Run(qc, true, "select * from default:orders")
	if err != nil || len(r) == 0 {
		t.Errorf("did not expect err %s", err.Error())
	}

	fileInfos, _ := ioutil.ReadDir("./json/default/orders")
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
		testCaseFile(t, m, qc)
	}
}

func testCaseFile(t *testing.T, fname string, qc *MockServer) {
	t.Logf("testCaseFile: %v\n", fname)
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
		return
	}
	var cases []map[string]interface{}
	err = json.Unmarshal(b, &cases)
	if err != nil {
		t.Errorf("couldn't json unmarshal: %v, err: %v", string(b), err)
		return
	}
	for i, c := range cases {
		d, ok := c["disabled"]
		if ok {
			disabled := d.(bool)
			if disabled == true {
				continue
			}
		}

		pretty := true
		p, ok := c["pretty"]
		if ok {
			pretty = p.(bool)
		}

		v, ok := c["preStatements"]
		if ok {
			preStatements := v.(string)
			_, _, err := Run(qc, pretty, preStatements)
			if err != nil {
				t.Errorf("preStatements resulted in error: %v, for case file: %v, index: %v", err, fname, i)
			}
		}

		v, ok = c["statements"]
		if !ok || v == nil {
			t.Errorf("missing statements for case file: %v, index: %v", fname, i)
			return
		}
		statements := v.(string)
		t.Logf("  %d: %v\n", i, statements)
		resultsActual, _, errActual := Run(qc, pretty, statements)

		v, ok = c["postStatements"]
		if ok {
			postStatements := v.(string)
			_, _, err := Run(qc, pretty, postStatements)
			if err != nil {
				t.Errorf("postStatements resulted in error: %v, for case file: %v, index: %v", err, fname, i)
			}
		}

		v, ok = c["matchStatements"]
		if ok {
			matchStatements := v.(string)
			resultsMatch, _, errMatch := Run(qc, pretty, matchStatements)
			if !reflect.DeepEqual(errActual, errActual) {
				t.Errorf("errors don't match, actual: %#v, expected: %#v"+
					", for case file: %v, index: %v",
					errActual, errMatch, fname, i)
			}
			doResultsMatch(t, resultsActual, resultsMatch, fname, i)
		}

		errExpected := ""
		v, ok = c["error"]
		if ok {
			errExpected = v.(string)
		}
		if errActual != nil {
			if errExpected == "" {
				t.Errorf("unexpected err: %v, statements: %v"+
					", for case file: %v, index: %v", errActual, statements, fname, i)
				return
			}
			// TODO: Check that the actual err matches the expected err.
			continue
		}
		if errExpected != "" {
			t.Errorf("did not see the expected err: %v, statements: %v"+
				", for case file: %v, index: %v", errActual, statements, fname, i)
			return
		}

		v, ok = c["results"]
		if ok {
			resultsExpected := v.([]interface{})
			doResultsMatch(t, resultsActual, resultsExpected, fname, i)
		}

		v, ok = c["resultAssertions"]
		if ok {
			resultAssertions := v.([]interface{})
			for _, rule := range resultAssertions {
				rule, ok := rule.(map[string]interface{})
				if ok {
					pointer, ok := rule["pointer"].(string)
					if ok {
						expectedVal, ok := rule["expect"]
						if ok {
							// FIXME the wrapper object here is temporary
							// while go-jsonpointer API changes slightly
							actualVal := jsonpointer.Get(map[string]interface{}{"wrap": resultsActual}, "/wrap"+pointer)

							if !reflect.DeepEqual(actualVal, expectedVal) {
								t.Errorf("did not see the expected value %v, got %v for pointer: %s", expectedVal, actualVal, pointer)
							}
						} else {
							t.Errorf("expected an expection")
						}
					} else {
						t.Errorf("expected pointer string")
					}
				} else {
					t.Errorf("expected resultAssertions to be objects")
				}
			}

		}

	}
}

func doResultsMatch(t *testing.T, resultsActual, resultsExpected []interface{}, fname string, i int) {
	if len(resultsActual) != len(resultsExpected) {
		t.Errorf("results len don't match, %v vs %v, %v vs %v"+
			", for case file: %v, index: %v",
			len(resultsActual), len(resultsExpected),
			resultsActual, resultsExpected, fname, i)
		return
	}

	if !reflect.DeepEqual(resultsActual, resultsExpected) {
		t.Errorf("results don't match, actual: %#v, expected: %#v"+
			", for case file: %v, index: %v",
			resultsActual, resultsExpected, fname, i)
		return
	}
}
