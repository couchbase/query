//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package knowledge

import (
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/test/gsi"
)

// _LONG_SCOPE and _LONG_COLL sit at Couchbase's 251-character scope/collection name limit - used
// to exercise getStorageKey's capKeySegment (which hashes a name segment down once it exceeds a
// UUID's length, to keep the resulting document key well clear of the 251-byte KV key limit).
var _LONG_SCOPE = strings.Repeat("a", 251)
var _LONG_COLL = strings.Repeat("b", 251)

// The test cases keyspace: orders.knowledge_test.{articles,flights}. Own scope + collections so
// the suite doesn't leave CREATE KNOWLEDGE entries lying around in shared test infrastructure
// keyspaces, and can be re-run without hitting stale "already exists" errors from a prior run.
func setup(qc *gsi.MockServer, t *testing.T) {
	runAdminStmt(qc, "DROP SCOPE orders.knowledge_test IF EXISTS")
	if rr := runAdminStmt(qc, "CREATE SCOPE orders.knowledge_test"); rr.Err != nil {
		t.Fatalf("failed to create scope orders.knowledge_test: %v", rr.Err)
	}
	if rr := runAdminStmt(qc, "CREATE COLLECTION orders.knowledge_test.articles"); rr.Err != nil {
		t.Fatalf("failed to create collection orders.knowledge_test.articles: %v", rr.Err)
	}
	if rr := runAdminStmt(qc, "CREATE COLLECTION orders.knowledge_test.flights"); rr.Err != nil {
		t.Fatalf("failed to create collection orders.knowledge_test.flights: %v", rr.Err)
	}

	runAdminStmt(qc, "DROP SCOPE orders.`"+_LONG_SCOPE+"` IF EXISTS")
	if rr := runAdminStmt(qc, "CREATE SCOPE orders.`"+_LONG_SCOPE+"`"); rr.Err != nil {
		t.Fatalf("failed to create long-named scope: %v", rr.Err)
	}
	if rr := runAdminStmt(qc, "CREATE COLLECTION orders.`"+_LONG_SCOPE+"`.`"+_LONG_COLL+"`"); rr.Err != nil {
		t.Fatalf("failed to create long-named collection: %v", rr.Err)
	}
}

func teardown(qc *gsi.MockServer) {
	runAdminStmt(qc, "DROP SCOPE orders.knowledge_test IF EXISTS")
	runAdminStmt(qc, "DROP SCOPE orders.`"+_LONG_SCOPE+"` IF EXISTS")
	runAdminStmt(qc, "DELETE FROM orders._system._query WHERE meta().id LIKE \"know::%\"")
}

func TestKnowledge(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	setup(qc, t)
	defer teardown(qc)

	runMatch("case_knowledge.json", false, false, qc, t)
}
