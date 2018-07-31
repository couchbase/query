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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/value"

	// For now we can't use go_json for unmarshalling
	// as it returns a map in a different order than
	// encoding/json (I suspect this is why the change
	// was never made here).
	// use go_json for jsonpointer only.
	jsonpointer "github.com/couchbase/go_json"
)

func start() *MockServer {
	return Start("dir:.", "json")
}

func TestSyntaxErr(t *testing.T) {
	qc := start()

	r, _, err := Run(qc, true, "this is a bad query", nil, nil)
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
	r, _, err = Run(qc, true, "", nil, nil) // empty string query
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
}

func TestRoleStatements(t *testing.T) {
	qc := start()

	pete := datastore.User{Name: "Peter Peterson", Id: "pete", Domain: "local",
		Roles: []datastore.Role{datastore.Role{Name: "cluster_admin"}, datastore.Role{Name: "bucket_admin", Bucket: "contacts"}}}
	sam := datastore.User{Name: "Sam Samson", Id: "sam", Domain: "local",
		Roles: []datastore.Role{datastore.Role{Name: "replication_admin"}, datastore.Role{Name: "bucket_admin", Bucket: "products"}}}

	ds := qc.dstore
	ds.PutUserInfo(&pete)
	ds.PutUserInfo(&sam)

	r, _, err := Run(qc, true, "GRANT bucket_admin ON products TO pete, sam", nil, nil)
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
				datastore.Role{Name: "bucket_admin", Bucket: "contacts"},
				datastore.Role{Name: "bucket_admin", Bucket: "products"},
			},
		},
		datastore.User{
			Name:   "Sam Samson",
			Id:     "sam",
			Domain: "local",
			Roles: []datastore.Role{
				datastore.Role{Name: "replication_admin"},
				datastore.Role{Name: "bucket_admin", Bucket: "products"},
			},
		},
	}
	compareUserLists(&expectedAfterGrant, &users, t)

	r, _, err = Run(qc, true, "REVOKE bucket_admin ON products FROM pete, sam", nil, nil)
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
				datastore.Role{Name: "bucket_admin", Bucket: "contacts"},
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

	r, _, err := Run(qc, true, "select 1 + 1", nil, nil)
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	r, _, err = Run(qc, true, "select * from system:keyspaces", nil, nil)
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 length result")
	}

	r, _, err = Run(qc, true, "select * from default:orders", nil, nil)
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 length result")
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

func dropResultEntry(result interface{}, e string) {
	switch v := result.(type) {
	case map[string]interface{}:
		delete(v, e)
		for _, f := range v {
			dropResultEntry(f, e)
		}
	case []interface{}:
		for _, f := range v {
			dropResultEntry(f, e)
		}
	}
}

func dropResultsEntry(results []interface{}, entry interface{}) {
	e := fmt.Sprintf("%v", entry)
	for _, r := range results {
		dropResultEntry(r, e)
	}
}

func addResultsEntry(newResults, results []interface{}, entry interface{}) {
	e := fmt.Sprintf("%v", entry)
	for i, r := range results {
		v, ok := r.(map[string]interface{})
		if ok {
			newV, ok := newResults[i].(map[string]interface{})
			if ok {
				newV[e] = v[e]
			}
		}
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
			_, _, err := Run(qc, pretty, preStatements, nil, nil)
			if err != nil {
				t.Errorf("preStatements resulted in error: %v, for case file: %v, index: %v", err, fname, i)
			}
		}

		var namedArgs map[string]value.Value
		var positionalArgs value.Values

		if n, ok1 := c["namedArgs"]; ok1 {
			nv := value.NewValue(n)
			size := len(nv.Fields())
			if size == 0 {
				size = 1
			}
			namedArgs = make(map[string]value.Value, size)
			for f, v := range nv.Fields() {
				namedArgs[f] = value.NewValue(v)
			}
		}
		if p, ok2 := c["positionalArgs"]; ok2 {
			if pa, ok3 := p.([]interface{}); ok3 {
				for _, v := range pa {
					positionalArgs = append(positionalArgs, value.NewValue(v))
				}
			}
		}

		v, ok = c["statements"]
		if !ok || v == nil {
			t.Errorf("missing statements for case file: %v, index: %v", fname, i)
			return
		}
		statements := v.(string)
		t.Logf("  %d: %v\n", i, statements)
		resultsActual, _, errActual := Run(qc, pretty, statements, namedArgs, positionalArgs)

		v, ok = c["postStatements"]
		if ok {
			postStatements := v.(string)
			_, _, err := Run(qc, pretty, postStatements, nil, nil)
			if err != nil {
				t.Errorf("postStatements resulted in error: %v, for case file: %v, index: %v", err, fname, i)
			}
		}

		v, ok = c["matchStatements"]
		if ok {
			matchStatements := v.(string)
			resultsMatch, _, errMatch := Run(qc, pretty, matchStatements, nil, nil)
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

		// ignore certain parts of the results if we need to
		// we handle scalars and array of scalars, ignore the rest
		// filter only applied to first level fields
		ignore, ok := c["ignore"]
		if ok {
			switch ignore.(type) {
			case []interface{}:
				for _, v := range ignore.([]interface{}) {
					switch v.(type) {
					case []interface{}:
					case map[string]interface{}:
					default:
						dropResultsEntry(resultsActual, v)
					}
				}
			case map[string]interface{}:
			default:
				dropResultsEntry(resultsActual, ignore)
			}
		}

		// opposite of ignore - only select certain fields
		// again, we handle scalars and the scalars in an array
		accept, ok := c["accept"]
		if ok {
			newResults := make([]interface{}, len(resultsActual))
			switch accept.(type) {
			case []interface{}:
				for i, _ := range resultsActual {
					newResults[i] = make(map[string]interface{}, len(accept.([]interface{})))
				}
				for _, v := range accept.([]interface{}) {
					switch v.(type) {
					case []interface{}:
					case map[string]interface{}:
					default:
						addResultsEntry(newResults, resultsActual, v)
					}
				}
			case map[string]interface{}:
			default:
				for i, _ := range resultsActual {
					newResults[i] = make(map[string]interface{}, 1)
				}
				addResultsEntry(newResults, resultsActual, accept)
			}
			resultsActual = newResults
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
