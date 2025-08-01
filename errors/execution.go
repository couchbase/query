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

func NewExecutionKeyValidationSpaceError() Error {
	return &err{level: EXCEPTION, ICode: E_EXECUTION_KEY_VALIDATION, IKey: "execution.key_validation",
		InternalMsg: "Out of key validation space.", InternalCaller: CallerN(1)}
}

func NewExecutionStatementStoppedError(statement string) Error {
	c := make(map[string]interface{})
	c["statement"] = statement
	return &err{level: EXCEPTION, ICode: E_EXECUTION_STATEMENT_STOPPED, IKey: "execution.stopped_error",
		InternalMsg: "Execution of statement has been stopped.", InternalCaller: CallerN(1), cause: c}
}

func NewParsingError(e error, ctx string) Error {
	return &err{level: EXCEPTION, ICode: E_PARSING, IKey: "execution.expression.parse.failed",
		ICause:         e,
		InternalMsg:    fmt.Sprintf("'%s' is not a valid expression.", ctx),
		InternalCaller: CallerN(1)}
}

func NewEvaluationError(e error, termType string) Error {
	if _, ok := e.(*AbortError); ok {
		return &err{level: EXCEPTION, ICode: E_EVALUATION_ABORT, IKey: "execution.abort_error", ICause: e,
			InternalMsg: fmt.Sprintf("Abort: %s.", e), InternalCaller: CallerN(1)}
	} else if ee, ok := e.(Error); ok {
		if ee.Level() == WARNING {
			return ee
		}
		return &err{level: EXCEPTION, ICode: E_EVALUATION, IKey: "execution.evaluation_error", cause: ee,
			InternalMsg: fmt.Sprintf("Error evaluating %s", termType), InternalCaller: CallerN(1)}
	}
	return &err{level: EXCEPTION, ICode: E_EVALUATION, IKey: "execution.evaluation_error", ICause: e,
		InternalMsg: fmt.Sprintf("Error evaluating %s", termType), InternalCaller: CallerN(1)}
}

func NewEvaluationWithCauseError(e error, termType string) Error {
	if _, ok := e.(*AbortError); ok {
		return &err{level: EXCEPTION, ICode: E_EVALUATION_ABORT, IKey: "execution.abort_error", ICause: e,
			InternalMsg: fmt.Sprintf("Abort: %s.", e), InternalCaller: CallerN(1)}
	} else if ee, ok := e.(Error); ok {
		return &err{level: EXCEPTION, ICode: E_EVALUATION, IKey: "execution.evaluation_error", cause: ee,
			InternalMsg: fmt.Sprintf("Error evaluating %s", termType), InternalCaller: CallerN(1)}
	}
	var c map[string]interface{}
	if e != nil {
		c = make(map[string]interface{})
		c["cause"] = e.Error()
	}
	return &err{level: EXCEPTION, ICode: E_EVALUATION, IKey: "execution.evaluation_error", cause: c,
		InternalMsg: fmt.Sprintf("Error evaluating %s", termType), InternalCaller: CallerN(1)}
}

var _de = map[ErrorCode]string{
	W_DATE:                     "",
	W_DATE_OVERFLOW:            ": Overflow",
	W_DATE_INVALID_FORMAT:      ": Invalid format",
	W_DATE_INVALID_DATE_STRING: ": Invalid date string",
	W_DATE_PARSE_FAILED:        ": Failed to parse",
	W_DATE_INVALID_COMPONENT:   ": Invalid component",
	W_DATE_NON_INT_VALUE:       ": Value is not an integer",
	W_DATE_INVALID_ARGUMENT:    ": Invalid argument",
	W_DATE_INVALID_TIMEZONE:    ": Invalid time zone",
}

func NewDateWarning(e ErrorCode, info interface{}) Error {
	var c interface{}
	if info != nil {
		cm := make(map[string]interface{})
		switch e := info.(type) {
		case map[string]interface{}:
			for k, v := range e {
				cm[k] = v
			}
		case Error:
			cm["error"] = e
		case interface{ Error() string }:
			cm["error"] = e.Error()
		default:
			cm["details"] = e
		}
		cm["caller"] = CallerN(1)
		c = cm
	}
	msg := "Date error"
	if m, ok := _de[e]; ok {
		if m != "" {
			msg += m
		}
	} else {
		panic("BUG: invalid date warning")
	}
	return &err{level: WARNING, ICode: e, IKey: "execution.date_error", cause: c,
		InternalMsg: msg, InternalCaller: CallerN(1), onceOnly: true}
}

