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

// System datastore error codes

func NewSystemDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11000, IKey: "datastore.system.generic_error", ICause: e,
		InternalMsg: "System datastore error " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11001, IKey: "datastore.system.namespace_not_found", ICause: e,
		InternalMsg: "Datastore : namespace not found " + msg, InternalCaller: CallerN(1)}

}

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
