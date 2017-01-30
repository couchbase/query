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
)

// admin level errors - errors that are created in the clustering and accounting packages

func NewAdminConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2000, IKey: "admin.clustering.connection_error", ICause: e,
		InternalMsg: "Error connecting to " + msg, InternalCaller: CallerN(1)}
}

func NewAdminInvalidURL(component string, url string) Error {
	return &err{level: EXCEPTION, ICode: 2010, IKey: "admin.invalid_url",
		InternalMsg: fmt.Sprintf("Invalid % url: %s", component, url), InternalCaller: CallerN(1)}
}

func NewAdminDecodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: 2020, IKey: "admin.json_decoding_error", ICause: e,
		InternalMsg: "Error in JSON decoding", InternalCaller: CallerN(1)}
}

func NewAdminEncodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: 2030, IKey: "admin.json_encoding_error", ICause: e,
		InternalMsg: "Error in JSON encoding", InternalCaller: CallerN(1)}
}

func NewAdminUnknownSettingError(setting string) Error {
	return &err{level: EXCEPTION, ICode: 2032, IKey: "admin.unknown_setting",
		InternalMsg: fmt.Sprintf("Unknown setting: %s", setting), InternalCaller: CallerN(1)}
}

func NewAdminSettingTypeError(setting string, value interface{}) Error {
	return &err{level: EXCEPTION, ICode: 2032, IKey: "admin.setting_type_error",
		InternalMsg: fmt.Sprintf("Incorrect value %v for setting: %s", value, setting), InternalCaller: CallerN(1)}
}

func NewAdminGetClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2040, IKey: "admin.clustering.get_cluster_error", ICause: e,
		InternalMsg: "Error retrieving cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAddClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2050, IKey: "admin.clustering.add_cluster_error", ICause: e,
		InternalMsg: "Error adding cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminRemoveClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2060, IKey: "admin.clustering.remove_cluster_error", ICause: e,
		InternalMsg: "Error removing cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminGetNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2070, IKey: "admin.clustering.get_node_error", ICause: e,
		InternalMsg: "Error retrieving node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminNoNodeError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 2080, IKey: "admin.clustering.no_such_node",
		InternalMsg: "No such  node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAddNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2090, IKey: "admin.clustering.add_node_error", ICause: e,
		InternalMsg: "Error adding node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminRemoveNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2100, IKey: "admin.clustering.remove_node_error", ICause: e,
		InternalMsg: "Error removing node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminMakeMetricError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2110, IKey: "admin.accounting.metric.create", ICause: e,
		InternalMsg: "Error creating metric " + msg, InternalCaller: CallerN(1)}
}

const ADMIN_AUTH_ERROR = 2120

func NewAdminAuthError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: ADMIN_AUTH_ERROR, IKey: "admin.clustering.authorize", ICause: e,
		InternalMsg: "Error authorizing against cluster " + msg, InternalCaller: CallerN(1)}
}

const ADMIN_ENDPOINT_ERROR = 2130

func NewAdminEndpointError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: ADMIN_ENDPOINT_ERROR, IKey: "admin.service.HttpEndpoint", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

const ADMIN_SSL_NOT_ENABLED = 2140

func NewAdminNotSSLEnabledError() Error {
	return &err{level: EXCEPTION, ICode: ADMIN_SSL_NOT_ENABLED, IKey: "admin.service.ssl_cert",
		InternalMsg: "server is not ssl enabled", InternalCaller: CallerN(1)}
}

const ADMIN_CREDS_ERROR = 2150

func NewAdminCredsError(creds string, e error) Error {
	return &err{level: EXCEPTION, ICode: ADMIN_CREDS_ERROR, IKey: "admin.accounting.bad_creds", ICause: e,
		InternalMsg: "Not a proper creds JSON array of user/pass structures: " + creds, InternalCaller: CallerN(1)}
}
