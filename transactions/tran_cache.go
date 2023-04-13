//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func CountValidTransContextBefore(before time.Time) int {
	cnt := 0
	tranContextCache.cache.ForEach(func(s string, i interface{}) bool {
		tranContext := i.(*TranContext)
		if (before.IsZero() || tranContext.startTime.Before(before)) && tranContext.TxValid() == nil {
			cnt++
		}
		return true
	}, nil)
	return cnt
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
