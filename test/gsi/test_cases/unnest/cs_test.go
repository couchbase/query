//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package unnest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestUnnestFunc(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON purchase")

	runMatch("case_unnest.json", false, false, qc, t)

	runStmt(qc, "CREATE INDEX idx2 ON shellTest (DISTINCT ARRAY [op.name, META().id] FOR op IN OBJECT_PAIRS(META().id) END)")
	runStmt(qc, "CREATE INDEX iax1 ON shellTest(DISTINCT ARRAY v.x FOR v IN a1 END)")
	runStmt(qc, "CREATE INDEX iax2 ON shellTest(DISTINCT ARRAY v.y FOR v IN a1 END)")
	runStmt(qc, "CREATE INDEX ix22 ON shellTest((ALL ARRAY (ALL ARRAY [op.name, op.val,x] FOR op IN x END) FOR x IN a1 END))")
	runStmt(qc, "CREATE INDEX idx11 ON shellTest(c1)")

	runMatch("case_unnest_scan_bugs.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX shellTest.idx2")
	runStmt(qc, "DROP INDEX shellTest.iax1")
	runStmt(qc, "DROP INDEX shellTest.iax2")
	runStmt(qc, "DROP INDEX shellTest.ix22")
	runStmt(qc, "DROP INDEX shellTest.idx11")

	_, _, errcs := runStmt(qc, "delete from purchase where test_id = \"unnest\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON purchase")
}
