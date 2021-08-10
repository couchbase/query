//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package errors

import (
	"fmt"
)

// rewrite errors 17000-17099

func NewTransactionError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 17099, IKey: "transaction_error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

func NewMemoryAllocationError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 17098, IKey: "nomemory",
		InternalMsg:    fmt.Sprintf("Memory allocation error: %s", msg),
		InternalCaller: CallerN(1)}
}

func NewTranCENotsupported() Error {
	return &err{level: EXCEPTION, ICode: 17097, IKey: "transaction.ce.not_supported",
		InternalMsg:    fmt.Sprintf("Transactions are not supported in Community Edition"),
		InternalCaller: CallerN(1)}
}

func NewTranDatastoreNotSupportedError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 17001, IKey: "transaction.datastore.not_supported",
		InternalMsg:    fmt.Sprintf("Transactions are not supported on %s store", msg),
		InternalCaller: CallerN(1)}
}

func NewTranStatementNotSupportedError(stmtType, msg string) Error {
	return &err{level: EXCEPTION, ICode: 17002, IKey: "transaction.statement.not_supported",
		InternalMsg:    fmt.Sprintf("%s statement is not supported %s transaction", stmtType, msg),
		InternalCaller: CallerN(1)}
}

func NewTranFunctionNotSupportedError(fn string) Error {
	return &err{level: EXCEPTION, ICode: 17003, IKey: "transaction.statement.not_supported",
		InternalMsg:    fmt.Sprintf("%s function is not supported within the transaction", fn),
		InternalCaller: CallerN(1)}
}

func NewTransactionContextError(e error) Error {
	return &err{level: EXCEPTION, ICode: 17004, IKey: "transaction.statement.txcontext",
		InternalMsg:    fmt.Sprintf("Transaction context error: %v", e),
		InternalCaller: CallerN(1)}
}

func NewTranStatementOutOfOrderError(prev, cur int64) Error {
	return &err{level: EXCEPTION, ICode: 17005, IKey: "transaction.statement.out_of_order",
		InternalMsg:    fmt.Sprintf("Transaction statement is out of order (%v, %v) ", prev, cur),
		InternalCaller: CallerN(1)}
}

func NewStartTransactionError(e error, c interface{}) Error {
	msg := "Start Transaction statement error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: 17006, IKey: "transaction.statement.start",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewCommitTransactionError(e error, c interface{}) Error {
	msg := "Commit Transaction statement error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: 17007, IKey: "transaction.statement.commit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewRollbackTransactionError(e error, c interface{}) Error {
	msg := "Rollback Transaction statement error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: 17008, IKey: "transaction.statement.rollback",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewNoSavepointError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 17009, IKey: "transaction.statement.nosavepoint",
		InternalMsg:    fmt.Sprintf("%s savepoint is not defined", msg),
		InternalCaller: CallerN(1)}
}

func NewTransactionExpired(c interface{}) Error {
	return &err{level: EXCEPTION, ICode: 17010, IKey: "transaction.expired",
		InternalMsg:    "Transaction timeout",
		InternalCaller: CallerN(1), cause: c}
}

func NewTransactionReleased() Error {
	return &err{level: EXCEPTION, ICode: 17011, IKey: "transaction.released",
		InternalMsg:    "Transaction is released",
		InternalCaller: CallerN(1)}
}

func NewDuplicateKeyError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 17012, IKey: "transaction.statement.duplicatekey",
		InternalMsg:    fmt.Sprintf("Duplicate Key: %s", msg),
		InternalCaller: CallerN(1)}
}

func NewTransactionInuse() Error {
	return &err{level: EXCEPTION, ICode: 17013, IKey: "transaction.parallel.disallowed",
		InternalMsg:    "Parallel execution of the statements are not allowed within the transaction",
		InternalCaller: CallerN(1)}
}

func NewKeyNotFoundError(k string, c interface{}) Error {
	return &err{level: EXCEPTION, ICode: 17014, IKey: "transaction.statement.keynotfound",
		InternalMsg:    fmt.Sprintf("Key not found : %v", k),
		InternalCaller: CallerN(1), cause: c}
}

func NewCasMissmatch(op, key string, aCas, eCas uint64) Error {
	return &err{level: EXCEPTION, ICode: 17015, IKey: "transaction.statement.keynotfound",
		InternalMsg:    fmt.Sprintf("%s cas (actual:%v, expected:%v) missmatch for key: %v", op, aCas, eCas, key),
		InternalCaller: CallerN(1)}
}

func NewTransactionMemoryQuotaExceededError(memQuota, memUsed int64) Error {
	return &err{level: EXCEPTION, ICode: 17016, IKey: "transaction.memory_quota.exceeded",
		InternalMsg:    fmt.Sprintf("Transaction memory (%v) exceeded quota (%v)", memUsed, memQuota),
		InternalCaller: CallerN(1)}
}

func NewTransactionFetchError(e error, c interface{}) Error {
	return &err{level: EXCEPTION, ICode: 17017, IKey: "transaction.fetch", ICause: e,
		InternalMsg:    "Transaction fetch error",
		InternalCaller: CallerN(1), cause: c}
}

func NewPostCommitTransactionError(e error, c interface{}) Error {
	msg := "Failed post commit"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: 17018, IKey: "transaction.statement.postcommit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewAmbiguousCommitTransactionError(e error, c interface{}) Error {
	msg := "Commit was ambiguous"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: 17019, IKey: "transaction.statement.ambiguouscommit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewWriteTransactionError(e error, c interface{}) Error {
	msg := "write error"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: EXCEPTION, ICode: 17020, IKey: "transaction.write.error",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}

func NewTransactionQueueFull() Error {
	return &err{level: EXCEPTION, ICode: 17021, IKey: "transaction.queue.full",
		InternalMsg:    "Transaction queue is full",
		InternalCaller: CallerN(1)}
}

func NewPostCommitTransactionWarning(e error, c interface{}) Error {
	msg := "Failed post commit"
	if e != nil {
		msg = fmt.Sprintf("%s: %v", msg, e)
	}
	return &err{level: WARNING, ICode: 17022, IKey: "transaction.statement.postcommit",
		InternalMsg: msg, InternalCaller: CallerN(1), cause: c}
}
