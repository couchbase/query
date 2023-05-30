//  Copyright 2018-Present Couchbase, Inc.
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

func NewFTSMissingPortErr(e string) Error {
	return &err{level: EXCEPTION, ICode: E_FTS_MISSING_PORT_ERR, IKey: "fts.url.format.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("Missing or Incorrect port in input url."),
		InternalCaller: CallerN(1)}
}

func NewNodeInfoAccessErr(e string) Error {
	return &err{level: EXCEPTION, ICode: E_NODE_INFO_ACCESS_ERR, IKey: "node.access.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("Issue with accessing node information for rest endpoint '%v'", e),
		InternalCaller: CallerN(1)}
}

func NewNodeServiceErr(e string) Error {
	return &err{level: EXCEPTION, ICode: E_NODE_SERVICE_ERR, IKey: "node.service.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("No FTS node in server %v", e),
		InternalCaller: CallerN(1)}
}

func NewFunctionsNotSupported(what string) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTIONS_NOT_SUPPORTED, IKey: "function.CE.error",
		InternalMsg:    fmt.Sprintf("Functions of type %v are only supported in Enterprise Edition", what),
		InternalCaller: CallerN(1)}
}

func NewMissingFunctionError(f string) Error {
	return &err{level: EXCEPTION, ICode: E_MISSING_FUNCTION, IKey: "function.missing.error",
		InternalMsg:    fmt.Sprintf("Function '%v' not found", f),
		InternalCaller: CallerN(1)}
}

func NewDuplicateFunctionError(f string) Error {
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_FUNCTION, IKey: "function.duplicate.error", ICause: fmt.Errorf("%v", f),
		InternalMsg:    fmt.Sprintf("Function '%v' already exists", f),
		InternalCaller: CallerN(1)}
}

func NewInternalFunctionError(e error, f string) Error {
	if f != "" {
		return &err{level: EXCEPTION, ICode: E_INTERNAL_FUNCTION, IKey: "function.internal.error", ICause: e,
			InternalMsg: fmt.Sprintf("Operation on function '%v' encountered an unexpected error: %v. "+
				"Please collect the failing statement and contact support", f, e),
			InternalCaller: CallerN(1)}
	} else {
		return &err{level: EXCEPTION, ICode: E_INTERNAL_FUNCTION, IKey: "function.internal.error", ICause: e,
			InternalMsg: fmt.Sprintf("Operation on function encountered an unexpected error: %v. "+
				"Please collect the failing statement and contact support", e),
			InternalCaller: CallerN(1)}
	}
}

func NewArgumentsMismatchError(f string) Error {
	return &err{level: EXCEPTION, ICode: E_ARGUMENTS_MISMATCH, IKey: "function.mismatching.error", ICause: fmt.Errorf("%v", f),
		InternalMsg:    fmt.Sprintf("Incorrect number of arguments supplied to function '%v'", f),
		InternalCaller: CallerN(1)}
}

func NewInvalidFunctionNameError(name string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_FUNCTION_NAME, IKey: "function.name.error", ICause: e,
		InternalMsg:    fmt.Sprintf("Invalid function name '%v'", name),
		InternalCaller: CallerN(1)}
}

func NewMetaKVError(where string, what error) Error {
	c := make(map[string]interface{})
	c["cause"] = where
	return &err{level: EXCEPTION, ICode: E_FUNCTIONS_STORAGE, IKey: "function.storage.error", ICause: what, cause: c,
		InternalMsg:    fmt.Sprintf("Could not access function definition for %v because %v", where, what),
		InternalCaller: CallerN(1)}
}

// same number and key as above, not an error
func NewMetaKVChangeCounterError(what error) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTIONS_STORAGE, IKey: "function.storage.error", ICause: what,
		InternalMsg:    fmt.Sprintf("Could not access functions change counter because %v", what),
		InternalCaller: CallerN(1)}
}

// same number and key as above, not an error
func NewStorageAccessError(where string, what error) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTIONS_STORAGE, IKey: "function.storage.error", ICause: what,
		InternalMsg:    fmt.Sprintf("Could not access functions definitions during %v because %v", where, what),
		InternalCaller: CallerN(1)}
}

func NewFunctionEncodingError(what string, name string, reason error) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTION_ENCODING, IKey: "function.encoding.error", ICause: reason,
		InternalMsg:    fmt.Sprintf("Could not %v function definition for '%v' because %v", what, name, reason),
		InternalCaller: CallerN(1)}
}

func NewFunctionsDisabledError(what string) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTIONS_DISABLED, IKey: "function.golang.disabled.error",
		InternalMsg: fmt.Sprintf("%v functions are disabled", what), InternalCaller: CallerN(1)}
}

