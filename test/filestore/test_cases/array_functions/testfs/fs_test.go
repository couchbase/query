//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package testfs

import (
	"fmt"
	"path/filepath"
	"testing"
)

/*
Insert data into the orders bucket created earlier
using the statements in insert_orders.json.
*/
func TestInsertCaseFiles(t *testing.T) {
	fmt.Println("\n\nInserting values into Bucket for Array Functions \n\n ")
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

	_, _, errfs := Run_test(qc, "delete from orders where test_id = \"array_func\"")
	if errfs != nil {
		t.Errorf("did not expect err %s", errfs.Error())
	}
}
