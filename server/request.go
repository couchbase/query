//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/logging/event"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type State int32

const (
	SUBMITTED State = iota
	PREPROCESSING
	RUNNING
	SUCCESS
	ERRORS
	COMPLETED
	STOPPED
	TIMEOUT
	CLOSED
	FATAL
	ABEND /* Changed from ABORTED to ABEND, flags that there was an internal error like panic */
)

var states = [...]string{
	"submitted",
	"pre-processing",
	"running",
	"success",
	"errors",
	"completed",
	"stopped",
	"timeout",
	"closed",
	"fatal",
	"aborted", // unchanged for backward compatibility, this is state string for ABEND
}

type Request interface {
	Id() RequestID
	ClientID() ClientContextID
	SetClientID(id string)
	Statement() string
	SetStatement(statement string)
	IncrementStatementCount()
	Natural() string
	SetNatural(natural string)
	SetNaturalContext(s string)
	NaturalContext() string
	SetNaturalCred(cred string)
	NaturalCred() string
	SetNaturalOrganizationId(orgId string)
	NaturalOrganizationId() string
	SetNaturalStatement(algebra.Statement)
	NaturalStatement() algebra.Statement
	NaturalShowOnly() bool
	Prepared() *plan.Prepared
	SetPrepared(prepared *plan.Prepared)
	Type() string
	SetType(string)
	IsPrepare() bool
	SetIsPrepare(bool)
	NamedArgs() map[string]value.Value
	SetNamedArgs(args map[string]value.Value)
	PositionalArgs() value.Values
	SetPositionalArgs(args value.Values)
	Namespace() string
	SetNamespace(namespace string)
	Timeout() time.Duration
	SetTimeout(timeout time.Duration)
	SetTimer(*time.Timer)
	MaxParallelism() int
	SetMaxParallelism(maxParallelism int)
	ScanCap() int64
	SetScanCap(scanCap int64)
	PipelineCap() int64
	SetPipelineCap(pipelineCap int64)
	PipelineBatch() int
	SetPipelineBatch(pipelineBatch int)
	Readonly() value.Tristate
	SetReadonly(readonly value.Tristate)
	Metrics() value.Tristate
	SetMetrics(metrics value.Tristate)
	Signature() value.Tristate
	SetSignature(signature value.Tristate)
	Pretty() value.Tristate
	SetPretty(pretty value.Tristate)
	Controls() value.Tristate
	SetControls(controls value.Tristate)
	Profile() Profile
	SetProfile(p Profile)
	ScanConsistency() datastore.ScanConsistency
	SetScanConfiguration(consistency ScanConfiguration)
	OriginalScanConsistency() datastore.ScanConsistency
	SetScanConsistency(consistency datastore.ScanConsistency)
	ScanVectorSource() timestamp.ScanVectorSource
	IndexApiVersion() int
	SetIndexApiVersion(ver int)
	FeatureControls() uint64
	SetFeatureControls(controls uint64)
	AutoPrepare() value.Tristate
	SetAutoPrepare(a value.Tristate)
	AutoExecute() value.Tristate
	SetAutoExecute(a value.Tristate)
	SetQueryContext(s string)
	QueryContext() string
	UseFts() bool
	SetUseFts(a bool)
	UseCBO() bool
	SetUseCBO(useCBO bool)
	UseReplica() value.Tristate
	SetUseReplica(useReplica value.Tristate)
	MemoryQuota() uint64
	SetMemoryQuota(q uint64)
	UsedMemory() uint64
	TxId() string
	SetTxId(s string)
	TxImplicit() bool
	SetTxImplicit(b bool)
	TxStmtNum() int64
	SetTxStmtNum(n int64)
	TxTimeout() time.Duration
	SetTxTimeout(d time.Duration)
	TxData() []byte
	SetTxData(b []byte)
	DurabilityLevel() datastore.DurabilityLevel
	SetDurabilityLevel(l datastore.DurabilityLevel)
	DurabilityTimeout() time.Duration
	SetDurabilityTimeout(d time.Duration)
	KvTimeout() time.Duration
	SetKvTimeout(d time.Duration)
	AtrCollection() string
	SetAtrCollection(s string)
	NumAtrs() int
	SetNumAtrs(n int)
	PreserveExpiry() bool
	SetPreserveExpiry(a bool)
	ExecutionContext() *execution.Context
	SetExecutionContext(ctx *execution.Context)
	SetExecTime(time time.Time)
	RequestTime() time.Time
	ServiceTime() time.Time
	TransactionStartTime() time.Time
	SetTransactionStartTime(t time.Time)
	Output() execution.Output
	Servicing()
	Fail(err errors.Error)
	CompletedNaturalRequest(srvr *Server)
	Error(err errors.Error)
	Execute(server *Server, context *execution.Context, reqType string, signature value.Value, startTx bool)
	NotifyStop(stop execution.Operator)
	Failed(server *Server)
	Expire(state State, timeout time.Duration)
	SortCount() uint64
	State() State
	SetState(State)
	Halted() bool
	Credentials() *auth.Credentials
	SetCredentials(credentials *auth.Credentials)
	RemoteAddr() string
	SetRemoteAddr(remoteAddr string)
	UserAgent() string
	SetUserAgent(userAgent string)
	SetTimings(o execution.Operator)
	GetTimings() execution.Operator
	SetFmtTimings(e []byte)
	GetFmtTimings() []byte
	SetFmtOptimizerEstimates(e map[string]interface{})
	GetFmtOptimizerEstimates() map[string]interface{}
	IsAdHoc() bool
	SetErrorLimit(limit int)
	GetErrorLimit() int
	SetTracked()
	Tracked() bool
	SetTenantCtx(ctx tenant.Context)
	TenantCtx() tenant.Context
	SortProjection() bool
	ThrottleTime() time.Duration
	CpuTime() time.Duration
	IoTime() time.Duration
	WaitTime() time.Duration
	SetThrottleTime(d time.Duration)
	Alive() bool
	Loga(logging.Level, func() string)
	LogLevel() logging.Level

	setSleep() // internal methods for load control
	sleep()
	release()

	DurationStyle() util.DurationStyle
	SetDurationStyle(util.DurationStyle)

	SetAdmissionWaitTime(time.Duration)
	AdmissionWaitTime() time.Duration

	Sensitive() bool
	RedactedStatement() string
	RedactedNamedArgs() map[string]value.Value
	RedactedPositionalArgs() value.Values

	SessionMemory() uint64

	Halt(err errors.Error)

	Format(util.DurationStyle, bool, bool, bool) map[string]interface{}
	NaturalTime() time.Duration
}

