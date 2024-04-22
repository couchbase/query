//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package flattenkeys

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

func TestFlattenkeys(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	indexes := []string{
		"CREATE INDEX %s ON %s(isbn, author DESC, DISTINCT ARRAY FLATTEN_KEYS(ch.num DESC, ch.name DESC, ch.description) FOR ch IN chapters END, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(isbn, author DESC, DISTINCT ARRAY (DISTINCT ARRAY FLATTEN_KEYS(pg.num DESC, pg.name DESC, ch.description) FOR pg IN ch.pages END) FOR ch IN chapters END, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(isbn, author DESC, DISTINCT ARRAY FLATTEN_KEYS(ch.name DESC, ch.description) FOR ch IN chapters WHEN ch.num = 1 END, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(isbn, author DESC, DISTINCT ARRAY (DISTINCT ARRAY FLATTEN_KEYS(pg.name DESC, ch.description) FOR pg IN ch.pages WHEN pg.num = 1 END) FOR ch IN chapters WHEN ch.num = 1 END, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(isbn, author DESC, DISTINCT ARRAY FLATTEN_KEYS(ch.num DESC, ch.name DESC, ch.description) FOR ch IN chapters END, year, name, chapters) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(isbn, author DESC, DISTINCT ARRAY (DISTINCT ARRAY FLATTEN_KEYS(pg.num DESC, pg.name DESC, ch.description) FOR pg IN ch.pages END) FOR ch IN chapters END, year, name, chapters) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(ALL ARRAY FLATTEN_KEYS(ch.num DESC, ch.name DESC, ch.description) FOR ch IN chapters END, isbn, author DESC, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(ALL ARRAY (ALL ARRAY FLATTEN_KEYS(pg.num DESC, pg.name DESC, ch.description) FOR pg IN ch.pages END) FOR ch IN chapters END, isbn, author DESC, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(ALL ARRAY FLATTEN_KEYS(ch.name DESC, ch.description) FOR ch IN chapters WHEN ch.num = 1 END, isbn, author DESC, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(ALL ARRAY (ALL ARRAY FLATTEN_KEYS(pg.name DESC, ch.description) FOR pg IN ch.pages WHEN pg.num = 1 END) FOR ch IN chapters WHEN ch.num = 1 END, isbn, author DESC, year, name) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(ALL ARRAY FLATTEN_KEYS(ch.num DESC, ch.name DESC, ch.description) FOR ch IN chapters END, isbn, author DESC, year, name, chapters) WHERE type = 'book'",
		"CREATE INDEX %s ON %s(ALL ARRAY (ALL ARRAY FLATTEN_KEYS(pg.num DESC, pg.name DESC, ch.description) FOR pg IN ch.pages END) FOR ch IN chapters END, isbn, author DESC, year, name, chapters) WHERE type = 'book'",
	}
	qc := start_cs()

	// Insert the test specific data
	runMatch("insert.json", false, false, qc, t) // non-prepared, no-explain

	pos := 0
	run_test("case_any.json", "ixf10", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_any_nested.json", "ixf10n", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_any_when.json", "ixf10w", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_any_nested_when.json", "ixf10wn", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_any_explicit.json", "ixf10e", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_any_nested_explicit.json", "ixf10en", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_unnest.json", "ixf10u", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_unnest_nested.json", "ixf10un", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_unnest_when.json", "ixf10uw", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_unnest_nested_when.json", "ixf10uwn", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_unnest_explicit.json", "ixf10ue", "orders._default.flattenkeys", indexes[pos], qc, t)
	pos++
	run_test("case_unnest_nested_explicit.json", "ixf10uen", "orders._default.flattenkeys", indexes[pos], qc, t)

	runStmt(qc, "CREATE INDEX ix1 ON shellTest(c1, DISTINCT ARRAY FLATTEN_KEYS(v1.type,v1.phone) FOR v1 IN contacts END, c2)")
	runStmt(qc, "CREATE INDEX ix2 ON shellTest(c11, DISTINCT ARRAY(DISTINCT ARRAY FLATTEN_KEYS(v1.type,v1.phone) FOR v1 IN v.contacts END) FOR v IN infos END, c12)")
	runMatch("case_bugs.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX shellTest.ix1")
	runStmt(qc, "DROP INDEX shellTest.ix2")

	case_clean(qc, t) // Delete the test specific data
}

func run_test(testcase, ixname, coll, createidx string, qc *gsi.MockServer, t *testing.T) {
	runStmt(qc, fmt.Sprintf("DROP INDEX %s ON %s", ixname, coll))
	runStmt(qc, fmt.Sprintf(createidx, ixname, coll))
	runMatch(testcase, false, true, qc, t) // non-prepared, explain
	runMatch(testcase, true, false, qc, t) // prepared, no-explain
	runStmt(qc, fmt.Sprintf("DROP INDEX %s ON %s", ixname, coll))
}

func case_clean(qc *gsi.MockServer, t *testing.T) {
	runStmt(qc, "CREATE PRIMARY INDEX ON orders._default.flattenkeys")
	runStmt(qc, "DELETE FROM orders._default.flattenkeys")
	runStmt(qc, "DROP PRIMARY INDEX ON orders._default.flattenkeys")
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
