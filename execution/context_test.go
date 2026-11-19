//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"sync"
	"testing"
)

// MB-74936: operators on system:active_requests (scan, fetch, count) look up and cache the user they restrict
// to concurrently. A reader must never see the cache set without the user: "" means no restriction.
func TestFirstCredsConcurrent(t *testing.T) {
	for i := 0; i < 100; i++ {
		context := &Context{}
		var wg sync.WaitGroup
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if creds, isSet := context.FirstCreds(); isSet && creds != "local:alice" {
					t.Errorf("cached user %q, expected local:alice", creds)
				}
				context.SetFirstCreds("local:alice")
			}()
		}
		wg.Wait()
		if creds, isSet := context.FirstCreds(); !isSet || creds != "local:alice" {
			t.Errorf("cached user %q (set: %v), expected local:alice", creds, isSet)
		}
	}
}
