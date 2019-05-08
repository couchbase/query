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

// System datastore error codes

func NewSystemDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11000, IKey: "datastore.system.generic_error", ICause: e,
		InternalMsg: "System datastore error " + msg, InternalCaller: CallerN(1)}

}

// Error code 11011 is retired. Do not reuse.

func NewSystemKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11002, IKey: "datastore.system.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11003, IKey: "datastore.system.not_implemented", ICause: e,
		InternalMsg: "System datastore :  Not implemented " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11004, IKey: "datastore.system.not_supported", ICause: e,
		InternalMsg: "System datastore : Not supported " + msg, InternalCaller: CallerN(1)}

}

func NewSystemIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11005, IKey: "datastore.system.idx_not_found", ICause: e,
		InternalMsg: "System datastore : Index not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemIdxNoDropError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11006, IKey: "datastore.system.idx_no_drop", ICause: e,
		InternalMsg: "System datastore : This  index cannot be dropped " + msg, InternalCaller: CallerN(1)}
}

func NewSystemStmtNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11007, IKey: "datastore.system.stmt_not_found", ICause: e,
		InternalMsg: "System datastore : Statement not found " + msg, InternalCaller: CallerN(1)}
}

func NewSystemRemoteWarning(e error, op string, ks string) Error {
	return &err{level: WARNING, ICode: 11008, IKey: "datastore.system.remote_warning", ICause: e,
		InternalMsg: "System datastore : " + op + " on " + ks + " failed", InternalCaller: CallerN(1)}
}

func NewSystemUnableToRetrieveError(e error) Error {
	return &err{level: EXCEPTION, ICode: 11009, IKey: "datastore.system.unable_to_retrieve", ICause: e,
		InternalMsg: "System datastore : unable to retrieve user roles from server", InternalCaller: CallerN(1), retry: true}
}

func NewSystemUnableToUpdateError(e error) Error {
	return &err{level: EXCEPTION, ICode: 11010, IKey: "datastore.system.unable_to_update", ICause: e,
		InternalMsg: "System datastore : unable to update user information in server", InternalCaller: CallerN(1)}
}

func NewSystemFilteredRowsWarning(keyspace string) Error {
	return &err{level: WARNING, ICode: 11011, IKey: "datastore.system.filtered_keyspaces", onceOnly: true,
		InternalMsg:    fmt.Sprintf("One or more documents were excluded from the %s bucket because of insufficient user permissions.", keyspace),
		InternalCaller: CallerN(1)}
}

func NewSystemMalformedKeyError(key string, keyspace string) Error {
	return &err{level: EXCEPTION, ICode: 11012, IKey: "datastore.system.malformed_key",
		InternalMsg:    fmt.Sprintf("System datastore : key %q is not of the correct format for keyspace %s", key, keyspace),
		InternalCaller: CallerN(1)}
}

func NewSystemNoBuckets() Error {
	return &err{level: EXCEPTION, ICode: 11013, IKey: "datastore.system.no_buckets",
		InternalMsg:    "The system namespace contains no buckets that contain scopes.",
		InternalCaller: CallerN(1)}
}
