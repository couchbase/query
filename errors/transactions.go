//  Copyright 2020-Present Couchbase, Inc.
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
)

// rewrite errors 17000-17099

func IsTransactionError(e Error) bool {
	return strings.HasPrefix(e.TranslationKey(), "transaction")
}

func NewTransactionError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: E_TRANSACTION, IKey: "transaction.error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

func NewMemoryAllocationError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_MEMORY_ALLOCATION, IKey: "nomemory",
		InternalMsg:    fmt.Sprintf("Memory allocation error: %s", msg),
		InternalCaller: CallerN(1)}
}

func NewTranCENotsupported() Error {
	return &err{level: EXCEPTION, ICode: E_TRAN_CE_NOTSUPPORTED, IKey: "transaction.ce.not_supported",
		InternalMsg:    fmt.Sprintf("Transactions are not supported in Community Edition"),
		InternalCaller: CallerN(1)}
}

func NewTranDatastoreNotSupportedError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_TRAN_DATASTORE_NOT_SUPPORTED, IKey: "transaction.datastore.not_supported",
		InternalMsg:    fmt.Sprintf("Transactions are not supported on %s store", msg),
		InternalCaller: CallerN(1)}
}

func NewTranStatementNotSupportedError(stmtType, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_TRAN_STATEMENT_NOT_SUPPORTED, IKey: "transaction.statement.not_supported",
		InternalMsg:    fmt.Sprintf("%s statement is not supported %s transaction", stmtType, msg),
		InternalCaller: CallerN(1)}
}

func NewTranFunctionNotSupportedError(fn string) Error {
	return &err{level: EXCEPTION, ICode: E_TRAN_FUNCTION_NOT_SUPPORTED, IKey: "transaction.statement.not_supported",
		InternalMsg:    fmt.Sprintf("%s function is not supported within the transaction", fn),
		InternalCaller: CallerN(1)}
}

func NewTransactionContextError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_CONTEXT, IKey: "transaction.statement.txcontext",
		InternalMsg:    fmt.Sprintf("Transaction context error: %v", e),
		InternalCaller: CallerN(1)}
}

func NewTranStatementOutOfOrderError(prev, cur int64) Error {
	return &err{level: EXCEPTION, ICode: E_TRAN_STATEMENT_OUT_OF_ORDER, IKey: "transaction.statement.out_of_order",
		InternalMsg:    fmt.Sprintf("Transaction statement is out of order (%v, %v) ", prev, cur),
		InternalCaller: CallerN(1)}
}

func NewStartTransactionError(e error, c interface{}) Error {
	msg := "Start Transaction statement error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: E_START_TRANSACTION, IKey: "transaction.statement.start",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewCommitTransactionError(e error, c interface{}) Error {
	msg := "Commit Transaction statement error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: E_COMMIT_TRANSACTION, IKey: "transaction.statement.commit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewRollbackTransactionError(e error, c interface{}) Error {
	msg := "Rollback Transaction statement error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: E_ROLLBACK_TRANSACTION, IKey: "transaction.statement.rollback",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewNoSavepointError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_SAVEPOINT, IKey: "transaction.statement.nosavepoint",
		InternalMsg:    fmt.Sprintf("%s savepoint is not defined", msg),
		InternalCaller: CallerN(1)}
}

func NewTransactionExpired(c interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_EXPIRED, IKey: "transaction.expired",
		InternalMsg:    "Transaction timeout",
		InternalCaller: CallerN(1), cause: c}
}

func NewTransactionReleased() Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_RELEASED, IKey: "transaction.released",
		InternalMsg:    "Transaction is released",
		InternalCaller: CallerN(1)}
}

func NewDuplicateKeyError(key, ks string, c interface{}) Error {
	msg := ""
	if ks != "" {
		msg = fmt.Sprintf(" for '%s'", ks)
	}
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_KEY, IKey: "dml.statement.duplicatekey",
		InternalMsg:    fmt.Sprintf("Duplicate Key%s: %s", msg, key),
		InternalCaller: CallerN(1), cause: c}
}

func NewTransactionInuse() Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_INUSE, IKey: "transaction.parallel.disallowed",
		InternalMsg:    "Parallel execution of the statements are not allowed within the transaction",
		InternalCaller: CallerN(1)}
}

func NewKeyNotFoundError(key, ks string, c interface{}) Error {
	msg := ""
	if ks != "" {
		msg = fmt.Sprintf(" for '%s'", ks)
	}
	return &err{level: EXCEPTION, ICode: E_KEY_NOT_FOUND, IKey: "datastore.couchbase.keynotfound",
		InternalMsg:    fmt.Sprintf("Key not found%s: %v", msg, key),
		InternalCaller: CallerN(1), cause: c}
}

func NewScasMismatch(op, key string, aCas, eCas uint64) Error {
	return &err{level: EXCEPTION, ICode: E_SCAS_MISMATCH, IKey: "transaction.statement.scasmismatch",
		InternalMsg:    fmt.Sprintf("%s cas (actual:%v, expected:%v) mismatch for key: %v", op, aCas, eCas, key),
		InternalCaller: CallerN(1)}
}

func NewTransactionMemoryQuotaExceededError(memQuota, memUsed int64) Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_MEMORY_QUOTA_EXCEEDED, IKey: "transaction.memory_quota.exceeded",
		InternalMsg:    fmt.Sprintf("Transaction memory (%v) exceeded quota (%v)", memUsed, memQuota),
		InternalCaller: CallerN(1)}
}

func NewTransactionFetchError(e error, c interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_FETCH, IKey: "transaction.fetch", ICause: e,
		InternalMsg:    "Transaction fetch error",
		InternalCaller: CallerN(1), cause: c}
}

func NewPostCommitTransactionError(e error, c interface{}) Error {
	msg := "Failed post commit"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: E_POST_COMMIT_TRANSACTION, IKey: "transaction.statement.postcommit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewAmbiguousCommitTransactionError(e error, c interface{}) Error {
	msg := "Commit was ambiguous"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: E_AMBIGUOUS_COMMIT_TRANSACTION, IKey: "transaction.statement.ambiguouscommit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewTransactionStagingError(e error, c interface{}) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		msg := "Transaction staging error"
		if e != nil {
			msg = fmt.Sprintf("%s: %v", msg, e)
		}
		return &err{level: EXCEPTION, ICode: E_TRANSACTION_STAGING, IKey: "transaction.staging.error",
			InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
	}
}

func NewTransactionQueueFull() Error {
	return &err{level: EXCEPTION, ICode: E_TRANSACTION_QUEUE_FULL, IKey: "transaction.queue.full",
		InternalMsg:    "Transaction queue is full",
		InternalCaller: CallerN(1)}
}

func NewPostCommitTransactionWarning(e error, c interface{}) Error {
	msg := "Failed post commit"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: WARNING, ICode: W_POST_COMMIT_TRANSACTION, IKey: "transaction.statement.postcommit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewGCAgentError(e error, op string) Error {
	c := make(map[string]interface{}, 2)
	c["operation"] = op
	c["error"] = e
	return &err{level: EXCEPTION, ICode: E_GC_AGENT, IKey: "transaction.gcagent",
		InternalMsg: "GC agent error", InternalCaller: CallerN(1), cause: c}
}
