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
)

// System datastore error codes

func NewSystemDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_DATASTORE, IKey: "datastore.system.generic_error", ICause: e,
		InternalMsg: "System datastore error " + msg, InternalCaller: CallerN(1)}

}

// Error code 11011 is retired. Do not reuse.

func NewSystemKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_KEYSPACE_NOT_FOUND, IKey: "datastore.system.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_NOT_IMPLEMENTED, IKey: "datastore.system.not_implemented", ICause: e,
		InternalMsg: "System datastore :  Not implemented " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_NOT_SUPPORTED, IKey: "datastore.system.not_supported", ICause: e,
		InternalMsg: "System datastore : Not supported " + msg, InternalCaller: CallerN(1)}

}

func NewSystemIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_IDX_NOT_FOUND, IKey: "datastore.system.idx_not_found", ICause: e,
		InternalMsg: "System datastore : Index not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemIdxNoDropError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_IDX_NO_DROP, IKey: "datastore.system.idx_no_drop", ICause: e,
		InternalMsg: "System datastore : This  index cannot be dropped " + msg, InternalCaller: CallerN(1)}
}

func NewSystemStmtNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_STMT_NOT_FOUND, IKey: "datastore.system.stmt_not_found", ICause: e,
		InternalMsg: "System datastore : Statement not found " + msg, InternalCaller: CallerN(1)}
}

func NewSystemRemoteWarning(e error, op string, ks string) Error {
	return &err{level: WARNING, ICode: E_SYSTEM_REMOTE_WARNING, IKey: "datastore.system.remote_warning", ICause: e,
		InternalMsg: "System datastore : " + op + " on " + ks + " failed", InternalCaller: CallerN(1)}
}

func NewSystemRemoteNodeSkippedWarning(node, op string, ks string) Error {
	return &err{level: WARNING, ICode: E_SYSTEM_REMOTE_WARNING, IKey: "datastore.system.remote_warning", ICause: fmt.Errorf("unhealthy %v", node),
		InternalMsg: "System datastore : skipping unheathy node " + node + " for " + op + " on " + ks + " failed", InternalCaller: CallerN(1)}
}

func NewSystemUnableToRetrieveError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_UNABLE_TO_RETRIEVE, IKey: "datastore.system.unable_to_retrieve", ICause: e,
		InternalMsg: "System datastore : unable to retrieve user roles from server", InternalCaller: CallerN(1), retry: TRUE}
}

func NewSystemUnableToUpdateError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_UNABLE_TO_UPDATE, IKey: "datastore.system.unable_to_update", ICause: e,
		InternalMsg: "System datastore : unable to update user information in server", InternalCaller: CallerN(1)}
}

func NewSystemFilteredRowsWarning(keyspace string) Error {
	return &err{level: WARNING, ICode: E_SYSTEM_FILTERED_ROWS_WARNING, IKey: "datastore.system.filtered_keyspaces", onceOnly: true,
		InternalMsg: fmt.Sprintf("One or more documents were excluded from the %s bucket because of insufficient user permissions. "+
			"In an EE system, add the query_system_catalog role to see all rows. In a CE system, add the administrator role to see all rows.", keyspace),
		InternalCaller: CallerN(1)}
}

func NewSystemMalformedKeyError(key string, keyspace string) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_MALFORMED_KEY, IKey: "datastore.system.malformed_key",
		InternalMsg:    fmt.Sprintf("System datastore : key %q is not of the correct format for keyspace %s", key, keyspace),
		InternalCaller: CallerN(1)}
}

func NewSystemNoBuckets() Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_NO_BUCKETS, IKey: "datastore.system.no_buckets",
		InternalMsg:    "The system namespace contains no buckets that contain scopes.",
		InternalCaller: CallerN(1)}
}

func NewInvalidPreparedAdminOp() Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_PREPARED_ADMIN_OP, IKey: "datastore.system.remote.invalid_prepared_op",
		ICause: nil, InternalMsg: "System datastore : Invalid prepared admin operation", InternalCaller: CallerN(1)}
}
