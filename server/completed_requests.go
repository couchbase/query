//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/couchbase/go_json"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type RequestsOp int

const (
	CMP_OP_ADD RequestsOp = iota
	CMP_OP_DEL
	CMP_OP_UPD
)

const _DEF_MAX_PLAN_SIZE = 256 * logging.KiB
const _MAX_PLAN_SIZE_LIMIT = 20*logging.MiB - 128*logging.KiB

type RequestLogEntry struct {
	RequestId                string
	ClientId                 string
	ElapsedTime              time.Duration
	ServiceTime              time.Duration
	TransactionElapsedTime   time.Duration
	TransactionRemainingTime time.Duration
	QueryContext             string
	Statement                string
	StatementType            string
	Plan                     *plan.Prepared
	State                    string
	ScanConsistency          string
	TxId                     string
	UseFts                   bool
	UseCBO                   bool
	UseReplica               bool
	FeatureControls          uint64
	ResultCount              int
	ResultSize               int
	ErrorCount               int
	Errors                   []map[string]interface{}
	Mutations                uint64
	PreparedName             string
	PreparedText             string
	Time                     time.Time
	PhaseTimes               map[string]interface{}
	PhaseCounts              map[string]interface{}
	PhaseOperators           map[string]interface{}
	timings                  []byte
	optEstimates             value.Value
	NamedArgs                map[string]value.Value
	PositionalArgs           value.Values
	MemoryQuota              uint64
	UsedMemory               uint64
	Users                    string
	RemoteAddr               string
	UserAgent                string
	Tag                      string
}

type qualifier interface {
	name() string
	unique() bool
	condition() interface{}
	isCondition(c interface{}) bool
	checkCondition(c interface{}) errors.Error
	evaluate(request *BaseRequest, req *http.Request) bool
}

type handler struct {
	handlerFunc func(e *RequestLogEntry)
	refCount    int
}

type RequestLog struct {
	sync.RWMutex
	extra            int
	pushed           int
	qualifiers       []qualifier
	taggedQualifiers map[string][]qualifier
	cache            *util.GenCache
	handlers         map[string]*handler
	maximumPlanSize  int
}

var requestLog = &RequestLog{}

// init completed requests

func RequestsInit(threshold int, limit int) {
	requestLog.Lock()
	defer requestLog.Unlock()

	// initial completed_request setup is that it only tracks
	// requests exceeding a time threshold
	tq, tErr := newTimeThreshold(threshold)
	if tErr == nil {
		requestLog.qualifiers = append(requestLog.qualifiers, tq)
	}
	aq, aErr := newAborted(nil)
	if aErr == nil {
		requestLog.qualifiers = append(requestLog.qualifiers, aq)
	}
	requestLog.taggedQualifiers = make(map[string][]qualifier)

	requestLog.cache = util.NewGenCache(limit)
	requestLog.handlers = make(map[string]*handler)
	requestLog.maximumPlanSize = _DEF_MAX_PLAN_SIZE
}

// configure completed requests

func RequestsMaxPlanSize() int {
	return requestLog.maximumPlanSize
}

func RequestsSetMaxPlanSize(max int) {
	if max < 0 || max > _MAX_PLAN_SIZE_LIMIT {
		max = _MAX_PLAN_SIZE_LIMIT
	}
	requestLog.Lock()
	requestLog.maximumPlanSize = max
	requestLog.Unlock()
}

func RequestsLimit() int {
	return requestLog.cache.Limit()
}

func RequestsSetLimit(limit int, op RequestsOp) {
	requestLog.Lock()
	switch op {
	case CMP_OP_ADD:
		oldLimit := requestLog.cache.Limit()

		// no temporary extra entries if already unlimited
		if oldLimit < 0 {
			requestLog.Unlock()
			return
		}
		requestLog.extra += limit
		requestLog.pushed++
		limit += oldLimit
	case CMP_OP_UPD:

		// remove temporary requests if going unlimited
		if limit < 0 {
			requestLog.extra = 0
			requestLog.pushed = 0
		} else if requestLog.pushed > 0 {
			limit += requestLog.extra
		}
	case CMP_OP_DEL:

		// don't remove temporary entries if there aren't any
		if requestLog.pushed == 0 {
			requestLog.Unlock()
			return
		}
		requestLog.pushed--
		if requestLog.pushed == 0 {
			limit = requestLog.cache.Limit() - requestLog.extra
			requestLog.extra = 0
		} else {
			requestLog.Unlock()
			return
		}
	}
	requestLog.cache.SetLimit(limit)
	requestLog.Unlock()
}

