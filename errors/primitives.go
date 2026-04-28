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

func NewUpdaterMaxErrors(name string, e error) Error {
	c := make(map[string]interface{})
	c["last_error"] = e
	return &err{level: EXCEPTION, ICode: E_UPDATER_MAX_ERRORS, IKey: "primitives.updater.max_errors", cause: c,
		InternalMsg: fmt.Sprintf("%s: Max failures reached. Last error: %v", name, e), InternalCaller: CallerN(1)}
}

func NewUpdaterNoHealthyNodesFound(name string) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATER_NO_HEALTHY_NODES, IKey: "primitives.updater.no_healthy_nodes",
		InternalMsg: fmt.Sprintf("%s: No healthy nodes found.", name), InternalCaller: CallerN(1)}
}

func NewUpdaterStreamingError(name string, e error) Error {
	c := make(map[string]interface{})
	c["stream_error"] = e
	return &err{level: EXCEPTION, ICode: E_UPDATER_STREAM_ERROR, IKey: "primitives.updater.stream_error", cause: c,
		InternalMsg: fmt.Sprintf("%s: Streaming error: %v", name, e), InternalCaller: CallerN(1)}
}

func NewUpdaterAuthError(name string, e error) Error {
	c := make(map[string]interface{})
	c["auth_error"] = e
	return &err{level: EXCEPTION, ICode: E_UPDATER_AUTH_ERROR, IKey: "primitives.updater.auth_error", cause: c,
		InternalMsg: fmt.Sprintf("%s: Authentication error: %v", name, e), InternalCaller: CallerN(1)}
}

func NewUpdaterFailedToConnectToHost(name string, status int, body interface{}) Error {
	c := make(map[string]interface{})
	c["status"] = status
	c["body"] = body
	return &err{level: EXCEPTION, ICode: E_UPDATER_CONNECTION_FAILED, IKey: "primitives.updater.connection_failed", cause: c,
		InternalMsg: fmt.Sprintf("%s: Failed to connect to host. Status %v Body %s", name, status, body), InternalCaller: CallerN(1)}
}

func NewUpdaterMappingError(name string, e error) Error {
	c := make(map[string]interface{})
	c["mapping_error"] = e
	return &err{level: EXCEPTION, ICode: E_UPDATER_ERROR_MAPPING, IKey: "primitives.updater.mapping", cause: c,
		InternalMsg: fmt.Sprintf("%s: Mapping error: %v", name, e), InternalCaller: CallerN(1)}
}

func NewUpdaterEndpointNotFoundError(name string) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATER_EP_NOT_FOUND, IKey: "primitives.updater.endpoint_not_found",
		InternalMsg: fmt.Sprintf("%s: Streaming endpoint not found", name), InternalCaller: CallerN(1)}
}
