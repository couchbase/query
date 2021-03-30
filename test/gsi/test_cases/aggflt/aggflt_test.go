//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package aggflt

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

/*
Basic test to ensure connections to both
Datastore and Couchbase server, work.
*/
func TestWindow(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	// Covered index
	runStmt(qc, "CREATE INDEX aggfltix1 ON orders (c1, c2, c3) WHERE test_id = \"aggflt\"")
	runMatch("testcases.json", false, true, qc, t) // non-prepared, explain
	runMatch("testcases.json", true, true, qc, t)  // prepared, explain
	runStmt(qc, "DROP INDEX orders.aggfltix1")

	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	_, _, errcs := runStmt(qc, "delete from orders where test_id IN [\"aggflt\"]")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
