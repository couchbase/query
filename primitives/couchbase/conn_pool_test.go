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
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/couchbase/gomemcached"
	"github.com/couchbase/gomemcached/client"
)

type testT struct {
	closed bool
}

func (t testT) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (t testT) Write([]byte) (int, error) {
	return 0, io.EOF
}

func (t testT) SetReadDeadline(time.Time) error {
	return nil
}

func (t testT) SetDeadline(time.Time) error {
	return nil
}

var errAlreadyClosed = errors.New("already closed")

func (t *testT) Close() error {
	if t.closed {
		return errAlreadyClosed
	}
	t.closed = true
	return nil
}

func testMkConn(h string, ah AuthHandler, tlsConfig *tls.Config, bucketName string) (*memcached.Client, string, error) {
	c, e := memcached.Wrap(&testT{})
	return c, "", e
}

func TestConnPool(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 6, nil, "", false)
	cp.mkConn = testMkConn

	seenClients := map[*memcached.Client]bool{}

	// build some connections

	for i := 0; i < 5; i++ {
		sc, err := cp.Get()
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		seenClients[sc] = true
	}

	if len(cp.connections) != 0 {
		t.Errorf("Expected 0 connections after gets, got %v",
			len(cp.connections))
	}

	// return them
	for k := range seenClients {
		cp.Return(k)
	}

	if len(cp.connections) != 3 {
		t.Errorf("Expected 3 connections after returning them, got %v",
			len(cp.connections))
	}

	// Try again.
	matched := 0
	grabbed := []*memcached.Client{}
	for i := 0; i < 5; i++ {
		sc, err := cp.Get()
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		if seenClients[sc] {
			matched++
		}
		grabbed = append(grabbed, sc)
	}

	if matched != 3 {
		t.Errorf("Expected to match 3 conns, matched %v", matched)
	}

	for _, c := range grabbed {
		cp.Return(c)
	}

	// Connect write error.
	sc, err := cp.Get()
	if err != nil {
		t.Fatalf("Error getting a connection: %v", err)
	}
	err = sc.Transmit(&gomemcached.MCRequest{})
	if err == nil {
		t.Fatalf("Expected error sending a request")
	}
	if sc.IsHealthy() {
		t.Fatalf("Expected unhealthy connection")
	}
	cp.Return(sc)

	if len(cp.connections) != 2 {
		t.Errorf("Expected to have 2 conns, have %v", len(cp.connections))
	}

	err = cp.Close()
	if err != nil {
		t.Errorf("Expected clean close, got %v", err)
	}

	err = cp.Close()
	if err == nil {
		t.Errorf("Expected error on second pool close")
	}
}

func TestConnPoolSoonAvailable(t *testing.T) {
	defer func(d time.Duration) { ConnPoolAvailWaitTime = d }(ConnPoolAvailWaitTime)
	defer func() { ConnPoolCallback = nil }()

	m := map[string]int{}
	timings := []time.Duration{}
	ConnPoolCallback = func(host string, source string, start time.Time, err error) {
		m[source] = m[source] + 1
		timings = append(timings, time.Since(start))
	}

	cp := newConnectionPool("h", &basicAuth{}, false, 3, 4, nil, "", false)
	cp.mkConn = testMkConn

	seenClients := map[*memcached.Client]bool{}

	// build some connections

	var aClient *memcached.Client
	for {
		sc, err := cp.GetWithTimeout(time.Millisecond)
		if err == ErrTimeout {
			break
		}
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		aClient = sc
		seenClients[sc] = true
	}

	time.AfterFunc(time.Millisecond, func() { cp.Return(aClient) })

	ConnPoolAvailWaitTime = time.Second

	sc, err := cp.Get()
	if err != nil || sc != aClient {
		t.Errorf("Expected a successful connection, got %v/%v", sc, err)
	}

	// Try again, but let's close it while we're stuck in secondary wait
	time.AfterFunc(time.Millisecond, func() { cp.Close() })

	sc, err = cp.Get()
	if err != errClosedPool {
		t.Errorf("Expected a closed pool, got %v/%v", sc, err)
	}

	t.Logf("Callback report: %v, timings: %v", m, timings)
}

