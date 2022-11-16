//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package aggregate_functions

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestAggregateFunctions(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON product")
	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	runStmt(qc, "CREATE PRIMARY INDEX ON customer")

	runMatch("case_distinct.json", false, false, qc, t)
	runMatch("case_group_by_having.json", false, false, qc, t)
	runMatch("case_median_stddev_variance.json", false, false, qc, t)

	runStmt(qc, "CREATE INDEX index_custId on orders(custId) WHERE test_id = \"agg_func\"")
	runMatch("case_group_by_group_as.json", false, true, qc, t)

	runStmt(qc, "delete from product where test_id IN [\"agg_func\"]")
	runStmt(qc, "delete from orders where test_id IN [\"agg_func\",\"median_agg_func\",\"cntn_agg_func\"]")
	runStmt(qc, "delete from customer where test_id IN [\"agg_func\"]")

	runStmt(qc, "DROP INDEX orders.index_custId")
	runStmt(qc, "DROP PRIMARY INDEX ON product")
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
	runStmt(qc, "DROP PRIMARY INDEX ON customer")
}
