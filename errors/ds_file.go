//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

// Datastore File based error codes

func NewFileDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_DATASTORE, IKey: "datastore.file.generic_file_error", ICause: e,
		InternalMsg: "Error in file datastore " + msg, InternalCaller: CallerN(1)}
}

func NewFileNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_NAMESPACE_NOT_FOUND, IKey: "datastore.file.namespace_not_found", ICause: e,
		InternalMsg: "Namespace not found in file store " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_KEYSPACE_NOT_FOUND, IKey: "datastore.file.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileDuplicateNamespaceError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_DUPLICATE_NAMESPACE, IKey: "datastore.file.duplicate_namespace", ICause: e,
		InternalMsg: "Duplicate Namespace " + msg, InternalCaller: CallerN(1)}
}

func NewFileDuplicateKeyspaceError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_DUPLICATE_KEYSPACE, IKey: "datastore.file.duplicate_keyspace", ICause: e,
		InternalMsg: "Duplicate Keyspace " + msg, InternalCaller: CallerN(1)}
}

func NewFileNoKeysInsertError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_NO_KEYS_INSERT, IKey: "datastore.file.no_keys_insert", ICause: e,
		InternalMsg: "No keys to insert " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyExists(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_KEY_EXISTS, IKey: "datastore.file.key_exists", ICause: e,
		InternalMsg: "Key Exists " + msg, InternalCaller: CallerN(1)}
}

func NewFileDMLError(e error, msg string) Error {
	return &err{level: ERROR, ICode: E_FILE_DML, IKey: "datastore.file.DML_error", ICause: e,
		InternalMsg: "DML Error " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyspaceNotDirError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_KEYSPACE_NOT_DIR, IKey: "datastore.file.keyspacenot_dir", ICause: e,
		InternalMsg: "Keyspace path must be a directory " + msg, InternalCaller: CallerN(1)}
}

func NewFileIdxNotFound(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_IDX_NOT_FOUND, IKey: "datastore.file.idx_not_found", ICause: e,
		InternalMsg: "Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileNotSupported(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_NOT_SUPPORTED, IKey: "datastore.file.not_supported", ICause: e,
		InternalMsg: "Operation not supported " + msg, InternalCaller: CallerN(1)}
}

func NewFilePrimaryIdxNoDropError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_FILE_PRIMARY_IDX_NO_DROP, IKey: "datastore.file.primary_idx_no_drop", ICause: e,
		InternalMsg: "Primary Index cannot be dropped " + msg, InternalCaller: CallerN(1)}
}
