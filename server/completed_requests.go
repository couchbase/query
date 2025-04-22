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
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type RequestsOp int

const (
	CMP_OP_ADD RequestsOp = iota
	CMP_OP_DEL
	CMP_OP_UPD
)

const _DEF_MAX_PLAN_SIZE = 256 * util.KiB
const _MAX_PLAN_SIZE_LIMIT = 20*util.MiB - 128*util.KiB

type RequestLogEntry struct {
	RequestId                string
	ClientId                 string
	ElapsedTime              time.Duration
	ServiceTime              time.Duration
	TransactionElapsedTime   time.Duration
	TransactionRemainingTime time.Duration
	ThrottleTime             time.Duration
	CpuTime                  time.Duration
	IoTime                   time.Duration
	WaitTime                 time.Duration
	Timeout                  time.Duration
	QueryContext             string
	Statement                string
	StatementType            string
	State                    string
	ScanConsistency          string
	TxId                     string
	UseFts                   bool
	UseCBO                   bool
	UseReplica               value.Tristate
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
	optEstimates             map[string]interface{}
	NamedArgs                map[string]value.Value
	PositionalArgs           value.Values
	MemoryQuota              uint64
	UsedMemory               uint64
	Users                    string
	RemoteAddr               string
	UserAgent                string
	Tag                      string
	Tenant                   string
	Qualifier                string
	SessionMemory            uint64
	Analysis                 []interface{}
	SqlID                    string
	NaturalLanguage          string
	NaturalOutput            string
	NaturalTime              time.Duration
	LogContent               []interface{}
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
	stream           requestLogStream
}

var requestLog = &RequestLog{}

var qualTypeMap = map[string]func(interface{}) (qualifier, errors.Error){
	"threshold":    newTimeThreshold,
	"aborted":      newAborted,
	"error":        newReqError,
	"errors":       newReqErrorCount,
	"user":         newUser,
	"client":       newClient,
	"context":      newContext,
	"results":      newResults,
	"size":         newSize,
	"mutations":    newMutations,
	"counts":       newCounts,
	"statement":    newStatement,
	"plan":         newPlanElement,
	"seqscan_keys": newSeqScanKeys,
	"used_memory":  newUsedMemory,
}

// init completed requests

