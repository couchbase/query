//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

var samplePools = `{
    "componentsVersion": {
        "ale": "eff9516",
        "couch": "1.2.0a-7dd003e-git",
        "couch_set_view": "1.2.0a-7dd003e-git",
        "crypto": "2.0.3",
        "ibrowse": "2.2.0",
        "inets": "5.6",
        "kernel": "2.14.4",
        "mnesia": "4.4.19",
        "mochiweb": "1.4.1",
        "ns_server": "2.0.0r-388-gf35126e-community",
        "oauth": "7d85d3ef",
        "os_mon": "2.2.6",
        "public_key": "0.12",
        "sasl": "2.1.9.4",
        "ssl": "4.1.5",
        "stdlib": "1.17.4"
    },
    "implementationVersion": "2.0.0r-388-gf35126e-community",
    "isAdminCreds": false,
    "pools": [
        {
            "name": "default",
            "streamingUri": "/poolsStreaming/default",
            "uri": "/pools/default"
        }
    ],
    "uuid": "debbb353c26a6ee1c9eceb748a8c6907"
}`

var samplePool = `{
    "alerts": [],
    "autoCompactionSettings": {
        "databaseFragmentationThreshold": 80,
        "parallelDBAndViewCompaction": false,
        "viewFragmentationThreshold": 80
    },
    "balanced": true,
    "buckets": {
        "uri": "/pools/default/buckets?v=118084983"
    },
    "controllers": {
        "addNode": {
            "uri": "/controller/addNode"
        },
        "ejectNode": {
            "uri": "/controller/ejectNode"
        },
        "failOver": {
            "uri": "/controller/failOver"
        },
        "reAddNode": {
            "uri": "/controller/reAddNode"
        },
        "rebalance": {
            "uri": "/controller/rebalance"
        },
        "replication": {
            "createURI": "/controller/createReplication",
            "infosURI": "/couchBase/_replicator/_design/_replicator_info/_view/infos?group_level=1",
            "replicatorDBURI": "/couchBase/_replicator"
        },
        "setAutoCompaction": {
            "uri": "/controller/setAutoCompaction",
            "validateURI": "/controller/setAutoCompaction?just_validate=1"
        }
    },
    "counters": {
        "rebalance_start": 1,
        "rebalance_success": 1
    },
    "failoverWarnings": [],
    "name": "default",
    "nodeStatusesUri": "/nodeStatuses",
    "nodes": [
        {
            "clusterCompatibility": 1,
            "clusterMembership": "active",
            "couchApiBase": "http://10.203.6.236:8092/",
            "hostname": "10.203.6.236:8091",
            "alternateAddresses": {
                "external": {
                    "hostname": "server0.example.com"
                }
            },
            "interestingStats": {
                "curr_items": 0,
                "curr_items_tot": 0,
                "vb_replica_curr_items": 0
            },
            "mcdMemoryAllocated": 5978,
            "mcdMemoryReserved": 5978,
            "memoryFree": 6891118592,
            "memoryTotal": 7836254208,
            "os": "x86_64-unknown-linux-gnu",
            "ports": {
                "direct": 11210,
                "proxy": 11211
            },
            "status": "healthy",
            "systemStats": {
                "cpu_utilization_rate": 0.5025125628140703,
                "swap_total": 4294963200,
                "swap_used": 0
            },
            "thisNode": true,
            "uptime": "20516",
            "version": "2.0.0r-388-gf35126e-community"
        },
        {
            "clusterCompatibility": 1,
            "clusterMembership": "active",
            "couchApiBase": "http://10.32.21.163:8092/",
            "hostname": "10.32.21.163:8091",
            "alternateAddresses": {
                "external": {
                    "hostname": "server1.example.com"
                }
            },
            "interestingStats": {
                "curr_items": 0,
                "curr_items_tot": 0,
                "vb_replica_curr_items": 0
            },
            "mcdMemoryAllocated": 5978,
            "mcdMemoryReserved": 5978,
            "memoryFree": 6959566848,
            "memoryTotal": 7836254208,
            "os": "x86_64-unknown-linux-gnu",
            "ports": {
                "direct": 11210,
                "proxy": 11211
            },
            "status": "healthy",
            "systemStats": {
                "cpu_utilization_rate": 0.7575757575757576,
                "swap_total": 4294963200,
                "swap_used": 0
            },
            "uptime": "20523",
            "version": "2.0.0r-388-gf35126e-community"
        },
        {
            "clusterCompatibility": 1,
            "clusterMembership": "active",
            "couchApiBase": "http://10.98.83.17:8092/",
            "hostname": "10.98.83.17:8091",
            "alternateAddresses": {
                "external": {
                    "hostname": "server2.example.com"
                }
            },
            "interestingStats": {
                "curr_items": 0,
                "curr_items_tot": 0,
                "vb_replica_curr_items": 0
            },
            "mcdMemoryAllocated": 5978,
            "mcdMemoryReserved": 5978,
            "memoryFree": 6960541696,
            "memoryTotal": 7836254208,
            "os": "x86_64-unknown-linux-gnu",
            "ports": {
                "direct": 11210,
                "proxy": 11211
            },
            "status": "healthy",
            "systemStats": {
                "cpu_utilization_rate": 0.24213075060532688,
                "swap_total": 4294963200,
                "swap_used": 0
            },
            "uptime": "20505",
            "version": "2.0.0r-388-gf35126e-community"
        },
        {
            "clusterCompatibility": 1,
            "clusterMembership": "active",
            "couchApiBase": "http://10.34.21.232:8092/",
            "hostname": "10.34.21.232:8091",
            "alternateAddresses": {
                "external": {
                    "hostname": "server3.example.com"
                }
            },
            "interestingStats": {
                "curr_items": 0,
                "curr_items_tot": 0,
                "vb_replica_curr_items": 0
            },
            "mcdMemoryAllocated": 5978,
            "mcdMemoryReserved": 5978,
            "memoryFree": 6961504256,
            "memoryTotal": 7836254208,
            "os": "x86_64-unknown-linux-gnu",
            "ports": {
                "direct": 11210,
                "proxy": 11211
            },
            "status": "healthy",
            "systemStats": {
                "cpu_utilization_rate": 0.7334963325183375,
                "swap_total": 4294963200,
                "swap_used": 0
            },
            "uptime": "20528",
            "version": "2.0.0r-388-gf35126e-community"
        },
        {
            "clusterCompatibility": 1,
            "clusterMembership": "active",
            "couchApiBase": "http://10.203.33.4:8092/",
            "hostname": "10.203.33.4:8091",
            "alternateAddresses": {
                "external": {
                    "hostname": "server4.example.com"
                }
            },
            "interestingStats": {
                "curr_items": 0,
                "curr_items_tot": 0,
                "vb_replica_curr_items": 0
            },
            "mcdMemoryAllocated": 5978,
            "mcdMemoryReserved": 5978,
            "memoryFree": 6960599040,
            "memoryTotal": 7836254208,
            "os": "x86_64-unknown-linux-gnu",
            "ports": {
                "direct": 11210,
                "proxy": 11211
            },
            "status": "healthy",
            "systemStats": {
                "cpu_utilization_rate": 0.7575757575757576,
                "swap_total": 4294963200,
                "swap_used": 0
            },
            "uptime": "20537",
            "version": "2.0.0r-388-gf35126e-community"
        }
    ],
    "rebalanceProgressUri": "/pools/default/rebalanceProgress",
    "rebalanceStatus": "none",
    "remoteClusters": {
        "uri": "/pools/default/remoteClusters",
        "validateURI": "/pools/default/remoteClusters?just_validate=1"
    },
    "stats": {
        "uri": "/pools/default/stats"
    },
    "stopRebalanceUri": "/controller/stopRebalance",
    "storageTotals": {
        "hdd": {
            "free": 1046325215240,
            "quotaTotal": 1056894156800,
            "total": 1056894156800,
            "used": 10568941560,
            "usedByData": 12543880
        },
        "ram": {
            "quotaTotal": 31341936640,
            "quotaUsed": 31341936640,
            "total": 39181271040,
            "used": 4447940608,
            "usedByData": 13557744
        }
    },
    "tasksProgressUri": "/pools/default/tasksProgress",
    "tasksStatus": "none"
}`

