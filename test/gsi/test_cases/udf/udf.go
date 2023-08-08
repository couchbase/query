// Copyright 2023-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.
package udf

import (
	"testing"

	"github.com/couchbase/query/test/gsi"
)

func start_cs() *gsi.MockServer {
	// External JS UDF tests require an HTTP server since library operations are via REST calls
	return gsi.Start_cs_http(true)
}

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}
