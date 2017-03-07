//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*
 Completed_requests provides a way to track completed requests that satisfy certain conditions
 (Currently - they last more than a certain threshold).
 The log itself is written in such a way to be of little burden to the operation of the engine.
 As an example - scanning the log is done acquiring and releasing the relevant mutex for each
 entry in the log.
 This will not provide an exact snapshot at a given moment in time, but more like a 99% accurate
 view - the advantage being that the service can continue to operate uninterrupted, rather than
 halt waiting for the scan to be completed.
*/
package server

import (
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type RequestLogEntry struct {
	RequestId       string
	ClientId        string
	ElapsedTime     time.Duration
	ServiceTime     time.Duration
	Statement       string
	Plan            *plan.Prepared
	State           string
	ScanConsistency string
	ResultCount     int
	ResultSize      int
	ErrorCount      int
	PreparedName    string
	PreparedText    string
	Time            time.Time
	PhaseTimes      map[string]interface{}
	PhaseCounts     map[string]interface{}
	PhaseOperators  map[string]interface{}
	Timings         execution.Operator
	NamedArgs       map[string]value.Value
	PositionalArgs  value.Values
	Users           string
	RemoteAddr      string
	UserAgent       string
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
	if requestLog.cache.Delete(id, func(r interface{}) {
		re := r.(*RequestLogEntry)
		if re.Timings != nil {
			re.Timings.Done()
			re.Timings = nil
		}
	}) {
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
	result_count int, result_size int, error_count int,
	request *BaseRequest, server *Server) {

	// negative threshold means log nothing
	// zero threshold means log everything (no threshold)
	// zero limit means log nothing (handled here to avoid time wasting in cache)
	// negative limit means no upper bound (handled in cache)
	if requestLog.threshold < 0 || requestLog.cache.Limit() == 0 ||
		(requestLog.threshold >= 0 && request_time < time.Millisecond*requestLog.threshold) {
		return
	}

	id := request.Id().String()
	re := &RequestLogEntry{
		RequestId:       id,
		ClientId:        request.ClientID().String(),
		State:           string(request.State()),
		ElapsedTime:     request_time,
		ServiceTime:     service_time,
		ResultCount:     result_count,
		ResultSize:      result_size,
		ErrorCount:      error_count,
		Time:            time.Now(),
		ScanConsistency: string(request.ScanConsistency()),
	}
	stmt := request.Statement()
	if stmt != "" {
		re.Statement = stmt
	}
	plan := request.Prepared()
	if plan != nil {
		re.PreparedName = plan.Name()
		re.PreparedText = plan.Text()
	}
	re.PhaseCounts = request.FmtPhaseCounts()
	re.PhaseOperators = request.FmtPhaseOperators()

	// in order not to bloat service memory, we only
	// store timings if they are turned on at the service
	// or request level when the request completes.
	// this may yield inconsistent output if different nodes
	// have different settings, but it's better than ever growing
	// memory because we are storing every plan in completed_requests
	// once timings get stored in completed_requests, it's this
	// module that's responsible for cleaning after them, hence
	// we nillify request.timings to signal that
	prof := request.Profile()
	if prof == ProfUnset {
		prof = server.Profile()
	}
	if prof != ProfOff {
		re.PhaseTimes = request.FmtPhaseTimes()
	}
	if prof == ProfOn {
		re.Timings = request.GetTimings()
		request.SetTimings(nil)
	}

	var ctrl bool
	ctr := request.Controls()
	if ctr == value.NONE {
		ctrl = server.Controls()
	} else {
		ctrl = (ctr == value.TRUE)
	}
	if ctrl {
		re.NamedArgs = request.NamedArgs()
		re.PositionalArgs = request.PositionalArgs()
	}

	re.Users = datastore.CredsString(request.Credentials())
	re.RemoteAddr = request.RemoteAddr()
	userAgent := request.UserAgent()
	if userAgent != "" {
		re.UserAgent = userAgent
	}

	requestLog.cache.Add(re, id)
}
