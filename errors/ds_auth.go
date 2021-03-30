//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
