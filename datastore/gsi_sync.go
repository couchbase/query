//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"sync/atomic"
	"time"
)

// MB-75507: when the GSI metadata first synced with the indexer nodes in this process (all GSI indexers share one
// GSI client); 0 before. Right after a restart, before the first sync, an index that exists may not be found
// (MB-73680).
// MB-75516: the first sync is with one indexer node: the metadata of other indexer nodes (e.g. slower to restart)
// may come later, so an index that is not found right after the first sync may still exist; see GsiSyncedSince().
var gsiSyncedAt atomic.Int64

func SetGsiSynced() {
	if gsiSyncedAt.Load() == 0 {
		gsiSyncedAt.CompareAndSwap(0, time.Now().UnixNano())
	}
}

func GsiSynced() bool {
	return gsiSyncedAt.Load() != 0
}

// MB-75516: how long ago the GSI metadata first synced, and whether it has
func GsiSyncedSince() (time.Duration, bool) {
	at := gsiSyncedAt.Load()
	if at == 0 {
		return 0, false
	}
	return time.Since(time.Unix(0, at)), true
}
