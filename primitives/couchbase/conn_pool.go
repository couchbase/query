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
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
	"github.com/couchbase/query/logging"
)

const _LOG_INTERVAL = 10

// GenericMcdAuthHandler is a kind of AuthHandler that performs
// special auth exchange (like non-standard auth, possibly followed by
// select-bucket).
type GenericMcdAuthHandler interface {
	AuthHandler
	AuthenticateMemcachedConn(host string, conn *memcached.Client) error
}

// Error raised when a connection can't be retrieved from a pool.
var TimeoutError = errors.New("timeout waiting to build connection")
var errClosedPool = errors.New("the connection pool is closed")
var errNoPool = errors.New("no connection pool")
var errHostChanged = errors.New("host address changed since pool was created")

// Default timeout for retrieving a connection from the pool.
var ConnPoolTimeout = time.Hour * 24 * 30

// overflow connection closer cycle time
var ConnCloserInterval = time.Second * 30

// interval after which check for restart and drain idle connections
const _SERVER_START_CHECK_INTERVAL = 5 * time.Minute // originally 5 minutes

// deadline for the uptime stat probe, so a hung node cannot stall connCloser.
var serverStartProbeTimeout = 10 * time.Second // originally 10 seconds

// to absorb small clock jitter
var serverStartTolerance = 5 * time.Second // originally 5 seconds

// ConnPoolAvailWaitTime is the amount of time to wait for an existing
// connection from the pool before considering the creation of a new
// one.
var ConnPoolAvailWaitTime = time.Millisecond

type connectionPool struct {
	host        string
	server      string // the node as the bucket's server list names it (host is the TLS port when TLS is on)
	initialAddr string
	mkConn      func(host string, ah AuthHandler, tlsConfig *tls.Config, bucketName string) (*memcached.Client, string, error)
	auth        AuthHandler
	connections chan *memcached.Client
	createsem   chan bool
	bailOut     chan bool
	poolSize    int
	connCount   uint64
	connOpen    uint64
	connClosed  uint64
	inUse       bool
	encrypted   bool
	// the client's tls config when the pool was built, whether or not the pool is encrypted (the local node isn't,
	// unless non-SSL ports are disabled), so that a new security config (a new tls config) can be detected; only an
	// encrypted pool dials with it (see dialTLSConfig)
	tlsConfig *tls.Config
	bucket    string

	// serverStart is the computed start time of the memcached process on this
	// node, derived from its "uptime" stat; lastServerCheck is when we last
	// probed it. Both are touched only from connCloser, so no locking needed.
	serverStart     time.Time
	lastServerCheck time.Time

	// connections parked in the pool, with when each was parked and whether its keepalive has
	// been relaxed since (see park, unpark and relaxIdleKeepAlive)
	parkLock sync.Mutex
	parked   map[*memcached.Client]parkedConn
}

type parkedConn struct {
	since   time.Time
	relaxed bool
}

func newConnectionPool(host string, ah AuthHandler, closer bool, poolSize, poolOverflow int, tlsConfig *tls.Config, bucket string,
	encrypted bool) *connectionPool {

	connSize := poolSize
	if closer {
		connSize += poolOverflow
	}
	rv := &connectionPool{
		host:        host,
		connections: make(chan *memcached.Client, connSize),
		createsem:   make(chan bool, poolSize+poolOverflow),
		mkConn:      defaultMkConn,
		auth:        ah,
		poolSize:    poolSize,
		bucket:      bucket,
		encrypted:   encrypted,
		parked:      make(map[*memcached.Client]parkedConn, connSize),
	}

	rv.tlsConfig = tlsConfig

	if closer {
		rv.bailOut = make(chan bool, 1)
		go rv.connCloser(ConnCloserInterval)
	}
	return rv
}

// ConnPoolTimeout is notified whenever connections are acquired from a pool.
var ConnPoolCallback func(host string, source string, start time.Time, err error)

// keepAliveConfig builds a full keepalive configuration for the given idle time. Setting the
// probe interval and count explicitly, rather than relying on SetKeepAliveOptions (which sets
// TCP_KEEPIDLE alone), is what bounds detection at idleTime + probeInterval*probeCount instead
// of leaving a ~10 minute tail at the OS default probe schedule.
func keepAliveConfig(idleTime int) net.KeepAliveConfig {
	return net.KeepAliveConfig{
		Enable:   true,
		Idle:     time.Duration(idleTime) * time.Second,
		Interval: time.Duration(TCPKeepaliveProbeInterval) * time.Second,
		Count:    TCPKeepaliveProbeCount,
	}
}

