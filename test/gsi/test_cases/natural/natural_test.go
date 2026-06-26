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

	nl "github.com/couchbase/query/natural"
)

func TestNatural(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	// Capella-credential pre-flight. When natural_cred/natural_orgid are
	// supplied (Capella iQ path), validate them up front against the sessions/
	// providers API so a credential problem fails immediately with a clear
	// message, instead of surfacing as a confusing 19202 deep inside
	// test_vendor.json after the keyspace setup has run.
	if cred := os.Getenv("natural_cred"); cred != "" {
		orgid := os.Getenv("natural_orgid")
		if _, err := nl.GetCapellaModelProviders(cred, orgid, false); err != nil {
			t.Fatalf("Capella credential pre-flight FAILED — natural_cred/natural_orgid "+
				"were rejected by iQ (check user:pass format, quoting, and prod-vs-dev "+
				"endpoint): %v", err)
		}
		t.Logf("Capella credential pre-flight OK")
	}

	qc := start_cs()
	runMatch("insert.json", false, false, qc, t)
	runMatch("testcases.json", false, false, qc, t)

	// Capella-path vendor/model validation tests
	runMatch("test_vendor.json", false, false, qc, t)

	runMatch("test_system_natural_chats.json", false, false, qc, t)

	// Capella-path end-to-end generation (SQL, jsudf, model refusal). These
	// carry no natural_config, so they route through the Capella iQ path and
	// require live credentials (natural_cred + natural_orgid, set by
	// runtest.sh). Without those exported they will fail, by design.
	runMatch("testcases_capella.json", false, false, qc, t)

	// natural_config validation / provider resolution (deterministic, no LLM)
	runMatch("test_config.json", false, false, qc, t)

	// End-to-end gateway round-trip against an in-process fake LLM.
	testNaturalFakeProvider(qc, t)

	runStmt(qc, "DELETE FROM orders")
}
