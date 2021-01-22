//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

import ()

// Couchbase authorization error
const DS_AUTH_ERROR = 10000

func NewDatastoreAuthorizationError(e error) Error {
	return &err{level: EXCEPTION, ICode: DS_AUTH_ERROR, IKey: "datastore.couchbase.authorization_error", ICause: e,
		InternalMsg: "Unable to authorize user.", InternalCaller: CallerN(1)}
}

// Error codes 13010-13011 are retired. Do not reuse.

func NewDatastoreClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13012, IKey: "datastore.couchbase.cluster_error", ICause: e,
		InternalMsg: "Error retrieving cluster " + msg, InternalCaller: CallerN(1)}
}

func NewDatastoreUnableToRetrieveRoles(e error) Error {
	return &err{level: EXCEPTION, ICode: 13013, IKey: "datastore.couchbase.retrieve_roles", ICause: e,
		InternalMsg: "Unable to retrieve roles from server.", InternalCaller: CallerN(1)}
}

func NewDatastoreInsufficientCredentials(msg string) Error {
	return &err{level: EXCEPTION, ICode: 13014, IKey: "datastore.couchbase.insufficient_credentials",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}
