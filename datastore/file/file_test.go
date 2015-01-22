//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package file

import (
	"fmt"
	"math"
	"testing"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
)

func TestFile(t *testing.T) {
	store, err := NewDatastore("../../test/json")
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

	go index.ScanEntries(math.MaxInt64, datastore.UNBOUNDED, nil, conn)

	ok := true
	for ok {
		entry, ok := <-conn.EntryChannel()
		if ok {
			fmt.Printf("\nScanned %s", entry.PrimaryKey)
		} else {
			break
		}
	}

	freds, err := keyspace.Fetch([]string{"fred"})
	if err != nil || len(freds) == 0 {
		t.Errorf("failed to fetch fred: %v", err)
	}

	// DML test cases

	fred := freds[0].Value
	var dmlKey datastore.Pair
	dmlKey.Key = "fred2"
	dmlKey.Value = fred

	_, err = keyspace.Insert([]datastore.Pair{dmlKey})
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	_, err = keyspace.Update([]datastore.Pair{dmlKey})
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	_, err = keyspace.Upsert([]datastore.Pair{dmlKey})
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	dmlKey.Key = "fred3"
	_, err = keyspace.Upsert([]datastore.Pair{dmlKey})
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	// negative cases
	_, err = keyspace.Insert([]datastore.Pair{dmlKey})
	if err == nil {
		t.Errorf("Insert should not have succeeded for fred2")
	}

	// delete all the freds
	deleted, err := keyspace.Delete([]string{"fred2", "fred3"})
	if err != nil && len(deleted) != 2 {
		fmt.Printf("Warning: Failed to delete. Error %v", err)
	}

	_, err = keyspace.Update([]datastore.Pair{dmlKey})
	if err == nil {
		t.Errorf("Update should have failed. Key fred3 doesn't exist")
	}

	// finally upsert the key. this should work
	_, err = keyspace.Upsert([]datastore.Pair{dmlKey})
	if err != nil {
		t.Errorf("failed to insert fred2: %v", err)
	}

	// some deletes should fail
	deleted, err = keyspace.Delete([]string{"fred2", "fred3"})
	if len(deleted) != 1 && deleted[0] != "fred2" {
		t.Errorf("failed to delete fred2: %v, #deleted=%d", deleted, len(deleted))
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
