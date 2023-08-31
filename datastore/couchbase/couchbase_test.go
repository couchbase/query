//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/value"
)

var TEST_URL = "http://localhost:9000/"

func init() {
	// For constructing URLs with raw IPv6 addresses- the IPv6 address
	// must be enclosed within ‘[‘ and ‘]’ brackets.
	TEST_URL = "http://" + server.GetIP(true) + ":9000"
}

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

	pair := make(map[string]value.AnnotatedValue, 1)
	errs := ks.Fetch([]string{"357", "aass_brewery"}, pair, datastore.NULL_QUERY_CONTEXT, nil, nil)
	if len(errs) > 0 {
		t.Fatalf(" Cannot fetch keys errors %v", errs)

	}

	fmt.Printf("Keys fetched %v", pair)
	insertKey := value.Pair{Name: "testBeerKey", Value: value.NewValue(("This is a random test key-value"))}

	_, _, errs = ks.Insert([]value.Pair{insertKey}, datastore.NULL_QUERY_CONTEXT, false)
	if len(errs) > 0 {
		t.Fatalf("Cannot insert key %v", insertKey)
	}

	_, deleted, errs := ks.Delete([]value.Pair{insertKey}, datastore.NULL_QUERY_CONTEXT, true)
	if len(errs) > 0 || (len(deleted) != 1 && deleted[0].Name != insertKey.Name) {
		t.Fatalf("Failed to delete %v", errs)
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

		entry, ok = conn.Sender().GetEntry()
		if ok {
			fmt.Printf("\n primary key %v", entry.PrimaryKey)
		}
	}
}
