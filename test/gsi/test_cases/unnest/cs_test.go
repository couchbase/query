//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package unnest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestUnnestFunc(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON purchase")

	runMatch("case_unnest.json", false, false, qc, t)

	runStmt(qc, "CREATE INDEX idx2 ON shellTest (DISTINCT ARRAY [op.name, META().id] FOR op IN OBJECT_PAIRS(META().id) END)")
	runStmt(qc, "CREATE INDEX iax1 ON shellTest(DISTINCT ARRAY v.x FOR v IN a1 END)")
	runStmt(qc, "CREATE INDEX iax2 ON shellTest(DISTINCT ARRAY v.y FOR v IN a1 END)")
	runStmt(qc, "CREATE INDEX ix22 ON shellTest((ALL ARRAY (ALL ARRAY [op.name, op.val,x] FOR op IN x END) FOR x IN a1 END))")
	runStmt(qc, "CREATE INDEX idx11 ON shellTest(c1)")
	runStmt(qc, "CREATE INDEX ix11 ON shellTest(type)")
	runStmt(qc, "CREATE INDEX ix12 ON shellTest(DISTINCT arr) WHERE type = \"doc\"")
	runStmt(qc, "CREATE INDEX ix101 ON shellTest(ALL ARRAY u.x FOR u IN arr10 END)")
	runStmt(qc, "CREATE INDEX ix102 ON shellTest(ARRAY_CONCAT(a, b))")

	runMatch("case_unnest_scan_bugs.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.idx2")
	runStmt(qc, "DROP INDEX shellTest.iax1")
	runStmt(qc, "DROP INDEX shellTest.iax2")
	runStmt(qc, "DROP INDEX shellTest.ix22")
	runStmt(qc, "DROP INDEX shellTest.idx11")
	runStmt(qc, "DROP INDEX shellTest.ix11")
	runStmt(qc, "DROP INDEX shellTest.ix12")
	runStmt(qc, "DROP INDEX shellTest.ix101")
	runStmt(qc, "DROP INDEX shellTest.ix102")

	rr := runStmt(qc, "delete from purchase where test_id = \"unnest\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON purchase")
}
