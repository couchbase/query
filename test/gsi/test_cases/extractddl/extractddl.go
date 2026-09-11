//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package extractddl

import (
	"testing"

	"github.com/couchbase/query/test/gsi"
)

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

// runAdminStmt runs a statement with full admin credentials, for the statements whose privileges
// the harness's regular (bucket-owner) credentials don't satisfy.
func runAdminStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunAdminStmt(mockServer, q)
}

// mustRun fails the test immediately if a setup statement did not succeed.  Setup failures are
// otherwise invisible - the RunResult is normally discarded - and surface only as unexplained
// missing rows when the extracted DDL is compared.
func mustRun(t *testing.T, rr *gsi.RunResult) {
	t.Helper()
	if rr.Err != nil {
		t.Fatalf("setup statement failed: %v", rr.Err)
	}
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}
