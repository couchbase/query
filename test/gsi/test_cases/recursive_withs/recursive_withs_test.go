//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package recursive_withs

import (
	"os"
	"strings"
	"testing"
)

func TestRecursiveWiths(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	// insert documents
	runMatch("insert.json", false, false, qc, t)

	// create indexs
	runStmt(qc, "CREATE INDEX e_idx ON `orders`._default.empSmall(reportsTo INCLUDE MISSING, name);")
	runStmt(qc, " CREATE INDEX c_id ON `shellTest`._default.cycleTest(_from, _to);")

	// queries
	runMatch("case_employee_hierarchy.json", false, true, qc, t)
	runMatch("case_cycle_clause", false, false, qc, t)
	runMatch("case_semcheck.json", false, false, qc, t)
	runMatch("case_with_options.json", false, false, qc, t)

	// clean up
	runStmt(qc, "DROP INDEX e_idx ON `orders`._default.empSmall;")
	runStmt(qc, "DROP INDEX c_id ON `shellTest`._default.cycleTest;")
	runStmt(qc, "DELETE FROM orders._default.empSmall WHERE name IS NOT MISSING")
	runStmt(qc, "DELETE FROM shellTest._default.cycleTest WHERE _from IS NOT MISSING")
	runStmt(qc, "DELETE FROM purchase WHERE name IS NOT MISSING")
}
