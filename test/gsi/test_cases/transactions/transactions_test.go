//  Copyright (c) 2021 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
