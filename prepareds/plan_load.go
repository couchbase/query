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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
	"github.com/couchbase/query/util"
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
	standIns map[string]*standIn // MB-75507: for unloaded names only
}

// MB-75507: under the MODERATE error policy, a plan that can not be used (e.g. its index was dropped) is
// re-planned for each execution, and the new plan used for that execution only. The new plan is kept on this
// node (it is not saved, distributed, or shown as the plan of the statement), and stands in for the plan that can
// not be used, instead of re-planning, while it is usable (see usable()).
type standIn struct {
	prepared  *plan.Prepared
	standsFor *plan.Prepared // the cached plan it stands in for, if any
	encoded   string         // the saved plan it stands in for, if not cached (unloaded)

	// the GSI indexers of the keyspaces of the statement, and their metadata versions when the plan was re-planned
	indexers []indexerVersion
}

// something whose metadata version tells a change, e.g. a datastore.Indexer
type versioned interface {
	Refresh() errors.Error
	MetadataVersion() uint64
}

type indexerVersion struct {
	indexer versioned
	version uint64
}

// prepared stands in for the cached plan standsFor, or for the saved plan encoded; nil if it can not be used as a
// stand-in. indexers are the GSI indexers of the keyspaces of the statement, with their metadata versions taken
// before the plan it stands in for was found not usable (see keyspaceIndexers()): a change while it was being
// verified, or re-planned, then also drops the stand-in. If they were not taken (nil, e.g. no keyspace references
// are known), they are taken now, from the keyspace references of prepared.
func newStandIn(prepared, standsFor *plan.Prepared, encoded string, indexers []indexerVersion) *standIn {
	if !standInUsable(prepared) {
		return nil
	}
	if indexers == nil {
		indexers = keyspaceIndexersFn(prepared.GetKeyspaceReferences())
	}
	// long lived, as a cached plan: don't let it keep the request that created it alive (see add())
	prepared.OwnStrings()
	return &standIn{prepared: prepared, standsFor: standsFor, encoded: strings.Clone(encoded),
		indexers: indexers}
}

// The stand-in is usable while its metadata (its MetadataCheck()) and the GSI indexers of the keyspaces of the
// statement do not change: the plan it stands in for can only become usable again through such a change (e.g. an
// index created again). The MetadataCheck() of the stand-in alone does not tell: it covers only the indexers its
// own operators use, e.g. not the GSI indexer for a sequential scan, which is what is left when the only index of
// the keyspace is dropped.
func (this *standIn) usable() bool {
	if !metadataCheckFn(this.prepared) {
		return false
	}
	for _, iv := range this.indexers {
		iv.indexer.Refresh()
		if iv.indexer.MetadataVersion() != iv.version {
			return false
		}
	}
	return true
}

// the GSI indexers of the keyspaces (refs, keyspace references of a plan), and their current metadata versions;
// nil if there are no keyspace references
func keyspaceIndexers(refs []string) []indexerVersion {
	if len(refs) == 0 {
		return nil
	}
	rv := make([]indexerVersion, 0, len(refs))
	for _, ref := range refs {
		keyspace, err := datastore.GetKeyspace(algebra.ParsePath(ref)...)
		if err != nil || keyspace == nil {
			logging.Debugf("Keyspace %s of a plan standing in for a plan that can not be used not found (%v): "+
				"only the metadata of the plan is checked", ref, err)
			continue
		}
		indexer, err := keyspace.Indexer(datastore.GSI)
		if err != nil || indexer == nil {
			logging.Debugf("GSI indexer of keyspace %s of a plan standing in for a plan that can not be used not "+
				"found (%v): only the metadata of the plan is checked", ref, err)
			continue
		}
		indexer.Refresh()
		rv = append(rv, indexerVersion{indexer: indexer, version: indexer.MetadataVersion()})
	}
	return rv
}

var planLoad = &planLoadState{
	unloaded: make(map[string]bool),
	saved:    make(map[string]bool),
	standIns: make(map[string]*standIn),
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
		delete(planLoad.standIns, name)
		atomic.StoreInt32(&planLoad.unloadedSize, int32(len(planLoad.unloaded)))
	}
	planLoad.Unlock()
}

// MB-75507: under the MODERATE error policy, a saved plan (encoded) that could not be used is re-planned for each
// execution, and is not in the cache (it is unloaded): keep the new plan for the (encoded) name, to be used
// instead of re-planning while the saved plan and the metadata of the new plan do not change
func setUnloadedStandIn(name, encoded string, prepared *plan.Prepared, indexers []indexerVersion) {
	standIn := newStandIn(prepared, nil, encoded, indexers)
	if standIn == nil {
		return
	}
	planLoad.Lock()
	if planLoad.unloaded[name] {
		// the key is long lived: do not share memory with the caller's string (see markUnloaded())
		planLoad.standIns[strings.Clone(name)] = standIn
	}
	planLoad.Unlock()
}

// MB-75507: the plan kept for the (encoded) name of an unloaded saved plan (see setUnloadedStandIn()), if the
// saved plan is still the one (encoded) it stands in for, and it is usable
func getUnloadedStandIn(name, encoded string) *plan.Prepared {
	if atomic.LoadInt32(&planLoad.unloadedSize) == 0 {
		return nil
	}
	planLoad.RLock()
	standIn, ok := planLoad.standIns[name]
	planLoad.RUnlock()
	if !ok {
		return nil
	} else if standIn.encoded == encoded && standIn.usable() {
		return standIn.prepared
	}
	planLoad.Lock()
	if cur, ok := planLoad.standIns[name]; ok && cur == standIn {
		delete(planLoad.standIns, name)
	}
	planLoad.Unlock()
	return nil
}

// MB-75507: a re-planned plan can stand in for a plan that could not be used only if the metadata versions it was
// planned against are recorded, so that its MetadataCheck() notices a change: reprepare() records them (Verify())
// only with the N1QL_IGNORE_IDXR_META bit set (the feature disabled, not the default), otherwise verify it here
func standInUsable(prepared *plan.Prepared) bool {
	if prepared == nil {
		return false
	}
	if util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_IGNORE_IDXR_META) {
		return verifyPreparedFn(prepared) == nil
	}
	return true
}

// test hooks
var keyspaceIndexersFn = keyspaceIndexers
var metadataCheckFn = func(prepared *plan.Prepared) bool {
	return prepared.MetadataCheck()
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
