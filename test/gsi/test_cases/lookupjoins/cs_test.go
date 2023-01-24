// Copyright 2013-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.
package lookupjoins

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestLookupjoins(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON purchase")
	runStmt(qc, "CREATE PRIMARY INDEX ON customer")
	runStmt(qc, "CREATE PRIMARY INDEX ON product")

	runMatch("case_innerjoin.json", false, false, qc, t)
	runMatch("case_joins.json", false, false, qc, t)
	runMatch("case_leftjoin.json", false, false, qc, t)

	_, _, errcs := runStmt(qc, "delete from purchase where test_id = \"joins\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	_, _, errcs = runStmt(qc, "delete from customer where test_id = \"joins\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	_, _, errcs = runStmt(qc, "delete from product where test_id = \"joins\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON purchase")
	runStmt(qc, "DROP PRIMARY INDEX ON customer")
	runStmt(qc, "DROP PRIMARY INDEX ON product")
}
