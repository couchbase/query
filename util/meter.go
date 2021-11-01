//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const _INTERVAL time.Duration = time.Second

type Meter interface {
	Count() int64
	Mark(int64, time.Time)
	Rate() float64
}

// NewMeter constructs a new StandardMeter and launches a goroutine.
func NewMeter(w, i time.Duration) Meter {
	return &meter{
		curTime: time.Now(),
		intvl:   float64(i) * float64(time.Second) / float64(time.Second),
		alpha:   1 - math.Exp(-5.0/float64(time.Minute/time.Second)/(float64(w)/float64(time.Minute))),
	}
}

type meter struct {
	count    int64
	curCount int64
	alpha    float64
	intvl    float64
	rate     float64
	curTime  time.Time
	sync.RWMutex
}

// Count returns the number of events recorded.
func (m *meter) Count() int64 {
	return m.count
}

// Mark records the occurance of n events.
func (m *meter) Mark(n int64, t time.Time) {
	atomic.AddInt64(&m.count, n)
	cur := atomic.AddInt64(&m.curCount, n)
	m.Lock()
	intvl := t.Sub(m.curTime)
	if intvl > _INTERVAL {
		atomic.AddInt64(&m.curCount, -cur)
		lastRate := float64(cur) / float64(intvl)
		m.rate += m.alpha * (lastRate - m.rate)
		m.curTime = t
	}
	m.Unlock()
}

// Rate returns the interval rate of events
func (m meter) Rate() float64 {
	m.RLock()
	count := m.curCount
	t := m.curTime
	rate := m.rate
	m.RUnlock()
	intvl := float64(time.Since(t))
	if intvl == 0.0 {
		return rate * m.intvl
	}
	return (rate + float64(count)/intvl) * m.intvl / (intvl + float64(_INTERVAL)) * float64(_INTERVAL)
}
