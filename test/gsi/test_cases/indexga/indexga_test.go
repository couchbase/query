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
)

func TestGroupagg(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()
	runStmt(qc, "DROP PRIMARY INDEX ON orders USING view")
	runMatch("insert.json", true, qc, t)
	run_regular_testcase(qc, t)
	case_delete(qc, t)

}

func case_delete(qc *MockServer, t *testing.T) {
	runStmt(qc, "CREATE PRIMARY INDEX oprimary ON orders")
	runStmt(qc, "DELETE FROM orders WHERE test_id = 'indexga'")
	runStmt(qc, "DROP INDEX orders.oprimary")
}

func run_testcase(primarycase int, qc *MockServer, t *testing.T, testcases []string) {
	var i int

	// Repeat for
	//      1) Using Primary Index non covered scan
	//      2) Secondary Index with coered scan
	//      3) Leading Hash Partion key of Secondary Index with coered scan
	//      4) Trailing Hash Partion key of Secondary Index with coered scan

	for i = 1; i+1 < len(testcases); i += 2 {
		t.Logf("Testing : %v For Index %v \n", testcases[0], testcases[i])
		runStmt(qc, testcases[i])                         // CREATE INDEX
		runMatch(testcases[0], (i != primarycase), qc, t) // Run the test,disable explain check for primary index
		runStmt(qc, testcases[i+1])                       // Drop the index
	}

}

func run_regular_testcase(qc *MockServer, t *testing.T) {
	// regular Secondary index
	regular := []string{"case_indexga_regular.json",
		"CREATE PRIMARY INDEX oprimary ON orders",
		"DROP INDEX orders.oprimary",
		"CREATE INDEX ixgar100 ON orders(c0,c1,c2,c3,c4) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgar100",
		"CREATE INDEX ixgap100 ON orders(c0,c1,c2,c3,c4) PARTITION BY HASH(c0) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgap100",
		"CREATE INDEX ixgatp ON orders(c0,c1,c2,c3,c4) PARTITION BY HASH(c4) WHERE test_id = 'indexga' AND type = 'numeric'",
		"DROP INDEX orders.ixgatp"}
	run_testcase(1, qc, t, regular)
}
