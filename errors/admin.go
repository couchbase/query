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
)

// admin level errors - errors that are created in the clustering and accounting packages

func NewAdminConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_CONNECTION, IKey: "admin.clustering.connection_error", ICause: e,
		InternalMsg: "Error connecting to " + msg, InternalCaller: CallerN(1)}
}

func NewAdminInvalidURL(component string, url string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_INVALIDURL, IKey: "admin.invalid_url",
		InternalMsg: fmt.Sprintf("Invalid %s url: %s", component, url), InternalCaller: CallerN(1)}
}

func NewAdminDecodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_DECODING, IKey: "admin.json_decoding_error", ICause: e,
		InternalMsg: "Error in JSON decoding", InternalCaller: CallerN(1)}
}

func NewAdminEncodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_ENCODING, IKey: "admin.json_encoding_error", ICause: e,
		InternalMsg: "Error in JSON encoding", InternalCaller: CallerN(1)}
}

func NewAdminUnknownSettingError(setting string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_UNKNOWN_SETTING, IKey: "admin.unknown_setting",
		InternalMsg: fmt.Sprintf("Unknown setting: %s", setting), InternalCaller: CallerN(1)}
}

func NewAdminSettingTypeError(setting string, value interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_SETTING_TYPE, IKey: "admin.setting_type_error",
		InternalMsg: fmt.Sprintf("Incorrect value %v (%T) for setting: %s", value, value, setting), InternalCaller: CallerN(1)}
}

func NewAdminSettingMinimumError(min int) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_SETTING_MIN, IKey: "admin.setting_min_error",
		InternalMsg: fmt.Sprintf("Value below minimum of %d", min), InternalCaller: CallerN(1)}
}

func NewAdminGetClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_GET_CLUSTER, IKey: "admin.clustering.get_cluster_error", ICause: e,
		InternalMsg: "Error retrieving cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAddClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_ADD_CLUSTER, IKey: "admin.clustering.add_cluster_error", ICause: e,
		InternalMsg: "Error adding cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminRemoveClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_REMOVE_CLUSTER, IKey: "admin.clustering.remove_cluster_error", ICause: e,
		InternalMsg: "Error removing cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminGetNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_GET_NODE, IKey: "admin.clustering.get_node_error", ICause: e,
		InternalMsg: "Error retrieving node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminNoNodeError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_NO_NODE, IKey: "admin.clustering.no_such_node",
		InternalMsg: "No such  node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAddNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_ADD_NODE, IKey: "admin.clustering.add_node_error", ICause: e,
		InternalMsg: "Error adding node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminRemoveNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_REMOVE_NODE, IKey: "admin.clustering.remove_node_error", ICause: e,
		InternalMsg: "Error removing node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminMakeMetricError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_MAKE_METRIC, IKey: "admin.accounting.metric.create", ICause: e,
		InternalMsg: "Error creating metric " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAuthError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_AUTH, IKey: "admin.clustering.authorize", ICause: e,
		InternalMsg: "Error authorizing against cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminEndpointError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_ENDPOINT, IKey: "admin.service.HttpEndpoint", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewAdminNotSSLEnabledError() Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_SSL_NOT_ENABLED, IKey: "admin.service.ssl_cert",
		InternalMsg: "server is not ssl enabled", InternalCaller: CallerN(1)}
}

func NewAdminCredsError(creds string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_CREDS, IKey: "admin.accounting.bad_creds", ICause: e,
		InternalMsg: "Not a proper creds JSON array of user/pass structures: " + creds, InternalCaller: CallerN(1)}
}

// completed requests qualifier settings
func NewCompletedQualifierExists(what string) Error {
	return &err{level: EXCEPTION, ICode: E_COMPLETED_QUALIFIER_EXISTS, IKey: "admin.accounting.completed.already_exists",
		InternalMsg: "Completed requests qualifier already set: " + what, InternalCaller: CallerN(1)}
}

func NewCompletedQualifierUnknown(what string) Error {
	return &err{level: EXCEPTION, ICode: E_COMPLETED_QUALIFIER_UNKNOWN, IKey: "admin.accounting.completed.unknown",
		InternalMsg: "Completed requests qualifier unknown: " + what, InternalCaller: CallerN(1)}
}

func NewCompletedQualifierNotFound(what string, cond interface{}) Error {
	var condString string

	if cond != nil {
		condString = fmt.Sprintf(" %v", cond)
	}
	return &err{level: EXCEPTION, ICode: E_COMPLETED_QUALIFIER_NOT_FOUND, IKey: "admin.accounting.completed.not_found",
		InternalMsg: "Completed requests qualifier not set: " + what + condString, InternalCaller: CallerN(1)}
}

func NewCompletedQualifierNotUnique(what string) Error {
	return &err{level: EXCEPTION, ICode: E_COMPLETED_QUALIFIER_NOT_UNIQUE, IKey: "admin.accounting.completed.not_unique",
		InternalMsg: "Completed requests qualifier can only be deployed once: " + what, InternalCaller: CallerN(1)}
}

func NewCompletedQualifierInvalidArgument(what string, cond interface{}) Error {
	var condString string

	if cond != nil {
		condString = fmt.Sprintf(" %v", cond)
	}
	return &err{level: EXCEPTION, ICode: E_COMPLETED_QUALIFIER_INVALID_ARGUMENT, IKey: "admin.accounting.completed.invalid",
		InternalMsg: "Completed requests qualifier " + what + " cannot accept argument " + condString, InternalCaller: CallerN(1)}
}

func NewCompletedInvalidPlanSize(sz int) Error {
	c := make(map[string]interface{}, 2)
	c["invalid_size"] = sz
	return &err{level: EXCEPTION, ICode: E_COMPLETED_BAD_MAX_SIZE, IKey: "admin.accounting.completed.bad_max_size", cause: c,
		InternalMsg: fmt.Sprintf("Completed requests maximum plan size (%v) is invalid.", sz), InternalCaller: CallerN(1)}
}

func NewAdminBadServicePort(port string) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_BAD_SERVICE_PORT, IKey: "admin.clustering.bad_port",
		InternalMsg: "Invalid service port: " + port, InternalCaller: CallerN(1)}
}

func NewAdminBodyError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_ADMIN_BODY, IKey: "admin.accounting.bad_body", ICause: e,
		InternalMsg: "Error getting request body", InternalCaller: CallerN(1)}
}

func NewAdminManualFFDCError(msg string, remaining int) Error {
	var c map[string]interface{}
	if msg != "" {
		c = make(map[string]interface{}, 2)
		c["seconds_before_next"] = remaining
		c["message"] = msg
	}
	return &err{level: EXCEPTION, ICode: E_ADMIN_FFDC, IKey: "admin.ffdc", cause: c,
		InternalMsg: "FFDC invocation failed.", InternalCaller: CallerN(1)}
}
