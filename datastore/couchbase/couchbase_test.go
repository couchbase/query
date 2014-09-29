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

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/logging/resolver"
	"github.com/couchbaselabs/query/value"
)

const TEST_URL = "http://localhost:9000/"

func TestServer(t *testing.T) {

	logger, _ := resolver.NewLogger("golog")
	if logger == nil {
		fmt.Printf("Invalid logger: %s\n", "logger")
		return
	}

	logging.SetLogger(logger)

	site, err := NewDatastore(TEST_URL)

	if err != nil {
		t.Errorf("Failed to create new datastore. %v", err)
	}

	namespaceNames, err := site.NamespaceNames()
	if err != nil {
		t.Errorf("Failed to get Namespace names . error %v", err)
	}

	fmt.Printf("Namespaces in this instance %v", namespaceNames)

	namespace, err := site.NamespaceByName("default")
	if err != nil {
		t.Errorf("Namespace default not found, error %v", err)
	}

	keyspaceNames, err := namespace.KeyspaceNames()
	if err != nil {
		t.Errorf(" Cannot fetch keyspaces names. error %v", err)
	}

	fmt.Printf("Keyspaces in this namespace %v", keyspaceNames)

	//connect to beer-sample
	ks, err := namespace.KeyspaceByName("beer-sample")
	if err != nil {
		t.Errorf(" Cannot connect to beer-sample. Error %v", err)
		return
	}

	// try create a primary index
	index, err := ks.CreatePrimaryIndex()
	if err != nil {
		t.Errorf(" Cannot create a primary index on bucket. Error %v", err)
	}

	fmt.Printf("primary index created %v", index)

	pair, err := ks.Fetch([]string{"357", "aass_brewery"})
	if err != nil {
		t.Errorf(" Cannot fetch keys error %v", err)

	}

	fmt.Printf("Keys fetched %v", pair)
	insertKey := datastore.Pair{Key: "testBeerKey", Value: value.NewValue(("This is a random test key-value"))}

	_, err = ks.Insert([]datastore.Pair{insertKey})
	if err != nil {
		t.Errorf("Cannot insert key %v", insertKey)
	}

	err = ks.Delete([]string{insertKey.Key})
	if err != nil {
		t.Errorf("Failed to delete %v", err)
	}

	pi, err := ks.IndexByPrimary()
	if err != nil {
		fmt.Printf("No primary index found")
		return
	}

	if pi == nil {
		fmt.Printf("no primary index found")
		return
	}

	//fmt.Printf(" got primary index %s", pi.name)
	conn := datastore.NewIndexConnection(nil)
	go pi.ScanEntries(math.MaxInt64, conn)

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
