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
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

type RequestLogEntry struct {
	RequestId      string
	ClientId       string
	ElapsedTime    time.Duration
	ServiceTime    time.Duration
	Statement      string
	Plan           *plan.Prepared
	State          string
	ResultCount    int
	ResultSize     int
	ErrorCount     int
	PreparedName   string
	PreparedText   string
	Time           time.Time
	PhaseTimes     map[string]interface{}
	PhaseCounts    map[string]interface{}
	PhaseOperators map[string]interface{}
}

const _CACHE_SIZE = 1 << 10
const _CACHES = 4

type RequestLog struct {
	threshold time.Duration

	cache *util.GenCache
}

var requestLog = &RequestLog{}

func RequestsInit(threshold int, limit int) {

	requestLog.threshold = time.Duration(threshold)

	// TODO: add further logging filters (users, buckets, etc)

	requestLog.cache = util.NewGenCache(limit)
}

func RequestsLimit() int {
	return requestLog.cache.Limit()
}

func RequestsSetLimit(limit int) {
	requestLog.cache.SetLimit(limit)
}

func RequestsThreshold() int {
	return int(requestLog.threshold)
}

func RequestsSetThreshold(threshold int) {
	requestLog.threshold = time.Duration(threshold)
}

func RequestEntry(id string) *RequestLogEntry {
	return requestLog.cache.Get(id, nil).(*RequestLogEntry)
}

func RequestDo(id string, f func(*RequestLogEntry)) {
	_ = requestLog.cache.Get(id, func(r interface{}) {
		f(r.(*RequestLogEntry))
	})
}

func RequestDelete(id string) errors.Error {
	if requestLog.cache.Delete(id, nil) {
		return nil
	} else {
		return errors.NewSystemStmtNotFoundError(nil, id)
	}
}

func RequestsIds() []string {
	return requestLog.cache.Names()
}

func RequestsCount() int {
	return requestLog.cache.Size()
}

func RequestsForeach(f func(string, *RequestLogEntry)) {
	dummyF := func(id string, r interface{}) {
		f(id, r.(*RequestLogEntry))
	}
	requestLog.cache.ForEach(dummyF)
}

func LogRequest(request_time time.Duration, service_time time.Duration,
	result_count int, result_size int,
	error_count int, stmt string,
	plan *plan.Prepared,
	phaseTimes map[string]interface{},
	phaseCounts map[string]interface{},
	phaseOperators map[string]interface{},
	state string, id string, clientId string) {

	if requestLog.threshold >= 0 && request_time < time.Millisecond*requestLog.threshold {
		return
	}

	re := &RequestLogEntry{
		RequestId:   id,
		ClientId:    clientId,
		State:       state,
		ElapsedTime: request_time,
		ServiceTime: service_time,
		ResultCount: result_count,
		ResultSize:  result_size,
		ErrorCount:  error_count,
		Time:        time.Now(),
	}
	if stmt != "" {
		re.Statement = stmt
	}
	if plan != nil {
		re.PreparedName = plan.Name()
		re.PreparedText = plan.Text()
	}
	re.PhaseTimes = phaseTimes
	re.PhaseCounts = phaseCounts
	re.PhaseOperators = phaseOperators

	requestLog.cache.Add(re, id)
}
