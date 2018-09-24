//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package n1qlFts

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Basic test on cobering indexes
func TestN1qlFts(t *testing.T) {

	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runStmt(qc, "create primary index on product")

	fmt.Println("\n\nInserting values into Bucket for N1QL FTS integration test \n\n ")
	runMatch("insert.json", false, false, qc, t)

	err := setupftsIndex()
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}

	runMatch("case_n1qlftsint.json", false, false, qc, t)

	err = deleteFTSIndex()
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	_, _, errcs := runStmt(qc, "delete from product where test_id = \"n1qlfts\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "drop primary index on product")
}
