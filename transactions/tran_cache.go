//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package transactions

import (
	go_atomic "sync/atomic"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

type TranContextCache struct {
	cache         *util.GenCache
	cleanupIntrvl time.Duration
	context       datastore.QueryContext
}

var tranContextCache = &TranContextCache{}

func TranContextCacheInit(intrvl time.Duration) {
	tranContextCache.cache = util.NewGenCache(-1)
	tranContextCache.cleanupIntrvl = intrvl
	go tranContextCache.doTransactionCleanup()
}

func AddTransContext(txContext *TranContext) errors.Error {
	return tranContextCache.add(txContext, true)
}

func GetTransContext(txId string) *TranContext {
	return tranContextCache.get(txId, true)
}

func SetQueryContext(context datastore.QueryContext) {
	tranContextCache.context = context
}

func DeleteTransContext(txId string, userDelete bool) errors.Error {
	if txId != "" {
		txContext := tranContextCache.get(txId, false)
		if txContext != nil {
			if userDelete {
				txContext.SetTxExpiry()
				return nil
			}

			tranContextCache.cache.Delete(txId, nil)
			txContext.SetTxStatus(TX_RELEASED)
		}
	}
	return nil
}

func CountTransContext() int {
	return tranContextCache.cache.Size()
}

func NameTransactions() []string {
	return tranContextCache.cache.Names()
}

func TransactionEntryDo(txId string, f func(interface{})) {
	tranContextCache.cache.Get(txId, f)
}

func TransactionEntriesForeach(nonBlocking func(string, interface{}) bool,
	blocking func() bool) {
	tranContextCache.cache.ForEach(nonBlocking, blocking)
}

func (this *TranContextCache) add(txContext *TranContext, track bool) errors.Error {
	if track {
		txContext.uses = 1
		txContext.lastUse = time.Now()
	}
	this.cache.FastAdd(txContext, txContext.TxId())
	return nil
}

func (this *TranContextCache) get(txId string, track bool) *TranContext {
	var cv interface{}

	if track {
		cv = this.cache.Use(txId, nil)
	} else {
		cv = this.cache.Get(txId, nil)
	}
	rv, ok := cv.(*TranContext)
	if ok && track {
		go_atomic.AddInt32(&rv.uses, 1)
		rv.lastUse = time.Now()
	}
	return rv
}

func (this *TranContextCache) doTransactionCleanup() {
	defer func() {
		recover()
		go this.doTransactionCleanup()
	}()

	data := make([]*TranContext, 0, tranContextCache.cache.Size())
	for {
		data = data[0:0]
		time.Sleep(this.cleanupIntrvl)

		context := tranContextCache.context
		ds := context.Datastore()

		snapshot := func(name string, d interface{}) bool {
			tranContext := d.(*TranContext)
			if tranContext.TxExpired() {
				if !tranContext.TxInUse() {
					data = append(data, tranContext)
				} else if tranContext.TxTimeRemaining() < _TX_CLEANUP_AFTER {
					// reset inuse expired more than minute back
					tranContext.SetTxInUse(false)
					data = append(data, tranContext)
				}
			}
			return true
		}

		tranContextCache.cache.ForEach(snapshot, nil)
		for _, tranContext := range data {
			if tranContext.TxExpired() && !tranContext.TxInUse() {
				tranContextCache.cache.Delete(tranContext.TxId(), nil)
				if context != nil && ds != nil {
					context.SetTxContext(tranContext)
					ds.RollbackTransaction(false, context, "")
				}
				tranContext.SetTxStatus(TX_RELEASED)
			}
		}
	}
}
