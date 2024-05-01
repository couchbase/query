//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package early_projection

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on early projection
func TestEarlyProjection(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for early projection\n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX st_idx1 on shellTest(c11, c12) WHERE type = \"left\"")
	runStmt(qc, "CREATE INDEX st_idx2 on shellTest(c21, c22) WHERE type = \"right\"")

	fmt.Println("Running Early Projection test cases")

	runMatch("case_early_proj.json", false, true, qc, t)
	runMatch("case_early_proj.json", true, true, qc, t)

	runMatch("case_early_proj_more.json", false, true, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX shellTest.st_idx1")
	runStmt(qc, "DROP INDEX shellTest.st_idx2")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM shellTest")
	runStmt(qc, "DELETE FROM product")
	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
