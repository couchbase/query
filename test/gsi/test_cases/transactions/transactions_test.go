//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ttl

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

/*
Basic test to ensure connections to both
Datastore and Couchbase server, work.
*/
func TestTransactions(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	time.Sleep(5 * time.Second)
	runStmt(qc, "CREATE INDEX ix1 ON orders._default.transactions (a);")
	runMatch("case_tx.json", false, false, qc, t) // non-prepared, no-explain
	time.Sleep(1 * time.Second)
	runMatch("case_tx.json", true, false, qc, t) // prepared, no-explain
	runStmt(qc, "DROP INDEX ix1 ON orders._default.transactions")
	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	runStmt(qc, "DELETE FROM orders AS d WHERE IS_BINARY(d)")
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error, int) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
