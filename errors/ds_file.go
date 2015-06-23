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

// Datastore File based error codes

func NewFileDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15000, IKey: "datastore.file.generic_file_error", ICause: e,
		InternalMsg: "Error in file datastore " + msg, InternalCaller: CallerN(1)}
}

func NewFileNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15001, IKey: "datastore.file.namespace_not_found", ICause: e,
		InternalMsg: "Namespace not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15002, IKey: "datastore.file.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileDuplicateNamespaceError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15003, IKey: "datastore.file.duplicate_namespace", ICause: e,
		InternalMsg: "Duplicate Namespace " + msg, InternalCaller: CallerN(1)}
}

func NewFileDuplicateKeyspaceError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15004, IKey: "datastore.file.duplicate_keyspace", ICause: e,
		InternalMsg: "Duplicate Keyspace " + msg, InternalCaller: CallerN(1)}
}

func NewFileNoKeysInsertError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15005, IKey: "datastore.file.no_keys_insert", ICause: e,
		InternalMsg: "No keys to insert " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyExists(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15006, IKey: "datastore.file.key_exists", ICause: e,
		InternalMsg: "Key Exists " + msg, InternalCaller: CallerN(1)}
}

func NewFileDMLError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15007, IKey: "datastore.file.DML_error", ICause: e,
		InternalMsg: "DML Error " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyspaceNotDirError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15008, IKey: "datastore.file.keyspacenot_dir", ICause: e,
		InternalMsg: "Keyspace path must be a directory " + msg, InternalCaller: CallerN(1)}
}

func NewFileIdxNotFound(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15009, IKey: "datastore.file.idx_not_found", ICause: e,
		InternalMsg: "Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileNotSupported(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15010, IKey: "datastore.file.not_supported", ICause: e,
		InternalMsg: "Operation not supported " + msg, InternalCaller: CallerN(1)}
}

func NewFilePrimaryIdxNoDropError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15011, IKey: "datastore.file.primary_idx_no_drop", ICause: e,
		InternalMsg: "Primary Index cannot be dropped " + msg, InternalCaller: CallerN(1)}
}