func TestConnPoolClosedFull(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 4, nil, "", false)
	cp.mkConn = testMkConn

	seenClients := map[*memcached.Client]bool{}

	// build some connections

	for {
		sc, err := cp.GetWithTimeout(time.Millisecond)
		if err == ErrTimeout {
			break
		}
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		seenClients[sc] = true
	}

	time.AfterFunc(2*time.Millisecond, func() { cp.Close() })

	sc, err := cp.Get()
	if err != errClosedPool {
		t.Errorf("Expected closed pool error after closed, got %v/%v", sc, err)
	}
}

func TestConnPoolWaitFull(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 4, nil, "", false)
	cp.mkConn = testMkConn

	seenClients := map[*memcached.Client]bool{}

	// build some connections

	var aClient *memcached.Client
	for {
		sc, err := cp.GetWithTimeout(time.Millisecond)
		if err == ErrTimeout {
			break
		}
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		aClient = sc
		seenClients[sc] = true
	}

	time.AfterFunc(2*time.Millisecond, func() { cp.Return(aClient) })

	sc, err := cp.Get()
	if err != nil || sc != aClient {
		t.Errorf("Expected a successful connection, got %v/%v", sc, err)
	}
}

func TestConnPoolWaitFailFull(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 4, nil, "", false)
	cp.mkConn = testMkConn

	seenClients := map[*memcached.Client]bool{}

	// build some connections

	var aClient *memcached.Client
	for {
		sc, err := cp.GetWithTimeout(time.Millisecond)
		if err == ErrTimeout {
			break
		}
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		aClient = sc
		seenClients[sc] = true
	}

	// causes failure
	aClient.Transmit(&gomemcached.MCRequest{})
	time.AfterFunc(2*time.Millisecond, func() { cp.Return(aClient) })

	sc, err := cp.Get()
	if err != nil || sc == aClient {
		t.Errorf("Expected a new successful connection, got %v/%v", sc, err)
	}
}

func TestConnPoolWaitDoubleFailFull(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 4, nil, "", false)
	cp.mkConn = testMkConn

	seenClients := map[*memcached.Client]bool{}

	// build some connections

	var aClient *memcached.Client
	for {
		sc, err := cp.GetWithTimeout(time.Millisecond)
		if err == ErrTimeout {
			break
		}
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		aClient = sc
		seenClients[sc] = true
	}

	cp.mkConn = func(h string, ah AuthHandler, tlsConfig *tls.Config, bucketName string) (*memcached.Client, string, error) {
		return nil, "", io.EOF
	}

	// causes failure
	aClient.Transmit(&gomemcached.MCRequest{})
	time.AfterFunc(2*time.Millisecond, func() { cp.Return(aClient) })

	sc, err := cp.Get()
	if err != io.EOF {
		t.Errorf("Expected to fail getting a new connection, got %v/%v", sc, err)
	}
}

func TestConnPoolNil(t *testing.T) {
	var cp *connectionPool
	c, err := cp.Get()
	if err == nil {
		t.Errorf("Expected an error getting from nil, got %v", c)
	}

	// This just shouldn't error.
	cp.Return(c)
}

func TestConnPoolClosed(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 6, nil, "", false)
	cp.mkConn = testMkConn
	c, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Close()

	// This should cause the connection to be closed
	cp.Return(c)
	if err = c.Close(); err != errAlreadyClosed {
		t.Errorf("Expected to close connection, wasn't closed (%v)", err)
	}

	sc, err := cp.Get()
	if err != errClosedPool {
		t.Errorf("Expected closed pool error after closed, got %v/%v", sc, err)
	}
}

