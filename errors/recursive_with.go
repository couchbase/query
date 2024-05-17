//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import "fmt"

func NewRecursiveWithSemanticError(cause string) Error {
	return &err{level: EXCEPTION, ICode: E_RECURSIVE_WITH_SEMANTIC,
		InternalMsg:    fmt.Sprintf("recursive_with semantics: %s", cause),
		InternalCaller: CallerN(1)}
}

func NewCycleFieldsValidationFailedError(cause error, alias string) Error {
	return &err{level: EXCEPTION, ICode: E_CYCLE_FIELDS_VALIDATION_FAILED,
		InternalMsg:    fmt.Sprintf("Cycle fields validation failed for with term: %s", alias),
		InternalCaller: CallerN(1),
		ICause:         cause,
	}
}

func NewMoreThanOneRecursiveRefError(ref string) Error {
	return &err{level: EXCEPTION, ICode: E_MORE_THAN_ONE_RECURSIVE_REF,
		InternalMsg:    fmt.Sprintf("Recursive reference '%s' must not appear more than once in the FROM clause", ref),
		InternalCaller: CallerN(2)}
}

func NewRecursiveAnchorError(termType string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_ANCHOR_RECURSIVE_REF, IKey: iKey, InternalCaller: CallerN(1),
		InternalMsg: fmt.Sprintf("Anchor Clause cannot have recursive reference in %s: %s", termType, alias)}
}

func NewInvalidConfigOptions(option string) Error {
	c := make(map[string]interface{})
	c["invalid_option"] = option
	return &err{level: EXCEPTION, ICode: E_CONFIG_INVALID_OPTION, cause: c,
		InternalMsg: fmt.Sprintf("Invalid config option %s", option), InternalCaller: CallerN(1)}
}

func NewRecursionUnsupportedError(clause string, cause string) Error {
	return &err{level: EXCEPTION, ICode: E_RECURSION_UNSUPPORTED,
		InternalMsg:    fmt.Sprintf("recursive_with_unsupported: %s", clause),
		cause:          cause,
		InternalCaller: CallerN(1)}
}

func NewRecursiveImplicitDocLimitError(alias string, limit int64) Error {
	c := make(map[string]interface{})
	c["alias"] = alias
	c["limit"] = limit
	return &err{level: WARNING, ICode: E_RECURSIVE_IMPLICIT_DOC_LIMIT, IKey: "recursive_with.implicit_docs_limit", cause: c,
		InternalMsg: fmt.Sprintf(
			"Recursive WITH '%s' limited to %v documents as no explicit document count limit or memory quota set",
			alias, limit), InternalCaller: CallerN(1)}
}

func NewRecursiveImplicitDepthLimitError(alias string, depth int64) Error {
	c := make(map[string]interface{})
	c["alias"] = alias
	c["depth"] = depth

	return &err{level: WARNING, ICode: E_RECURSIVE_IMPLICIT_DEPTH_LIMIT, IKey: "recursive_with.implicit_depth_limit", cause: c,
		InternalMsg: fmt.Sprintf(
			"Recursive WITH '%s' stopped at %v level as no explicit level limit or memory quota set",
			alias, depth), InternalCaller: CallerN(1)}
}
