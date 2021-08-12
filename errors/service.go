//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package errors

import (
	"fmt"
	"github.com/couchbase/query/value"
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
	return &err{level: EXCEPTION, ICode: 1080, IKey: "timeout", InternalMsg: fmt.Sprintf("Timeout %v exceeded", timeout),
		InternalCaller: CallerN(1), retry: value.TRUE}
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
	return &err{level: EXCEPTION, ICode: 1140, IKey: "service.io.request.scan_vector.length",
		InternalMsg: fmt.Sprintf("Array %v should be of length 2", vec), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadSequenceNumber(seq interface{}) Error {
	return &err{level: EXCEPTION, ICode: 1150, IKey: "service.io.request.scan_vector.sequence",
		InternalMsg: fmt.Sprintf("Bad sequence number %v. Expected an unsigned 64-bit integer.", seq), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadUUID(uuid interface{}) Error {
	return &err{level: EXCEPTION, ICode: 1155, IKey: "service.io.request.scan_vector.uuid",
		InternalMsg: fmt.Sprintf("Bad UUID %v. Expected a string.", uuid), InternalCaller: CallerN(1)}
}

func NewServiceErrorDecodeNil() Error {
	return &err{level: EXCEPTION, ICode: 1160, IKey: "service.io.request.nil",
		InternalMsg: "Failed to decode nil value.", InternalCaller: CallerN(1)}
}

func NewServiceErrorHttpMethod(method string) Error {
	return &err{level: EXCEPTION, ICode: 1170, IKey: "service.io.request.method",
		InternalMsg: fmt.Sprintf("Unsupported method %s", method), InternalCaller: CallerN(1)}
}

func NewServiceShuttingDownError() Error {
	return &err{level: EXCEPTION, ICode: 1180, IKey: "service.shuttingdown",
		InternalMsg: "Service shutting down", InternalCaller: CallerN(1)}
}

func NewServiceShutDownError() Error {
	return &err{level: EXCEPTION, ICode: 1181, IKey: "service.shutdown",
		InternalMsg: "Service shut down", InternalCaller: CallerN(1)}
}

func NewServiceUserRequestExceededError() Error {
	return &err{level: EXCEPTION, ICode: 1191, IKey: "service.requests.exceeded",
		InternalMsg: "User has more requests running than allowed", InternalCaller: CallerN(1)}
}

func NewServiceUserRequestRateExceededError() Error {
	return &err{level: EXCEPTION, ICode: 1192, IKey: "service.request.rate.exceeded",
		InternalMsg: "User has exceeded request rate limit", InternalCaller: CallerN(1)}
}

func NewServiceUserRequestSizeExceededError() Error {
	return &err{level: EXCEPTION, ICode: 1193, IKey: "service.request.size.exceeded",
		InternalMsg: "User has exceeded input network traffic limit", InternalCaller: CallerN(1)}
}

func NewServiceUserResultsSizeExceededError() Error {
	return &err{level: EXCEPTION, ICode: 1194, IKey: "service.result.size.exceeded",
		InternalMsg: "User has exceeded results size limit", InternalCaller: CallerN(1)}
}
