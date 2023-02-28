//  Copyright 2023-Present Couchbase, Inc.
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
)

/*
Basic test to ensure connections to both
Datastore and Couchbase server, work.
*/
func TestIndexCBO(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket for Indexing with CBO \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE INDEX p_productId ON product(productId) WHERE test_id = \"indexCBO\"")
	runStmt(qc, "CREATE INDEX p_productId_reviews ON product(productId, DISTINCT reviewList) WHERE test_id = \"indexCBO\"")
	runStmt(qc, "CREATE INDEX iorix1 ON orders (c1, c2, c3) WHERE test_id = \"indexCBO\"")
	runStmt(qc, "CREATE INDEX iorix2 ON orders (c1, c2, c4, c6) WHERE test_id = \"indexCBO\"")

	// run UPDATE STATISTICS statements
	runStmt(qc, "UPDATE STATISTICS FOR product INDEX(p_productId_reviews)")
	runStmt(qc, "UPDATE STATISTICS FOR orders INDEX(iorix1, iorix2)")

	runMatch("case_intersect_scan.json", false, true, qc, t)

	runMatch("case_union_scan.json", false, true, qc, t)

	// DELETE optimizer statistics
	runStmt(qc, "UPDATE STATISTICS FOR product DELETE ALL")
	runStmt(qc, "UPDATE STATISTICS FOR orders DELETE ALL")

	runStmt(qc, "DROP INDEX product.p_productId")
	runStmt(qc, "DROP INDEX product.p_productId_reviews")
	runStmt(qc, "DROP INDEX purchase.iorix1")
	runStmt(qc, "DROP INDEX purchase.iorix2")

	runStmt(qc, "create primary index on product ")
	runStmt(qc, "create primary index on purchase")
	runStmt(qc, "create primary index on orders")

	rr := runStmt(qc, "delete from product where test_id IN [\"indexCBO\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "delete from purchase where test_id = \"indexCBO\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "delete from orders where test_id = \"indexCBO\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "drop primary index on product")
	runStmt(qc, "drop primary index on purchase")
	runStmt(qc, "drop primary index on orders")
}