type RequestID interface {
	String() string
}

type ClientContextID interface {
	IsValid() bool
	String() string
}

type ScanConsistency int

const (
	NOT_SET ScanConsistency = iota
	NOT_BOUNDED
	REQUEST_PLUS
	STATEMENT_PLUS
	AT_PLUS
	UNDEFINED_CONSISTENCY
)

type ScanConfiguration interface {
	ScanConsistency() datastore.ScanConsistency
	ScanWait() time.Duration
	ScanVectorSource() timestamp.ScanVectorSource
	SetScanConsistency(consistency datastore.ScanConsistency) interface{}
}

// API for tracking active requests
type ActiveRequests interface {

	// adds a request to the active queue
	Put(Request) errors.Error

	// processes a request
	Get(string, func(Request)) errors.Error

	// removes a request from the active queue / returns success
	Delete(string, bool, func(Request) bool) bool

	// request count
	Count() (int, errors.Error)

	// processes all requests
	// first function processes within lock (must be non blocking)
	// second function processes outside of a lock (can be blocking)
	// both return false if no more processing should be done
	ForEach(func(string, Request) bool, func() bool)

	// current active request server load
	Load() int
}

var actives ActiveRequests

func ActiveRequestsCount() (int, errors.Error) {
	if actives != nil {
		return actives.Count()
	}
	return 0, nil
}

func ActiveRequestsDelete(id string) bool {
	if actives != nil {
		return actives.Delete(id, true, nil)
	}
	return false
}

func ActiveRequestsDeleteFunc(id string, f func(Request) bool) bool {
	if actives != nil {
		return actives.Delete(id, true, f)
	}
	return false
}

func ActiveRequestsGet(id string, f func(Request)) errors.Error {
	if actives != nil {
		return actives.Get(id, f)
	}
	return nil
}

func ActiveRequestsForEach(nonBlocking func(string, Request) bool, blocking func() bool) {
	if actives != nil {
		actives.ForEach(nonBlocking, blocking)
	}
}

func ActiveRequestsLoad() int {
	if actives != nil {
		return actives.Load()
	}
	return 0
}

func SetActives(ar ActiveRequests) {
	actives = ar
}

type BaseRequest struct {
	// Aligned ints need to be declared right at the top
	// of the struct to avoid alignment issues on x86 platforms
	usedMemory    atomic.AlignedUint64
	mutationCount atomic.AlignedUint64
	sortCount     atomic.AlignedUint64
	cpuTime       atomic.AlignedUint64
	ioTime        atomic.AlignedUint64
	waitTime      atomic.AlignedUint64
	phaseStats    [execution.PHASES]phaseStat
	tenantUnits   tenant.Services

	sync.RWMutex
	id                   requestIDImpl
	client_id            clientContextIDImpl
	statement            string
	redactedStatement    string
	prepared             *plan.Prepared
	reqType              string
	isPrepare            bool
	namedArgs            map[string]value.Value
	positionalArgs       value.Values
	namespace            string
	timeout              time.Duration
	timer                *time.Timer
	maxParallelism       int
	scanCap              int64
	pipelineCap          int64
	pipelineBatch        int
	readonly             value.Tristate
	signature            value.Tristate
	metrics              value.Tristate
	pretty               value.Tristate
	consistency          ScanConfiguration
	credentials          *auth.Credentials
	remoteAddr           string
	userAgent            string
	requestTime          time.Time
	serviceTime          time.Time
	execTime             time.Time
	transactionStartTime time.Time
	state                State
	abend                bool
	errorLimit           int
	errorCount           int
	duplicateErrorCount  int
	warningCount         int
	errors               []errors.Error
	warnings             []errors.Error
	stopGate             sync.WaitGroup
	servicerGate         sync.WaitGroup
	stopResult           chan bool          // stop consuming results
	stopExecute          chan bool          // stop executing request
	stopOperator         execution.Operator // notified when request execution stops
	timings              execution.Operator
	fmtTimings           []byte
	fmtEstimates         map[string]interface{}
	controls             value.Tristate
	profile              Profile
	indexApiVersion      int    // Index API version
	featureControls      uint64 // feature bit controls
	autoPrepare          value.Tristate
	autoExecute          value.Tristate
	useFts               bool
	useCBO               bool
	useReplica           value.Tristate
	queryContext         string
	memoryQuota          uint64
	txId                 string
	txImplicit           bool
	txStmtNum            int64
	txTimeout            time.Duration
	txData               []byte
	durabilityTimeout    time.Duration
	durabilityLevel      datastore.DurabilityLevel
	kvTimeout            time.Duration
	atrCollection        string
	numAtrs              int
	preserveExpiry       bool
	executionContext     *execution.Context
	resultCount          int64
	resultSize           int64
	serviceDuration      time.Duration
	totalDuration        time.Duration
	tracked              bool
	tenantCtx            tenant.Context
	sortProjection       bool
	throttleTime         time.Duration
	logLevel             logging.Level
	durationStyle        util.DurationStyle
	seqScanKeys          int64
	natural              string
	naturalCred          string
	naturalOrgId         string
	naturalContext       string
	nlStatement          algebra.Statement
	nlShowOnly           bool
}

type requestIDImpl struct {
	id string
}

type phaseStat struct {
	count     atomic.AlignedUint64
	operators atomic.AlignedUint64
	duration  atomic.AlignedUint64
}

// requestIDImpl implements the RequestID interface
func (r *requestIDImpl) String() string {
	return r.id
}

type clientContextIDImpl struct {
	id string
}

func (this *clientContextIDImpl) IsValid() bool {
	return len(this.id) > 0
}

func (this *clientContextIDImpl) String() string {
	return this.id
}

