//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ttl

import (
	//	"fmt"
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

	runMatch("insert_test.json", false, false, qc, t)  // non-prepared, no-explain
	runMatch("insert_test.json", true, false, qc, t)   // prepared, no-explain
	runMatch("upsert_test.json", false, false, qc, t)  // non-prepared, no-explain
	runMatch("upsert_test.json", true, false, qc, t)   // prepared, no-explain
	runMatch("update_test.json", false, false, qc, t)  // non-prepared, no-explain
	runMatch("update_test.json", true, false, qc, t)   // prepared, no-explain
	runMatch("merge_test.json", false, false, qc, t)   // non-prepared, no-explain
	runMatch("merge_test.json", true, false, qc, t)    // prepared, no-explain
	runMatch("preserve_ttl.json", false, false, qc, t) // non-prepared, no-explain
	runMatch("preserve_ttl.json", true, false, qc, t)  // prepared, no-explain
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error, int) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
