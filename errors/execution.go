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

// Execution errors - errors that are created in the execution package

func NewExecutionPanicError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_EXECUTION_PANIC, IKey: "execution.panic", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewExecutionInternalError(what string) Error {
	return &err{level: EXCEPTION, ICode: E_EXECUTION_INTERNAL, IKey: "execution.internal_error",
		InternalMsg: fmt.Sprintf("Execution internal error: %v", what), InternalCaller: CallerN(1)}
}

func NewExecutionParameterError(what string) Error {
	return &err{level: EXCEPTION, ICode: E_EXECUTION_PARAMETER, IKey: "execution.parameter_error",
		InternalMsg: fmt.Sprintf("Execution parameter error: %v", what), InternalCaller: CallerN(1)}
}

func NewParsingError(e error, ctx string) Error {
	return &err{level: EXCEPTION, ICode: E_PARSING, IKey: "execution.expression.parse.failed",
		ICause:         e,
		InternalMsg:    fmt.Sprintf("Expression parsing%s failed.", ctx),
		InternalCaller: CallerN(1)}
}

func NewEvaluationError(e error, termType string) Error {
	if _, ok := e.(*AbortError); ok {
		return &err{level: EXCEPTION, ICode: E_EVALUATION, IKey: "execution.abort_error", ICause: e,
			InternalMsg: fmt.Sprintf("Abort: %s.", e), InternalCaller: CallerN(1)}
	} else if ee, ok := e.(Error); ok {
		return &err{level: EXCEPTION, ICode: E_EVALUATION_ABORT, IKey: "execution.evaluation_error", cause: ee,
			InternalMsg: fmt.Sprintf("Error evaluating %s", termType), InternalCaller: CallerN(1)}
	}
	return &err{level: EXCEPTION, ICode: E_EVALUATION_ABORT, IKey: "execution.evaluation_error", ICause: e,
		InternalMsg: fmt.Sprintf("Error evaluating %s", termType), InternalCaller: CallerN(1)}
}

func NewExplainError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_EXPLAIN, IKey: "execution.explain_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewGroupUpdateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_GROUP_UPDATE, IKey: "execution.group_update_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewInvalidValueError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_VALUE, IKey: "execution.invalid_value_error",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewRangeError(termType string) Error {
	return &err{level: EXCEPTION, ICode: E_RANGE, IKey: "execution.range_error",
		InternalMsg: fmt.Sprintf("Out of range evaluating %s.", termType), InternalCaller: CallerN(1)}
}

func NewDivideByZeroWarning() Error {
	return &err{level: WARNING, ICode: W_DIVIDE_BY_ZERO, IKey: "execution.divide_by_zero",
		InternalMsg: "Division by 0.", InternalCaller: CallerN(1)}
}

func NewDuplicateFinalGroupError() Error {
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_FINAL_GROUP, IKey: "execution.duplicate_final_group",
		InternalMsg: "Duplicate Final Group.", InternalCaller: CallerN(1)}
}

func NewInsertKeyError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_INSERT_KEY, IKey: "execution.insert_key_error",
		InternalMsg: fmt.Sprintf("No INSERT key for %v", v), InternalCaller: CallerN(1)}
}

func NewInsertValueError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_INSERT_VALUE, IKey: "execution.insert_value_error",
		InternalMsg: fmt.Sprintf("No INSERT value for %v", v), InternalCaller: CallerN(1)}
}

func NewInsertKeyTypeError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_INSERT_KEY_TYPE, IKey: "execution.insert_key_type_error",
		InternalMsg:    fmt.Sprintf("Cannot INSERT non-string key %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewInsertOptionsTypeError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_INSERT_OPTIONS_TYPE, IKey: "execution.insert_options_type_error",
		InternalMsg:    fmt.Sprintf("Cannot INSERT non-OBJECT options %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewUpsertKeyError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_UPSERT_KEY, IKey: "execution.upsert_key_error",
		InternalMsg: fmt.Sprintf("No UPSERT key for %v", v), InternalCaller: CallerN(1)}
}

func NewUpsertValueError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_UPSERT_VALUE, IKey: "execution.upsert_value_error",
		InternalMsg: fmt.Sprintf("No UPSERT value for %v", v), InternalCaller: CallerN(1)}
}

