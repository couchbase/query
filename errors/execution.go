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

func NewEvaluationError(e error, termType string) Error {
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

func NewNoValueForKey(key string) Error {
	return &err{level: EXCEPTION, ICode: 5200, IKey: "execution.no_value_for_key",
		InternalMsg: fmt.Sprintf("Unable to find a value for key %s.", key), InternalCaller: CallerN(1)}
}

func NewUserNotFoundError(u string) Error {
	return &err{level: EXCEPTION, ICode: 5210, IKey: "execution.user_not_found",
		InternalMsg: fmt.Sprintf("Unable to find user %s.", u), InternalCaller: CallerN(1)}
}

func NewAdminInputNotObject(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5220, IKey: "execution.admin.input_not_object",
		InternalMsg: fmt.Sprintf("Input to DO_ADMIN should be a JSON object but is not: %v.", input), InternalCaller: CallerN(1)}
}

func NewAdminActionNotPresent(input map[string]interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5230, IKey: "execution.admin.action_not_present",
		InternalMsg: fmt.Sprintf("Input to DO_ADMIN should have an 'action' entry: %v.", input), InternalCaller: CallerN(1)}
}

func NewAdminActionMustBeString(action interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5240, IKey: "execution.admin.action_must_be_string",
		InternalMsg: fmt.Sprintf("Input to DO_ADMIN should have an 'action' entry that is a string: %v.", action), InternalCaller: CallerN(1)}
}

func NewAdminUnknownAction(action string) Error {
	return &err{level: EXCEPTION, ICode: 5250, IKey: "execution.admin.unknown_action",
		InternalMsg: fmt.Sprintf("Input to DO_ADMIN has an 'action' entry that is not valid: %s.", action), InternalCaller: CallerN(1)}
}

func NewGrantRoleHasUserOrUsers(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5260, IKey: "execution.admin.user_or_users",
		InternalMsg:    fmt.Sprintf("The input to a GRANT_ROLE action should have a 'user' field or a 'users' field but not both: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleUsersMustBeArray(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5270, IKey: "execution.admin.users_must_be_array",
		InternalMsg:    fmt.Sprintf("In a GRANT_ROLE action the 'users' field should be an array: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleUserMustBeString(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5280, IKey: "execution.admin.user_must_be_string",
		InternalMsg:    fmt.Sprintf("In a GRANT_ROLE action users must be strings: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleHasRoleOrRoles(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5290, IKey: "execution.admin.role_or_roles",
		InternalMsg:    fmt.Sprintf("The input to a GRANT_ROLE action should have a 'role' field or a 'roles' field but not both: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleRolesMustBeArray(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5300, IKey: "execution.admin.roles_must_be_array",
		InternalMsg:    fmt.Sprintf("In a GRANT_ROLE action the 'roles' field should be an array: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleRoleMustBeObject(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5310, IKey: "execution.admin.role_must_be_object",
		InternalMsg:    fmt.Sprintf("In a GRANT_ROLE action every role must be a JSON object: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleRoleNameMustBePresent(input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5320, IKey: "execution.admin.role_name_must_be_present",
		InternalMsg:    fmt.Sprintf("In a GRANT_ROLE action every role must have a 'name' field: %v.", input),
		InternalCaller: CallerN(1)}
}

func NewGrantRoleFieldMustBeString(fieldName string, input interface{}) Error {
	return &err{level: EXCEPTION, ICode: 5280, IKey: "execution.admin.field_must_be_string",
		InternalMsg:    fmt.Sprintf("In a GRANT_ROLE action the %s field of each role must have be a string: %v.", fieldName, input),
		InternalCaller: CallerN(1)}
}
