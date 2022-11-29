//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package statements

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestStatements(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE INDEX index_custId ON customer(custId) WHERE test_id = \"delete_statement\"")

	runMatch("case_delete_tests.json", false, true, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON customer")
	runStmt(qc, "DELETE FROM customer WHERE test_id = \"delete_statement\"")

	runStmt(qc, "DROP INDEX customer.index_custId")
	runStmt(qc, "DROP PRIMARY INDEX ON customer")
}
