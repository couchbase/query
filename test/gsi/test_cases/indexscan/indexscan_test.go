//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package indexscan

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
func TestIndexScan(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket for Array Indexing \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE INDEX reviewlistidx on product(ALL DISTINCT ARRAY r  FOR r IN reviewList END)")
	runStmt(qc, "CREATE INDEX reviewlistidx2 on product(ALL DISTINCT ARRAY r  FOR r IN reviewList END, productId)")
	runStmt(qc, "CREATE INDEX reviewlistidx3 on product(productId, ALL DISTINCT ARRAY r  FOR r IN reviewList END)")
	runStmt(qc, "CREATE INDEX plistidx on purchase (ALL DISTINCT ARRAY l for l in lineItems END)")

	runMatch("case_array_index_test1.json", false, false, qc, t)

	runStmt(qc, "DROP INDEX product.reviewlistidx")
	runStmt(qc, "DROP INDEX product.reviewlistidx2")
	runStmt(qc, "DROP INDEX product.reviewlistidx3")
	runStmt(qc, "DROP INDEX purchase.plistidx")

	runStmt(qc, "CREATE INDEX reviewlistidxall on product(ALL ARRAY r  FOR r IN reviewList END)")
	runStmt(qc, "CREATE INDEX reviewlistidx2all on product(ALL ARRAY r  FOR r IN reviewList END, productId)")
	runStmt(qc, "CREATE INDEX reviewlistidx3all on product(productId, ALL ARRAY r  FOR r IN reviewList END)")
	runStmt(qc, "CREATE INDEX plistidx on purchase (ALL DISTINCT ARRAY l for l in lineItems END)")

	runMatch("case_array_index_test2.json", false, false, qc, t)

	runStmt(qc, "DROP INDEX product.reviewlistidxall")
	runStmt(qc, "DROP INDEX product.reviewlistidx2all")
	runStmt(qc, "DROP INDEX product.reviewlistidx3all")
	runStmt(qc, "DROP INDEX purchase.plistidx")

	// Single-level Indexes
	runStmt(qc, "CREATE INDEX iv ON product( DISTINCT ARRAY v FOR v IN b END )")
	runStmt(qc, "CREATE INDEX ix ON product( DISTINCT ARRAY v.x FOR v IN b END )")
	runStmt(qc, "CREATE INDEX ixy ON product( DISTINCT ARRAY v.x + v.y FOR v IN b END )")

	//As covering indexes
	runStmt(qc, "CREATE INDEX cover_iv ON product( DISTINCT ARRAY v FOR v IN b END, b, type)")
	runStmt(qc, "CREATE INDEX cover_ix ON product( DISTINCT ARRAY v.x FOR v IN b END,b, type )")
	runStmt(qc, "CREATE INDEX cover_ixy ON product( DISTINCT ARRAY v.x + v.y FOR v IN b END,b, type )")

	//		runMatch("case_explain_test3.json", false, false,qc, t)

	//Multi-level index
	runStmt(qc, "DROP INDEX product.iv")
	runStmt(qc, "DROP INDEX product.ix")
	runStmt(qc, "DROP INDEX product.ixy")
	runStmt(qc, "DROP INDEX product.cover_iv")
	runStmt(qc, "DROP INDEX product.cover_ix")
	runStmt(qc, "DROP INDEX product.cover_ixy")

	runStmt(qc, "CREATE INDEX ivw ON product( DISTINCT ARRAY ( DISTINCT ARRAY w FOR w IN v END ) FOR v IN b END )")
	runStmt(qc, "CREATE INDEX cover_ivw ON product( DISTINCT ARRAY ( DISTINCT ARRAY w FOR w IN v END ) FOR v IN b END,b, type )")

	//		runMatch("case_explain_test4.json", false, false,qc, t)

	runStmt(qc, "DROP INDEX product.ivw")
	runStmt(qc, "DROP INDEX product.cover_ivw")

	// Create array index on TOKENS()
	runStmt(qc, "CREATE INDEX tokenindex ON product ((distinct (array lower(to_string(d)) for d in tokens(description) end)))")
	runStmt(qc, "CREATE INDEX tokenindex1 on product(ALL ARRAY r  FOR r IN tokens(name) END)")
	runStmt(qc, "CREATE INDEX tokenindex2 on product (ALL DISTINCT ARRAY l for l in tokens(imageURL) END)")

	runMatch("case_array_index_tokens.json", false, false, qc, t)

	runStmt(qc, "DROP INDEX product.tokenindex")
	runStmt(qc, "DROP INDEX product.tokenindex1")
	runStmt(qc, "DROP INDEX product.tokenindex2")

	// Create array indexes for unnest scan
	runStmt(qc, "CREATE INDEX iax1 ON orders(ALL ARRAY v1 FOR v1 IN a1 END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax2 ON orders(ALL ARRAY (ALL ARRAY v2 FOR v2 IN v1 END) FOR v1 IN a2 END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax3 ON orders(ALL ARRAY v1.id FOR v1 IN a3 WHEN v1.type = \"n\" END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax4 ON orders(ALL ARRAY (ALL ARRAY v2.id FOR v2 IN v1.aa END)  FOR v1 IN a4 END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax5 ON orders(ALL ARRAY (ALL ARRAY [v2.id,v1, c1] FOR v2 IN v1.aa END)  FOR v1 IN a4 END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax6 ON orders(ALL ARRAY v1.val FOR v1 IN a3 WHEN v1.type = \"n\" END,c1,c2) WHERE test_id = \"ua\"")

	runMatch("case_array_index_unnest_scan.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX orders.iax1")
	runStmt(qc, "DROP INDEX orders.iax2")
	runStmt(qc, "DROP INDEX orders.iax3")
	runStmt(qc, "DROP INDEX orders.iax4")
	runStmt(qc, "DROP INDEX orders.iax5")
	runStmt(qc, "DROP INDEX orders.iax6")

	// Create skip ranges
	runStmt(qc, "CREATE INDEX is01 ON orders (c0,c1,c2,DISTINCT a1,c3,c4) WHERE test_id = \"skipranges\"")
	runStmt(qc, "CREATE INDEX is02 ON orders (ALL ARRAY v.id FOR v IN a3 END, c0,c1,c2,c3,c4) WHERE test_id = \"skipranges\"")
	runStmt(qc, "CREATE INDEX is11 ON orders (c0) WHERE test_id = \"skipranges\"")
	runStmt(qc, "CREATE INDEX is12 ON orders (c0,c4) WHERE test_id = \"skipranges\"")
	runStmt(qc, "CREATE INDEX is13 ON orders (c0,c1,c4) WHERE test_id = \"skipranges\"")
	runStmt(qc, "CREATE INDEX is14 ON orders (c0,c1,c2,c4) WHERE test_id = \"skipranges\"")
	runStmt(qc, "CREATE INDEX is15 ON orders (c10,c11) WHERE test_id = \"skipranges\" AND c14 = 1000")
	runStmt(qc, "CREATE INDEX is16 ON orders (c10,c11,c14) WHERE test_id = \"skipranges\"")

	runMatch("case_skipranges.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX orders.is01")
	runStmt(qc, "DROP INDEX orders.is02")
	runStmt(qc, "DROP INDEX orders.is11")
	runStmt(qc, "DROP INDEX orders.is12")
	runStmt(qc, "DROP INDEX orders.is13")
	runStmt(qc, "DROP INDEX orders.is14")
	runStmt(qc, "DROP INDEX orders.is15")
	runStmt(qc, "DROP INDEX orders.is16")

	// order nulls ASC index
	runStmt(qc, "CREATE INDEX noix1 ON orders (c1, c2, c3, c4) WHERE test_id = \"ordernulls\"")
	runMatch("case_ordernulls.json", false, true, qc, t)
	runMatch("case_ordernulls.json", true, true, qc, t)
	runMatch("case_let.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX orders.noix1")

	// order nulls DESC index
	runStmt(qc, "CREATE INDEX noix1 ON orders (c1, c2 DESC, c3, c4) WHERE test_id = \"ordernulls\"")
	runMatch("case_ordernullsdesc.json", false, true, qc, t)
	runMatch("case_ordernullsdesc.json", true, true, qc, t)
	runStmt(qc, "DROP INDEX orders.noix1")

	// query named and positional parameters
	runStmt(qc, "CREATE INDEX poix1 ON orders (c1, c2, c3, c4) WHERE test_id = \"parameters\"")
	runStmt(qc, "CREATE INDEX poix2 ON orders (DISTINCT ARRAY v.id FOR v IN a3 END) WHERE test_id = \"parameters\"")
	runMatch("case_parameters.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX orders.poix1")
	runStmt(qc, "DROP INDEX orders.poix2")

	runStmt(qc, "create primary index on product ")
	runStmt(qc, "create primary index on purchase")
	runStmt(qc, "create primary index on orders")

	_, _, errcs := runStmt(qc, "delete from product where test_id IN [\"arrayIndex\", \"coveredIndex\"]")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}
	_, _, errcs = runStmt(qc, "delete from purchase where test_id = \"arrayIndex\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}
	_, _, errcs = runStmt(qc, "delete from orders where test_id IN [\"ua\", \"skipranges\", \"ordernulls\", \"parameters\"]")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "drop primary index on product")
	runStmt(qc, "drop primary index on purchase")
	runStmt(qc, "drop primary index on orders")
}
