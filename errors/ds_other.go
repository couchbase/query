//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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

func NewScopesNotSupportedError(k string) Error {
	return &err{level: EXCEPTION, ICode: 16022, IKey: "datastore.other.no_spopes",
		InternalMsg: "Keyspace does not support scopes: " + k, InternalCaller: CallerN(1)}
}

func NewStatUpdaterNotFoundError(e error) Error {
	return &err{level: EXCEPTION, ICode: 16030, IKey: "datastore.other.statUpdater_not_found", ICause: e,
		InternalMsg: "StatUpdater not found", InternalCaller: CallerN(1)}
}

func NewNoFlushError(k string) Error {
	return &err{level: EXCEPTION, ICode: 16040, IKey: "datastore.other.flush_disabled",
		InternalMsg: "Keyspace does not support flush: " + k, InternalCaller: CallerN(1)}
}
