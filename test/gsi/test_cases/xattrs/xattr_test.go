//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package xattrs

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
func TestXattrs(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runStmt(qc, "create primary index on product")

	fmt.Println("\n\nInserting values into Bucket for Xattrs test \n\n ")
	runMatch("insert.json", false, false, qc, t)

	gocb_SetupXattr()

	// Test for deleted xattrs
	runStmt(qc, "delete from product where meta().id = 'product0_xattrs'")

	// Test non covering index
	runMatch("case_xattrs.json", false, false, qc, t)

	_, _, errcs := runStmt(qc, "delete from product where test_id = \"xattrs\"")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "drop primary index on product")
}