func NewFunctionExecutionError(what string, name string, reason interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTION_EXECUTION, IKey: "function.execution.error", cause: reason,
		InternalMsg:    fmt.Sprintf("Error executing function '%v' %v", name, what),
		InternalCaller: CallerN(1)}
}

func NewInnerFunctionExecutionError(what string, name string, reason error) Error {
	return &err{level: EXCEPTION, ICode: E_INNER_FUNCTION_EXECUTION, IKey: "function.inner.error",
		InternalMsg:    fmt.Sprintf("%s", reason.Error()),
		InternalCaller: CallerN(1)}
}

func NewFunctionExecutionNestedError(levels int, name string) Error {
	return &err{level: EXCEPTION, ICode: E_TOO_MANY_NESTED_FUNCTIONS, IKey: "function.nested.error",
		InternalMsg:    fmt.Sprintf("Error executing function '%v': %v nested javascript calls", name, levels),
		InternalCaller: CallerN(1)}
}

func NewFunctionLibraryPathError(path string) Error {
	return &err{level: EXCEPTION, ICode: E_LIBRARY_PATH_ERROR, IKey: "function.library.path.error",
		InternalMsg:    fmt.Sprintf("Invalid javascript library path: %v. Use a root level path, the same path as the function scope, or a local path ('./library')", path),
		InternalCaller: CallerN(1)}
}

func NewAdvisorSessionNotFoundError(s string) Error {
	c := make(map[string]interface{})
	c["unknown_session"] = s
	return &err{level: EXCEPTION, ICode: E_ADVISOR_SESSION_NOT_FOUND, IKey: "function.advisor.session_not_found",
		InternalMsg: "Advisor: Session not found.", cause: c, InternalCaller: CallerN(1)}
}

func NewAdvisorActionNotValid(a string) Error {
	c := make(map[string]interface{})
	c["invalid_action"] = a
	return &err{level: EXCEPTION, ICode: E_ADVISOR_INVALID_ACTION, IKey: "function.advisor.invalid_action",
		InternalMsg: "Advisor: Invalid value for 'action'", cause: c, InternalCaller: CallerN(1)}
}

func NewAdvisorActionMissing() Error {
	return &err{level: EXCEPTION, ICode: E_ADVISOR_ACTION_MISSING, IKey: "function.advisor.action_missing",
		InternalMsg: "Advisor: missing argument for 'action'", InternalCaller: CallerN(1)}
}

func NewAdvisorInvalidArgs(args []string) Error {
	c := make(map[string]interface{})
	c["args"] = args
	return &err{level: EXCEPTION, ICode: E_ADVISOR_INVALID_ARGS, IKey: "function.advisor.invalid_arguments",
		InternalMsg: "Advisor: Invalid arguments.", cause: c, InternalCaller: CallerN(1)}
}

func NewFunctionLoadingError(function string, reason interface{}) Error {
	c := make(map[string]interface{})
	c["function"] = function
	c["reason"] = reason
	return &err{level: EXCEPTION, ICode: E_FUNCTION_LOADING, IKey: "function.loading.error", cause: c,
		InternalMsg:    fmt.Sprintf("Error loading function '%v'", function),
		InternalCaller: CallerN(1)}
}

func NewEvaluatorLoadingError(tenant string, reason interface{}) Error {
	c := make(map[string]interface{})
	c["tenant"] = tenant
	c["reason"] = reason
	return &err{level: EXCEPTION, ICode: E_FUNCTION_LOADING, IKey: "function.tenant.loading.error", cause: c,
		InternalMsg:    fmt.Sprintf("Error loading evaluator for '%v'", tenant),
		InternalCaller: CallerN(1)}
}

func NewEvaluatorInflatingError(tenant string, reason interface{}) Error {
	c := make(map[string]interface{})
	c["tenant"] = tenant
	c["reason"] = reason
	return &err{level: EXCEPTION, ICode: E_FUNCTION_LOADING, IKey: "function.tenant.inflating.error", cause: c,
		InternalMsg:    fmt.Sprintf("Error adding javascript runners to evaluator for '%v'", tenant),
		InternalCaller: CallerN(1)}
}

func NewFunctionUnsupportedActionError(fType string, action string) Error {
	return &err{level: EXCEPTION, ICode: E_FUNCTIONS_UNSUPPORTED_ACTION, IKey: "function.unsupported.action.error",
		InternalMsg:    fmt.Sprintf("%s is not supported for functions of type %s", action, fType),
		InternalCaller: CallerN(1)}
}
