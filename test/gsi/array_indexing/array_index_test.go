//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package arrayIndex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.
*/
func start_cs() *MockServer {
	return Start(Site_CBS, Auth_param+"@"+Pool_CBS, Namespace_CBS)
}

/*
Basic test to ensure connections to both
Datastore and Couchbase server, work.
*/
func TestArrayIndex(t *testing.T) {
	var RunTest bool

	val := os.Getenv("GSI_TEST")
	if strings.ToLower(val) == "true" {
		RunTest = true
	} else {
		RunTest = false
	}

	if RunTest {
		qc := start_cs()

		fmt.Println("\n\nInserting values into Bucket for Array Indexing \n\n ")
		runMatch("insert.json", qc, t)

		Run(qc, "CREATE INDEX reviewlistidx on product(ALL DISTINCT ARRAY r  FOR r IN reviewList END)", Namespace_CBS)
		Run(qc, "CREATE INDEX reviewlistidx2 on product(ALL DISTINCT ARRAY r  FOR r IN reviewList END, productId)", Namespace_CBS)
		Run(qc, "CREATE INDEX reviewlistidx3 on product(productId, ALL DISTINCT ARRAY r  FOR r IN reviewList END)", Namespace_CBS)
		Run(qc, "CREATE INDEX plistidx on purchase (ALL DISTINCT ARRAY l for l in lineItems END)", Namespace_CBS)

		runMatch("case_array_index_test1.json", qc, t)

		Run(qc, "DROP INDEX product.reviewlistidx", Namespace_CBS)
		Run(qc, "DROP INDEX product.reviewlistidx2", Namespace_CBS)
		Run(qc, "DROP INDEX product.reviewlistidx3", Namespace_CBS)
		Run(qc, "DROP INDEX purchase.plistidx", Namespace_CBS)

		Run(qc, "CREATE INDEX reviewlistidxall on product(ALL ARRAY r  FOR r IN reviewList END)", Namespace_CBS)
		Run(qc, "CREATE INDEX reviewlistidx2all on product(ALL ARRAY r  FOR r IN reviewList END, productId)", Namespace_CBS)
		Run(qc, "CREATE INDEX reviewlistidx3all on product(productId, ALL ARRAY r  FOR r IN reviewList END)", Namespace_CBS)
		Run(qc, "CREATE INDEX plistidx on purchase (ALL DISTINCT ARRAY l for l in lineItems END)", Namespace_CBS)

		runMatch("case_array_index_test2.json", qc, t)

		Run(qc, "DROP INDEX product.reviewlistidxall", Namespace_CBS)
		Run(qc, "DROP INDEX product.reviewlistidx2all", Namespace_CBS)
		Run(qc, "DROP INDEX product.reviewlistidx3all", Namespace_CBS)
		Run(qc, "DROP INDEX purchase.plistidx", Namespace_CBS)

		// Single-level Indexes
		Run(qc, "CREATE INDEX iv ON product( DISTINCT ARRAY v FOR v IN b END )", Namespace_CBS)
		Run(qc, "CREATE INDEX ix ON product( DISTINCT ARRAY v.x FOR v IN b END )", Namespace_CBS)
		Run(qc, "CREATE INDEX ixy ON product( DISTINCT ARRAY v.x + v.y FOR v IN b END )", Namespace_CBS)

		//As covering indexes
		Run(qc, "CREATE INDEX cover_iv ON product( DISTINCT ARRAY v FOR v IN b END, b, type)", Namespace_CBS)
		Run(qc, "CREATE INDEX cover_ix ON product( DISTINCT ARRAY v.x FOR v IN b END,b, type )", Namespace_CBS)
		Run(qc, "CREATE INDEX cover_ixy ON product( DISTINCT ARRAY v.x + v.y FOR v IN b END,b, type )", Namespace_CBS)

		//		runMatch("case_explain_test3.json", qc, t)

		//Multi-level index
		Run(qc, "DROP INDEX product.iv", Namespace_CBS)
		Run(qc, "DROP INDEX product.ix", Namespace_CBS)
		Run(qc, "DROP INDEX product.ixy", Namespace_CBS)
		Run(qc, "DROP INDEX product.cover_iv", Namespace_CBS)
		Run(qc, "DROP INDEX product.cover_ix", Namespace_CBS)
		Run(qc, "DROP INDEX product.cover_ixy", Namespace_CBS)

		Run(qc, "CREATE INDEX ivw ON product( DISTINCT ARRAY ( DISTINCT ARRAY w FOR w IN v END ) FOR v IN b END )", Namespace_CBS)
		Run(qc, "CREATE INDEX cover_ivw ON product( DISTINCT ARRAY ( DISTINCT ARRAY w FOR w IN v END ) FOR v IN b END,b, type )", Namespace_CBS)

		//		runMatch("case_explain_test4.json", qc, t)

		Run(qc, "DROP INDEX product.ivw", Namespace_CBS)
		Run(qc, "DROP INDEX product.cover_ivw", Namespace_CBS)

		// Create array index on TOKENS()
		Run(qc, "CREATE INDEX tokenindex ON product ((distinct (array lower(to_string(d)) for d in tokens(description) end)))", Namespace_CBS)
		Run(qc, "CREATE INDEX tokenindex1 on product(ALL ARRAY r  FOR r IN tokens(name) END)", Namespace_CBS)
		Run(qc, "CREATE INDEX tokenindex2 on product (ALL DISTINCT ARRAY l for l in tokens(imageURL) END)", Namespace_CBS)

		runMatch("case_array_index_tokens.json", qc, t)

		Run(qc, "DROP INDEX product.tokenindex", Namespace_CBS)
		Run(qc, "DROP INDEX product.tokenindex1", Namespace_CBS)
		Run(qc, "DROP INDEX product.tokenindex2", Namespace_CBS)

		Run(qc, "create primary index on product ", Namespace_CBS)
		Run(qc, "create primary index on purchase", Namespace_CBS)

		_, _, errcs := Run(qc, "delete from product where test_id = \"arrayIndex\"", Namespace_CBS)
		if errcs != nil {
			t.Errorf("did not expect err %s", errcs.Error())
		}

		_, _, errcs = Run(qc, "delete from product where type = \"coveredIndex\"", Namespace_CBS)
		if errcs != nil {
			t.Errorf("did not expect err %s", errcs.Error())
		}
		_, _, errcs = Run(qc, "delete from purchase where test_id = \"arrayIndex\"", Namespace_CBS)
		if errcs != nil {
			t.Errorf("did not expect err %s", errcs.Error())
		}
		Run(qc, "drop primary index on product", Namespace_CBS)
		Run(qc, "drop primary index on purchase", Namespace_CBS)
	}
}

func runMatch(filename string, qc *MockServer, t *testing.T) {

	matches, err := filepath.Glob(filename)
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}

	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, errcs := FtestCaseFile(m, qc, Namespace_CBS)

		if errcs != nil {
			t.Errorf("Error : %s", errcs.Error())
			return
		}

		if stmt != "" {
			t.Logf(" %v\n", stmt)
		}

		fmt.Println("\nQuery : ", m, "\n\n")
	}

}
