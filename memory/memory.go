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
)

type MemoryManager interface {
	Register() MemorySession
}

type MemorySession interface {
	Track(s uint64) (uint64, uint64, errors.Error)
	Allocated() uint64
	Release()
}

type memoryManager struct {
	setting uint64
	max     uint64
	curr    uint64
}

type memorySession struct {
	inUseMemory  uint64
	currentLimit uint64
	manager      *memoryManager
}

const _MB = 1024 * 1024
const _MEMORY_TOKEN uint64 = 1 * _MB

func Config(max uint64, servicers []int) {
	manager.setting = max
	manager.max = max * _MB
	manager.curr = 0

	// we reserve a memory token for each configured servicer so that
	// we don't have to keep track how many more could be starting
	for _, v := range servicers {
		manager.curr += uint64(v) * _MEMORY_TOKEN
	}
}

var manager memoryManager

func Quota() uint64 {
	return manager.setting
}

func Manager() MemoryManager {
	return &manager
}

func (this *memoryManager) Register() MemorySession {
	return &memorySession{0, _MEMORY_TOKEN, this}
}

func (this *memorySession) Track(size uint64) (uint64, uint64, errors.Error) {
	var newSize uint64

	top := atomic.AddUint64(&this.inUseMemory, size)
	currentLimit := this.currentLimit
	max := this.manager.max

	// only amend the curren memory limit if the manager has a limit
	if max > 0 && top > currentLimit {
		newSize = currentLimit - top
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
