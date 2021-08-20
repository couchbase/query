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

// Parse errors - errors that are created in the parse package
func NewParseSyntaxError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: E_PARSE_SYNTAX, IKey: "parse.syntax_error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

// An error (albeit always text in another error) so that we can make use of translation
func NewErrorContext(line, column int) Error {
	return &err{level: EXCEPTION, ICode: E_ERROR_CONTEXT, IKey: "expression.error.context", ICause: nil,
		InternalMsg: fmt.Sprintf(" (near line %d, column %d)", line, column), InternalCaller: CallerN(1)}
}