func assert(t *testing.T, name string, got any, expected any) {
	if got != expected {
		t.Fatalf("Expected %v for %s, got %v", expected, name, got)
	}
}

func testParse(t *testing.T, s string, rv any) {
	if err := json.Unmarshal([]byte(s), rv); err != nil {
		t.Fatalf("Error decoding:  %v", err)
	}
}

func TestPoolsResponse(t *testing.T) {
	res := Pools{}
	testParse(t, samplePools, &res)

	assert(t, "couch", res.ComponentsVersion["couch"],
		"1.2.0a-7dd003e-git")
	assert(t, "implementationVersion", res.ImplementationVersion,
		"2.0.0r-388-gf35126e-community")
	assert(t, "uuid", res.UUID, "debbb353c26a6ee1c9eceb748a8c6907")
	assert(t, "IsAdmin", res.IsAdmin, false)
	assert(t, "pool name", res.Pools[0].Name, "default")
	assert(t, "pool streamingUri", res.Pools[0].StreamingURI,
		"/poolsStreaming/default")
	assert(t, "pool URI", res.Pools[0].URI, "/pools/default")
}

func TestPool(t *testing.T) {
	res := Pool{}
	testParse(t, samplePool, &res)
	assert(t, "len(pools)", 5, len(res.Nodes))
}

