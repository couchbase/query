//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package indexga

import (
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/test/gsi"
)

func TestGroupagg(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	// Drop Primary View Index
	runStmt(qc, "DROP PRIMARY INDEX ON orders USING view")

	// Insert the test specific data
	runMatch("insert.json", false, false, qc, t)

	run_test(qc, t, false) // non prepare statements

	run_test(qc, t, true) // prepare statements

	// Delete the test specific data
	case_delete(qc, t)

}

func run_test(qc *gsi.MockServer, t *testing.T, prepare bool) {
	cases := []string{"case_indexga_regular.json",
		"case_indexga_regular_noncoverd.json",
		"case_indexga_regular_or.json",
		"case_indexga_primary.json",
		"case_indexga_unionscan.json",
		"case_indexga_intersectscan.json",
		"case_indexga_bugs.json",
	}
	indexes := []string{
		"CREATE PRIMARY INDEX oprimary ON orders",
		"DROP INDEX orders.oprimary",
		"CREATE INDEX ixgar100 ON orders(c0,c1,c2,c3,c4) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgar100",
		"CREATE INDEX ixgap100 ON orders(c0,c1,c2,c3,c4) PARTITION BY HASH(c0) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgap100",
		"CREATE INDEX ixgatp ON orders(c0,c1,c2,c3,c4) PARTITION BY HASH(c4) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgatp",
		"CREATE INDEX ixgar101 ON orders(c1,c0) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgar101",
		"CREATE INDEX ixgar102 ON orders(c10) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgar102",
		"CREATE INDEX ixga201 ON orders(c1, a1) WHERE test_id = 'indexga' AND type = 'bugs'",
		"DROP INDEX orders.ixga201",
	}

	var primary int
	var testcases []string

	// Run positive pushdowns on regular index
	primary, testcases = buildtestcase(cases, indexes, 0, 0, 7)
	run_testcase(primary, prepare, qc, t, testcases)

	// Run negative non covered queries on regular index
	primary, testcases = buildtestcase(cases, indexes, 1, 0, 7)
	run_testcase(primary, prepare, qc, t, testcases)

	// Run positive pushdowns on regular index with OR
	primary, testcases = buildtestcase(cases, indexes, 2, 0, 7)
	run_testcase(primary, prepare, qc, t, testcases)

	// Run positive pushdowns on primary index
	primary, testcases = buildtestcase(cases, indexes, 3, 0, 1)
	run_testcase(primary, prepare, qc, t, testcases)

	// Run negative pushdowns on Union Scan
	runStmt(qc, indexes[8])
	primary, testcases = buildtestcase(cases, indexes, 4, 2, 7)
	run_testcase(primary, prepare, qc, t, testcases)
	runStmt(qc, indexes[9])

	// Run negative pushdowns on Intersect Scan
	runStmt(qc, indexes[10])
	primary, testcases = buildtestcase(cases, indexes, 5, 2, 7)
	run_testcase(primary, prepare, qc, t, testcases)
	runStmt(qc, indexes[11])

	// misc bugs
	primary, testcases = buildtestcase(cases, indexes, 6, 12, 13)
	run_testcase(primary, prepare, qc, t, testcases)
}

func case_delete(qc *gsi.MockServer, t *testing.T) {
	runStmt(qc, "CREATE PRIMARY INDEX oprimary ON orders")
	runStmt(qc, "DELETE FROM orders WHERE test_id = 'indexga'")
	runStmt(qc, "DROP INDEX orders.oprimary")
}

func buildtestcase(cases, indexes []string, cid, istart, iend int) (primary int, testcases []string) {
	testcases = make([]string, 0, 10)
	testcases = append(testcases, cases[cid])
	for i := istart; i <= iend; i++ {
		testcases = append(testcases, indexes[i])
		if strings.Contains(indexes[i], "CREATE PRIMARY INDEX") {
			primary = i - istart + 1
		}
	}
	return
}

func run_testcase(primarycase int, prepared bool, qc *gsi.MockServer, t *testing.T, testcases []string) {
	var i int

	// Repeat for
	//      1) Using Primary Index non covered scan
	//      2) Secondary Index with coered scan
	//      3) Leading Hash Partion key of Secondary Index with coered scan
	//      4) Trailing Hash Partion key of Secondary Index with coered scan

	for i = 1; i+1 < len(testcases); i += 2 {
		t.Logf("Testing : %v For Index %v \n", testcases[0], testcases[i])
		runStmt(qc, testcases[i]) // CREATE INDEX
		explainCheck := !prepared && (i != primarycase)
		runMatch(testcases[0], prepared, explainCheck, qc, t) // Run the test,disable explain check for primary index
		runStmt(qc, testcases[i+1])                           // Drop the index
	}

}
