//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func NewParseInvalidEscapeSequenceError() Error {
	return &err{level: EXCEPTION, ICode: E_PARSE_INVALID_ESCAPE_SEQUENCE, IKey: "parse.invalid_escape_sequence",
		InternalMsg: "invalid escape sequence", InternalCaller: CallerN(1)}
}

func NewParseInvalidStringError() Error {
	return &err{level: EXCEPTION, ICode: E_PARSE_INVALID_STRING, IKey: "parse.invalid_string",
		InternalMsg: "invalid string", InternalCaller: CallerN(1)}
}

func NewParseMissingClosingQuoteError() Error {
	return &err{level: EXCEPTION, ICode: E_PARSE_MISSING_CLOSING_QUOTE, IKey: "parse.missing_closing_quote",
		InternalMsg: "missing closing quote", InternalCaller: CallerN(1)}
}

func NewParseUnescapedEmbeddedQuoteError() Error {
	return &err{level: EXCEPTION, ICode: E_PARSE_UNESCAPED_EMBEDDED_QUOTE, IKey: "parse.unescaped_embedded_quote",
		InternalMsg: "unescaped embedded quote", InternalCaller: CallerN(1)}
}