func NewBaseRequest(rv *BaseRequest) {
	rv.timeout = -1
	rv.txTimeout = datastore.DEF_TXTIMEOUT
	rv.state = SUBMITTED
	rv.abend = false
	rv.stopResult = make(chan bool, 1)
	rv.stopExecute = make(chan bool, 1)
	rv.metrics = value.NONE
	rv.pretty = value.NONE
	rv.readonly = value.NONE
	rv.signature = value.NONE
	rv.profile = ProfUnset
	rv.controls = value.NONE
	rv.autoPrepare = value.NONE
	rv.indexApiVersion = util.GetMaxIndexAPI()
	rv.featureControls = util.GetN1qlFeatureControl()
	rv.id.id, _ = util.UUIDV4()
	rv.client_id.id = ""
	rv.SetMaxParallelism(1)
	rv.useCBO = util.GetUseCBO()
	rv.useReplica = value.NONE
	rv.durabilityTimeout = datastore.DEF_DURABILITY_TIMEOUT
	rv.kvTimeout = datastore.DEF_KVTIMEOUT
	rv.durabilityLevel = datastore.DL_UNSET
	rv.errorLimit = -1
	rv.durationStyle = util.DEFAULT
}

func (this *BaseRequest) SetRequestTime(time time.Time) {
	this.requestTime = time
}

func (this *BaseRequest) SetExecTime(time time.Time) {
	this.execTime = time
}

func (this *BaseRequest) SetTimer(timer *time.Timer) {
	this.timer = timer
}

func (this *BaseRequest) Id() RequestID {
	return &this.id
}

func (this *BaseRequest) ClientID() ClientContextID {
	return &this.client_id
}

func (this *BaseRequest) SetClientID(id string) {
	this.client_id.id = id
}

func (this *BaseRequest) Statement() string {
	return this.statement
}

const _REDACT_TOKEN = "****"

func (this *BaseRequest) RedactedStatement() string {
	if len(this.statement) == 0 || !this.Sensitive() {
		return this.statement
	}
	if len(this.redactedStatement) > 0 {
		return this.redactedStatement
	}

	// attempt to redact literal strings coming after the keyword PASSWORD
	var buf strings.Builder
	buf.Grow(len(this.statement))

	var n int
	var r, pr rune
	quote := utf8.RuneError
	comment := utf8.RuneError
	var escaped, redact bool

	for i := 0; i < len(this.statement); i += n {
		r, n = utf8.DecodeRuneInString(this.statement[i:])
		if r == utf8.RuneError {
			break
		}
		if escaped {
			buf.WriteRune(r)
			escaped = false
		} else if comment != utf8.RuneError {
			buf.WriteRune(r)
			if (comment == '/' && r == '/' && pr == '*') || (comment == '-' && r == '\n') {
				comment = utf8.RuneError
			}
			pr = r
		} else if quote != utf8.RuneError {
			if r == quote {
				quote = utf8.RuneError
				redact = false
				buf.WriteRune(r)
			} else if !redact {
				buf.WriteRune(r)
			}
		} else if r == '"' || r == '\'' || r == '`' {
			quote = r
			buf.WriteRune(r)
			if redact {
				buf.WriteString(_REDACT_TOKEN)
			}
		} else if len(this.statement) > i+n &&
			((r == '/' && this.statement[i+n] == '*') || (r == '-' && this.statement[i+n] == '-')) {

			buf.WriteRune(r)
			comment = r
			pr = r
		} else if i+8 < len(this.statement) && strings.ToLower(this.statement[i:i+8]) == "password" {
			buf.WriteString(this.statement[i : i+8])
			i += 7
			redact = true
		} else {
			if redact && !unicode.IsSpace(r) {
				redact = false
			}
			buf.WriteRune(r)
		}
	}
	this.redactedStatement = buf.String()
	return this.redactedStatement
}

func (this *BaseRequest) Sensitive() bool {
	switch this.reqType {
	case "CREATE_USER":
		return true
	case "ALTER_USER":
		return true
	default:
		return false
	}
}

func (this *BaseRequest) SetStatement(statement string) {
	this.statement = statement
}

func (this *BaseRequest) Prepared() *plan.Prepared {
	return this.prepared
}

func (this *BaseRequest) Type() string {
	return this.reqType
}

func (this *BaseRequest) IsPrepare() bool {
	return this.isPrepare
}

func (this *BaseRequest) NamedArgs() map[string]value.Value {
	return this.namedArgs
}

var _REDACTED_VALUE = value.NewValue(map[string]interface{}{"redacted": true})

func (this *BaseRequest) RedactedNamedArgs() map[string]value.Value {
	if !this.Sensitive() {
		return this.namedArgs
	}
	rv := make(map[string]value.Value, len(this.namedArgs))
	for k, _ := range this.namedArgs {
		rv[k] = _REDACTED_VALUE
	}
	return rv
}

func (this *BaseRequest) SetNamedArgs(args map[string]value.Value) {
	this.namedArgs = args
}

func (this *BaseRequest) PositionalArgs() value.Values {
	return this.positionalArgs
}

func (this *BaseRequest) RedactedPositionalArgs() value.Values {
	if !this.Sensitive() {
		return this.positionalArgs
	}
	rv := make(value.Values, len(this.positionalArgs))
	for i := range this.positionalArgs {
		rv[i] = _REDACTED_VALUE
	}
	return rv
}

func (this *BaseRequest) SetPositionalArgs(args value.Values) {
	this.positionalArgs = args
}

func (this *BaseRequest) Namespace() string {
	return this.namespace
}

func (this *BaseRequest) SetNamespace(namespace string) {
	this.namespace = namespace
}

func (this *BaseRequest) Timeout() time.Duration {
	return this.timeout
}

func (this *BaseRequest) SetTimeout(timeout time.Duration) {
	this.timeout = timeout
}

func (this *BaseRequest) MaxParallelism() int {
	return this.maxParallelism
}

func (this *BaseRequest) SetMaxParallelism(maxParallelism int) {
	if maxParallelism <= 0 {
		maxParallelism = util.NumCPU()
	}
	this.maxParallelism = maxParallelism
}

func (this *BaseRequest) ScanCap() int64 {
	return this.scanCap
}

func (this *BaseRequest) SetScanCap(scanCap int64) {
	this.scanCap = scanCap
}

func (this *BaseRequest) PipelineCap() int64 {
	return this.pipelineCap
}

func (this *BaseRequest) SetPipelineCap(pipelineCap int64) {
	this.pipelineCap = pipelineCap
}

func (this *BaseRequest) PipelineBatch() int {
	return this.pipelineBatch
}

func (this *BaseRequest) SetPipelineBatch(pipelineBatch int) {
	this.pipelineBatch = pipelineBatch
}

func (this *BaseRequest) Readonly() value.Tristate {
	return this.readonly
}

func (this *BaseRequest) SetReadonly(readonly value.Tristate) {
	this.readonly = readonly
}

