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

func NewMoreThanOneRecursiveRefError(ref string) Error {
	return &err{level: EXCEPTION, ICode: E_MORE_THAN_ONE_RECURSIVE_REF,
		InternalMsg:    fmt.Sprintf("recursive ref:%s must not appear more than once in the FROM clause", ref),
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
