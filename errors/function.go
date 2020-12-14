//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

import (
	"fmt"
)

const (
	//FTS errors
	FTS_MISSING_PORT = 10003
	NODE_ACCESS_ERR  = 10004
	NODE_SERVICE_ERR = 10005
)

func NewFTSMissingPortErr(e string) Error {
	return &err{level: EXCEPTION, ICode: FTS_MISSING_PORT, IKey: "fts.url.format.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("Missing or Incorrect port in input url."),
		InternalCaller: CallerN(1)}
}

func NewNodeInfoAccessErr(e string) Error {
	return &err{level: EXCEPTION, ICode: NODE_ACCESS_ERR, IKey: "node.access.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("Issue with accessing node information for rest endpoint %v", e),
		InternalCaller: CallerN(1)}
}

func NewNodeServiceErr(e string) Error {
	return &err{level: EXCEPTION, ICode: NODE_SERVICE_ERR, IKey: "node.service.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("No FTS node in server %v", e),
		InternalCaller: CallerN(1)}
}

func NewFunctionsNotSupported(what string) Error {
	return &err{level: EXCEPTION, ICode: 10100, IKey: "function.CE.error",
		InternalMsg:    fmt.Sprintf("Functions of type %v are only supported in Enterprise Edition", what),
		InternalCaller: CallerN(1)}
}

func NewMissingFunctionError(f string) Error {
	return &err{level: EXCEPTION, ICode: 10101, IKey: "function.missing.error",
		InternalMsg:    fmt.Sprintf("Function not found %v", f),
		InternalCaller: CallerN(1)}
}

func NewDuplicateFunctionError(f string) Error {
	return &err{level: EXCEPTION, ICode: 10102, IKey: "function.duplicate.error", ICause: fmt.Errorf("%v", f),
		InternalMsg:    fmt.Sprintf("Function already exists %v", f),
		InternalCaller: CallerN(1)}
}

func NewInternalFunctionError(e error, f string) Error {
	return &err{level: EXCEPTION, ICode: 10103, IKey: "function.internal.error", ICause: e,
		InternalMsg:    fmt.Sprintf("Operation on function %v encountered an unexpected error %v. Please collect the failing statement and contact support", f, e),
		InternalCaller: CallerN(1)}
}

func NewArgumentsMismatchError(f string) Error {
	return &err{level: EXCEPTION, ICode: 10104, IKey: "function.mismatching.error", ICause: fmt.Errorf("%v", f),
		InternalMsg:    fmt.Sprintf("Incorrect number of arguments supplied to function %v", f),
		InternalCaller: CallerN(1)}
}

func NewInvalidFunctionNameError(name string, e error) Error {
	return &err{level: EXCEPTION, ICode: 10105, IKey: "function.name.error", ICause: e,
		InternalMsg:    fmt.Sprintf("Invalid function name %v", name),
		InternalCaller: CallerN(1)}
}

func NewMetaKVError(where string, what error) Error {
	return &err{level: EXCEPTION, ICode: 10106, IKey: "function.storage.error", ICause: what,
		InternalMsg:    fmt.Sprintf("Could not access function definition for %v because %v", where, what),
		InternalCaller: CallerN(1)}
}

// same number and key as above, not an error
func NewMetaKVChangeCounterError(what error) Error {
	return &err{level: EXCEPTION, ICode: 10106, IKey: "function.storage.error", ICause: what,
		InternalMsg:    fmt.Sprintf("Could not access functions change counter because %v", what),
		InternalCaller: CallerN(1)}
}

// same number and key as above, not an error
func NewMetaKVIndexError(what error) Error {
	return &err{level: EXCEPTION, ICode: 10106, IKey: "function.storage.error", ICause: what,
		InternalMsg:    fmt.Sprintf("Could not access functions definitions because %v", what),
		InternalCaller: CallerN(1)}
}

func NewFunctionEncodingError(what string, name string, reason error) Error {
	return &err{level: EXCEPTION, ICode: 10107, IKey: "function.encoding.error", ICause: reason,
		InternalMsg:    fmt.Sprintf("Could not %v function definition for %v because %v", what, name, reason),
		InternalCaller: CallerN(1)}
}

func NewFunctionsDisabledError(what string) Error {
	return &err{level: EXCEPTION, ICode: 10108, IKey: "function.golang.disabled.error",
		InternalMsg: fmt.Sprintf("%v functions are disabled", what), InternalCaller: CallerN(1)}
}

func NewFunctionExecutionError(what string, name string, reason error) Error {
	return &err{level: EXCEPTION, ICode: 10109, IKey: "function.execution.error", ICause: reason,
		InternalMsg:    fmt.Sprintf("Error executing function %v %v: %v", name, what, reason),
		InternalCaller: CallerN(1)}
}
