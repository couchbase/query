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
	"path/filepath"
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.
*/
func start_cs() *gsi.MockServer {
	ms := gsi.Start(gsi.Site_CBS, gsi.Auth_param+"@"+gsi.Pool_CBS, gsi.Namespace_CBS)

	return ms
}

func runMatch(filename string, qc *gsi.MockServer, t *testing.T) {

	matches, err := filepath.Glob(filename)
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}

	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, errcs := gsi.FtestCaseFile(m, qc, gsi.Namespace_CBS)

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

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.Run(mockServer, q, gsi.Namespace_CBS)
}
