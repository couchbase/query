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

	"github.com/couchbase/query/value"
)

// Execution errors - errors that are created in the execution package

func NewExecutionPanicError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5001, IKey: "execution.panic", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewExecutionInternalError(what string) Error {
	return &err{level: EXCEPTION, ICode: 5002, IKey: "execution.internal_error",
		InternalMsg: fmt.Sprintf("Execution internal error: %v", what), InternalCaller: CallerN(1)}
}

func NewExecutionParameterError(what string) Error {
	return &err{level: EXCEPTION, ICode: 5003, IKey: "execution.parameter_error",
		InternalMsg: fmt.Sprintf("Execution parameter error: %v", what), InternalCaller: CallerN(1)}
}

func NewEvaluationError(e error, termType string) Error {
	_, ok := e.(*AbortError)
	if ok {
		return &err{level: EXCEPTION, ICode: 5011, IKey: "execution.abort_error", ICause: e,
			InternalMsg: fmt.Sprintf("Abort: %s.", e), InternalCaller: CallerN(1)}
	}
	return &err{level: EXCEPTION, ICode: 5010, IKey: "execution.evaluation_error", ICause: e,
		InternalMsg: fmt.Sprintf("Error evaluating %s.", termType), InternalCaller: CallerN(1)}
}

func NewExplainError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5015, IKey: "execution.explain_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewGroupUpdateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5020, IKey: "execution.group_update_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewInvalidValueError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 5030, IKey: "execution.invalid_value_error",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewRangeError(termType string) Error {
	return &err{level: EXCEPTION, ICode: 5035, IKey: "execution.range_error",
		InternalMsg: fmt.Sprintf("Out of range evaluating %s.", termType), InternalCaller: CallerN(1)}
}

func NewDuplicateFinalGroupError() Error {
	return &err{level: EXCEPTION, ICode: 5040, IKey: "execution.duplicate_final_group",
		InternalMsg: "Duplicate Final Group.", InternalCaller: CallerN(1)}
}

func NewInsertKeyError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5050, IKey: "execution.insert_key_error",
		InternalMsg: fmt.Sprintf("No INSERT key for %v", v), InternalCaller: CallerN(1)}
}

func NewInsertValueError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5060, IKey: "execution.insert_value_error",
		InternalMsg: fmt.Sprintf("No INSERT value for %v", v), InternalCaller: CallerN(1)}
}

func NewInsertKeyTypeError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5070, IKey: "execution.insert_key_type_error",
		InternalMsg:    fmt.Sprintf("Cannot INSERT non-string key %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewInsertOptionsTypeError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5071, IKey: "execution.insert_options_type_error",
		InternalMsg:    fmt.Sprintf("Cannot INSERT non-OBJECT options %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewUpsertKeyError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5072, IKey: "execution.upsert_key_error",
		InternalMsg: fmt.Sprintf("No UPSERT key for %v", v), InternalCaller: CallerN(1)}
}

func NewUpsertValueError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5075, IKey: "execution.upsert_value_error",
		InternalMsg: fmt.Sprintf("No UPSERT value for %v", v), InternalCaller: CallerN(1)}
}

