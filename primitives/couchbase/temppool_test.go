//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// a bucket whose client has TLS on, so that getTempConnPool fetches the node services from a stub ns_server, which
// runs during (an optional) hook while the call is in progress
func tlsTempPoolBucket(t *testing.T, status int, during func()) (*Bucket, *atomic.Int32) {
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pools/default/nodeServices" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		time.Sleep(50 * time.Millisecond) // let concurrent callers pile up
		if during != nil {
			during()
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Write([]byte(`{"rev":1,"nodesExt":[{"hostname":"n3","services":{"kv":11210,"kvSSL":11207}}]}`))
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	client := &Client{BaseURL: u, ah: basicAuth{"u", "p"}, tlsConfig: &tls.Config{},
		Info: Pools{Pools: []RestPool{{Name: "default"}}}}
	b := &Bucket{Name: "default", ah: basicAuth{"u", "p"}, pool: &Pool{client: client}}
	t.Cleanup(func() {
		for _, p := range b.tempConnPools {
			if p != nil {
				p.Close()
			}
		}
	})
	return b, calls
}

// concurrent callers for the same new node share one ns_server call and one pool, and the bucket lock is free
// while the call is in progress
func TestGetTempConnPoolTLSConcurrent(t *testing.T) {
	var b *Bucket
	var calls *atomic.Int32
	lockHeld := &atomic.Bool{}
	release := make(chan struct{})
	b, calls = tlsTempPoolBucket(t, http.StatusOK, func() {
		// hold the reply until every caller has started, so that they all miss the pool
		<-release
		// a read lock fails only if the bucket is write locked (the other callers wait on tempPoolLock, without
		// it); a write lock would also fail on a caller's brief read lock
		if b.TryRLock() {
			b.RUnlock()
		} else {
			lockHeld.Store(true)
		}
	})
	servers := []string{"n0:11210", "n1:11210", "n2:11210"}

	const callers = 20
	pools := make([]*connectionPool, callers)
	var wg, started sync.WaitGroup
	for i := range pools {
		wg.Add(1)
		started.Add(1)
		go func() {
			defer wg.Done()
			started.Done()
			pools[i] = b.getTempConnPool(1, servers, "n3:11210")
		}()
	}
	started.Wait()
	// the first fetch is in progress (and held), then give the others time to reach getTempConnPool
	for deadline := time.Now().Add(10 * time.Second); calls.Load() == 0 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Errorf("%d ns_server calls for one node, expected 1", n)
	}
	if lockHeld.Load() {
		t.Errorf("bucket locked during the ns_server call")
	}
	if pools[0] == nil || pools[0].host != "n3:11207" {
		t.Fatalf("expected a TLS pool for n3, got %+v", pools[0])
	}
	for i, p := range pools {
		if p != pools[0] {
			t.Errorf("caller %d got a different pool %p, expected %p", i, p, pools[0])
		}
	}
}

// a refresh while the node services are being fetched: the pool isn't recorded against the old server list
func TestGetTempConnPoolTLSRefreshed(t *testing.T) {
	var b *Bucket
	b, _ = tlsTempPoolBucket(t, http.StatusOK, func() {
		// what refresh does to the temporary pools, under the bucket lock
		b.Lock()
		b.tempServers, b.tempConnPools = nil, nil
		b.refreshes++
		b.Unlock()
	})
	servers := []string{"n0:11210", "n1:11210", "n2:11210"}

	if p := b.getTempConnPool(1, servers, "n3:11210"); p != nil {
		t.Errorf("expected no pool after a refresh, got %+v", p)
	}
	if len(b.tempServers) != 0 || len(b.tempConnPools) != 0 {
		t.Errorf("temporary pool recorded for the old server list: %v %v", b.tempServers, b.tempConnPools)
	}
}

// the bucket is closed while the node services are being fetched: no pool is created
func TestGetTempConnPoolTLSClosed(t *testing.T) {
	var b *Bucket
	b, _ = tlsTempPoolBucket(t, http.StatusOK, func() {
		b.Lock()
		b.closed = true
		b.Unlock()
	})
	servers := []string{"n0:11210", "n1:11210", "n2:11210"}

	if p := b.getTempConnPool(1, servers, "n3:11210"); p != nil {
		t.Errorf("expected no pool for a closed bucket, got %+v", p)
	}
	for _, p := range b.tempConnPools {
		if p != nil {
			t.Errorf("temporary pool created for a closed bucket: %v", b.tempConnPools)
		}
	}
}

// a failed ns_server call records the node without a pool (so that it isn't retried until the next refresh)
func TestGetTempConnPoolTLSFetchFails(t *testing.T) {
	b, calls := tlsTempPoolBucket(t, http.StatusInternalServerError, nil)
	servers := []string{"n0:11210", "n1:11210", "n2:11210"}

	if p := b.getTempConnPool(1, servers, "n3:11210"); p != nil {
		t.Errorf("expected no pool when ns_server fails, got %+v", p)
	}
	if len(b.tempServers) != len(servers) || b.tempServers[1] != "n3:11210" || b.tempConnPools[1] != nil {
		t.Errorf("failed node not recorded: %v %v", b.tempServers, b.tempConnPools)
	}
	if p := b.getTempConnPool(1, servers, "n3:11210"); p != nil || calls.Load() != 1 {
		t.Errorf("failed node retried: pool %v, %d calls", p, calls.Load())
	}
}
