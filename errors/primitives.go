//  Copyright 2017-Present Couchbase, Inc.
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

func NewBucketUpdaterMaxErrors(e error) Error {
	c := make(map[string]interface{})
	c["last_error"] = e
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_MAX_ERRORS, IKey: "primitives.updater.max_errors", cause: c,
		InternalMsg: fmt.Sprintf("Max failures reached. Last error: %v", e), InternalCaller: CallerN(1)}
}

func NewBucketUpdaterNoHealthyNodesFound() Error {
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_NO_HEALTHY_NODES, IKey: "primitives.updater.no_healthy_nodes",
		InternalMsg: "No healthy nodes found.", InternalCaller: CallerN(1)}
}

func NewBucketUpdaterStreamingError(e error) Error {
	c := make(map[string]interface{})
	c["stream_error"] = e
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_STREAM_ERROR, IKey: "primitives.updater.stream_error", cause: c,
		InternalMsg: fmt.Sprintf("Streaming error: %v", e), InternalCaller: CallerN(1)}
}

func NewBucketUpdaterAuthError(e error) Error {
	c := make(map[string]interface{})
	c["auth_error"] = e
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_AUTH_ERROR, IKey: "primitives.updater.auth_error", cause: c,
		InternalMsg: fmt.Sprintf("Authentication error: %v", e), InternalCaller: CallerN(1)}
}

func NewBucketUpdaterFailedToConnectToHost(status int, body interface{}) Error {
	c := make(map[string]interface{})
	c["status"] = status
	c["body"] = body
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_CONNECTION_FAILED, IKey: "primitives.updater.connection_failed", cause: c,
		InternalMsg: fmt.Sprintf("Failed to connect to host. Status %v Body %s", status, body), InternalCaller: CallerN(1)}
}

func NewBucketUpdaterMappingError(e error) Error {
	c := make(map[string]interface{})
	c["mapping_error"] = e
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_ERROR_MAPPING, IKey: "primitives.updater.mapping", cause: c,
		InternalMsg: fmt.Sprintf("Mapping error: %v", e), InternalCaller: CallerN(1)}
}

func NewBucketUpdaterEndpointNotFoundError() Error {
	return &err{level: EXCEPTION, ICode: E_BUCKET_UPDATER_EP_NOT_FOUND, IKey: "primitives.updater.endpoint_not_found",
		InternalMsg: "Streaming endpoint not found", InternalCaller: CallerN(1)}
}