func NewExplainError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_EXPLAIN, IKey: "execution.explain_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewExplainFunctionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_EXPLAIN_FUNCTION, IKey: "execution.explain_function_error", ICause: e,
		InternalMsg: fmt.Sprintf("EXPLAIN FUNCTION: %s", msg), InternalCaller: CallerN(1)}
}

func NewGroupUpdateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_GROUP_UPDATE, IKey: "execution.group_update_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewInvalidValueError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_VALUE, IKey: "execution.invalid_value_error",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewInvalidExpressionError(exp string, details interface{}) Error {
	c := make(map[string]interface{})
	c["expression"] = exp
	if details != nil {
		c["details"] = details
	}
	return &err{level: EXCEPTION, ICode: E_INVALID_EXPRESSION, IKey: "execution.invalid_expression", cause: c,
		InternalMsg: "Invalid expression", InternalCaller: CallerN(1)}
}

func NewUnsupportedExpressionError(exp string, details interface{}) Error {
	c := make(map[string]interface{})
	c["expression"] = exp
	if details != nil {
		c["details"] = details
	}
	return &err{level: EXCEPTION, ICode: E_UNSUPPORTED_EXPRESSION, IKey: "execution.unsupported_expression", cause: c,
		InternalMsg: "Unsupported expression", InternalCaller: CallerN(1)}
}

func NewRangeError(termType string) Error {
	return &err{level: EXCEPTION, ICode: E_RANGE, IKey: "execution.range_error",
		InternalMsg: fmt.Sprintf("Out of range evaluating %s.", termType), InternalCaller: CallerN(1)}
}

