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
	"strings"
	"testing"
)

// the bucket config the stub ns_server serves: a remote node n3, and the local node
const tlsPoolBucketJSON = `{"name":"default","nodes":[{"hostname":"127.0.0.1:8091"}],` +
	`"vBucketServerMap":{"numReplicas":0,"serverList":["n3:11210","127.0.0.1:11210"],"vBucketMap":[[0],[1]]}}`

// a bucket with TLS on (tls1) against a stub ns_server whose node services give both nodes a kvSSL port; the bucket
// streaming endpoint sends the bucket config streamed times
func tlsPoolBucket(t *testing.T, streamed int) (*Bucket, *Client, *tls.Config) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pools/default/nodeServices":
			w.Write([]byte(`{"rev":1,"nodesExt":[{"hostname":"n3","services":{"kv":11210,"kvSSL":11207}},` +
				`{"thisNode":true,"services":{"kv":11210,"kvSSL":11207}}]}`))
		case "/pools/default/buckets/default":
			w.Write([]byte(tlsPoolBucketJSON))
		case "/pools/default/bucketsStreaming/default":
			w.Write([]byte(strings.Repeat(tlsPoolBucketJSON+"\n\n\n\n", streamed)))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	tls1 := &tls.Config{}
	client := &Client{BaseURL: u, ah: basicAuth{"u", "p"}, tlsConfig: tls1,
		Info: Pools{Pools: []RestPool{{Name: "default"}}}}
	b := &Bucket{Name: "default", UUID: "0123456789abcdef", URI: "/pools/default/buckets/default", ah: basicAuth{"u", "p"},
		pool: &Pool{client: client}}
	t.Cleanup(func() { b.Close() })
	return b, client, tls1
}

// pools built under tls: n3's encrypted on its kvSSL port, and the local node's unencrypted. Each keeps the tls config
// it was built under, the local node's unencrypted one too, so memcached.go's security config change check
// (pool.tlsConfig != the client's) doesn't fire for it
func checkTLSPools(t *testing.T, what string, pools []*connectionPool, tls *tls.Config) {
	if len(pools) != 2 || pools[0].host != "n3:11207" || !pools[0].encrypted || pools[0].tlsConfig != tls ||
		pools[1].host != "127.0.0.1:11210" || pools[1].encrypted || pools[1].tlsConfig != tls {
		t.Fatalf("%s: unexpected pools %+v", what, pools)
	}
	if pools[0].dialTLSConfig() != tls || pools[1].dialTLSConfig() != nil {
		t.Errorf("%s: unencrypted pool dials with TLS, or encrypted pool without it", what)
	}
}

// pools after an update with the same security config are the ones before it, still open
func checkPoolsReused(t *testing.T, what string, before, after []*connectionPool) {
	for i, p := range after {
		if p != before[i] {
			t.Errorf("%s: pool %d (%s) rebuilt with the same config", what, i, before[i].host)
		}
		if poolClosed(before[i]) {
			t.Errorf("%s: reused pool %d (%s) closed", what, i, before[i].host)
		}
	}
}

// pools after an update with a new security config are new, and the old ones are closed; the unencrypted one too, as
// it was built under the old security config
func checkPoolsRebuilt(t *testing.T, what string, before, after []*connectionPool) {
	for i, p := range after {
		if p == before[i] || !poolClosed(before[i]) {
			t.Errorf("%s: pool %d (%s) not rebuilt with the new security config", what, i, before[i].host)
		}
	}
}

// refresh keeps the bucket's pools while the connection security config is unchanged, under TLS too (encrypted pools'
// hosts are the TLS port, and the local node's pool is unencrypted), and rebuilds them when it changes
func TestRefreshReusesTLSPools(t *testing.T) {
	b, client, tls1 := tlsPoolBucket(t, 0)

	if err := b.refresh(true); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	first := append([]*connectionPool{}, b.getConnPools(false)...)
	checkTLSPools(t, "first refresh", first, tls1)

	if err := b.refresh(true); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	checkPoolsReused(t, "second refresh", first, b.getConnPools(false))

	tls2 := &tls.Config{}
	client.tlsConfig = tls2
	if err := b.refresh(true); err != nil {
		t.Fatalf("third refresh: %v", err)
	}
	pools := b.getConnPools(false)
	checkTLSPools(t, "third refresh", pools, tls2)
	checkPoolsRebuilt(t, "third refresh", first, pools)
}

// the bucket streaming updater (UpdateBucket2) keeps the bucket's pools the same way: reused across vbmap updates
// while the connection security config is unchanged, rebuilt when it changes
func TestUpdateBucket2ReusesTLSPools(t *testing.T) {
	b, client, tls1 := tlsPoolBucket(t, 3)

	// the updater needs the bucket's node list and pools
	if err := b.refresh(true); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	first := append([]*connectionPool{}, b.getConnPools(false)...)
	checkTLSPools(t, "refresh", first, tls1)

	// each update runs the callback before the next one is decoded, so a config change in it applies to the next
	tls2 := &tls.Config{}
	updates := 0
	err := b.UpdateBucket2("test", func(_ *Bucket, _ string) bool {
		updates++
		pools := b.getConnPools(false)
		switch updates {
		case 1, 2:
			checkTLSPools(t, "update", pools, tls1)
			checkPoolsReused(t, "update", first, pools)
			if updates == 2 {
				client.tlsConfig = tls2
			}
		case 3:
			checkTLSPools(t, "update after the config change", pools, tls2)
			checkPoolsRebuilt(t, "update after the config change", first, pools)
		}
		return updates < 3
	})
	if err != nil {
		t.Fatalf("UpdateBucket2: %v", err)
	}
	if updates != 3 {
		t.Fatalf("%d updates, expected 3", updates)
	}
}