func TestConnPoolCloseWrongPool(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 6, nil, "", false)
	cp.mkConn = testMkConn
	c, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Close()

	// Return to a different pool.  Should still be OK.
	cp = newConnectionPool("h", &basicAuth{}, false, 3, 6, nil, "", false)
	cp.mkConn = testMkConn
	c, err = cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Close()

	cp.Return(c)
	if err = c.Close(); err != errAlreadyClosed {
		t.Errorf("Expected to close connection, wasn't closed (%v)", err)
	}
}

func TestConnPoolCloseNil(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 6, nil, "", false)
	cp.mkConn = testMkConn
	c, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Close()

	cp = nil
	cp.Return(c)
	if err = c.Close(); err != errAlreadyClosed {
		t.Errorf("Expected to close connection, wasn't closed (%v)", err)
	}
}

func BenchmarkBestCaseCPGet(b *testing.B) {
	cp := newConnectionPool("h", &basicAuth{}, false, 3, 6, nil, "", false)
	cp.mkConn = testMkConn

	for i := 0; i < b.N; i++ {
		c, err := cp.Get()
		if err != nil {
			b.Fatalf("Error getting from pool: %v", err)
		}
		cp.Return(c)
	}
}

// Get and Return from many goroutines at once, as fetches do, on a pool with the asynchronous
// closer (not running, so as not to interfere).
func BenchmarkParallelCPGet(b *testing.B) {
	cp := newConnectionPool("h", &basicAuth{}, false, 16, 16, nil, "", false)
	cp.mkConn = testMkConn
	cp.bailOut = make(chan bool, 1)
	defer cp.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c, err := cp.Get()
			if err != nil {
				b.Errorf("Error getting from pool: %v", err)
				return
			}
			cp.Return(c)
		}
	})
}

// keepAliveLog records the last keepalive idle time applied to each connection, in place of the
// setsockopt calls that the test connections, not being sockets, can't take.
type keepAliveLog struct {
	sync.Mutex
	idle map[*memcached.Client]int
}

func recordKeepAlive(t *testing.T) *keepAliveLog {
	l := &keepAliveLog{idle: map[*memcached.Client]int{}}
	saved := setKeepAlive
	setKeepAlive = func(c *memcached.Client, idleTime int) {
		l.Lock()
		l.idle[c] = idleTime
		l.Unlock()
	}
	t.Cleanup(func() { setKeepAlive = saved })
	return l
}

func (l *keepAliveLog) get(c *memcached.Client) (int, bool) {
	l.Lock()
	defer l.Unlock()
	idle, ok := l.idle[c]
	return idle, ok
}

func isParked(cp *connectionPool, c *memcached.Client) (parkedConn, bool) {
	cp.parkLock.Lock()
	defer cp.parkLock.Unlock()
	p, ok := cp.parked[c]
	return p, ok
}