func RequestsAddHandler(f func(e *RequestLogEntry), name string) {
	requestLog.Lock()
	found := requestLog.handlers[name]
	if found != nil {
		found.refCount++
	} else {
		requestLog.handlers[name] = &handler{handlerFunc: f, refCount: 1}
	}
	requestLog.Unlock()
}

func RequestsRemoveHandler(name string) bool {
	requestLog.Lock()
	found := requestLog.handlers[name]
	if found != nil {
		if found.refCount == 1 {
			delete(requestLog.handlers, name)
		} else {
			found.refCount--
		}
	}
	requestLog.Unlock()
	return found != nil
}

func (this *RequestLog) getQualList(tag string) []qualifier {
	if tag == "" {
		return this.qualifiers
	} else {
		return this.taggedQualifiers[tag]
	}
}

func (this *RequestLog) setQualList(quals []qualifier, tag string) {
	if tag == "" {
		this.qualifiers = quals
	} else {
		this.taggedQualifiers[tag] = quals
	}
}

func RequestsCheckQualifier(name string, op RequestsOp, condition interface{}, tag string) errors.Error {
	var err errors.Error

	requestLog.Lock()
	defer requestLog.Unlock()
	quals := requestLog.getQualList(tag)
	for _, q := range quals {
		if q.name() == name {
			switch op {
			case CMP_OP_ADD:
				if q.unique() || tag != "" || q.isCondition(condition) {
					return errors.NewCompletedQualifierExists(name)
				}
			case CMP_OP_UPD:
				if q.unique() {
					return q.checkCondition(condition)
				} else {
					return errors.NewCompletedQualifierNotUnique(name)
				}
			case CMP_OP_DEL:
				if q.isCondition(condition) {
					return nil
				}
			}
		}
	}
	if op != CMP_OP_ADD {
		return errors.NewCompletedQualifierNotFound(name, condition)
	}
	switch name {
	case "threshold":
		_, err = newTimeThreshold(condition)
	case "aborted":
		_, err = newAborted(condition)
	case "error":
		_, err = newReqError(condition)
	case "user":
		_, err = newUser(condition)
	case "client":
		_, err = newClient(condition)
	case "context":
		_, err = newContext(condition)
	case "results":
		_, err = newResults(condition)
	case "size":
		_, err = newSize(condition)
	case "mutations":
		_, err = newMutations(condition)
	case "counts":
		_, err = newCounts(condition)
	default:
		return errors.NewCompletedQualifierUnknown(name)
	}
	return err
}

func RequestsAddQualifier(name string, condition interface{}, tag string) errors.Error {
	var q qualifier
	var err errors.Error

	requestLog.Lock()
	defer requestLog.Unlock()
	quals := requestLog.getQualList(tag)

	// create tag if missing
	if quals == nil {
		requestLog.taggedQualifiers[tag] = make([]qualifier, 0)
		quals = requestLog.taggedQualifiers[tag]
	}

	for _, q := range quals {
		if q.name() == name && (q.unique() || tag != "" || q.isCondition(condition)) {
			return errors.NewCompletedQualifierExists(name)
		}
	}
	switch name {
	case "threshold":
		q, err = newTimeThreshold(condition)
	case "aborted":
		q, err = newAborted(condition)
	case "error":
		q, err = newReqError(condition)
	case "user":
		q, err = newUser(condition)
	case "client":
		q, err = newClient(condition)
	case "context":
		q, err = newContext(condition)
	case "results":
		q, err = newResults(condition)
	case "size":
		q, err = newSize(condition)
	case "mutations":
		q, err = newMutations(condition)
	case "counts":
		q, err = newCounts(condition)
	default:
		return errors.NewCompletedQualifierUnknown(name)
	}
	if err == nil && q != nil {
		requestLog.setQualList(append(quals, q), tag)
	}
	return err
}

