//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package mockstore

// this test should go in auth, or possibly algebra, but it can't because of circular references
// it lived, at one time, in planner, but since it needs to reference datastore/system, it has
// even more circular references, so its best that it lives in an isolated place

import (
	"testing"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/mock"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/parser/n1ql"
)

func verifyPrivs(t *testing.T, id string, expectedPrivs *auth.Privileges, gotPrivs *auth.Privileges) {
	if expectedPrivs == nil {
		t.Fatalf("Case %s: Unexpected nil in expectedPrivs", id)
	}
	if gotPrivs == nil {
		t.Fatalf("Case %s: Unexpected nil in gotPrivs", id)
	}
	if expectedPrivs.Num() != gotPrivs.Num() {
		t.Fatalf("Case %s: privileges are wrong length. Expected %v, got %v.", id, *expectedPrivs, *gotPrivs)
	}

outer:
	for _, pair := range expectedPrivs.List {
	inner:
		for _, gPair := range gotPrivs.List {
			if pair.Priv != gPair.Priv || pair.Props != gPair.Props {
				continue
			}
			elems := algebra.ParsePath(pair.Target)
			gElems := algebra.ParsePath(gPair.Target)
			if len(elems) != len(gElems) {
				continue
			}

			// we always expect at least one element
			if len(elems) > 1 {
				if elems[0] == "" {
					elems[0] = "default"
				}
				if gElems[0] == "" {
					gElems[0] = "default"
				}
			}
			for i, _ := range elems {
				if elems[i] != gElems[i] {
					continue inner
				}
			}
			break outer
		}
		t.Fatalf("Case %s: Expected pair %v does not appear in received value %v", id, pair, *gotPrivs)
	}
}

type testCase struct {
	id            string
	text          string
	queryContext  string
	expectedPrivs *auth.Privileges
}

func runCase(t *testing.T, c *testCase) {
	stmt, err := n1ql.ParseStatement2(c.text, "default", c.queryContext)
	if err != nil {
		t.Fatalf("Case %s: Unable to parse text: %v", c.id, err)
	}

	privs, err := stmt.Privileges()
	if err != nil {
		t.Fatalf("Case %s: Unable to get privileges of statement: %v", c.id, err)
	}

	verifyPrivs(t, c.id, c.expectedPrivs, privs)
}

