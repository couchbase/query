//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package cover

import (
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error, int) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
