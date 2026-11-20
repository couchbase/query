//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/couchbase/query/settings"
)

// Tracks when a prepared cache miss has to look for a saved plan in QUERY_METADATA, on top of a
// saved plan being possible for the statement (see checkSavedPlan()):
//   - while the cache is being primed from the saved plans at startup: requests are accepted
//     before priming completes, and a request must not re-plan (and overwrite) a saved plan
//     that simply has not been loaded yet
//   - for saved plans that priming (or a later load) could not put in the cache, e.g. the plan
//     failed verification under the MODERATE or STRICT error policy: each execution has to load
//     the saved plan and apply the error policy, rather than re-plan and overwrite it
//
// MB-75506: it also tracks the names known on this node to have a plan saved with PREPARE SAVE: with plan
// stability off, only those are looked for (see checkSavedPlan())
type planLoadState struct {
	priming      int32 // atomic
	unloadedSize int32 // atomic, len(unloaded), to skip the lock in the common case
	savedSize    int32 // atomic, len(saved)
	sync.RWMutex
	unloaded map[string]bool
	saved    map[string]bool
}

var planLoad = &planLoadState{
	unloaded: make(map[string]bool),
	saved:    make(map[string]bool),
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
		// the key is long lived (e.g. the saved plan can not be put in the cache): do not share memory with the
		// caller's string (e.g. a substring of a request's)
		planLoad.unloaded[strings.Clone(name)] = true
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

// MB-75506: a plan saved with PREPARE SAVE was put in the cache under the (encoded) name: kept after the
// entry is evicted
func markSaved(name string) {
	planLoad.Lock()
	if !planLoad.saved[name] {
		// the key is long lived (it stays after the entry is evicted): do not share memory with the caller's string
		planLoad.saved[strings.Clone(name)] = true
		atomic.StoreInt32(&planLoad.savedSize, int32(len(planLoad.saved)))
	}
	planLoad.Unlock()
}

// no (more) plan saved with PREPARE SAVE under the (encoded) name, e.g. it was removed, or not found
func clearSaved(name string) {
	if atomic.LoadInt32(&planLoad.savedSize) == 0 {
		return
	}
	planLoad.Lock()
	if planLoad.saved[name] {
		delete(planLoad.saved, name)
		atomic.StoreInt32(&planLoad.savedSize, int32(len(planLoad.saved)))
	}
	planLoad.Unlock()
}

// e.g. QUERY_METADATA has been dropped
func clearAllSaved() {
	planLoad.Lock()
	planLoad.saved = make(map[string]bool)
	atomic.StoreInt32(&planLoad.savedSize, 0)
	planLoad.Unlock()
}

func isSaved(name string) bool {
	if atomic.LoadInt32(&planLoad.savedSize) == 0 {
		return false
	}
	planLoad.RLock()
	rv := planLoad.saved[name]
	planLoad.RUnlock()
	return rv
}

// what a prepared cache miss is for
type lookupKind int

const (
	_LOOKUP_PREPARED   lookupKind = iota // a prepared statement (EXECUTE), or an inline UDF entry
	_LOOKUP_AD_HOC                       // an ad hoc (auto prepared) statement
	_LOOKUP_CACHE_ONLY                   // the caller has the plan (e.g. distributed by another node)
)

// whether a prepared cache miss for the (encoded) name should look for a saved plan
// MB-75506: whenever a saved plan can exist for the statement, whatever the cache state: a saved plan may not
// be in the cache because it was evicted (the cache has no eviction hook, and need not be full any more,
// e.g. after raising the limit) or its distribution did not reach this node; a miss that does not find it
// fails EXECUTE (4040), or under AD_HOC saves a new plan over the saved plan. A saved plan can exist for:
//   - a prepared statement: with plan stability on; with plan stability off, only a name known on this node
//     to have a plan saved with PREPARE SAVE (e.g. evicted), so that a PREPARE or EXECUTE does not read
//     QUERY_METADATA when plan stability is not used: a plan saved with PREPARE SAVE on another node, whose
//     distribution did not reach this node, is not found (it has to be prepared again on this node)
//   - an inline UDF entry: with plan stability on, same as a prepared statement (its name is never saved with
//     PREPARE SAVE)
//   - an ad hoc statement: under AD_HOC and AD_HOC_READ_ONLY (saved under AD_HOC)
func checkSavedPlan(name string, kind lookupKind, mode settings.PlanStabilityMode) bool {
	if isPriming() || isUnloaded(name) {
		return true
	}
	switch kind {
	case _LOOKUP_PREPARED:
		return mode != settings.PS_MODE_OFF || isSaved(name)
	case _LOOKUP_AD_HOC:
		return mode == settings.PS_MODE_AD_HOC || mode == settings.PS_MODE_AD_HOC_READ_ONLY
	}
	return false
}