func (this *BaseRequest) Signature() value.Tristate {
	return this.signature
}

func (this *BaseRequest) SetSignature(signature value.Tristate) {
	this.signature = signature
}

func (this *BaseRequest) Metrics() value.Tristate {
	return this.metrics
}

func (this *BaseRequest) SetMetrics(metrics value.Tristate) {
	this.metrics = metrics
}

func (this *BaseRequest) Pretty() value.Tristate {
	return this.pretty
}

func (this *BaseRequest) SetPretty(pretty value.Tristate) {
	this.pretty = pretty
}

func (this *BaseRequest) OriginalScanConsistency() datastore.ScanConsistency {
	if this.consistency == nil {
		return datastore.NOT_SET
	}
	return this.consistency.ScanConsistency()
}

func (this *BaseRequest) SetScanConsistency(consistency datastore.ScanConsistency) {
	this.consistency = this.consistency.SetScanConsistency(consistency).(ScanConfiguration)
}

func (this *BaseRequest) ScanConsistency() datastore.ScanConsistency {
	consistency := this.OriginalScanConsistency()
	if consistency == datastore.NOT_SET {
		consistency = datastore.UNBOUNDED
	}
	return consistency
}

func (this *BaseRequest) SetScanConfiguration(consistency ScanConfiguration) {
	this.consistency = consistency
}

func (this *BaseRequest) ScanVectorSource() timestamp.ScanVectorSource {
	if this.consistency == nil {
		return nil
	}
	return this.consistency.ScanVectorSource()
}

func (this *BaseRequest) RequestTime() time.Time {
	return this.requestTime
}

func (this *BaseRequest) ServiceTime() time.Time {
	return this.serviceTime
}

func (this *BaseRequest) ExecTime() time.Time {
	return this.execTime
}

func (this *BaseRequest) TransactionStartTime() time.Time {
	return this.transactionStartTime
}

func (this *BaseRequest) SetTransactionStartTime(t time.Time) {
	this.transactionStartTime = t
}

func (this *BaseRequest) SetPrepared(prepared *plan.Prepared) {
	this.Lock()
	defer this.Unlock()
	this.prepared = prepared
}

func (this *BaseRequest) SetType(reqType string) {
	this.Lock()
	defer this.Unlock()
	this.reqType = reqType
}

func (this *BaseRequest) SetIsPrepare(ip bool) {
	this.Lock()
	defer this.Unlock()
	this.isPrepare = ip
}

func (this *BaseRequest) SetState(state State) {
	this.Lock()
	defer this.Unlock()

	// Once we transition to TIMEOUT or CLOSE, we don't transition
	// to STOPPED or COMPLETED to allow the request to close
	// gracefully on timeout or network errors and report the
	// right state. Ditto for FATAL.
	if this.state == FATAL ||
		((this.state == TIMEOUT || this.state == CLOSED || this.state == STOPPED) &&
			(state == STOPPED || state == COMPLETED)) {
		return
	}
	this.state = state
}

func (this *BaseRequest) State() State {
	this.RLock()
	defer this.RUnlock()
	if this.abend {
		return ABEND
	}
	return this.state
}

func (this State) StateName() string {
	return states[int(this)]
}

func (this *BaseRequest) Halted() bool {

	// we purposly do not take the lock
	// as this is used repeatedly in Execution()
	// if we mistakenly report the State as RUNNING,
	// we'll catch the right state in other places...
	state := State(atomic.LoadInt32((*int32)(&this.state)))
	return state != RUNNING && state != SUBMITTED && state != PREPROCESSING
}

func (this *BaseRequest) Credentials() *auth.Credentials {
	return this.credentials
}

func (this *BaseRequest) SetCredentials(credentials *auth.Credentials) {
	this.credentials = credentials
}

func (this *BaseRequest) RemoteAddr() string {
	return this.remoteAddr
}

func (this *BaseRequest) SetRemoteAddr(remoteAddr string) {
	this.remoteAddr = remoteAddr
}

func (this *BaseRequest) UserAgent() string {
	return this.userAgent
}

func (this *BaseRequest) SetUserAgent(userAgent string) {
	this.userAgent = userAgent
}

func (this *BaseRequest) SetServiceTime() {
	this.serviceTime = time.Now()
}

func (this *BaseRequest) Servicing() {
	this.serviceTime = time.Now()
	this.state = RUNNING
}

func (this *BaseRequest) Fatal(err errors.Error) {
	this.Error(err)
	this.Stop(FATAL)
}

func (this *BaseRequest) Abort(err errors.Error) {
	this.abend = true
	this.Error(err)
	this.Stop(FATAL)
}

func (this *BaseRequest) SetErrorLimit(limit int) {
	if limit < 0 {
		limit = 0
	}
	this.errorLimit = limit
}

func (this *BaseRequest) GetErrorLimit() int {
	return this.errorLimit
}

func (this *BaseRequest) GetErrorCount() int {
	return this.errorCount
}

func (this *BaseRequest) GetWarningCount() int {
	return this.warningCount
}

func (this *BaseRequest) Error(err errors.Error) {
	if err.Level() == errors.WARNING {
		this.Warning(err)
		return
	}

	this.Lock()
	if err.Level() == errors.EXCEPTION {
		this.errors = append(this.errors, err)
		this.errorCount++
		this.Unlock()
		this.Stop(FATAL)
		return
	}

	if this.errorLimit > 0 && this.errorCount+this.duplicateErrorCount >= this.errorLimit {
		this.errors = append(this.errors,
			errors.NewErrorLimit(this.errorLimit, this.errorCount, this.duplicateErrorCount, this.MutationCount()))
		this.errorCount++
		this.Unlock()
		this.Stop(FATAL)
		return
	}
	this.addErrorLOCKED(err)
	this.Unlock()
}

func (this *BaseRequest) addErrorLOCKED(err errors.Error) {
	// don't add duplicate errors
	for _, e := range this.errors {
		if err.Code() != 0 && err.Code() == e.Code() && err.Error() == e.Error() {
			e.Repeat()
			this.duplicateErrorCount++
			return
		}
	}
	this.errors = append(this.errors, err)
	this.errorCount++
}

func (this *BaseRequest) Warning(wrn errors.Error) {
	this.Lock()
	this.addWarningLOCKED(wrn)
	this.Unlock()
}

