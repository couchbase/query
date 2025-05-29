//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package vectors

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on Vector Search
func TestVectors(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nChecking import into vectors collection\n\n")
	runMatch("check_import.json", false, false, qc, t)
	if !t.Failed() {
		fmt.Print("Import succeeded\n\n")
		os.Remove("cbimport.out")
	}

	fmt.Println("Running Advise test cases")

	runMatch("case_advise.json", false, false, qc, t)

	fmt.Println("Running VECTOR_DISTANCE test cases")

	runStmt(qc, "CREATE INDEX ix_prod_id on product._default.vectors(id)")

	runMatch("case_knn.json", false, false, qc, t)

	fmt.Println("Vector index with only vector key")

	runStmt(qc, "CREATE INDEX idx_vec1 on product._default.vectors(vec VECTOR) WITH {'dimension': 128, 'train_list': 10000, 'description': 'IVF32,SQ8', 'similarity': 'L2'}")

	runMatch("case_single_key.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX idx_vec1 on product._default.vectors")

	fmt.Println("Vector index with leading scalar keys and vector key")

	runStmt(qc, "CREATE INDEX idx_vec2 on product._default.vectors(size, brand, vec VECTOR, color) WITH {'dimension': 128, 'train_list': 10000, 'description': 'IVF32,SQ8', 'similarity': 'L2'}")

	runMatch("case_composite_nonleading.json", false, true, qc, t)

	runMatch("case_composite_no_pushdown.json", false, true, qc, t)

	runMatch("case_composite_rerank.json", false, true, qc, t)

	runMatch("case_composite_error.json", false, false, qc, t)

	runStmt(qc, "DROP INDEX idx_vec2 on product._default.vectors")

	// Bhive index test cases

	fmt.Println("Bhive vector index")

	runStmt(qc, "CREATE VECTOR INDEX idx_vec3 on product._default.vectors(vec VECTOR) WITH {'dimension': 128, 'train_list': 10000, 'description': 'IVF32,SQ8', 'similarity': 'L2'}")

	runMatch("case_bhive_single_key.json", false, true, qc, t)

	runStmt(qc, "DROP INDEX idx_vec3 on product._default.vectors")

	fmt.Println("Bhive vector index with include columns")

	runStmt(qc, "CREATE VECTOR INDEX idx_vec4 on product._default.vectors(vec VECTOR) INCLUDE (brand, size, color) WITH {'dimension': 128, 'train_list': 10000, 'description': 'IVF32,SQ8', 'similarity': 'L2'}")

	runMatch("case_bhive_include.json", false, true, qc, t)

	runMatch("case_bhive_error.json", false, false, qc, t)

	runStmt(qc, "DROP INDEX idx_vec4 on product._default.vectors")

	fmt.Println("Dropping indexes")
	runStmt(qc, "DROP INDEX ix_prod_id on product._default.vectors")

	// create primary indexes
	runStmt(qc, "CREATE PRIMARY INDEX ON product._default.vectors")

	// delete all rows from keyspaces used
	runStmt(qc, "DELETE FROM product._default.vectors")

	// drop primary indexes
	runStmt(qc, "DROP PRIMARY INDEX ON product._default.vectors")
}
