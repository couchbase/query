// Copyright 2013-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.
package date_functions

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDateFunctions(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON orders")

	runMatch("case_func_date.json", false, false, qc, t)
	runMatch("case_bugs.json", false, false, qc, t)

	rr := runStmt(qc, "delete from orders where test_id IN [\"datefunc\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}
