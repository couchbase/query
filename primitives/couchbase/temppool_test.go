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

// whether a pool has been closed (Close closes its connections channel). It receives from p.connections, so on an
// open pool with an idle connection it takes that connection (and reports false): use it only on pools that have no
// idle connections.
func poolClosed(p *connectionPool) bool {
	select {
	case _, ok := <-p.connections:
		return !ok
	default:
		return false
	}
}

// closing the bucket closes its temporary pools
func TestBucketCloseTempPools(t *testing.T) {
	b := &Bucket{Name: "default", ah: basicAuth{"u", "p"}, pool: &Pool{client: &Client{}}}
	servers := []string{"n0:11210", "n1:11210", "n2:11210"}
	p := b.getTempConnPool(1, servers, "n3:11210")
	if p == nil {
		t.Fatalf("no temporary pool created")
	}
	b.Close()
	if !poolClosed(p) {
		p.Close()
		t.Errorf("temporary pool not closed with the bucket")
	}
	if b.tempServers != nil || b.tempConnPools != nil {
		t.Errorf("temporary pools still recorded: %v %v", b.tempServers, b.tempConnPools)
	}
	if p := b.getTempConnPool(1, servers, "n3:11210"); p != nil {
		p.Close()
		t.Errorf("temporary pool created for a closed bucket")
	}
}

// a temporary pool is adopted by refresh only if it matches the current connection security config
func TestKVPoolConfigMatchesPool(t *testing.T) {
	ps := PoolServices{NodesExt: []NodeServices{
		{Hostname: "n3", Services: map[string]int{"kv": 11210, "kvSSL": 11207}},
		{Hostname: "", ThisNode: true, Services: map[string]int{"kv": 11210, "kvSSL": 11207}},
	}}
	oldTLS, newTLS := &tls.Config{}, &tls.Config{}
	noTLS := &kvPoolConfig{}
	withTLS := &kvPoolConfig{tlsConfig: newTLS, poolServices: ps}
	forced := &kvPoolConfig{tlsConfig: newTLS, disableNonSSLPorts: true, poolServices: ps}

	plainNoTLS := &connectionPool{host: "n3:11210"}
	plain := &connectionPool{host: "n3:11210", tlsConfig: newTLS}
	local := &connectionPool{host: "127.0.0.1:11210", tlsConfig: newTLS}
	localOld := &connectionPool{host: "127.0.0.1:11210", tlsConfig: oldTLS}
	encNew := &connectionPool{host: "n3:11207", tlsConfig: newTLS, encrypted: true}
	encOld := &connectionPool{host: "n3:11207", tlsConfig: oldTLS, encrypted: true}
	localEnc := &connectionPool{host: "127.0.0.1:11207", tlsConfig: newTLS, encrypted: true}

	for _, c := range []struct {
		name string
		cfg  *kvPoolConfig
		node string
		pool *connectionPool
		exp  bool
	}{
		{"no TLS, plain pool", noTLS, "n3:11210", plainNoTLS, true},
		{"no TLS, pool built under TLS", noTLS, "n3:11210", plain, false},
		{"no TLS, encrypted pool", noTLS, "n3:11210", encNew, false},
		{"TLS, current encrypted pool", withTLS, "n3:11210", encNew, true},
		{"TLS, encrypted pool with an old config", withTLS, "n3:11210", encOld, false},
		{"TLS, plain pool", withTLS, "n3:11210", plain, false},
		{"TLS, local node stays plain", withTLS, "127.0.0.1:11210", local, true},
		{"TLS, local plain pool from an old config", withTLS, "127.0.0.1:11210", localOld, false},
		{"non-SSL ports disabled, local plain pool", forced, "127.0.0.1:11210", local, false},
		{"non-SSL ports disabled, local encrypted pool", forced, "127.0.0.1:11210", localEnc, true},
		{"TLS, pool for another host", withTLS, "n9:11210", plain, false},
	} {
		if got := c.cfg.matchesPool(c.node, c.pool); got != c.exp {
			t.Errorf("%s: matchesPool = %v, expected %v", c.name, got, c.exp)
		}
	}
}

