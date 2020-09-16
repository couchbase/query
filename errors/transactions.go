//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
		InternalMsg:    fmt.Sprintf("Memory allocation errorr: %s", msg),
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

func NewStartTransactionError(e error) Error {
	return &err{level: EXCEPTION, ICode: 17006, IKey: "transaction.statement.start",
		InternalMsg:    fmt.Sprintf("Start Transaction statement error: %v", e),
		InternalCaller: CallerN(1)}
}

func NewCommitTransactionError(e error, diagnostics interface{}) Error {
	return &err{level: EXCEPTION, ICode: 17007, IKey: "transaction.statement.commit",
		InternalMsg:    fmt.Sprintf("Commit Transaction statement error: %v", e),
		InternalCaller: CallerN(1), diagnostics: diagnostics}
}

func NewRollbackTransactionError(e error, diagnostics interface{}) Error {
	return &err{level: EXCEPTION, ICode: 17008, IKey: "transaction.statement.rollback",
		InternalMsg:    fmt.Sprintf("Rollback Transaction statement error: %v", e),
		InternalCaller: CallerN(1), diagnostics: diagnostics}
}

func NewNoSavepointError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 17009, IKey: "transaction.statement.nosavepoint",
		InternalMsg:    fmt.Sprintf("%s savepoint is not defined", msg),
		InternalCaller: CallerN(1)}
}

func NewTransactionExpired() Error {
	return &err{level: EXCEPTION, ICode: 17010, IKey: "transaction.expired",
		InternalMsg:    "Transaction timeout",
		InternalCaller: CallerN(1)}
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

func NewTransactionQueueFull() Error {
	return &err{level: EXCEPTION, ICode: 17013, IKey: "transaction.queue.full",
		InternalMsg:    "Transaction queue is full",
		InternalCaller: CallerN(1)}
}

func NewTransactionInuse() Error {
	return &err{level: EXCEPTION, ICode: 17013, IKey: "transaction.parallel.disallowed",
		InternalMsg:    "Parallel execution of the statements are not allowed within the transaction",
		InternalCaller: CallerN(1)}
}
