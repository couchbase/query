//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package ansijoin

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on ANSI JOIN
func TestAnsiJoin(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into Buckets for ANSI JOIN \n\n")
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
	runStmt(qc, "CREATE INDEX st_ix11 on shellTest(c11, DISTINCT a11) WHERE type = \"left\"")
	runStmt(qc, "CREATE INDEX st_ix12 on shellTest(c11, a11) WHERE type = \"left\"")
	runStmt(qc, "CREATE INDEX st_ix21 on shellTest(c21, DISTINCT a21) WHERE type = \"right\"")
	runStmt(qc, "CREATE INDEX st_ix22 on shellTest(a22) WHERE type = \"right\"")
	runStmt(qc, "CREATE INDEX st_ix23 on shellTest(c22, c21) WHERE type = \"right\"")

	fmt.Println("Running ANSI JOIN test cases")

	// test ANSI JOIN on meta().id
	runMatch("case_ansijoin_metaid.json", false, false, qc, t)

	// test simple ANSI JOIN
	// this test case has variations of covering vs non-covering
	// index scans on both sides of the ANSI JOIN
	runMatch("case_ansijoin_simple.json", false, false, qc, t)

	// test named parameters and positional parameters
	runMatch("case_ansijoin_parameters.json", true, false, qc, t)

	// test ANSI JOIN with IN and OR clauses
	runMatch("case_ansijoin_or_in.json", false, false, qc, t)

	// test multiple ANSI JOINs
	runMatch("case_ansijoin_multi.json", false, false, qc, t)

	// test ANSI OUTER JOIN
	runMatch("case_ansijoin_outer.json", false, false, qc, t)

	// test ANSI JOIN with index hints
	runMatch("case_ansijoin_hints.json", false, false, qc, t)

	// test ANSI NEST on meta().id
	runMatch("case_ansinest_metaid.json", false, false, qc, t)

	// test ANSI NEST
	runMatch("case_ansinest_simple.json", false, false, qc, t)

	// test ANSI JOIN on arrays
	runMatch("case_ansijoin_array_simple.json", false, false, qc, t)

	// test ANSI JOIN on arrays -- more
	runMatch("case_ansijoin_array_more.json", false, false, qc, t)

	// test ANSI JOIN with ON-clause filters is pushed to left-hand-side
	runMatch("case_ansijoin_push_onclause.json", false, false, qc, t)

	// test ANSI JOIN with derived IS NOT NULL predicate
	runMatch("case_ansijoin_derive.json", false, false, qc, t)

	// test ANSI JOIN with UNNEST scan
	runMatch("case_ansijoin_unnest.json", false, false, qc, t)

	// test ANSI JOIN on expression term and subquery term
	runMatch("case_ansijoin_exprsubq.json", false, false, qc, t)

	// test ANSI OUTER JOIN to ANSI INNER JOIN transformation
	runMatch("case_ansijoin_oj2ij.json", false, true, qc, t)

	// test ANSI JOIN bug fixes
	runMatch("case_ansijoin_bugs.json", false, true, qc, t)

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
	runStmt(qc, "DROP INDEX shellTest.st_ix11")
	runStmt(qc, "DROP INDEX shellTest.st_ix12")
	runStmt(qc, "DROP INDEX shellTest.st_ix21")
	runStmt(qc, "DROP INDEX shellTest.st_ix22")
	runStmt(qc, "DROP INDEX shellTest.st_ix23")

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
