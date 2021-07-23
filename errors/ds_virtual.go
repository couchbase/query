//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package errors

func NewVirtualKSNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18000, IKey: "datastore.virtual.keyspace.not_supported", ICause: e,
		InternalMsg: "Virtual Keyspace : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualKSNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18001, IKey: "datastore.virtual.keyspace.not_implemented", ICause: e,
		InternalMsg: "Virtual Keyspace : Not yet implemented " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualKSIdxerNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18002, IKey: "datastore.virtual.keyspace.not_found", ICause: e,
		InternalMsg: "Virtual keyspace : Indexer not found " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18003, IKey: "datastore.virtual.indexer.not_found", ICause: e,
		InternalMsg: "Virtual indexer : Index not found " + msg, InternalCaller: CallerN(1)}

}

func NewVirtualIdxerNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18004, IKey: "datastore.virtual.indexer.not_supported", ICause: e,
		InternalMsg: "Virtual Indexer : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18005, IKey: "datastore.virtual.index.not_implemented", ICause: e,
		InternalMsg: "Virtual index : Not yet implemented " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18006, IKey: "datastore.virtual.index.not_supported", ICause: e,
		InternalMsg: "Virtual Index : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualScopeNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18007, IKey: "datastore.virtual.scope_not_found", ICause: e,
		InternalMsg: "Scope not found in virtual datastore " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualBucketCreateScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 18009, IKey: "datastore.virtual.create_scope", ICause: e,
		InternalMsg: "Error while creating scope " + s, InternalCaller: CallerN(1)}
}

func NewVirtualBucketDropScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 18010, IKey: "datastore.virtual.drop_scope", ICause: e,
		InternalMsg: "Error while dropping scope " + s, InternalCaller: CallerN(1)}
}

func NewVirtualKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 18011, IKey: "datastore.virtual.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found in CB datastore: " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualBucketCreateCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 18012, IKey: "datastore.virtual.create_collection", ICause: e,
		InternalMsg: "Error while creating collection " + c, InternalCaller: CallerN(1)}
}

func NewVirtualBucketDropCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 18013, IKey: "datastore.virtual.drop_collection", ICause: e,
		InternalMsg: "Error while dropping collection " + c, InternalCaller: CallerN(1)}
}
