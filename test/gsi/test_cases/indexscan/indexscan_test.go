//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	runStmt(qc, "CREATE INDEX iax1 ON orders(ALL a1,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax2 ON orders(ALL ARRAY (ALL v1) FOR v1 IN a2 END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax3 ON orders(ALL ARRAY v1.id FOR v1 IN a3 WHEN v1.type = \"n\" END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax4 ON orders(ALL ARRAY (ALL ARRAY v2.id FOR v2 IN v1.aa END)  FOR v1 IN a4 END,c1,c2) "+
		"WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax5 ON orders(ALL ARRAY (ALL ARRAY [v2.id,v1, c1] FOR v2 IN v1.aa END)  FOR v1 IN a4 END,c1,c2) "+
		"WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax6 ON orders(ALL ARRAY v1.val FOR v1 IN a3 WHEN v1.type = \"n\" END,c1,c2) WHERE test_id = \"ua\"")
	runStmt(qc, "CREATE INDEX iax7 ON orders(ALL ARRAY v1.val FOR v1 IN a3 WHEN v1.type = \"n\" END DESC,c1,c2) WHERE test_id = \"ua\"")

	runMatch("case_array_index_unnest_scan.json", false, true, qc, t)
	runMatch("case_array_index_unnest_scan2.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX orders.iax1")
	runStmt(qc, "DROP INDEX orders.iax2")
	runStmt(qc, "DROP INDEX orders.iax3")
	runStmt(qc, "DROP INDEX orders.iax4")
	runStmt(qc, "DROP INDEX orders.iax5")
	runStmt(qc, "DROP INDEX orders.iax6")
	runStmt(qc, "DROP INDEX orders.iax7")

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
	runStmt(qc, "CREATE INDEX poix3 ON orders (DISTINCT ARRAY v.id FOR v IN a4 WHEN v.name = \"abc\" END) "+
		"WHERE test_id = \"parameters\"")
	runStmt(qc, "CREATE INDEX poix4 ON orders (c1, c2, c3, c4) WHERE test_id LIKE \"parameter%\"")
	runMatch("case_parameters.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX orders.poix1")
	runStmt(qc, "DROP INDEX orders.poix2")
	runStmt(qc, "DROP INDEX orders.poix3")
	runStmt(qc, "DROP INDEX orders.poix4")

	runStmt(qc, "CREATE INDEX ifloix1 ON orders (c1, c2, c3, c4, c5) WHERE test_id = \"idxfltr\"")
	runStmt(qc, "CREATE INDEX ifloix2 ON orders (c6, a1) WHERE test_id = \"idxfltr\"")
	runMatch("case_index_filter.json", false, true, qc, t)
	runMatch("case_index_filter.json", true, true, qc, t)
	runStmt(qc, "DROP INDEX orders.ifloix1")
	runStmt(qc, "DROP INDEX orders.ifloix2")

	runStmt(qc, "CREATE INDEX ieopix1 ON purchase (customerId, purchaseId, purchasedAt) WHERE test_id = \"arrayIndex\"")
	runMatch("case_early_order.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX purchase.ieopix1")

	runStmt(qc, "CREATE INDEX ioaix1 ON orders (ALL a1) WHERE test_id = \"parameters\"")
	runStmt(qc, "CREATE INDEX iorix1 ON orders (c1, c2, c3) WHERE test_id = \"idxfltr\"")
	runStmt(qc, "CREATE INDEX iorix2 ON orders (c1, c2, c4, c6) WHERE test_id = \"idxfltr\"")
	runStmt(qc, "CREATE INDEX iorix3 ON orders (attr.id, attr) WHERE test_id = \"indexbugs\"")
	runStmt(qc, "CREATE INDEX iorix4 ON orders (_type)")
	runStmt(qc, "CREATE INDEX iorix5 ON orders (_type, attr.id) WHERE _type = \"doc\"")
	runStmt(qc, "CREATE INDEX iorix6 ON orders (c11, c12) WHERE type = \"doc1\"")
	runStmt(qc, "CREATE INDEX iorix7 ON orders (c11, c12, type) WHERE type IN [ \"doc1\", \"doc2\", \"doc3\" ]")
	runStmt(qc, "CREATE INDEX ishix1 ON shellTest (c1, c2)")
	runStmt(qc, "CREATE INDEX ishix2 ON shellTest (c2, c1)")
	runStmt(qc, "CREATE INDEX ishix3 ON shellTest(c5, c6, c7, c8, c9)")
	runStmt(qc, "CREATE INDEX ishix4 ON shellTest(c5, c8, c10) WHERE c6 != 1 AND c7 NOT IN [1,2] AND c11 != 0")
	runStmt(qc, "CREATE INDEX ishix5 ON shellTest(id) WHERE type = \"type1\"")
	runStmt(qc, "CREATE INDEX ishix6 ON shellTest(id, type) WHERE type NOT IN [\"type2\",\"type3\"]")
	runMatch("case_index_scan_bugs.json", false, true, qc, t)
	runStmt(qc, "DROP INDEX orders.ioaix1")
	runStmt(qc, "DROP INDEX orders.iorix1")
	runStmt(qc, "DROP INDEX orders.iorix2")
	runStmt(qc, "DROP INDEX orders.iorix3")
	runStmt(qc, "DROP INDEX orders.iorix4")
	runStmt(qc, "DROP INDEX orders.iorix5")
	runStmt(qc, "DROP INDEX orders.iorix6")
	runStmt(qc, "DROP INDEX orders.iorix7")
	runStmt(qc, "DROP INDEX shellTest.ishix1")
	runStmt(qc, "DROP INDEX shellTest.ishix2")
	runStmt(qc, "DROP INDEX shellTest.ishix3")
	runStmt(qc, "DROP INDEX shellTest.ishix4")
	runStmt(qc, "DROP INDEX shellTest.ishix5")
	runStmt(qc, "DROP INDEX shellTest.ishix6")

	runStmt(qc, "create primary index on product ")
	runStmt(qc, "create primary index on purchase")
	runStmt(qc, "create primary index on orders")

	rr := runStmt(qc, "delete from product where test_id IN [\"arrayIndex\", \"coveredIndex\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "delete from purchase where test_id = \"arrayIndex\"")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}
	rr = runStmt(qc, "delete from orders where test_id IN [\"ua\", \"skipranges\", \"ordernulls\", \"parameters\", \"idxfltr\", "+
		"\"indexbugs\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "drop primary index on product")
	runStmt(qc, "drop primary index on purchase")
	runStmt(qc, "drop primary index on orders")
}
