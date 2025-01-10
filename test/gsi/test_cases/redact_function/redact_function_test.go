//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package redact_function

import (
	"os"
	"strings"
	"testing"
)

func TestRedactFunction(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runMatch("insert.json", false, false, qc, t)
	runMatch("case_redact_function.json", false, false, qc, t)
	runStmt(qc, "DELETE FROM orders")
}
