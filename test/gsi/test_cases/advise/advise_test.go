/*
Copyright 2019-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package advise

import (
	"fmt"
	"os"
	"strings"

	"testing"
)

func TestAdvise(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runMatch("case_virtual.json", false, false, qc, t)
	runMatch("case_advise_select.json", false, false, qc, t)
	runMatch("case_advise_others.json", false, false, qc, t)
	runMatch("case_advise_edgecase.json", false, false, qc, t)
	runMatch("case_advise_pushdown.json", false, false, qc, t)
	runMatch("case_advise_unnest.json", false, false, qc, t)
	runMatch("case_advise_meta.json", false, false, qc, t)
	runMatch("case_advise_lkmissing.json", false, false, qc, t)
	runMatch("case_advise_bugs.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")
	rr := runStmt(qc, "delete from shellTest where test_id IN [\"advise\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
