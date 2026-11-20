//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan_stability

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/test/gsi"
)

// Basic test on plan stability
func TestPlanStability(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for plan stability\n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX st_idx1 on shellTest(c11, c12) WHERE type = \"left\"")
	runStmt(qc, "CREATE INDEX st_idx2 on shellTest(c21, c22) WHERE type = \"right\"")

	fmt.Println("Running Plan Stability test cases")

	runMatch("case_plan_stability_start.json", false, false, qc, t)

	runMatch("case_plan_stability_simple.json", false, true, qc, t)

	runMatch("case_plan_stability_subquery.json", false, true, qc, t)

	// in case the functions are left over from previous runs
	runStmt(qc, "DROP FUNCTION UDF_PlanStability_inline1 IF EXISTS")
	runStmt(qc, "DROP FUNCTION UDF_PlanStability_inline2 IF EXISTS")
	// scope functions need a role the default test users don't have
	runStmtAsAdmin(qc, "DROP FUNCTION default:shellTest._default.UDF_PlanStability_inline3 IF EXISTS")
	runStmt(qc, "DROP FUNCTION UDF_PlanStability_inline4 IF EXISTS")
	runStmt(qc, "DROP FUNCTION UDF_PlanStability_inline5 IF EXISTS")

	runMatch("case_plan_stability_inline_udf.json", false, true, qc, t)

	// plan stability is off at this point (case_plan_stability_inline_udf.json), QUERY_METADATA exists
	runMatch("case_plan_stability_save.json", false, false, qc, t)

	// MB-75506: saved plans evicted from the prepared cache, which is no longer full
	runMatch("case_plan_stability_evict_setup.json", false, false, qc, t)
	evictPrepareds(qc)
	runMatch("case_plan_stability_evict_check.json", false, false, qc, t)

	// MB-75507: under FLEXIBLE, a saved plan whose index was dropped is replaced, once
	runStmt(qc, "DROP INDEX shellTest.mb75507_idx IF EXISTS")
	runMatch("case_plan_stability_replan_setup.json", false, false, qc, t)
	evictPrepareds(qc)
	runMatch("case_plan_stability_replan_check1.json", false, false, qc, t)
	evictPrepareds(qc)
	runMatch("case_plan_stability_replan_check2.json", false, false, qc, t)

	// MB-75514: ad hoc statements with a query context, and in a transaction, under AD_HOC
	runMatch("case_plan_stability_adhoc_context.json", false, false, qc, t)

	// MB-75515: inline UDF entries loaded from QUERY_METADATA, removed, and UDFs without subquery
	runStmt(qc, "DROP FUNCTION UDF_PlanStability_inline6 IF EXISTS")
	runStmt(qc, "DROP FUNCTION UDF_PlanStability_inline7 IF EXISTS")
	runStmt(qc, "DROP INDEX shellTest.mb75515_idx IF EXISTS")
	runMatch("case_plan_stability_inline_udf_loaded.json", false, false, qc, t)

	runMatch("case_plan_stability_stop.json", false, false, qc, t)

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

// evict every statement from the prepared cache: with a limit of 1, adding a statement evicts the others; the
// cache is then left not full
func evictPrepareds(qc *gsi.MockServer) {
	limit := prepareds.PreparedsLimit()
	defer prepareds.PreparedsSetLimit(limit)
	prepareds.PreparedsSetLimit(1)
	runStmt(qc, "PREPARE FORCE mb75506_evict FROM SELECT 1")
}