func NewSizeError(termType string, elemSize uint64, nelem int, size uint64, limit uint64) Error {
	c := make(map[string]interface{})
	c["limit"] = limit
	c["size"] = size
	c["element_size"] = elemSize
	c["element_count"] = nelem
	c["term_type"] = termType
	return &err{level: EXCEPTION, ICode: E_SIZE, IKey: "execution.size_error", cause: c,
		InternalMsg: fmt.Sprintf("Size of %s result exceeds limit (%v > %v).", termType, size, limit), InternalCaller: CallerN(1)}
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

func NewUpsertKeyAlreadyMutatedError(ks string, key string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = ks
	c["key"] = key
	return &err{level: EXCEPTION, ICode: E_UPSERT_KEY_ALREADY_MUTATED, IKey: "execution.upsert_key_already_mutated",
		InternalMsg: "Cannot act on the same key multiple times in an UPSERT statement.", cause: c,
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

// restricted field updates in system keyspace
func NewUpdateInvalidField(key string, field string) Error {
	c := make(map[string]interface{})
	c["key"] = key
	c["field"] = field
	return &err{level: EXCEPTION, ICode: E_UPDATE_INVALID_FIELD, IKey: "execution.update_invalid_field",
		InternalMsg: "Invalid field update.", cause: c, InternalCaller: CallerN(1)}
}

func NewUnnestInvalidPosition(pos interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_UNNEST_INVALID_POSITION, IKey: "execution.unnest_invalid_position",
		InternalMsg: fmt.Sprintf("Invalid UNNEST position of type %T.", pos), InternalCaller: CallerN(1)}
}

func NewScanVectorTooManyScannedBuckets(buckets []string) Error {
	return &err{level: EXCEPTION, ICode: E_SCAN_VECTOR_TOO_MANY_SCANNED_BUCKETS,
		IKey: "execution.scan_vector_too_many_scanned_vectors",
		InternalMsg: fmt.Sprintf("The scan_vector parameter should not be used for queries accessing more than one keyspace. "+
			"Use scan_vectors instead. Keyspaces: %v", buckets), InternalCaller: CallerN(1)}
}

// Error code 5200 is retired. Do not reuse.

func NewUserExistsError(u string) Error {
	c := make(map[string]interface{})
	c["user"] = u
	return &err{level: EXCEPTION, ICode: E_USER_EXISTS, IKey: "execution.user_exists", cause: c,
		InternalMsg: fmt.Sprintf("User %s already exists.", u), InternalCaller: CallerN(1)}
}

func NewUserNotFoundError(u string) Error {
	c := make(map[string]interface{})
	c["user"] = u
	return &err{level: EXCEPTION, ICode: E_USER_NOT_FOUND, IKey: "execution.user_not_found", cause: c,
		InternalMsg: fmt.Sprintf("Unable to find user %s.", u), InternalCaller: CallerN(1)}
}

func NewUserAttributeError(d string, a string, r string) Error {
	c := make(map[string]interface{})
	c["domain"] = d
	c["attribute"] = a
	c["reason"] = r
	return &err{level: EXCEPTION, ICode: E_USER_ATTRIBUTE, IKey: "execution.user_attribute",
		cause: c, InternalMsg: fmt.Sprintf("Attribute '%s' %s for %s users.", a, r, d), InternalCaller: CallerN(1)}
}

func NewGroupExistsError(g string) Error {
	c := make(map[string]interface{})
	c["group"] = g
	return &err{level: EXCEPTION, ICode: E_GROUP_EXISTS, IKey: "execution.group_exists", cause: c,
		InternalMsg: fmt.Sprintf("Group %s already exists.", g), InternalCaller: CallerN(1)}
}

func NewGroupNotFoundError(g string) Error {
	c := make(map[string]interface{})
	c["group"] = g
	return &err{level: EXCEPTION, ICode: E_GROUP_NOT_FOUND, IKey: "execution.group_not_found", cause: c,
		InternalMsg: fmt.Sprintf("Unable to find group %s.", g), InternalCaller: CallerN(1)}
}

func NewGroupAttributeError(a string, r string) Error {
	c := make(map[string]interface{})
	c["attribute"] = a
	c["reason"] = r
	return &err{level: EXCEPTION, ICode: E_GROUP_ATTRIBUTE, IKey: "execution.group_attribute",
		cause: c, InternalMsg: fmt.Sprintf("Attribute '%s' %s for groups.", a, r), InternalCaller: CallerN(1)}
}

func NewMissingAttributesError(what string) Error {
	c := make(map[string]interface{})
	c["entity"] = what
	return &err{level: EXCEPTION, ICode: E_MISSING_ATTRIBUTES, IKey: "execution.missing_attributes",
		cause: c, InternalMsg: fmt.Sprintf("Missing attributes for %s.", what), InternalCaller: CallerN(1)}
}

func NewRoleRequiresKeyspaceError(role string) Error {
	c := make(map[string]interface{})
	c["role"] = role
	return &err{level: EXCEPTION, ICode: E_ROLE_REQUIRES_KEYSPACE, IKey: "execution.role_requires_keyspace", cause: c,
		InternalMsg: fmt.Sprintf("Role %s requires a keyspace.", role), InternalCaller: CallerN(1)}
}

func NewRoleIncorrectLevelError(role string, level string) Error {
	c := make(map[string]interface{})
	c["role"] = role
	return &err{level: EXCEPTION, ICode: E_ROLE_INCORRECT_LEVEL, IKey: "execution:role_incorrect_level", cause: c,
		InternalMsg: fmt.Sprintf("Role %s cannot be specified at the %s level.", role, level), InternalCaller: CallerN(1)}
}

func NewRoleTakesNoKeyspaceError(role string) Error {
	c := make(map[string]interface{})
	c["role"] = role
	return &err{level: EXCEPTION, ICode: E_ROLE_TAKES_NO_KEYSPACE, IKey: "execution.role_takes_no_keyspace", cause: c,
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

func NewRoleAlreadyPresent(what string, id string, role string, target string) Error {
	var msg string
	c := make(map[string]interface{})
	c["role"] = role
	c["id"] = id
	c["what"] = strings.ToLower(what)
	if target == "" {
		msg = fmt.Sprintf("%s %s already has role %s.", what, id, role)
	} else {
		msg = fmt.Sprintf("%s %s already has role %s on %s.", what, id, role, strings.ReplaceAll(target, ":", "."))
		c["path"] = strings.Split("default:"+target, ":")
	}
	return &err{level: WARNING, ICode: W_ROLE_ALREADY_PRESENT, IKey: "execution.role_already_present",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewRoleNotPresent(what string, id string, role string, target string) Error {
	var msg string
	c := make(map[string]interface{})
	c["role"] = role
	c["id"] = id
	c["what"] = strings.ToLower(what)
	if target == "" {
		msg = fmt.Sprintf("%s %s did not have role %s.", what, id, role)
	} else {
		msg = fmt.Sprintf("%s %s did not have role %s on %s.", what, id, role, strings.ReplaceAll(target, ":", "."))
		c["path"] = strings.Split("default:"+target, ":")
	}
	return &err{level: WARNING, ICode: W_ROLE_NOT_PRESENT, IKey: "execution.role_not_present",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewUserWithNoRoles(user string) Error {
	return &err{level: WARNING, ICode: W_USER_WITH_NO_ROLES, IKey: "execution.user_with_no_roles",
		InternalMsg:    fmt.Sprintf("User %s has no roles. Connecting with this user may not be possible", user),
		InternalCaller: CallerN(1)}
}

func NewGroupWithNoRoles(group string) Error {
	return &err{level: WARNING, ICode: W_GROUP_WITH_NO_ROLES, IKey: "execution.group_with_no_roles",
		InternalMsg:    fmt.Sprintf("Group %s has no roles.", group),
		InternalCaller: CallerN(1)}
}

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

func NewUpdateStatisticsError(msg string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_STATISTICS, IKey: "execution.update_statistics", ICause: e,
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
	return &err{level: EXCEPTION, ICode: E_INDEX_LEADING_KEY_MISSING_NOT_SUPPORTED,
		IKey:           "execution.indexing.leadingkey_missing_not_supported",
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
		cause: c,
		InternalMsg: fmt.Sprintf("Inner of nested-loop join (%s) cannot have more than %d documents without appropriate "+
			"secondary index", alias, limit),
		InternalCaller: CallerN(1)}
}

func NewSubqueryNumDocsExceeded(keyspace string, limit int) Error {
	c := make(map[string]interface{})
	c["keyspace"] = keyspace
	c["limit"] = limit
	return &err{level: EXCEPTION,
		ICode: E_SUBQUERY_PRIMARY_DOCS_EXCEEDED,
		IKey:  "execution.corrsubq_primary.docs_exceeded",
		cause: c,
		InternalMsg: fmt.Sprintf("Correlated subquery's keyspace (%s) cannot have more than %d documents"+
			" without appropriate secondary index", keyspace, limit),
		InternalCaller: CallerN(1)}
}

func NewInvalidQueryVector(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_QUERY_VECTOR, IKey: "execution.vector_index.query_vector",
		InternalMsg:    "Invalid parameter (query vector) specified for vector search function: " + msg,
		InternalCaller: CallerN(1)}
}

func NewInvalidProbes(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_PROBES, IKey: "execution.vector_index.probes",
		InternalMsg:    "Invalid parameter (probes) specified for vector search function: " + msg,
		InternalCaller: CallerN(1)}
}

func NewInvalidReRank(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_RERANK, IKey: "execution.vector_index.rerank",
		InternalMsg:    "Invalid parameter (rerank) specified for vector search function: " + msg,
		InternalCaller: CallerN(1)}
}

func NewInvalidTopNScan(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_TOPNSCAN, IKey: "execution.vector_index.topnscan",
		InternalMsg:    "Invalid parameter (TopNScan) specified for vector search function: " + msg,
		InternalCaller: CallerN(1)}
}

func NewMaxHeapSizeExceeded(heapSize, maxHeapSize int, name string) Error {
	return &err{level: EXCEPTION, ICode: E_MAXHEAP_SIZE_EXCEEDED, IKey: "execution.vector_index.maxheap_size",
		InternalMsg: fmt.Sprintf("Total heap size for (Limit + Offset) (%d) exceeded maximum heap size (%d)"+
			" allowed for vector index %s", heapSize, maxHeapSize, name),
		InternalCaller: CallerN(1)}
}

func NewMemoryQuotaExceededError(inuse, limit uint64) Error {
	c := make(map[string]interface{})
	c["caller"] = CallerN(2)
	return &err{level: EXCEPTION, ICode: E_MEMORY_QUOTA_EXCEEDED, IKey: "execution.memory_quota.exceeded",
		InternalMsg: fmt.Sprintf("Request has exceeded memory quota (inuse: %v limit: %v)", inuse, limit),
		cause:       c, InternalCaller: CallerN(1)}
}

func NewNodeQuotaExceededError(curr, limit uint64) Error {
	c := make(map[string]interface{})
	c["caller"] = CallerN(3)
	return &err{level: EXCEPTION, ICode: E_NODE_QUOTA_EXCEEDED, IKey: "execution.node_quota.exceeded",
		InternalMsg: fmt.Sprintf("Query node has run out of memory (curr: %v limit: %v)", curr, limit),
		cause:       c, InternalCaller: CallerN(1)}
}

func NewTenantQuotaExceededError(t string, u string, r, l uint64) Error {
	var msg string
	var c interface{}
	if t != "" {
		mc := make(map[string]interface{}, 1)
		mc["tenant"] = t
		c = mc
		msg = fmt.Sprintf("Tenant %v has run out of memory: requested %v, limit %v", t, r, l)
	} else {
		msg = fmt.Sprintf("User %v has run out of memory: requested %v, limit %v", u, r, l)
	}
	return &err{level: EXCEPTION, ICode: E_TENANT_QUOTA_EXCEEDED, IKey: "execution.tenant_quota.exceeded",
		InternalMsg: msg, cause: c,
		InternalCaller: CallerN(1)}
}

func NewLowMemory(threshold int) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_LOW_MEMORY, IKey: "service.request.halted",
		InternalMsg: fmt.Sprintf("request halted: free memory below %v", threshold) + "% " + "of available memory", InternalCaller: CallerN(1)}
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
	E_VALUE_RECONSTRUCT:     {"reconstruct", "Failed to reconstruct value"},
	E_VALUE_INVALID:         {"invalid", "Invalid reconstructed value"},
	E_VALUE_SPILL_CREATE:    {"spill.create", "Failed to create spill file"},
	E_VALUE_SPILL_READ:      {"spill.read", "Failed to read from spill file"},
	E_VALUE_SPILL_WRITE:     {"spill.write", "Failed to write to spill file"},
	E_VALUE_SPILL_SIZE:      {"spill.size", "Failed to determine spill file size"},
	E_VALUE_SPILL_SEEK:      {"spill.seek", "Failed to seek in spill file"},
	E_VALUE_SPILL_MAX_FILES: {"spill.max_files", "Too many spill files"},
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

func NewCurlExecutionError(e error) Error {
	c := make(map[string]interface{})
	c["error"] = e
	return &err{level: EXCEPTION, ICode: E_EXECUTION_CURL, IKey: "execution.curl",
		InternalMsg: "Error executing CURL function", cause: c, InternalCaller: CallerN(1)}
}

func NewDynamicAuthError(e error) Error {
	c := make(map[string]interface{}, 1)
	c["error"] = e
	return &err{level: EXCEPTION, ICode: E_DYNAMIC_AUTH, IKey: "execution.dynamic_auth",
		InternalMsg: "Dynamic auth error", cause: c, InternalCaller: CallerN(1)}
}

func NewTransactionalAuthError(e error) Error {
	c := make(map[string]interface{}, 1)
	c["error"] = e
	return &err{level: EXCEPTION, ICode: E_TRANSACTIONAL_AUTH, IKey: "execution.transactional_auth",
		InternalMsg: "Transactional auth error", cause: c, InternalCaller: CallerN(1)}
}

func NewAdviseInvalidResultsError() Error {
	return &err{level: EXCEPTION, ICode: E_ADVISE_INVALID_RESULTS, IKey: "execution.advise.invalid_results",
		InternalMsg: "Invalid advise results", InternalCaller: CallerN(1)}
}

func NewInvalidDocumentKeyTypeWarning(v interface{}, t string) Error {
	c := make(map[string]interface{}, 2)
	c["value"] = v
	c["type"] = t
	return &err{level: WARNING, ICode: W_DOCUMENT_KEY_TYPE, IKey: "execution.document_key.type", cause: c,
		InternalMsg: fmt.Sprintf("Document key must be a string: %v", v), InternalCaller: CallerN(1)}
}
