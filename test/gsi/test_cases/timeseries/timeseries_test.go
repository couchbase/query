//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package timeseries

import (
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/test/gsi"
)

func TestTimeSeries(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	// Insert the test specific data
	runMatch("insert.json", false, false, qc, t) // non-prepared, no-explain
	runStmt(qc, "CREATE INDEX ix1 ON orders._default.ts(ticker, ts_start, ts_end)")

	runMatch("case_ts.json", false, false, qc, t) // non-prepared, no-explain
	runMatch("case_ts.json", true, false, qc, t)  // prepared, no-explain

	runStmt(qc, "DELETE FROM orders._default.ts WHERE ticker IS NOT NULL")
	runStmt(qc, "DROP INDEX ix1 ON orders._default.ts")

}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
