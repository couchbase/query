//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package hashjoin

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on HASH JOIN
func TestHashJoin(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for HASH JOIN \n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Println("Creating indexes")
	runStmt(qc, "CREATE INDEX cust_lastName_firstName_customerId on customer(lastName, firstName, customerId)")
	runStmt(qc, "CREATE INDEX cust_customerId_lastName_firstName on customer(customerId, lastName, firstName)")
	runStmt(qc, "CREATE INDEX prod_productId on product(productId)")
	runStmt(qc, "CREATE INDEX purch_customerId_purchaseId on purchase(customerId, purchaseId)")
	runStmt(qc, "CREATE INDEX purch_purchaseId on purchase(purchaseId)")
	runStmt(qc, "CREATE INDEX purch_customerId_metaid on purchase(customerId || \"_\" || test_id)")
	runStmt(qc, "CREATE INDEX purch_arrProduct_customerId on purchase(DISTINCT ARRAY pd.product FOR pd IN lineItems END, customerId)")
	runStmt(qc, "CREATE INDEX ord_customerId_ordersId on orders(customerId, orderId)")
	runStmt(qc, "CREATE INDEX st_ix11 on shellTest(c11, DISTINCT a11) WHERE type = \"left\"")

	fmt.Println("Running HASH JOIN test cases")

	// test HASH JOIN on meta().id
	runStmt(qc, "CREATE PRIMARY INDEX ON customer")
	runMatch("case_hashjoin_metaid.json", false, true, qc, t)
	runStmt(qc, "DROP PRIMARY INDEX ON customer")

	// test simple HASH JOIN
	// this test case has variations of covering vs non-covering
	// index scans on both sides of the HASH JOIN
	runMatch("case_hashjoin_simple.json", false, true, qc, t)

	// named parameters and positional parameters
	runMatch("case_hashjoin_parameters.json", true, true, qc, t)

	// test multiple HASH JOINs
	runMatch("case_hashjoin_multi.json", false, true, qc, t)

	// test OUTER HASH JOIN
	runMatch("case_hashjoin_outer.json", false, true, qc, t)

	// test HASH JOIN with index hints
	runMatch("case_hashjoin_hints.json", false, true, qc, t)

	// test HASH NEST on meta().id
	runMatch("case_hashnest_metaid.json", false, true, qc, t)

	// test HASH NEST
	runMatch("case_hashnest_simple.json", false, true, qc, t)

	// test HASH JOIN with expression term and subquery term
	runMatch("case_hashjoin_exprsubq.json", false, true, qc, t)

	// test ANSI OUTER JOIN to ANSI INNER JOIN transformation
	runMatch("case_hashjoin_oj2ij.json", false, true, qc, t)

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX customer.cust_lastName_firstName_customerId")
	runStmt(qc, "DROP INDEX customer.cust_customerId_lastName_firstName")
	runStmt(qc, "DROP INDEX product.prod_productId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_purchaseId")
	runStmt(qc, "DROP INDEX purchase.purch_purchaseId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_metaid")
	runStmt(qc, "DROP INDEX purchase.purch_arrProduct_customerId")
	runStmt(qc, "DROP INDEX orders.ord_customerId_ordersId")
	runStmt(qc, "DROP INDEX shellTest.st_ix11")

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
