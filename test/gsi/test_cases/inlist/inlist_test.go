//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package inlist

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on hash table optimization for IN-list evaluation
func TestInlist(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for IN-list evaluation \n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX st_idx1 on shellTest(c21, c22)")
	runStmt(qc, "CREATE INDEX st_idx2 on shellTest(type, c11)")

	fmt.Println("Running IN-list test cases")

	// test hash table handling of long IN-list evaluation
	runMatch("case_inlist_hash_simple.json", true, false, qc, t)

	// test hash table handling of IN-list with subquery
	runMatch("case_inlist_hash_subquery.json", false, false, qc, t)

	// test dynamic index span expansion on IN-list as host variables
	runMatch("case_inlist_dynamic_span.json", true, true, qc, t)

	// bug fixes
	runMatch("case_inlist_hash_bugs.json", false, false, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX shellTest.st_idx1")
	runStmt(qc, "DROP INDEX shellTest.st_idx2")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM shellTest")

	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}

// MB-65773: an IN-list whose values are covered by different partial indexes
// should use a UnionScan instead of falling back to a primary scan
func TestInlistPartialIndexUnion(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for partial index IN-list evaluation \n\n")
	runMatch("insert_partial.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")

	fmt.Println("Creating partial indexes")
	runStmt(qc, "CREATE INDEX ix_pu_fr ON shellTest(pu_city) WHERE pu_country = \"France\"")
	runStmt(qc, "CREATE INDEX ix_pu_us ON shellTest(pu_city) WHERE pu_country = \"United States\"")

	fmt.Println("Running partial index IN-list test cases")
	runMatch("case_inlist_partial_index.json", false, true, qc, t)
	runMatch("case_inlist_partial_index_params.json", true, true, qc, t)

	// an additional, broader partial index covers the remaining IN-list value
	runStmt(qc, "CREATE INDEX ix_pu_eu ON shellTest(pu_city) WHERE pu_country IN [\"Germany\", \"Spain\", \"Italy\"]")
	runMatch("case_inlist_partial_index_broad.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX shellTest.ix_pu_eu")

	// a regular (non-partial) index serves the whole predicate on its own
	runStmt(qc, "CREATE INDEX ix_pu_country ON shellTest(pu_country, pu_city)")
	runMatch("case_inlist_partial_index_regular.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX shellTest.ix_pu_country")

	// a single partial index whose condition covers every IN-list value
	runStmt(qc, "DROP INDEX shellTest.ix_pu_fr")
	runStmt(qc, "DROP INDEX shellTest.ix_pu_us")
	runStmt(qc, "CREATE INDEX ix_pu_eu ON shellTest(pu_city) WHERE pu_country IN [\"Germany\", \"Spain\", \"Italy\"]")
	runMatch("case_inlist_partial_index_only_broad.json", false, true, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX shellTest.ix_pu_eu")

	// delete the rows inserted by this test
	runStmt(qc, "DELETE FROM shellTest WHERE test_id = \"inlist_partial\"")

	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