func (this *BaseRequest) addWarningLOCKED(wrn errors.Error) {
	// de-duplicate warnings
	if wrn.OnceOnly() {
		for _, w := range this.warnings {
			if wrn.Code() == w.Code() && wrn.Error() == w.Error() {
				w.Repeat()
				return
			}
		}
	}
	this.warnings = append(this.warnings, wrn)
	this.warningCount++
}

func (this *BaseRequest) AddMutationCount(i uint64) {
	atomic.AddUint64(&this.mutationCount, i)
}

func (this *BaseRequest) MutationCount() uint64 {
	return atomic.LoadUint64(&this.mutationCount)
}

func (this *BaseRequest) SetSortCount(i uint64) {
	atomic.StoreUint64(&this.sortCount, i)
}

func (this *BaseRequest) SortCount() uint64 {
	return atomic.LoadUint64(&this.sortCount)
}

func (this *BaseRequest) AddPhaseCount(p execution.Phases, c uint64) {
	atomic.AddUint64(&this.phaseStats[p].count, c)
}

func (this *BaseRequest) PhaseCount(p execution.Phases) uint64 {
	return uint64(this.phaseStats[p].count)
}

func (this *BaseRequest) FmtPhaseCounts() map[string]interface{} {
	var p map[string]interface{} = nil

	// Use simple iteration rather than a range clause to avoid a spurious data race report. MB-20692
	nr := len(this.phaseStats)
	for i := 0; i < nr; i++ {
		count := atomic.LoadUint64(&this.phaseStats[i].count)
		if count > 0 {
			if p == nil {
				p = make(map[string]interface{}, execution.PHASES)
			}
			p[execution.Phases(i).String()] = count
		}
	}
	return p
}

func (this *BaseRequest) AddPhaseOperator(p execution.Phases) {
	atomic.AddUint64(&this.phaseStats[p].operators, 1)
}

func (this *BaseRequest) PhaseOperator(p execution.Phases) uint64 {
	return uint64(this.phaseStats[p].operators)
}

func (this *BaseRequest) FmtPhaseOperators() map[string]interface{} {
	var p map[string]interface{} = nil

	// Use simple iteration rather than a range clause to avoid a spurious data race report. MB-20692
	nr := len(this.phaseStats)
	for i := 0; i < nr; i++ {
		operators := atomic.LoadUint64(&this.phaseStats[i].operators)
		if operators > 0 {
			if p == nil {
				p = make(map[string]interface{}, execution.PHASES)
			}
			p[execution.Phases(i).String()] = operators
		}
	}
	return p
}

func (this *BaseRequest) AddPhaseTime(p execution.Phases, duration time.Duration) {
	atomic.AddUint64(&this.phaseStats[p].duration, uint64(duration))
}

func (this *BaseRequest) FmtPhaseTimes(style util.DurationStyle) map[string]interface{} {
	var p map[string]interface{} = nil

	// Use simple iteration rather than a range clause to avoid a spurious data race report. MB-20692
	nr := len(this.phaseStats)
	for i := 0; i < nr; i++ {
		duration := atomic.LoadUint64(&this.phaseStats[i].duration)
		if duration > 0 {
			if p == nil {
				p = make(map[string]interface{}, execution.PHASES)
			}
			p[execution.Phases(i).String()] = util.FormatDuration(time.Duration(duration), style)
		}
	}
	return p
}

func (this *BaseRequest) RawPhaseTimes() map[string]interface{} {
	var p map[string]interface{} = nil

	nr := len(this.phaseStats)
	for i := 0; i < nr; i++ {
		duration := atomic.LoadUint64(&this.phaseStats[i].duration)
		if duration > 0 {
			if p == nil {
				p = make(map[string]interface{},
					execution.PHASES)
			}
			p[execution.Phases(i).String()] = time.Duration(duration)
		}
	}
	return p
}

func (this *BaseRequest) FmtOptimizerEstimates(op execution.Operator) map[string]interface{} {
	var p map[string]interface{} = nil

	if op != nil {
		planOp := op.PlanOp()
		if planOp != nil && planOp.Cost() > 0.0 && planOp.Cardinality() > 0.0 {
			p = make(map[string]interface{}, 2)
			p["cost"] = planOp.Cost()
			p["cardinality"] = planOp.Cardinality()
		}
	}

	return p
}

func (this *BaseRequest) AddTenantUnits(s tenant.Service, cu tenant.Unit) {
	tenant.AddUnit(&this.tenantUnits[s], cu)
}

func (this *BaseRequest) GetTenantUnits(s tenant.Service) tenant.Unit {
	return this.tenantUnits[s]
}

func (this *BaseRequest) TenantUnits() tenant.Services {
	return this.tenantUnits
}

func (this *BaseRequest) AddCpuTime(duration time.Duration) {
	atomic.AddUint64(&(this.cpuTime), uint64(duration))
}

func (this *BaseRequest) CpuTime() time.Duration {
	return time.Duration(this.cpuTime)
}

func (this *BaseRequest) AddIoTime(duration time.Duration) {
	atomic.AddUint64(&(this.ioTime), uint64(duration))
}

func (this *BaseRequest) IoTime() time.Duration {
	return time.Duration(this.ioTime)
}

func (this *BaseRequest) AddWaitTime(duration time.Duration) {
	atomic.AddUint64(&(this.waitTime), uint64(duration))
}

func (this *BaseRequest) WaitTime() time.Duration {
	return time.Duration(this.waitTime)
}

func (this *BaseRequest) TrackMemory(size uint64) {
	util.TestAndSetUint64(&this.usedMemory, size,
		func(old, new uint64) bool { return old < new }, 1)
}

func (this *BaseRequest) UsedMemory() uint64 {
	return uint64(this.usedMemory)
}

func (this *BaseRequest) SetTimings(o execution.Operator) {
	this.timings = o
}

func (this *BaseRequest) GetTimings() execution.Operator {
	return this.timings
}

func (this *BaseRequest) SetFmtTimings(t []byte) {
	this.fmtTimings = t
}

func (this *BaseRequest) GetFmtTimings() []byte {
	return this.fmtTimings
}

func (this *BaseRequest) SetFmtOptimizerEstimates(e map[string]interface{}) {
	this.fmtEstimates = e
}

func (this *BaseRequest) GetFmtOptimizerEstimates() map[string]interface{} {
	return this.fmtEstimates
}

func (this *BaseRequest) SetControls(c value.Tristate) {
	this.controls = c
}

