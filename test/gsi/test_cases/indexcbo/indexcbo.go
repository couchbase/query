//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package indexcbo

import (
	"testing"
	"time"

	"github.com/couchbase/query/test/gsi"
)

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

func mustRun(t *testing.T, mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.MustRunStmt(t, mockServer, q)
}

func retryRun(t *testing.T, mockServer *gsi.MockServer, q string, retries int, interval time.Duration) *gsi.RunResult {
	return gsi.RetryRunStmt(t, mockServer, q, retries, interval)
}

func mustRunAdmin(t *testing.T, mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.MustRunAdminStmt(t, mockServer, q)
}

func retryRunAdmin(t *testing.T, mockServer *gsi.MockServer, q string, retries int, interval time.Duration) *gsi.RunResult {
	return gsi.RetryRunAdminStmt(t, mockServer, q, retries, interval)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
