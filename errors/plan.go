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
	"strings"
)

// Plan errors - errors that are created in the prepared, planner and plan packages

func NewPlanError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 4000, IKey: "plan_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

func NewReprepareError(e error) Error {
	return &err{level: EXCEPTION, ICode: 4001, IKey: "reprepare_error", ICause: e, InternalMsg: "Reprepare error", InternalCaller: CallerN(1)}
}

/* error numbers 4010, 4020, 4025 moved to semantics.go */

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

func NewNoSuchPreparedWithContextError(name string, queryContext string) Error {
	if queryContext == "" {
		queryContext = "unset"
	}
	return &err{level: EXCEPTION, ICode: NO_SUCH_PREPARED, IKey: "plan.build_prepared.no_such_name",
		InternalMsg: fmt.Sprintf("No such prepared statement: %s, context: %s", name, queryContext), InternalCaller: CallerN(1)}
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

func NewEncodingNameMismatchError(expected, found string) Error {
	return &err{level: EXCEPTION, ICode: PLAN_NAME_MISMATCH, IKey: "plan.build_prepared.name_not_in_encoded_plan",
		InternalMsg: fmt.Sprintf("Mismatching name in encoded plan, expecting: %s, found: %s", expected, found), InternalCaller: CallerN(1)}
}

const PLAN_CONTEXT_MISMATCH = 4091

func NewEncodingContextMismatchError(name, expected, found string) Error {
	if expected == "" {
		expected = "unset"
	}
	if found == "" {
		found = "unset"
	}
	return &err{level: EXCEPTION, ICode: PLAN_CONTEXT_MISMATCH, IKey: "plan.build_prepared.context_not_in_encoded_plan",
		InternalMsg: fmt.Sprintf("Mismatching query_context in encoded plan, expecting: %s, found: %s", expected, found), InternalCaller: CallerN(1)}
}

func NewPredefinedPreparedNameError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 4092, IKey: "plan.build_prepared.reserved",
		InternalMsg: fmt.Sprintf("Prepared name %s is predefined (reserved). ", msg), InternalCaller: CallerN(1)}
}

const NO_INDEX_JOIN = 4100

func NewNoIndexJoinError(alias, op string) Error {
	return &err{level: EXCEPTION, ICode: NO_INDEX_JOIN, IKey: fmt.Sprintf("plan.index_%s.no_index", op),
		InternalMsg: fmt.Sprintf("No index available for join term %s", alias), InternalCaller: CallerN(1)}
}

/* error number 4110 moved to semantics.go */

const NOT_GROUP_KEY_OR_AGG = 4210

func NewNotGroupKeyOrAggError(expr string) Error {
	return &err{level: EXCEPTION, ICode: NOT_GROUP_KEY_OR_AGG, IKey: "plan.not_group_key_or_agg",
		InternalMsg: fmt.Sprintf("Expression %s must depend only on group keys or aggregates.", expr), InternalCaller: CallerN(1)}
}

const NEW_INDEX_ALREADY_EXISTS = 4300

func NewIndexAlreadyExistsError(idx string) Error {
	return &err{level: EXCEPTION, ICode: NEW_INDEX_ALREADY_EXISTS,
		IKey:           "plan.new_index_already_exists",
		InternalMsg:    fmt.Sprintf("The index %s already exists.", idx),
		InternalCaller: CallerN(1)}
}

const AMBIGUOUS_META = 4310

func NewAmbiguousMetaError(fn string) Error {
	return &err{level: EXCEPTION, ICode: AMBIGUOUS_META, IKey: "plan.ambiguous_meta",
		InternalMsg: fmt.Sprintf("%s() in query with multiple FROM terms requires an argument.", strings.ToUpper(fn)), InternalCaller: CallerN(1)}
}

const NOT_SUPPORTED_DESC_COLLATION = 4320

func NewIndexerDescCollationError() Error {
	return &err{level: EXCEPTION, ICode: NOT_SUPPORTED_DESC_COLLATION, IKey: "plan.not_supported_desc_collation",
		InternalMsg: fmt.Sprintf("DESC option in the index keys is not supported by indexer."), InternalCaller: CallerN(1)}
}

const PLAN_INTERNAL_ERROR = 4321

func NewPlanInternalError(what string) Error {
	return &err{level: EXCEPTION, ICode: PLAN_INTERNAL_ERROR, IKey: "plan.internal_error",
		InternalMsg: fmt.Sprintf("Plan error: %v", what), InternalCaller: CallerN(1)}
}

const ALTER_INDEX_ERROR = 4322

func NewAlterIndexError() Error {
	return &err{level: EXCEPTION, ICode: ALTER_INDEX_ERROR, IKey: "plan.alter.index.not.supported",
		InternalMsg: fmt.Sprintf("ALTER INDEX not supported"), InternalCaller: CallerN(1)}
}

const NO_ANSI_JOIN = 4330

func NewNoAnsiJoinError(alias, op string) Error {
	return &err{level: EXCEPTION, ICode: NO_ANSI_JOIN, IKey: fmt.Sprintf("plan.ansi_%s.no_index", op),
		InternalMsg: fmt.Sprintf("No index available for ANSI %s term %s", op, alias), InternalCaller: CallerN(1)}
}

const PARTITION_INDEX_NOT_SUPPORTED = 4340

func NewPartitionIndexNotSupportedError() Error {
	return &err{level: EXCEPTION, ICode: PARTITION_INDEX_NOT_SUPPORTED, IKey: "plan.partition_index_not_supported",
		InternalMsg: fmt.Sprintf("PARTITION index is not supported by indexer."), InternalCaller: CallerN(1)}
}

// errors for CBO (cost-based optimizer) starts at 4600

const CBO_ERROR = 4600

func NewCBOError(ikey, what string) Error {
	return &err{level: EXCEPTION, ICode: CBO_ERROR, IKey: ikey,
		InternalMsg: fmt.Sprintf("Error occured during cost-based optimization: %s", what), InternalCaller: CallerN(1)}
}

const INDEX_STAT_ERROR = 4610

func NewIndexStatError(name, what string) Error {
	return &err{level: EXCEPTION, ICode: INDEX_STAT_ERROR, IKey: "optimizer.index_stat_error",
		InternalMsg: fmt.Sprintf("Invalid index statistics for index %s: %s", name, what), InternalCaller: CallerN(1)}
}

// error numbers 4901, 4902, 4903, 4904 and 4905 are retired, and cannot be reused