// applyActiveKeepAlive tightens TCP keepalive on a connection that is about to be handed out
// for active use, so a half-dead socket is detected in seconds rather than waiting on the
// caller's own (potentially long, statement-timeout-derived) read deadline.
func applyActiveKeepAlive(c *memcached.Client) {
	if TCPKeepalive {
		setKeepAlive(c, TCPKeepaliveActiveIdleTime)
	}
}

// setKeepAlive applies the keepalive configuration for the given idle time to c; a variable so
// that tests can see what is applied to connections that are not real sockets
var setKeepAlive = func(c *memcached.Client, idleTime int) {
	c.SetKeepAliveConfig(keepAliveConfig(idleTime))
}

// applyPooledKeepAlive relaxes the idle time on a connection parked in the pool, since nothing is
// blocked on it until it is next checked out. With the asynchronous closer this is only done once
// a connection has been parked for a whole ConnCloserInterval, so within about two intervals of
// it being returned (see relaxIdleKeepAlive), so that a connection that is checked out again soon,
// the common case under load, does not pay setsockopt calls on every Return and Get. The probe interval and count stay as they are: a parked
// connection that does fail its probes simply dies quietly and the next checkout takes the
// existing discard-and-retry path.
func applyPooledKeepAlive(c *memcached.Client) {
	if TCPKeepalive {
		setKeepAlive(c, TCPKeepalivePooledIdleTime)
	}
}

// park records c as parked in the pool; it must be called before c is put in cp.connections.
// Without the asynchronous closer nothing relaxes parked connections later, so their keepalive is
// relaxed straight away.
func (cp *connectionPool) park(c *memcached.Client) {
	relaxed := false
	if cp.bailOut == nil && TCPKeepalive {
		applyPooledKeepAlive(c)
		relaxed = true
	}
	cp.parkLock.Lock()
	cp.parked[c] = parkedConn{since: time.Now(), relaxed: relaxed}
	cp.parkLock.Unlock()
}

// unpark forgets c, just taken out of cp.connections (or never put in), and reports whether its
// keepalive may have been relaxed while parked. A connection taken out to be used, and possibly
// returned, must go through checkout instead, or a relaxed connection could be parked again as
// not relaxed and later handed out without being tightened.
func (cp *connectionPool) unpark(c *memcached.Client) bool {
	cp.parkLock.Lock()
	p, ok := cp.parked[c]
	delete(cp.parked, c)
	cp.parkLock.Unlock()
	return !ok || p.relaxed
}

// checkout readies c, just taken out of cp.connections, for active use: its keepalive is
// tightened again only if it was relaxed while parked.
func (cp *connectionPool) checkout(c *memcached.Client) {
	if cp.unpark(c) {
		applyActiveKeepAlive(c)
	}
}

// relaxIdleKeepAlive relaxes the keepalive of the connections that have been parked for at least
// idle. They stay in the pool meanwhile. parkLock is held across each connection's setsockopt
// calls, one connection at a time: whoever takes that connection out waits on it in unpark, so the
// calls are over before the connection is used, and then sees it relaxed; other Gets and Returns
// on the pool wait for at most one connection's calls.
func (cp *connectionPool) relaxIdleKeepAlive(idle time.Duration) int {
	if !TCPKeepalive {
		return 0
	}
	now := time.Now()
	var idleConns []*memcached.Client
	cp.parkLock.Lock()
	for c, p := range cp.parked {
		if !p.relaxed && now.Sub(p.since) >= idle {
			idleConns = append(idleConns, c)
		}
	}
	cp.parkLock.Unlock()

	relaxed := 0
	for _, c := range idleConns {
		cp.parkLock.Lock()

		// skip connections taken out, or taken out and parked again, since the scan
		if p, ok := cp.parked[c]; ok && !p.relaxed && now.Sub(p.since) >= idle {
			applyPooledKeepAlive(c)
			p.relaxed = true
			cp.parked[c] = p
			relaxed++
		}
		cp.parkLock.Unlock()
	}
	return relaxed
}

