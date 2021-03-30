//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package datastore

import (
	"sync"
	"time"
)

type TransactionSettings struct {
	atrCollection         string
	numAtrs               int
	txTimeout             time.Duration
	cleanupWindow         time.Duration
	cleanupClientAttempts bool
	cleanupLostAttempts   bool
	mutex                 sync.RWMutex
}

var transactionSettings *TransactionSettings

func init() {
	transactionSettings = &TransactionSettings{
		txTimeout:             DEF_TXTIMEOUT,
		numAtrs:               DEF_NUMATRS,
		cleanupWindow:         time.Minute,
		cleanupClientAttempts: true,
		cleanupLostAttempts:   true,
	}
}

func GetTransactionSettings() *TransactionSettings {
	return transactionSettings
}

func (this *TransactionSettings) AtrCollection() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.atrCollection
}

func (this *TransactionSettings) SetAtrCollection(atrCollection string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.atrCollection = atrCollection
}

func (this *TransactionSettings) NumAtrs() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.numAtrs
}

func (this *TransactionSettings) SetNumAtrs(numAtrs int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.numAtrs = numAtrs
}

func (this *TransactionSettings) TxTimeout() time.Duration {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txTimeout
}

func (this *TransactionSettings) SetTxTimeout(d time.Duration) {
	if d > 0 {
		this.mutex.Lock()
		defer this.mutex.Unlock()
		this.txTimeout = d
	}
}

func (this *TransactionSettings) CleanupWindow() time.Duration {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.cleanupWindow
}

func (this *TransactionSettings) SetCleanupWindow(d time.Duration) {
	if d > 0 {
		this.mutex.Lock()
		defer this.mutex.Unlock()
		this.cleanupWindow = d
	}
}

func (this *TransactionSettings) CleanupClientAttempts() bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.cleanupClientAttempts
}

func (this *TransactionSettings) SetCleanupClientAttempts(b bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.cleanupClientAttempts = b
}

func (this *TransactionSettings) CleanupLostAttempts() bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.cleanupLostAttempts
}

func (this *TransactionSettings) SetCleanupLostAttempts(b bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.cleanupLostAttempts = b
}
