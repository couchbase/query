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

func NewVirtualIdxerNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17003, IKey: "datastore.virtual.indexer.not_found", ICause: e,
		InternalMsg: "Virtual Indexer : Indexer not found " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxerNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17004, IKey: "datastore.virtual.indexer.not_supported", ICause: e,
		InternalMsg: "Virtual Indexer : Not supported " + msg, InternalCaller: CallerN(1)}
}

func NewVirtualIdxerNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17005, IKey: "datastore.virtual.indxer.not_implemented", ICause: e,
		InternalMsg: "Virtual indexer : Not yet implemented " + msg, InternalCaller: CallerN(1)}
}
