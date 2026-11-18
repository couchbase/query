//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"sync"
	"sync/atomic"
)

// Tracks when a prepared cache miss has to look for a saved plan in QUERY_METADATA (on top of
// the cache being full, where an entry may have been evicted):
//   - while the cache is being primed from the saved plans at startup: requests are accepted
//     before priming completes, and a request must not re-plan (and overwrite) a saved plan
//     that simply has not been loaded yet
//   - for saved plans that priming (or a later load) could not put in the cache, e.g. the plan
//     failed verification under the MODERATE or STRICT error policy: each execution has to load
//     the saved plan and apply the error policy, rather than re-plan and overwrite it
type planLoadState struct {
	priming      int32 // atomic
	unloadedSize int32 // atomic, len(unloaded), to skip the lock in the common case
	sync.RWMutex
	unloaded map[string]bool
}

var planLoad = &planLoadState{
	unloaded: make(map[string]bool),
}

// called before the query service starts accepting requests; PreparedsFromPersisted() ends priming
func PreparedsStartPriming() {
	atomic.StoreInt32(&planLoad.priming, 1)
}

func endPriming() {
	atomic.StoreInt32(&planLoad.priming, 0)
}

func isPriming() bool {
	return atomic.LoadInt32(&planLoad.priming) != 0
}

// a saved plan exists for the (encoded) name but is not in the prepared cache
func markUnloaded(name string) {
	planLoad.Lock()
	if !planLoad.unloaded[name] {
		planLoad.unloaded[name] = true
		atomic.StoreInt32(&planLoad.unloadedSize, int32(len(planLoad.unloaded)))
	}
	planLoad.Unlock()
}

// the (encoded) name is in the prepared cache, or there is no saved plan for it
func clearUnloaded(name string) {
	if atomic.LoadInt32(&planLoad.unloadedSize) == 0 {
		return
	}
	planLoad.Lock()
	if planLoad.unloaded[name] {
		delete(planLoad.unloaded, name)
		atomic.StoreInt32(&planLoad.unloadedSize, int32(len(planLoad.unloaded)))
	}
	planLoad.Unlock()
}

func isUnloaded(name string) bool {
	if atomic.LoadInt32(&planLoad.unloadedSize) == 0 {
		return false
	}
	planLoad.RLock()
	rv := planLoad.unloaded[name]
	planLoad.RUnlock()
	return rv
}

// whether a prepared cache miss for the (encoded) name should look for a saved plan
func checkSavedPlan(name string) bool {
	return isPriming() || isUnloaded(name) || prepareds.cache.Size() >= prepareds.cache.Limit()
}
