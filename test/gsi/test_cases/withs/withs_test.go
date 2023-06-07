//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package withs

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on with clause (common table expression)
func TestWiths(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for withs\n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX st_idx1 on shellTest(c11, c12) WHERE type = \"left\"")
	runStmt(qc, "CREATE INDEX st_idx2 on shellTest(c21, c22) WHERE type = \"right\"")
	runStmt(qc, "CREATE INDEX st_idx3 on shellTest(c12, a11, c11) WHERE type = \"left\"")

	fmt.Println("Running Withs test cases")

	runMatch("case_withs_simple.json", false, true, qc, t)

	runMatch("case_withs_setop.json", false, true, qc, t)

	runMatch("case_withs_bugs.json", false, true, qc, t)
	runMatch("case_withs_bugs.json", true, true, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX shellTest.st_idx1")
	runStmt(qc, "DROP INDEX shellTest.st_idx2")
	runStmt(qc, "DROP INDEX shellTest.st_idx3")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM shellTest")

	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
