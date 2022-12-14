//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package lkmissing

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/couchbase/query/test/gsi"
)

func TestLkMissing(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	collname := "orders._default.lkm"
	ixnames := []string{
		"",
		"",
		"",
		"",
		"mix1",
		"mix2",
		"ix_city_state",
		"",
	}
	indexes := []string{
		"CREATE PRIMARY INDEX %s ON %s;",
		"CREATE INDEX %s ON %s(team, fname DESC, lname);",
		"CREATE INDEX %s ON %s(ALL ARRAY FLATTEN_KEYS(c.id INCLUDE MISSING, c.type, c.default) FOR c IN contacts END, fname, lname, team)",
		"CREATE INDEX %s ON %s(ALL ARRAY FLATTEN_KEYS(c.id, c.type, c.default) FOR c IN contacts END, team)",
		"CREATE INDEX %s ON %s(fname INCLUDE MISSING DESC, lname, team)",
		"CREATE INDEX %s ON %s(fname INCLUDE MISSING DESC, lname, team, DISTINCT ARRAY FLATTEN_KEYS(v.id, v.type, v.default) FOR v IN contacts END)",
		"CREATE INDEX %s ON %s(city, state)",
		"CREATE INDEX %s ON %s(fname INCLUDE MISSING, lname, team) WHERE type = 'contacts'",
	}

	qc := start_cs()

	// Insert the test specific data
	runMatch("insert.json", false, false, qc, t) // non-prepared, no-explain

	for pos, name := range ixnames {
		run_drop_index(name, collname, qc, t)
		run_create_index(name, collname, indexes[pos], qc, t)
	}

	run_test("case_primary.json", collname, "ix_primary", indexes[0], qc, t)
	run_test("case_nonmissing.json", collname, "ix_team_fname", indexes[1], qc, t)
	run_test("case_missing.json", collname, "", "", qc, t)
	run_test("case_any.json", collname, "", "", qc, t)
	run_test("case_unnest.json", collname, "", "", qc, t)

	run_drop_index("maix1", collname, qc, t)
	run_create_index("maix1", collname, indexes[2], qc, t)
	run_test("case_unnest_nonmissing.json", collname, "aix2", indexes[3], qc, t)

	run_test("case_unnest_missing.json", collname, "", "", qc, t)
	run_drop_index("maix1", collname, qc, t)

	run_drop_index("pix1", collname, qc, t)
	run_create_index("pix1", collname, indexes[7], qc, t)
	run_test("case_bugs.json", collname, "", "", qc, t)
	run_drop_index("pix1", collname, qc, t)

	runStmt(qc, fmt.Sprintf("DELETE FROM %s;", collname)) // Delete the test specific data

	for _, name := range ixnames {
		run_drop_index(name, collname, qc, t)
	}
}

func run_test(testcase, coll, ixname, createidx string, qc *gsi.MockServer, t *testing.T) {
	run_drop_index(ixname, coll, qc, t)
	run_create_index(ixname, coll, createidx, qc, t)
	if testcase != "" {
		runMatch(testcase, false, true, qc, t) // non-prepared, explain
		runMatch(testcase, true, false, qc, t) // prepared, no-explain
		run_drop_index(ixname, coll, qc, t)
	}
}

func run_create_index(ixname, coll, createidx string, qc *gsi.MockServer, t *testing.T) {
	if ixname != "" {
		runStmt(qc, fmt.Sprintf(createidx, ixname, coll))
	}
}

func run_drop_index(ixname, coll string, qc *gsi.MockServer, t *testing.T) {
	if ixname != "" {
		runStmt(qc, fmt.Sprintf("DROP INDEX %s ON %s", ixname, coll))
	}
}

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
