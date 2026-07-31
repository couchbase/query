//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Node-local read cache for knowledge documents, invalidated via a metakv change counter -
// mirrors the scheme used by functions/metakv.go for the UDF cache. This is a performance
// optimization only, not a correctness/consistency mechanism: cross-node propagation is
// best-effort (a node that misses the metakv notification simply keeps serving a stale cached
// entry until its own next write bumps its local counter), and a cache miss always falls back to
// checking the system collection directly and populating the cache from that read.
package knowledge

import (
	"strconv"

	"github.com/couchbase/cbauth/metakv"
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const _CHANGE_COUNTER_PATH = "/query/knowledge_cache/"
const _CHANGE_COUNTER = _CHANGE_COUNTER_PATH + "counter"

var changeCounter int32

// Init starts watching for other nodes' knowledge writes so this node's cache entries get
// invalidated in response. Call once at startup; safe to omit entirely (the cache then just never
// learns of other nodes' writes and relies on this node's own writes to keep itself current).
func Init() {
	err := metakv.Add(_CHANGE_COUNTER, fmtChangeCounter())
	if err != nil && err != metakv.ErrRevMismatch {
		logging.Infof("Unable to initialize knowledge cache monitor: %v", err)
	}
	go metakv.RunObserveChildren(_CHANGE_COUNTER_PATH, changeCallback, make(chan struct{}))
}

func changeCallback(kve metakv.KVEntry) error {
	if kve.Path != _CHANGE_COUNTER {
		return nil
	}
	node, _ := distributed.RemoteAccess().SplitKey(string(kve.Value))
	// unclustered nodes can't check against themselves, as there may be many of them, and all
	// present themselves with an empty name
	if node == "" || node != distributed.RemoteAccess().WhoAmI() {
		atomic.AddInt32(&changeCounter, 1)
	}
	return nil
}

func setChange() {
	atomic.AddInt32(&changeCounter, 1)
	err := metakv.Set(_CHANGE_COUNTER, fmtChangeCounter(), nil)
	if err != nil && err.Error() == "Not found" {
		err = metakv.Add(_CHANGE_COUNTER, fmtChangeCounter())
	}
	if err != nil {
		logging.Infof("Unable to update knowledge cache monitor: %v", err)
	}
}

// propagate the node name so a node doesn't act on its own change counter change coming back to
// it, and the counter value so repeated changes from the same node aren't lumped as one
func fmtChangeCounter() []byte {
	return []byte(distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(),
		strconv.Itoa(int(atomic.LoadInt32(&changeCounter)))))
}

type cacheEntry struct {
	doc      value.AnnotatedValue // nil if exists is false (a confirmed miss is cached too)
	exists   bool
	revision int32
}

// each cache entry holds one collection's entire knowledge document (all of its hints as a single
// object), not one hint - at up to 100 hints of up to 1KiB each per collection, a fully-populated
// entry runs to roughly 100-150KiB. Unlike functions/sequences' caches (small, fixed-size metadata
// blobs, hence their much larger entry-count limits), that per-entry size means an unbounded or
// very large limit here could reach into the GBs cluster-wide. This is a perf-only cache (a miss
// just costs one live Fetch, never a correctness issue), so it's fine to keep this modest - worst
// case (every entry maxed out) is on the order of a couple hundred MB.
const _CACHE_LIMIT = 1024

var cache = util.NewGenCache(_CACHE_LIMIT)

// loadDoc returns the document stored at key, from the local cache if it's still current for the
// latest known change counter, otherwise by fetching it live from the system collection and
// caching the result (hit or miss) for next time.
func loadDoc(b datastore.Keyspace, key string) (value.AnnotatedValue, bool, errors.Error) {
	current := atomic.LoadInt32(&changeCounter)

	if v := cache.Get(key, nil); v != nil {
		if ce := v.(*cacheEntry); ce.revision == current {
			return ce.doc, ce.exists, nil
		}
	}

	res := make(map[string]value.AnnotatedValue, 1)
	errs := b.Fetch([]string{key}, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	if errs != nil && len(errs) > 0 && !errors.IsNotFoundError("", errs[0]) && !errs[0].HasCause(errors.E_CB_BULK_GET) {
		return nil, false, errs[0]
	}
	doc, exists := res[key]
	cache.Add(&cacheEntry{doc: doc, exists: exists, revision: current}, key, nil)
	return doc, exists, nil
}

// invalidate discards any cached entry for key, forcing the next loadDoc to fetch live.
func invalidate(key string) {
	cache.Delete(key, nil)
}

// InvalidateCacheKey discards any cached entry for a raw storage key. Exported for external
// reconciliation sweeps (datastore/couchbase.CleanupSystemCollection) that delete stale know::
// documents this package didn't itself write - e.g. orphaned by a scope/collection drop that
// happened before this node ever watched the bucket.
func InvalidateCacheKey(key string) {
	invalidate(key)
}

// refreshDoc re-populates the cache for key from a live Fetch, replacing whatever is cached (or
// absent) for it. Used after a successful Insert/Update: the document we just wrote is
// client-constructed and carries no META_CAS/META_FLAGS (Insert/Update don't write those back
// into the object passed to them), so caching it as-is would make the next Update on this key
// fail. A live Fetch immediately after a completed write is safe - a single-key KV Fetch is
// immediately consistent, unlike the GSI-index-backed scans elsewhere in this package - and
// leaves the cache holding a proper, meta-complete copy for both the next Update and for Scan
// (which serves already-cached entries without waiting on the index).
func refreshDoc(b datastore.Keyspace, key string) {
	res := make(map[string]value.AnnotatedValue, 1)
	current := atomic.LoadInt32(&changeCounter)
	if errs := b.Fetch([]string{key}, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false); len(errs) > 0 {
		invalidate(key)
		return
	}
	doc, exists := res[key]
	cache.Add(&cacheEntry{doc: doc, exists: exists, revision: current}, key, nil)
}

// forEachCachedDoc invokes f for every currently-cached, confirmed-existing document. f must not
// block or call back into the cache.
func forEachCachedDoc(f func(doc value.AnnotatedValue)) {
	cache.ForEach(func(id string, entry interface{}) bool {
		if ce, ok := entry.(*cacheEntry); ok && ce.exists && ce.doc != nil {
			f(ce.doc)
		}
		return true
	}, nil)
}

// purgeScope discards every locally-cached knowledge entry belonging to bucket.scope, independent
// of whether the underlying system collection can still be scanned. Needed because DropScope can
// run after the bucket itself has already been dropped (a whole-bucket delete drops each of its
// scopes too, via KeyspaceDeleteCallback -> dropDictCacheEntries -> clearOldScope), at which point
// a live scan can no longer enumerate the keys to invalidate them individually - this instead
// matches directly against what's already cached, so it works even then.
func purgeScope(bucket, scope string) {
	var keys []string
	cache.ForEach(func(id string, entry interface{}) bool {
		if ce, ok := entry.(*cacheEntry); ok && ce.exists && ce.doc != nil {
			bk, bok := ce.doc.Field("bucket")
			sc, sok := ce.doc.Field("scope")
			if bok && sok && bk.ToString() == bucket && sc.ToString() == scope {
				keys = append(keys, id)
			}
		}
		return true
	}, nil)
	for _, key := range keys {
		invalidate(key)
	}
}
