//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"testing"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

// remapTestIndex implements just enough of datastore.Index for IndexInfoChecksum()
type remapTestIndex struct {
	datastore.Index
	name string
}

func (this *remapTestIndex) Name() string                     { return this.name }
func (this *remapTestIndex) Type() datastore.IndexType        { return datastore.GSI }
func (this *remapTestIndex) Condition() expression.Expression { return nil }
func (this *remapTestIndex) IsPrimary() bool                  { return false }
func (this *remapTestIndex) RangeKey() expression.Expressions { return nil }

// remapTestIndexer finds index (if set) by name, and counts IndexByName and Refresh calls
type remapTestIndexer struct {
	datastore.Indexer
	keyspace  string
	index     datastore.Index
	calls     int
	refreshes int
}

func (this *remapTestIndexer) BucketId() string          { return "b" }
func (this *remapTestIndexer) ScopeId() string           { return "s" }
func (this *remapTestIndexer) KeyspaceId() string        { return this.keyspace }
func (this *remapTestIndexer) Name() datastore.IndexType { return datastore.GSI }

func (this *remapTestIndexer) Refresh() errors.Error {
	this.refreshes++
	return nil
}

func (this *remapTestIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	this.calls++
	if this.index != nil && this.index.Name() == name {
		return this.index, nil
	}
	return nil, errors.NewIndexNotFoundError(name, "", nil)
}

func shortRemapRetry() func() {
	saveInterval := remapRetryInterval
	remapRetryInterval = time.Millisecond
	return func() { remapRetryInterval = saveInterval }
}

// MB-75632: a not found index is retried with waits only once per backup/restore request; later
// lookups refresh and look it up once more, without waiting
func TestRemapCacheNotFound(t *testing.T) {
	defer shortRemapRetry()()

	cache := NewRemapCache()
	pc := newPlanContext(nil, nil, true, cache, 0)

	ix1 := &remapTestIndexer{keyspace: "c1"}
	for i := 0; i < 3; i++ {
		index, err := getRemapIndex("ix1", ix1, "checksum", pc)
		if index != nil || !errors.IsIndexNotFoundError(err) {
			t.Fatalf("call %d: expected index not found error, got index %v, err %v", i, index, err)
		}
	}
	if ix1.calls != remapMaxRetry+2 {
		t.Errorf("expected %d IndexByName calls for 3 lookups of the same index, got %d", remapMaxRetry+2, ix1.calls)
	}

	// the index shows up later in the same request: it is used, and the cached failure is cleared
	ix1.index = &remapTestIndex{name: "ix1"}
	ix1.calls, ix1.refreshes = 0, 0
	index, err := getRemapIndex("ix1", ix1, "", pc)
	if index == nil || err != nil {
		t.Fatalf("expected index ix1 after it was created, got index %v, err %v", index, err)
	}
	if ix1.calls != 1 || ix1.refreshes != 1 {
		t.Errorf("expected 1 IndexByName and 1 Refresh call, got %d and %d", ix1.calls, ix1.refreshes)
	}
	if cache.get(remapCacheKey(ix1, "ix1")) != nil {
		t.Errorf("expected cached not found error for ix1 to be cleared")
	}

	// same index name on a different keyspace is not affected by a cached failure
	cache2 := NewRemapCache()
	pc2 := newPlanContext(nil, nil, true, cache2, 0)
	ixa := &remapTestIndexer{keyspace: "c1"}
	ixb := &remapTestIndexer{keyspace: "c2"}
	getRemapIndex("ix1", ixa, "checksum", pc2)
	getRemapIndex("ix1", ixb, "checksum", pc2)
	if ixb.calls != remapMaxRetry {
		t.Errorf("expected %d IndexByName calls on a different keyspace, got %d", remapMaxRetry, ixb.calls)
	}

	// no cache: every lookup retries
	ix3 := &remapTestIndexer{keyspace: "c1"}
	pcNoCache := newPlanContext(nil, nil, true, nil, 0)
	getRemapIndex("ix1", ix3, "checksum", pcNoCache)
	getRemapIndex("ix1", ix3, "checksum", pcNoCache)
	if ix3.calls != 2*remapMaxRetry {
		t.Errorf("expected %d IndexByName calls without a cache, got %d", 2*remapMaxRetry, ix3.calls)
	}
}

// MB-75632: a definition checksum mismatch is cached per expected checksum
func TestRemapCacheChecksumMismatch(t *testing.T) {
	defer shortRemapRetry()()

	cache := NewRemapCache()
	pc := newPlanContext(nil, nil, true, cache, 0)
	idx := &remapTestIndex{name: "ix1"}
	ixr := &remapTestIndexer{keyspace: "c1", index: idx}

	good, err1 := datastore.IndexInfoChecksum(idx, pc.planVersion)
	if err1 != nil {
		t.Fatalf("unexpected checksum error: %v", err1)
	}

	for i := 0; i < 2; i++ {
		index, err := getRemapIndex("ix1", ixr, "stale-checksum", pc)
		if index != nil || err == nil || err.Code() != errors.W_INDEX_CHECKSUM_MISMATCH {
			t.Fatalf("call %d: expected checksum mismatch error, got index %v, err %v", i, index, err)
		}
	}
	if ixr.calls != 1 {
		t.Errorf("expected 1 IndexByName call for 2 lookups with the same stale checksum, got %d", ixr.calls)
	}

	// a plan saved with the matching checksum still remaps
	index, err := getRemapIndex("ix1", ixr, good, pc)
	if index == nil || err != nil {
		t.Fatalf("expected index ix1 for matching checksum, got index %v, err %v", index, err)
	}
	if ixr.calls != 2 {
		t.Errorf("expected 2 IndexByName calls in total, got %d", ixr.calls)
	}
}
