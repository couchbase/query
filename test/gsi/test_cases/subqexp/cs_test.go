//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package subqexp

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestSubqexp(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")
	runStmt(qc, "CREATE INDEX ix1 ON shellTest(`from`)")

	runMatch("case_select.json", false, false, qc, t)
	runMatch("case_keyspace.json", false, false, qc, t)
	runMatch("case_keyspace.json", true, false, qc, t)
	runMatch("case_bugs.json", false, false, qc, t)
	runMatch("case_allow_primary_seqscan_for_corr_subq.json", false, false, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ix1")

	rr := runStmt(qc, "delete from orders where test_id = \"subqexp\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "delete from shellTest where test_id = \"subqexp\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "delete from customer")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
