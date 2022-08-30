//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package file

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

func TestFile(t *testing.T) {
	store, err := NewDatastore("../../test/filestore/json")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	namespaceIds, err := store.NamespaceIds()
	if err != nil {
		t.Errorf("failed to get namespace ids: %v", err)
	}

	if len(namespaceIds) != 1 || namespaceIds[0] != "default" {
		t.Errorf("expected 1 namespace id'd default")
	}

	namespace, err := store.NamespaceById("default")
	if err != nil {
		t.Errorf("failed to get namespace: %v", err)
	}

	namespaceNames, err := store.NamespaceNames()
	if err != nil {
		t.Errorf("failed to get namespace names: %v", err)
	}

	if len(namespaceNames) != 1 || namespaceNames[0] != "default" {
		t.Errorf("expected 1 namespace named json")
	}

	fmt.Printf("Found namespaces %v", namespaceNames)

	namespace, err = store.NamespaceByName("default")
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	ks, err := namespace.KeyspaceIds()
	if err != nil {
		t.Errorf("failed to get keyspace ids: %v", err)
	}

	fmt.Printf("Keyspace ids %v", ks)

	keyspace, err := namespace.KeyspaceById("contacts")
	if err != nil {
		t.Errorf("failed to get keyspace by id: contacts")
	}

	_, err = namespace.KeyspaceNames()
	if err != nil {
		t.Errorf("failed to get keyspace names: %v", err)
	}

	keyspace, err = namespace.KeyspaceByName("contacts")
	if err != nil {
		t.Fatalf("failed to get keyspace by name: contacts")
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		t.Errorf("failed to get indexers")
	}

	indexes, err := indexers[0].Indexes()
	if err != nil {
		t.Errorf("failed to get indexes")
	}

	if len(indexes) < 1 {
		t.Errorf("Expected at least 1 index for keyspace")
	}

	pindexes, err := indexers[0].PrimaryIndexes()
	if err != nil {
		t.Errorf("failed to get primary indexes")
	}

	if len(pindexes) < 1 {
		t.Errorf("Expected at least 1 primary index for keyspace")
	}

	index := pindexes[0]

	context := &testingContext{t}
	conn := datastore.NewIndexConnection(context)

	go index.ScanEntries("", math.MaxInt64, datastore.UNBOUNDED, nil, conn)

	var entry *datastore.IndexEntry
	ok := true
	for ok {
		entry, ok = conn.Sender().GetEntry()
		if entry != nil {
			fmt.Printf("\nScanned %s", entry.PrimaryKey)
		} else {
			ok = false
			break
		}
	}

	freds := make(map[string]value.AnnotatedValue, 1)
	key := "fred"
	errs := keyspace.Fetch([]string{key}, freds, datastore.NULL_QUERY_CONTEXT, nil)
	if len(errs) > 0 || len(freds) == 0 {
		t.Errorf("failed to fetch fred: %v", errs)
	}

	// DML test cases

	fred := freds[key]
	var dmlKey value.Pair
	dmlKey.Name = "fred2"
	dmlKey.Value = fred

	_, errs = keyspace.Insert([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	_, errs = keyspace.Update([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	_, errs = keyspace.Upsert([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if len(errs) > 0 {
		t.Errorf("failed to insert fred2: %v", errs)
	}

	dmlKey.Name = "fred3"
	_, errs = keyspace.Upsert([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if len(errs) > 0 {
		t.Errorf("failed to insert fred2: %v", errs)
	}

	// negative cases
	_, errs = keyspace.Insert([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if len(errs) == 0 {
		t.Errorf("Insert should not have succeeded for fred2")
	}

	// delete all the freds

	deleted, errs := keyspace.Delete([]value.Pair{value.Pair{Name: "fred2"}, value.Pair{Name: "fred3"}}, datastore.NULL_QUERY_CONTEXT)
	if len(errs) > 0 && len(deleted) != 2 {
		fmt.Printf("Warning: Failed to delete. Error %v", errs)
	}

	_, errs = keyspace.Update([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if len(errs) == 0 {
		t.Errorf("Update should have failed. Key fred3 doesn't exist")
	}

	// finally upsert the key. this should work
	_, errs = keyspace.Upsert([]value.Pair{dmlKey}, datastore.NULL_QUERY_CONTEXT)
	if len(errs) > 0 {
		t.Errorf("failed to insert fred2: %v", errs)
	}

	// some deletes should fail
	deleted, errs = keyspace.Delete([]value.Pair{value.Pair{Name: "fred2"}, value.Pair{Name: "fred3"}}, datastore.NULL_QUERY_CONTEXT)
	if len(deleted) != 1 && deleted[0].Name != "fred2" {
		t.Errorf("failed to delete fred2: %v, #deleted=%d", deleted, len(deleted))
	}

}

type testingContext struct {
	t *testing.T
}

func (this *testingContext) GetScanCap() int64 {
	return 16
}

func (this *testingContext) MaxParallelism() int {
	return 1
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

func (this *testingContext) GetReqDeadline() time.Time {
	return time.Time{}
}

func (this *testingContext) RecordFtsRU(ru tenant.Unit) {
}

func (this *testingContext) RecordGsiRU(ru tenant.Unit) {
}

func (this *testingContext) RecordKvRU(ru tenant.Unit) {
}

func (this *testingContext) RecordKvWU(wu tenant.Unit) {
}

func (this *testingContext) Credentials() *auth.Credentials {
	return nil
}

func (this *testingContext) SkipKey(key string) bool {
	return false
}
