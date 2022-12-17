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
	"time"
)

// service level errors - errors that are created in the service package

func NewServiceErrorReadonly(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_READONLY, IKey: "service.io.readonly", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewServiceErrorHTTPMethod(method string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_HTTP_UNSUPPORTED_METHOD, IKey: "service.io.http.unsupported_method",
		InternalMsg: fmt.Sprintf("Unsupported http method: %s", method), InternalCaller: CallerN(1)}
}

func NewServiceErrorNotImplemented(feature string, value string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_NOT_IMPLEMENTED, IKey: "service.io.request.unimplemented",
		InternalMsg: fmt.Sprintf("%s %s not yet implemented", value, feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorUnrecognizedValue(feature string, value string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_UNRECOGNIZED_VALUE, IKey: "service.io.request.unrecognized_value",
		InternalMsg: fmt.Sprintf("Unknown %s value: %s", feature, value), InternalCaller: CallerN(1)}
}

func NewServiceErrorBadValue(e error, feature string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_BAD_VALUE, IKey: "service.io.request.bad_value", ICause: e,
		InternalMsg: fmt.Sprintf("Error processing %s", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorMissingValue(feature string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_MISSING_VALUE, IKey: "service.io.request.missing_value",
		InternalMsg: fmt.Sprintf("No %s value", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorMultipleValues(feature string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_MULTIPLE_VALUES, IKey: "service.io.request.multiple_values",
		InternalMsg: fmt.Sprintf("Multiple values for %s.", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorUnrecognizedParameter(parameter string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_UNRECOGNIZED_PARAMETER, IKey: "service.io.request.unrecognized_parameter",
		InternalMsg: fmt.Sprintf("Unrecognized parameter in request: %s", parameter), InternalCaller: CallerN(1)}
}

func NewServiceErrorTypeMismatch(feature string, expected string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_TYPE_MISMATCH, IKey: "service.io.request.type_mismatch",
		InternalMsg: fmt.Sprintf("%s has to be of type %s", feature, expected), InternalCaller: CallerN(1)}
}

func NewTimeoutError(timeout time.Duration) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_TIMEOUT, IKey: "timeout",
		InternalMsg: fmt.Sprintf("Timeout %v exceeded", timeout), InternalCaller: CallerN(1), retry: TRUE}
}

func NewServiceErrorInvalidJSON(e error) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_INVALID_JSON, IKey: "service.io.response.invalid_json", ICause: e,
		InternalMsg: "Invalid JSON in results", InternalCaller: CallerN(1)}
}

func NewServiceErrorClientID(id string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_CLIENTID, IKey: "service.io.response.client_id",
		InternalMsg: "forbidden character (\\ or \") in client_context_id", InternalCaller: CallerN(1)}
}

func NewServiceErrorMediaType(mediaType string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_MEDIA_TYPE, IKey: "service.io.request.media_type",
		InternalMsg: fmt.Sprintf("Unsupported media type: %s", mediaType), InternalCaller: CallerN(1)}
}

func NewServiceErrorHttpReq(id string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_HTTP_REQ, IKey: "service.io.request.type",
		InternalMsg: fmt.Sprintf("Request %s is not a http request", id), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadLength(vec []interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_SCAN_VECTOR_BAD_LENGTH, IKey: "service.io.request.scan_vector.length",
		InternalMsg: fmt.Sprintf("Array %v should be of length 2", vec), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadSequenceNumber(seq interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_SCAN_VECTOR_BAD_SEQUENCE_NUMBER, IKey: "service.io.request.scan_vector.sequence",
		InternalMsg: fmt.Sprintf("Bad sequence number %v. Expected an unsigned 64-bit integer.", seq), InternalCaller: CallerN(1)}
}

func NewServiceErrorScanVectorBadUUID(uuid interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_SCAN_VECTOR_BADUUID, IKey: "service.io.request.scan_vector.uuid",
		InternalMsg: fmt.Sprintf("Bad UUID %v. Expected a string.", uuid), InternalCaller: CallerN(1)}
}

func NewServiceErrorDecodeNil() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_DECODE_NIL, IKey: "service.io.request.nil",
		InternalMsg: "Failed to decode nil value.", InternalCaller: CallerN(1)}
}

