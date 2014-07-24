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

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/datastore/mock"
	"github.com/couchbaselabs/query/errors"
)

func TestSystem(t *testing.T) {
	// Use mock to test system; 2 namespaces with 5 keyspaces per namespace
	m, err := mock.NewDatastore("mock:namespaces=2,keyspaces=5,items=5000")
	if err != nil {
		t.Errorf("failed to create mock store: %v", err)
	}

	// Create systems store with mock m as the ActualStore
	s, err := NewDatastore(m)
	if err != nil {
		t.Errorf("failed to create system store: %v", err)
	}

	// The systems store should have keyspaces "system", "namespaces", "keyspaces", "indexes"
	p, err := s.NamespaceByName("system")
	if err != nil {
		t.Errorf("failed to get system namespace: %v", err)
	}

	pb, err := p.KeyspaceByName("namespaces")
	if err != nil {
		t.Errorf("failed to get keyspace by name %v", err)
	}

	bb, err := p.KeyspaceByName("keyspaces")
	if err != nil {
		t.Errorf("failed to get keyspace by name %v", err)
	}

	ib, err := p.KeyspaceByName("indexes")
	if err != nil {
		t.Errorf("failed to get keyspace by name %v", err)
	}

	// Expect count of 2 namespaces for the namespaces keyspace
	pb_c, err := pb.Count()
	if err != nil || pb_c != 2 {
		t.Errorf("failed to get expected namespaces keyspace count %v", err)
	}

	// Expect count of 10 for the keyspaces keyspace
	bb_c, err := bb.Count()
	if err != nil || bb_c != 10 {
		t.Errorf("failed to get expected keyspaces keyspace count %v", err)
	}

	// Expect count of 10 for the indexes keyspace (all the primary indexes)
	ib_c, err := ib.Count()
	if err != nil || ib_c != 10 {
		t.Errorf("failed to get expected indexes keyspace count %v", err)
	}

	// Scan all Primary Index entries of the keyspaces keyspace
	bb_e, err := doPrimaryIndexScan(t, bb)

	// Check for expected and unexpected names:
	if !bb_e["p0/b1"] {
		t.Errorf("failed to get expected keyspace name from index scan: p0/b1")
	}

	if bb_e["not a name"] {
		t.Errorf("found unexpected name in index scan")
	}

	// Scan all Primary Index entries of the indexes keyspace
	ib_e, err := doPrimaryIndexScan(t, ib)

	// Check for expected and unexpected names:
	if !ib_e["p1/b4/all_docs"] {
		t.Errorf("failed to get expected keyspace name from index scan: p1/b4/all_docs")
	}

	if ib_e["p0/b4"] {
		t.Errorf("found unexpected name in index scan")
	}

	// Fetch on the keyspaces keyspace - expect to find a value for this key:
	vals, err := bb.Fetch([]string{"p0/b1"})
	if err != nil {
		t.Errorf("error in key fetch %v", err)
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value == nil) {
		t.Errorf("failed to fetch expected key from keyspaces keyspace")
	}

	// Fetch on the indexes keyspace - expect to find a value for this key:
	vals, err = ib.Fetch([]string{"p0/b1/all_docs"})
	if err != nil {
		t.Errorf("error in key fetch %v", err)
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value == nil) {
		t.Errorf("failed to fetch expected key from indexes keyspace")
	}

	// Fetch on the keyspaces keyspace - expect to not find a value for this key:
	vals, err = bb.Fetch([]string{"p0/b5"})
	if err != nil {
		t.Errorf("error in key fetch %v", err)
	}

	if vals == nil || (len(vals) == 1 && vals[0].Value != nil) {
		t.Errorf("Found unexpected key in keyspaces keyspace")
	}

}

// Helper function to perform a primary index scan on the given keyspace. Returns a map of
// all primary key names.
func doPrimaryIndexScan(t *testing.T, b datastore.Keyspace) (m map[string]bool, excp errors.Error) {
	warnChan := make(errors.ErrorChannel)
	errChan := make(errors.ErrorChannel)
	defer close(warnChan)
	defer close(errChan)
	conn := datastore.NewIndexConnection(warnChan, errChan)

	m = map[string]bool{}

	nitems, excp := b.Count()
	if excp != nil {
		t.Errorf("failed to get keyspace count")
		return
	}

	idx, excp := b.IndexByPrimary()
	if excp != nil {
		t.Errorf("failed to retrieve Primary index")
		return
	}

	go idx.ScanEntries(nitems, conn)
	for {
		select {
		case v, conn_open := <-conn.EntryChannel():
			if !conn_open {
				// Channel closed => Scan complete
				return
			}
			m[v.PrimaryKey] = true
		case _excp, _ := <-errChan:
			excp = _excp
			return
		}
	}
}