// Use regular in-the-clear connection if tlsConfig is nil.
// Use secure connection (TLS) if tlsConfig is set.
func defaultMkConn(host string, ah AuthHandler, tlsConfig *tls.Config, bucketName string) (*memcached.Client, string, error) {
	var features memcached.Features

	var conn *memcached.Client
	var err error
	if tlsConfig == nil {
		conn, err = memcached.Connect("tcp", host)
	} else {
		conn, err = memcached.ConnectTLS("tcp", host, tlsConfig)
	}

	if err != nil {
		return nil, "", err
	}

	var ip string
	if c, ok := conn.Conn().(net.Conn); ok {
		if a := c.RemoteAddr(); a != nil {
			ip = a.String()
		}
	}

	if DefaultTimeout > 0 {
		dl, _ := getDeadline(noDeadline, _NO_TIMEOUT, DefaultTimeout)
		conn.SetDeadline(dl)
	}

	// freshly created connections go straight into active use
	applyActiveKeepAlive(conn)

	if EnableMutationToken == true {
		features = append(features, memcached.FeatureMutationToken)
	}
	if EnableDataType == true {
		features = append(features, memcached.FeatureDataType)
	}

	if EnableSnappyCompression == true {
		features = append(features, memcached.FeatureSnappyCompression)
	}

	if EnableXattr == true {
		features = append(features, memcached.FeatureXattr)
	}

	if EnableSyncReplication {
		features = append(features, memcached.FeatureSyncReplication)
	}

	if EnableCollections {
		features = append(features, memcached.FeatureCollections)
	}

	if EnablePreserveExpiry {
		features = append(features, memcached.FeaturePreserveExpiry)
	}

	if EnableXerror {
		features = append(features, memcached.FeatureXerror)
	}

	if EnableComputeUnits {
		features = append(features, memcached.FeatureComputeUnits)
	}

	if EnableHandleThrottle {
		features = append(features, memcached.FeatureHandleThrottle)
	}

	if EnableTracing {
		features = append(features, memcached.FeatureTracing)
	}

	if len(features) > 0 {
		res, err := conn.EnableFeatures(features)
		if err != nil && isTimeoutError(err) {
			conn.Close()
			return nil, "", err
		}

		if err != nil || res.Status != gomemcached.SUCCESS {
			logging.Warnf("Unable to enable features %v", err)
		}
	}

	if gah, ok := ah.(GenericMcdAuthHandler); ok {
		err = gah.AuthenticateMemcachedConn(host, conn)
		if err != nil {
			conn.Close()
			return nil, "", err
		}

		if DefaultTimeout > 0 {
			conn.SetDeadline(noDeadline)
		}

		return conn, ip, nil
	}
	name, pass, bucket := ah.GetCredentials()
	if bucket == "" {
		// Authenticator does not know specific bucket.
		bucket = bucketName
	}

	if name != "default" {
		_, err = conn.Auth(name, pass)
		if err != nil {
			conn.Close()
			return nil, "", err
		}
		// Select bucket (Required for cb_auth creds)
		// Required when doing auth with _admin credentials
		if bucket != "" && bucket != name {
			_, err = conn.SelectBucket(bucket)
			if err != nil {
				conn.Close()
				return nil, "", err
			}
		}
	}

	if DefaultTimeout > 0 {
		conn.SetDeadline(noDeadline)
	}

	return conn, ip, nil
}

func (cp *connectionPool) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("connectionPool.Close error")
		}
	}()
	if cp.bailOut != nil {

		// defensively, we won't wait if the channel is full
		select {
		case cp.bailOut <- false:
		default:
		}
	}
	close(cp.connections)
	for c := range cp.connections {
		c.Close()
	}
	cp.parkLock.Lock()
	clear(cp.parked)
	cp.parkLock.Unlock()
	return
}

// the tls config to dial with: none for an unencrypted pool
func (cp *connectionPool) dialTLSConfig() *tls.Config {
	if !cp.encrypted {
		return nil
	}
	return cp.tlsConfig
}

func (cp *connectionPool) Node() string {
	return cp.host
}

