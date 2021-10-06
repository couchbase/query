//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package n1qlFts

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Basic test on cobering indexes
func TestN1qlFts(t *testing.T) {

	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" || !isFTSPresent() {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket for N1QL FTS integration test \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "create primary index on product")

	err := setupftsIndex()
	if err != nil {
		t.Logf("did not expect err %s", err.Error())
	}

	time.Sleep(time.Second * 1)

	runMatch("case_fts.json", false, false, qc, t)

	time.Sleep(time.Second * 1)

	err = deleteFTSIndex()
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	_, _, errcs, _ := runStmt(qc, "delete from product where test_id = \"n1qlfts\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "drop primary index on product")
}
