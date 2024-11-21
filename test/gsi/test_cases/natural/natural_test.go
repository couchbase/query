/*
Copyright 2024-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package natural

import (
	"os"
	"strings"
	"testing"
)

func TestNatural(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()
	runMatch("insert.json", false, false, qc, t)
	runMatch("testcases.json", false, false, qc, t)

	runStmt(qc, "DELETE FROM orders")
}
