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
)

// MB-75507: whether the GSI metadata has synced with the indexer nodes at least once in this process (all GSI
// indexers share one GSI client). Right after a restart, before the first sync, an index that exists may not
// be found (MB-73680); after it, an index that is not found does not exist.
var gsiSynced atomic.Bool

func SetGsiSynced() {
	if !gsiSynced.Load() {
		gsiSynced.Store(true)
	}
}

func GsiSynced() bool {
	return gsiSynced.Load()
}
