//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package indexcbo

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/migration"
	"github.com/couchbase/query/test/gsi"
)

const (
	_MB75410_BUCKET = "orders"
	_MB75410_KEEP   = "mb75410_keep"
	_MB75410_GONE   = "mb75410_gone"
	_MB75410_COLL   = "c"
)

// MB-75410: dropping a scope must remove the CBO statistics of its collections from the
// bucket's _system._query collection, and must not touch statistics of a collection with
// the same name in another scope
func TestDropScopeCBOStats(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\nTest is waiting for CBO migration to complete.\n")
	migration.Await("CBO_STATS")

	b := _MB75410_BUCKET
	keyspaces := []string{
		b + "." + _MB75410_KEEP + "." + _MB75410_COLL,
		b + "." + _MB75410_GONE + "." + _MB75410_COLL,
		b + "._default." + _MB75410_COLL,
	}

	mb75410Cleanup(qc)
	defer mb75410Cleanup(qc)

	mustRunAdmin(t, qc, "CREATE SCOPE "+b+"."+_MB75410_KEEP)
	mustRunAdmin(t, qc, "CREATE SCOPE "+b+"."+_MB75410_GONE)
	for _, ks := range keyspaces {
		// the new scope may take a moment to become visible to this query node
		retryRunAdmin(t, qc, "CREATE COLLECTION "+ks, 30, time.Second)
	}

	for _, ks := range keyspaces {
		// the new collection may take a moment to become visible to this query node
		retryRunAdmin(t, qc, "INSERT INTO "+ks+" (KEY k, VALUE v) SELECT \"k\" || TO_STRING(i) AS k, "+
			"{\"a\": i, \"b\": i % 5} AS v FROM ARRAY_RANGE(0, 50) AS i", 30, time.Second)
		mustRunAdmin(t, qc, "CREATE PRIMARY INDEX ON "+ks)
		mustRunAdmin(t, qc, "UPDATE STATISTICS FOR "+ks+"(a, b)")
	}

	// locate the CBO statistics documents of each collection via the main (keyspace) document,
	// i.e. the one without an "(index)"/"(histogram)" suffix; all documents for that collection
	// share its key as a prefix
	prefixes := make(map[string]string, len(keyspaces))
	before := make(map[string]int, len(keyspaces))
	for _, scope := range []string{_MB75410_KEEP, _MB75410_GONE, "_default"} {
		rr := mustRunAdmin(t, qc, fmt.Sprintf("SELECT RAW META().id FROM %s._system._query "+
			"WHERE META().id LIKE \"cbo::%%\" AND NOT CONTAINS(META().id, \"(\") "+
			"AND `scope` = \"%s\" AND `collection` = \"%s\"",
			b, scope, _MB75410_COLL))
		if len(rr.Results) != 1 {
			t.Fatalf("expected one CBO statistics keyspace document for %s.%s.%s, got %v",
				b, scope, _MB75410_COLL, rr.Results)
		}
		prefix, ok := rr.Results[0].(string)
		if !ok {
			t.Fatalf("unexpected document key %v for %s.%s", rr.Results[0], scope, _MB75410_COLL)
		}
		prefixes[scope] = prefix
		before[scope] = countCBOStatsDocs(t, qc, prefix)
		if before[scope] < 2 {
			t.Fatalf("expected keyspace and histogram documents for %s.%s.%s (prefix %s), got %d",
				b, scope, _MB75410_COLL, prefix, before[scope])
		}
	}

	mustRunAdmin(t, qc, "DROP SCOPE "+b+"."+_MB75410_GONE)

	// statistics are removed asynchronously once the manifest change is processed
	remaining := -1
	for i := 0; i < 60; i++ {
		remaining = countCBOStatsDocs(t, qc, prefixes[_MB75410_GONE])
		if remaining == 0 {
			break
		}
		time.Sleep(time.Second)
	}
	if remaining != 0 {
		t.Errorf("%d CBO statistics documents (prefix %s) remain after dropping scope %s.%s",
			remaining, prefixes[_MB75410_GONE], b, _MB75410_GONE)
	}

	// statistics for same-named collections in other scopes must be untouched
	for _, scope := range []string{_MB75410_KEEP, "_default"} {
		if n := countCBOStatsDocs(t, qc, prefixes[scope]); n != before[scope] {
			t.Errorf("CBO statistics documents for %s.%s.%s changed from %d to %d after dropping scope %s",
				b, scope, _MB75410_COLL, before[scope], n, _MB75410_GONE)
		}
	}

	// dropping a single collection removes its statistics (already worked before MB-75410)
	mustRunAdmin(t, qc, "DROP COLLECTION "+b+"."+_MB75410_KEEP+"."+_MB75410_COLL)
	remaining = -1
	for i := 0; i < 60; i++ {
		remaining = countCBOStatsDocs(t, qc, prefixes[_MB75410_KEEP])
		if remaining == 0 {
			break
		}
		time.Sleep(time.Second)
	}
	if remaining != 0 {
		t.Errorf("%d CBO statistics documents (prefix %s) remain after dropping collection %s.%s.%s",
			remaining, prefixes[_MB75410_KEEP], b, _MB75410_KEEP, _MB75410_COLL)
	}
	if n := countCBOStatsDocs(t, qc, prefixes["_default"]); n != before["_default"] {
		t.Errorf("CBO statistics documents for %s._default.%s changed from %d to %d after dropping collection %s.%s",
			b, _MB75410_COLL, before["_default"], n, _MB75410_KEEP, _MB75410_COLL)
	}
}

func countCBOStatsDocs(t *testing.T, qc *gsi.MockServer, prefix string) int {
	// match the keyspace document itself and its "(index)"/"(histogram)" documents, but not
	// another collection whose name merely starts with the same characters
	rr := mustRunAdmin(t, qc, fmt.Sprintf("SELECT RAW COUNT(*) FROM %s._system._query "+
		"WHERE META().id = \"%s\" OR META().id LIKE \"%s(%%\"", _MB75410_BUCKET, prefix, prefix))
	if len(rr.Results) != 1 {
		t.Fatalf("unexpected count result %v", rr.Results)
	}
	switch n := rr.Results[0].(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	}
	t.Fatalf("unexpected count result type %T", rr.Results[0])
	return 0
}

// scope and collection DDL needs more than the bucket_full_access of the default test users
// (which may silently do nothing), so this test runs all its statements as the admin user
func mb75410Cleanup(qc *gsi.MockServer) {
	b := _MB75410_BUCKET
	ks := b + "._default." + _MB75410_COLL
	gsi.RunAdminStmt(qc, "UPDATE STATISTICS FOR "+ks+" DELETE ALL")
	gsi.RunAdminStmt(qc, "DROP COLLECTION "+ks+" IF EXISTS")
	gsi.RunAdminStmt(qc, "DROP SCOPE "+b+"."+_MB75410_KEEP+" IF EXISTS")
	gsi.RunAdminStmt(qc, "DROP SCOPE "+b+"."+_MB75410_GONE+" IF EXISTS")
}
