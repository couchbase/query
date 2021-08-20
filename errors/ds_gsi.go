//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package errors

import (
	"fmt"
)

func NewIndexScanSizeError(size int64) Error {
	return &err{level: EXCEPTION, ICode: E_INDEX_SCAN_SIZE, IKey: "datastore.index.scan_size_error",
		InternalMsg: fmt.Sprintf("Unacceptable size for index scan: %d", size), InternalCaller: CallerN(1)}
}