func NewUpsertKeyTypeError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5078, IKey: "execution.upsert_key_type_error",
		InternalMsg:    fmt.Sprintf("Cannot UPSERT non-string key %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewUpsertOptionsTypeError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5079, IKey: "execution.upsert_options_type_error",
		InternalMsg:    fmt.Sprintf("Cannot UPSERT non-OBJECT options %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewDeleteAliasMissingError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5080, IKey: "execution.missing_delete_alias",
		InternalMsg:    fmt.Sprintf("DELETE alias %s not found in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewDeleteAliasMetadataError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5090, IKey: "execution.delete_alias_metadata",
		InternalMsg:    fmt.Sprintf("DELETE alias %s has no metadata in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateAliasMissingError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5100, IKey: "execution.missing_update_alias",
		InternalMsg:    fmt.Sprintf("UPDATE alias %s not found in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateAliasMetadataError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5110, IKey: "execution.update_alias_metadata",
		InternalMsg:    fmt.Sprintf("UPDATE alias %s has no metadata in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateMissingClone() Error {
	return &err{level: EXCEPTION, ICode: 5120, IKey: "execution.update_missing_clone",
		InternalMsg: "Missing UPDATE clone.", InternalCaller: CallerN(1)}
}

func NewUnnestInvalidPosition(pos interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5180, IKey: "execution.unnest_invalid_position",
		InternalMsg: fmt.Sprintf("Invalid UNNEST position of type %T.", pos), InternalCaller: CallerN(1)}
}

func NewScanVectorTooManyScannedBuckets(buckets []string) Error {
	return &err{level: EXCEPTION, ICode: 5190, IKey: "execution.scan_vector_too_many_scanned_vectors",
		InternalMsg: fmt.Sprintf("The scan_vector parameter should not be used for queries accessing more than one keyspace. "+
			"Use scan_vectors instead. Keyspaces: %v", buckets), InternalCaller: CallerN(1)}
}

// Error code 5200 is retired. Do not reuse.

func NewUserNotFoundError(u string) Error {
	return &err{level: EXCEPTION, ICode: 5210, IKey: "execution.user_not_found",
		InternalMsg: fmt.Sprintf("Unable to find user %s.", u), InternalCaller: CallerN(1)}
}

func NewRoleRequiresKeyspaceError(role string) Error {
	return &err{level: EXCEPTION, ICode: 5220, IKey: "execution.role_requires_keyspace",
		InternalMsg: fmt.Sprintf("Role %s requires a keyspace.", role), InternalCaller: CallerN(1)}
}

func NewRoleTakesNoKeyspaceError(role string) Error {
	return &err{level: EXCEPTION, ICode: 5230, IKey: "execution.role_takes_no_keyspace",
		InternalMsg: fmt.Sprintf("Role %s does not take a keyspace.", role), InternalCaller: CallerN(1)}
}

func NewNoSuchKeyspaceError(bucket string) Error {
	return &err{level: EXCEPTION, ICode: 5240, IKey: "execution.no_such_keyspace",
		InternalMsg: fmt.Sprintf("Keyspace %s is not valid.", bucket), InternalCaller: CallerN(1)}
}

func NewNoSuchScopeError(scope string) Error {
	return &err{level: EXCEPTION, ICode: 5241, IKey: "execution.no_such_scope",
		InternalMsg: fmt.Sprintf("Scope %s is not valid.", scope), InternalCaller: CallerN(1)}
}

func NewNoSuchBucketError(bucket string) Error {
	return &err{level: EXCEPTION, ICode: 5242, IKey: "execution.no_such_bucket",
		InternalMsg: fmt.Sprintf("Bucket %s is not valid.", bucket), InternalCaller: CallerN(1)}
}

func NewRoleNotFoundError(role string) Error {
	return &err{level: EXCEPTION, ICode: 5250, IKey: "execution.role_not_found",
		InternalMsg: fmt.Sprintf("Role %s is not valid.", role), InternalCaller: CallerN(1)}
}

func NewRoleAlreadyPresent(user string, role string, bucket string) Error {
	var msg string
	if bucket == "" {
		msg = fmt.Sprintf("User %s already has role %s.", user, role)
	} else {
		msg = fmt.Sprintf("User %s already has role %s(%s).", user, role, bucket)
	}
	return &err{level: WARNING, ICode: 5260, IKey: "execution.role_already_present",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewRoleNotPresent(user string, role string, bucket string) Error {
	var msg string
	if bucket == "" {
		msg = fmt.Sprintf("User %s did not have role %s.", user, role)
	} else {
		msg = fmt.Sprintf("User %s did not have role %s(%s).", user, role, bucket)
	}
	return &err{level: WARNING, ICode: 5270, IKey: "execution.role_not_present",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewUserWithNoRoles(user string) Error {
	return &err{level: WARNING, ICode: 5280, IKey: "execution.user_with_no_roles",
		InternalMsg:    fmt.Sprintf("User %s has no roles. Connecting with this user may not be possible", user),
		InternalCaller: CallerN(1)}
}

// Error code 5290 is retired. Do not reuse.

func NewHashTablePutError(e error) Error {
	return &err{level: EXCEPTION, ICode: 5300, IKey: "execution.hash_table_put_error", ICause: e,
		InternalMsg:    fmt.Sprintf("Hash Table Put failed"),
		InternalCaller: CallerN(1)}
}

func NewHashTableGetError(e error) Error {
	return &err{level: EXCEPTION, ICode: 5310, IKey: "execution.hash_table_get_error", ICause: e,
		InternalMsg:    fmt.Sprintf("Hash Table Get failed"),
		InternalCaller: CallerN(1)}
}

func NewMergeMultiUpdateError(key string) Error {
	return &err{level: EXCEPTION, ICode: 5320, IKey: "execution.merge_multiple_update",
		InternalMsg:    fmt.Sprintf("Multiple UPDATE/DELETE of the same document (document key '%s') in a MERGE statement", key),
		InternalCaller: CallerN(1)}
}

func NewMergeMultiInsertError(key string) Error {
	return &err{level: EXCEPTION, ICode: 5330, IKey: "execution.merge_multiple_insert",
		InternalMsg:    fmt.Sprintf("Multiple INSERT of the same document (document key '%s') in a MERGE statement", key),
		InternalCaller: CallerN(1)}
}

func NewWindowEvaluationError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5340, IKey: "execution.window_aggregate_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewAdviseIndexError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5350, IKey: "execution.advise_index_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewUpdateStatisticsError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 5360, IKey: "execution.update_statistics",
		InternalMsg:    msg,
		InternalCaller: CallerN(1)}
}

const SUBQUERY_BUILD = 5370

func NewSubqueryBuildError(e error) Error {
	return &err{level: EXCEPTION, ICode: SUBQUERY_BUILD, IKey: "execution.subquery.build", ICause: e,
		InternalMsg:    "Unable to run subquery",
		InternalCaller: CallerN(1)}
}

func NewIndexLeadingKeyMissingNotSupportedError() Error {
	return &err{level: EXCEPTION, ICode: 5380, IKey: "execution.indexing.leadingkey_missing_not_supported",
		InternalMsg:    fmt.Sprintf("Indexing leading key MISSING values are not supported by indexer."),
		InternalCaller: CallerN(1)}
}

func NewIndexNotInMemory(msg string) Error {
	return &err{level: EXCEPTION, ICode: 5390, IKey: "execution.update_statistics.index_not_in_memory",
		InternalMsg:    msg,
		InternalCaller: CallerN(1)}
}

func NewMissingSystemCBOStatsError() Error {
	return &err{level: EXCEPTION, ICode: 5400, IKey: "execution.update_statistics.missing_system_cbostats",
		InternalMsg:    "System Collection 'N1QL_CBO_STATS' is required for UPDATE STATISTICS (ANALYZE)",
		InternalCaller: CallerN(1)}
}

func NewInvalidIndexNameError(name interface{}, ikey string) Error {
	return &err{level: EXCEPTION, ICode: 5410, IKey: ikey,
		InternalMsg:    fmt.Sprintf("index name(%v) must be a string", name),
		InternalCaller: CallerN(1)}
}

func NewIndexNotFoundError(name string, ikey string, e error) Error {
	return &err{level: EXCEPTION, ICode: 5411, IKey: ikey, ICause: e,
		InternalMsg:    fmt.Sprintf("index %s is not found", name),
		InternalCaller: CallerN(1)}
}

func NewMemoryQuotaExceededError() Error {
	return &err{level: EXCEPTION, ICode: 5500, IKey: "execution.memory_quota.exceeded",
		InternalMsg:    "Request has exceeded memory quota",
		InternalCaller: CallerN(1)}
}
