//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"container/list"
	"crypto/tls"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/couchbase/gomemcached"
	qerrors "github.com/couchbase/query/errors"
)

func TestScanCancelSliceSort(t *testing.T) {
	b1 := &Bucket{Name: "b1"}
	b2 := &Bucket{Name: "b2"}
	cr := scanCancelSlice{
		{vbucket: 7, b: b2},
		{vbucket: 3, b: b1},
		{vbucket: 9, b: b1},
		{vbucket: 1, b: b2},
		{vbucket: 3, b: b1},
	}
	sort.Sort(cr)
	expected := []struct {
		name string
		vb   uint16
	}{{"b1", 3}, {"b1", 3}, {"b1", 9}, {"b2", 1}, {"b2", 7}}
	for i, e := range expected {
		if cr[i].b.Name != e.name || cr[i].vbucket != e.vb {
			t.Errorf("position %d: got %s/%d, expected %s/%d", i, cr[i].b.Name, cr[i].vbucket, e.name, e.vb)
		}
	}
}

func TestGroupsToParamFormat(t *testing.T) {
	cases := []struct {
		groups   []string
		expected string
	}{
		{[]string{"g1"}, "g1"},
		{[]string{"g1", "g2", "g3"}, "g1,g2,g3"},
		{[]string{"", "g1", "g2"}, "g1,g2"},
		{[]string{"g1", "", "g2", ""}, "g1,g2"},
		{[]string{"", ""}, ""},
	}
	for _, c := range cases {
		if got := groupsToParamFormat(c.groups); got != c.expected {
			t.Errorf("%q: got %q, expected %q", c.groups, got, c.expected)
		}
	}
}

func TestGetConnectionToVBucketBounds(t *testing.T) {
	b := &Bucket{vBucketServerMap: unsafe.Pointer(&VBucketServerMap{
		VBucketMap: [][]int{{-1}, {-1}, {-1}, {-1}},
	})}
	desc := &doDescriptor{}
	// vbucket == len(VBucketMap) must be rejected, not index past the end
	for _, vb := range []uint32{4, 5, 1024} {
		conn, pool, err := b.getConnectionToVBucket(vb, desc)
		if err == nil || conn != nil || pool != nil {
			t.Errorf("vb %d: expected an error, got conn %v pool %v err %v", vb, conn, pool, err)
		}
	}
	// last valid vbucket is looked up (and fails here only because it has no master)
	_, _, err := b.getConnectionToVBucket(3, desc)
	if err == nil || err.Error() != "primitives/couchbase: No master for vbucket 3" {
		t.Errorf("vb 3: unexpected error %v", err)
	}
}

func TestReadCurrentReusesHead(t *testing.T) {
	// keys "aaaa", "b", "cccccc" packed in the buffer; keys holds the cumulative end offsets
	this := &vbRangeScan{
		buffer: []byte("aaaabcccccc"),
		keys:   []uint32{4, 5, 11},
		head:   make([]byte, 0, 8),
	}
	head := unsafe.SliceData(this.head)
	for i, e := range []string{"aaaa", "b", "cccccc"} {
		this.currentKey = i
		if !this.readCurrent() {
			t.Fatalf("key %d: readCurrent failed", i)
		}
		if string(this.current()) != e {
			t.Errorf("key %d: got %q, expected %q", i, this.current(), e)
		}
		// a key that fits the capacity must not reallocate, even if longer than the previous key
		if unsafe.SliceData(this.head) != head {
			t.Errorf("key %d: head reallocated", i)
		}
	}
	this.currentKey = 3
	if this.readCurrent() {
		t.Errorf("readCurrent past the last key succeeded")
	}
}

