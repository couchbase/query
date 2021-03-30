//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package errors

// Couchbase datastore path parsing errors

const DS_BAD_PATH = 10200

func NewDatastoreInvalidPathError(w string) Error {
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error",
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
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error.bucket",
		InternalMsg: "Bucket resolves to " + path + " - 2 path parts are expected: check query_context?", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidScopePartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error.scope",
		InternalMsg: "Scope resolves to " + path + " - 3 path parts are expected.", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidCollectionPartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error.collection",
		InternalMsg: "Collection resolves to " + path + " - 4 path parts are expected: check query_context?", InternalCaller: CallerN(1)}
}

func NewDatastoreInvalidKeyspacePartsError(parts ...string) Error {
	path := partsToPath(parts...)
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error.keyspace",
		InternalMsg: "Keyspace resolves to " + path + " - only 2 or 4 parts are valid: check query_context?", InternalCaller: CallerN(1)}
}

const DS_BAD_CONTEXT = 10201

func NewQueryContextError(w string) Error {
	if w != "" {
		w = ": " + w
	}
	return &err{level: EXCEPTION, ICode: DS_BAD_CONTEXT, IKey: "datastore.generic.context_error",
		InternalMsg: "Invalid query_context specified: " + w, InternalCaller: CallerN(1)}
}

const DS_NO_DEFAULT_COLLECTION = 10202

func NewBucketNoDefaultCollectionError(b string) Error {
	return &err{level: EXCEPTION, ICode: DS_NO_DEFAULT_COLLECTION, IKey: "datastore.generic.no_default_collection",
		InternalMsg: "Bucket " + b + " does not have a default collection", InternalCaller: CallerN(1)}
}

const DS_NO_DATASTORE = 10203

func NewNoDatastoreError() Error {
	return &err{level: EXCEPTION, ICode: DS_NO_DATASTORE, IKey: "datastore.generic.no_datastore",
		InternalMsg: "No datastore is available", InternalCaller: CallerN(1)}
}
