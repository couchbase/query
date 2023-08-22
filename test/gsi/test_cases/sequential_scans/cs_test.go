//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ss_functions

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestSequentialScans(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	// Tests require an empty bucket as we don't want to add the test_id filter to statements therein
	rr := runStmt(qc, "DELETE FROM orders")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "DELETE FROM orders._default.ss")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	fmt.Println("\n\nInserting test data...\n\n ")
	runMatch("insert.json", false, false, qc, t)

	runMatch("case_ss.json", false, false, qc, t)

	rr = runStmt(qc, "DELETE FROM orders WHERE test_id = \"ss\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "DELETE FROM orders._default.ss")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
}
