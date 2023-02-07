//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package hints

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on optimizer hints
func TestHints(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for hints\n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX cust_lastName_firstName_customerId on customer(lastName, firstName, customerId)")
	runStmt(qc, "CREATE INDEX cust_customerId_lastName_firstName on customer(customerId, lastName, firstName)")
	runStmt(qc, "CREATE INDEX prod_productId on product(productId)")
	runStmt(qc, "CREATE INDEX purch_customerId_purchaseId on purchase(customerId, purchaseId)")
	runStmt(qc, "CREATE INDEX purch_purchaseId on purchase(purchaseId)")
	runStmt(qc, "CREATE INDEX purch_customerId_metaid on purchase(customerId || \"_\" || test_id)")
	runStmt(qc, "CREATE INDEX ord_customerId_ordersId on orders(customerId, orderId)")
	runStmt(qc, "CREATE INDEX purch_arrProduct_customerId on purchase(DISTINCT ARRAY pd.product FOR pd IN lineItems END, customerId)")
	runStmt(qc, "CREATE INDEX prod_reviewList_productId on product(DISTINCT reviewList, productId)")
	runStmt(qc, "CREATE INDEX st_source_idx on shellTest(c11, c12)")
	runStmt(qc, "CREATE INDEX st_target_idx on shellTest(c21, c22)")

	fmt.Println("Running Hints test cases")

	// simple hints
	runMatch("case_hints_simple.json", false, true, qc, t)

	// hints with errors
	runMatch("case_hints_errors.json", false, true, qc, t)

	// hints in DML statements
	runMatch("case_hints_dml.json", false, true, qc, t)

	// negative hints
	runMatch("case_hints_negative.json", false, true, qc, t)
	runMatch("case_hints_avoid.json", false, true, qc, t)

	// INDEX_ALL hint
	runMatch("case_hints_index_all.json", false, true, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX customer.cust_lastName_firstName_customerId")
	runStmt(qc, "DROP INDEX customer.cust_customerId_lastName_firstName")
	runStmt(qc, "DROP INDEX product.prod_productId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_purchaseId")
	runStmt(qc, "DROP INDEX purchase.purch_purchaseId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_metaid")
	runStmt(qc, "DROP INDEX orders.ord_customerId_ordersId")
	runStmt(qc, "DROP INDEX purchase.purch_arrProduct_customerId")
	runStmt(qc, "DROP INDEX product.prod_reviewList_productId")
	runStmt(qc, "DROP INDEX shellTest.st_source_idx")
	runStmt(qc, "DROP INDEX shellTest.st_target_idx")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON customer")
	runStmt(qc, "CREATE PRIMARY INDEX ON product")
	runStmt(qc, "CREATE PRIMARY INDEX ON purchase")
	runStmt(qc, "CREATE PRIMARY INDEX ON orders")
	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM customer")
	runStmt(qc, "DELETE FROM product")
	runStmt(qc, "DELETE FROM purchase")
	runStmt(qc, "DELETE FROM orders")
	runStmt(qc, "DELETE FROM shellTest")

	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON customer")
	runStmt(qc, "DROP PRIMARY INDEX ON product")
	runStmt(qc, "DROP PRIMARY INDEX ON purchase")
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