func TestGetTempConnPool(t *testing.T) {
	// no TLS: getKVPoolConfig does not need ns_server
	b := &Bucket{Name: "default", ah: basicAuth{"u", "p"}, pool: &Pool{client: &Client{}}}
	defer func() {
		for _, p := range b.tempConnPools {
			if p != nil {
				p.Close()
			}
		}
	}()
	servers := []string{"n0:11210", "n1:11210", "n2:11210"}

	if b.getTempConnPool(-1, servers, "n3:11210") != nil || b.getTempConnPool(3, servers, "n3:11210") != nil ||
		b.getTempConnPool(1, servers, "") != nil {
		t.Errorf("invalid arguments returned a pool")
	}

	p1 := b.getTempConnPool(1, servers, "n3:11210")
	if p1 == nil || p1.host != "n3:11210" {
		t.Fatalf("expected a new pool for n3, got %v", p1)
	}
	if len(b.tempServers) != len(servers) || b.tempServers[1] != "n3:11210" || b.tempConnPools[1] != p1 {
		t.Errorf("temporary pool not recorded: %v %v", b.tempServers, b.tempConnPools)
	}

	// populated entry is reused
	if p := b.getTempConnPool(1, servers, "n3:11210"); p != p1 {
		t.Errorf("expected the existing pool to be reused, got %v", p)
	}

	// entry populated for a different node: the vbmap has changed again, ignore the node
	if p := b.getTempConnPool(1, servers, "n4:11210"); p != nil {
		t.Errorf("expected no pool for a node replacing a populated entry, got %v", p)
	}

	// a server list of a different size is ignored
	if p := b.getTempConnPool(1, servers[:2], "n3:11210"); p != nil {
		t.Errorf("expected no pool for a server list of a different size, got %v", p)
	}

	// a failed attempt (node recorded, no pool) is not retried
	b.tempServers[2] = "n5:11210"
	if p := b.getTempConnPool(2, servers, "n5:11210"); p != nil || b.tempConnPools[2] != nil {
		t.Errorf("expected a failed entry not to be retried, got %v", p)
	}
}