func (cp *connectionPool) GetWithTimeout(d time.Duration) (rv *memcached.Client, err error) {
	if cp == nil {
		return nil, errNoPool
	}

	path := ""

	if ConnPoolCallback != nil {
		defer func(path *string, start time.Time) {
			ConnPoolCallback(cp.host, *path, start, err)
		}(&path, time.Now())
	}

	path = "short-circuit"

	// short-circuit available connetions.
	select {
	case rv, isopen := <-cp.connections:
		if !isopen {
			return nil, errClosedPool
		}
		atomic.AddUint64(&cp.connCount, 1)
		cp.checkout(rv)
		return rv, nil
	default:
	}

	t := time.NewTimer(ConnPoolAvailWaitTime)
	defer t.Stop()

	// Try to grab an available connection within 1ms
	select {
	case rv, isopen := <-cp.connections:
		path = "avail1"
		if !isopen {
			return nil, errClosedPool
		}
		atomic.AddUint64(&cp.connCount, 1)
		cp.checkout(rv)
		return rv, nil
	case <-t.C:
		// No connection came around in time, let's see
		// whether we can get one or build a new one first.
		t.Reset(d) // Reuse the timer for the full timeout.
		select {
		case rv, isopen := <-cp.connections:
			path = "avail2"
			if !isopen {
				return nil, errClosedPool
			}
			atomic.AddUint64(&cp.connCount, 1)
			cp.checkout(rv)
			return rv, nil
		case cp.createsem <- true:
			path = "create"
			// Build a connection if we can't get a real one.
			// This can potentially be an overflow connection, or
			// a pooled connection.
			rv, ip, err := cp.mkConn(cp.host, cp.auth, cp.dialTLSConfig(), cp.bucket)
			if err != nil {
				// On error, release our create hold
				<-cp.createsem
			} else {
				// Record IP on first connection then validate all others match
				if cp.initialAddr == "" {
					cp.initialAddr = ip
				} else if cp.initialAddr != ip {
					rv.Close()
					<-cp.createsem
					return nil, errHostChanged
				}
				atomic.AddUint64(&cp.connCount, 1)
				atomic.AddUint64(&cp.connOpen, 1)
			}
			return rv, err
		case <-t.C:
			return nil, ErrTimeout
		}
	}
}

func (cp *connectionPool) Get() (*memcached.Client, error) {
	return cp.GetWithTimeout(ConnPoolTimeout)
}

func (cp *connectionPool) Return(c *memcached.Client) {
	if c == nil {
		return
	}

	if cp == nil {
		c.Close()
		return
	}

	if c.IsHealthy() {
		defer func() {
			if recover() != nil {
				// This happens when the pool has already been
				// closed and we're trying to return a
				// connection to it anyway.  Just close the
				// connection.
				if cp != nil {
					cp.unpark(c)
					atomic.AddUint64(&cp.connClosed, 1)
				}
				c.Close()
			}
		}()

		// keepalive is left as it is: see applyPooledKeepAlive
		cp.park(c)

		select {
		case cp.connections <- c:
		default:
			cp.unpark(c)
			<-cp.createsem
			atomic.AddUint64(&cp.connClosed, 1)
			c.Close()
		}
	} else {
		<-cp.createsem
		atomic.AddUint64(&cp.connClosed, 1)
		c.Close()
	}
}

// give the ability to discard a connection from a pool
// useful for ditching connections to the wrong node after a rebalance
func (cp *connectionPool) Discard(c *memcached.Client) {
	<-cp.createsem
	atomic.AddUint64(&cp.connClosed, 1)
	c.Close()
}

// asynchronous connection closer
func (cp *connectionPool) connCloser(interval time.Duration) {
	var connCount uint64
	var logCount = _LOG_INTERVAL

	t := time.NewTimer(interval)
	defer t.Stop()

	for {
		connCount = atomic.LoadUint64(&cp.connCount)

		// we don't exist anymore! bail out!
		select {
		case <-cp.bailOut:
			return
		case <-t.C:
		}
		t.Reset(interval)
		logCount--
		if logCount == 0 {
			opened, closed := atomic.LoadUint64(&cp.connOpen), atomic.LoadUint64(&cp.connClosed)
			logging.Infof("bucket %s node %s connections opened %v closed %v open %v (re)used %v",
				cp.bucket, cp.host, opened, closed, opened-closed, atomic.LoadUint64(&cp.connCount))
			logCount = _LOG_INTERVAL
		}

		// drain idle connection if server has restarted
		cp.refreshServerStart()

		// close overflow connections if there are any and they are not needed, that is
		// unless there were sustained requests for connections
		if len(cp.connections) > cp.poolSize &&
			interval/ConnPoolAvailWaitTime >= time.Duration(atomic.LoadUint64(&cp.connCount)-connCount) {
		overflow:
			for len(cp.connections) > cp.poolSize {
				select {
				case <-cp.bailOut:
					return
				default:
				}

				// don't wait if the connections have been taken meanwhile: keepalive
				// relaxing and restart detection depend on this loop terminating
				var c *memcached.Client
				select {
				case conn, isOpen := <-cp.connections:
					if !isOpen {
						return
					}
					c = conn
				default:
					break overflow
				}
				cp.unpark(c)

				// bail out if close did not work out
				if !cp.connCleanup(c) {
					return
				}
			}
		}

		// relax keepalive on what has been parked since the last cycle
		cp.relaxIdleKeepAlive(interval)
	}
}

