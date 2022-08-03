//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package tenant

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/memory"
)

type memoryManager struct {
	inUseMemory uint64
	sessions    int32
	tenant      string
	timer       *time.Timer
	sync.Mutex
}

type memorySession struct {
	manager *memoryManager
	session memory.MemorySession
	context Context
}

const _MB = 1024 * 1024
const _TENANT_QUOTA_RATIO = 2
const _CLEANUP_INTERVAL = 30 * time.Minute
const _MAX_TENANTS = 80

var managers map[string]*memoryManager = make(map[string]*memoryManager, _MAX_TENANTS)
var lock sync.Mutex
var perTenantQuota uint64

func Config(quota uint64) {
	perTenantQuota = quota * _MB / _TENANT_QUOTA_RATIO
}

func Register(context Context) memory.MemorySession {
	tenant := Bucket(context)
	lock.Lock()
	manager := managers[tenant]
	if manager == nil {
		manager = &memoryManager{inUseMemory: 0, sessions: 0, tenant: tenant}
		managers[tenant] = manager
	}
	lock.Unlock()

	atomic.AddInt32(&manager.sessions, 1)
	session := memory.Register()
	atomic.AddUint64(&manager.inUseMemory, session.Allocated())
	return &memorySession{manager, session, context}
}

func (this *memoryManager) expire() {
	this.Lock()

	// avoid race condition among doubly scheduled cleaners
	if this.timer == nil {
		this.Unlock()
		return
	}
	this.timer.Stop()
	this.timer = nil
	this.Unlock()
	lock.Lock()
	delete(managers, this.tenant)
	lock.Unlock()
	for _, f := range resourceManagers {
		f(this.tenant)
	}
}

func (this *memorySession) Track(size uint64) (uint64, uint64, errors.Error) {
	top, allocated, err := this.session.Track(size)
	if err != nil {
		return top, allocated, err
	}
	if allocated != 0 {
		inUse := atomic.AddUint64(&this.manager.inUseMemory, allocated)

		// TODO TENANT there is an opportunity here to give tenants different quotas
		if perTenantQuota > 0 && inUse > perTenantQuota {
			return top, allocated, errors.NewTenantQuotaExceededError(this.manager.tenant, this.context.User())
		}
	}
	return top, allocated, nil
}

func (this *memorySession) Allocated() uint64 {
	return this.session.Allocated()
}

func (this *memorySession) Release() {
	remaining := atomic.AddInt32(&this.manager.sessions, -1)
	size := this.session.Allocated()
	this.session.Release()
	atomic.AddUint64(&this.manager.inUseMemory, ^(size - 1))

	// no need to cleanup anything if this was a privileged session
	// any buckets loaded by this session will be unloaded via the streaming subscription
	if remaining == 0 && this.manager.tenant != "" {
		this.manager.Lock()
		if this.manager.timer != nil {
			this.manager.timer.Stop()
			this.manager.timer.Reset(_CLEANUP_INTERVAL)
		} else {
			this.manager.timer = time.AfterFunc(_CLEANUP_INTERVAL, func() { this.manager.expire() })
		}
		this.manager.Unlock()
		logging.Infof("Scheduling cleanup of tenant %v for %v", this.manager.tenant, time.Now().Add(_CLEANUP_INTERVAL))
	}
}