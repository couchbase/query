//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package recursivecte_function

import (
	"os"
	"strings"
	"testing"
)

func TestRecursiveCteFunction(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runMatch("case_rcte_iterativeTasks.json", false, false, qc, t)

	runMatch("insert.json", false, false, qc, t)

	// options test
	runMatch("case_levelTest.json", false, false, qc, t)

	runMatch("case_docTest.json", false, false, qc, t)

	runMatch("case_cycleTest.json", false, false, qc, t)

	// recursive hierarchial query
	runStmt(qc, "CREATE INDEX cover_idx_employees ON `orders`._default.empSmall( reportsTo, name);")
	runMatch("case_rcte_recursiveHierarchy.json", false, false, qc, t)
	runStmt(qc, "DROP INDEX `cover_idx_employees` ON `orders`._default.empSmall;")

	// graphlookup like query
	runStmt(qc, " CREATE PRIMARY INDEX p_airports ON `orders`._default.airports;")
	runStmt(qc, "CREATE INDEX adv_airport_idx1 ON `orders`.`_default`.`airports`(`airport`, `connects`);")
	runMatch("case_rcte_graphdata.json", false, false, qc, t)
	runStmt(qc, "DROP INDEX p_airports ON `orders`._default.airports")
	runStmt(qc, "DROP INDEX adv_airport_idx1 ON `orders`._default.airports")

	runStmt(qc, "DELETE FROM orders._default.empSmall WHERE name IS NOT MISSING")
	runStmt(qc, "DELETE FROM orders._default.airports WHERE airport IS NOT MISSING")
	runStmt(qc, "DELETE FROM orders._default.travelers WHERE name IS NOT MISSING")
	runStmt(qc, "DELETE FROM shellTest._default.cycleTest WHERE _from IS NOT MISSING")
}