func NewUpsertKeyTypeError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_UPSERT_KEY_TYPE, IKey: "execution.upsert_key_type_error",
		InternalMsg:    fmt.Sprintf("Cannot UPSERT non-string key %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewUpsertOptionsTypeError(v interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_UPSERT_OPTIONS_TYPE, IKey: "execution.upsert_options_type_error",
		InternalMsg:    fmt.Sprintf("Cannot UPSERT non-OBJECT options %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewDeleteAliasMissingError(alias string) Error {
	return &err{level: EXCEPTION, ICode: E_DELETE_ALIAS_MISSING, IKey: "execution.missing_delete_alias",
		InternalMsg:    fmt.Sprintf("DELETE alias %s not found in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewDeleteAliasMetadataError(alias string) Error {
	return &err{level: EXCEPTION, ICode: E_DELETE_ALIAS_METADATA, IKey: "execution.delete_alias_metadata",
		InternalMsg:    fmt.Sprintf("DELETE alias %s has no metadata in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateAliasMissingError(alias string) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_ALIAS_MISSING, IKey: "execution.missing_update_alias",
		InternalMsg:    fmt.Sprintf("UPDATE alias %s not found in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateAliasMetadataError(alias string) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_ALIAS_METADATA, IKey: "execution.update_alias_metadata",
		InternalMsg:    fmt.Sprintf("UPDATE alias %s has no metadata in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateMissingClone() Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_MISSING_CLONE, IKey: "execution.update_missing_clone",
		InternalMsg: "Missing UPDATE clone.", InternalCaller: CallerN(1)}
}

func NewUnnestInvalidPosition(pos interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_UNNEST_INVALID_POSITION, IKey: "execution.unnest_invalid_position",
		InternalMsg: fmt.Sprintf("Invalid UNNEST position of type %T.", pos), InternalCaller: CallerN(1)}
}

func NewScanVectorTooManyScannedBuckets(buckets []string) Error {
	return &err{level: EXCEPTION, ICode: E_SCAN_VECTOR_TOO_MANY_SCANNED_BUCKETS, IKey: "execution.scan_vector_too_many_scanned_vectors",
		InternalMsg: fmt.Sprintf("The scan_vector parameter should not be used for queries accessing more than one keyspace. "+
			"Use scan_vectors instead. Keyspaces: %v", buckets), InternalCaller: CallerN(1)}
}

// Error code 5200 is retired. Do not reuse.

func NewUserNotFoundError(u string) Error {
	return &err{level: EXCEPTION, ICode: E_USER_NOT_FOUND, IKey: "execution.user_not_found",
		InternalMsg: fmt.Sprintf("Unable to find user %s.", u), InternalCaller: CallerN(1)}
}

func NewRoleRequiresKeyspaceError(role string) Error {
	return &err{level: EXCEPTION, ICode: E_ROLE_REQUIRES_KEYSPACE, IKey: "execution.role_requires_keyspace",
		InternalMsg: fmt.Sprintf("Role %s requires a keyspace.", role), InternalCaller: CallerN(1)}
}

func NewRoleTakesNoKeyspaceError(role string) Error {
	return &err{level: EXCEPTION, ICode: E_ROLE_TAKES_NO_KEYSPACE, IKey: "execution.role_takes_no_keyspace",
		InternalMsg: fmt.Sprintf("Role %s does not take a keyspace.", role), InternalCaller: CallerN(1)}
}

func NewNoSuchKeyspaceError(bucket string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_SUCH_KEYSPACE, IKey: "execution.no_such_keyspace",
		InternalMsg: fmt.Sprintf("Keyspace %s is not valid.", bucket), InternalCaller: CallerN(1)}
}

func NewNoSuchScopeError(scope string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_SUCH_SCOPE, IKey: "execution.no_such_scope",
		InternalMsg: fmt.Sprintf("Scope %s is not valid.", scope), InternalCaller: CallerN(1)}
}

func NewNoSuchBucketError(bucket string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_SUCH_BUCKET, IKey: "execution.no_such_bucket",
		InternalMsg: fmt.Sprintf("Bucket %s is not valid.", bucket), InternalCaller: CallerN(1)}
}

func NewRoleNotFoundError(role string) Error {
	return &err{level: EXCEPTION, ICode: E_ROLE_NOT_FOUND, IKey: "execution.role_not_found",
		InternalMsg: fmt.Sprintf("Role %s is not valid.", role), InternalCaller: CallerN(1)}
}

func NewRoleAlreadyPresent(user string, role string, bucket string) Error {
	var msg string
	if bucket == "" {
		msg = fmt.Sprintf("User %s already has role %s.", user, role)
	} else {
		msg = fmt.Sprintf("User %s already has role %s(%s).", user, role, bucket)
	}
	return &err{level: WARNING, ICode: E_ROLE_ALREADY_PRESENT, IKey: "execution.role_already_present",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewRoleNotPresent(user string, role string, bucket string) Error {
	var msg string
	if bucket == "" {
		msg = fmt.Sprintf("User %s did not have role %s.", user, role)
	} else {
		msg = fmt.Sprintf("User %s did not have role %s(%s).", user, role, bucket)
	}
	return &err{level: WARNING, ICode: E_ROLE_NOT_PRESENT, IKey: "execution.role_not_present",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewUserWithNoRoles(user string) Error {
	return &err{level: WARNING, ICode: E_USER_WITH_NO_ROLES, IKey: "execution.user_with_no_roles",
		InternalMsg:    fmt.Sprintf("User %s has no roles. Connecting with this user may not be possible", user),
		InternalCaller: CallerN(1)}
}

// Error code 5290 is retired. Do not reuse.

func NewHashTablePutError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_HASH_TABLE_PUT, IKey: "execution.hash_table_put_error", ICause: e,
		InternalMsg:    fmt.Sprintf("Hash Table Put failed"),
		InternalCaller: CallerN(1)}
}

func NewHashTableGetError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_HASH_TABLE_GET, IKey: "execution.hash_table_get_error", ICause: e,
		InternalMsg:    fmt.Sprintf("Hash Table Get failed"),
		InternalCaller: CallerN(1)}
}

func NewMergeMultiUpdateError(key string) Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_MULTI_UPDATE, IKey: "execution.merge_multiple_update",
		InternalMsg:    fmt.Sprintf("Multiple UPDATE/DELETE of the same document (document key '%s') in a MERGE statement", key),
		InternalCaller: CallerN(1)}
}

