//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	runStmt(qc, "CREATE INDEX ord_customerId_ordersId on orders(customerId, orderId)")

	fmt.Println("Running HASH JOIN test cases")

	// test HASH JOIN on meta().id
	runStmt(qc, "CREATE PRIMARY INDEX ON customer")
	runMatch("case_hashjoin_metaid.json", false, true, qc, t)
	runStmt(qc, "DROP PRIMARY INDEX ON customer")

	// test simple HASH JOIN
	// this test case has variations of covering vs non-covering
	// index scans on both sides of the HASH JOIN
	runMatch("case_hashjoin_simple.json", false, true, qc, t)

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

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX customer.cust_lastName_firstName_customerId")
	runStmt(qc, "DROP INDEX customer.cust_customerId_lastName_firstName")
	runStmt(qc, "DROP INDEX product.prod_productId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_purchaseId")
	runStmt(qc, "DROP INDEX purchase.purch_purchaseId")
	runStmt(qc, "DROP INDEX purchase.purch_customerId_metaid")
	runStmt(qc, "DROP INDEX orders.ord_customerId_ordersId")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON customer")
	runStmt(qc, "CREATE PRIMARY INDEX ON product")
	runStmt(qc, "CREATE PRIMARY INDEX ON purchase")
	runStmt(qc, "CREATE PRIMARY INDEX ON orders")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM customer")
	runStmt(qc, "DELETE FROM product")
	runStmt(qc, "DELETE FROM purchase")
	runStmt(qc, "DELETE FROM orders")

	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON customer")
	runStmt(qc, "DROP PRIMARY INDEX ON product")
	runStmt(qc, "DROP PRIMARY INDEX ON purchase")
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}
