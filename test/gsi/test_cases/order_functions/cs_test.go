//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package order_functions

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestOrderFunctions(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	runStmt(qc, "CREATE INDEX order_cust ON orders(custId)")
	runStmt(qc, "CREATE INDEX order_cust_shipped ON orders(custId,`shipped-on`,orderId,test_id)")

	runMatch("case_first.json", false, false, qc, t)
	runMatch("case_orderby_limit.json", false, true, qc, t)
	runMatch("case_orderby.json", false, false, qc, t)

	rr := runStmt(qc, "delete from orders where test_id IN [\"order_func\",\"order_limit_prune_sort\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON order")
	runStmt(qc, "DROP INDEX orders.order_cust")
	runStmt(qc, "DROP INDEX orders.order_cust_shipped")
}
