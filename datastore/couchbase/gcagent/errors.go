//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