func TestPoolAlternateNames(t *testing.T) {
	res := Pool{}
	testParse(t, samplePool, &res)
	assert(t, "node[0].AlternateAddresses[\"external\"].Hostname",
		res.Nodes[0].AlternateNames["external"].Hostname, "server0.example.com")
	assert(t, "node[1].AlternateAddresses[\"external\"].Hostname",
		res.Nodes[1].AlternateNames["external"].Hostname, "server1.example.com")
	assert(t, "node[2].AlternateAddresses[\"external\"].Hostname",
		res.Nodes[2].AlternateNames["external"].Hostname, "server2.example.com")
	assert(t, "node[3].AlternateAddresses[\"external\"].Hostname",
		res.Nodes[3].AlternateNames["external"].Hostname, "server3.example.com")
	assert(t, "node[4].AlternateAddresses[\"external\"].Hostname",
		res.Nodes[4].AlternateNames["external"].Hostname, "server4.example.com")
}

func TestBucketConnPool(t *testing.T) {
	b := Bucket{}
	b.replaceConnPools([]*connectionPool{})
	p := b.getConnPool(3)
	if p != nil {
		t.Fatalf("Successfully got a pool where there was none: %v", p)
	}
	// TODO: I have a few more cases to cover here.
}

// No assertions, but this is meant to be tested with the race
// detector to verify the connection pool stuff is clean.
func TestBucketConnPoolConcurrent(t *testing.T) {
	b := Bucket{}

	wg := sync.WaitGroup{}
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				b.replaceConnPools([]*connectionPool{})
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

// MB-75083: a newly created system bucket whose vBucket map / server list has not been
// published yet must be waited for, not treated as a fatal error
func TestBucketTransientError(t *testing.T) {
	uri := "/pools/default/buckets/QUERY_METADATA?bucket_uuid=572180496b804fcda692c437a3e58faf"
	tests := []struct {
		err       error
		transient bool
	}{
		{&BucketNotFoundError{bucket: "QUERY_METADATA"}, true},
		{&BucketNotReadyError{uri: uri, reason: "empty vBucketMap"}, true},
		{&BucketNotReadyError{uri: uri, reason: "empty serverList"}, true},
		{fmt.Errorf("HTTP error 404 - bucket being deleted"), true},
		{fmt.Errorf("HTTP error 401 Unauthorized"), false},
		{fmt.Errorf("connection refused"), false},
	}
	for _, test := range tests {
		if rv := isBucketTransientError(test.err); rv != test.transient {
			t.Errorf("isBucketTransientError(%q) = %v, expected %v", test.err, rv, test.transient)
		}
	}

	// the error message must not change
	err := &BucketNotReadyError{uri: uri, reason: "empty vBucketMap"}
	if exp := "Invalid URL (" + uri + ") response: empty vBucketMap"; err.Error() != exp {
		t.Errorf("BucketNotReadyError.Error() = %q, expected %q", err.Error(), exp)
	}
}

func newDownNodesTestMap() *VBucketServerMap {
	return &VBucketServerMap{
		NumReplicas: 2,
		ServerList:  []string{"a:11210", "b:11210", "c:11210"},
		VBucketMap:  [][]int{{0, 1, 2}, {1, 2, -1}, {-1, 0, 1}},
	}
}

// like a real bucket, which has its own decoded copy of the map besides the live one that MarkDown() updates
func newDownNodesTestBucket(vbm *VBucketServerMap) *Bucket {
	return &Bucket{
		VBSMJson:         VBucketServerMap{NumReplicas: vbm.NumReplicas, ServerList: vbm.ServerList, VBucketMap: vbm.VBucketMap},
		vBucketServerMap: unsafe.Pointer(vbm),
	}
}

func TestMarkDown(t *testing.T) {
	vbm := newDownNodesTestMap()
	for n := -1; n <= len(vbm.ServerList); n++ {
		if vbm.IsDown(n) {
			t.Errorf("node %d down before any MarkDown", n)
		}
	}

	if marked := vbm.MarkDown(0, 1); len(marked) != 1 || marked[0] != "a:11210" {
		t.Errorf("MarkDown(0, 1) marked %v, expected [a:11210]", marked)
	}
	if !vbm.IsDown(0) || vbm.IsDown(1) || vbm.IsDown(2) {
		t.Errorf("after MarkDown(0, 1): down %v %v %v, expected true false false", vbm.IsDown(0), vbm.IsDown(1),
			vbm.IsDown(2))
	}

	// already down: not reported again
	if marked := vbm.MarkDown(0, 1); len(marked) != 0 {
		t.Errorf("second MarkDown(0, 1) marked %v, expected none", marked)
	}

	// read from the second replica: both nodes ahead of it are down, only the new one is reported
	if marked := vbm.MarkDown(0, 2); len(marked) != 1 || marked[0] != "b:11210" {
		t.Errorf("MarkDown(0, 2) marked %v, expected [b:11210]", marked)
	}

	// no active copy
	if marked := vbm.MarkDown(2, 1); len(marked) != 0 {
		t.Errorf("MarkDown(2, 1) marked %v, expected none", marked)
	}
	if vbm.IsDown(-1) || vbm.IsDown(len(vbm.ServerList)) {
		t.Errorf("out of range node reported down")
	}
}

func TestMarkDownExpiry(t *testing.T) {
	defer func(d time.Duration) { downNodeExpiry = d }(downNodeExpiry)
	downNodeExpiry = 50 * time.Millisecond

	vbm := newDownNodesTestMap()
	vbm.MarkDown(1, 1)
	if !vbm.IsDown(1) {
		t.Fatalf("node 1 not down after MarkDown(1, 1)")
	}
	time.Sleep(2 * downNodeExpiry)
	if vbm.IsDown(1) {
		t.Errorf("node 1 still down after the mark expired")
	}

	// once expired, a new mark is reported again
	if marked := vbm.MarkDown(1, 1); len(marked) != 1 || marked[0] != "b:11210" {
		t.Errorf("MarkDown(1, 1) after expiry marked %v, expected [b:11210]", marked)
	}
	if !vbm.IsDown(1) {
		t.Errorf("node 1 not down after it was marked again")
	}
}

// meant to be run with the race detector: replica reads check the mark while others set it
func TestMarkDownConcurrent(t *testing.T) {
	vbm := newDownNodesTestMap()

	var reported atomic.Int32
	wg := sync.WaitGroup{}
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 100; j++ {
				reported.Add(int32(len(vbm.MarkDown(0, 1))))
				vbm.IsDown(0)
				vbm.IsDown(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if r := reported.Load(); r != 1 {
		t.Errorf("node 0 reported as newly down %d times, expected 1", r)
	}
}

// the vitals debug output reports the nodes currently marked down
// the debug output marshals the bucket, so the marks must show through Bucket.MarshalJSON(), i.e. from the live map
func TestMarkDownMarshal(t *testing.T) {
	vbm := newDownNodesTestMap()
	b := newDownNodesTestBucket(vbm)
	downNodes := func() string {
		var m struct {
			VBSMJson map[string]any
		}
		bytes, err := json.Marshal(b)
		if err == nil {
			err = json.Unmarshal(bytes, &m)
		}
		if err != nil {
			t.Fatalf("marshalling the bucket: %v", err)
		}
		return fmt.Sprint(m.VBSMJson["DownNodes"])
	}

	if d := downNodes(); d != "<nil>" {
		t.Errorf("DownNodes before any MarkDown: %v, expected <nil>", d)
	}
	vbm.MarkDown(1, 1)
	if d := downNodes(); d != "[false true false]" {
		t.Errorf("DownNodes after MarkDown(1, 1): %v, expected [false true false]", d)
	}
}

// marshalling the live map while it's being marked down must not race (run with -race)
func TestMarkDownMarshalConcurrent(t *testing.T) {
	vbm := newDownNodesTestMap()
	b := newDownNodesTestBucket(vbm)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			vbm.MarkDown(0, 2)
		}()
		go func() {
			defer wg.Done()
			<-start
			if _, err := json.Marshal(b); err != nil {
				t.Errorf("marshalling the bucket: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
}

// a streaming config update can install a map with no vbuckets yet, and the bucket may have no map at all
func TestMarshalEmptyVBucketMap(t *testing.T) {
	for _, b := range []*Bucket{{vBucketServerMap: unsafe.Pointer(&VBucketServerMap{})}, {}} {
		if _, err := json.Marshal(b); err != nil {
			t.Errorf("marshalling the bucket: %v", err)
		}
	}
}
