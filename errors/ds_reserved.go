//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import "fmt"

// Couchbase datastore path parsing errors

func NewDatastoreInvalidPathError(w string) Error {
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INVALID_BUCKET_PARTS, IKey: "datastore.generic.path_error",
		InternalMsg: "Invalid path specified: " + w, InternalCaller: CallerN(1)}
}

func partsToPath(parts ...string) string {
	switch len(parts) {
	case 0:
		return "''"
	case 1:
		return parts[0]
	default:
		path := parts[0] + ":" + parts[1]
		for i := 2; i < len(parts); i++ {
			path = path + "." + parts[i]
		}
		return path
	}
}

// same ICode as before
func NewDatastoreInvalidBucketPartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INVALID_BUCKET_PARTS, IKey: "datastore.generic.path_error.bucket",
		InternalMsg: "Bucket resolves to " + path + " - 2 path parts are expected: check query_context?", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidScopePartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INVALID_BUCKET_PARTS, IKey: "datastore.generic.path_error.scope",
		InternalMsg: "Scope resolves to " + path + " - 3 path parts are expected.", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidCollectionPartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INVALID_BUCKET_PARTS, IKey: "datastore.generic.path_error.collection",
		InternalMsg: "Collection resolves to " + path + " - 4 path parts are expected: check query_context?", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidKeyspacePartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INVALID_BUCKET_PARTS, IKey: "datastore.generic.path_error.keyspace",
		InternalMsg: "Keyspace resolves to " + path + " - only 2 or 4 parts are valid: check query_context?", InternalCaller: CallerN(1)}
}

func NewQueryContextError(w string) Error {
	if w != "" {
		w = ": " + w
	}
	return &err{level: EXCEPTION, ICode: E_QUERY_CONTEXT, IKey: "datastore.generic.context_error",
		InternalMsg: "Invalid query_context specified: " + w, InternalCaller: CallerN(1)}
}

func NewBucketNoDefaultCollectionError(b string) Error {
	return &err{level: EXCEPTION, ICode: E_BUCKET_NO_DEFAULT_COLLECTION, IKey: "datastore.generic.no_default_collection",
		InternalMsg: "Bucket " + b + " does not have a default collection", InternalCaller: CallerN(1)}
}

func NewNoDatastoreError() Error {
	return &err{level: EXCEPTION, ICode: E_NO_DATASTORE, IKey: "datastore.generic.no_datastore",
		InternalMsg: "No datastore is available", InternalCaller: CallerN(1)}
}

func NewDatastoreNotSetError() Error {
	return &err{level: EXCEPTION, ICode: E_DATASTORE_NOT_SET, IKey: "datastore.not_set",
		InternalMsg: "Datastore not set", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidURIError(uri string) Error {
	return &err{level: EXCEPTION, ICode: E_DATASTORE_INVALID_URI, IKey: "datastore.invalid_uri",
		InternalMsg: fmt.Sprintf("Invalid datastore uri: %s", uri), InternalCaller: CallerN(1)}
}
