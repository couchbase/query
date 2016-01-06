//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*
 Package accounting provides a common API for workload and monitoring data - metrics, statistics, events.

 Request_log provides a way to track completed requests that satisfy certain conditions
 (Currently - they last more than a certain threshold).
 The log itself is written in such a way to be of little burden to the operation of the engine.
 As an example - scanning the log is done acquiring and releasing the relevant mutex for each
 entry in the log.
 This will not provide an exact snapshot at a given moment in time, but more like a 99% accurate
 view - the advantage being that the service can continue to operate uninterrupted, rather than
 halt waiting for the scan to be completed.
*/
package accounting

import (
	"sync"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

type RequestLogEntry struct {
	RequestId    string
	ElapsedTime  time.Duration
	ServiceTime  time.Duration
	Statement    string
	Plan         *plan.Prepared
	ResultCount  int
	ResultSize   int
	ErrorCount   int
	SortCount    uint64
	PreparedName string
	PreparedText string
	Time         time.Time
}

// A log map ties a RequestId to a cache position
// Needed for Request Log cleanup
type RequestLogMap struct {
	logEntry *RequestLogEntry
	logIdx   int
}

const _CACHE_SIZE = 1 << 10
const _CACHES = 4

type RequestLog struct {
	threshold time.Duration

	// one lock per cache bucket to aid concurrency
	locks [_CACHES]sync.RWMutex

	// we need slices to to go through the requests releasing
	// the lock after each entry, and maps to map ids to entries
	// for the get and names methods
	requestCaches [_CACHES][]*RequestLogEntry
	requestMaps   [_CACHES]map[string]RequestLogMap
}

var requestLog = &RequestLog{}

func init() {

	// Don't log requests < 500ms.
	// TODO: make configurable: disable, all, or choose threshold duration
	requestLog.threshold = 500

	// TODO: add further logging filters (users, buckets, etc)

	for b := 0; b < _CACHES; b++ {
		requestLog.requestCaches[b] = make([]*RequestLogEntry, _CACHE_SIZE)
		requestLog.requestMaps[b] = make(map[string]RequestLogMap, _CACHE_SIZE)
	}
}

func (this *RequestLog) add(entry *RequestLogEntry) {
	var requestLogMap RequestLogMap

	cacheNum := util.HashString(entry.RequestId, _CACHES)
	this.locks[cacheNum].Lock()
	defer this.locks[cacheNum].Unlock()
	mLen := len(this.requestMaps[cacheNum])
	cLen := len(this.requestCaches[cacheNum])

	// Resize the slice if required
	// The map gets handled automatically
	// TODO handle memory allocation failures
	if mLen >= cLen {
		caches := make([]*RequestLogEntry, cLen*2)
		copy(caches, this.requestCaches[cacheNum])
		this.requestCaches[cacheNum] = caches
	}
	requestLogMap.logEntry = entry
	requestLogMap.logIdx = mLen
	this.requestMaps[cacheNum][entry.RequestId] = requestLogMap
	this.requestCaches[cacheNum][mLen] = entry
}

// Remove entry from request log and compact
func (this *RequestLog) ditch(id string) errors.Error {
	cacheNum := util.HashString(id, _CACHES)
	this.locks[cacheNum].Lock()
	defer this.locks[cacheNum].Unlock()
	logMap, ok := this.requestMaps[cacheNum][id]
	if ok {
		delete(this.requestMaps[cacheNum], id)
		l := len(this.requestMaps[cacheNum])

		// Nature abhors a vacuum!
		// We copy the last map entry onto the currently empty entry.
		// This is a quick way to keep the two caches in sync without
		// reallocating memory and / or copying huge quantities of data,
		// but it does mean that for scans occurring during deletes,
		// later entries might be skipped.
		// Given the fact that deletes will likely purge huge numbers of
		// entries, copying subslices on a per deleted row basis has the
		// potential for a significant bottleneck.
		// Given the huge benefit in lock contention and memory usage, we
		// are willing to take that risk.
		if logMap.logIdx < l {
			this.requestCaches[cacheNum][logMap.logIdx] = this.requestCaches[cacheNum][l]
			newMap := this.requestMaps[cacheNum][this.requestCaches[cacheNum][l].RequestId]
			newMap.logIdx = logMap.logIdx
		}
		return nil
	} else {
		return errors.NewSystemStmtNotFoundError(nil, id)
	}
}

func (this *RequestLog) get(id string) *RequestLogEntry {
	cacheNum := util.HashString(id, _CACHES)
	this.locks[cacheNum].RLock()
	defer this.locks[cacheNum].RUnlock()
	return this.requestMaps[cacheNum][id].logEntry
}

func (this *RequestLog) size() int {
	sz := 0

	for b := 0; b < _CACHES; b++ {
		this.locks[b].RLock()
		sz += len(this.requestMaps[b])
		this.locks[b].RUnlock()
	}
	return sz
}

func (this *RequestLog) names() []string {
	i := 0

	// we have emergency extra space not to have to append
	// if we can avoid it
	sz := _CACHES + this.size()
	n := make([]string, sz)
	this.forEach(func(id string, entry *RequestLogEntry) {
		if i < sz {
			n[i] = entry.RequestId
		} else {
			n = append(n, entry.RequestId)
		}
		i++
	})
	return n
}

// As noted in the starting comments, this is not a consistent snapshot
// but rather a a low cost, almost accurate view
func (this *RequestLog) forEach(f func(string, *RequestLogEntry)) {
	for b := 0; b < _CACHES; b++ {
		this.locks[b].RLock()
		for e := 0; e < len(this.requestMaps[b]); e++ {
			f(this.requestCaches[b][e].RequestId, this.requestCaches[b][e])
			this.locks[b].RUnlock()
			this.locks[b].RLock()
		}
		this.locks[b].RUnlock()
	}
}

func RequestsEntry(id string) *RequestLogEntry {
	return requestLog.get(id)
}

func RequestDelete(id string) errors.Error {
	return requestLog.ditch(id)
}

func RequestIds() []string {
	return requestLog.names()
}

func RequestsCount() int {
	return requestLog.size()
}

func RequestsForeach(f func(string, *RequestLogEntry)) {
	requestLog.forEach(f)
}

func LogRequest(acctstore AccountingStore,
	request_time time.Duration, service_time time.Duration,
	result_count int, result_size int,
	error_count int, warn_count int, stmt string,
	sort_count uint64, plan *plan.Prepared, id string) {

	if requestLog.threshold >= 0 && request_time < time.Millisecond*requestLog.threshold {
		return
	}

	rv := &RequestLogEntry{
		RequestId:   id,
		ElapsedTime: request_time,
		ServiceTime: service_time,
		ResultCount: result_count,
		ResultSize:  result_size,
		ErrorCount:  error_count,
		SortCount:   sort_count,
		Time:        time.Now(),
	}
	if stmt != "" {
		rv.Statement = stmt
	}
	if plan != nil {
		rv.PreparedName = plan.Name()
		rv.PreparedText = plan.Text()
	}
	requestLog.add(rv)
}
