//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

func NewVirtualKSNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17000, IKey: "datastore.virtual.keyspace.not_supported", ICause: e,
		InternalMsg: "Virtual Keyspace : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualKSNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17001, IKey: "datastore.virtual.keyspace.not_implemented", ICause: e,
		InternalMsg: "Virtual Keyspace : Not yet implemented " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualKSIdxerNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17002, IKey: "datastore.virtual.keyspace.not_found", ICause: e,
		InternalMsg: "Virtual keyspace : Indexer not found " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17003, IKey: "datastore.virtual.indexer.not_found", ICause: e,
		InternalMsg: "Virtual indexer : Index not found " + msg, InternalCaller: CallerN(1)}

}

func NewVirtualIdxerNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17004, IKey: "datastore.virtual.indexer.not_supported", ICause: e,
		InternalMsg: "Virtual Indexer : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17005, IKey: "datastore.virtual.index.not_implemented", ICause: e,
		InternalMsg: "Virtual index : Not yet implemented " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17006, IKey: "datastore.virtual.index.not_supported", ICause: e,
		InternalMsg: "Virtual Index : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualScopeNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17007, IKey: "datastore.virtual.scope_not_found", ICause: e,
		InternalMsg: "Scope not found in virtual datastore " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualBucketNoDefaultCollectionError(b string) Error {
	return &err{level: EXCEPTION, ICode: 17008, IKey: "datastore.virtual.no_default_collection",
		InternalMsg: "Bucket " + b + " does not have a default collection", InternalCaller: CallerN(1)}
}

func NewVirtualBucketCreateScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 17009, IKey: "datastore.virtual.create_scope", ICause: e,
		InternalMsg: "Error while creating scope " + s, InternalCaller: CallerN(1)}
}

func NewVirtualBucketDropScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 17010, IKey: "datastore.virtual.drop_scope", ICause: e,
		InternalMsg: "Error while dropping scope " + s, InternalCaller: CallerN(1)}
}

func NewVirtualKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17011, IKey: "datastore.virtual.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found in CB datastore: " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualBucketCreateCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 17012, IKey: "datastore.virtual.create_collection", ICause: e,
		InternalMsg: "Error while creating collection " + c, InternalCaller: CallerN(1)}
}

func NewVirtualBucketDropCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 17013, IKey: "datastore.virtual.drop_collection", ICause: e,
		InternalMsg: "Error while dropping collection " + c, InternalCaller: CallerN(1)}
}