func NewMergeMultiInsertError(key string) Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_MULTI_INSERT, IKey: "execution.merge_multiple_insert",
		InternalMsg:    fmt.Sprintf("Multiple INSERT of the same document (document key '%s') in a MERGE statement", key),
		InternalCaller: CallerN(1)}
}

func NewWindowEvaluationError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_WINDOW_EVALUATION, IKey: "execution.window_aggregate_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewAdviseIndexError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADVISE_INDEX, IKey: "execution.advise_index_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewUpdateStatisticsError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_STATISTICS, IKey: "execution.update_statistics",
		InternalMsg:    msg,
		InternalCaller: CallerN(1)}
}

func NewSubqueryBuildError(e error) Error {
	if er, ok := e.(Error); ok && er.Code() == E_SUBQUERY_BUILD {
		return er
	}
	return &err{level: EXCEPTION, ICode: E_SUBQUERY_BUILD, IKey: "execution.subquery.build", ICause: e,
		InternalMsg:    "Unable to run subquery",
		InternalCaller: CallerN(1)}
}

func NewIndexLeadingKeyMissingNotSupportedError() Error {
	return &err{level: EXCEPTION, ICode: E_INDEX_LEADING_KEY_MISSING_NOT_SUPPORTED, IKey: "execution.indexing.leadingkey_missing_not_supported",
		InternalMsg:    fmt.Sprintf("Indexing leading key MISSING values are not supported by indexer."),
		InternalCaller: CallerN(1)}
}

func NewIndexNotInMemory(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INDEX_NOT_IN_MEMORY, IKey: "execution.update_statistics.index_not_in_memory",
		InternalMsg:    msg,
		InternalCaller: CallerN(1)}
}

func NewMissingSystemCBOStatsError() Error {
	return &err{level: EXCEPTION, ICode: E_MISSING_SYSTEMCBO_STATS, IKey: "execution.update_statistics.missing_system_cbostats",
		InternalMsg:    "System Collection 'N1QL_CBO_STATS' is required for UPDATE STATISTICS (ANALYZE)",
		InternalCaller: CallerN(1)}
}

func NewInvalidIndexNameError(name interface{}, ikey string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_INDEX_NAME, IKey: ikey,
		InternalMsg:    fmt.Sprintf("index name(%v) must be a string", name),
		InternalCaller: CallerN(1)}
}

func NewIndexNotFoundError(name string, ikey string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_INDEX_NOT_FOUND, IKey: ikey, ICause: e,
		InternalMsg:    fmt.Sprintf("index %s is not found", name),
		InternalCaller: CallerN(1)}
}

