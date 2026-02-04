//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import "fmt"

var _encryptionErrs = map[ErrorCode][2]string{
	E_ENCRYPTION_READER_CREATE: {"encryption.reader.create", "Failed to create encryption reader"},
	E_ENCRYPTION_WRITER_CREATE: {"encryption.writer.create", "Failed to create encryption writer"},
}

func NewEncryptionError(code ErrorCode, e error, msgArgs ...interface{}) Error {
	ee, ok := _encryptionErrs[code]
	if !ok {
		panic(fmt.Sprintf("No encryption error message defined for error code: %d", code))
	}

	rv := &err{level: EXCEPTION, ICode: code, InternalCaller: CallerN(1), IKey: ee[0], InternalMsg: ee[1], cause: e}

	if len(msgArgs) > 0 {
		rv.InternalMsg = fmt.Sprintf(rv.InternalMsg, msgArgs...)
	}

	return rv
}
