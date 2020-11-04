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
	"encoding/json"
	"sync"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

const (
	_DEFAULT_DATE_FORMAT = "2006-01-02T15:04:05.999Z07:00"
	_TXCONTEXT_SIZE      = 256
	_TX_CLEANUP_AFTER    = -5 * time.Minute
)

type TxStatus uint32

const (
	TX_INUSE TxStatus = 1 << iota
	TX_EXPIRED
	TX_RELEASED
)

type TranContext struct {
	txId                string
	txTimeout           time.Duration
	kvTimeout           time.Duration
	txDurabilityTimeout time.Duration
	txDurabilityLevel   datastore.DurabilityLevel
	txIsolationLevel    datastore.IsolationLevel
	txScanConsistency   datastore.ScanConsistency
	atrCollection       string
	numAtrs             int
	txData              []byte
	txImplicit          bool
	txStatus            TxStatus
	txLastStmtNum       int64
	mutex               sync.RWMutex
	lastUse             time.Time
	expiryTime          time.Time
	uses                int32
	txMutations         interface{}
	memoryQuota         uint64
}

func NewTxContext(txImplicit bool, txData []byte, txTimeout, txDurabilityTimeout, kvTimeout time.Duration,
	txDurabilityLevel datastore.DurabilityLevel, txIsolationLevel datastore.IsolationLevel,
	txScanConsistency datastore.ScanConsistency, atrCollection string, numAtrs int, memoryQuota uint64) *TranContext {

	rv := &TranContext{txTimeout: txTimeout,
		kvTimeout:           kvTimeout,
		txDurabilityTimeout: txDurabilityTimeout,
		txDurabilityLevel:   txDurabilityLevel,
		txIsolationLevel:    txIsolationLevel,
		txScanConsistency:   txScanConsistency,
		txImplicit:          txImplicit,
		txData:              txData,
		atrCollection:       atrCollection,
		numAtrs:             numAtrs,
		memoryQuota:         memoryQuota,
	}

	return rv
}

func (this *TranContext) TxId() string {
	return this.txId
}

func (this *TranContext) SetTxId(txId string, expiryTime time.Time) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.txId = txId
	this.expiryTime = expiryTime
}

func IsBitOn(txStatus, bit TxStatus) bool {
	return (txStatus & bit) != 0
}

func (this *TranContext) TxTimeout() time.Duration {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txTimeout
}

func (this *TranContext) TxValid() errors.Error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.txStatus&TX_RELEASED != 0 {
		return errors.NewTransactionReleased()
	} else if this.txExpired() {
		return errors.NewTransactionExpired()
	}
	return nil
}

func (this *TranContext) TxExpired() bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.txExpired()
}

func (this *TranContext) txExpired() bool {
	if this.txId == "" {
		return false
	} else if this.txStatus&TX_EXPIRED != 0 {
		return true
	} else if time.Now().After(this.expiryTime) {
		this.txStatus |= TX_EXPIRED
		return true
	} else {
		return false
	}
}

func (this *TranContext) TxTimeRemaining() time.Duration {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	timeoutMS := this.expiryTime.Sub(time.Now()) * time.Millisecond
	return timeoutMS
}

func (this *TranContext) SetTxExpiry() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.expiryTime = time.Now()
	this.txStatus |= TX_EXPIRED
}

func (this *TranContext) TxData() []byte {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txData
}

func (this *TranContext) KvTimeout() time.Duration {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.kvTimeout
}

func (this *TranContext) TxDurabilityTimeout() time.Duration {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txDurabilityTimeout
}

func (this *TranContext) TxDurabilityLevel() datastore.DurabilityLevel {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txDurabilityLevel
}

func (this *TranContext) SetTxIsolationLevel(isolationLevel datastore.IsolationLevel) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.txIsolationLevel = isolationLevel
}

func (this *TranContext) TxIsolationLevel() datastore.IsolationLevel {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txIsolationLevel
}

func (this *TranContext) TxScanConsistency() datastore.ScanConsistency {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txScanConsistency
}

func (this *TranContext) AtrCollection() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.atrCollection
}

func (this *TranContext) NumAtrs() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.numAtrs
}

func (this *TranContext) SetTxImplicit(b bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.txImplicit = b
}

func (this *TranContext) TxImplicit() bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txImplicit
}

func (this *TranContext) TxStatus() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	if this.TxExpired() {
		return "Expired"
	} else if this.txStatus&TX_INUSE != 0 {
		return "Inuse"
	} else {
		return "Idle"
	}
}

func (this *TranContext) SetTxStatus(bits TxStatus) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.txStatus |= bits

}

func (this *TranContext) TxInUse() bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txStatus&TX_INUSE != 0
}

func (this *TranContext) SetTxInUse(inuse bool) errors.Error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if !inuse {
		if this.txStatus&TX_INUSE != 0 {
			this.txStatus &^= TX_INUSE
		}
	} else if this.txStatus&TX_EXPIRED != 0 || time.Now().After(this.expiryTime) {
		return errors.NewTransactionExpired()
	} else if this.txStatus&TX_INUSE != 0 {
		return errors.NewTransactionInuse()
	} else {
		this.txStatus |= TX_INUSE
	}
	return nil
}

func (this *TranContext) SetTxLastStmtNum(snum int64) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.txLastStmtNum = snum
}

func (this *TranContext) TxLastStmtNum() int64 {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txLastStmtNum
}

func (this *TranContext) TxMutations() interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txMutations

}

func (this *TranContext) SetTxMutations(txMutations interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.txMutations = txMutations
}

func (this *TranContext) MemoryQuota() uint64 {
	return uint64(this.memoryQuota)
}

func (this *TranContext) Content(r map[string]interface{}) {
	r["id"] = this.txId
	r["timeout"] = this.txTimeout.String()
	if this.kvTimeout > 0 {
		r["kvTimeout"] = this.kvTimeout.String()
	}
	if this.txDurabilityTimeout > 0 {
		r["durabilityTimeout"] = this.txDurabilityTimeout.String()
	}
	r["durabilityLevel"] = datastore.DurabilityLevelToName(this.txDurabilityLevel)
	r["isolationLevel"] = datastore.IsolationLevelToName(this.txIsolationLevel)
	r["scanConsistency"] = this.txScanConsistency
	if this.atrCollection != "" {
		r["atrCollection"] = this.atrCollection
	}
	if this.numAtrs > 0 {
		r["numAtrs"] = this.numAtrs
	}
	if this.txImplicit {
		r["implicit"] = this.txImplicit
	}
	if this.txLastStmtNum > 0 {
		r["lastStmtNum"] = this.txLastStmtNum
	}
	r["lastUse"] = this.lastUse.Format(_DEFAULT_DATE_FORMAT)
	r["expiryTime"] = this.expiryTime.Format(_DEFAULT_DATE_FORMAT)
	r["uses"] = this.uses
	r["status"] = this.txStatus
	if this.memoryQuota > 0 {
		r["memoryQuota"] = this.memoryQuota
	}

	usedMemory := int64(_TXCONTEXT_SIZE + len(this.txData) + len(this.AtrCollection()))
	if txMutations := this.TxMutations(); txMutations != nil {
		if tranMemory, ok := txMutations.(datastore.TransactionMemory); ok {
			usedMemory += tranMemory.TransactionUsedMemory()
		}
	}

	if usedMemory > 0 {
		r["usedMemory"] = usedMemory
	}

	if len(this.txData) > 0 {
		var drv map[string]interface{}
		if err := json.Unmarshal(this.txData, &drv); err != nil {
			r["data"] = drv
		}
	}
}