func RequestsInit(threshold int, limit int, seqscan_keys int) {
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
	if seqscan_keys > 0 {
		sq, sErr := newSeqScanKeys(seqscan_keys)
		if sErr == nil {
			requestLog.qualifiers = append(requestLog.qualifiers, sq)
		}
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
	constr, ok := qualTypeMap[name]
	if !ok {
		return errors.NewCompletedQualifierUnknown(name)
	}
	_, err = constr(condition)
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
	constr, ok := qualTypeMap[name]
	if !ok {
		return errors.NewCompletedQualifierUnknown(name)
	}
	q, err = constr(condition)
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
	constr, ok := qualTypeMap[name]
	if !ok {
		return errors.NewCompletedQualifierUnknown(name)
	}
	nq, err = constr(condition)
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

func RequestDelete(id string, f func(*RequestLogEntry) bool) errors.Error {
	if requestLog.cache.DeleteWithCheck(id, func(r interface{}) bool {
		re := r.(*RequestLogEntry)
		if f != nil && !f(re) {
			return false
		}
		return true
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
	request *BaseRequest, server *Server, seq_scan_keys int64, forceCapture bool) {

	// negative limit means no upper bound (handled in cache)
	// zero limit means log nothing (handled here to avoid time wasting in cache)
	if requestLog.cache.Limit() == 0 {
		return
	}

	// these assignments are a bit hacky, but simplify our life
	request.resultCount = int64(result_count)
	request.resultSize = int64(result_size)
	request.serviceDuration = service_time
	request.totalDuration = request_time
	request.seqScanKeys = seq_scan_keys

	sqlID := AwrCB.recordWorkload(request)

	requestLog.RLock()
	defer requestLog.RUnlock()

	// first try all tags
	// all the qualifiers in a tag set must apply
	doLog := forceCapture
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
	qualifier := ""
	if !doLog {
		for _, q := range requestLog.qualifiers {
			doLog = q.evaluate(request, req)
			if doLog {
				qualifier = q.name()
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
		Tenant:          tenant.Bucket(request.TenantCtx()),
		SessionMemory:   request.SessionMemory(),
		SqlID:           sqlID,
		LogContent:      request.GetLogContent(),
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
	stmt := request.RedactedStatement()
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
	re.PhaseTimes = request.RawPhaseTimes()
	re.UsedMemory = request.UsedMemory()

	var start execution.Operator
	if !request.Sensitive() {
		// in order not to bloat service memory, we marshal the timings into a value
		// at the expense of request execution time
		timings := request.GetTimings()
		maxPlanSize := RequestsMaxPlanSize()
		if timings != nil {
			start = timings
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
				re.optEstimates = estimates
			} else {
				re.optEstimates = request.FmtOptimizerEstimates(timings)
			}
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
		memoryQuota := request.MemoryQuota()
		if memoryQuota != 0 {
			re.MemoryQuota = memoryQuota
		}
	}

	re.NamedArgs = request.FormattedRedactedNamedArgs()
	re.PositionalArgs = request.RedactedPositionalArgs()

	re.Users = datastore.CredsString(request.Credentials())
	re.RemoteAddr = request.RemoteAddr()
	userAgent := request.UserAgent()
	if userAgent != "" {
		re.UserAgent = userAgent
	}

	if natural := request.Natural(); natural != "" {
		re.NaturalLanguage = natural
		re.NaturalOutput = request.NaturalOutput()
		re.NaturalTime = request.NaturalTime()
	}

	clientId := request.ClientID().String()
	if clientId != "" {
		re.ClientId = clientId
	}
	if tag != "" {
		re.Tag = tag
	}
	re.ThrottleTime = request.ThrottleTime()
	if start != nil {
		re.Analysis, _ = execution.AnalyseExecution(start)
	}
	re.CpuTime = request.CpuTime()
	re.IoTime = request.IoTime()
	re.WaitTime = request.WaitTime()
	re.Timeout = request.Timeout()

	if qualifier != "" {
		re.Qualifier = qualifier
	}

	requestLog.cache.Add(re, id, nil)
	for _, h := range requestLog.handlers {
		h.handlerFunc(re) // Deliberately synchronous to limit the number of routines spawned
	}
}

func (this *RequestLogEntry) Timings() []byte {
	return this.timings
}

func (this *RequestLogEntry) OptEstimates() map[string]interface{} {
	return this.optEstimates
}

func (request *RequestLogEntry) Format(profiling bool, redact bool, durStyle util.DurationStyle) interface{} {
	reqMap := map[string]interface{}{
		"requestId": request.RequestId,
	}
	if request.SqlID != "" {
		reqMap["sqlID"] = request.SqlID
	}
	if request.ClientId != "" {
		reqMap["clientContextID"] = request.ClientId
	}
	reqMap["state"] = request.State
	reqMap["scanConsistency"] = request.ScanConsistency
	if request.UseFts {
		reqMap["useFts"] = request.UseFts
	}
	if request.UseCBO {
		reqMap["useCBO"] = request.UseCBO
	}
	if request.UseReplica == value.TRUE {
		reqMap["useReplica"] = value.TristateToString(request.UseReplica)
	}
	reqMap["n1qlFeatCtrl"] = request.FeatureControls
	if request.QueryContext != "" {
		reqMap["queryContext"] = request.QueryContext
	}
	if request.NaturalLanguage != "" {
		reqMap["naturalLanguagePrompt"] = util.Redacted(request.NaturalLanguage, redact)
	}
	if request.Statement != "" {
		reqMap["statement"] = util.Redacted(request.Statement, redact)
	}
	if request.StatementType != "" {
		reqMap["statementType"] = request.StatementType
	}
	if request.PreparedName != "" {
		reqMap["preparedName"] = request.PreparedName
		reqMap["preparedText"] = util.Redacted(request.PreparedText, redact)
	}
	if request.TxId != "" {
		reqMap["txid"] = request.TxId
	}
	reqMap["requestTime"] = request.Time.Format(expression.DEFAULT_FORMAT)
	reqMap["elapsedTime"] = util.FormatDuration(request.ElapsedTime, durStyle)
	reqMap["serviceTime"] = util.FormatDuration(request.ServiceTime, durStyle)
	if request.Timeout > time.Duration(0) {
		reqMap["timeout"] = util.FormatDuration(request.Timeout, durStyle)
	}
	if request.TransactionElapsedTime > 0 {
		reqMap["transactionElapsedTime"] = util.FormatDuration(request.TransactionElapsedTime, durStyle)
	}
	if request.TransactionRemainingTime > 0 {
		reqMap["transactionRemainingTime"] = util.FormatDuration(request.TransactionRemainingTime, durStyle)
	}
	if request.NaturalTime != 0 {
		reqMap["naturalLanguageProcessingTime"] = util.FormatDuration(request.NaturalTime, durStyle)
	}
	reqMap["resultCount"] = request.ResultCount
	reqMap["resultSize"] = request.ResultSize
	reqMap["errorCount"] = request.ErrorCount
	if request.Mutations != 0 {
		reqMap["mutations"] = request.Mutations
	}
	if request.PhaseCounts != nil {
		reqMap["phaseCounts"] = request.PhaseCounts
	}
	if request.PhaseOperators != nil {
		reqMap["phaseOperators"] = request.PhaseOperators
	}
	if request.PhaseTimes != nil {
		m := make(map[string]interface{}, len(request.PhaseTimes))
		for k, v := range request.PhaseTimes {
			if d, ok := v.(time.Duration); ok {
				m[k] = util.FormatDuration(d, durStyle)
			} else {
				m[k] = v
			}
		}
		reqMap["phaseTimes"] = m
	}
	if request.UsedMemory != 0 {
		reqMap["usedMemory"] = request.UsedMemory
	}
	if request.SessionMemory != 0 {
		reqMap["sessionMemory"] = request.SessionMemory
	}
	if request.Tag != "" {
		reqMap["~tag"] = request.Tag
	}

	if profiling {
		if request.NamedArgs != nil {
			reqMap["namedArgs"] = util.InterfaceRedacted(request.NamedArgs, redact)
		}
		if request.PositionalArgs != nil {
			reqMap["positionalArgs"] = util.InterfaceRedacted(request.PositionalArgs, redact)
		}
		timings := request.Timings()
		if timings != nil {
			reqMap["timings"] = util.InterfaceRedacted(string(util.ApplyDurationStyle(durStyle, timings)), redact)
		}
		if request.CpuTime > time.Duration(0) {
			reqMap["cpuTime"] = util.FormatDuration(request.CpuTime, durStyle)
		}
		if request.IoTime > time.Duration(0) {
			reqMap["ioTime"] = util.FormatDuration(request.IoTime, durStyle)
		}
		if request.WaitTime > time.Duration(0) {
			reqMap["waitTime"] = util.FormatDuration(request.WaitTime, durStyle)
		}
		optEstimates := request.OptEstimates()
		if optEstimates != nil {
			reqMap["optimizerEstimates"] = value.NewValue(util.InterfaceRedacted(optEstimates, redact))
		}
		if request.Errors != nil {
			// value.NewValue needs to understand the type
			errs := make([]interface{}, len(request.Errors))
			for i := range request.Errors {
				errs[i] = request.Errors[i]
			}
			reqMap["errors"] = errs
		}
		if request.MemoryQuota != 0 {
			reqMap["memoryQuota"] = request.MemoryQuota
		}
	}
	if request.Users != "" {
		reqMap["users"] = util.Redacted(request.Users, redact)
	}
	if request.RemoteAddr != "" {
		reqMap["remoteAddr"] = request.RemoteAddr
	}
	if request.UserAgent != "" {
		reqMap["userAgent"] = request.UserAgent
	}
	if request.ThrottleTime > time.Duration(0) {
		reqMap["throttleTime"] = util.FormatDuration(request.ThrottleTime, durStyle)
	}
	if request.Qualifier != "" {
		reqMap["~qualifier"] = request.Qualifier
	}
	if len(request.Analysis) > 0 {
		reqMap["~analysis"] = request.Analysis
	}
	if len(request.LogContent) > 0 {
		reqMap["~log"] = request.LogContent
	}
	return reqMap
}

// request qualifiers

// 1- threshold
type timeThreshold struct {
	threshold time.Duration
}

func newTimeThreshold(c interface{}) (qualifier, errors.Error) {
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
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *timeThreshold) evaluate(request *BaseRequest, req *http.Request) bool {

	// negative threshold means log nothing
	// zero threshold means log everything (no threshold)
	switch {
	case this.threshold < 0:
		return false
	case this.threshold == 0:
		return true
	default:
		if tenant.IsServerless() {
			return (request.serviceDuration >= time.Millisecond*this.threshold ||
				request.totalDuration >= time.Millisecond*this.threshold ||
				request.throttleTime >= time.Millisecond*this.threshold)
		} else {
			return request.serviceDuration >= time.Millisecond*this.threshold
		}
	}
}

// 2- aborted
type aborted struct {
	// run along, nothing to see here
}

func newAborted(c interface{}) (qualifier, errors.Error) {
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
	return request.State() == ABEND
}

// 3- errors
type reqError struct {
	errCode int
}

func newReqError(c interface{}) (qualifier, errors.Error) {
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

func newUser(c interface{}) (qualifier, errors.Error) {
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

func newClient(c interface{}) (qualifier, errors.Error) {
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

func newContext(c interface{}) (qualifier, errors.Error) {
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

func newResults(c interface{}) (qualifier, errors.Error) {
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
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *results) evaluate(request *BaseRequest, req *http.Request) bool {
	return request.resultCount > this.count
}

// 8- mutation count
type mutations struct {
	count int64
}

func newMutations(c interface{}) (qualifier, errors.Error) {
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
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *mutations) evaluate(request *BaseRequest, req *http.Request) bool {
	return int64(request.mutationCount) > this.count
}

// 9- output size
type size struct {
	size int64
}

func newSize(c interface{}) (qualifier, errors.Error) {
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
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *size) evaluate(request *BaseRequest, req *http.Request) bool {
	return request.resultSize > this.size
}

// 10- operator counts
type counts struct {
	count int64
}

func newCounts(c interface{}) (qualifier, errors.Error) {
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
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *counts) evaluate(request *BaseRequest, req *http.Request) bool {
	return int64(request.phaseStats[execution.FETCH].count) > this.count ||
		int64(request.phaseStats[execution.PRIMARY_SCAN].count) > this.count ||
		int64(request.phaseStats[execution.INDEX_SCAN].count) > this.count ||
		int64(request.phaseStats[execution.PRIMARY_SCAN_GSI].count) > this.count ||
		int64(request.phaseStats[execution.INDEX_SCAN_GSI].count) > this.count ||
		int64(request.phaseStats[execution.PRIMARY_SCAN_FTS].count) > this.count ||
		int64(request.phaseStats[execution.INDEX_SCAN_FTS].count) > this.count ||
		int64(request.phaseStats[execution.PRIMARY_SCAN_SEQ].count) > this.count ||
		int64(request.phaseStats[execution.INDEX_SCAN_SEQ].count) > this.count ||
		int64(request.phaseStats[execution.NL_JOIN].count) > this.count ||
		int64(request.phaseStats[execution.HASH_JOIN].count) > this.count ||
		int64(request.phaseStats[execution.SORT].count) > this.count
}

// 11- statement text
type statement struct {
	pattern *regexp.Regexp
	like    string
}

func newStatement(c interface{}) (qualifier, errors.Error) {
	switch c := c.(type) {
	case string:
		re, _, err := expression.LikeCompile(c, '\\')
		if err == nil || len(c) == 0 {
			return &statement{like: c, pattern: re}, nil
		}
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("statement", c)
}

func (this *statement) name() string {
	return "statement"
}

func (this *statement) unique() bool {
	return false
}

func (this *statement) condition() interface{} {
	return this.like
}

func (this *statement) isCondition(c interface{}) bool {
	switch c.(type) {
	case string:
		return c.(string) == this.like
	}
	return false
}

func (this *statement) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case string:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *statement) evaluate(request *BaseRequest, req *http.Request) bool {
	switch request.Type() {
	case "SELECT", "UPDATE", "INSERT", "UPSERT", "DELETE", "MERGE":
	default:
		return false
	}
	if this.pattern == nil {
		return false
	}
	return this.pattern.MatchString(request.Statement())
}

// 12- plan element
type elemfilter struct {
	key     string
	value   interface{}
	jsonKey string
	jsonVal string
	result  bool
}

type elemfilters []*elemfilter

func (this *elemfilter) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	k := "+"
	if !this.result {
		k = "-"
	}
	k += this.key
	m[k] = this.value
	return json.Marshal(m)
}

func (this *elemfilter) Equals(c interface{}) bool {
	if f, ok := c.(*elemfilter); ok {
		if this.key != f.key || this.result != f.result {
			return false
		}
		return compare(this.value, f.value)
	}
	return false
}

func (this elemfilters) Equals(c interface{}) bool {
	if ca, ok := c.(elemfilters); ok {
		if len(ca) != len(this) {
			return false
		}
		for i := range this {
			if !this[i].Equals(ca[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (this elemfilters) String() string {
	s := ""
	for _, v := range this {
		s += fmt.Sprintf(", %v", v)
	}
	if len(s) > 0 {
		return s[2:]
	}
	return ""
}

type plan_element struct {
	criteria elemfilters
}

func newPlanElement(c interface{}) (qualifier, errors.Error) {
	switch c := c.(type) {
	case map[string]interface{}:
		f := makeMapFilters(c)
		return &plan_element{criteria: f}, nil
	case []interface{}:
		f, ok := makeFilters(c)
		if ok {
			return &plan_element{criteria: f}, nil
		}
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("plan", c)
}

func (this *plan_element) name() string {
	return "plan"
}

func (this *plan_element) unique() bool {
	return false
}

func (this *plan_element) condition() interface{} {
	return this.criteria
}

func (this *plan_element) Equals(c interface{}) bool {
	return this.isCondition(c)
}

func (this *plan_element) isCondition(c interface{}) bool {
	switch c := c.(type) {
	case []interface{}:
		f, ok := makeFilters(c)
		if !ok {
			return false
		}
		return compare(this.criteria, f)
	case map[string]interface{}:
		f := makeMapFilters(c)
		return compare(this.criteria, f)
	case elemfilters:
		return compare(this.criteria, c)
	}
	return false
}

func makeFilters(a []interface{}) (elemfilters, bool) {
	var filters elemfilters
	for i := range a {
		switch m := a[i].(type) {
		case map[string]interface{}:
			filters = append(filters, makeMapFilters(m)...)
		case []interface{}:
			f, ok := makeFilters(m)
			if !ok {
				return nil, false
			}
			filters = append(filters, f...)
		default:
			return nil, false
		}
	}
	return filters, true
}

func makeMapFilters(m map[string]interface{}) elemfilters {
	var filters []*elemfilter
	for mk, mv := range m {
		expectedRes := true
		if mk[0] == '+' {
			mk = mk[1:]
		} else if mk[0] == '-' {
			expectedRes = false
			mk = mk[1:]
		}
		jk := "\"" + mk + "\":"
		jv := ""
		b, err := json.Marshal(mv)
		if err == nil {
			jv = fmt.Sprintf("%s", string(b))
		} else {
			jv = fmt.Sprintf("%v", mv)
		}
		filters = append(filters, &elemfilter{mk, mv, jk, jv, expectedRes})
	}
	return filters
}

func (this *plan_element) checkCondition(c interface{}) errors.Error {
	switch c.(type) {
	case map[string]interface{}:
		return nil
	}
	return errors.NewCompletedQualifierInvalidArgument(this.name(), c)
}

func (this *plan_element) evaluate(request *BaseRequest, req *http.Request) bool {
	switch request.Type() {
	case "SELECT", "UPDATE", "INSERT", "UPSERT", "DELETE", "MERGE":
	default:
		return false
	}
	t := request.GetTimings()
	if t == nil {
		return false
	}
	b := request.GetFmtTimings()
	if b == nil {
		var err error
		b, err = json.Marshal(t)
		if err != nil {
			return false
		}
		request.SetFmtTimings(b)
	}
	// plan may or may not be in indented format, match key then value ignoring whitespace
	plan := string(b)
	for _, p := range this.criteria {
		start := 0
		i := strings.Index(plan[start:], p.jsonKey)
		found := false
		for i != -1 {
			i += start + len(p.jsonKey)
			found = true
			quoted := false
			escaped := false
			for j := range p.jsonVal {
				if !quoted {
					for unicode.IsSpace(rune(plan[i])) {
						i++
					}
				}
				if !escaped && p.jsonVal[j] == '"' {
					quoted = !quoted
				}
				if !escaped && p.jsonVal[j] == '\\' {
					escaped = true
				} else {
					escaped = false
				}
				if plan[i] != p.jsonVal[j] {
					found = false
					break
				}
				i++
			}
			if found && p.result {
				break
			}
			start = i
			i = strings.Index(plan[start:], p.jsonKey)
		}
		if found != p.result {
			return false
		}
	}
	return true
}

// 13- error count (errors)
type reqErrorCount struct {
	count int
}

func newReqErrorCount(c interface{}) (qualifier, errors.Error) {
	switch c := c.(type) {
	case int:
		if c >= 0 {
			return &reqErrorCount{count: c}, nil
		}
	case int64:
		if c >= 0 {
			return &reqErrorCount{count: int(c)}, nil
		}
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("errors", c)
}

func (this *reqErrorCount) name() string {
	return "errors"
}

func (this *reqErrorCount) unique() bool {
	return true
}

func (this *reqErrorCount) condition() interface{} {
	return this.count
}

func (this *reqErrorCount) isCondition(c interface{}) bool {
	switch c.(type) {
	case int:
		return c.(int) == this.count
	case int64:
		return int(c.(int64)) == this.count
	}
	return false
}

func (this *reqErrorCount) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *reqErrorCount) evaluate(request *BaseRequest, req *http.Request) bool {
	if request.GetErrorCount() >= this.count {
		return true
	}
	return false
}

// 14- sequential scan key count (seqscan_count)
type seqScanKeys struct {
	count int64
}

func newSeqScanKeys(c interface{}) (qualifier, errors.Error) {
	switch c := c.(type) {
	case int:
		if c >= 0 {
			return &seqScanKeys{count: int64(c)}, nil
		}
	case int64:
		if c >= 0 {
			return &seqScanKeys{count: c}, nil
		}
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("seqscan_keys", c)
}

func (this *seqScanKeys) name() string {
	return "seqscan_keys"
}

func (this *seqScanKeys) unique() bool {
	return true
}

func (this *seqScanKeys) condition() interface{} {
	return this.count
}

func (this *seqScanKeys) isCondition(c interface{}) bool {
	switch c := c.(type) {
	case int:
		return int64(c) == this.count
	case int64:
		return c == this.count
	}
	return false
}

func (this *seqScanKeys) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *seqScanKeys) evaluate(request *BaseRequest, req *http.Request) bool {
	if request.seqScanKeys >= this.count {
		return true
	}
	return false
}

// 15- used memory
type usedMemory struct {
	size uint64
}

func newUsedMemory(s interface{}) (qualifier, errors.Error) {
	switch s := s.(type) {
	case int:
		if s >= 0 {
			return &usedMemory{size: uint64(s)}, nil
		}
	case int64:
		if s >= 0 {
			return &usedMemory{size: uint64(s)}, nil
		}
	}
	return nil, errors.NewCompletedQualifierInvalidArgument("used_memory", s)
}

func (this *usedMemory) name() string {
	return "used_memory"
}

func (this *usedMemory) unique() bool {
	return true
}

func (this *usedMemory) condition() interface{} {
	return this.size
}

func (this *usedMemory) isCondition(c interface{}) bool {
	switch c := c.(type) {
	case int:
		return uint64(c) == this.size
	case int64:
		return uint64(c) == this.size
	}
	return false
}

func (this *usedMemory) checkCondition(c interface{}) errors.Error {
	return nil
}

func (this *usedMemory) evaluate(request *BaseRequest, req *http.Request) bool {
	return request.UsedMemory() >= this.size
}
