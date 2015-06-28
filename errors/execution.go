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

func NewEvaluationError(e error, termType string) Error {
	return &err{level: EXCEPTION, ICode: 5010, IKey: "execution.evaluation_error", ICause: e,
		InternalMsg: fmt.Sprintf("Error evaluating %s.", termType), InternalCaller: CallerN(1)}
}

func NewGroupUpdateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5020, IKey: "execution.group_update_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewInvalidValueError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 5030, IKey: "execution.invalid_value_error",
		InternalMsg: msg, InternalCaller: CallerN(1)}
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
