//  Copyright (c) 2014 Couchbase, Inc.
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

// Plan errors - errors that are created in the plan and algebra packages

func NewPlanError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 4000, IKey: "plan_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

const NO_TERM_NAME = 4010

func NewNoTermNameError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: NO_TERM_NAME, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term must have a name or alias", termType), InternalCaller: CallerN(1)}
}

const DUPLICATE_ALIAS = 4020

func NewDuplicateAliasError(termType string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: DUPLICATE_ALIAS, IKey: iKey,
		InternalMsg: fmt.Sprintf("Duplicate %s alias %s", termType, alias), InternalCaller: CallerN(1)}
}

const SUBQUERY_MISSING_KEYS = 4030

func NewSubqueryMissingKeysError(keyspace string) Error {
	return &err{level: EXCEPTION, ICode: SUBQUERY_MISSING_KEYS, IKey: "plan.build_select.subquery_missing_keys",
		InternalMsg: fmt.Sprintf("FROM in subquery must use KEYS clause: FROM %s.", keyspace), InternalCaller: CallerN(1)}
}

const NO_SUCH_PREPARED = 4040

func NewNoSuchPreparedError(name string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_PREPARED, IKey: "plan.build_prepared.no_such_name",
		InternalMsg: fmt.Sprintf("No such prepared statement: %s", name), InternalCaller: CallerN(1)}
}

func NewUnrecognizedPreparedError(e error) Error {
	return &err{level: EXCEPTION, ICode: 4050, IKey: "plan.build_prepared.unrecognized_prepared",
		ICause: e, InternalMsg: "Unrecognizable prepared statement", InternalCaller: CallerN(1)}
}

func NewPreparedNameError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 4060, IKey: "plan.build_prepared.no_such_name",
		InternalMsg: fmt.Sprintf("Unable to add name: %s", msg), InternalCaller: CallerN(1)}
}

func NewPreparedDecodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: 4070, IKey: "plan.build_prepared.decoding",
		ICause: e, InternalMsg: "Unable to decode prepared statement", InternalCaller: CallerN(1)}
}