func RequestsUpdateQualifier(name string, condition interface{}, tag string) errors.Error {
	var nq qualifier
	var err errors.Error

	iq := -1
	requestLog.Lock()
	defer requestLog.Unlock()
	quals := requestLog.getQualList(tag)
	if quals == nil {
		return errors.NewCompletedQualifierNotFound(name, "")
	}

	for i, q := range quals {
		if q.name() == name {
			if !q.unique() {
				return errors.NewCompletedQualifierNotUnique(name)
			}
			iq = i
			break
		}
	}
	if iq < 0 {
		return errors.NewCompletedQualifierNotFound(name, "")
	}
	switch name {
	case "threshold":
		nq, err = newTimeThreshold(condition)
	default:
		return errors.NewCompletedQualifierUnknown(name)
	}
	if err == nil && nq != nil {
		quals[iq] = nq
		requestLog.setQualList(quals, tag)
	}
	return err
}

func RequestsRemoveQualifier(name string, condition interface{}, tag string) errors.Error {
	requestLog.Lock()
	defer requestLog.Unlock()

	quals := requestLog.getQualList(tag)
	if quals == nil {
		return errors.NewCompletedQualifierNotFound(name, "")
	}

	count := 0
	for i, q := range quals {
		if q.name() == name {
			if condition == nil {
				quals = append(quals[:i], quals[i+1:]...)
				count++
			} else if q.unique() || q.isCondition(condition) {
				quals = append(quals[:i], quals[i+1:]...)
				count++
				break
			}
		}
	}
	if count == 0 {
		return errors.NewCompletedQualifierNotFound(name, condition)
	}

	// delete tag if empty
	if tag != "" && len(quals) == 0 {
		delete(requestLog.taggedQualifiers, tag)
	} else {
		requestLog.setQualList(quals, tag)
	}
	return nil
}

func RequestsGetQualifier(name string, tag string) (interface{}, errors.Error) {
	requestLog.RLock()
	defer requestLog.RUnlock()

	quals := requestLog.getQualList(tag)
	if quals == nil {
		return nil, errors.NewCompletedQualifierNotFound(name, nil)
	}
	for _, q := range quals {
		if q.name() == name {
			if q.unique() {
				return q.condition(), nil
			}
			return nil, errors.NewCompletedQualifierNotUnique(name)
		}
	}
	return nil, errors.NewCompletedQualifierNotFound(name, nil)
}

func RequestsGetQualifiers() interface{} {
	requestLog.RLock()
	defer requestLog.RUnlock()
	if len(requestLog.taggedQualifiers) == 0 {
		return getQualifiers(requestLog.qualifiers)
	}

	rv := make([]interface{}, len(requestLog.taggedQualifiers)+1)
	i := 0
	for tag, quals := range requestLog.taggedQualifiers {
		obj := getQualifiers(quals)
		obj["tag"] = tag
		rv[i] = obj
		i++
	}
	rv[i] = getQualifiers(requestLog.qualifiers)
	return rv
}

func getQualifiers(quals []qualifier) map[string]interface{} {
	qualifiers := make(map[string]interface{})
	for _, q := range quals {
		qEntry := qualifiers[q.name()]
		if qEntry == nil {
			qualifiers[q.name()] = q.condition()
		} else {
			switch qEntry.(type) {
			case []interface{}:
				qualifiers[q.name()] = append(qualifiers[q.name()].([]interface{}), q.condition())
			default:
				slice := []interface{}{qEntry, q.condition()}
				qualifiers[q.name()] = slice
			}
		}
	}
	return qualifiers
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
		re.timings = nil
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

func RequestsForeach(nonBlocking func(string, *RequestLogEntry) bool, blocking func() bool) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*RequestLogEntry))
	}
	requestLog.cache.ForEach(dummyF, blocking)
}