func (this *BaseRequest) Controls() value.Tristate {
	return this.controls
}

func (this *BaseRequest) SetProfile(p Profile) {
	this.profile = p
}

func (this *BaseRequest) Profile() Profile {
	return this.profile
}

func (this *BaseRequest) SetIndexApiVersion(ver int) {
	// By default this.indexApiVersion is Server level. request level needs to be lessthan server level
	if ver < this.indexApiVersion {
		this.indexApiVersion = ver
	}
}

func (this *BaseRequest) IndexApiVersion() int {
	return this.indexApiVersion
}

func (this *BaseRequest) SetFeatureControls(controls uint64) {
	// By default this.featureControls is Server level. request level can only turn off server level
	this.featureControls = this.featureControls | controls
}

func (this *BaseRequest) FeatureControls() uint64 {
	return this.featureControls
}

func (this *BaseRequest) SetAutoPrepare(a value.Tristate) {
	this.autoPrepare = a
}

func (this *BaseRequest) AutoPrepare() value.Tristate {
	return this.autoPrepare
}

func (this *BaseRequest) SetAutoExecute(a value.Tristate) {
	this.autoExecute = a
}

func (this *BaseRequest) AutoExecute() value.Tristate {
	return this.autoExecute
}

func (this *BaseRequest) SetUseFts(a bool) {
	this.useFts = a
}

func (this *BaseRequest) UseFts() bool {
	return this.useFts && util.IsFeatureEnabled(this.featureControls, util.N1QL_FLEXINDEX)
}

func (this *BaseRequest) SetMemoryQuota(q uint64) {
	this.memoryQuota = q
}

func (this *BaseRequest) MemoryQuota() uint64 {
	return this.memoryQuota
}

func (this *BaseRequest) SetQueryContext(s string) {
	this.queryContext = s
}

func (this *BaseRequest) QueryContext() string {
	return this.queryContext
}

func (this *BaseRequest) UseCBO() bool {
	return this.useCBO && util.IsFeatureEnabled(this.featureControls, util.N1QL_CBO)
}

func (this *BaseRequest) SetUseCBO(useCBO bool) {
	// use-cbo can only be set at request level if it is not turned off in n1ql-feat-ctrl
	if util.IsFeatureEnabled(this.featureControls, util.N1QL_CBO) {
		this.useCBO = useCBO
	}
}

func (this *BaseRequest) UseReplica() value.Tristate {
	return this.useReplica
}

func (this *BaseRequest) SetUseReplica(useReplica value.Tristate) {
	this.useReplica = useReplica
}

func (this *BaseRequest) SetTxId(s string) {
	this.txId = s
}

func (this *BaseRequest) TxId() string {
	return this.txId
}

func (this *BaseRequest) SetTxImplicit(b bool) {
	this.txImplicit = b
}

func (this *BaseRequest) TxImplicit() bool {
	if this.txId == "" {
		return this.txImplicit
	}
	return false
}

func (this *BaseRequest) SetTxStmtNum(n int64) {
	this.txStmtNum = n
}

func (this *BaseRequest) TxStmtNum() int64 {
	return this.txStmtNum
}

func (this *BaseRequest) SetTxTimeout(d time.Duration) {
	if d > 0 {
		this.txTimeout = d
	}
}

func (this *BaseRequest) TxTimeout() time.Duration {
	return this.txTimeout
}

func (this *BaseRequest) SetTxData(b []byte) {
	this.txData = b
}

func (this *BaseRequest) TxData() []byte {
	return this.txData
}

func (this *BaseRequest) SetDurabilityLevel(l datastore.DurabilityLevel) {
	this.durabilityLevel = l
}

func (this *BaseRequest) DurabilityLevel() datastore.DurabilityLevel {
	return this.durabilityLevel
}

func (this *BaseRequest) SetDurabilityTimeout(d time.Duration) {
	this.durabilityTimeout = d
}

func (this *BaseRequest) DurabilityTimeout() time.Duration {
	return this.durabilityTimeout
}

func (this *BaseRequest) SetKvTimeout(d time.Duration) {
	this.kvTimeout = d
}

func (this *BaseRequest) KvTimeout() time.Duration {
	return this.kvTimeout
}

func (this *BaseRequest) SetAtrCollection(s string) {
	this.atrCollection = s
}

func (this *BaseRequest) AtrCollection() string {
	return this.atrCollection
}

func (this *BaseRequest) SetNumAtrs(n int) {
	this.numAtrs = n
}

func (this *BaseRequest) NumAtrs() int {
	return this.numAtrs
}

func (this *BaseRequest) SetPreserveExpiry(a bool) {
	this.preserveExpiry = a
}

func (this *BaseRequest) PreserveExpiry() bool {
	return this.preserveExpiry
}

func (this *BaseRequest) SetExecutionContext(ctx *execution.Context) {
	this.executionContext = ctx
}

func (this *BaseRequest) ExecutionContext() *execution.Context {
	return this.executionContext
}

func (this *BaseRequest) SetTracked() {
	this.tracked = true
}

func (this *BaseRequest) Tracked() bool {
	return this.tracked
}

func (this *BaseRequest) SetTenantCtx(ctx tenant.Context) {
	this.tenantCtx = ctx
}

func (this *BaseRequest) TenantCtx() tenant.Context {
	return this.tenantCtx
}

func (this *BaseRequest) ThrottleTime() time.Duration {
	return this.throttleTime
}

func (this *BaseRequest) SetThrottleTime(d time.Duration) {
	this.throttleTime = d
}

func (this *BaseRequest) Results() chan bool {
	return this.stopResult
}

func (this *BaseRequest) CloseResults() {
	sendStop(this.stopResult)
}

func (this *BaseRequest) Errors() []errors.Error {
	this.RLock()
	rv := make([]errors.Error, len(this.errors))
	copy(rv, this.errors)
	this.RUnlock()
	return rv
}

func (this *BaseRequest) Warnings() []errors.Error {
	this.RLock()
	rv := make([]errors.Error, len(this.warnings))
	copy(rv, this.warnings)
	this.RUnlock()
	return rv
}

func (this *BaseRequest) NotifyStop(o execution.Operator) {
	this.Lock()
	this.stopOperator = o
	this.Unlock()
}

func (this *BaseRequest) StopNotify() execution.Operator {
	this.RLock()
	rv := this.stopOperator
	this.RUnlock()
	return rv
}

func (this *BaseRequest) StopExecute() chan bool {
	return this.stopExecute
}

