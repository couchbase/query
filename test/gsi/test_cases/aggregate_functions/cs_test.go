//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

	runMatch("case_distinct.json", false, false, qc, t)
	runMatch("case_group_by_having.json", false, false, qc, t)
	runMatch("case_median_stddev_variance.json", false, false, qc, t)

	runStmt(qc, "delete from product where test_id IN [\"agg_func\"]")
	runStmt(qc, "delete from orders where test_id IN [\"agg_func\",\"median_agg_func\",\"cntn_agg_func\"]")

	runStmt(qc, "DROP PRIMARY INDEX ON product")
	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}
