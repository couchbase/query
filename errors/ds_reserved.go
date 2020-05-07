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

const DS_BAD_CONTEXT = 10101

func NewQueryContextError(w string) Error {
	if w != "" {
		w = ": " + w
	}
	return &err{level: EXCEPTION, ICode: DS_BAD_CONTEXT, IKey: "datastore.generic.context_error",
		InternalMsg: "Invalid query context specified" + w, InternalCaller: CallerN(1)}
}
