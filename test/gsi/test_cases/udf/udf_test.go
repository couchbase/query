//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package udf

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestUDFs(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON customer")

	runMatch("case_inline_udf_tests.json", false, true, qc, t)
	runMatch("test.json", false, false, qc, t)

	// Drop functions created in the tests
	runStmt(qc, "DROP FUNCTION inline1 IF EXISTS")
	runStmt(qc, "DROP FUNCTION inline2 IF EXISTS")
	runStmt(qc, "DROP FUNCTION inline3 IF EXISTS")

	runStmt(qc, "DELETE FROM customer WHERE test_id = \"udf\"")
	runStmt(qc, "DROP PRIMARY INDEX ON customer")
}