func LogRequest(request_time, service_time, transactionElapsedTime time.Duration,
	result_count int, result_size int, error_count int, req *http.Request,
	request *BaseRequest, server *Server) {

	// negative limit means no upper bound (handled in cache)
	// zero limit means log nothing (handled here to avoid time wasting in cache)
	if requestLog.cache.Limit() == 0 {
		return
	}

	// these assignments are a bit hacky, but simplify our life
	request.resultCount = int64(result_count)
	request.resultSize = int64(result_size)
	request.serviceDuration = service_time
	requestLog.RLock()
	defer requestLog.RUnlock()

	// first try all tags
	// all the qualifiers in a tag set must apply
	doLog := false
	tag := ""
	for n, _ := range requestLog.taggedQualifiers {
		good := true
		for _, q := range requestLog.taggedQualifiers[n] {
			yes := q.evaluate(request, req)
			if !yes {
				good = false
				break
			}
		}
		if good {
			doLog = true
			tag = n
			break
		}
	}

	// finally do the untagged
	// apply all the qualifiers until one is satisfied
	if !doLog {
		for _, q := range requestLog.qualifiers {
			doLog = q.evaluate(request, req)
			if doLog {
				break
			}
		}
	}

	// request does not qualify
	if !doLog {
		return
	}

	id := request.Id().String()
	re := &RequestLogEntry{
		RequestId:       id,
		State:           request.State().StateName(),
		ElapsedTime:     request_time,
		ServiceTime:     service_time,
		ResultCount:     result_count,
		ResultSize:      result_size,
		ErrorCount:      error_count,
		Time:            request.RequestTime(),
		ScanConsistency: string(request.ScanConsistency()),
		UseFts:          request.UseFts(),
		UseCBO:          request.UseCBO(),
		UseReplica:      request.UseReplica(),
		FeatureControls: request.FeatureControls(),
		Mutations:       request.MutationCount(),
		QueryContext:    request.QueryContext(),
		TxId:            request.TxId(),
	}
	errs := request.Errors()
	re.Errors = make([]map[string]interface{}, 0, len(errs))
	for _, e := range errs {
		re.Errors = append(re.Errors, e.Object())
	}
	if !request.TransactionStartTime().IsZero() {
		re.TransactionElapsedTime = transactionElapsedTime
		if request.Type() != "COMMIT" && request.Type() != "ROLLBACK" {
			remTime := request.TxTimeout() - time.Since(request.TransactionStartTime())
			if remTime > 0 {
				re.TransactionRemainingTime = remTime
			}
		}
	}
	stmt := request.Statement()
	if stmt != "" {
		re.Statement = stmt
	}
	stmtType := request.Type()
	if stmtType != "" {
		re.StatementType = stmtType
	}
	plan := request.Prepared()
	if plan != nil {
		re.PreparedName = plan.Name()
		re.PreparedText = plan.Text()
	}
	re.PhaseCounts = request.FmtPhaseCounts()
	re.PhaseOperators = request.FmtPhaseOperators()
	re.PhaseTimes = request.FmtPhaseTimes()
	re.UsedMemory = request.UsedMemory()

	// in order not to bloat service memory, we marshal the timings into a value
	// at the expense of request execution time
	timings := request.GetTimings()
	maxPlanSize := RequestsMaxPlanSize()
	if timings != nil {
		parsed := request.GetFmtTimings()
		if len(parsed) > 0 {
			re.timings = parsed
		} else if maxPlanSize == 0 {
			re.timings = []byte("{\"WARNING\":\"Plan inclusion disabled.\"}")
		} else {
			v, err := json.Marshal(timings)
			if len(v) > 0 && err == nil && len(v) <= maxPlanSize {
				re.timings = v
			} else if len(v) > maxPlanSize {
				re.timings = []byte(fmt.Sprintf("{\"WARNING\":\"Plan (%v) exceeds maximum permitted (%v) size.\"}",
					logging.HumanReadableSize(int64(len(v)), false), logging.HumanReadableSize(int64(maxPlanSize), false)))
			}
		}
		estimates := request.GetFmtOptimizerEstimates()
		if len(parsed) > 0 {
			re.optEstimates = value.NewValue(estimates)
		} else {
			re.optEstimates = value.NewValue(request.FmtOptimizerEstimates(timings))
		}
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
		memoryQuota := request.MemoryQuota()
		if memoryQuota != 0 {
			re.MemoryQuota = memoryQuota
		}
	}

	re.Users = datastore.CredsString(request.Credentials())
	re.RemoteAddr = request.RemoteAddr()
	userAgent := request.UserAgent()
	if userAgent != "" {
		re.UserAgent = userAgent
	}

	clientId := request.ClientID().String()
	if clientId != "" {
		re.ClientId = clientId
	}
	if tag != "" {
		re.Tag = tag
	}

	requestLog.cache.Add(re, id, nil)
	for _, h := range requestLog.handlers {
		go h.handlerFunc(re)
	}
}