// a bucket with TLS on whose server list is n3 and the node being added, n4; the stub ns_server lists n4 in the node
// services only if withN4 is set
func refreshTLSBucket(t *testing.T, withN4 *atomic.Bool) (*Bucket, *Client) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pools/default/nodeServices":
			n4 := ""
			if withN4.Load() {
				n4 = `,{"hostname":"n4","services":{"kv":11210,"kvSSL":11207}}`
			}
			w.Write([]byte(`{"rev":1,"nodesExt":[{"hostname":"n3","services":{"kv":11210,"kvSSL":11207}}` + n4 + `]}`))
		case "/pools/default/buckets/default":
			w.Write([]byte(`{"name":"default","vBucketServerMap":{"numReplicas":0,` +
				`"serverList":["n3:11210","n4:11210"],"vBucketMap":[[0],[1]]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	client := &Client{BaseURL: u, ah: basicAuth{"u", "p"}, tlsConfig: &tls.Config{},
		Info: Pools{Pools: []RestPool{{Name: "default"}}}}
	b := &Bucket{Name: "default", UUID: "0123456789abcdef", URI: "/pools/default/buckets/default",
		ah: basicAuth{"u", "p"}, pool: &Pool{client: client}}
	t.Cleanup(b.Close)
	return b, client
}

// refresh adopts a temporary pool built with the current tls config, and closes and rebuilds one built with an older
// one
func TestRefreshTempPoolTLS(t *testing.T) {
	withN4 := &atomic.Bool{}
	withN4.Store(true)
	for _, stale := range []bool{false, true} {
		b, client := refreshTLSBucket(t, withN4)
		tlsCfg := client.tlsConfig
		if stale {
			tlsCfg = &tls.Config{}
		}
		temp := newConnectionPool("n4:11207", basicAuth{"u", "p"}, false, 1, 1, tlsCfg, "default", true)
		temp.server = "n4:11210"
		b.tempServers, b.tempConnPools = []string{"", "n4:11210"}, []*connectionPool{nil, temp}

		if err := b.refresh(true); err != nil {
			t.Fatalf("stale %v: refresh: %v", stale, err)
		}
		p := b.getConnPools(false)[1]
		if stale {
			if p == temp || !poolClosed(temp) || p.host != "n4:11207" || p.tlsConfig != client.tlsConfig {
				t.Errorf("stale temporary pool not replaced: %+v (closed %v)", p, poolClosed(temp))
			}
		} else if p != temp || poolClosed(temp) {
			t.Errorf("temporary pool not adopted: %+v (closed %v)", p, poolClosed(temp))
		}
		if b.tempServers != nil || b.tempConnPools != nil {
			t.Errorf("stale %v: temporary pools left after refresh: %v %v", stale, b.tempServers, b.tempConnPools)
		}
	}
}

// the node services don't list a node being added yet: refresh keeps its temporary pool rather than failing to
// build one
func TestRefreshTempPoolNodeNotListed(t *testing.T) {
	withN4 := &atomic.Bool{}
	b, client := refreshTLSBucket(t, withN4)
	temp := newConnectionPool("n4:11207", basicAuth{"u", "p"}, false, 1, 1, client.tlsConfig, "default", true)
	temp.server = "n4:11210"
	b.tempServers, b.tempConnPools = []string{"", "n4:11210"}, []*connectionPool{nil, temp}

	if err := b.refresh(true); err != nil {
		t.Fatalf("refresh failed for a node with a temporary pool: %v", err)
	}
	if p := b.getConnPools(false)[1]; p != temp || poolClosed(temp) {
		t.Errorf("temporary pool not kept: %+v (closed %v)", p, poolClosed(temp))
	}

	// a plaintext temporary pool (built before TLS was turned on) is closed, not adopted: unencrypted traffic to the
	// node doesn't carry on (however refresh then handles the node)
	b, _ = refreshTLSBucket(t, withN4)
	plain := newConnectionPool("n4:11210", basicAuth{"u", "p"}, false, 1, 1, nil, "default", false)
	plain.server = "n4:11210"
	b.tempServers, b.tempConnPools = []string{"", "n4:11210"}, []*connectionPool{nil, plain}
	b.refresh(true)
	for _, p := range b.getConnPools(false) {
		if p == plain {
			t.Errorf("plaintext temporary pool adopted under TLS")
		}
	}
	if !poolClosed(plain) {
		plain.Close()
		t.Errorf("plaintext temporary pool not closed")
	}
}
