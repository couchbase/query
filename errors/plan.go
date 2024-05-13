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

// Plan errors - errors that are created in the prepared, planner and plan packages

func NewPlanError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: E_PLAN, IKey: "plan_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

// for situations where we want to maintain previous error code of 4000 but a proper enclosed error
func NewWrapPlanError(e error) Error {
	var c interface{}
	if er, ok := e.(Error); ok {
		if er.Code() == E_PLAN {
			return er
		}
		c = er.Cause()
	}
	return &err{level: EXCEPTION, ICode: E_PLAN, IKey: "plan_error", ICause: e, cause: c, InternalCaller: CallerN(1)}
}

func IsWrapPlanError(e error, code ErrorCode) bool {
	if er, ok := e.(Error); ok && er.Code() == E_PLAN {
		if cause := er.GetICause(); cause != nil {
			if enclosed, ok := cause.(Error); ok && enclosed.Code() == code {
				return true
			}
		}
	}
	return false
}

func NewReprepareError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_REPREPARE, IKey: "reprepare_error", ICause: e, InternalMsg: "Reprepare error",
		InternalCaller: CallerN(1)}
}

/* error numbers 4010, 4020, 4025 moved to semantics.go */

func NewSubqueryMissingKeysError(keyspace string) Error {
	return &err{level: EXCEPTION, ICode: E_SUBQUERY_MISSING_KEYS, IKey: "plan.build_select.subquery_missing_keys",
		InternalMsg:    fmt.Sprintf("FROM in correlated subquery must have USE KEYS clause: FROM %s.", keyspace),
		InternalCaller: CallerN(1)}
}

func NewSubqueryMissingIndexError(keyspace string) Error {
	return &err{level: EXCEPTION, ICode: E_SUBQUERY_MISSING_INDEX, IKey: "plan.build_select.subquery_missing_index",
		InternalMsg:    fmt.Sprintf("No secondary index available for keyspace %s in correlated subquery.", keyspace),
		InternalCaller: CallerN(1)}
}

func NewNoSuchPreparedError(name string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_SUCH_PREPARED, IKey: "plan.build_prepared.no_such_name",
		InternalMsg: fmt.Sprintf("No such prepared statement: %s", name), InternalCaller: CallerN(1)}
}

func NewNoSuchPreparedWithContextError(name string, queryContext string) Error {
	if queryContext == "" {
		queryContext = "unset"
	}
	return &err{level: EXCEPTION, ICode: E_NO_SUCH_PREPARED, IKey: "plan.build_prepared.no_such_name",
		InternalMsg: fmt.Sprintf("No such prepared statement: %s, context: %s", name, queryContext), InternalCaller: CallerN(1)}
}

func NewUnrecognizedPreparedError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_UNRECOGNIZED_PREPARED, IKey: "plan.build_prepared.unrecognized_prepared",
		ICause: fmt.Errorf("JSON unmarshalling error: %v", e), InternalMsg: "Unrecognizable prepared statement",
		InternalCaller: CallerN(1)}
}

func NewPreparedNameError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_PREPARED_NAME, IKey: "plan.build_prepared.no_such_name",
		InternalMsg: fmt.Sprintf("Unable to add name: %s", msg), InternalCaller: CallerN(1)}
}

func NewPreparedDecodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_PREPARED_DECODING, IKey: "plan.build_prepared.decoding",
		ICause: e, InternalMsg: "Unable to decode prepared statement", InternalCaller: CallerN(1)}
}

func NewPreparedEncodingMismatchError(name string) Error {
	return &err{level: EXCEPTION, ICode: E_PREPARED_ENCODING_MISMATCH, IKey: "plan.build_prepared.name_encoded_plan_mismatch",
		InternalMsg: fmt.Sprintf("Encoded plan parameter does not match encoded plan of %s", name), InternalCaller: CallerN(1)}
}

func NewEncodingNameMismatchError(expected, found string) Error {
	return &err{level: EXCEPTION, ICode: E_ENCODING_NAME_MISMATCH, IKey: "plan.build_prepared.name_not_in_encoded_plan",
		InternalMsg:    fmt.Sprintf("Mismatching name in encoded plan, expecting: %s, found: %s", expected, found),
		InternalCaller: CallerN(1)}
}

func NewEncodingContextMismatchError(name, expected, found string) Error {
	if expected == "" {
		expected = "unset"
	}
	if found == "" {
		found = "unset"
	}
	return &err{level: EXCEPTION, ICode: E_ENCODING_CONTEXT_MISMATCH, IKey: "plan.build_prepared.context_not_in_encoded_plan",
		InternalMsg:    fmt.Sprintf("Mismatching query_context in encoded plan, expecting: %s, found: %s", expected, found),
		InternalCaller: CallerN(1)}
}

func NewPredefinedPreparedNameError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_PREDEFINED_PREPARED_NAME, IKey: "plan.build_prepared.reserved",
		InternalMsg: fmt.Sprintf("Prepared name %s is predefined (reserved). ", msg), InternalCaller: CallerN(1)}
}

func NewNoIndexJoinError(alias, op string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_INDEX_JOIN, IKey: fmt.Sprintf("plan.index_%s.no_index", op),
		InternalMsg: fmt.Sprintf("No index available for join term %s", alias), InternalCaller: CallerN(1)}
}

