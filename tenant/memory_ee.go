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
	"fmt"
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
	ticks       int32
	ticker      *time.Ticker
	sync.Mutex
}

type memorySession struct {
	manager *memoryManager
	session memory.MemorySession
	context Context
}

const _MB = 1024 * 1024
const _TENANT_QUOTA_RATIO = 2
const _CLEANUP_TICKER = time.Minute
const _CLEANUP_COUNT = 30
const _CLEANUP_INTERVAL = _CLEANUP_TICKER * _CLEANUP_COUNT
const _MAX_TENANTS = 80

var managers map[string]*memoryManager = make(map[string]*memoryManager, _MAX_TENANTS)
var managersLock sync.Mutex
var perTenantQuota uint64

func Config(quota uint64) {
	perTenantQuota = quota * _MB / _TENANT_QUOTA_RATIO
	if IsServerless() {
		logging.Infoa(func() string {
			if perTenantQuota == 0 {
				return "Tenant quota is not set."
			} else {
				return fmt.Sprintf("Tenant quota is %v", logging.HumanReadableSize(int64(perTenantQuota), true))
			}
		})
	}
}

func Register(context Context) memory.MemorySession {
	tenant := Bucket(context)
	managersLock.Lock()
	manager := managers[tenant]
	if manager == nil {
		manager = &memoryManager{inUseMemory: 0, sessions: 0, tenant: tenant}
		manager.ticker = time.NewTicker(_CLEANUP_TICKER)
		go manager.checkExpire()
		managers[tenant] = manager
	}

	// make sure the session is accounted for before we unlock the manager list
	atomic.AddInt32(&manager.sessions, 1)
	managersLock.Unlock()

	session := memory.Register()
	atomic.StoreInt32(&manager.ticks, 0)
	atomic.AddUint64(&manager.inUseMemory, session.Allocated())
	return &memorySession{manager, session, context}
}

func Foreach(f func(string, memory.MemoryManager)) {
	managersLock.Lock()
	defer managersLock.Unlock()
	for n, m := range managers {
		if m.tenant != "" {
			f(n, m)
		}
		managersLock.Unlock()
		managersLock.Lock()
	}
}

func (this *memoryManager) AllocatedMemory() uint64 {
	return this.inUseMemory
}

func (this *memoryManager) Expire() {
	this.Lock()

	// avoid race conditions between forced and timed expiries
	if this.ticker == nil {
		this.Unlock()
		return
	}
	if this.sessions == 0 {
		this.ticks = _CLEANUP_COUNT
	}
	this.Unlock()
}

func (this *memoryManager) checkExpire() {

	// we cannot just panic
	defer func() {
		if recover() != nil {
			go this.checkExpire()
		}
	}()

	for now := range this.ticker.C {
		scheduled := false
		this.Lock()
		if atomic.LoadInt32(&this.sessions) == 0 {
			ticks := atomic.AddInt32(&this.ticks, 1)
			if ticks == 1 {
				scheduled = true
			}
			if ticks > _CLEANUP_COUNT {
				break
			}
		}
		this.Unlock()
		if scheduled {
			logging.Infof("Scheduling cleanup of tenant %v for %v", this.tenant, now.Add(_CLEANUP_INTERVAL))
		}
	}
	managersLock.Lock()

	// at this point we are sure that no one can increase the session count
	if atomic.LoadInt32(&this.sessions) != 0 {

		// deal with the session that started as we decided to cleanup
		managersLock.Unlock()
		this.Unlock()
		go this.checkExpire()
		return
	}

	delete(managers, this.tenant)
	managersLock.Unlock()
	this.ticker.Stop()
	this.ticker = nil
	this.Unlock()
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
			return top, allocated, errors.NewTenantQuotaExceededError(this.manager.tenant, this.context.User(), inUse, perTenantQuota)
		}
	}
	return top, allocated, nil
}

func (this *memorySession) Allocated() uint64 {
	return this.session.Allocated()
}

func (this *memorySession) Release() {
	atomic.AddInt32(&this.manager.sessions, -1)
	size := this.session.Allocated()
	this.session.Release()
	atomic.AddUint64(&this.manager.inUseMemory, ^(size - 1))
}

func (this *memorySession) AvailableMemory() uint64 {
	return perTenantQuota - this.manager.inUseMemory
}

func (this *memorySession) InUseMemory() uint64 {
	return this.manager.inUseMemory
}