func NewServiceErrorHttpMethod(method string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_HTTP_METHOD, IKey: "service.io.request.method",
		InternalMsg: fmt.Sprintf("Unsupported method %s", method), InternalCaller: CallerN(1)}
}

func NewServiceShuttingDownError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_SHUTTING_DOWN, IKey: "service.shuttingdown",
		InternalMsg: "Service shutting down", InternalCaller: CallerN(1)}
}

func NewServiceShutDownError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_SHUT_DOWN, IKey: "service.shutdown",
		InternalMsg: "Service shut down", InternalCaller: CallerN(1)}
}

func NewServiceUnavailableError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_UNAVAILABLE, IKey: "service.unavailable",
		InternalMsg: "Service cannot handle requests", InternalCaller: CallerN(1)}
}

func NewServiceUserRequestExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_USER_REQUEST_EXCEEDED, IKey: "service.requests.exceeded",
		InternalMsg: "User has more requests running than allowed", InternalCaller: CallerN(1)}
}

func NewServiceUserRequestRateExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_USER_REQUEST_RATE_EXCEEDED, IKey: "service.request.rate.exceeded",
		InternalMsg: "User has exceeded request rate limit", InternalCaller: CallerN(1)}
}

func NewServiceUserRequestSizeExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_USER_REQUEST_SIZE_EXCEEDED, IKey: "service.request.size.exceeded",
		InternalMsg: "User has exceeded input network traffic limit", InternalCaller: CallerN(1)}
}

func NewServiceUserResultsSizeExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_USER_RESULT_SIZE_EXCEEDED, IKey: "service.result.size.exceeded",
		InternalMsg: "User has exceeded results size limit", InternalCaller: CallerN(1)}
}

func NewErrorLimit(limit int, num int, dups int, mut uint64) Error {
	c := make(map[string]interface{})
	c["errorLimit"] = limit
	c["distinctErrors"] = num
	if dups > 0 {
		c["duplicateErrors"] = dups
	}
	if mut > 0 {
		c["mutationCount"] = mut
	}
	return &err{level: EXCEPTION, ICode: E_REQUEST_ERROR_LIMIT, IKey: "service.request.error_limit", cause: c,
		InternalMsg:    "Request execution aborted as the number of errors raised has reached the maximum permitted.",
		InternalCaller: CallerN(1)}
}

func NewServiceTenantThrottledError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_TENANT_THROTTLED, IKey: "service.tenant.throttled", ICause: e,
		InternalMsg: "Request has been declined", InternalCaller: CallerN(1)}
}

func NewServiceTenantMissingError() Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_TENANT_MISSING, IKey: "service.tenant.missing",
		InternalMsg: "Request does not have a valid tenant", InternalCaller: CallerN(1)}
}

func NewServiceTenantNotAuthorizedError(bucket string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_TENANT_NOT_AUTHORIZED, IKey: "service.tenant.not.authorized",
		InternalMsg: fmt.Sprintf("Request is not authorized for tenant %v", bucket), InternalCaller: CallerN(1)}
}

func NewServiceTenantNotFoundError(bucket string) Error {
	return &err{level: EXCEPTION, ICode: E_SERVICE_TENANT_NOT_FOUND, IKey: "service.tenant.not.found",
		InternalMsg: fmt.Sprintf("Tenant not found %v", bucket), InternalCaller: CallerN(1)}
}

func NewServiceTenantRejectedError(duration time.Duration) Error {
	var (
		message string
		cause   map[string]interface{} = make(map[string]interface{})
	)
	if duration == 0 {
		message = "Request rejected due to limiting or throttling. Retry later"
	} else {
		message = fmt.Sprintf("Request rejected due to limiting or throttling. Retry after %v", duration)
	}
	cause["retry_after"] = duration
	return &err{level: EXCEPTION, ICode: E_SERVICE_TENANT_REJECTED, IKey: "service.tenant.rejected", cause: cause,
		InternalMsg: message, InternalCaller: CallerN(1), retry: TRUE}
}

func NewEncodedPlanUseNotAllowedError() Error {
	return &err{level: EXCEPTION, ICode: E_ENCODED_PLAN_NOT_ALLOWED, IKey: "server.encoded_plan_use_not_allowed_error", InternalMsg: "Encoded plan use is not allowed in serverless mode.", InternalCaller: CallerN(1)}
}
