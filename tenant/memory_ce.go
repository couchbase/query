//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package tenant

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/system"
)

type memorySession struct {
}

type memoryManager struct {
}

func Config(quota uint64) {
}

func Register(context Context) memory.MemorySession {
	return &memorySession{}
}

func Foreach(f func(string, memory.MemoryManager)) {
	// noop
}

func (this *memoryManager) Expire() {
	// noop
}

func (this *memorySession) Track(size uint64) (uint64, uint64, errors.Error) {
	return size, 0, nil
}

func (this *memorySession) Allocated() uint64 {
	return 0
}

func (this *memorySession) Release() {
}

func (this *memorySession) AvailableMemory() uint64 {
	return system.GetMemFree()
}

func (this *memorySession) InUseMemory() uint64 {
	return 0
}