func (this *RequestLogEntry) Timings() interface{} {
	return this.timings
}

func (this *RequestLogEntry) OptEstimates() interface{} {
	return this.optEstimates
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
	case int64:
		return &timeThreshold{threshold: time.Duration(c.(int64))}, nil
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
	case int64:
		return time.Duration(c.(int64)) == this.threshold
	}
	return false
}

func (this *timeThreshold) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case int:
		return nil
	case int64:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument("threshold", c)
}

func (this *timeThreshold) evaluate(request *BaseRequest, req *http.Request) bool {

	// negative threshold means log nothing
	// zero threshold means log everything (no threshold)
	if this.threshold < 0 ||
		(this.threshold >= 0 &&
			request.serviceDuration < time.Millisecond*this.threshold) {
		return false
	}
	return true
}

// 2- aborted
type aborted struct {
	// run along, nothing to see here
}

func newAborted(c interface{}) (*aborted, errors.Error) {
	return &aborted{}, nil
}

func (this *aborted) name() string {
	return "aborted"
}

func (this *aborted) unique() bool {
	return true
}

func (this *aborted) condition() interface{} {
	return nil
}

func (this *aborted) isCondition(c interface{}) bool {
	return true
}

func (this *aborted) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *aborted) evaluate(request *BaseRequest, req *http.Request) bool {
	return request.State() == ABORTED
}

// 3- errors
type reqError struct {
	errCode int
}

func newReqError(c interface{}) (*reqError, errors.Error) {
	switch c.(type) {
	case int:
		return &reqError{errCode: c.(int)}, nil
	case int64:
		return &reqError{errCode: int(c.(int64))}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("error", c)
}

func (this *reqError) name() string {
	return "error"
}

func (this *reqError) unique() bool {
	return false
}

func (this *reqError) condition() interface{} {
	return this.errCode
}

func (this *reqError) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return c.(int) == this.errCode
	case int64:
		return int(c.(int64)) == this.errCode
	}
	return false
}

func (this *reqError) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *reqError) evaluate(request *BaseRequest, req *http.Request) bool {
	for _, e := range request.Errors() {
		if int(e.Code()) == this.errCode {
			return true
		}
	}
	return false
}

// 4- users
type user struct {
	id string
}

