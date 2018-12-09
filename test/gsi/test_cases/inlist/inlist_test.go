//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
