//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package err provides user-visible errors and warnings. These errors
include error codes and will eventually provide multi-language
messages.

*/
package errors

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"
	"strings"
	"time"
)

const (
	EXCEPTION = iota
	WARNING
	NOTICE
	INFO
	LOG
	DEBUG
)

type Errors []Error

// Error will eventually include code, message key, and internal error
// object (cause) and message
type Error interface {
	error
	Code() int32
	TranslationKey() string
	Cause() error
	Level() int
	IsFatal() bool
}

type ErrorChannel chan Error

func NewError(e error, internalMsg string) Error {
	return &err{level: EXCEPTION, ICode: 5000, IKey: "Internal Error", ICause: e, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewWarning(internalMsg string) Error {
	return &err{level: WARNING, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewNotice(internalMsg string) Error {
	return &err{level: NOTICE, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewInfo(internalMsg string) Error {
	return &err{level: INFO, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewLog(internalMsg string) Error {
	return &err{level: LOG, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewDebug(internalMsg string) Error {
	return &err{level: DEBUG, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

type err struct {
	ICode          int32
	IKey           string
	ICause         error
	InternalMsg    string
	InternalCaller string
	level          int
}

func (e *err) Error() string {
	switch {
	default:
		return "Unspecified error."
	case e.InternalMsg != "" && e.ICause != nil:
		return e.InternalMsg + " - cause: " + e.ICause.Error()
	case e.InternalMsg != "":
		return e.InternalMsg
	case e.ICause != nil:
		return e.ICause.Error()
	}
}

func (e *err) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"code":    e.ICode,
		"key":     e.IKey,
		"message": e.InternalMsg,
	}
	if e.ICause != nil {
		m["cause"] = e.ICause.Error()
	}
	if e.InternalCaller != "" &&
		!strings.HasPrefix("e.InternalCaller", "unknown:") {
		m["caller"] = e.InternalCaller
	}
	return json.Marshal(m)
}

func (e *err) Level() int {
	return e.level
}

func (e *err) IsFatal() bool {
	if e.level == EXCEPTION {
		return true
	}
	return false
}

func (e *err) Code() int32 {
	return e.ICode
}

func (e *err) TranslationKey() string {
	return e.IKey
}

func (e *err) Cause() error {
	return e.ICause
}

func NewParseError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 4100, IKey: "parse_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewSemanticError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 4200, IKey: "semantic_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewBucketDoesNotExist(bucket string) Error {
	return &err{level: EXCEPTION, ICode: 4040, IKey: "bucket_not_found", InternalMsg: fmt.Sprintf("Bucket %s does not exist", bucket), InternalCaller: CallerN(1)}
}

func NewPoolDoesNotExist(pool string) Error {
	return &err{level: EXCEPTION, ICode: 4041, IKey: "pool_not_found", InternalMsg: fmt.Sprintf("Pool %s does not exist", pool), InternalCaller: CallerN(1)}
}

func NewTimeoutError(timeout *time.Duration) Error {
	return &err{level: EXCEPTION, ICode: 4080, IKey: "timeout", InternalMsg: fmt.Sprintf("Timeout %v exceeded", timeout), InternalCaller: CallerN(1)}
}

func NewTotalRowsInfo(rows int) Error {
	return &err{level: INFO, ICode: 100, IKey: "total_rows", InternalMsg: fmt.Sprintf("%d", rows), InternalCaller: CallerN(1)}
}

func NewTotalElapsedTimeInfo(time string) Error {
	return &err{level: INFO, ICode: 101, IKey: "total_elapsed_time", InternalMsg: fmt.Sprintf("%s", time), InternalCaller: CallerN(1)}
}

func NewNotImplemented(feature string) Error {
	return &err{level: EXCEPTION, ICode: 1001, IKey: "not_implemented", InternalMsg: fmt.Sprintf("Not yet implemented: %v", feature), InternalCaller: CallerN(1)}
}

// service level errors - errors that are created in the service package

func NewServiceErrorReadonly(msg string) Error {
	return &err{level: EXCEPTION, ICode: 1000, IKey: "service.io.readonly", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewServiceErrorHTTPMethod(method string) Error {
	return &err{level: EXCEPTION, ICode: 1010, IKey: "service.io.http.unsupported_method",
		InternalMsg: fmt.Sprintf("Unsupported http method: %s", method), InternalCaller: CallerN(1)}
}

func NewServiceErrorNotImplemented(feature string, value string) Error {
	return &err{level: EXCEPTION, ICode: 1020, IKey: "service.io.request.unimplemented",
		InternalMsg: fmt.Sprintf("%s %s not yet implemented", value, feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorUnrecognizedValue(feature string, value string) Error {
	return &err{level: EXCEPTION, ICode: 1030, IKey: "service.io.request.unrecognized_value",
		InternalMsg: fmt.Sprintf("Unknown %s value: %s", feature, value), InternalCaller: CallerN(1)}
}

func NewServiceErrorBadValue(e error, feature string) Error {
	return &err{level: EXCEPTION, ICode: 1040, IKey: "service.io.request.bad_value", ICause: e,
		InternalMsg: fmt.Sprintf("Error processing %s", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorMissingValue(feature string) Error {
	return &err{level: EXCEPTION, ICode: 1050, IKey: "service.io.request.missing_value",
		InternalMsg: fmt.Sprintf("No %s value", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorMultipleValues(feature string) Error {
	return &err{level: EXCEPTION, ICode: 1060, IKey: "service.io.request.multiple_values",
		InternalMsg: fmt.Sprintf("Multiple values for %s.", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorTypeMismatch(feature string, expected string) Error {
	return &err{level: EXCEPTION, ICode: 1070, IKey: "service.io.request.type_mismatch",
		InternalMsg: fmt.Sprintf("%s has to be of type %s", feature, expected), InternalCaller: CallerN(1)}
}

func NewServiceErrorInvalidJSON(e error) Error {
	return &err{level: EXCEPTION, ICode: 1100, IKey: "service.io.response.invalid_json", ICause: e,
		InternalMsg: "Invalid JSON in results", InternalCaller: CallerN(1)}
}

// Parse errors - errors that are created in the parse package
func NewParseSyntaxError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 3000, IKey: "parse.syntax_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

// Plan errors - errors that are created in the plan package
func NewPlanError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 4000, IKey: "plan_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

// admin level errors - errors that are created in the clustering and accounting packages

func NewAdminConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2000, IKey: "admin.clustering.connection_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewAdminClusterConfigError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2010, IKey: "admin.clustering.cluster_config_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewAdminNodeConfigError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2020, IKey: "admin.clustering.node_config_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

// Returns "FileName:LineNum" of caller.
func Caller() string {
	return CallerN(1)
}

// Returns "FileName:LineNum" of the Nth caller on the call stack,
// where level of 0 is the caller of CallerN.
func CallerN(level int) string {
	_, fname, lineno, ok := runtime.Caller(1 + level)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d",
		strings.Split(path.Base(fname), ".")[0], lineno)
}