func TestStatementPrivileges(t *testing.T) {
	testCases := []testCase{
		//
		// statements with CURL()
		//
		testCase{id: "Simple Select", text: "select CURL('http://ip.jsontest.com') as res",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS}}}},
		testCase{id: "Select in FROM", text: "select * from CURL('http://ip.jsontest.com') res",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS}}}},
		testCase{id: "Select in LIMIT", text: "select * from testbucket limit CURL('http://ip.jsontest.com')",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select in WHERE", text: "select * from testbucket where foo = CURL('http://ip.jsontest.com')",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Insert from VALUES", text: "insert into testbucket values ('foo', CURL('http://ip.jsontest.com'))",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
			}}},
		testCase{id: "Select with USE KEYS", text: "select * from testbucket use keys CURL('http://ip.jsontest.com')",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Update with Subselect",
			text: "update default use keys 'mykey' SET myWebServiceEndpoint = (SELECT raw result FROM CURL('http://ip.jsontest.com') result )",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: "default", Priv: auth.PRIV_QUERY_UPDATE},
			}}},
		//
		// ROLE statements
		//
		testCase{id: "Grant Role",
			text: "grant data_reader on foo to don",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_SECURITY_WRITE},
			}}},
		//
		// SELECT statements
		//
		testCase{id: "Empty Select",
			text:          "SELECT 1",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{}}},
		testCase{id: "Select from One Bucket",
			text: "select * from testbucket",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from Join",
			text: "select * from testbucket a join otherbucket b ON KEYS a.ref",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from Subquery",
			text: "SELECT * FROM testbucket WHERE foo = (SELECT max(bar) FROM otherbucket)",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from Union",
			text: "SELECT * FROM testbucket WHERE foo = (SELECT max(bar) FROM otherbucket)",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select with namespace",
			text: "select * from default:testbucket",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "default:testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from collection",
			text: "select * from testbucket.testscope.testcollection",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket.testscope.testcollection", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from collection with namespace",
			text: "select * from default:testbucket.testscope.testcollection",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "default:testbucket.testscope.testcollection", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from collection with query context",
			text:         "select * from testcollection",
			queryContext: "testbucket.testscope",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket.testscope.testcollection", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from collection with query context with namespace",
			text:         "select * from testcollection",
			queryContext: "default:testbucket.testscope",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "default:testbucket.testscope.testcollection", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select from placeholder",
			text: "select * from $1 as a use keys $2",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "$1", Priv: auth.PRIV_QUERY_SELECT, Props: auth.PRIV_PROPS_DYNAMIC_TARGET},
			}}},
		//
		// INSERT statements
		//
		testCase{id: "Simple Insert",
			text: "INSERT INTO testbucket VALUES ('key1', { 'a' : 'b' })",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
			}}},
		testCase{id: "Insert Select",
			text: "INSERT INTO testbucket (KEY foo, VALUE bar) SELECT foo, bar FROM otherbucket",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Insert with Returning",
			text: "INSERT INTO testbucket VALUES ('key1r', { 'a' : 'b' }) RETURNING meta().cas",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// UPDATE statements
		//
		testCase{id: "Simple Update",
			text: "UPDATE testbucket SET foo = 5",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
			}}},
		testCase{id: "Update with WHERE",
			text: "UPDATE testbucket SET foo = 9 WHERE bar = (SELECT max(id) FROM otherbucket)",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Update with Returning",
			text: "UPDATE testbucket SET foo = 9 WHERE bar = baz RETURNING *",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// DELETE statements
		//
		testCase{id: "Simple Delete",
			text: "DELETE FROM testbucket WHERE f = 10",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_DELETE},
			}}},
		testCase{id: "Delete with Returning",
			text: "DELETE FROM testbucket WHERE f = 9 RETURNING *",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_DELETE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Delete with Subquery",
			text: "DELETE FROM testbucket WHERE f = (SELECT max(foo) FROM otherbucket)",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_DELETE},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// UPSERT statements
		//
		testCase{id: "Upsert with Values",
			text: "UPSERT INTO testbucket VALUES ('key1', { 'a' : 'b' })",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
			}}},
		testCase{id: "Upsert Select",
			text: "UPSERT INTO testbucket (KEY foo, VALUE bar) SELECT foo, bar FROM otherbucket",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Upsert with Returning",
			text: "UPSERT INTO testbucket VALUES ('key1', { 'a' : 'b' }) RETURNING meta().cas",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// EXPLAIN statements
		//
		testCase{id: "Explain Insert",
			text: "EXPLAIN INSERT INTO testbucket (KEY foo, VALUE bar) SELECT foo, bar FROM otherbucket",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Explain Upsert",
			text: "EXPLAIN UPSERT INTO testbucket VALUES ('key1', { 'a' : 'b' }) RETURNING meta().cas",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_INSERT},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// PREPARE statements
		//
		testCase{id: "Prepare Select",
			text: "PREPARE SELECT * FROM testbucket WHERE foo = (SELECT max(bar) FROM otherbucket)",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
				auth.PrivilegePair{Target: "otherbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Prepare Update",
			text: "PREPARE UPDATE testbucket SET foo = 9 WHERE bar = baz RETURNING *",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// INFER statements
		//
		testCase{id: "Infer",
			text: "infer testbucket",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		//
		// MERGE statements
		//
		testCase{id: "Merge with Update Delete",
			text: "MERGE INTO product p USING orders o ON KEY o.productId " +
				"WHEN MATCHED THEN UPDATE SET p.lastSaleDate = o.orderDate " +
				"WHEN MATCHED THEN DELETE WHERE p.inventoryCount  <= 0",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "orders", Priv: auth.PRIV_QUERY_SELECT},
				auth.PrivilegePair{Target: "product", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "product", Priv: auth.PRIV_QUERY_DELETE},
			}}},
		testCase{id: "Merge with Update Insert",
			text: "MERGE INTO all_empts a USING emps_deptb b ON KEY b.empId " +
				"WHEN MATCHED THEN UPDATE SET a.depts = a.depts + 1, a.title = b.title || ', ' || b.title " +
				"WHEN NOT MATCHED THEN " +
				"INSERT  { 'name': b.name, 'title': b.title, 'depts': b.depts, 'empId': b.empId, 'dob': b.dob }",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "emps_deptb", Priv: auth.PRIV_QUERY_SELECT},
				auth.PrivilegePair{Target: "all_empts", Priv: auth.PRIV_QUERY_UPDATE},
				auth.PrivilegePair{Target: "all_empts", Priv: auth.PRIV_QUERY_INSERT},
			}}},
		//
		// system tables
		//
		testCase{id: "system:namespaces",
			text:          "select * from system:namespaces",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{}}},
		testCase{id: "system:keyspaces",
			text:          "select * from system:keyspaces",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{}}},
		testCase{id: "system:prepareds",
			text: "select * from system:prepareds",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_SYSTEM_READ},
			}}},
		//
		// INDEX statements
		//
		testCase{id: "Create Index",
			text: "create index testidx on testbucket(foo)",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_CREATE_INDEX},
			}}},
		testCase{id: "Drop Index",
			text: "drop index testbucket.testidx",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_DROP_INDEX},
			}}},
	}

	n1ql.SetNamespaces(map[string]interface{}{"default": true, "system": true})

	// technically this is not needed, but for completeness...
	m, err := mock.NewDatastore("mock:namespaces=2,keyspaces=5,items=5000")
	if err != nil {
		t.Fatalf("failed to create mock store: %v", err)
	}
	datastore.SetDatastore(m)

	// Create systems store with mock m as the ActualStore
	s, err := system.NewDatastore(m, nil)
	if err != nil {
		t.Fatalf("failed to create system store: %v", err)
	}
	datastore.SetSystemstore(s)
	for _, testCase := range testCases {
		runCase(t, &testCase)
	}
}
