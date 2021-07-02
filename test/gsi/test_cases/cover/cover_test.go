//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package cover

import (
	"os"
	"strings"
	"testing"
)

// Basic test on covering indexes
func TestCover(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runStmt(qc, "CREATE PRIMARY INDEX on shellTest")
	runStmt(qc, "CREATE INDEX ixCover on shellTest(f1, f2)")
	runStmt(qc, "CREATE INDEX ixCover2 on shellTest(k0, k1)")
	runStmt(qc, "CREATE INDEX ixCover3 on shellTest(x, id)")
	runStmt(qc, "CREATE INDEX ixCover4 on shellTest(docid, name)")
	runStmt(qc, "CREATE INDEX ixCover5 on shellTest (email,VMs,join_day) WHERE (10 < join_day)")
	runStmt(qc, "CREATE INDEX ixCover6 on shellTest(main.status)")
	runStmt(qc, "CREATE INDEX ixCover7 on shellTest(main.owner)")

	runMatch("case_cover.json", false, false, qc, t)

	runStmt(qc, "DROP PRIMARY INDEX on shellTest")
	runStmt(qc, "DROP INDEX shellTest.ixCover")
	runStmt(qc, "DROP INDEX shellTest.ixCover2")
	runStmt(qc, "DROP INDEX shellTest.ixCover3")
	runStmt(qc, "DROP INDEX shellTest.ixCover4")
	runStmt(qc, "DROP INDEX shellTest.ixCover5")
	runStmt(qc, "DROP INDEX shellTest.ixCover6")
	runStmt(qc, "DROP INDEX shellTest.ixCover7")

	runStmt(qc, "CREATE INDEX ixCover8 on shellTest(ALL ARRAY v.fname FOR v IN Names END) WHERE type=\"doc\" AND owner=\"xyz\"")
	runStmt(qc, "CREATE INDEX ixCover9 on shellTest((DISTINCT (ARRAY (DISTINCT (ARRAY (((v.country) || \".\") || c) FOR c IN (v.cities) END)) FOR v IN visited_places END)))")

	runMatch("case_cover2.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ixCover8")
	runStmt(qc, "DROP INDEX shellTest.ixCover9")

	runStmt(qc, "CREATE INDEX ixCover10 on shellTest(ALL items)")
	runStmt(qc, "CREATE INDEX ixCover11 on shellTest(ALL ARRAY [v, zipcode] FOR v IN items2 END)")

	runMatch("case_cover3.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ixCover10")
	runStmt(qc, "DROP INDEX shellTest.ixCover11")

	runStmt(qc, "CREATE INDEX ixCover12 on shellTest(ALL ARRAY v.f1 FOR v IN items END)")

	runMatch("case_cover4.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ixCover12")

	// include entire array as separate index key to provide cover
	runStmt(qc, "CREATE INDEX ixCover13 on shellTest(ALL ARRAY v FOR v IN items END, items)")
	runStmt(qc, "CREATE INDEX ixCover14 on shellTest(ALL ARRAY v.f1 FOR v IN items2 END, items2)")

	runMatch("case_cover5.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ixCover13")
	runStmt(qc, "DROP INDEX shellTest.ixCover14")

	runStmt(qc, "CREATE INDEX ixCover15 on shellTest(DISTINCT ARRAY v.f1 FOR v IN items WHEN a = 10 AND b = 20 END)")

	runMatch("case_cover6.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ixCover15")

	runStmt(qc, "CREATE INDEX ixCover16 on shellTest(DISTINCT arr1, to_number(c1), c1)")

	runMatch("case_cover_bugs.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.ixCover16")
}