// Connections in the pool are tracked as parked, and forgotten when taken out. With the
// asynchronous closer, returning a connection leaves its keepalive alone, and only connections
// parked for the whole idle time are relaxed, in place; without it they are relaxed on return.
func TestConnPoolParkedKeepAlive(t *testing.T) {
	saved := TCPKeepalive
	TCPKeepalive = true
	defer func() { TCPKeepalive = saved }()
	ka := recordKeepAlive(t)

	for _, closer := range []bool{true, false} {
		cp := newConnectionPool("h", &basicAuth{}, closer, 3, 6, nil, "", false)
		cp.mkConn = testMkConn

		var conns []*memcached.Client
		for i := 0; i < 3; i++ {
			c, err := cp.Get()
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := isParked(cp, c); ok {
				t.Fatalf("closer %v: checked out connection still tracked as parked", closer)
			}
			conns = append(conns, c)
		}

		// two connections returned a while ago, one just now
		cp.Return(conns[0])
		cp.Return(conns[1])
		cp.parkLock.Lock()
		for _, c := range conns[:2] {
			p := cp.parked[c]
			p.since = p.since.Add(-time.Minute)
			cp.parked[c] = p
		}
		cp.parkLock.Unlock()
		cp.Return(conns[2])

		for _, c := range conns {
			p, ok := isParked(cp, c)
			if !ok {
				t.Fatalf("closer %v: returned connection not tracked as parked", closer)
			}
			if p.relaxed != !closer {
				t.Fatalf("closer %v: returned connection relaxed %v", closer, p.relaxed)
			}
			// testMkConn, unlike defaultMkConn, applies no keepalive to new connections
			if idle, _ := ka.get(c); (idle == TCPKeepalivePooledIdleTime) != !closer {
				t.Fatalf("closer %v: returned connection has keepalive idle %v", closer, idle)
			}
		}

		if closer {
			if n := cp.relaxIdleKeepAlive(30 * time.Second); n != 2 {
				t.Fatalf("expected 2 connections relaxed, got %v", n)
			}
			for i, c := range conns {
				p, _ := isParked(cp, c)
				if p.relaxed != (i < 2) {
					t.Fatalf("connection %v relaxed %v", i, p.relaxed)
				}
				if idle, _ := ka.get(c); (idle == TCPKeepalivePooledIdleTime) != (i < 2) {
					t.Fatalf("connection %v has keepalive idle %v", i, idle)
				}
			}
			if n := cp.relaxIdleKeepAlive(30 * time.Second); n != 0 {
				t.Fatalf("expected relaxed connections not to be relaxed again, got %v", n)
			}
			TCPKeepalive = false
			if n := cp.relaxIdleKeepAlive(0); n != 0 {
				t.Fatalf("expected nothing relaxed with keepalive disabled, got %v", n)
			}
			TCPKeepalive = true
		}

		if len(cp.connections) != 3 {
			t.Fatalf("closer %v: expected 3 parked connections, got %v", closer, len(cp.connections))
		}
		for i := 0; i < 3; i++ {
			c, err := cp.Get()
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := isParked(cp, c); ok {
				t.Fatalf("closer %v: checked out connection still tracked as parked", closer)
			}
			if idle, _ := ka.get(c); idle == TCPKeepalivePooledIdleTime {
				t.Fatalf("closer %v: checked out connection has keepalive idle %v", closer, idle)
			}
			cp.Return(c)
		}

		cp.Close()
		cp.parkLock.Lock()
		n := len(cp.parked)
		cp.parkLock.Unlock()
		if n != 0 {
			t.Fatalf("closer %v: %v connections still tracked as parked after Close", closer, n)
		}
	}
}

// Returning a connection to a closed pool, or to a full one, closes it and leaves nothing
// tracked as parked.
func TestConnPoolParkedClosedOrFull(t *testing.T) {
	cp := newConnectionPool("h", &basicAuth{}, false, 1, 0, nil, "", false)
	cp.mkConn = testMkConn
	c, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Close()
	cp.Return(c)
	if err := c.Close(); err != errAlreadyClosed {
		t.Fatalf("expected connection closed on return to a closed pool, got %v", err)
	}
	if _, ok := isParked(cp, c); ok {
		t.Fatal("connection returned to a closed pool tracked as parked")
	}

	// room for one connection in the pool, and for building one more
	cp = newConnectionPool("h", &basicAuth{}, false, 1, 1, nil, "", false)
	cp.mkConn = testMkConn
	c1, _ := cp.Get()
	c2, _, _ := cp.mkConn("h", nil, nil, "")
	cp.createsem <- true // account for c2 as if the pool had built it
	cp.Return(c1)
	cp.Return(c2)
	if err := c2.Close(); err != errAlreadyClosed {
		t.Fatalf("expected connection closed on return to a full pool, got %v", err)
	}
	if _, ok := isParked(cp, c2); ok {
		t.Fatal("connection returned to a full pool tracked as parked")
	}
	if _, ok := isParked(cp, c1); !ok {
		t.Fatal("connection returned to the pool not tracked as parked")
	}
	cp.Close()
}