// close connection with recovery on error
func (cp *connectionPool) connCleanup(c *memcached.Client) (rv bool) {

	// just in case we are closing a connection after
	// bailOut has been sent but we haven't yet read it
	defer func() {
		if recover() != nil {
			rv = false
		}
	}()
	rv = true

	c.Close()
	<-cp.createsem
	return
}

// refresh serverStart every server start check interval
// if new serverStart time detected -> node has restarted -> drain idle connections
func (cp *connectionPool) refreshServerStart() {
	now := time.Now()
	if !cp.lastServerCheck.IsZero() && now.Sub(cp.lastServerCheck) < _SERVER_START_CHECK_INTERVAL {
		return
	}
	cp.lastServerCheck = now

	uptimeSecs, ok := cp.probeUptime(now)
	if !ok {
		// could not read uptime this cycle
		return
	}

	newServerStart := now.Add(-time.Duration(uptimeSecs) * time.Second)

	if cp.serverStart.IsZero() {
		// first observation
		cp.serverStart = newServerStart
		return
	}

	if newServerStart.After(cp.serverStart.Add(serverStartTolerance)) {
		prevServerStart := cp.serverStart
		cp.serverStart = newServerStart
		numDrained := cp.drainIdleConns()
		logging.Infof("bucket %s node %s memcached restart detected (start %v -> %v), "+
			"proactively drained %d stale idle connection(s)",
			cp.bucket, cp.host, prevServerStart, newServerStart, numDrained)
	}
}

func (cp *connectionPool) probeUptime(now time.Time) (int64, bool) {
	// try a idle connection from pool
	select {
	case c, isPoolOpen := <-cp.connections:
		if !isPoolOpen {
			return 0, false
		}

		// it may be returned to the pool afterwards, so check it out properly
		cp.checkout(c)

		c.SetDeadline(now.Add(serverStartProbeTimeout)) // prevent hung node from stalling proveUptime
		stats, err := c.StatsMap("")
		if err == nil {
			c.SetDeadline(noDeadline)
			secs, ok := parseUptime(cp, stats)
			cp.Return(c) // return connection to pool for reuse
			return secs, ok
		}

		// stale idle connection
		cp.Discard(c)
	default:
		// no idle connection to borrow
	}

	// we can still get uptime by creating connection to the node even when whole pool is stale
	c, _, err := cp.mkConn(cp.host, cp.auth, cp.dialTLSConfig(), cp.bucket)
	if err != nil || c == nil {
		// node still unreachable
		return 0, false
	}
	c.SetDeadline(now.Add(serverStartProbeTimeout)) // prevent hung node from stalling probeUptime
	stats, err := c.StatsMap("")
	c.Close() // this connection is not part of pool therefore close it

	if err != nil {
		return 0, false
	}
	return parseUptime(cp, stats)
}

func parseUptime(cp *connectionPool, stats map[string]string) (int64, bool) {
	uptimeStr, ok := stats["uptime"]
	if !ok {
		return 0, false
	}
	secs, err := strconv.ParseInt(uptimeStr, 10, 64)
	if err != nil {
		logging.Debugf("node %s unparseable uptime stat %q: %v", cp.host, uptimeStr, err)
		return 0, false
	}
	return secs, true
}

func (cp *connectionPool) drainIdleConns() int {
	drained := 0
	for {
		select {
		case c, isPoolOpen := <-cp.connections:
			if !isPoolOpen {
				return drained
			}
			cp.unpark(c)
			if !cp.connCleanup(c) { // clean idle connection
				return drained
			}
			drained++
		default:
			return drained
		}
	}
}
