//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

// Couchbase authorization error
func NewDatastoreAuthorizationError(e error) Error {
	var c interface{}
	if e != nil {
		m := make(map[string]interface{})
		m["error"] = e
		c = m
	}
	return &err{level: EXCEPTION, ICode: E_DATASTORE_AUTHORIZATION, IKey: "datastore.couchbase.authorization_error", ICause: e,
		InternalMsg: "Unable to authorize user.", InternalCaller: CallerN(1), cause: c}
}

// Error codes 13010-13011 are retired. Do not reuse.

func NewDatastoreClusterError(e error, msg string) Error {
	var c interface{}
	if e != nil {
		m := make(map[string]interface{})
		m["error"] = e
		c = m
	}
	return &err{level: EXCEPTION, ICode: E_DATASTORE_CLUSTER, IKey: "datastore.couchbase.cluster_error", ICause: e,
		InternalMsg: "Error retrieving cluster " + msg, InternalCaller: CallerN(1), cause: c}
}

func NewDatastoreUnableToRetrieveRoles(e error) Error {
	var c interface{}
	if e != nil {
		m := make(map[string]interface{})
		m["error"] = e
		c = m
	}
	return &err{level: EXCEPTION, ICode: E_DATASTORE_UNABLE_TO_RETRIEVE_ROLES, IKey: "datastore.couchbase.retrieve_roles",
		ICause: e, InternalMsg: "Unable to retrieve roles from server.", InternalCaller: CallerN(1), cause: c}
}

func NewDatastoreInsufficientCredentials(msg string, e error, path string) Error {
	c := make(map[string]interface{})
	if e != nil {
		c["error"] = e
	}
	c["path"] = path
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INSUFFICIENT_CREDENTIALS,
		IKey:        "datastore.couchbase.insufficient_credentials",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewDatastoreUnableToRetrieveBuckets(e error) Error {
	var c interface{}
	if e != nil {
		m := make(map[string]interface{})
		m["error"] = e
		c = m
	}
	return &err{level: EXCEPTION, ICode: E_DATASTORE_UNABLE_TO_RETRIEVE_BUCKETS, IKey: "datastore.couchbase.retrieve_buckets",
		ICause: e, InternalMsg: "Unable to retrieve buckets from server.", InternalCaller: CallerN(1), cause: c}
}

func NewNoAdminPrivilegeError(e error) Error {
	var c interface{}
	if e != nil {
		m := make(map[string]interface{})
		m["error"] = e
		c = m
	}
	return &err{level: EXCEPTION, ICode: E_DATASTORE_NO_ADMIN,
		IKey:        "datastore.couchbase.no.admin",
		InternalMsg: "Unable to determine admin credentials", InternalCaller: CallerN(1), cause: c}
}
