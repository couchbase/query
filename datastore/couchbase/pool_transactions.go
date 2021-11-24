//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"time"

	"github.com/couchbase/query/util"
)

var _TRANSACTIONMUTATIONS_POOL util.FastPool
var _DELTAKEYSPACE_POOL util.FastPool
var _MUTATIONVALUE_POOL util.FastPool
var _TRANSACTIONLOGVALUE_POOL util.FastPool

var _DELTAKEYSPACE_MAPPOOL *DeltaKeyspaceMapPool
var _MUTATIONVALUE_MAPPOOL *MutationValueMapPool
var _SAVEPOINTS_MAPPOOL *SavepointsMapPool

var _TRANSACTIONLOGVALUES_POOL *TransactionLogValuesPool
var _STRING_POOL = util.NewStringPool(_DK_DEF_SIZE)

const (
	_FASTPOOL_DRAIN_LOW      = 20              // low water mark of free entries to drian
	_FASTPOOL_DRAIN_HIGH     = 40              // high water mark of free entries to drain
	_FASTPOOL_DRAIN_HIGH2    = 60              // high water mark2 of free entries to drain
	_FASTPOOL_DRAIN_INTERVAL = 5 * time.Minute //  drian interval check
)

type DeltaKeyspaceMapPool struct {
	pool util.FastPool
	size int
}

type MutationValueMapPool struct {
	pool util.FastPool
	size int
}

type SavepointsMapPool struct {
	pool util.FastPool
	size int
}

type TransactionLogValuesPool struct {
	pool util.FastPool
	size int
}

func init() {
	util.NewFastPool(&_TRANSACTIONMUTATIONS_POOL, func() interface{} {
		return &TransactionMutations{}
	})
	_TRANSACTIONMUTATIONS_POOL.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH, _FASTPOOL_DRAIN_INTERVAL)

	util.NewFastPool(&_DELTAKEYSPACE_POOL, func() interface{} {
		return &DeltaKeyspace{}
	})
	_DELTAKEYSPACE_POOL.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH, _FASTPOOL_DRAIN_INTERVAL)

	util.NewFastPool(&_MUTATIONVALUE_POOL, func() interface{} {
		return &MutationValue{}
	})
	_MUTATIONVALUE_POOL.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH2, _FASTPOOL_DRAIN_INTERVAL)

	util.NewFastPool(&_TRANSACTIONLOGVALUE_POOL, func() interface{} {
		return &TransactionLogValue{}
	})
	_TRANSACTIONLOGVALUE_POOL.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH2, _FASTPOOL_DRAIN_INTERVAL)

	_DELTAKEYSPACE_MAPPOOL = NewDeltaKeyspaceMapPool(_TM_DEF_KEYSPACES)
	_MUTATIONVALUE_MAPPOOL = NewMutationValueMapPool(_DK_DEF_SIZE)
	_SAVEPOINTS_MAPPOOL = NewSavepointsMapPool(_TM_DEF_SAVEPOINTS)
	_TRANSACTIONLOGVALUES_POOL = NewTransactionLogValuesPool(_TM_DEF_LOGSIZE)
}

func NewDeltaKeyspaceMapPool(size int) *DeltaKeyspaceMapPool {
	rv := &DeltaKeyspaceMapPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]*DeltaKeyspace, rv.size)
	})
	rv.pool.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH, _FASTPOOL_DRAIN_INTERVAL)

	return rv
}

func (this *DeltaKeyspaceMapPool) Get() map[string]*DeltaKeyspace {
	return this.pool.Get().(map[string]*DeltaKeyspace)
}

func (this *DeltaKeyspaceMapPool) Put(s map[string]*DeltaKeyspace) {
	if s == nil {
		return
	}

	for k, v := range s {
		s[k] = nil
		delete(s, k)
		if v != nil {
			_MUTATIONVALUE_MAPPOOL.Put(v.values)
			v.values = nil
			_DELTAKEYSPACE_POOL.Put(v)
		}
	}

	this.pool.Put(s)
}

func (this *DeltaKeyspaceMapPool) Size() int {
	return this.size
}

func NewMutationValueMapPool(size int) *MutationValueMapPool {
	rv := &MutationValueMapPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]*MutationValue, rv.size)
	})
	rv.pool.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH, _FASTPOOL_DRAIN_INTERVAL)

	return rv
}

func (this *MutationValueMapPool) Get() map[string]*MutationValue {
	return this.pool.Get().(map[string]*MutationValue)
}

func (this *MutationValueMapPool) Put(s map[string]*MutationValue) {
	if s == nil {
		return
	}

	for k, v := range s {
		s[k] = nil
		delete(s, k)
		if v != nil {
			*v = MutationValue{}
			_MUTATIONVALUE_POOL.Put(v)
		}
	}

	this.pool.Put(s)
}

func (this *MutationValueMapPool) Size() int {
	return this.size
}

func NewSavepointsMapPool(size int) *SavepointsMapPool {
	rv := &SavepointsMapPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]uint64, rv.size)
	})
	rv.pool.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH, _FASTPOOL_DRAIN_INTERVAL)

	return rv
}

func (this *SavepointsMapPool) Get() map[string]uint64 {
	return this.pool.Get().(map[string]uint64)
}

func (this *SavepointsMapPool) Put(s map[string]uint64) {
	if s == nil {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}

func (this *SavepointsMapPool) Size() int {
	return this.size
}

func NewTransactionLogValuesPool(size int) *TransactionLogValuesPool {
	rv := &TransactionLogValuesPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(TransactionLogValues, 0, rv.size)
	})
	rv.pool.Drain(_FASTPOOL_DRAIN_LOW, _FASTPOOL_DRAIN_HIGH, _FASTPOOL_DRAIN_INTERVAL)

	return rv
}

func (this *TransactionLogValuesPool) Get() TransactionLogValues {
	return this.pool.Get().(TransactionLogValues)
}

func (this *TransactionLogValuesPool) Put(s TransactionLogValues) {
	if cap(s) != this.size {
		return
	}

	for _, v := range s {
		if v != nil {
			*v = TransactionLogValue{}
			_TRANSACTIONLOGVALUE_POOL.Put(v)
		}
	}

	this.pool.Put(s[0:0])
}

func (this *TransactionLogValuesPool) Size() int {
	return this.size
}