func TestGetConnPoolByServer(t *testing.T) {
	b := &Bucket{Name: "default", ah: basicAuth{"u", "p"}}
	plain, err := b.mkConnPool("n0:11210", &kvPoolConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	ps := PoolServices{NodesExt: []NodeServices{{Hostname: "n1", Services: map[string]int{"kv": 11210, "kvSSL": 11207}}}}
	tlsPool, err := b.mkConnPool("n1:11210", &kvPoolConfig{tlsConfig: &tls.Config{}, poolServices: ps})
	if err != nil {
		t.Fatal(err)
	}
	defer tlsPool.Close()
	if tlsPool.host != "n1:11207" || tlsPool.server != "n1:11210" || plain.host != "n0:11210" || plain.server != "n0:11210" {
		t.Fatalf("unexpected pool names: %s %s, %s %s", plain.host, plain.server, tlsPool.host, tlsPool.server)
	}
	pools := []*connectionPool{plain, nil, tlsPool}
	b.connPools = unsafe.Pointer(&pools)

	// a scan records its pool's server list name, so the pool is found whether or not TLS is on
	if b.getConnPoolByServer("n0:11210") != plain || b.getConnPoolByServer("n1:11210") != tlsPool {
		t.Errorf("pool not found by server list name")
	}
	if b.getConnPoolByServer("n1:11207") != nil || b.getConnPoolByServer("n2:11210") != nil {
		t.Errorf("unexpected pool found")
	}

	// a node the bucket's pools don't have yet (rebalance in) is reached through a temporary pool
	temp, err := b.mkConnPool("n2:11210", &kvPoolConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer temp.Close()
	b.tempConnPools = []*connectionPool{nil, nil, temp}
	if b.getConnPoolByServer("n2:11210") != temp {
		t.Errorf("temporary pool not found by server list name")
	}
}

type fakeCanceller struct {
	b           *Bucket
	node        string
	connected   bool
	goneNodes   map[string]bool  // nodes that no longer hold the buckets
	closed      map[*Bucket]bool // closed buckets
	downNodes   map[string]bool  // nodes connect fails for with a network error
	badNodes    map[string]bool  // nodes connect fails for, for bucket b1 only, with a non network error
	busyNodes   map[string]bool  // nodes whose pool is exhausted on connect
	hungNodes   map[string]bool  // nodes the cancel times out on
	noBucket    map[string]bool  // nodes the cancel gets NO_BUCKET from, for bucket b1
	noAccess    map[string]bool  // nodes the cancel gets EACCESS from, for bucket b1
	staleOnce   map[string]bool  // bucket/vb whose first cancel finds the connection closed
	staleTwice  map[string]bool  // bucket/vb whose first two cancels find the connection closed
	downOnStale bool             // the node goes down when a cancel finds the connection closed
	connects    []string
	cancels     []string
}

func (this *fakeCanceller) connectedTo(b *Bucket, node string) bool {
	return this.connected && this.b == b && this.node == node
}

func (this *fakeCanceller) connect(b *Bucket, node string) (cancelFailScope, error) {
	this.release()
	this.connects = append(this.connects, b.Name+"/"+node)
	switch {
	case this.closed[b]:
		return _CANCEL_FAIL_BUCKET, errScanGone
	case this.goneNodes[node]:
		return _CANCEL_FAIL_ONE, errScanGone
	case this.downNodes[node]:
		err := fmt.Errorf("dial tcp %s: connect: connection refused", node)
		return connectFailScope(err), err
	case this.badNodes[node] && b.Name == "b1":
		return _CANCEL_FAIL_BUCKET_NODE, fmt.Errorf("select bucket failed")
	case this.busyNodes[node]:
		return connectFailScope(ErrTimeout), ErrTimeout
	}
	this.b, this.node, this.connected = b, node, true
	return _CANCEL_FAIL_ONE, nil
}

func (this *fakeCanceller) cancel(vb uint16, uuid []byte) (cancelFailScope, error) {
	key := fmt.Sprintf("%s/%d", this.b.Name, vb)
	this.cancels = append(this.cancels, key)
	var err error
	switch {
	case this.hungNodes[this.node]:
		err = fmt.Errorf("read tcp 127.0.0.1:1->%s: i/o timeout", this.node)
	case this.noAccess[this.node] && this.b.Name == "b1":
		err = &gomemcached.MCResponse{Status: gomemcached.EACCESS}
	case this.noBucket[this.node] && this.b.Name == "b1":
		err = &gomemcached.MCResponse{Status: gomemcached.NO_BUCKET}
	case this.staleOnce[key]:
		delete(this.staleOnce, key)
		if this.downOnStale {
			if this.downNodes == nil {
				this.downNodes = map[string]bool{}
			}
			this.downNodes[this.node] = true
		}
		err = io.EOF
	case this.staleTwice[key]:
		delete(this.staleTwice, key)
		if this.staleOnce == nil {
			this.staleOnce = map[string]bool{}
		}
		this.staleOnce[key] = true
		err = io.EOF
	}
	scope, keep, err := cancelResult(err)
	if !keep {
		this.connected = false
	}
	return scope, err
}

func (this *fakeCanceller) release() {
	this.connected = false
}

// entries are bucket, node, vbucket triples
func newCancels(entries ...any) scanCancelSlice {
	cr := make(scanCancelSlice, 0, len(entries)/3)
	for i := 0; i < len(entries); i += 3 {
		cr = append(cr, &scanCancel{b: entries[i].(*Bucket), node: entries[i+1].(string), vbucket: uint16(entries[i+2].(int)),
			uuid: make([]byte, 16)})
	}
	sort.Sort(cr)
	return cr
}

func checkCancels(t *testing.T, name string, c *fakeCanceller, notSent, failed int,
	expConnects, expCancels []string, expNotSent, expFailed int) {

	if fmt.Sprint(c.connects) != fmt.Sprint(expConnects) {
		t.Errorf("%s: connects: got %v, expected %v", name, c.connects, expConnects)
	}
	if fmt.Sprint(c.cancels) != fmt.Sprint(expCancels) {
		t.Errorf("%s: cancels: got %v, expected %v", name, c.cancels, expCancels)
	}
	if notSent != expNotSent || failed != expFailed {
		t.Errorf("%s: got notSent %d, failed %d, expected %d, %d", name, notSent, failed, expNotSent, expFailed)
	}
}

func TestSendCancels(t *testing.T) {
	b1, b2, b3 := &Bucket{Name: "b1"}, &Bucket{Name: "b2"}, &Bucket{Name: "b3"}
	const n0, n1 = "n0:11210", "n1:11210"
	var failures []string

	// one connection per bucket and node, whatever the vbuckets
	c := &fakeCanceller{}
	notSent, failed, _ := sendCancels(newCancels(b1, n0, 4, b1, n0, 0, b1, n1, 1, b1, n0, 2, b2, n1, 3), c)
	checkCancels(t, "connections", c, notSent, failed,
		[]string{"b1/" + n0, "b1/" + n1, "b2/" + n1}, []string{"b1/0", "b1/2", "b1/4", "b1/1", "b2/3"}, 0, 0)

	// n1 is down (network error): tried once, then skipped for every bucket; the cancels for n0 continue
	c = &fakeCanceller{downNodes: map[string]bool{n1: true}}
	notSent, failed, _ = sendCancels(newCancels(b1, n0, 0, b1, n1, 1, b1, n1, 3, b2, n1, 5, b2, n0, 2, b3, n0, 4), c)
	checkCancels(t, "down node", c, notSent, failed,
		[]string{"b1/" + n0, "b1/" + n1, "b2/" + n0, "b3/" + n0}, []string{"b1/0", "b2/2", "b3/4"}, 3, 0)

	// a node that no longer holds the bucket (removed or failed over) took its scans with it: nothing to send
	c = &fakeCanceller{goneNodes: map[string]bool{n1: true}}
	notSent, failed, _ = sendCancels(newCancels(b1, n1, 1, b1, n1, 3, b1, n0, 0), c)
	checkCancels(t, "node gone", c, notSent, failed, []string{"b1/" + n0, "b1/" + n1}, []string{"b1/0"}, 0, 0)

	// a closed bucket has no scans left on any node: it is looked at once, and the other buckets carry on
	c = &fakeCanceller{closed: map[*Bucket]bool{b1: true}}
	notSent, failed, _ = sendCancels(newCancels(b1, n0, 0, b1, n1, 1, b1, n1, 3, b2, n1, 5), c)
	checkCancels(t, "closed bucket", c, notSent, failed, []string{"b1/" + n0, "b2/" + n1}, []string{"b2/5"}, 0, 0)

	// EACCESS (access refused, which can be particular to the node) skips the bucket on that node only
	c = &fakeCanceller{noAccess: map[string]bool{n0: true}}
	notSent, failed, failures = sendCancels(newCancels(b1, n0, 0, b1, n0, 2, b1, n1, 1, b2, n0, 5), c)
	checkCancels(t, "no access", c, notSent, failed, []string{"b1/" + n0, "b1/" + n1, "b2/" + n0},
		[]string{"b1/0", "b1/1", "b2/5"}, 1, 1)
	if len(failures) != 1 || !strings.HasPrefix(failures[0], "bucket b1 on node "+n0+": ") {
		t.Errorf("no access: failures %v", failures)
	}

	// a non network connect error on n1 for b1 skips n1 for b1 only
	c = &fakeCanceller{badNodes: map[string]bool{n1: true}}
	notSent, failed, _ = sendCancels(newCancels(b1, n1, 1, b1, n1, 3, b2, n1, 1), c)
	checkCancels(t, "bucket node", c, notSent, failed, []string{"b1/" + n1, "b2/" + n1}, []string{"b2/1"}, 2, 0)

	// a busy pool skips the node for that bucket only (the fake fails straight away; the real connect waits up to
	// _RS_CANCEL_TIMEOUT for a connection, so a saturated pool costs that once per bucket and node in a batch)
	c = &fakeCanceller{busyNodes: map[string]bool{n0: true}}
	notSent, failed, _ = sendCancels(newCancels(b1, n0, 0, b1, n0, 2, b1, n1, 1), c)
	checkCancels(t, "busy pool", c, notSent, failed, []string{"b1/" + n0, "b1/" + n1}, []string{"b1/1"}, 2, 0)

	// a cancel timing out on a hung node skips the node for every bucket
	c = &fakeCanceller{hungNodes: map[string]bool{n1: true}}
	notSent, failed, failures = sendCancels(newCancels(b1, n1, 1, b1, n1, 3, b1, n0, 0, b2, n1, 5), c)
	checkCancels(t, "hung node", c, notSent, failed, []string{"b1/" + n0, "b1/" + n1}, []string{"b1/0", "b1/1"}, 2, 1)
	if len(failures) != 1 || !strings.HasPrefix(failures[0], "node "+n1+": ") {
		t.Errorf("hung node: failures %v", failures)
	}

	// NO_BUCKET on a cancel: the bucket is gone from the node with its scans, so the bucket's other cancels on that
	// node have nothing to cancel, and none of them count as failed or not sent
	c = &fakeCanceller{noBucket: map[string]bool{n1: true}}
	notSent, failed, failures = sendCancels(newCancels(b1, n1, 1, b1, n1, 3, b1, n0, 0, b2, n1, 5), c)
	checkCancels(t, "no bucket", c, notSent, failed,
		[]string{"b1/" + n0, "b1/" + n1, "b2/" + n1}, []string{"b1/0", "b1/1", "b2/5"}, 0, 0)
	if len(failures) != 0 {
		t.Errorf("no bucket: failures %v", failures)
	}

	// a connection closed while idle is retried once on a new connection, and the node stays usable
	c = &fakeCanceller{staleOnce: map[string]bool{"b1/0": true}}
	notSent, failed, _ = sendCancels(newCancels(b1, n0, 0, b1, n0, 2), c)
	checkCancels(t, "stale connection", c, notSent, failed,
		[]string{"b1/" + n0, "b1/" + n0}, []string{"b1/0", "b1/0", "b1/2"}, 0, 0)

	// a second closed connection on the retry fails only that cancel
	c = &fakeCanceller{staleTwice: map[string]bool{"b1/0": true}}
	notSent, failed, failures = sendCancels(newCancels(b1, n0, 0, b1, n0, 2), c)
	checkCancels(t, "stale twice", c, notSent, failed,
		[]string{"b1/" + n0, "b1/" + n0, "b1/" + n0}, []string{"b1/0", "b1/0", "b1/2"}, 0, 1)
	// the failed count has a reason
	if len(failures) != 1 || !strings.HasPrefix(failures[0], "first cancel failure: bucket b1 on node "+n0+", vb 0: ") {
		t.Errorf("stale twice: failures %v", failures)
	}

	// a failed reconnect on the retry fails the cancel (it was sent once), and skips the node
	c = &fakeCanceller{staleOnce: map[string]bool{"b1/0": true}, downOnStale: true}
	notSent, failed, _ = sendCancels(newCancels(b1, n0, 0, b1, n0, 2), c)
	checkCancels(t, "stale then down", c, notSent, failed, []string{"b1/" + n0, "b1/" + n0}, []string{"b1/0"}, 1, 1)

	// two bucket objects with the same name (dropped and recreated) sort into separate groups
	b1new := &Bucket{Name: "b1"}
	c = &fakeCanceller{}
	notSent, failed, _ = sendCancels(newCancels(b1, n0, 0, b1new, n0, 0, b1, n0, 1, b1new, n0, 1), c)
	checkCancels(t, "same name", c, notSent, failed,
		[]string{"b1/" + n0, "b1/" + n0}, []string{"b1/0", "b1/1", "b1/0", "b1/1"}, 0, 0)
}

func TestCancelFailScopes(t *testing.T) {
	connectCases := []struct {
		err error
		exp cancelFailScope
	}{
		{fmt.Errorf("dial tcp 10.0.0.1:11210: connect: connection refused"), _CANCEL_FAIL_NODE},
		{fmt.Errorf("dial tcp 10.0.0.1:11210: i/o timeout"), _CANCEL_FAIL_NODE},
		{fmt.Errorf("dial tcp 10.0.0.1:11210: connect: no route to host"), _CANCEL_FAIL_NODE},
		{io.EOF, _CANCEL_FAIL_NODE},
		{ErrTimeout, _CANCEL_FAIL_BUCKET_NODE},
		{errHostChanged, _CANCEL_FAIL_BUCKET_NODE},
		{fmt.Errorf("%s: default", BUCKET_NOT_FOUND), _CANCEL_FAIL_BUCKET},
		{fmt.Errorf("%s: default", BUCKET_UUID_MISMATCH), _CANCEL_FAIL_BUCKET},
		{fmt.Errorf("auth failed"), _CANCEL_FAIL_BUCKET_NODE},
		// may be particular to the node
		{&gomemcached.MCResponse{Status: gomemcached.AUTH_ERROR}, _CANCEL_FAIL_BUCKET_NODE},
		{&gomemcached.MCResponse{Status: gomemcached.EACCESS}, _CANCEL_FAIL_BUCKET_NODE},
		{fmt.Errorf("Auth failure"), _CANCEL_FAIL_BUCKET_NODE},
		{&gomemcached.MCResponse{Status: gomemcached.KEY_ENOENT}, _CANCEL_FAIL_BUCKET_NODE},
	}
	for _, c := range connectCases {
		if got := connectFailScope(c.err); got != c.exp {
			t.Errorf("connect %v: got %v, expected %v", c.err, got, c.exp)
		}
	}

	cancelCases := []struct {
		err   error
		scope cancelFailScope
		keep  bool
		isErr bool
	}{
		{nil, _CANCEL_FAIL_ONE, true, false},
		{&gomemcached.MCResponse{Status: gomemcached.KEY_ENOENT}, _CANCEL_FAIL_ONE, true, false},
		{&gomemcached.MCResponse{Status: gomemcached.CONFIG_ONLY}, _CANCEL_FAIL_BUCKET_NODE, false, true},
		{&gomemcached.MCResponse{Status: gomemcached.NO_BUCKET}, _CANCEL_FAIL_BUCKET_NODE, false, true},
		{&gomemcached.MCResponse{Status: gomemcached.EACCESS}, _CANCEL_FAIL_BUCKET_NODE, false, true},
		{&gomemcached.MCResponse{Status: gomemcached.NOT_MY_VBUCKET}, _CANCEL_FAIL_ONE, true, false},
		{&gomemcached.MCResponse{Status: gomemcached.EINVAL}, _CANCEL_FAIL_ONE, true, true},
		{fmt.Errorf("read tcp 127.0.0.1:1->127.0.0.1:11210: i/o timeout"), _CANCEL_FAIL_NODE, false, true},
		{io.EOF, _CANCEL_FAIL_STALE, false, true},
		{fmt.Errorf("write tcp 127.0.0.1:1->127.0.0.1:11210: write: broken pipe"), _CANCEL_FAIL_STALE, false, true},
		{fmt.Errorf("read tcp 127.0.0.1:1->127.0.0.1:11210: read: connection reset by peer"), _CANCEL_FAIL_STALE, false, true},
		{fmt.Errorf("something else"), _CANCEL_FAIL_ONE, false, true},
	}
	for _, c := range cancelCases {
		scope, keep, err := cancelResult(c.err)
		if scope != c.scope || keep != c.keep || (err != nil) != c.isErr {
			t.Errorf("cancel %v: got %v %v %v, expected %v %v error:%v", c.err, scope, keep, err, c.scope, c.keep, c.isErr)
		}
	}
}

// Cancelling a seq scan's queued vbscans queues a cancel for the scan each throttled one left open on KV, without
// taking _RSW's read lock twice: a second read lock waits behind a pending write lock (monitorWorkers), which waits for
// the first.
func TestCancelQueuedScansOpenScans(t *testing.T) {
	saved := _RSW
	defer func() { _RSW = saved }()
	cqueue := &rswCancelQueue{}
	cqueue.cond.L = cqueue
	q := &rswQueue{scans: list.New(), cqueue: cqueue}
	q.cond.L = q
	_RSW = &rswControl{queues: []*rswQueue{q}}

	b := &Bucket{Name: "b1"}
	stop := make(chan bool)
	writerDone := make(chan bool)
	go func() {
		defer close(writerDone)
		for {
			select {
			case <-stop:
				return
			default:
				_RSW.Lock()
				_RSW.Unlock()
			}
		}
	}()

	done := make(chan bool)
	go func() {
		defer close(done)
		for i := 0; i < 2000; i++ {
			ss, other := &seqScan{}, &seqScan{}
			q.addScan(&vbRangeScan{scan: ss, b: b, vb: 1, uuid: []byte{1}, uuidNode: "n0:11210"})
			q.addScan(&vbRangeScan{scan: ss, b: b, vb: 2}) // nothing open
			q.addScan(&vbRangeScan{scan: other, b: b, vb: 3, uuid: []byte{3}, uuidNode: "n1:11210"})
			_RSW.cancelQueuedScans(ss)
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("cancelQueuedScans deadlocked")
	}
	close(stop)
	<-writerDone

	// one cancel for each open scan of the cancelled seq scans, sent to its node; the other seq scan's vbscans stay queued
	if len(cqueue.scans) != 2000 || q.scans.Len() != 2000 {
		t.Fatalf("got %d cancels and %d queued vbscans, expected 2000 of each", len(cqueue.scans), q.scans.Len())
	}
	for _, sc := range cqueue.scans {
		if sc.vbucket != 1 || sc.node != "n0:11210" || sc.b != b {
			t.Fatalf("unexpected cancel %+v", sc)
		}
	}
	for e := q.scans.Front(); e != nil; e = e.Next() {
		if vbs := e.Value.(*vbRangeScan); vbs.vb != 3 || vbs.uuid == nil {
			t.Fatalf("unexpected queued vbscan %v", vbs.vb)
		}
	}
}

// A vbscan that reports an error queues a cancel for the scan it left open on KV (e.g. throttled, then cancelled or
// failed to reconnect), to the node holding it, and only once.
func TestReportErrorCancelsOpenScan(t *testing.T) {
	saved := _RSW
	defer func() { _RSW = saved }()
	cqueue := &rswCancelQueue{}
	cqueue.cond.L = cqueue
	q := &rswQueue{scans: list.New(), cqueue: cqueue}
	q.cond.L = q
	_RSW = &rswControl{queues: []*rswQueue{q}}

	b := &Bucket{Name: "b1"}
	ss := &seqScan{}
	ss.readyQueue.init()
	open := &vbRangeScan{scan: ss, b: b, vb: 7, uuid: []byte{7}, uuidNode: "n1:11210", state: _VBS_WORKING}
	open.reportError(qerrors.NewSSError(qerrors.E_SS_CONN))
	open.reportError(qerrors.NewSSError(qerrors.E_SS_CONN))
	(&vbRangeScan{scan: ss, b: b, vb: 8, state: _VBS_WORKING}).reportError(qerrors.NewSSError(qerrors.E_SS_CONN))
	if len(cqueue.scans) != 1 || cqueue.scans[0].vbucket != 7 || cqueue.scans[0].node != "n1:11210" || open.uuid != nil {
		t.Fatalf("expected one cancel for vb 7 on n1, got %d", len(cqueue.scans))
	}
}

// queueCancel, which now also runs outside the scan worker, reads the cancel queue under _RSW's read lock, as
// monitorWorkers clears it under the write lock when it removes the queue as idle (the race detector catches a read
// after the unlock); a cancel for a removed queue is dropped
func TestQueueCancelQueueRemoved(t *testing.T) {
	saved := _RSW
	defer func() { _RSW = saved }()
	cqueue := &rswCancelQueue{}
	cqueue.cond.L = cqueue
	q := &rswQueue{scans: list.New(), cqueue: cqueue}
	q.cond.L = q
	_RSW = &rswControl{queues: []*rswQueue{q}}

	stop := make(chan bool)
	monitorDone := make(chan bool)
	go func() {
		defer close(monitorDone)
		for {
			select {
			case <-stop:
				return
			default:
				// what monitorWorkers does to an idle queue, then put it back for the next round
				_RSW.Lock()
				q.Lock()
				q.cqueue = nil
				q.Unlock()
				_RSW.Unlock()
				_RSW.Lock()
				q.Lock()
				q.cqueue = cqueue
				q.Unlock()
				_RSW.Unlock()
			}
		}
	}()
	b := &Bucket{Name: "b1"}
	for i := 0; i < 2000; i++ {
		_RSW.queueCancel(&vbRangeScan{b: b, vb: 1}, []byte{1}, "n0:11210")
	}
	close(stop)
	<-monitorDone
	cqueue.Lock()
	n := len(cqueue.scans)
	cqueue.Unlock()
	if n > 2000 {
		t.Fatalf("%d cancels queued for 2000 calls", n)
	}
}

// Where a new scan starts after a throttled scan is lost (e.g. its vbucket moved): after the last key received, else
// where the run that created the scan started; a run that continued an earlier run's scan keeps the earlier position.
func TestSetResumePoint(t *testing.T) {
	startOf := func(this *vbRangeScan) string {
		if this.continueFrom == nil {
			return "<range start>"
		}
		v, excl := this.startFrom()
		return fmt.Sprintf("%s/%v", v, excl)
	}

	// throttled twice with no keys: created from "k1", then resumed (start nil) and throttled again
	this := &vbRangeScan{}
	this.setResumePoint([]byte("k1"), false)
	this.setResumePoint(nil, false)
	if s := startOf(this); s != "k1/false" {
		t.Errorf("no keys, throttled twice: got %s, expected k1/false", s)
	}

	// a resumed run throttled with nothing saved before leaves the range start (not an empty key)
	this = &vbRangeScan{}
	this.setResumePoint(nil, false)
	if s := startOf(this); s != "<range start>" {
		t.Errorf("no keys, resumed: got %s, expected the range start", s)
	}

	// keys received: after the last one, however the run started
	for _, start := range [][]byte{[]byte("k1"), nil} {
		this = &vbRangeScan{buffer: append(make([]byte, 0, 64), "aaaabcccccc"...), keys: []uint32{4, 5, 11}, offset: 11,
			head: make([]byte, 0, 8)}
		this.setResumePoint([]byte("k0"), false)
		this.setResumePoint(start, false)
		if s := startOf(this); s != "cccccc/true" {
			t.Errorf("keys, start %q: got %s, expected cccccc/true", start, s)
		}
		if this.currentKey != 0 {
			t.Errorf("current key moved to %d", this.currentKey)
		}
	}

	// a sample scan has no position
	this = &vbRangeScan{sampleSize: 10}
	this.setResumePoint([]byte("k1"), false)
	if this.continueFrom != nil {
		t.Errorf("sample scan: position saved")
	}
}
