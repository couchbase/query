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

// ConnPoolAvailWaitTime is the amount of time to wait for an existing
// connection from the pool before considering the creation of a new
// one.
var ConnPoolAvailWaitTime = time.Millisecond

type connectionPool struct {
	host        string
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
	tlsConfig   *tls.Config
	bucket      string
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
	}

	if encrypted {
		rv.tlsConfig = tlsConfig
	}

	if closer {
		rv.bailOut = make(chan bool, 1)
		go rv.connCloser()
	}
	return rv
}

// ConnPoolTimeout is notified whenever connections are acquired from a pool.
var ConnPoolCallback func(host string, source string, start time.Time, err error)

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

	if TCPKeepalive == true {
		conn.SetKeepAliveOptions(time.Duration(TCPKeepaliveInterval) * time.Second)
	}

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
	return
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
			return rv, nil
		case cp.createsem <- true:
			path = "create"
			// Build a connection if we can't get a real one.
			// This can potentially be an overflow connection, or
			// a pooled connection.
			rv, ip, err := cp.mkConn(cp.host, cp.auth, cp.tlsConfig, cp.bucket)
			if err != nil {
				// On error, release our create hold
				<-cp.createsem
			} else {
				// Record IP on first connection then validate all others match
				if cp.initialAddr == "" {
					cp.initialAddr = ip
				} else if cp.initialAddr != ip {
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
					atomic.AddUint64(&cp.connClosed, 1)
				}
				c.Close()
			}
		}()

		select {
		case cp.connections <- c:
		default:
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
func (cp *connectionPool) connCloser() {
	var connCount uint64
	var logCount = _LOG_INTERVAL

	t := time.NewTimer(ConnCloserInterval)
	defer t.Stop()

	for {
		connCount = cp.connCount

		// we don't exist anymore! bail out!
		select {
		case <-cp.bailOut:
			return
		case <-t.C:
		}
		t.Reset(ConnCloserInterval)
		logCount--
		if logCount == 0 {
			logging.Infof("bucket %s node %s connections opened %v closed %v open %v (re)used %v",
				cp.bucket, cp.host, cp.connOpen, cp.connClosed, cp.connOpen-cp.connClosed, cp.connCount)
			logCount = _LOG_INTERVAL
		}

		// no overflow connections open or sustained requests for connections
		// nothing to do until the next cycle
		if len(cp.connections) <= cp.poolSize ||
			ConnCloserInterval/ConnPoolAvailWaitTime < time.Duration(cp.connCount-connCount) {
			continue
		}

		// close overflow connections now that they are not needed
		for c := range cp.connections {
			select {
			case <-cp.bailOut:
				return
			default:
			}

			// bail out if close did not work out
			if !cp.connCleanup(c) {
				return
			}
			if len(cp.connections) <= cp.poolSize {
				break
			}
		}
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
