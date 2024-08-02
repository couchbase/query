//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package xattrs

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

/*
Basic test to ensure connections to both
Datastore and Couchbase server, work.
*/
func TestXattrs(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runStmt(qc, "create primary index on product")
	runStmt(qc, "create index idx1 on shellTest(c1) where test_id = 'xattrs'")

	fmt.Println("\n\nInserting values into Bucket for Xattrs test \n\n ")
	runMatch("insert.json", false, false, qc, t)

	gocb_SetupXattr()

	// Test for deleted xattrs
	runStmt(qc, "delete from product where meta().id = 'product0_xattrs'")

	// Test non covering index
	runMatch("case_xattrs.json", false, false, qc, t)

	runStmt(qc, "create index idx2 on product(meta().xattrs.a1, meta().xattrs.b1, meta().xattrs.c1, meta().xattrs.d1,"+
		" meta().xattrs.e1, meta().xattrs.f1, meta().xattrs.g1, meta().xattrs.h1 ,meta().xattrs.i1, meta().xattrs.j1,"+
		" meta().xattrs.k1, meta().xattrs.l1, meta().xattrs.m1, meta().xattrs.n1, meta().xattrs.o1, meta().xattrs.p1)"+
		" where test_id = 'xattrs'")

	// Test bug fixes
	runMatch("case_xattrs_bugs.json", false, true, qc, t)

	runStmt(qc, "drop index shellTest.idx1")
	runStmt(qc, "drop index idx2 on product")

	rr := runStmt(qc, "delete from product where test_id = \"xattrs\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "drop primary index on product")
}
