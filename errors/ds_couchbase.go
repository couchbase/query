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

// Datastore/couchbase error codes
func NewCbConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12000, IKey: "datastore.couchbase.connection_error", ICause: e,
		InternalMsg: "Cannot connect " + msg, InternalCaller: CallerN(1)}

}

// Error code 12001 is retired. Do not reuse.

func NewCbNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12002, IKey: "datastore.couchbase.namespace_not_found", ICause: e,
		InternalMsg: "Namespace not found in CB datastore: " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12003, IKey: "datastore.couchbase.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found in CB datastore: " + msg, InternalCaller: CallerN(1)}
}

func NewCbPrimaryIndexNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12004, IKey: "datastore.couchbase.primary_idx_not_found", ICause: e,
		InternalMsg: "Primary Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbIndexerNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12005, IKey: "datastore.couchbase.indexer_not_implemented", ICause: e,
		InternalMsg: "Indexer not implemented " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceCountError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12006, IKey: "datastore.couchbase.keyspace_count_error", ICause: e,
		InternalMsg: "Failed to get count for keyspace " + msg, InternalCaller: CallerN(1), retry: true}
}

// Error code 12007 is retired. Do not reuse.

func NewCbBulkGetError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12008, IKey: "datastore.couchbase.bulk_get_error", ICause: e,
		InternalMsg: "Error performing bulk get operation " + msg, InternalCaller: CallerN(1), retry: true}
}

func NewCbDMLError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12009, IKey: "datastore.couchbase.DML_error", ICause: e,
		InternalMsg: "DML Error, possible causes include CAS mismatch or concurrent modification" + msg, InternalCaller: CallerN(1)}
}

// Error code 12010 is retired. Do not reuse.

func NewCbDeleteFailedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12011, IKey: "datastore.couchbase.delete_failed", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewCbLoadIndexesError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12012, IKey: "datastore.couchbase.load_index_failed", ICause: e,
		InternalMsg: "Failed to load indexes " + msg, InternalCaller: CallerN(1)}
}

func NewCbBucketTypeNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12013, IKey: "datastore.couchbase.bucket_type_not_supported", ICause: e,
		InternalMsg: "This bucket type is not supported " + msg, InternalCaller: CallerN(1)}
}

// Error code 12014 is retired. Do not reuse.

const INDEX_SCAN_TIMEOUT = 12015

func NewCbIndexScanTimeoutError(e error) Error {
	return &err{level: EXCEPTION, ICode: INDEX_SCAN_TIMEOUT, IKey: "datastore.couchbase.index_scan_timeout", ICause: e,
		InternalMsg: "Index scan timed out", InternalCaller: CallerN(1)}
}

const INDEX_NOT_FOUND = 12016

func NewCbIndexNotFoundError(e error) Error {
	return &err{level: EXCEPTION, ICode: INDEX_NOT_FOUND, IKey: "datastore.couchbase.index_not_found", ICause: e,
		InternalMsg: "Index Not Found", InternalCaller: CallerN(1), retry: true}
}

const GET_RANDOM_ENTRY_ERROR = 12017

func NewCbGetRandomEntryError(e error) Error {
	return &err{level: EXCEPTION, ICode: GET_RANDOM_ENTRY_ERROR, IKey: "datastore.couchbase.get_random_entry_error", ICause: e,
		InternalMsg: "Error getting random entry from keyspace", InternalCaller: CallerN(1)}
}

func NewUnableToInitCbAuthError(e error) Error {
	return &err{level: EXCEPTION, ICode: 12018, IKey: "datastore.couchbase.unable_to_init_cbauth_error", ICause: e,
		InternalMsg: "Unable to initialize authorization system as required", InternalCaller: CallerN(1)}
}

func NewAuditStreamHandlerFailed(e error) Error {
	return &err{level: EXCEPTION, ICode: 12019, IKey: "datastore.couchbase.audit_stream_failed event id", ICause: e,
		InternalMsg: "Audit stream handler failed", InternalCaller: CallerN(1)}
}

func NewCbBucketNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12020, IKey: "datastore.couchbase.bucket_not_found", ICause: e,
		InternalMsg: "Bucket not found in CB datastore " + msg, InternalCaller: CallerN(1)}
}

func NewCbScopeNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12021, IKey: "datastore.couchbase.scope_not_found", ICause: e,
		InternalMsg: "Scope not found in CB datastore " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceSizeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12022, IKey: "datastore.couchbase.keyspace_size_error", ICause: e,
		InternalMsg: "Failed to get size for keyspace" + msg, InternalCaller: CallerN(1), retry: true}
}

func NewCbSecurityConfigNotProvided(bucket string) Error {
	return &err{level: EXCEPTION, ICode: 12023, IKey: "datastore.couchbase.security_config_not_provided",
		InternalMsg: "Connection security config not provided. Unable to load bucket " + bucket, InternalCaller: CallerN(1), retry: true}
}

func NewCbCreateSystemBucketError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 12024, IKey: "datastore.couchbase.create_system_bucket", ICause: e,
		InternalMsg: "Error while creating system bucket " + s, InternalCaller: CallerN(1)}
}

func NewCbBucketCreateScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 12025, IKey: "datastore.couchbase.create_scope", ICause: e,
		InternalMsg: "Error while creating scope " + s, InternalCaller: CallerN(1)}
}

func NewCbBucketDropScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: 12026, IKey: "datastore.couchbase.drop_scope", ICause: e,
		InternalMsg: "Error while dropping scope " + s, InternalCaller: CallerN(1)}
}

func NewCbBucketCreateCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 12027, IKey: "datastore.couchbase.create_collection", ICause: e,
		InternalMsg: "Error while creating collection " + c, InternalCaller: CallerN(1)}
}

func NewCbBucketDropCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 12028, IKey: "datastore.couchbase.drop_collection", ICause: e,
		InternalMsg: "Error while dropping collection " + c, InternalCaller: CallerN(1)}
}

func NewCbBucketFlushCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: 12029, IKey: "datastore.couchbase.flush_collection", ICause: e,
		InternalMsg: "Error while flushing collection " + c, InternalCaller: CallerN(1)}
}

func NewBinaryDocumentMutationError(op, key string) Error {
	return &err{level: EXCEPTION, ICode: 12030, IKey: "mutation.binarydocument.not_supported",
		InternalMsg:    op + " of binary document is not supported: " + key,
		InternalCaller: CallerN(1)}
}
