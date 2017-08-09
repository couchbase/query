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

// Basic test on cobering indexes
func TestCover(t *testing.T) {
	var RunTest bool

	val := os.Getenv("GSI_TEST")
	if strings.ToLower(val) == "true" {
		RunTest = true
	} else {
		RunTest = false
	}

	if RunTest {
		qc := start_cs()

		runStmt(qc, "CREATE PRIMARY INDEX on shellTest")
		runStmt(qc, "CREATE INDEX ixCover on shellTest(f1, f2)")
		runStmt(qc, "CREATE INDEX ixCover2 on shellTest(k0, k1)")
		runStmt(qc, "CREATE INDEX ixCover3 on shellTest(x, id)")
		runStmt(qc, "CREATE INDEX ixCover4 on shellTest(docid, name)")
		runStmt(qc, "CREATE INDEX ixCover5 on shellTest (email,VMs,join_day) WHERE (10 < join_day)")

		runMatch("case_cover.json", qc, t)

		runStmt(qc, "DROP PRIMARY INDEX on shellTest")
		runStmt(qc, "DROP INDEX shellTest.ixCover")
		runStmt(qc, "DROP INDEX shellTest.ixCover2")
		runStmt(qc, "DROP INDEX shellTest.ixCover3")
		runStmt(qc, "DROP INDEX shellTest.ixCover4")
		runStmt(qc, "DROP INDEX shellTest.ixCover5")
	}
}
