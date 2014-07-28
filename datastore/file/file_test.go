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
	"math"
	"testing"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
)

func TestFile(t *testing.T) {
	store, err := NewDatastore("../../test")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	namespaceIds, err := store.NamespaceIds()
	if err != nil {
		t.Errorf("failed to get namespace ids: %v", err)
	}

	if len(namespaceIds) != 1 || namespaceIds[0] != "json" {
		t.Errorf("expected 1 namespace id'd json")
	}

	namespace, err := store.NamespaceById("json")
	if err != nil {
		t.Errorf("failed to get namespace: %v", err)
	}

	namespaceNames, err := store.NamespaceNames()
	if err != nil {
		t.Errorf("failed to get namespace names: %v", err)
	}

	if len(namespaceNames) != 1 || namespaceNames[0] != "json" {
		t.Errorf("expected 1 namespace named json")
	}

	namespace, err = store.NamespaceByName("json")
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	_, err = namespace.KeyspaceIds()
	if err != nil {
		t.Errorf("failed to get keyspace ids: %v", err)
	}

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

	indexes, err := keyspace.Indexes()
	if err != nil {
		t.Errorf("failed ot get indexes")
	}

	if len(indexes) < 1 {
		t.Errorf("Expected at least 1 index for keyspace")
	}

	index, err := keyspace.IndexByPrimary()
	if err != nil {
		t.Fatalf("failed to get primary index: %v", err)
	}

	context := &testingContext{t}
	conn := datastore.NewIndexConnection(context)

	go index.ScanEntries(math.MaxInt64, conn)

	ok := true
	for ok {
		entry, ok := <-conn.EntryChannel()
		if ok {
			t.Logf("Scanned %s", entry.PrimaryKey)
		}
	}

	_, err = keyspace.FetchOne("fred")
	if err != nil {
		t.Errorf("failed to fetch fred: %v", err)
	}
}

type testingContext struct {
	t *testing.T
}

func (this *testingContext) Error(err errors.Error) {
	this.t.Logf("Scan error: %v", err)
}

func (this *testingContext) Warning(wrn errors.Error) {
	this.t.Logf("Scan warning: %v", wrn)
}
