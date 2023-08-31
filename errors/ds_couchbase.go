//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/value"
)

// Datastore/couchbase error codes
func NewCbConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_CONNECTION, IKey: "datastore.couchbase.connection_error", ICause: e,
		InternalMsg: "Cannot connect " + msg, InternalCaller: CallerN(1)}

}

// Error code 12001 is retired. Do not reuse.

func NewCbNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_NAMESPACE_NOT_FOUND, IKey: "datastore.couchbase.namespace_not_found", ICause: e,
		InternalMsg: "Namespace not found in CB datastore: " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_KEYSPACE_NOT_FOUND, IKey: "datastore.couchbase.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found in CB datastore: " + msg, InternalCaller: CallerN(1)}
}

func NewCbBucketClosedError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_CLOSED, IKey: "datastore.couchbase.bucket_closed",
		InternalMsg: "Bucket is closed: " + msg, InternalCaller: CallerN(1)}
}

func NewCbPrimaryIndexNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_PRIMARY_INDEX_NOT_FOUND, IKey: "datastore.couchbase.primary_idx_not_found", ICause: e,
		InternalMsg: "Primary Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbIndexerNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_INDEXER_NOT_IMPLEMENTED, IKey: "datastore.couchbase.indexer_not_implemented", ICause: e,
		InternalMsg: "Indexer not implemented " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceCountError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_KEYSPACE_COUNT, IKey: "datastore.couchbase.keyspace_count_error", ICause: e,
		InternalMsg: "Failed to get count for keyspace " + msg, InternalCaller: CallerN(1), retry: value.TRUE}
}

// Error code 12007 is retired. Do not reuse.

func NewCbBulkGetError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BULK_GET, IKey: "datastore.couchbase.bulk_get_error", ICause: e,
		InternalMsg: "Error performing bulk get operation " + msg, InternalCaller: CallerN(1), retry: value.TRUE}
}

func NewCbDMLError(e error, msg string, casMismatch bool, r value.Tristate, k string, ks string) Error {
	if casMismatch {
		ce := newCASMismatchError()
		c := ce.Object()
		c["keyspace"] = ks
		c["document_key"] = k
		r = value.FALSE
		return &err{level: ERROR, ICode: E_CB_DML, IKey: "datastore.couchbase.DML_error", ICause: ce, cause: c, retry: r,
			InternalMsg: "DML Error, possible causes include CAS mismatch. " + msg, InternalCaller: CallerN(1)}
	} else {
		return &err{level: ERROR, ICode: E_CB_DML, IKey: "datastore.couchbase.DML_error", ICause: e, cause: e, retry: r,
			InternalMsg: "DML Error, possible causes include concurrent modification. " + msg, InternalCaller: CallerN(1)}
	}
}

// Error code 12010 is retired. Do not reuse.

func NewCbDeleteFailedError(e error, key string, msg string) Error {
	c := make(map[string]interface{})
	c["key"] = key
	c["cause"] = e
	return &err{level: EXCEPTION, ICode: E_CB_DELETE_FAILED, IKey: "datastore.couchbase.delete_failed", ICause: e, cause: c,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewCbLoadIndexesError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_LOAD_INDEXES, IKey: "datastore.couchbase.load_index_failed", ICause: e,
		InternalMsg: "Failed to load indexes " + msg, InternalCaller: CallerN(1)}
}

func NewCbBucketTypeNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_TYPE_NOT_SUPPORTED, IKey: "datastore.couchbase.bucket_type_not_supported", ICause: e,
		InternalMsg: "This bucket type is not supported " + msg, InternalCaller: CallerN(1)}
}

// Error code 12014 is retired. Do not reuse.

func NewCbIndexScanTimeoutError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_INDEX_SCAN_TIMEOUT, IKey: "datastore.couchbase.index_scan_timeout", ICause: e,
		InternalMsg: "Index scan timed out", InternalCaller: CallerN(1)}
}

func NewCbIndexNotFoundError(args ...interface{}) Error {
	var e error
	var name string
	for _, a := range args {
		switch at := a.(type) {
		case error:
			if e == nil {
				e = at
			}
		case string:
			if name == "" {
				name = at
			}
		}
	}
	if name == "" && e != nil {
		s := strings.Split(e.Error(), " ")
		if len(s) > 3 && s[0] == "GSI" {
			name = s[len(s)-3]
		}
	}
	var c map[string]interface{}
	if name != "" {
		c = make(map[string]interface{})
		c["name"] = name
	}
	return &err{level: EXCEPTION, ICode: E_CB_INDEX_NOT_FOUND, IKey: "datastore.couchbase.index_not_found", ICause: e, cause: c,
		InternalMsg: "Index Not Found", InternalCaller: CallerN(1)}
}

func NewCbGetRandomEntryError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_GET_RANDOM_ENTRY, IKey: "datastore.couchbase.get_random_entry_error", ICause: e,
		InternalMsg: "Error getting random entry from keyspace", InternalCaller: CallerN(1)}
}

func NewUnableToInitCbAuthError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_UNABLE_TO_INIT_CB_AUTH, IKey: "datastore.couchbase.unable_to_init_cbauth_error",
		ICause: e, InternalMsg: "Unable to initialize authorization system as required", InternalCaller: CallerN(1)}
}

func NewAuditStreamHandlerFailed(e error) Error {
	return &err{level: EXCEPTION, ICode: E_AUDIT_STREAM_HANDLER_FAILED, IKey: "datastore.couchbase.audit_stream_failed event id", ICause: e,
		InternalMsg: "Audit stream handler failed", InternalCaller: CallerN(1)}
}

func NewCbBucketNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_NOT_FOUND, IKey: "datastore.couchbase.bucket_not_found", ICause: e,
		InternalMsg: "Bucket not found in CB datastore " + msg, InternalCaller: CallerN(1)}
}

func NewCbScopeNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_SCOPE_NOT_FOUND, IKey: "datastore.couchbase.scope_not_found", ICause: e,
		InternalMsg: "Scope not found in CB datastore " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceSizeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_KEYSPACE_SIZE, IKey: "datastore.couchbase.keyspace_size_error", ICause: e,
		InternalMsg: "Failed to get size for keyspace" + msg, InternalCaller: CallerN(1), retry: value.TRUE}
}

func NewCbSecurityConfigNotProvided(bucket string) Error {
	return &err{level: EXCEPTION, ICode: E_CB_SECURITY_CONFIG_NOT_PROVIDED, IKey: "datastore.couchbase.security_config_not_provided",
		InternalMsg: "Connection security config not provided. Unable to load bucket " + bucket, InternalCaller: CallerN(1), retry: value.TRUE}
}

func NewCbCreateSystemBucketError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_CREATE_SYSTEM_BUCKET, IKey: "datastore.couchbase.create_system_bucket", ICause: e,
		InternalMsg: "Error while creating system bucket " + s, InternalCaller: CallerN(1)}
}

func NewCbBucketCreateScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_CREATE_SCOPE, IKey: "datastore.couchbase.create_scope", ICause: e,
		InternalMsg: "Error while creating scope " + s, InternalCaller: CallerN(1)}
}

func NewCbBucketDropScopeError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_DROP_SCOPE, IKey: "datastore.couchbase.drop_scope", ICause: e,
		InternalMsg: "Error while dropping scope " + s, InternalCaller: CallerN(1)}
}

func NewCbBucketCreateCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_CREATE_COLLECTION, IKey: "datastore.couchbase.create_collection", ICause: e,
		InternalMsg: "Error while creating collection " + c, InternalCaller: CallerN(1)}
}

func NewCbBucketDropCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_DROP_COLLECTION, IKey: "datastore.couchbase.drop_collection", ICause: e,
		InternalMsg: "Error while dropping collection " + c, InternalCaller: CallerN(1)}
}

func NewCbBucketFlushCollectionError(c string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_CB_BUCKET_FLUSH_COLLECTION, IKey: "datastore.couchbase.flush_collection", ICause: e,
		InternalMsg: "Error while flushing collection " + c, InternalCaller: CallerN(1)}
}

func NewBinaryDocumentMutationError(op, key string) Error {
	return &err{level: EXCEPTION, ICode: E_BINARY_DOCUMENT_MUTATION, IKey: "mutation.binarydocument.not_supported",
		InternalMsg:    op + " of binary document is not supported: " + key,
		InternalCaller: CallerN(1)}
}

func NewDurabilityNotSupported() Error {
	return &err{level: EXCEPTION, ICode: E_DURABILITY_NOT_SUPPORTED, IKey: "datastore.couchbase.durability",
		InternalMsg:    "Durability is not supported.",
		InternalCaller: CallerN(1)}
}

func NewPreserveExpiryNotSupported() Error {
	return &err{level: EXCEPTION, ICode: E_PRESERVE_EXPIRY_NOT_SUPPORTED, IKey: "datastore.couchbase.preserve_expiration",
		InternalMsg:    "Preserve expiration is not supported.",
		InternalCaller: CallerN(1)}
}

// this is only embedded in 12009

func newCASMismatchError() Error {
	return &err{level: EXCEPTION, ICode: E_CAS_MISMATCH, IKey: "datastore.couchbase.CAS_mismatch",
		InternalMsg: "CAS mismatch", InternalCaller: CallerN(2)} // note caller level
}

func NewCbDMLMCError(s string, k string, ks string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = ks
	c["document_key"] = k
	c["mc_status"] = s
	return &err{level: ERROR, ICode: E_DML_MC, IKey: "datastore.couchbase.mc_error",
		InternalMsg: "MC error " + s, cause: c, InternalCaller: CallerN(1)}
}

func NewCbNotPrimaryIndexError(name string) Error {
	c := make(map[string]interface{})
	c["name"] = name
	c["reason"] = "not primary index"
	return &err{level: EXCEPTION, ICode: E_CB_NOT_PRIMARY_INDEX, IKey: "datastore.couchbase.not_primary_index",
		InternalMsg: "Index " + name + " exists but is not a primary index", cause: c, retry: value.FALSE,
		InternalCaller: CallerN(1)}
}

func NewInsertError(e error, key string) Error {
	c := make(map[string]interface{})
	c["key"] = key
	c["cause"] = e
	return &err{level: ERROR, ICode: E_DML_INSERT, IKey: "datastore.couchbase.insert.error",
		InternalMsg: "Error in INSERT of key: " + key, cause: c, InternalCaller: CallerN(1)}
}

func NewBucketActionError(e interface{}, attempts int) Error {
	c := make(map[string]interface{})
	c["attempts"] = attempts
	c["cause"] = e
	return &err{level: EXCEPTION, ICode: E_BUCKET_ACTION, IKey: "datastore.couchbase.bucket.action",
		InternalMsg: fmt.Sprintf("Unable to complete action after %v attempts", attempts), cause: c, InternalCaller: CallerN(1)}
}
