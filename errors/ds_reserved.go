//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

// Couchbase datastore path parsing errors

const DS_BAD_PATH = 10100

func NewDatastoreInvalidPathError(w string) Error {
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error",
		InternalMsg: "Invalid path specified: " + w, InternalCaller: CallerN(1)}
}

// same ICode as before
func NewDatastoreInvalidPathPartsError(parts ...string) Error {
	var path string

	switch len(parts) {
	case 0:
		path = "path"
	case 1:
		path = parts[0]
	default:
		path = parts[0] + ":" + parts[1]
		for i := 2; i < len(parts); i++ {
			path = path + "." + parts[i]
		}
	}
	return &err{level: EXCEPTION, ICode: DS_BAD_PATH, IKey: "datastore.generic.path_error",
		InternalMsg: "Invalid path specified: " + path + " has invalid number of parts", InternalCaller: CallerN(1)}
}

const DS_BAD_CONTEXT = 10101

func NewQueryContextError(w string) Error {
	if w != "" {
		w = ": " + w
	}
	return &err{level: EXCEPTION, ICode: DS_BAD_CONTEXT, IKey: "datastore.generic.context_error",
		InternalMsg: "Invalid query context specified" + w, InternalCaller: CallerN(1)}
}

const DS_NO_DEFAULT_COLLECTION = 10102

func NewBucketNoDefaultCollectionError(b string) Error {
	return &err{level: EXCEPTION, ICode: DS_NO_DEFAULT_COLLECTION, IKey: "datastore.generic.no_default_collection",
		InternalMsg: "Bucket " + b + " does not have a default collection", InternalCaller: CallerN(1)}
}

const DS_NO_DATASTORE = 10103

func NewNoDatastoreError() Error {
	return &err{level: EXCEPTION, ICode: DS_NO_DATASTORE, IKey: "datastore.generic.no_datastore",
		InternalMsg: "No datastore is available", InternalCaller: CallerN(1)}
}
