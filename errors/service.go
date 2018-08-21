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
	"time"
)

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

const SERVICE_MULTIPLE_VALUES = 1060

func NewServiceErrorMultipleValues(feature string) Error {
	return &err{level: EXCEPTION, ICode: SERVICE_MULTIPLE_VALUES, IKey: "service.io.request.multiple_values",
		InternalMsg: fmt.Sprintf("Multiple values for %s.", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorUnrecognizedParameter(parameter string) Error {
	return &err{level: EXCEPTION, ICode: 1065, IKey: "service.io.request.unrecognized_parameter",
		InternalMsg: fmt.Sprintf("Unrecognized parameter in request: %s", parameter), InternalCaller: CallerN(1)}
}

func NewServiceErrorTypeMismatch(feature string, expected string) Error {
	return &err{level: EXCEPTION, ICode: 1070, IKey: "service.io.request.type_mismatch",
		InternalMsg: fmt.Sprintf("%s has to be of type %s", feature, expected), InternalCaller: CallerN(1)}
}

func NewTimeoutError(timeout time.Duration) Error {
	return &err{level: EXCEPTION, ICode: 1080, IKey: "timeout", InternalMsg: fmt.Sprintf("Timeout %v exceeded", timeout), InternalCaller: CallerN(1)}
}

func NewServiceErrorInvalidJSON(e error) Error {
	return &err{level: EXCEPTION, ICode: 1100, IKey: "service.io.response.invalid_json", ICause: e,
		InternalMsg: "Invalid JSON in results", InternalCaller: CallerN(1)}
}

func NewServiceErrorClientID(id string) Error {
	return &err{level: EXCEPTION, ICode: 1110, IKey: "service.io.response.client_id",
		InternalMsg: "forbidden character (\\ or \") in client_context_id", InternalCaller: CallerN(1)}
}

func NewServiceErrorMediaType(mediaType string) Error {
	return &err{level: EXCEPTION, ICode: 1120, IKey: "service.io.request.media_type",
		InternalMsg: fmt.Sprintf("Unsupported media type: %s", mediaType), InternalCaller: CallerN(1)}
}

func NewServiceErrorHttpReq(id string) Error {
	return &err{level: EXCEPTION, ICode: 1130, IKey: "service.io.request.type",
		InternalMsg: fmt.Sprintf("Request %s is not a http request", id), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadLength(vec []interface{}) Error {
	return &err{level: EXCEPTION, ICode: 1140, IKey: "service.io.request.type",
		InternalMsg: fmt.Sprintf("Array %v should be of length 2", vec), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadSequenceNumber(seq interface{}) Error {
	return &err{level: EXCEPTION, ICode: 1150, IKey: "service.io.request.type",
		InternalMsg: fmt.Sprintf("Bad sequence number %v. Expected an unsigned 64-bit integer.", seq), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadUUID(uuid interface{}) Error {
	return &err{level: EXCEPTION, ICode: 1150, IKey: "service.io.request.type",
		InternalMsg: fmt.Sprintf("Bad UUID %v. Expected a string.", uuid), InternalCaller: CallerN(1)}
}

func NewServiceErrorDecodeNil() Error {
	return &err{level: EXCEPTION, ICode: 1160, IKey: "service.io.request.type",
		InternalMsg: "Failed to decode nil value.", InternalCaller: CallerN(1)}
}

func NewServiceErrorHttpMethod(method string) Error {
	return &err{level: EXCEPTION, ICode: 1170, IKey: "service.io.request.method",
		InternalMsg: fmt.Sprintf("Unsupported method %s", method), InternalCaller: CallerN(1)}
}
