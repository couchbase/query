//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package merge

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on MERGE
func TestMerge(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for MERGE \n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX o_productId on orders(productId)")
	runStmt(qc, "CREATE INDEX purch_customerId_productId on purchase(customerId, DISTINCT ARRAY l.product FOR l IN lineItems END)")
	runStmt(qc, "CREATE INDEX st_source_idx on shellTest(c11, c12)")
	runStmt(qc, "CREATE INDEX st_target_idx on shellTest(c21, c22)")
	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")

	fmt.Println("Running MERGE test cases")

	// test simple MERGE
	runMatch("case_merge_simple.json", false, false, qc, t)

	// test MERGE with index hints
	runMatch("case_merge_indexhint.json", false, true, qc, t)

	// test MERGE with join hints
	runMatch("case_merge_joinhint.json", false, true, qc, t)

	// test MERGE with expression term (as source)
	runMatch("case_merge_expr.json", false, false, qc, t)

	// test MERGE with subquery term (as source)
	runMatch("case_merge_subq.json", false, false, qc, t)

	// test MERGE with complex conditions
	runMatch("case_merge_complex.json", false, false, qc, t)

	// test MERGE with legacy ON KEY clause
	runMatch("case_merge_onkey.json", false, false, qc, t)

	// test MERGE with RETURNING clause
	runMatch("case_merge_returning.json", false, false, qc, t)

	// test MERGE error cases
	runMatch("case_merge_error.json", false, false, qc, t)
	runMatch("case_merge_error_legacy.json", false, false, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX orders.o_productId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_productId")
	runStmt(qc, "DROP INDEX shellTest.st_source_idx")
	runStmt(qc, "DROP INDEX shellTest.st_target_idx")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON purchase")
	runStmt(qc, "CREATE PRIMARY INDEX ON orders")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM purchase")
	runStmt(qc, "DELETE FROM orders")
	runStmt(qc, "DELETE FROM shellTest")

	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON purchase")
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