func NewIndexUpdStatsError(names, msg string, e error) Error {
	c := make(map[string]interface{})
	c["index_names"] = names
	if e != nil {
		c["cause"] = e
	}
	return &err{level: EXCEPTION, ICode: E_INDEX_UPD_STATS, IKey: "execution.index.upd_stats",
		ICause: e, cause: c,
		InternalMsg:    fmt.Sprintf("Error with UPDATE STATISTICS for indexes (%s): %s", names, msg),
		InternalCaller: CallerN(1)}
}

func NewTimeParseError(str string, e error) Error {
	c := make(map[string]interface{})
	c["time_string"] = str
	if e != nil {
		c["cause"] = e
	}
	return &err{level: EXCEPTION, ICode: E_TIME_PARSE, IKey: "execution.upd_stats.time_parse",
		ICause: e, cause: c,
		InternalMsg:    fmt.Sprintf("Error parsing time string %s", str),
		InternalCaller: CallerN(1)}
}

func NewNLInnerPrimaryDocsExceeded(alias string, limit int) Error {
	c := make(map[string]interface{})
	c["keyspace_alias"] = alias
	c["limit"] = limit
	return &err{level: EXCEPTION, ICode: E_JOIN_ON_PRIMARY_DOCS_EXCEEDED, IKey: "execution.nljoin_inner_primary.docs_exceeded",
		cause:          c,
		InternalMsg:    fmt.Sprintf("Inner of nested-loop join (%s) cannot have more than %d documents without appropriate secondary index", alias, limit),
		InternalCaller: CallerN(1)}
}

func NewMemoryQuotaExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_MEMORY_QUOTA_EXCEEDED, IKey: "execution.memory_quota.exceeded",
		InternalMsg:    "Request has exceeded memory quota",
		InternalCaller: CallerN(1)}
}

func NewNodeQuotaExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_NODE_QUOTA_EXCEEDED, IKey: "execution.node_quota.exceeded",
		InternalMsg:    "Query node has run out of memory",
		InternalCaller: CallerN(1)}
}

func NewTenantQuotaExceededError(t string, u string) Error {
	var msg string
	var c interface{}
	if t != "" {
		mc := make(map[string]interface{}, 1)
		mc["tenant"] = t
		c = mc
		msg = fmt.Sprintf("Tenant %v has run out of memory", t)
	} else {
		msg = fmt.Sprintf("User %v has run out of memory", u)
	}
	return &err{level: EXCEPTION, ICode: E_TENANT_QUOTA_EXCEEDED, IKey: "execution.tenant_quota.exceeded",
		InternalMsg: msg, cause: c,
		InternalCaller: CallerN(1)}
}

func NewNilEvaluateParamError(param string) Error {
	return &err{level: EXCEPTION, ICode: E_NIL_EVALUATE_PARAM, IKey: "execution.evaluate.nil.param",
		InternalMsg:    fmt.Sprintf("nil '%s' parameter for evaluation", param),
		InternalCaller: CallerN(1)}
}

func NewMissingKeysWarning(count int, ks string, keys ...interface{}) Error {
	c := make(map[string]interface{})
	c["num_missing_keys"] = count
	c["keyspace"] = ks
	c["keys"] = keys
	return &err{level: WARNING, ICode: W_MISSING_KEY, IKey: "execution.use.missing_key",
		InternalMsg: "Key(s) in USE KEYS hint not found", cause: c,
		InternalCaller: CallerN(1)}
}

var _ve = map[ErrorCode][2]string{
	E_VALUE_RECONSTRUCT:  {"reconstruct", "Failed to reconstruct value"},
	E_VALUE_INVALID:      {"invalid", "Invalid reconstructed value"},
	E_VALUE_SPILL_CREATE: {"spill.create", "Failed to create spill file"},
	E_VALUE_SPILL_READ:   {"spill.read", "Failed to read from spill file"},
	E_VALUE_SPILL_WRITE:  {"spill.write", "Failed to write to spill file"},
	E_VALUE_SPILL_SIZE:   {"spill.size", "Failed to determine spill file size"},
	E_VALUE_SPILL_SEEK:   {"spill.seek", "Failed to seek in spill file"},
}

func NewValueError(code ErrorCode, args ...interface{}) Error {
	e := &err{level: EXCEPTION, ICode: code, InternalCaller: CallerN(1),
		IKey: "value." + _ve[code][0], InternalMsg: _ve[code][1]}
	var fmtArgs []interface{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			fmtArgs = append(fmtArgs, a)
		case Error:
			e.cause = a
		case error:
			e.cause = a
		default:
			panic("invalid arguments to NewValueError")
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}
