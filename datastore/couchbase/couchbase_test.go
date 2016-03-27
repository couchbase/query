//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package file provides a couchbase-server implementation of the datasite
package.

*/

package couchbase

import (
	"fmt"
	//"reflect"
	"math"
	"testing"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/value"
)

const TEST_URL = "http://localhost:9000/"

func TestServer(t *testing.T) {

	logger, _ := log_resolver.NewLogger("golog")
	if logger == nil {
		t.Fatalf("Invalid logger")
	}

	logging.SetLogger(logger)

	site, err := NewDatastore(TEST_URL)
	if err != nil {
		t.Skipf("SKIPPING TEST: %v", err)
	}

	namespaceNames, err := site.NamespaceNames()
	if err != nil {
		t.Fatalf("Failed to get Namespace names . error %v", err)
	}

	fmt.Printf("Namespaces in this instance %v", namespaceNames)

	namespace, err := site.NamespaceByName("default")
	if err != nil {
		t.Fatalf("Namespace default not found, error %v", err)
	}

	keyspaceNames, err := namespace.KeyspaceNames()
	if err != nil {
		t.Fatalf(" Cannot fetch keyspaces names. error %v", err)
	}

	fmt.Printf("Keyspaces in this namespace %v", keyspaceNames)

	//connect to beer-sample
	ks, err := namespace.KeyspaceByName("beer-sample")
	if err != nil {
		t.Fatalf(" Cannot connect to beer-sample. Error %v", err)
		return
	}

	indexer, err := ks.Indexer(datastore.VIEW)
	if err != nil {
		fmt.Printf("No indexers found")
		return
	}

	// try create a primary index
	index, err := indexer.CreatePrimaryIndex("", "#primary", nil)
	if err != nil {
		// keep going. maybe index already exists
		fmt.Printf(" Cannot create a primary index on bucket. Error %v", err)
	} else {

		fmt.Printf("primary index created %v", index)
	}

	pair, errs := ks.Fetch([]string{"357", "aass_brewery"})
	if errs != nil {
		t.Fatalf(" Cannot fetch keys errors %v", errs)

	}

	fmt.Printf("Keys fetched %v", pair)
	insertKey := value.Pair{Key: "testBeerKey", Value: value.NewValue(("This is a random test key-value"))}

	_, err = ks.Insert([]value.Pair{insertKey})
	if err != nil {
		t.Fatalf("Cannot insert key %v", insertKey)
	}

	deleted, err := ks.Delete([]string{insertKey.Key})
	if err != nil || (len(deleted) != 1 && deleted[0] != insertKey.Key) {
		t.Fatalf("Failed to delete %v", err)
	}

	pi, err := indexer.PrimaryIndexes()
	if err != nil || len(pi) < 1 {
		fmt.Printf("No primary index found")
		return
	}

	//fmt.Printf(" got primary index %s", pi.name)
	conn := datastore.NewIndexConnection(nil)
	go pi[0].ScanEntries("", math.MaxInt64, datastore.UNBOUNDED, nil, conn)

	var entry *datastore.IndexEntry

	ok := true
	for ok {

		select {
		case entry, ok = <-conn.EntryChannel():
			if ok {
				fmt.Printf("\n primary key %v", entry.PrimaryKey)
			}
		}
	}
}
