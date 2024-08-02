//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package xattrs

import (
	"testing"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/couchbase/query/test/gsi"
)

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.
*/
func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func gocb_SetupXattr() {

	cluster, _ := gocb.Connect(gsi.Pool_CBS, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: gsi.Username,
			Password: gsi.Password,
		},
	})

	bucket := cluster.Bucket("product")

	bucket.WaitUntilReady(2*time.Second, nil)
	var _sync = map[string]interface{}{"name": "Information about products", "id": 23}
	var userAttr = [2]string{"Product 1", "Product 10"}

	// Add key-value pairs to hotel_10138, representing traveller-Ids and associated discount percentages
	bucket.DefaultCollection().MutateIn("product0_xattrs", []gocb.MutateInSpec{
		gocb.UpsertSpec("_sync", _sync, &gocb.UpsertSpecOptions{IsXattr: true}),
	}, nil)

	_sync["id"] = 231

	bucket.DefaultCollection().MutateIn("product10_xattrs", []gocb.MutateInSpec{
		gocb.UpsertSpec("_sync", _sync, &gocb.UpsertSpecOptions{IsXattr: true}),
	}, nil)

	bucket.DefaultCollection().MutateIn("product1_xattrs", []gocb.MutateInSpec{
		gocb.UpsertSpec("userAttr", userAttr, &gocb.UpsertSpecOptions{IsXattr: true}),
	}, nil)

	bucket.DefaultCollection().MutateIn("product100_xattrs", []gocb.MutateInSpec{
		gocb.UpsertSpec("a", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 1
		gocb.UpsertSpec("b", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 2
		gocb.UpsertSpec("c", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 3
		gocb.UpsertSpec("d", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 4
		gocb.UpsertSpec("e", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 5
		gocb.UpsertSpec("f", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 6
		gocb.UpsertSpec("g", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 7
		gocb.UpsertSpec("h", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 8
		gocb.UpsertSpec("i", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 9
		gocb.UpsertSpec("j", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 10
		gocb.UpsertSpec("k", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 11
		gocb.UpsertSpec("l", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 12
		gocb.UpsertSpec("m", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 13
		gocb.UpsertSpec("n", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 14
		gocb.UpsertSpec("o", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 15
		gocb.UpsertSpec("p", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 16
	}, nil)

	bucket.DefaultCollection().MutateIn("product100_xattrs_dup", []gocb.MutateInSpec{
		gocb.UpsertSpec("a1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 1
		gocb.UpsertSpec("b1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 2
		gocb.UpsertSpec("c1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 3
		gocb.UpsertSpec("d1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 4
		gocb.UpsertSpec("e1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 5
		gocb.UpsertSpec("f1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 6
		gocb.UpsertSpec("g1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 7
		gocb.UpsertSpec("h1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 8
		gocb.UpsertSpec("i1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 9
		gocb.UpsertSpec("j1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 10
		gocb.UpsertSpec("k1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 11
		gocb.UpsertSpec("l1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 12
		gocb.UpsertSpec("m1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 13
		gocb.UpsertSpec("n1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 14
		gocb.UpsertSpec("o1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 15
		gocb.UpsertSpec("p1", 1, &gocb.UpsertSpecOptions{IsXattr: true}), // 16
	}, nil)

	time.Sleep(time.Millisecond * 10)
}