/* error number 4110 moved to semantics.go */

func NewNoPrimaryIndexError(alias string) Error {
	c := make(map[string]interface{})
	c["user_action"] = "Verify the Index service is present and running."
	return &err{level: EXCEPTION, ICode: E_NO_PRIMARY_INDEX, IKey: "plan.build_primary_index.no_index",
		InternalMsg: fmt.Sprintf("No index available on keyspace %s that matches your query. Use CREATE PRIMARY INDEX ON "+
			"%s to create a primary index, or check that your expected index is online.", alias, alias),
		cause: c, InternalCaller: CallerN(1)}
}

func NewNoIndexServiceError() Error {
	return &err{level: EXCEPTION, ICode: E_NO_INDEX_SERVICE, IKey: "plan.build_primary_index.no_index_service",
		InternalMsg: "Index service not available.", InternalCaller: CallerN(1)}
}

func NewPrimaryIndexOfflineError(name string) Error {
	return &err{level: EXCEPTION, ICode: E_PRIMARY_INDEX_OFFLINE, IKey: "plan.build_primary_index.index_offline",
		InternalMsg: fmt.Sprintf("Primary index %s not online.", name), InternalCaller: CallerN(1)}
}

func NewListSubqueryError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_LIST_SUBQUERIES, IKey: "plan.stmt.list_subqueries",
		ICause: e, InternalMsg: "Error listing subqueries.", InternalCaller: CallerN(1)}
}

func NewNotGroupKeyOrAggError(expr string) Error {
	return &err{level: EXCEPTION, ICode: E_NOT_GROUP_KEY_OR_AGG, IKey: "plan.not_group_key_or_agg",
		InternalMsg: fmt.Sprintf("Expression %s must depend only on group keys or aggregates.", expr), InternalCaller: CallerN(1)}
}

func NewIndexAlreadyExistsError(idx string) Error {
	c := make(map[string]interface{})
	c["name"] = idx
	return &err{level: EXCEPTION, ICode: E_INDEX_ALREADY_EXISTS,
		IKey: "plan.new_index_already_exists", cause: c,
		InternalMsg:    fmt.Sprintf("The index %s already exists.", idx),
		InternalCaller: CallerN(1)}
}

func NewAmbiguousMetaError(fn string, ctx string) Error {
	return &err{level: EXCEPTION, ICode: E_AMBIGUOUS_META, IKey: "plan.ambiguous_meta", InternalCaller: CallerN(1),
		InternalMsg: fmt.Sprintf("%s() in query with multiple FROM terms requires an argument%s.", strings.ToUpper(fn), ctx)}
}

func NewIndexerDescCollationError() Error {
	return &err{level: EXCEPTION, ICode: E_INDEXER_DESC_COLLATION, IKey: "plan.not_supported_desc_collation",
		InternalMsg: fmt.Sprintf("DESC option in the index keys is not supported by indexer."), InternalCaller: CallerN(1)}
}

func NewPlanInternalError(what string) Error {
	return &err{level: EXCEPTION, ICode: E_PLAN_INTERNAL, IKey: "plan.internal_error",
		InternalMsg: fmt.Sprintf("Plan error: %v", what), InternalCaller: CallerN(1)}
}

func NewAlterIndexError() Error {
	return &err{level: EXCEPTION, ICode: E_ALTER_INDEX, IKey: "plan.alter.index.not.supported",
		InternalMsg: fmt.Sprintf("ALTER INDEX not supported"), InternalCaller: CallerN(1)}
}

func NewNoAnsiJoinError(alias, op string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_ANSI_JOIN, IKey: fmt.Sprintf("plan.ansi_%s.no_index", op),
		InternalMsg: fmt.Sprintf("No index available for ANSI %s term %s", op, alias), InternalCaller: CallerN(1)}
}

func NewPartitionIndexNotSupportedError() Error {
	return &err{level: EXCEPTION, ICode: E_PARTITION_INDEX_NOT_SUPPORTED, IKey: "plan.partition_index_not_supported",
		InternalMsg: fmt.Sprintf("PARTITION index is not supported by indexer."), InternalCaller: CallerN(1)}
}

// errors for CBO (cost-based optimizer) starts at 4600

func NewCBOError(ikey, what string) Error {
	return &err{level: EXCEPTION, ICode: E_CBO, IKey: ikey,
		InternalMsg: fmt.Sprintf("Error occured during cost-based optimization: %s", what), InternalCaller: CallerN(1)}
}

func NewIndexStatError(name, what string) Error {
	return &err{level: EXCEPTION, ICode: E_INDEX_STAT, IKey: "optimizer.index_stat_error",
		InternalMsg: fmt.Sprintf("Invalid index statistics for index %s: %s", name, what), InternalCaller: CallerN(1)}
}

func NewPlanNoPlaceholderError() Error {
	return &err{level: EXCEPTION, ICode: E_PLAN_NO_PLACEHOLDER, IKey: "plan.no_placeholder",
		InternalMsg: "Placeholder is not allowed in keyspace", InternalCaller: CallerN(1)}
}

// error numbers 4901, 4902, 4903, 4904 and 4905 are retired, and cannot be reused
