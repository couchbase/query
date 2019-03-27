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

// Error codes for all other datastores, e.g Mock

func NewOtherDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16000, IKey: "datastore.other.datastore_generic_error", ICause: e,
		InternalMsg: "Error in datastore " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16001, IKey: "datastore.other.namespace_not_found", ICause: e,
		InternalMsg: "Namespace Not Found " + msg, InternalCaller: CallerN(1)}
}

func NewOtherKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16002, IKey: "datastore.other.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace Not Found " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16003, IKey: "datastore.other.not_implemented", ICause: e,
		InternalMsg: "Not Implemented " + msg, InternalCaller: CallerN(1)}
}

func NewOtherIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16004, IKey: "datastore.other.idx_not_found", ICause: e,
		InternalMsg: "Index not found  " + msg, InternalCaller: CallerN(1)}
}

func NewOtherIdxNoDrop(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16005, IKey: "datastore.other.idx_no_drop", ICause: e,
		InternalMsg: "Index Cannot be dropped " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16006, IKey: "datastore.other.not_supported", ICause: e,
		InternalMsg: "Not supported for this datastore " + msg, InternalCaller: CallerN(1)}
}

func NewOtherKeyNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16007, IKey: "datastore.other.key_not_found", ICause: e,
		InternalMsg: "Key not found " + msg, InternalCaller: CallerN(1)}
}

func NewInferencerNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16020, IKey: "datastore.other.inferencer_not_found", ICause: e,
		InternalMsg: "Inferencer not found " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNoBuckets(dsName string) Error {
	return &err{level: EXCEPTION, ICode: 16021, IKey: "datastore.other.no_buckets",
		InternalMsg: "Datastore " + dsName + "contains no buckets that contain scopes.", InternalCaller: CallerN(1)}
}

func NewStatUpdaterNotFoundError(e error) Error {
	return &err{level: EXCEPTION, ICode: 16030, IKey: "datastore.other.statUpdater_not_found", ICause: e,
		InternalMsg: "StatUpdater not found", InternalCaller: CallerN(1)}
}
