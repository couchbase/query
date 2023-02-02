//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package memory

import (
	"fmt"
	"math"
	"os"
	"runtime/debug"
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
	setting  uint64
	max      uint64
	curr     uint64
	reserved uint64
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

	// we reserve a memory token for each configured servicer so that
	// we don't have to keep track how many more could be starting
	c := uint64(0)
	for _, v := range servicers {
		c += uint64(v) * _MEMORY_TOKEN
	}
	if manager.max > 0 && manager.max < c {
		manager.max = c
		manager.setting = c / _MEMORY_TOKEN
		logging.Infof("amending memory manager max from requested %v to %v", max, manager.setting)
	}
	atomic.AddUint64(&manager.curr, ^(manager.reserved - 1))
	atomic.AddUint64(&manager.curr, c)
	manager.reserved = c

	setMemoryLimit(int64(max))
}

var manager memoryManager

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
	return system.GetMemFree()
}

func (this *memorySession) InUseMemory() uint64 {
	return this.inUseMemory
}

const _MEMORY_LIMIT = 0.9
const _NODE_QUOTA_MULTIPLIER = 1.5

func setMemoryLimit(ml int64) {
	var extra string
	max := int64(math.MaxInt64)

	ss, err := system.NewSystemStats()
	if err == nil {
		defer ss.Close()
		t, err := ss.SystemTotalMem()
		if err == nil {
			max = int64(float64(t) * _MEMORY_LIMIT)
		}
	}

	if os.Getenv("GOMEMLIMIT") != "" {
		extra = "(GOMEMLIMIT)"
		ml = debug.SetMemoryLimit(-1)
	} else if ml > 0 {
		ml = int64(float64(ml) * _NODE_QUOTA_MULTIPLIER)
		if ml > max {
			ml = max
			extra = "(NODE QUOTA - LIMITED)"
		} else {
			extra = "(NODE QUOTA)"
		}
		debug.SetMemoryLimit(ml)
	} else {
		extra = fmt.Sprintf("(%v%% of total)", _MEMORY_LIMIT*100)
		ml = max
		debug.SetMemoryLimit(ml)
	}
	logging.Infoa(func() string {
		return fmt.Sprintf("Soft memory limit: %v %v", ml, extra)
	})
}
