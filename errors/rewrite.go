//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package errors

// rewrite errors 6500-6599

const _REWRITE_ERROR = 6500

func NewRewriteError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: _REWRITE_ERROR, IKey: "rewrite_error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}
