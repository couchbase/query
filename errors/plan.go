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

const UNKNOWN_FOR = 4025

func NewUnknownForError(termType string, keyFor string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: UNKNOWN_FOR, IKey: iKey,
		InternalMsg: fmt.Sprintf("Unknown %s for alias %s", termType, keyFor), InternalCaller: CallerN(1)}
}

const SUBQUERY_MISSING_KEYS = 4030

func NewSubqueryMissingKeysError(keyspace string) Error {
	return &err{level: EXCEPTION, ICode: SUBQUERY_MISSING_KEYS, IKey: "plan.build_select.subquery_missing_keys",
		InternalMsg: fmt.Sprintf("FROM in correlated subquery must have USE KEYS clause: FROM %s.", keyspace), InternalCaller: CallerN(1)}
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

func NewPreparedEncodingMismatchError(name string) Error {
	return &err{level: EXCEPTION, ICode: 4080, IKey: "plan.build_prepared.name_encoded_plan_mismatch",
		InternalMsg: fmt.Sprintf("Encoded plan parameter does not match encoded plan of %s", name), InternalCaller: CallerN(1)}
}

const PLAN_NAME_MISMATCH = 4090

func NewEncodingNameMismatchError(name string) Error {
	return &err{level: EXCEPTION, ICode: PLAN_NAME_MISMATCH, IKey: "plan.build_prepared.name_not_in_encoded_plan",
		InternalMsg: fmt.Sprintf("Prepared name in encoded plan parameter is not %s", name), InternalCaller: CallerN(1)}
}

const NO_INDEX_JOIN = 4100

func NewNoIndexJoinError(alias, op string) Error {
	return &err{level: EXCEPTION, ICode: NO_INDEX_JOIN, IKey: fmt.Sprintf("plan.index_%s.no_index", op),
		InternalMsg: fmt.Sprintf("No index available for join term %s", alias), InternalCaller: CallerN(1)}
}

const NOT_GROUP_KEY_OR_AGG = 4210

func NewNotGroupKeyOrAggError(expr string) Error {
	return &err{level: EXCEPTION, ICode: NOT_GROUP_KEY_OR_AGG, IKey: "plan.not_group_key_or_agg",
		InternalMsg: fmt.Sprintf("Expression must be a group key or aggregate: %s", expr), InternalCaller: CallerN(1)}
}

const NEW_INDEX_ALREADY_EXISTS = 4300

func NewIndexAlreadyExistsError(idx string) Error {
	return &err{level: EXCEPTION, ICode: NEW_INDEX_ALREADY_EXISTS,
		IKey:           "plan.new_index_already_exists",
		InternalMsg:    fmt.Sprintf("The index %s already exists.", idx),
		InternalCaller: CallerN(1)}
}
