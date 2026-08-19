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
	"errors"
	"testing"
	"time"
)

// The exact strings the runtime produces for the syscalls we care about. ENETUNREACH is the
// one that motivated MB-73071: before it was listed it matched none of the predicates in the
// processOpError chain (isOutOfBounds -> isConnError -> isAddrNotAvailable ->
// isSeveredConnectionError -> IsReadTimeOutError), so a KV op that hit it neither discarded
// the connection nor retried - it surfaced to the caller as an unclassified failure. Note that
// isNoRouteError covers only EHOSTUNREACH ("no route to host"), which is a different errno.
const (
	errENETUNREACH  = "read tcp 10.0.0.1:53214->10.0.0.2:11210: connect: network is unreachable"
	errEHOSTUNREACH = "dial tcp 10.0.0.2:11210: connect: no route to host"
	errETIMEDOUT    = "read tcp 10.0.0.1:53214->10.0.0.2:11210: connection timed out"
	errIOTimeout    = "read tcp 10.0.0.1:53214->10.0.0.2:11210: i/o timeout"
	errECONNRESET   = "read tcp 10.0.0.1:53214->10.0.0.2:11210: connection reset by peer"
	errECONNREFUSED = "dial tcp 10.0.0.2:11210: connect: connection refused"
)

func TestIsReadTimeOutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		exp  bool
	}{
		{"nil", nil, false},
		{"ENETUNREACH", errors.New(errENETUNREACH), true},
		{"ETIMEDOUT", errors.New(errETIMEDOUT), true},
		{"i/o timeout", errors.New(errIOTimeout), true},
		// matches on the "read tcp" prefix rather than the errno
		{"ECONNRESET on a read", errors.New(errECONNRESET), true},
		{"EHOSTUNREACH", errors.New(errEHOSTUNREACH), false},
		{"ECONNREFUSED", errors.New(errECONNREFUSED), false},
		{"unrelated", errors.New("Auth failure"), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsReadTimeOutError(test.err); got != test.exp {
				t.Errorf("IsReadTimeOutError(%v) = %v, expected %v", test.err, got, test.exp)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		exp  bool
	}{
		{"ENETUNREACH", errors.New(errENETUNREACH), true},
		{"EHOSTUNREACH", errors.New(errEHOSTUNREACH), true},
		{"ETIMEDOUT", errors.New(errETIMEDOUT), true},
		{"i/o timeout", errors.New(errIOTimeout), true},
		{"ECONNRESET", errors.New(errECONNRESET), false},
		{"ECONNREFUSED", errors.New(errECONNREFUSED), false},
		{"unrelated", errors.New("Auth failure"), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTimeoutError(test.err); got != test.exp {
				t.Errorf("isTimeoutError(%v) = %v, expected %v", test.err, got, test.exp)
			}
		})
	}
}

// ENETUNREACH must not be swallowed by a predicate earlier in the processOpError chain, or
// adding it to IsReadTimeOutError would have had no effect.
func TestENETUNREACHNotShadowed(t *testing.T) {
	err := errors.New(errENETUNREACH)

	if isConnError(err) {
		t.Error("isConnError matches ENETUNREACH; it precedes IsReadTimeOutError and would shadow it")
	}
	if isOutOfBoundsError(err) {
		t.Error("isOutOfBoundsError matches ENETUNREACH")
	}
	if isNoRouteError(err) {
		t.Error("isNoRouteError matches ENETUNREACH; it should cover EHOSTUNREACH only")
	}
}

func TestKeepAliveConfig(t *testing.T) {
	saveActive, saveInterval, saveCount := TCPKeepaliveActiveIdleTime, TCPKeepaliveProbeInterval,
		TCPKeepaliveProbeCount
	defer func() {
		TCPKeepaliveActiveIdleTime = saveActive
		TCPKeepaliveProbeInterval = saveInterval
		TCPKeepaliveProbeCount = saveCount
	}()

	TCPKeepaliveProbeInterval = 2
	TCPKeepaliveProbeCount = 3

	cfg := keepAliveConfig(5)
	if !cfg.Enable {
		t.Error("keepalive not enabled")
	}
	// the point of the config: detection is bounded at idleTime + probeInterval*probeCount, not
	// left to the OS default probe schedule
	if budget := cfg.Idle + cfg.Interval*time.Duration(cfg.Count); budget != 11*time.Second {
		t.Errorf("detection budget = %v, expected 11s", budget)
	}
}

func TestSetTcpKeepaliveActive(t *testing.T) {
	savePooled, saveActive, saveInterval, saveCount := TCPKeepalivePooledIdleTime,
		TCPKeepaliveActiveIdleTime, TCPKeepaliveProbeInterval, TCPKeepaliveProbeCount
	defer func() {
		TCPKeepalivePooledIdleTime = savePooled
		TCPKeepaliveActiveIdleTime = saveActive
		TCPKeepaliveProbeInterval = saveInterval
		TCPKeepaliveProbeCount = saveCount
	}()

	TCPKeepalivePooledIdleTime = 30 * 60
	TCPKeepaliveActiveIdleTime = 5
	TCPKeepaliveProbeInterval = 2
	TCPKeepaliveProbeCount = 3

	// non-positive values leave the current setting alone
	SetTcpKeepaliveActive(0, -1, 0)
	if TCPKeepaliveActiveIdleTime != 5 || TCPKeepaliveProbeInterval != 2 || TCPKeepaliveProbeCount != 3 {
		t.Errorf("non-positive values changed settings: %d %d %d", TCPKeepaliveActiveIdleTime,
			TCPKeepaliveProbeInterval, TCPKeepaliveProbeCount)
	}

	SetTcpKeepaliveActive(10, 5, 4)
	if TCPKeepaliveActiveIdleTime != 10 || TCPKeepaliveProbeInterval != 5 || TCPKeepaliveProbeCount != 4 {
		t.Errorf("settings not applied: %d %d %d", TCPKeepaliveActiveIdleTime,
			TCPKeepaliveProbeInterval, TCPKeepaliveProbeCount)
	}

	// an active idle time above the pooled one would invert the two tiers, making a parked
	// connection probe more aggressively than one in use
	SetTcpKeepaliveActive(TCPKeepalivePooledIdleTime+1, 0, 0)
	if TCPKeepaliveActiveIdleTime != 10 {
		t.Errorf("active idle time %d exceeds pooled idle time %d", TCPKeepaliveActiveIdleTime,
			TCPKeepalivePooledIdleTime)
	}

	// and the same guard from the other direction
	SetTcpKeepalive(true, TCPKeepaliveActiveIdleTime-1)
	if TCPKeepalivePooledIdleTime < TCPKeepaliveActiveIdleTime {
		t.Errorf("pooled idle time %d dropped below active idle time %d", TCPKeepalivePooledIdleTime,
			TCPKeepaliveActiveIdleTime)
	}
}
