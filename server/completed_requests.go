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
 The log itself is written in such a way to be of little burden to the operation of the engine.
 As an example - scanning the log is done acquiring and releasing the relevant mutex for each
 entry in the log.
 This will not provide an exact snapshot at a given moment in time, but more like a 99% accurate
 view - the advantage being that the service can continue to operate uninterrupted, rather than
 halt waiting for the scan to be completed.
*/
package server

import (
	"net/http"
	"sync"
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

type qualifier interface {
	name() string
	unique() bool
	condition() interface{}
	isCondition(c interface{}) bool
	evaluate(request *BaseRequest) bool
}

type RequestLog struct {
	sync.RWMutex
	qualifiers []qualifier

	cache *util.GenCache
}

var requestLog = &RequestLog{}

// init completed requests

func RequestsInit(threshold int, limit int) {
	requestLog.Lock()
	defer requestLog.Unlock()

	// initial completed_request setup is that it only tracks
	// requests exceeding a time threshold
	q, err := newTimeThreshold(threshold)
	if err == nil {
		requestLog.qualifiers = []qualifier{q}
	}

	requestLog.cache = util.NewGenCache(limit)
}

// configure completed requests

func RequestsLimit() int {
	return requestLog.cache.Limit()
}

func RequestsSetLimit(limit int) {
	requestLog.cache.SetLimit(limit)
}

func RequestsAddQualifier(name string, condition interface{}) errors.Error {
	var q qualifier
	var err errors.Error

	requestLog.Lock()
	defer requestLog.Unlock()
	for _, q := range requestLog.qualifiers {
		if q.name() == name && q.unique() {
			return errors.NewCompletedQualifierExists(name)
		}
	}
	switch name {
	case "threshold":
		q, err = newTimeThreshold(condition)
	default:
		return errors.NewCompletedQualifierUnknown(name)
	}
	if q != nil {
		requestLog.qualifiers = append(requestLog.qualifiers, q)
	}
	return err
}

func RequestsUpdateQualifier(name string, condition interface{}) errors.Error {
	var q qualifier
	var err errors.Error

	requestLog.Lock()
	defer requestLog.Unlock()
	for i, q := range requestLog.qualifiers {
		if q.name() == name {
			if !q.unique() {
				return errors.NewCompletedQualifierNotUnique(name)
			}
			requestLog.qualifiers = append(requestLog.qualifiers[:i], requestLog.qualifiers[i+1:]...)
		}
	}
	switch name {
	case "threshold":
		q, err = newTimeThreshold(condition)
	default:
		return errors.NewCompletedQualifierUnknown(name)
	}
	if q != nil {
		requestLog.qualifiers = append(requestLog.qualifiers, q)
	}
	return err
}

func RequestsRemoveQualifier(name string, condition interface{}) errors.Error {
	requestLog.Lock()
	defer requestLog.Unlock()
	for i, q := range requestLog.qualifiers {
		if q.name() == name && (q.unique() || q.isCondition(condition)) {
			requestLog.qualifiers = append(requestLog.qualifiers[:i], requestLog.qualifiers[i+1:]...)
			return nil
		}
	}
	return errors.NewCompletedQualifierNotFound(name, condition)
}

func RequestsGetQualifier(name string) (interface{}, errors.Error) {
	requestLog.RLock()
	defer requestLog.RUnlock()
	for _, q := range requestLog.qualifiers {
		if q.name() == name {
			if q.unique() {
				return q.condition(), nil
			}
			return nil, errors.NewCompletedQualifierNotUnique(name)
		}
	}
	return nil, errors.NewCompletedQualifierNotFound(name, nil)
}

func RequestsGetQualifiers() (qualifiers []struct {
	name      string
	condition interface{}
}) {
	requestLog.RLock()
	defer requestLog.RUnlock()
	for _, q := range requestLog.qualifiers {
		theQual := struct {
			name      string
			condition interface{}
		}{q.name(), q.condition()}
		qualifiers = append(qualifiers, theQual)
	}
	return
}

// completed requests operations

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
	result_count int, result_size int, error_count int, req *http.Request,
	request *BaseRequest, server *Server) {

	// negative limit means no upper bound (handled in cache)
	// zero limit means log nothing (handled here to avoid time wasting in cache)
	if requestLog.cache.Limit() == 0 {
		return
	}
	requestLog.RLock()
	defer requestLog.RUnlock()

	// apply all the qualifiers until one is satisfied
	doLog := false
	for _, q := range requestLog.qualifiers {
		doLog = q.evaluate(request)
		if doLog {
			break
		}
	}

	// request does not qualify
	if !doLog {
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

	re.Users = datastore.CredsString(request.Credentials(), req)
	re.RemoteAddr = request.RemoteAddr()
	userAgent := request.UserAgent()
	if userAgent != "" {
		re.UserAgent = userAgent
	}

	requestLog.cache.Add(re, id, nil)
}

// request qualifiers

// 1- threshold
type timeThreshold struct {
	threshold time.Duration
}

func newTimeThreshold(c interface{}) (*timeThreshold, errors.Error) {
	switch c.(type) {
	case int:
		return &timeThreshold{threshold: time.Duration(c.(int))}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("threshold", c)
}

func (this *timeThreshold) name() string {
	return "threshold"
}

func (this *timeThreshold) unique() bool {
	return true
}

func (this *timeThreshold) condition() interface{} {
	return this.threshold
}

func (this *timeThreshold) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return time.Duration(c.(int)) == this.threshold
	}
	return false
}

func (this *timeThreshold) evaluate(request *BaseRequest) bool {

	// negative threshold means log nothing
	// zero threshold means log everything (no threshold)
	if this.threshold < 0 ||
		(this.threshold >= 0 &&
			time.Since(request.ServiceTime()) < time.Millisecond*this.threshold) {
		return false
	}
	return true
}