// connCloser relaxes connections that stay parked.
func TestConnPoolCloserRelaxes(t *testing.T) {
	saved := TCPKeepalive
	TCPKeepalive = true
	defer func() { TCPKeepalive = saved }()

	// start connCloser by hand, with a short interval, once the test mkConn is in place
	const interval = 20 * time.Millisecond
	cp := newConnectionPool("h", &basicAuth{}, false, 2, 2, nil, "", false)
	cp.mkConn = testMkConn
	cp.bailOut = make(chan bool, 1)

	// keep connCloser's server restart probe, which would borrow the parked test connection and
	// discard it, from running for the duration of the test
	cp.lastServerCheck = time.Now()
	done := make(chan struct{})
	go func() {
		cp.connCloser(interval)
		close(done)
	}()

	// wait for connCloser to exit before the deferred restore of TCPKeepalive, which it reads
	defer func() {
		cp.Close()
		<-done
	}()

	c, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Return(c)

	deadline := time.Now().Add(5 * time.Second)
	for {
		if p, ok := isParked(cp, c); ok && p.relaxed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("connCloser did not relax a parked connection")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// statT answers every request with an empty STAT response, which ends a stats exchange, so that
// the server restart probe can borrow and return a connection.
type statT struct {
	testT
	resp bytes.Reader
}

var statEnd = func() []byte {
	b := make([]byte, gomemcached.HDR_LEN)
	b[0] = gomemcached.RES_MAGIC
	b[1] = byte(gomemcached.STAT)
	return b
}()

func (t *statT) Write(b []byte) (int, error) {
	t.resp.Reset(statEnd)
	return len(b), nil
}

func (t *statT) Read(b []byte) (int, error) {
	return t.resp.Read(b)
}

// The server restart probe borrows a parked connection and returns it: if it had been relaxed,
// it must be tightened again, or it would be parked as not relaxed and later handed out with the
// pooled keepalive idle time.
func TestConnPoolProbeUptimeTightens(t *testing.T) {
	saved := TCPKeepalive
	TCPKeepalive = true
	defer func() { TCPKeepalive = saved }()
	ka := recordKeepAlive(t)

	// a pool with an asynchronous closer, without running it
	cp := newConnectionPool("h", &basicAuth{}, false, 2, 2, nil, "", false)
	cp.mkConn = func(h string, ah AuthHandler, tlsConfig *tls.Config, bucketName string) (*memcached.Client, string, error) {
		c, err := memcached.Wrap(&statT{})
		return c, "", err
	}
	cp.bailOut = make(chan bool, 1)
	defer cp.Close()

	c, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	cp.Return(c)
	if n := cp.relaxIdleKeepAlive(0); n != 1 {
		t.Fatalf("expected 1 connection relaxed, got %v", n)
	}
	if idle, _ := ka.get(c); idle != TCPKeepalivePooledIdleTime {
		t.Fatalf("relaxed connection has keepalive idle %v", idle)
	}

	cp.probeUptime(time.Now())
	if p, ok := isParked(cp, c); !ok || p.relaxed {
		t.Fatalf("probed connection parked %v relaxed %v, expected parked and not relaxed", ok, p.relaxed)
	}
	if idle, _ := ka.get(c); idle != TCPKeepaliveActiveIdleTime {
		t.Fatalf("probed connection parked as not relaxed with keepalive idle %v", idle)
	}

	c2, err := cp.Get()
	if err != nil {
		t.Fatal(err)
	}
	if c2 != c {
		t.Fatalf("expected the probed connection back")
	}
	if idle, _ := ka.get(c2); idle != TCPKeepaliveActiveIdleTime {
		t.Fatalf("connection handed out with keepalive idle %v", idle)
	}
}
