//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ttl

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/test/gsi"
)

// MB-75435: with auto-prepare on, later executions of a statement reuse the cached auto-prepared plan, in
// and out of a transaction. That plan had no statement type, which failed the statement in a transaction
// (17002), and a transaction plan of an auto-prepared statement panicked in reprepare().
func TestAutoPrepareTransactions(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()
	time.Sleep(5 * time.Second)
	qc.SetAutoPrepare(true)
	defer qc.SetAutoPrepare(false)

	const sel = "SELECT META(d).id, d.a FROM orders._default.transactions AS d WHERE d.a > 7500 ORDER BY d.a"
	const del = "DELETE FROM orders._default.transactions USE KEYS 'mb75435_none'"

	// keys differ from ix1 (a) of TestTransactions, so a left-over index can't change its plans
	mustRun(qc, t, nil, "CREATE INDEX ix_mb75435 ON orders._default.transactions(a, META().id)")
	defer runStmt(qc, "DROP INDEX ix_mb75435 ON orders._default.transactions")
	mustRun(qc, t, nil, "UPSERT INTO orders._default.transactions VALUES "+
		"('mb75435_1', {'a': 7501}), ('mb75435_2', {'a': 7502})")
	defer runStmt(qc, "DELETE FROM orders._default.transactions USE KEYS ['mb75435_1', 'mb75435_2']")

	expected := `[{"a":7501,"id":"mb75435_1"},{"a":7502,"id":"mb75435_2"}]`

	// the first execution plans the statement and caches the plan, the others use the cached plan
	// (gsi.Run uses request_plus scan consistency, gsi.Consistency_parameter)
	for i := 0; i < 2; i++ {
		checkResults(t, mustRun(qc, t, nil, sel), expected, "outside a transaction")
	}

	// implicit transaction
	tximplicit := map[string]interface{}{"tximplicit": true, "durability_level": "none"}
	for i := 0; i < 2; i++ {
		checkResults(t, mustRun(qc, t, tximplicit, sel), expected, "tximplicit SELECT")
		mustRun(qc, t, tximplicit, del)
	}

	// explicit transaction: after the mutation the SELECT needs a transaction plan (delta keyspace)
	txgroup := map[string]interface{}{"txgroup": float64(1), "durability_level": "none"}
	mustRun(qc, t, txgroup, "START WORK")
	open := true
	defer func() {
		if open {
			gsi.Run(qc, txgroup, "ROLLBACK", gsi.Namespace_CBS, nil, nil, nil)
		}
	}()
	checkResults(t, mustRun(qc, t, txgroup, sel), expected, "transaction, before the mutation")
	mustRun(qc, t, txgroup, "UPDATE orders._default.transactions USE KEYS 'mb75435_1' SET a = 7509")
	for i := 0; i < 2; i++ {
		checkResults(t, mustRun(qc, t, txgroup, sel),
			`[{"a":7502,"id":"mb75435_2"},{"a":7509,"id":"mb75435_1"}]`, "transaction, after the mutation")
		mustRun(qc, t, txgroup, del)
	}
	mustRun(qc, t, txgroup, "ROLLBACK")
	open = false
}

func mustRun(qc *gsi.MockServer, t *testing.T, queryParams map[string]interface{}, q string) *gsi.RunResult {
	if queryParams == nil {
		queryParams = map[string]interface{}{}
	}
	rr := gsi.Run(qc, queryParams, q, gsi.Namespace_CBS, nil, nil, nil)
	if rr.Err != nil {
		t.Fatalf("%s (%v): %v", q, queryParams, rr.Err)
	}
	return rr
}

func checkResults(t *testing.T, rr *gsi.RunResult, expected, what string) {
	b, _ := json.Marshal(rr.Results)
	if string(b) != expected {
		t.Errorf("%s: results %s, expected %s", what, string(b), expected)
	}
}
