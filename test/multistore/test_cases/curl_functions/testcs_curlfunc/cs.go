//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
package testcs_curlfunc

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	js "github.com/couchbase/query/test/multistore"
)

func Start_test() *js.MockServer {
	// We need custom credentials in this case, because the CURL function requires
	// special permission. The right permission to use in this case would be
	// the query_external_access role, but that hasn't been implemented yet. Hence
	// the use of Administrator.
	js.AUTH_USERS = auth.Credentials{"Administrator": "password"}

	return js.Start(js.Site_CBS, js.Auth_param+"@"+js.Pool_CBS, js.Namespace_CBS)
}

func testCaseFile(fname string, qc *js.MockServer) (fin_stmt string, errstring error) {
	fin_stmt, errstring = js.FtestCaseFile(fname, qc, js.Namespace_CBS)
	return
}

func Run_test(mockServer *js.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return js.Run(mockServer, q, js.Namespace_CBS)
}
