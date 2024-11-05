//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package memory

import (
	"sync/atomic"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/system"
)

type MemoryManager interface {
	AllocatedMemory() uint64
	Expire()
}

type MemorySession interface {
	Track(s uint64) (uint64, uint64, errors.Error)
	Allocated() uint64
	Release()
	AvailableMemory() uint64
	InUseMemory() uint64
}

type memoryManager struct {
	setting    uint64
	max        uint64
	curr       uint64
	reserved   uint64
	nodeQuota  uint64
	valPercent uint
}

type memorySession struct {
	inUseMemory  uint64
	currentLimit uint64
	manager      *memoryManager
}

const _MB = 1024 * 1024
const _MEMORY_TOKEN uint64 = 1 * _MB

var memLimitFunc func(int64)

func SetMemoryLimitFunction(f func(int64)) {
	memLimitFunc = f
}

func Config(maxMiB uint64, valPercent uint, servicers []int) {
	if valPercent > 100 {
		valPercent = 100
	}
	manager.valPercent = valPercent
	manager.nodeQuota = maxMiB
	manager.setting = maxMiB * uint64(valPercent) / 100
	manager.max = manager.setting * _MB

	// we reserve a memory token for each configured servicer so that
	// we don't have to keep track how many more could be starting
	c := uint64(0)
	for _, v := range servicers {
		c += uint64(v) * _MEMORY_TOKEN
	}
	if manager.max > 0 && manager.max < c {
		logging.Infof("Amending memory manager max from requested %d%% of %d MiB (%s) to %s", valPercent, maxMiB,
			logging.HumanReadableSize(int64(manager.max), false), logging.HumanReadableSize(int64(c), false))
		manager.max = c
		manager.setting = c / _MEMORY_TOKEN
	}
	atomic.AddUint64(&manager.curr, ^(manager.reserved - 1))
	atomic.AddUint64(&manager.curr, c)
	manager.reserved = c

	if memLimitFunc != nil {
		memLimitFunc(int64(maxMiB * _MB))
	}
}

var manager memoryManager

func NodeQuota() uint64 {
	return manager.nodeQuota
}

func ValPercent() uint {
	return manager.valPercent
}

func Quota() uint64 {
	return manager.setting
}

func Register() MemorySession {
	return &memorySession{0, _MEMORY_TOKEN, &manager}
}

func AllocatedMemory() uint64 {
	return manager.curr
}

func (this *memorySession) Track(size uint64) (uint64, uint64, errors.Error) {
	var newSize uint64

	top := atomic.AddUint64(&this.inUseMemory, size)
	currentLimit := this.currentLimit
	max := this.manager.max

	// only amend the curren memory limit if the manager has a limit
	if max > 0 && top > currentLimit {
		newSize = top - currentLimit
		if newSize < _MEMORY_TOKEN {
			newSize = _MEMORY_TOKEN
		}
		newCurr := atomic.AddUint64(&this.manager.curr, newSize)
		if newCurr > max {
			atomic.AddUint64(&this.manager.curr, ^(newSize - 1))
			return top, newSize, errors.NewNodeQuotaExceededError()
		}
		atomic.AddUint64(&this.currentLimit, newSize)
	}
	return top, newSize, nil
}

func (this *memorySession) Allocated() uint64 {
	return this.currentLimit
}

func (this *memorySession) Release() {
	size := this.currentLimit - _MEMORY_TOKEN
	if size > 0 {
		atomic.AddUint64(&this.manager.curr, ^(size - 1))
	}
}

func (this *memorySession) AvailableMemory() uint64 {
	return system.GetMemActualFree()
}

func (this *memorySession) InUseMemory() uint64 {
	return this.inUseMemory
}