func newUser(c interface{}) (*user, errors.Error) {
	switch c.(type) {
	case string:
		return &user{id: c.(string)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("error", c)
}

func (this *user) name() string {
	return "user"
}

func (this *user) unique() bool {
	return false
}

func (this *user) condition() interface{} {
	return this.id
}

func (this *user) isCondition(c interface{}) bool {
	switch c.(type) {
	case string:
		return c.(string) == this.id
	}
	return false
}

func (this *user) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *user) evaluate(request *BaseRequest, req *http.Request) bool {
	var iid, icred int

	credString := datastore.CredsString(request.Credentials())

	// split in space separated tokens
loop:
	for icred = 0; icred < len(credString); icred++ {
		if credString[icred] == ',' {
			continue loop
		}

		// compare each token
		for iid = 0; iid < len(this.id); iid++ {
			if this.id[iid] != credString[icred] {

				// don't match, skip token
				for ; icred < len(credString) && credString[icred] != ','; icred++ {
				}
				continue loop
			}
			icred++
		}
		return true
	}
	return false
}

// 5- client ip addresses
type client struct {
	address string
}

func newClient(c interface{}) (*client, errors.Error) {
	switch c.(type) {
	case string:
		return &client{address: c.(string)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("client", c)
}

func (this *client) name() string {
	return "client"
}

func (this *client) unique() bool {
	return false
}

func (this *client) condition() interface{} {
	return this.address
}

func (this *client) isCondition(c interface{}) bool {
	switch c.(type) {
	case string:
		return c.(string) == this.address
	}
	return false
}

func (this *client) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *client) evaluate(request *BaseRequest, req *http.Request) bool {

	// assuming that address is a valid IPv4 or IPv6 address, this is a
	// quick and dirty way to ignore the port part of the RemoteAddress()
	return this.address == request.RemoteAddr()[0:len(this.address)]
}

// 6- client contex ID
type context struct {
	id string
}

func newContext(c interface{}) (*context, errors.Error) {
	switch c.(type) {
	case string:
		return &context{id: c.(string)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("context", c)
}

func (this *context) name() string {
	return "context"
}

func (this *context) unique() bool {
	return false
}

func (this *context) condition() interface{} {
	return this.id
}

func (this *context) isCondition(c interface{}) bool {
	switch c.(type) {
	case string:
		return c.(string) == this.id
	}
	return false
}

func (this *context) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *context) evaluate(request *BaseRequest, req *http.Request) bool {
	return this.id == request.ClientContextId()
}

// 7- results count
type results struct {
	count int64
}

func newResults(c interface{}) (*results, errors.Error) {
	switch c.(type) {
	case int:
		return &results{count: int64(c.(int))}, nil
	case int64:
		return &results{count: c.(int64)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("results", c)
}

func (this *results) name() string {
	return "results"
}

func (this *results) unique() bool {
	return true
}

func (this *results) condition() interface{} {
	return this.count
}

func (this *results) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return int64(c.(int)) == this.count
	case int64:
		return c.(int64) == this.count
	}
	return false
}

func (this *results) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case int:
		return nil
	case int64:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument("results", c)
}

func (this *results) evaluate(request *BaseRequest, req *http.Request) bool {
	return request.resultCount > this.count
}

// 8- mutation count
type mutations struct {
	count int64
}

func newMutations(c interface{}) (*mutations, errors.Error) {
	switch c.(type) {
	case int:
		return &mutations{count: int64(c.(int))}, nil
	case int64:
		return &mutations{count: c.(int64)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("mutations", c)
}

func (this *mutations) name() string {
	return "mutations"
}

func (this *mutations) unique() bool {
	return true
}

func (this *mutations) condition() interface{} {
	return this.count
}

func (this *mutations) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return int64(c.(int)) == this.count
	case int64:
		return c.(int64) == this.count
	}
	return false
}

func (this *mutations) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case int:
		return nil
	case int64:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument("mutations", c)
}

func (this *mutations) evaluate(request *BaseRequest, req *http.Request) bool {
	return int64(request.mutationCount) > this.count
}

// 9- output size
type size struct {
	size int64
}

func newSize(c interface{}) (*size, errors.Error) {
	switch c.(type) {
	case int:
		return &size{size: int64(c.(int))}, nil
	case int64:
		return &size{size: c.(int64)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("size", c)
}

func (this *size) name() string {
	return "size"
}

func (this *size) unique() bool {
	return true
}

func (this *size) condition() interface{} {
	return this.size
}

func (this *size) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return int64(c.(int)) == this.size
	case int64:
		return c.(int64) == this.size
	}
	return false
}

func (this *size) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case int:
		return nil
	case int64:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument("size", c)
}

func (this *size) evaluate(request *BaseRequest, req *http.Request) bool {
	return request.resultSize > this.size
}

// 10- operator counts
type counts struct {
	count int64
}

func newCounts(c interface{}) (*counts, errors.Error) {
	switch c.(type) {
	case int:
		return &counts{count: int64(c.(int))}, nil
	case int64:
		return &counts{count: c.(int64)}, nil
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("counts", c)
}

func (this *counts) name() string {
	return "counts"
}

func (this *counts) unique() bool {
	return true
}

func (this *counts) condition() interface{} {
	return this.count
}

func (this *counts) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return int64(c.(int)) == this.count
	case int64:
		return c.(int64) == this.count
	}
	return false
}

func (this *counts) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case int:
		return nil
	case int64:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument("counts", c)
}

func (this *counts) evaluate(request *BaseRequest, req *http.Request) bool {
	return int64(request.phaseStats[execution.FETCH].count) > this.count ||
		int64(request.phaseStats[execution.PRIMARY_SCAN].count) > this.count ||
		int64(request.phaseStats[execution.INDEX_SCAN].count) > this.count ||
		int64(request.phaseStats[execution.NL_JOIN].count) > this.count ||
		int64(request.phaseStats[execution.HASH_JOIN].count) > this.count ||
		int64(request.phaseStats[execution.SORT].count) > this.count
}
