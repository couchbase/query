//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"testing"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/mock"
	"github.com/couchbase/query/errors"
)

type queryContextImpl struct {
}

func (ci *queryContextImpl) Credentials() datastore.Credentials {
	return make(datastore.Credentials, 0)
}

func (ci *queryContextImpl) AuthenticatedUsers() datastore.AuthenticatedUsers {
	return datastore.AuthenticatedUsers{"ivanivanov", "petrpetrov"}
}

func TestSystem(t *testing.T) {
	// Use mock to test system; 2 namespaces with 5 keyspaces per namespace
	m, err := mock.NewDatastore("mock:namespaces=2,keyspaces=5,items=5000")
	if err != nil {
		t.Fatalf("failed to create mock store: %v", err)
	}
	datastore.SetDatastore(m)

	// Create systems store with mock m as the ActualStore
	s, err := NewDatastore(m)
	if err != nil {
		t.Fatalf("failed to create system store: %v", err)
	}

	// The systems store should have keyspaces "system", "namespaces", "keyspaces", "indexes"
	p, err := s.NamespaceByName("#system")
	if err != nil {
		t.Fatalf("failed to get system namespace: %v", err)
	}

	pb, err := p.KeyspaceByName("namespaces")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	bb, err := p.KeyspaceByName("keyspaces")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	ib, err := p.KeyspaceByName("indexes")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	ui, err := p.KeyspaceByName("user_info")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	mui, err := p.KeyspaceByName("my_user_info")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	// Should be able to get a Value for UserInfo.
	v, err := s.UserInfo()
	if err != nil {
		t.Fatalf("failed to get stub user info: %v", err)
	}
	if v == nil {
		t.Fatalf("failed to get value for user info: nil")
	}

	// Expect count of 2 namespaces for the namespaces keyspace
	pb_c, err := pb.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || pb_c != 2 {
		t.Fatalf("failed to get expected namespaces keyspace count %v", err)
	}

	// Expect count of 10 for the keyspaces keyspace
	bb_c, err := bb.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || bb_c != 10 {
		t.Fatalf("failed to get expected keyspaces keyspace count %v", err)
	}

	// Expect count of 2 for the user_info keyspace
	ui_c, err := ui.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || ui_c != 2 {
		t.Fatalf("faied to get expect user_info keyspace count %v", err)

	}

	// Expect count of 2 for the my_user_info keyspace
	mui_c, err := mui.Count(&queryContextImpl{})
	if err != nil || mui_c != 2 {
		t.Fatalf("faied to get expect my_user_info keyspace count %v", err)
	}

	// Expect count of 10 for the indexes keyspace (all the primary indexes)
	ib_c, err := ib.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || ib_c != 10 {
		t.Fatalf("failed to get expected indexes keyspace count %v", err)
	}

	// Scan all Primary Index entries of the keyspaces keyspace
	bb_e, err := doPrimaryIndexScan(t, bb)

	// Check for expected and unexpected names:
	if !bb_e["p0/b1"] {
		t.Fatalf("failed to get expected keyspace name from index scan: p0/b1")
	}

	if bb_e["not a name"] {
		t.Fatalf("found unexpected name in index scan")
	}

	// Scan all Primary Index Entries of the user_info keyspace
	ui_e, err := doPrimaryIndexScan(t, ui)
	if err != nil {
		t.Fatalf("unable to scan index of system:user_info: %v", err)
	}
	if !ui_e["ivanivanov"] || !ui_e["petrpetrov"] {
		t.Fatalf("unexpected results from scan of syste:user_info: %v", ui_e)
	}

	// Scan Primary Index Entries of the my_user_info keyspace
	/*
		au := &datastore.AuthenticatedUsers{"ivanivanov"}
		mui_e, err := doPrimaryIndexScanForUsers(t, mui, *au)
		if err != nil {
			t.Fatalf("unable to scan index of system:my_user_info: %v", err)
		}
		if !mui_e["ivanivanov"] || mui_e["petrpetrov"] {
			t.Fatalf("unexpected results from scan of system:my_user_info: %v", ui_e)
		}
	*/

	// Scan all Primary Index entries of the indexes keyspace
	ib_e, err := doPrimaryIndexScan(t, ib)

	// Check for expected and unexpected names:
	if !ib_e["p1/b4/#primary"] {
		t.Fatalf("failed to get expected keyspace name from index scan: p1/b4/#primary")
	}

	if ib_e["p0/b4"] {
		t.Fatalf("found unexpected name in index scan")
	}

	// Fetch on the keyspaces keyspace - expect to find a value for this key:
	vals, errs := bb.Fetch([]string{"p0/b1"}, datastore.NULL_QUERY_CONTEXT)
	if errs != nil {
		t.Fatalf("errors in key fetch %v", errs)
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value == nil) {
		t.Fatalf("failed to fetch expected key from keyspaces keyspace")
	}

	// Fetch on the user_info keyspace - expect to find a value for this key:
	vals, errs = ui.Fetch([]string{"ivanivanov"}, datastore.NULL_QUERY_CONTEXT)
	if errs != nil {
		t.Fatalf("errors in key fetch %v", errs)
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value == nil) {
		t.Fatalf("failed to fetch expected key from keyspaces keyspace")
	}

	// Fetch on the indexes keyspace - expect to find a value for this key:
	vals, errs = ib.Fetch([]string{"p0/b1/#primary"}, datastore.NULL_QUERY_CONTEXT)
	if errs != nil {
		t.Fatalf("errors in key fetch %v", errs)
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value == nil) {
		t.Fatalf("failed to fetch expected key from indexes keyspace")
	}

	// Fetch on the keyspaces keyspace - expect to not find a value for this key:
	vals, errs = bb.Fetch([]string{"p0/b5"}, datastore.NULL_QUERY_CONTEXT)
	if errs == nil {
		t.Fatalf("Expected not found error for key fetch on %s", "p0/b5")
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value != nil) {
		t.Fatalf("Found unexpected key in keyspaces keyspace")
	}

}

type testingContext struct {
	t *testing.T
}

func (this *testingContext) Error(err errors.Error) {
	this.t.Logf("Scan error: %v", err)
}

func (this *testingContext) Warning(wrn errors.Error) {
	this.t.Logf("scan warning: %v", wrn)
}

func (this *testingContext) Fatal(fatal errors.Error) {
	this.t.Logf("scan fatal: %v", fatal)
}

// Helper function to perform a primary index scan on the given keyspace. Returns a map of
// all primary key names.
func doPrimaryIndexScan(t *testing.T, b datastore.Keyspace) (m map[string]bool, excp errors.Error) {
	conn := datastore.NewIndexConnection(&testingContext{t})

	m = map[string]bool{}

	nitems, excp := b.Count(datastore.NULL_QUERY_CONTEXT)
	if excp != nil {
		t.Fatalf("failed to get keyspace count")
		return
	}

	indexers, excp := b.Indexers()
	if excp != nil {
		t.Fatalf("failed to retrieve indexers")
		return
	}

	pindexes, excp := indexers[0].PrimaryIndexes()
	if excp != nil || len(pindexes) < 1 {
		t.Fatalf("failed to retrieve primary indexes")
		return
	}

	idx := pindexes[0]
	go idx.ScanEntries("", nitems, datastore.UNBOUNDED, nil, conn)
	for {
		v, ok := <-conn.EntryChannel()
		if !ok {
			// Channel closed => Scan complete
			return
		}

		m[v.PrimaryKey] = true
	}
}
