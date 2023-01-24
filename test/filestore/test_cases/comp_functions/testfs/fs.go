// Copyright 2013-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.
package testfs

import (
	"github.com/couchbase/query/errors"
	js "github.com/couchbase/query/test/filestore"
)

func start() *js.MockServer {
	return js.Start("dir:", "../../../data/", js.Namespace_FS)
}

func testCaseFile(fname string, qc *js.MockServer) (fin_stmt string, errstring error) {
	fin_stmt, errstring = js.FtestCaseFile(fname, qc, js.Namespace_FS)
	return
}

func Run_test(mockServer *js.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return js.Run(mockServer, true, q, nil, nil, js.Namespace_FS)
}
