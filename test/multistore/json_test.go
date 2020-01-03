//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package multistore

import (
	"fmt"
	"io/ioutil"
	"testing"
)

/*
Method to pass in parameters for site, pool and
namespace to Start method for Datastore.
*/
func start_ds() *MockServer {
	return Start("dir:", "../../data/sampledb/", Namespace_FS)
}

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
func TestSyntaxErr(t *testing.T) {
	qc := start_ds()

	r, _, err := Run(qc, "this is a bad query", Namespace_FS)
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}
	r, _, err = Run(qc, "", Namespace_FS) // empty string query
	if err == nil || len(r) != 0 {
		t.Errorf("expected err")
	}

	qccs := start_cs()

	rcs, _, errcs := Run(qccs, "this is a bad query", Namespace_CBS)
	if errcs == nil || len(rcs) != 0 {
		t.Errorf("expected err")
	}
	rcs, _, errcs = Run(qccs, "", Namespace_CBS) // empty string query
	if errcs == nil || len(rcs) != 0 {
		t.Errorf("expected err")
	}

}

/*
Basic test to ensure connections to both
Datastore and Couchbase server, work.
*/
func TestSimpleSelect(t *testing.T) {
	qc := start_ds()
	qccs := start_cs()

	r, _, err := Run(qc, "select 1 + 1", Namespace_FS)
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 result length")
	}

	r, _, err = Run(qccs, "select * from system:keyspaces", Namespace_CBS)
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 result length")
	}

	r, _, err = Run(qc, "select * from customer", Namespace_FS)
	if err != nil {
		t.Errorf("did not expect err %s", err.Error())
	}
	if len(r) == 0 {
		t.Errorf("unexpected 0 result length")
	}

	fileInfos, _ := ioutil.ReadDir("../../data/sampledb/dimestore/customer")
	if len(r) != len(fileInfos) {
		fmt.Printf("num results : %#v, fileInfos: %#v\n", len(r), len(fileInfos))
		t.Errorf("expected # of results to match directory listing")
	}

}

/*
Insert data into the buckets created earlier
using the statements in case_insert.json.
func TestInsertCaseFiles(t *testing.T) {
	fmt.Print("\n\nInserting values into Buckets \n\n ")
	qc := start_cs()
	matches, err := filepath.Glob("./case_insert.json")
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}
	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, err := FtestCaseFile(m, qc)
		if err != nil {
			t.Errorf("Error received : %s \n", err)
			return
		}
		if stmt != "" {
			t.Logf(" %v\n", stmt)
		}
		fmt.Print("\nQuery matched: ", m, "\n\n")
	}
}
*/