func (this *BaseRequest) Stop(state State) {
	this.SetState(state)
	this.Lock()
	if this.executionContext != nil {
		this.executionContext.Pause(false)
	}
	stopOperator := this.stopOperator

	// make sure that a stop can only be sent once (eg close OR timeout)
	this.stopOperator = nil
	this.Unlock()

	// guard against the root operator not being set (eg fatal error)
	if stopOperator != nil {
		// only one in between Stop() and Done() can happen at any one time
		this.stopGate.Wait()
		this.stopGate.Add(1)
		execution.OpStop(stopOperator)
		this.stopGate.Done()
	}
	sendStop(this.stopExecute)
}

// load control gate
func (this *BaseRequest) setSleep() {
	this.servicerGate.Add(1)
}
func (this *BaseRequest) sleep() {
	this.servicerGate.Wait()
}

func (this *BaseRequest) release() {
	this.servicerGate.Done()
}

// this logs the request if needed and takes any other action required to
// put this request to rest
func (this *BaseRequest) CompleteRequest(requestTime, serviceTime, transaction_time time.Duration,
	resultCount int, resultSize int, errorCount int, req *http.Request, server *Server, seqScanCount int64) {

	if this.timer != nil {
		this.timer.Stop()
		this.timer = nil
	}

	LogRequest(requestTime, serviceTime, transaction_time, resultCount, resultSize, errorCount, req, this, server, seqScanCount)

	// Request Profiling - signal that request has completed and
	// resources can be pooled / released as necessary
	if this.timings != nil {

		// only one in between Stop() and Done() can happen at any one time
		this.stopGate.Wait()
		this.stopGate.Add(1)

		// sending a stop is illegal after this point
		this.NotifyStop(nil)
		this.done()
		this.stopGate.Done()
		this.timings = nil
	}
}

// this function exists to make sure that if a panic occurs in the Done() machinery
// the servicer can still be released
func (this *BaseRequest) done() {
	defer func() {
		err := recover()
		if err != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			stmt := "<ud>" + this.RedactedStatement() + "</ud>"
			qc := "<ud>" + this.QueryContext() + "</ud>"
			logging.Severef("panic: %v ", err, this.ExecutionContext())
			logging.Severef("request text: %v", stmt, this.ExecutionContext())
			logging.Severef("query context: %v", qc, this.ExecutionContext())
			logging.Severef("stack: %v", s, this.ExecutionContext())
			os.Stderr.WriteString(s)
			os.Stderr.Sync()
			event.Report(event.CRASH, event.ERROR, "error", err, "request-id", this.Id().String(),
				"statement", event.UpTo(stmt, 500), "query_context", event.UpTo(qc, 250), "stack", event.CompactStack(s, 2000))

		}
	}()
	this.timings.Done()
}

func sendStop(ch chan bool) {
	select {
	case ch <- false:
	default:
	}
}

// For audit.Auditable interface.
func (this *BaseRequest) EventStatement() string {
	prep := this.Prepared()
	if prep != nil {
		return prep.Text()
	}
	return this.Statement()
}

// For audit.Auditable interface.
func (this *BaseRequest) EventErrorMessage() []errors.Error {
	return this.errors
}

// For audit.Auditable interface.
func (this *BaseRequest) EventQueryContext() string {
	return this.QueryContext()
}

// For audit.Auditable interface.
func (this *BaseRequest) EventTxId() string {
	return this.TxId()
}

// For audit.Auditable interface.
func (this *BaseRequest) PreparedId() string {
	prep := this.Prepared()
	if prep != nil {
		return prep.Name()
	}
	return ""
}

// For audit.Auditable interface.
func (this *BaseRequest) EventId() string {
	return this.Id().String()
}

// For audit.Auditable interface.
func (this *BaseRequest) EventType() string {
	t := this.Type()
	if t == "" && this.IsPrepare() {
		t = "PREPARE"
	}
	return t
}

// For audit.Auditable interface.
func (this *BaseRequest) EventUsers() []string {
	return datastore.CredsArray(this.credentials)
}

// For audit.Auditable interface.
func (this *BaseRequest) EventNamedArgs() map[string]interface{} {
	argsMap := this.RedactedNamedArgs()
	ret := make(map[string]interface{}, len(argsMap))
	for name, argValue := range argsMap {
		ret[name] = argValue.Actual()
	}
	return ret
}

// For audit.Auditable interface.
func (this *BaseRequest) EventPositionalArgs() []interface{} {
	args := this.RedactedPositionalArgs()
	ret := make([]interface{}, len(args))
	for i, v := range args {
		ret[i] = v.Actual()
	}
	return ret
}

// For audit.Auditable interface.
func (this *BaseRequest) IsAdHoc() bool {
	return this.Prepared() == nil
}

// For audit.Auditable interface.
func (this *BaseRequest) ClientContextId() string {
	return this.ClientID().String()
}

func (this *BaseRequest) SetSortProjection(on bool) {
	this.sortProjection = on
}

func (this *BaseRequest) SortProjection() bool {
	return this.sortProjection
}

// Add a list of errors
// If the number of errors exceeds the error limit , append a E_REQUEST_ERROR_LIMIT error to the error list
func (this *BaseRequest) SetErrors(errs errors.Errors) {
	this.Lock()

	for _, err := range errs {
		if err.Level() == errors.EXCEPTION {
			this.errors = append(this.errors, err)
			this.errorCount++
			this.Unlock()
			this.Stop(FATAL)
			return
		}

		if err.Level() == errors.WARNING {
			this.addWarningLOCKED(err)
		} else if this.errorLimit <= 0 || (this.errorCount+this.duplicateErrorCount) <= this.errorLimit {
			this.addErrorLOCKED(err)
		}
	}

	// Append a single E_REQUEST_ERROR_LIMIT error
	if this.errorLimit > 0 && ((this.errorCount + this.duplicateErrorCount) > this.errorLimit) {
		this.errors = append(this.errors, errors.NewErrorLimit(this.errorLimit, this.errorCount, this.duplicateErrorCount,
			this.MutationCount()))
		this.Unlock()
		this.Stop(FATAL)
	} else {
		this.Unlock()
	}
}

func (this *BaseRequest) DurationStyle() util.DurationStyle {
	return this.durationStyle
}

func (this *BaseRequest) SetDurationStyle(style util.DurationStyle) {
	this.durationStyle = style
}

