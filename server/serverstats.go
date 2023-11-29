//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"runtime"
	"runtime/debug"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/logging"
)

// Cut-down basic monitor to trigger FFDC when runtime memory stats indicate high useage

const (
	_DEF_LIMIT        = 0.9
	_STATS_INTRVL     = time.Second * 30
	_SAMPLES_2_HOURS  = int((time.Hour * 2) / _STATS_INTRVL)
	_SAMPLES_MIN      = int((time.Minute * 5) / _STATS_INTRVL)
	_FFDC_MEM_THRESH  = 0.8
	_FFDC_MEM_RATE    = 0.2
	_LOG_INTRVL       = 4
	_FFDC_RATE_THRESH = 0.333
)

var memLimit uint64
var memLimitFunc func(uint64) uint64

func getDefaultMemLimit() uint64 {
	return uint64(float64(getTotalMemory()) * _DEF_LIMIT)
}

func SetMemLimitFunc(f func(uint64) uint64) {
	memLimitFunc = f
}

func SetMemLimit(l uint64) {
	if l == 0 {
		l = getDefaultMemLimit()
	}
	if nil == memLimitFunc {
		memLimit = l
	} else {
		memLimit = memLimitFunc(l)
	}
}

func (this *Server) StartMonitor() {
	logging.Infof("Starting server monitor.")
	go this.monitor()
}

func (this *Server) monitor() {
	defer func() {
		e := recover()
		logging.Debugf("Server monitor failed with: %v.  Restarting.", e)
		go this.monitor()
	}()

	var ms runtime.MemStats
	var prevCr int64
	ac, _ := this.AccountingStore().(interface{ CompletedRequests() int64 })
	averageMemoryUsage := newRunningAverage(_SAMPLES_2_HOURS)
	mstats := make(map[string]interface{}, 3)
	index := 0

	start := time.Now()
	for {
		duration := time.Since(start)
		if duration < _STATS_INTRVL {
			time.Sleep(_STATS_INTRVL - duration)
		}
		start = time.Now()

		threshold := uint64(float64(memLimit) * _FFDC_MEM_THRESH)
		runtime.ReadMemStats(&ms)
		trigger := true

		if memLimit > 0 && ms.HeapAlloc > threshold {
			logging.Warnf("Memory threshold exceeded: %v > %v", logging.HumanReadableSize(int64(ms.HeapAlloc), true),
				logging.HumanReadableSize(int64(threshold), true))
			ffdc.Capture(ffdc.MemoryThreshold)
			debug.FreeOSMemory()
			trigger = false
		} else {
			ffdc.Reset(ffdc.MemoryThreshold)
		}

		last := averageMemoryUsage.last()
		averageMemoryUsage.record(ms.HeapAlloc)
		delta := int64(ms.HeapAlloc) - int64(last)
		if delta > 0 && averageMemoryUsage.count() > _SAMPLES_MIN && last > uint64(float64(memLimit)*_FFDC_RATE_THRESH) &&
			delta > int64(float64(averageMemoryUsage.value())*_FFDC_MEM_RATE) {

			logging.Warnf("Memory growth rate threshold exceeded: %v > %v", logging.HumanReadableSize(delta, true),
				logging.HumanReadableSize(int64(float64(averageMemoryUsage.value())*_FFDC_MEM_RATE), true))
			if trigger {
				ffdc.Capture(ffdc.MemoryRate)
				trigger = false
			}
		} else {
			ffdc.Reset(ffdc.MemoryRate)
		}

		var cr int64
		if ac != nil {
			cr = ac.CompletedRequests()
			if cr == prevCr {
				ratio := this.QueuedRequests() / (this.Servicers() + this.PlusServicers())
				if ratio >= 3 {
					logging.Warnf("No processed requests with queue of %v", this.QueuedRequests())
					ffdc.Capture(ffdc.StalledQueue)
				} else {
					ffdc.Reset(ffdc.StalledQueue)
				}
			} else {
				ffdc.Reset(ffdc.StalledQueue)
			}
			prevCr = cr
		}

		if index >= _LOG_INTRVL {

			if v, err := this.AccountingStore().Vitals(); err == nil {
				if b, err := json.Marshal(v); err == nil {
					json.Unmarshal(b, &mstats)
				}
			}
			if memLimit > 0 {
				mstats["memory.limit"] = memLimit
				mstats["memory.limit.h"] = logging.HumanReadableSize(int64(memLimit), false)
			}
			mstats["memory.usage"] = ms.HeapAlloc
			mstats["memory.usage.h"] = logging.HumanReadableSize(int64(ms.HeapAlloc), false)
			mstats["request.queued.count"] = this.QueuedRequests()
			mstats["request.completed.count"] = cr
			ffdc.Stats("ffdc.", mstats, false)

			if b, e := json.Marshal(mstats); e == nil {
				logging.Infof("Query Engine Stats %v", string(b))
			}
			index = 0
		}

		if time.Now().UnixNano()-int64(ms.LastGC) > int64(_STATS_INTRVL) {
			logging.Debugf("Running GC")
			runtime.GC()
		}
		index++
	}
}

type runningAverage struct {
	total   uint64
	samples []uint64
	index   int
	lastVal uint64
}

func newRunningAverage(samples int) *runningAverage {
	rv := &runningAverage{samples: make([]uint64, samples)}
	return rv
}

func (this *runningAverage) value() uint64 {
	return this.total / uint64(this.count())
}

func (this *runningAverage) count() int {
	if this.index >= len(this.samples) {
		return len(this.samples)
	}
	return this.index
}

func (this *runningAverage) record(v uint64) {
	i := this.index % len(this.samples)
	this.total -= this.samples[i] // zero before wrapping
	this.samples[i] = v
	this.total += v
	this.index++
	this.lastVal = v
}

func (this *runningAverage) last() uint64 {
	return this.lastVal
}
