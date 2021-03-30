//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package gcagent

import (
	"errors"
)

var (
	// indicates transactions are not supported in ce
	ErrCENotSupported = errors.New("transactions are not supported in community edition")

	// indicates init transactions is not called
	ErrNoInitTransactions = errors.New("init transactions is not called")

	// indicates transaction is not started
	ErrNoTransaction = errors.New("transaction is not started")

	// indicates SubDoc operations are performed in the transaction
	ErrNoSubDocInTransaction = errors.New("SubDoc operations are not supported in the transaction")

	// indicates not supported operation
	ErrUnknownOperation = errors.New("Operation is not supported in the transaction")

	// indicates compression value
	ErrCompression = errors.New("unexpected value compression")

	// indicates cas missmatch
	ErrCasMissmatch = errors.New("cas missmatch")
)