func (this *BaseRequest) SessionMemory() uint64 {
	if this.executionContext != nil {
		return this.executionContext.SessionMemory()
	}
	return 0
}

func (this *BaseRequest) SetNatural(natural string) {
	this.natural = natural
}

func (this *BaseRequest) Natural() string {
	return this.natural
}

func (this *BaseRequest) SetNaturalCred(cred string) {
	this.naturalCred = cred
}

func (this *BaseRequest) NaturalCred() string {
	return this.naturalCred
}

func (this *BaseRequest) SetNaturalOrganizationId(orgId string) {
	this.naturalOrgId = orgId
}

func (this *BaseRequest) NaturalOrganizationId() string {
	return this.naturalOrgId
}

func (this *BaseRequest) SetNaturalContext(naturalContext string) {
	this.naturalContext = naturalContext
}

func (this *BaseRequest) NaturalContext() string {
	return this.naturalContext
}

func (this *BaseRequest) SetNaturalStatement(nlstmt algebra.Statement) {
	this.nlStatement = nlstmt
}

func (this *BaseRequest) NaturalStatement() algebra.Statement {
	return this.nlStatement
}

func (this *BaseRequest) SetNaturalShowOnly(show bool) {
	this.nlShowOnly = show
}

func (this *BaseRequest) NaturalShowOnly() bool {
	return this.nlShowOnly
}

func (this *BaseRequest) Format(durStyle util.DurationStyle, controls bool, prof bool, redact bool) map[string]interface{} {
	item := make(map[string]interface{}, 32)
	item["requestId"] = this.Id().String()
	item["requestTime"] = this.RequestTime().Format(util.DEFAULT_FORMAT)
	item["elapsedTime"] = util.FormatDuration(time.Since(this.RequestTime()), durStyle)
	if this.ServiceTime().IsZero() {
		item["executionTime"] = util.FormatDuration(0, durStyle)
	} else {
		item["executionTime"] = util.FormatDuration(time.Since(this.ServiceTime()), durStyle)
	}
	item["state"] = this.State().StateName()
	item["scanConsistency"] = this.ScanConsistency()
	item["n1qlFeatCtrl"] = this.FeatureControls()
	if cId := this.ClientID().String(); cId != "" {
		item["clientContextID"] = cId
	}
	if this.Statement() != "" {
		item["statement"] = this.RedactedStatement()
	}
	if this.Type() != "" {
		item["statementType"] = this.Type()
	}
	if this.QueryContext() != "" {
		item["queryContext"] = this.QueryContext()
	}
	if this.UseFts() {
		item["useFts"] = this.UseFts()
	}
	if this.UseCBO() {
		item["useCBO"] = this.UseCBO()
	}
	if this.UseReplica() == value.TRUE {
		item["useReplica"] = value.TristateToString(this.UseReplica())
	}
	if this.TxId() != "" {
		item["txid"] = this.TxId()
	}
	if !this.TransactionStartTime().IsZero() {
		item["transactionElapsedTime"] = util.FormatDuration(time.Since(this.TransactionStartTime()), durStyle)
		remTime := this.TxTimeout() - time.Since(this.TransactionStartTime())
		if remTime > 0 {
			item["transactionRemainingTime"] = util.FormatDuration(remTime, durStyle)
		}
	}
	if this.ThrottleTime() > time.Duration(0) {
		item["throttleTime"] = util.FormatDuration(this.ThrottleTime(), durStyle)
	}
	if this.CpuTime() > time.Duration(0) {
		item["cpuTime"] = util.FormatDuration(this.CpuTime(), durStyle)
	}
	if this.IoTime() > time.Duration(0) {
		item["ioTime"] = util.FormatDuration(this.IoTime(), durStyle)
	}
	if this.WaitTime() > time.Duration(0) {
		item["waitTime"] = util.FormatDuration(this.WaitTime(), durStyle)
	}
	p := this.FmtPhaseCounts()
	if p != nil {
		item["phaseCounts"] = p
	}
	p = this.FmtPhaseOperators()
	if p != nil {
		item["phaseOperators"] = p
	}
	p = this.FmtPhaseTimes(durStyle)
	if p != nil {
		item["phaseTimes"] = p
	}
	if usedMemory := this.UsedMemory(); usedMemory != 0 {
		item["usedMemory"] = usedMemory
	}
	if sessionMemory := this.SessionMemory(); sessionMemory != 0 {
		item["sessionMemory"] = sessionMemory
	}

	if p := this.Prepared(); p != nil {
		item["preparedName"] = p.Name()
		item["preparedText"] = p.Text()
	}
	if credsString := datastore.CredsString(this.Credentials()); credsString != "" {
		item["users"] = credsString
	}
	if remoteAddr := this.RemoteAddr(); remoteAddr != "" {
		item["remoteAddr"] = remoteAddr
	}
	if userAgent := this.UserAgent(); userAgent != "" {
		item["userAgent"] = userAgent
	}
	if memoryQuota := this.MemoryQuota(); memoryQuota != 0 {
		item["memoryQuota"] = memoryQuota
	}

	if prof {
		timings := this.GetTimings()
		if timings != nil {
			item["timings"] = value.ApplyDurationStyleToValue(durStyle, value.NewMarshalledValue(timings))
			p := this.FmtOptimizerEstimates(timings)
			if p != nil {
				item["optimizerEstimates"] = value.NewValue(p)
			}
		}
	}

	if controls {
		na := this.RedactedNamedArgs()
		if na != nil {
			item["namedArgs"] = util.InterfaceRedacted(na, redact)
		}
		pa := this.RedactedPositionalArgs()
		if pa != nil {
			item["positionalArgs"] = util.InterfaceRedacted(pa, redact)
		}
	}

	if n := this.Natural(); n != "" {
		item["naturalLanguagePrompt"] = util.Redacted(n, redact)
		if nt := this.NaturalTime(); nt != 0 {
			item["naturalLanguageProcessingTime"] = util.FormatDuration(nt, durStyle)
		}
	}

	return item
}

func (this *BaseRequest) NaturalTime() time.Duration {
	return time.Duration(this.phaseStats[execution.NLPARSE].duration + this.phaseStats[execution.GETJWT].duration +
		this.phaseStats[execution.INFERSCHEMA].duration + this.phaseStats[execution.CHATCOMPLETIONSREQ].duration +
		this.phaseStats[execution.NLWAIT].duration)
}
