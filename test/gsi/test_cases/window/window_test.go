//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package window

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
func TestWindow(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	// Covered index
	runStmt(qc, "CREATE INDEX wix1 ON orders (c1, c2, c3, c4, c5) WHERE test_id = \"window\"")
	runMatch("case_window.json", false, false, qc, t) // non-prepared, no explain
	runMatch("case_window.json", true, false, qc, t)  // prepared, no explain
	runStmt(qc, "DROP INDEX orders.wix1")

	// On Primary index
	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	runMatch("case_window.json", false, false, qc, t) // non-prepared, no explain
	runMatch("case_window.json", true, false, qc, t)  // prepared, no explain

	runMatch("case_windowname.json", false, false, qc, t) // non-prepared, no explain
	runMatch("case_windowname.json", true, false, qc, t)  // prepared, no explain
	_, _, errcs := runStmt(qc, "delete from orders where test_id IN [\"window\"]")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}
