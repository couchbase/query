//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package testfs

import (
	"fmt"
	"path/filepath"
	"testing"
)

/*
Insert data into thei product bucket created earlier
using the statements in insert_product.json.
*/
func TestInsertCaseFiles(t *testing.T) {
	fmt.Println("\n\nInserting values into Bucket for Aggregate Functions \n\n ")
	qc := start()
	matches, err := filepath.Glob("../insert.json")
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}
	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, err := testCaseFile(m, qc)
		if err != nil {
			t.Errorf("Error received : %s \n", err)
			return
		}
		if stmt != "" {
			t.Logf(" %v\n", stmt)
		}
		fmt.Print("\nQuery matched: ", m, "\n\n")
	}
}

func TestAllCaseFiles(t *testing.T) {
	qc := start()
	matches, err := filepath.Glob("../case_*.json")
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}
	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, err := testCaseFile(m, qc)
		if err != nil {
			t.Errorf("Error received : %s \n", err)
			return
		}
		if stmt != "" {
			t.Logf(" %v\n", stmt)
		}
		fmt.Print("\nQuery matched: ", m, "\n\n")
	}
}

func TestCleanupData(t *testing.T) {
	qc := start()

	rr := Run_test(qc, "delete from product where test_id = \"agg_func\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	rr = Run_test(qc, "delete from orders where (test_id = \"agg_func\" OR test_id = \"cntn_agg_func\" OR "+
		"test_id = \"median_agg_func\")")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

}
