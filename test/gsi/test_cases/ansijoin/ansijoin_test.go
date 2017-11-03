//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package cover

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on ANSI JOIN
func TestCover(t *testing.T) {
	var RunTest bool

	val := os.Getenv("GSI_TEST")
	if strings.ToLower(val) == "true" {
		RunTest = true
	} else {
		RunTest = false
	}

	if RunTest {
		qc := start_cs()

		fmt.Println("\n\nInserting values into Buckets for ANSI JOIN \n\n")
		runMatch("insert.json", qc, t)

		fmt.Println("Creating indexes")
		runStmt(qc, "CREATE INDEX cust_lastName_firstName_customerId on customer(lastName, firstName, customerId)")
		runStmt(qc, "CREATE INDEX cust_customerId_lastName_firstName on customer(customerId, lastName, firstName)")
		runStmt(qc, "CREATE INDEX prod_productId on product(productId)")
		runStmt(qc, "CREATE INDEX purch_customerId_purchaseId on purchase(customerId, purchaseId)")
		runStmt(qc, "CREATE INDEX purch_purchaseId on purchase(purchaseId)")
		runStmt(qc, "CREATE INDEX purch_customerId_metaid on purchase(customerId || \"_\" || test_id)")
		runStmt(qc, "CREATE INDEX ord_customerId_ordersId on orders(customerId, orderId)")

		fmt.Println("Running ANSI JOIN test cases")

		// test ANSI JOIN on meta().id
		runMatch("case_ansijoin_metaid.json", qc, t)

		// test simple ANSI JOIN
		runMatch("case_ansijoin_simple.json", qc, t)

		// test ANSI JOIN with IN and OR clauses
		runMatch("case_ansijoin_or_in.json", qc, t)

		// test multiple ANSI JOINs
		runMatch("case_ansijoin_multi.json", qc, t)

		// test ANSI OUTER JOIN
		runMatch("case_ansijoin_outer.json", qc, t)

		// test ANSI JOIN on meta().id
		runMatch("case_ansinest_metaid.json", qc, t)

		// test ANSI NEST
		runMatch("case_ansinest_simple.json", qc, t)

		fmt.Println("Dropping indexes")
		runStmt(qc, "DROP INDEX customer.cust_lastName_firstName_customerId")
		runStmt(qc, "DROP INDEX customer.cust_customerId_lastName_firstName")
		runStmt(qc, "DROP INDEX product.prod_productId")
		runStmt(qc, "DROP INDEX purchase.purch_customerId_purchaseId")
		runStmt(qc, "DROP INDEX purchase.purch_purchaseId")
		runStmt(qc, "DROP INDEX purchase.purch_customerId_metaid")
		runStmt(qc, "DROP INDEX orders.ord_customerId_ordersId")

		// delete all rows from keyspaces used
		runStmt(qc, "DELETE FROM customer")
		runStmt(qc, "DELETE FROM product")
		runStmt(qc, "DELETE FROM purchase")
		runStmt(qc, "DELETE FROM orders")
	}
}
