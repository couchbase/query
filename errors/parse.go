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
	"strings"
)

// Parse errors - errors that are created in the parse package
func NewParseSyntaxError(e interface{}, msg string) Error {
	var basicError error
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	case []string:
		basicError = fmt.Errorf("%s", strings.Join(e, " \n "))
	case error:
		basicError = e
	case nil:
	default:
		basicError = fmt.Errorf("%v", e)
	}
	return &err{level: EXCEPTION, ICode: E_PARSE_SYNTAX, IKey: "parse.syntax_error", ICause: basicError,
		InternalMsg: msg, InternalCaller: CallerN(1)}
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

func NewParseInvalidInput(what string) Error {
	c := make(map[string]interface{})
	c["expected"] = what
	return &err{level: EXCEPTION, ICode: E_PARSE_INVALID_INPUT, IKey: "parse.invalid_input", cause: c,
		InternalMsg: "Invalid input.", InternalCaller: CallerN(1)}
}

func NewAmbiguousReferenceError(ident string, errorContext string) Error {
	return &err{level: EXCEPTION, ICode: E_AMBIGUOUS_REFERENCE, IKey: "formalize.ambiguous_reference",
		InternalMsg:    fmt.Sprintf("Ambiguous reference to field '%v'%v.", ident, errorContext),
		InternalCaller: CallerN(1)}
}

func NewDuplicateVariableError(variable string, errorContext string) Error {
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_VARIABLE, IKey: "formalize.duplicate_variable",
		InternalMsg:    fmt.Sprintf("Duplicate variable: '%v' already in scope%s.", variable, errorContext),
		InternalCaller: CallerN(1)}
}

func NewFormalizerInternalError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FORMALIZER_INTERNAL, IKey: "formalize.internal_error",
		InternalMsg:    fmt.Sprintf("Formalizer internal error: %s", msg),
		InternalCaller: CallerN(1)}
}
