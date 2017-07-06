//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

import ()

// Datastore/couchbase/view index error codes
func NewCbViewCreateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13000, IKey: "datastore.couchbase.view.create_failed", ICause: e,
		InternalMsg: "Failed to create view " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13001, IKey: "datastore.couchbase.view.not_found", ICause: e,
		InternalMsg: "View Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewExistsError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13003, IKey: "datastore.couchbase.view.exists", ICause: e,
		InternalMsg: "View index exists " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsWithNotAllowedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13004, IKey: "datastore.couchbase.view.with_not_allowed", ICause: e,
		InternalMsg: "Views not allowed for WITH keyword " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13005, IKey: "datastore.couchbase.view.not_supported", ICause: e,
		InternalMsg: "View indexes not supported " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsDropIndexError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13006, IKey: "datastore.couchbase.view.drop_index_error", ICause: e,
		InternalMsg: "Failed to drop index " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsAccessError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13007, IKey: "datastore.couchbase.view.access_error", ICause: e,
		InternalMsg: "Failed to access view " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewIndexesLoadingError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13008, IKey: "datastore.couchbase.view.not_found", ICause: e,
		InternalMsg: "Failed to load indexes for keyspace " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewDefError(e error) Error {
	return &err{level: EXCEPTION, ICode: 13009, IKey: "datastore.couchbase.view.def_failed", ICause: e,
		InternalMsg: "Unable to store the view definition. Not all index target expressions are supported. " +
			"Check whether the JavaScript of the view definition is valid. The map function has been output to query.log.",
		InternalCaller: CallerN(1)}
}

func NewDatastoreNoUserSupplied() Error {
	return &err{level: EXCEPTION, ICode: 13010, IKey: "datastore.couchbase.no_user",
		InternalMsg: "No user supplied for query.", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidUsernamePassword() Error {
	return &err{level: EXCEPTION, ICode: 13011, IKey: "datastore.couchbase.invalid_username_password",
		InternalMsg: "Invalid username/password.", InternalCaller: CallerN(1)}
}

func NewDatastoreClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13012, IKey: "datastore.couchbase.cluster_error", ICause: e,
		InternalMsg: "Error retrieving cluster " + msg, InternalCaller: CallerN(1)}
}

func NewDatastoreUnableToRetrieveRoles(e error) Error {
	return &err{level: EXCEPTION, ICode: 13013, IKey: "datastore.couchbase.retrieve_roles", ICause: e,
		InternalMsg: "Unable to retrieve roles from server.", InternalCaller: CallerN(1)}
}

func NewDatastoreInsufficientCredentials(msg string) Error {
	return &err{level: EXCEPTION, ICode: 13014, IKey: "datastore.couchbase.insufficient_credentiasl",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}
