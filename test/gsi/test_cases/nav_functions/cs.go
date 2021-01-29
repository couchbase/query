//  Copyright (c) 2021 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
package nav_functions

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
	"testing"
)

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}
