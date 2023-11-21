//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/logging/event"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Phases int

const (
	// Execution layer
	AUTHORIZE = Phases(iota)
	FETCH
	INDEX_SCAN
	PRIMARY_SCAN
	JOIN
	INDEX_JOIN
	NL_JOIN
	HASH_JOIN
	NEST
	INDEX_NEST
	NL_NEST
	HASH_NEST
	COUNT
	INDEX_COUNT
	FILTER
	SORT
	PROJECT
	STREAM
	INSERT
	DELETE
	UPDATE
	UPSERT
	MERGE
	INFER
	FTS_SEARCH
	UPDATE_STAT
	INDEX_SCAN_GSI
	PRIMARY_SCAN_GSI
	INDEX_SCAN_FTS
	PRIMARY_SCAN_FTS

	// Expression layer
	ADVISOR

	// Server layer
	INSTANTIATE
	PARSE
	PLAN
	REPREPARE
	RUN
	PHASES // Sizer
)

func (phase Phases) String() string {
	return _PHASE_NAMES[phase]
}

var _PHASE_NAMES = []string{
	AUTHORIZE:        "authorize",
	FETCH:            "fetch",
	INDEX_SCAN:       "indexScan",
	PRIMARY_SCAN:     "primaryScan",
	JOIN:             "join",
	INDEX_JOIN:       "indexJoin",
	NL_JOIN:          "nestedLoopJoin",
	HASH_JOIN:        "hashJoin",
	NEST:             "nest",
	INDEX_NEST:       "indexNest",
	NL_NEST:          "nestedLoopNest",
	HASH_NEST:        "hashNest",
	COUNT:            "count",
	INDEX_COUNT:      "indexCount",
	SORT:             "sort",
	FILTER:           "filter",
	PROJECT:          "project",
	STREAM:           "stream",
	INSERT:           "insert",
	DELETE:           "delete",
	UPDATE:           "update",
	UPSERT:           "upsert",
	MERGE:            "merge",
	INFER:            "inferKeySpace",
	FTS_SEARCH:       "ftsSearch",
	UPDATE_STAT:      "updateStatistics",
	INDEX_SCAN_GSI:   "indexScan.GSI",
	PRIMARY_SCAN_GSI: "primaryScan.GSI",
	INDEX_SCAN_FTS:   "indexScan.FTS",
	PRIMARY_SCAN_FTS: "primaryScan.FTS",

	ADVISOR: "advisor",

	INSTANTIATE: "instantiate",
	PARSE:       "parse",
	PLAN:        "plan",
	REPREPARE:   "reprepare",
	RUN:         "run",
}

const _PHASE_UPDATE_COUNT uint64 = 100

type Output interface {
	SetUp()                                // Any action necessary before processing results
	Result(item value.AnnotatedValue) bool // Process individual items
	CloseResults()                         // Signal that results are through
	Abort(err errors.Error)
	Fatal(err errors.Error)
	Error(err errors.Error)
	Warning(wrn errors.Error)
	AddMutationCount(uint64)
	MutationCount() uint64
	SortCount() uint64
	SetSortCount(i uint64)
	AddPhaseOperator(p Phases)
	AddPhaseCount(p Phases, c uint64)
	FmtPhaseCounts() map[string]interface{}
	FmtPhaseOperators() map[string]interface{}
	AddPhaseTime(phase Phases, duration time.Duration)
	FmtPhaseTimes() map[string]interface{}
	FmtOptimizerEstimates(op Operator) map[string]interface{}
	TrackMemory(size uint64)
	SetTransactionStartTime(t time.Time)
}

// context flags
const (
	CONTEXT_IS_ADVISOR = 1 << iota // Advisor() function
)

// Per operator context
type opContext struct {
	base *base // base is named to make sure that it doesn't inherit anything from the base type
	*Context
}

func (this *opContext) Park(stop func(bool)) {
	this.base.setExternalStop(stop)
	this.base.switchPhase(_CHANTIME)
}
func (this *opContext) Resume() {
	this.base.setExternalStop(nil)
	this.base.switchPhase(_EXECTIME)
}

func (this *opContext) IsActive() bool {
	return this.base.opState == _RUNNING
}

func (this *opContext) Copy() *opContext {
	return &opContext{this.base, this.Context.Copy()}
}

func (this *opContext) NewQueryContext(queryContext string, readonly bool) interface{} {
	return &opContext{this.base, this.Context.NewQueryContext(queryContext, readonly).(*Context)}
}

func NewOpContext(context *Context) *opContext {
	b := &base{}
	newBase(b, context)
	return &opContext{b, context}
}

// Per request context

type Context struct {
	inUseMemory         uint64
	requestId           string
	datastore           datastore.Datastore
	systemstore         datastore.Systemstore
	namespace           string
	indexApiVersion     int
	featureControls     uint64
	queryContext        string
	useFts              bool
	useCBO              bool
	useReplica          bool
	optimizer           planner.Optimizer
	readonly            bool
	maxParallelism      int
	scanCap             int64
	pipelineCap         int64
	pipelineBatch       int
	isPrepared          bool
	reqDeadline         time.Time
	now                 time.Time
	namedArgs           map[string]value.Value
	positionalArgs      value.Values
	credentials         *auth.Credentials
	consistency         datastore.ScanConsistency
	originalConsistency datastore.ScanConsistency
	scanVectorSource    timestamp.ScanVectorSource
	output              Output
	prepared            *plan.Prepared
	subExecTrees        *subqueryArrayMap
	subqueryPlans       *algebra.SubqueryPlans
	subresults          *subqueryMap
	httpRequest         *http.Request
	authenticatedUsers  auth.AuthenticatedUsers
	mutex               sync.RWMutex
	allowlist           map[string]interface{}
	inlistHashMap       map[*expression.In]*expression.InlistHash
	inlistHashLock      sync.RWMutex
	memoryQuota         uint64
	reqTimeout          time.Duration
	deltaKeyspaces      map[string]bool
	durabilityLevel     datastore.DurabilityLevel
	durabilityTimeout   time.Duration
	txContext           *transactions.TranContext
	txTimeout           time.Duration
	txImplicit          bool
	txData              []byte
	txDataVal           value.Value
	atrCollection       string
	numAtrs             int
	kvTimeout           time.Duration
	preserveExpiry      bool
	flags               uint32
	recursionCount      int32
	result              func(context *Context, item value.AnnotatedValue) bool
	likeRegexMap        map[*expression.Like]*expression.LikeRegex
	udfValueMap         *sync.Map
	udfHandleMap        map[*executionHandle]bool
	tracked             bool
	// Key: unique identifier of the Inline UDF, Value: Info on the function body of Inline UDFs executed in the query
	inlineUdfEntries map[string]*inlineUdfEntry

	// queryMutex is a pointer and is meant to be shared when we share certain fields
	// when calling NewQueryContext(). Since the fields are shared the mutex that protect
	// them also needs to be shared
	queryMutex *sync.RWMutex
}

func NewContext(requestId string, datastore datastore.Datastore, systemstore datastore.Systemstore,
	namespace string, readonly bool, maxParallelism int, scanCap, pipelineCap int64,
	pipelineBatch int, namedArgs map[string]value.Value, positionalArgs value.Values,
	credentials *auth.Credentials, consistency datastore.ScanConsistency,
	scanVectorSource timestamp.ScanVectorSource, output Output,
	prepared *plan.Prepared, indexApiVersion int, featureControls uint64, queryContext string,
	useFts, useCBO bool, optimizer planner.Optimizer, kvTimeout, reqTimeout time.Duration) *Context {

	rv := &Context{
		requestId:        requestId,
		datastore:        datastore,
		systemstore:      systemstore,
		namespace:        namespace,
		readonly:         readonly,
		maxParallelism:   maxParallelism,
		scanCap:          scanCap,
		pipelineCap:      pipelineCap,
		pipelineBatch:    pipelineBatch,
		now:              time.Now(),
		namedArgs:        namedArgs,
		positionalArgs:   positionalArgs,
		credentials:      credentials,
		consistency:      consistency,
		scanVectorSource: scanVectorSource,
		output:           output,
		subresults:       nil,
		prepared:         prepared,
		indexApiVersion:  indexApiVersion,
		featureControls:  featureControls,
		useReplica:       false,
		queryContext:     queryContext,
		useFts:           useFts,
		useCBO:           useCBO,
		optimizer:        optimizer,
		inlistHashMap:    nil,
		kvTimeout:        kvTimeout,
		result:           setup,
		likeRegexMap:     nil,
		reqTimeout:       reqTimeout,
		udfValueMap:      &sync.Map{},
		subqueryPlans:    algebra.NewSubqueryPlans(),
		subExecTrees:     newSubqueryArrayMap(),
		inlineUdfEntries: make(map[string]*inlineUdfEntry),
		queryMutex:       &sync.RWMutex{},
	}

	if rv.maxParallelism <= 0 || rv.maxParallelism > util.NumCPU() {
		rv.maxParallelism = util.NumCPU()
	}

	return rv
}

func (this *Context) Copy() *Context {
	rv := &Context{
		requestId:           this.requestId,
		datastore:           this.datastore,
		systemstore:         this.systemstore,
		namespace:           this.namespace,
		readonly:            this.readonly,
		maxParallelism:      this.maxParallelism,
		scanCap:             this.scanCap,
		pipelineCap:         this.pipelineCap,
		pipelineBatch:       this.pipelineBatch,
		now:                 this.now,
		credentials:         this.credentials,
		consistency:         this.consistency,
		originalConsistency: this.originalConsistency,
		scanVectorSource:    this.scanVectorSource,
		output:              this.output,
		result:              this.result,
		httpRequest:         this.httpRequest,
		indexApiVersion:     this.indexApiVersion,
		featureControls:     this.featureControls,
		useReplica:          this.useReplica,
		queryContext:        this.queryContext,
		useFts:              this.useFts,
		useCBO:              this.useCBO,
		deltaKeyspaces:      this.deltaKeyspaces,
		txTimeout:           this.txTimeout,
		txImplicit:          this.txImplicit,
		txContext:           this.txContext,
		txData:              this.txData,
		txDataVal:           this.txDataVal,
		kvTimeout:           this.kvTimeout,
		atrCollection:       this.atrCollection,
		numAtrs:             this.numAtrs,
		preserveExpiry:      this.preserveExpiry,
		flags:               this.flags,
		reqTimeout:          this.reqTimeout,
		allowlist:           this.allowlist,
		udfValueMap:         this.udfValueMap,
		recursionCount:      this.recursionCount,
		// suqbery/udf information should share the same info
		inlineUdfEntries: this.inlineUdfEntries,
		subExecTrees:     this.subExecTrees,
		subqueryPlans:    this.subqueryPlans,
		queryMutex:       this.queryMutex,
	}

	if this.optimizer != nil {
		rv.optimizer = this.optimizer.Copy()
	}

	rv.SetDurability(this.DurabilityLevel(), this.DurabilityTimeout())

	return rv
}

func (this *Context) NewQueryContext(queryContext string, readonly bool) interface{} {
	rv := this.Copy()
	rv.queryContext = queryContext
	rv.readonly = readonly
	return rv
}

func (this *Context) IsActive() bool {
	return true
}

func (this *Context) QueryContext() string {
	return this.queryContext
}

func (this *Context) RequestId() string {
	return this.requestId
}

func (this *Context) Type() string {
	if this.prepared != nil {
		return this.prepared.Type()
	}
	return ""
}

func (this *Context) Datastore() datastore.Datastore {
	return this.datastore
}

func (this *Context) SetNamedArgs(namedArgs map[string]value.Value) {
	this.namedArgs = namedArgs
}

func (this *Context) SetPositionalArgs(positionalArgs value.Values) {
	this.positionalArgs = positionalArgs
}

func (this *Context) SetPrepared(prepared *plan.Prepared) {
	this.prepared = prepared
}

func (this *Context) SetAllowlist(val map[string]interface{}) {
	this.allowlist = val
}

func (this *Context) GetAllowlist() map[string]interface{} {
	return this.allowlist
}

func (this *Context) Optimizer() planner.Optimizer {
	return this.optimizer
}

func (this *Context) DatastoreVersion() string {
	return this.datastore.Info().Version()
}

func (this *Context) Systemstore() datastore.Systemstore {
	return this.systemstore
}

func (this *Context) Namespace() string {
	return this.namespace
}

func (this *Context) Readonly() bool {
	return this.readonly
}

func (this *Context) MaxParallelism() int {
	return this.maxParallelism
}

func (this *Context) Now() time.Time {
	return this.now
}

func (this *Context) NamedArg(name string) (value.Value, bool) {
	val, ok := this.namedArgs[name]
	return val, ok
}

// The position is 1-based (i.e. 1 is the first position)
func (this *Context) PositionalArg(position int) (value.Value, bool) {
	position--

	if position >= 0 && position < len(this.positionalArgs) {
		return this.positionalArgs[position], true
	} else {
		return nil, false
	}
}

func (this *Context) GetTimeout() time.Duration {
	return this.reqTimeout
}

func (this *Context) GetTxContext() interface{} {
	return this.txContext
}

func (this *Context) TxContext() *transactions.TranContext {
	return this.txContext
}

func (this *Context) TxDataVal() value.Value {
	return this.txDataVal
}

func (this *Context) SetTxContext(tc interface{}) {
	this.txContext, _ = tc.(*transactions.TranContext)
}

func (this *Context) AdjustTimeout(timeout time.Duration, stmtType string, isPrepare bool) time.Duration {

	if this.txContext != nil {
		if !isPrepare && (stmtType == "COMMIT" || stmtType == "ROLLBACK") {
			// Don't start timer for the commit and rollback
			return 0
		}
		timeout = this.txContext.TxTimeRemaining()
		// At least keep 10ms so that executor can lanunch before timer kicks in.
		if timeout <= 10*time.Millisecond {
			timeout = 10 * time.Millisecond
		}
	}

	return timeout
}

func (this *Context) Credentials() *auth.Credentials {
	return this.credentials
}

func (this *Context) UrlCredentials(urlS string) *auth.Credentials {
	// For the cases where the request doesnt have credentials but uses an auth
	// token or x509 certs, we need to derive the credentials to be able to query
	// the fts index.
	if urlS == "" {
		dUrl, _ := url.Parse(this.DatastoreURL())
		urlS = dUrl.Hostname() + ":" + dUrl.Port()
	}

	authenticator := cbauth.Default
	u, p, _ := authenticator.GetHTTPServiceAuth(urlS)
	return &auth.Credentials{map[string]string{u: p}, nil, nil, nil}
}

func (this *Context) UseReplica() bool {
	return this.useReplica
}

func (this *Context) SetUseReplica(r bool) {
	this.useReplica = r
}

func (this *Context) ScanConsistency() datastore.ScanConsistency {
	return this.consistency
}

func (this *Context) SetScanConsistency(consistency, originalConsistency datastore.ScanConsistency) {
	this.consistency = consistency
	this.originalConsistency = originalConsistency
}

func (this *Context) ScanVectorSource() timestamp.ScanVectorSource {
	return this.scanVectorSource
}

// Return []string rather than datastore.AuthenticatedUsers to avoid a circular dependency
// in /expression
func (this *Context) AuthenticatedUsers() []string {
	return this.authenticatedUsers
}

func (this *Context) GetScanCap() int64 {
	if this.scanCap > 0 {
		return this.scanCap
	} else {
		return datastore.GetScanCap()
	}
}

func (this *Context) ScanCap() int64 {
	return this.scanCap
}

func (this *Context) SetScanCap(scanCap int64) {
	this.scanCap = scanCap
}

func (this *Context) GetReqDeadline() time.Time {
	return this.reqDeadline
}

func (this *Context) SetReqDeadline(reqDeadline time.Time) {
	this.reqDeadline = reqDeadline
}

func (this *Context) GetMemoryQuota() uint64 {
	return this.memoryQuota
}

func (this *Context) SetMemoryQuota(memoryQuota uint64) {
	this.memoryQuota = memoryQuota * 1024 * 1024
}

func (this *Context) GetPipelineCap() int64 {
	if this.pipelineCap > 0 {
		return this.pipelineCap
	} else {
		return GetPipelineCap()
	}
}

func (this *Context) PipelineCap() int64 {
	return this.pipelineCap
}

func (this *Context) SetPipelineCap(pipelineCap int64) {
	this.pipelineCap = pipelineCap
}

func (this *Context) GetPipelineBatch() int {
	if this.pipelineBatch > 0 {
		return this.pipelineBatch
	} else {
		return PipelineBatchSize()
	}
}

func (this *Context) PipelineBatch() int {
	return this.pipelineBatch
}

func (this *Context) SetPipelineBatch(pipelineBatch int) {
	this.pipelineBatch = pipelineBatch
}

func (this *Context) IsPrepared() bool {
	return this.isPrepared
}

func (this *Context) SetIsPrepared(isPrepared bool) {
	this.isPrepared = isPrepared
}

func (this *Context) AddMutationCount(i uint64) {
	this.output.AddMutationCount(i)
}

func (this *Context) MutationCount() uint64 {
	return this.output.MutationCount()
}

func (this *Context) SetSortCount(i uint64) {
	this.output.SetSortCount(i)
}

func (this *Context) SortCount() uint64 {
	return this.output.SortCount()
}

func (this *Context) AddPhaseOperator(p Phases) {
	this.output.AddPhaseOperator(p)
}

func (this *Context) AddPhaseCount(p Phases, c uint64) {
	this.output.AddPhaseCount(p, c)
}

func (this *Context) AddPhaseTime(phase Phases, duration time.Duration) {
	this.output.AddPhaseTime(phase, duration)
}

func (this *Context) AddPhaseOperatorWithAgg(p Phases) {
	this.output.AddPhaseOperator(p)
	if a, ok := phaseAggs[p]; ok {
		this.output.AddPhaseOperator(a)
	}
}
func (this *Context) AddPhaseCountWithAgg(p Phases, c uint64) {
	this.output.AddPhaseCount(p, c)
	if a, ok := phaseAggs[p]; ok {
		this.output.AddPhaseCount(a, c)
	}
}
func (this *Context) AddPhaseTimeWithAgg(p Phases, duration time.Duration) {
	this.output.AddPhaseTime(p, duration)
	if a, ok := phaseAggs[p]; ok {
		this.output.AddPhaseTime(a, duration)
	}
}

func setup(context *Context, item value.AnnotatedValue) bool {
	context.output.SetUp()
	context.result = result
	return context.output.Result(item)
}

func result(context *Context, item value.AnnotatedValue) bool {
	return context.output.Result(item)
}

func (this *Context) Result(item value.AnnotatedValue) bool {
	return this.result(this, item)
}

func (this *Context) CloseResults() {
	this.output.CloseResults()
}

func (this *Context) RecursionCount() int {
	return int(this.recursionCount)
}

func (this *Context) IncRecursionCount(inc int) int {
	return int(atomic.AddInt32(&this.recursionCount, int32(inc)))
}

type eventError struct {
	t event.EventType
	l event.EventLevel
}

var eventErrors = map[errors.ErrorCode]eventError{
	errors.E_MEMORY_QUOTA_EXCEEDED: {event.QUOTA_EXCEEDED, event.INFO},
}

func (this *Context) Error(err errors.Error) {
	if evt, ok := eventErrors[err.Code()]; ok {
		event.Report(evt.t, evt.l, "request-id", this.RequestId())
	}
	this.output.Error(err)
}

func (this *Context) Errors(errs errors.Errors) {
	for _, err := range errs {
		this.output.Error(err)
	}
}

func (this *Context) Abort(err errors.Error) {
	this.output.Abort(err)
}

func (this *Context) Fatal(err errors.Error) {
	this.output.Fatal(err)
}

func (this *Context) Warning(wrn errors.Error) {
	this.output.Warning(wrn)
}

// memory quota

func (this *Context) UseRequestQuota() bool {
	return this.memoryQuota > 0
}

func (this *Context) ProducerThrottleQuota() uint64 {
	return this.memoryQuota / 10
}

func (this *Context) TrackValueSize(size uint64) bool {
	sz := atomic.AddUint64(&this.inUseMemory, size)
	this.output.TrackMemory(sz)
	return sz > this.memoryQuota
}

func (this *Context) ReleaseValueSize(size uint64) {
	atomic.AddUint64(&this.inUseMemory, ^(size - 1))
}

func (this *Context) CurrentQuotaUsage() float64 {
	if this.memoryQuota == 0 {
		return 0.0
	}
	sz := atomic.LoadUint64(&this.inUseMemory)
	return float64(sz) / float64(this.memoryQuota)
}

// UDF memory storage

func (this *Context) StoreValue(key string, val interface{}) {
	this.udfValueMap.Store(key, val)
}

func (this *Context) RetrieveValue(key string) interface{} {
	res, _ := this.udfValueMap.Load(key)
	return res
}

func (this *Context) ReleaseValue(key string) {
	this.udfValueMap.Delete(key)
}

func (this *Context) SetDeltaKeyspaces(d map[string]bool) {
	this.deltaKeyspaces = d
}

func (this *Context) DeltaKeyspaces() map[string]bool {
	return this.deltaKeyspaces
}

func (this *Context) SetDurability(l datastore.DurabilityLevel, d time.Duration) {
	this.durabilityLevel = l
	this.durabilityTimeout = d
}

func (this *Context) DurabilityLevel() datastore.DurabilityLevel {
	return this.durabilityLevel
}

func (this *Context) DurabilityTimeout() time.Duration {
	return this.durabilityTimeout
}

func (this *Context) KvTimeout() time.Duration {
	return this.kvTimeout
}

func (this *Context) SetPreserveExpiry(preserve bool) {
	this.preserveExpiry = preserve
}

func (this *Context) PreserveExpiry() bool {
	return this.preserveExpiry
}

func (this *Context) ResetTxContext() {
	if this.txContext != nil {
		this.txContext = nil
	}
}

func (this *Context) SetTxTimeout(txTimeout time.Duration) {
	this.txTimeout = txTimeout
}

func (this *Context) SetTransactionInfo(txId string, txStmtNum int64) (err errors.Error) {
	txContext := transactions.GetTransContext(txId)
	if txContext == nil {
		return errors.NewTransactionContextError(fmt.Errorf("transaction (%s) is not present", txId))
	} else if err := txContext.TxValid(); err != nil {
		return err
	}

	this.txTimeout = txContext.TxTimeout()
	if this.originalConsistency == datastore.NOT_SET {
		this.consistency = txContext.TxScanConsistency()
	} else if this.originalConsistency == datastore.AT_PLUS {
		this.consistency = datastore.SCAN_PLUS
	}
	this.SetDurability(txContext.TxDurabilityLevel(), txContext.TxDurabilityTimeout())
	this.atrCollection = txContext.AtrCollection()
	this.numAtrs = txContext.NumAtrs()

	this.txImplicit = false
	if txStmtNum > 0 {
		lastStmtNum := txContext.TxLastStmtNum()
		if lastStmtNum >= txStmtNum {
			return errors.NewTranStatementOutOfOrderError(lastStmtNum, txStmtNum)
		}
		txContext.SetTxLastStmtNum(txStmtNum)
	}
	this.txContext = txContext
	return nil
}

func (this *Context) SetTransactionContext(stmtType string, txImplicit bool, rTxTimeout, sTxTimeout time.Duration,
	atrCollection string, numAtrs int, txData []byte) (err errors.Error) {

	if this.txContext != nil || stmtType == "START_TRANSACTION" || (txImplicit && stmtType != "EXECUTE_FUNCTION") {
		this.txData = txData
		if len(txData) > 0 {
			this.txDataVal = value.NewValue(txData)
		}

		if this.txContext == nil {
			// start transaction or implicit transaction
			if sTxTimeout > 0 && sTxTimeout < rTxTimeout {
				rTxTimeout = sTxTimeout
			}
			this.txTimeout = rTxTimeout
			this.atrCollection = atrCollection
			this.numAtrs = numAtrs

			if stmtType != "START_TRANSACTION" {
				// start implicit transaction
				this.txImplicit = txImplicit
				txId, _, err := this.ExecuteTranStatement("START", !txImplicit)
				if err != nil {
					return err
				}
				this.txContext = transactions.GetTransContext(txId)
				if this.txContext == nil {
					return errors.NewTransactionContextError(fmt.Errorf("transaction (%s) is not present", txId))
				}
				if this.txImplicit {
					this.SetDeltaKeyspaces(make(map[string]bool, 1))
				}
			}
		} else {
			switch stmtType {
			case "START_TRANSACTION", "COMMIT", "ROLLBACK":
			case "ROLLBACK_SAVEPOINT", "SAVEPOINT", "SET_TRANSACTION_ISOLATION":
			default:
				// setup atomicity
				_, dks, err := this.ExecuteTranStatement("START", true)
				if err != nil {
					return err
				}
				this.SetDeltaKeyspaces(dks)
			}
		}
	} else if stmtType == "EXECUTE_FUNCTION" {

		// set up transaction timeout in case the function starts a transaction
		if sTxTimeout > 0 && sTxTimeout < rTxTimeout {
			rTxTimeout = sTxTimeout
		}
		this.txTimeout = rTxTimeout
	}
	return nil
}

func (this *Context) SetAtrCollection(atrCollection string, numAtrs int) {
	this.atrCollection = atrCollection
	this.numAtrs = numAtrs
}

func (this *Context) AtrCollection() (string, int) {
	return this.atrCollection, this.numAtrs
}

func (this *Context) TxExpired() bool {
	return this.txContext != nil && this.txContext.TxExpired()
}

// Generate given Select subquery plan. At any given time one plan generation allowed. One per algebra Select.

func (this *opContext) SubqueryPlan(node *algebra.Select, mutex *sync.RWMutex, deltaKeyspaces map[string]bool,
	namedArgs map[string]value.Value, positionalArgs value.Values,
	lock bool) (interface{}, interface{}, error) {
	var prepContext planner.PrepareContext
	var optimizer planner.Optimizer
	var err1 errors.Error

	if mutex != nil && lock {
		mutex.Lock()
	}
	if this.optimizer != nil {
		optimizer = this.optimizer.Copy()
	}
	planner.NewPrepareContext(&prepContext, this.requestId, this.queryContext, namedArgs, positionalArgs,
		this.indexApiVersion, this.featureControls, this.useFts, this.useCBO, optimizer,
		deltaKeyspaces, this)
	qp, subplanIsks, err := planner.Build(node, this.datastore, this.systemstore, this.namespace,
		true, false, &prepContext)
	if err != nil {
		err1 = errors.NewSubqueryBuildError(err)
		this.Error(err1)
	}
	if mutex != nil && lock {
		mutex.Unlock()
	}
	return qp, subplanIsks, err1
}

// Generate all subqueries (even nested) plans. At any given time one plan generation allowed. One per algebra Select.
// At present this is used in Inline UDF.

func (this *opContext) SubqueryPlans(expr expression.Expression, subqPlans *algebra.SubqueryPlans,
	deltaKeyspaces map[string]bool, namedArgs map[string]value.Value, positionalArgs value.Values, lock bool) (bool, error) {
	subqueries, err := expression.ListSubqueries(expression.Expressions{expr}, true)
	if err != nil {
		err := errors.NewSubqueryBuildError(err)
		this.Error(err)
		return false, err
	}
	if len(subqueries) == 0 {
		subqPlans.Set(nil, expr, nil, nil, lock)
		return false, nil
	}

	mutex := subqPlans.GetMutex()

	for _, sq := range subqueries {
		if subq, ok := sq.(*algebra.Subquery); ok {
			query := subq.Select()
			plan, isk, err := this.SubqueryPlan(query, mutex, deltaKeyspaces, namedArgs, positionalArgs, false)
			if err != nil {
				return false, err
			}
			subqPlans.Set(query, expr, plan, isk, lock)
		}
	}
	return true, nil
}

/*
Setup Inline UDF plans.
generate   -- true. Plans are generated and stored in UDF. Also Dummy prepared created so that Metadata check can be done.
trans      -- true.  plan is part of transaction and delta table involved. Generate plan again and store in the context Subquery plans.
Otherwise     copy Inline UDF subquery plans by reference into local context so that same plan will be re-used repeated execution.
lock       -- indicate second argument require lock
*/
func (this *opContext) SetupSubqueryPlans(expr expression.Expression, subqPlans *algebra.SubqueryPlans, lock,
	generate, trans bool) (err error) {
	if generate {
		var hasSubquery bool
		hasSubquery, err = this.SubqueryPlans(expr, subqPlans, nil, nil, nil, lock)
		if err == nil && hasSubquery {
			subqPlans.SetPrepared(plan.NewPrepared(nil, nil, nil, nil), lock)
		}
	} else if trans {
		// last argument will be true because second argument is from context
		_, err = this.SubqueryPlans(expr, this.GetSubqueryPlans(true), this.deltaKeyspaces, this.namedArgs, this.positionalArgs, true)
	} else {
		// lock is meant source (subqPlans). Destination this.... always do lock
		subqPlans.Copy(this.GetSubqueryPlans(true), lock)
	}
	return err
}

/*
Inline UDFs.
Verify Metadata check (if any addition/drop index, scope/collections may trigger auto regenrate plan before first execution.
Also decides paln is part of transaction and delta tabe involved.
*/
func (this *opContext) VerifySubqueryPlans(expr expression.Expression, subqPlans *algebra.SubqueryPlans, lock bool) (bool, bool) {
	var prepared *plan.Prepared
	prep := subqPlans.GetPrepared(lock)
	if prep != nil {
		prepared, _ = prep.(*plan.Prepared)
		if prepared != nil && !prepared.MetadataCheck() {
			return false, false
		}
	}

	verifyF := func(key *algebra.Select, options uint32, splan, isk interface{}) (bool, bool) {
		var good, local bool
		if qp, ok := splan.(*plan.QueryPlan); ok {
			good = qp.Verify(prepared)
		}

		if good && isk != nil {
			for ks, _ := range isk.(map[string]bool) {
				if _, ok := this.deltaKeyspaces[ks]; ok {
					local = true
					break
				}
			}
		}
		return good, local
	}

	return subqPlans.ForEach(expr, uint32(0), lock, verifyF)
}

type inlineUdfEntry struct {
	expr     expression.Expression // UDF expression
	varNames []string              // argument names
}

/*
  Inline UDFs.
  First execution in the statement gets upto expression and store as part of the context and resuse.
  Any change UDF will not be reused. This gives consistent for each UDF execution.
  If any drop of collection/index will raise error and new statement will detect those.
*/

func (this *Context) GetInlineUdf(udf string) (expression.Expression, []string, bool) {
	this.queryMutex.RLock()
	e, ok := this.inlineUdfEntries[udf]
	this.queryMutex.RUnlock()
	if ok {
		return e.expr, e.varNames, ok
	} else {
		return nil, nil, false
	}

}

func (this *Context) SetInlineUdf(udf string, expr expression.Expression, varNames []string) {
	this.queryMutex.Lock()
	this.inlineUdfEntries[udf] = &inlineUdfEntry{expr, varNames}
	this.queryMutex.Unlock()
}

// subquery evaluation

func (this *opContext) EvaluateSubquery(query *algebra.Select, parent value.Value) (value.Value, error) {

	var subplan, subplanIsks interface{}
	var err error
	planFound := false

	useCache := useSubqCachedResult(query)
	if useCache {
		subresults := this.getSubresults()
		subresult, ok := subresults.get(query)
		if ok {
			return subresult.(value.Value), nil
		}
	}

	subplans := this.GetSubqueryPlans(true)
	_, subplan, subplanIsks, planFound = subplans.Get(query, true)

	if !planFound && this.IsPrepared() {
		// MB-34749 make subquery plans a property of the prepared statement
		psubplans := this.prepared.GetSubqueryPlans(true)
		_, subplan, subplanIsks, planFound = psubplans.Get(query, true)
		if !planFound {
			mutex := psubplans.GetMutex()
			mutex.Lock()
			// check again, just in case somebody has done it while we were waiting
			_, subplan, subplanIsks, planFound = psubplans.Get(query, false)
			if !planFound {
				subplan, subplanIsks, err = this.SubqueryPlan(query, mutex, nil, nil, nil, false)
				if err != nil {
					mutex.Unlock()
					return nil, err
				}
				planFound = true
				psubplans.Set(query, nil, subplan, subplanIsks, false)
			}
			mutex.Unlock()
		}
		if subplanIsks != nil {
			for ks, _ := range subplanIsks.(map[string]bool) {
				if _, ok := this.deltaKeyspaces[ks]; ok {
					planFound = false
					break
				}
			}
		}
	}

	if !planFound {
		// adhoc + transaction plans. mutex needed to handle max_parallelism
		_, subplan, subplanIsks, planFound = subplans.Get(query, true)
		if !planFound {
			mutex := subplans.GetMutex()
			mutex.Lock()
			// check again, just in case somebody has done it while we were waiting
			_, subplan, subplanIsks, planFound = subplans.Get(query, false)
			if !planFound {
				subplan, subplanIsks, err = this.SubqueryPlan(query, mutex, this.deltaKeyspaces,
					this.namedArgs, this.positionalArgs, false)
				if err != nil {
					mutex.Unlock()
					return nil, err
				}
				subplans.Set(query, nil, subplan, subplanIsks, false)
			}
			mutex.Unlock()
		}
	}

	var sequence *Sequence
	var collect *Collect
	var ok bool

	subExecTrees := this.getSubExecTrees()
	ops, opc, opsFound := subExecTrees.get(query)
	if opsFound {
		sequence = ops
		collect = opc
		ok = sequence.reopen(this.Context)
		if !ok {
			logging.Warna(func() string { return fmt.Sprintf("Failed to reopen: %v", query.String()) })

			// Since the tree cannot be re-opened - clean up the resources associated with it.
			// Do not use ANY references to the cleaned up sequence, collect operators after this Done() call
			// until the variables pointing to the cleanued up operators are re-assigned
			sequence.Done()
		}
	}
	if !ok {
		qp := subplan.(*plan.QueryPlan)
		pipeline, err := Build(qp.PlanOp(), this.Context)
		if err != nil {
			// Generate our own error for this subquery, in addition to whatever the query above is doing.
			err1 := errors.NewSubqueryBuildError(err)
			this.Error(err1)
			return nil, err1
		}

		// Collect subquery results
		collect = NewCollect(plan.NewCollect(), this.Context)
		sequence = NewSequence(plan.NewSequence(), this.Context, pipeline, collect)
	}
	var track int32
	av, stashTracking := parent.(value.AnnotatedValue)
	if stashTracking {
		track = av.Stash()
	}

	this.Park(func(stop bool) {
		if stop {
			sequence.SendAction(_ACTION_STOP)
		} else {
			sequence.SendAction(_ACTION_PAUSE)
		}
	})

	sequence.RunOnce(this.Context, parent)

	// Await completion
	collect.waitComplete()
	this.Resume()

	results := collect.ValuesOnce()

	// mark execution tree for reuse if possible
	if collect.opState == _DONE {
		collect.opState = _COMPLETED
	}
	subExecTrees.set(query, sequence, collect)

	if stashTracking && track > av.RefCnt() {
		av.Restore(track)
	}

	// Cache results
	if useCache {
		subresults := this.getSubresults()
		subresults.set(query, results)
	}

	return results, nil
}

func useSubqCachedResult(query *algebra.Select) bool {
	return !query.IsCorrelated()
}

func (this *Context) DatastoreURL() string {
	return this.datastore.URL()
}

func (this *Context) getSubqueryTimes() interface{} {
	trees := this.contextSubExecTrees()
	if trees != nil {
		trees.mutex.RLock()
		times := make([]interface{}, 0, len(trees.entries))
		for q, e := range trees.entries {
			var t Operator

			switch len(e) {
			case 0:
				continue
			case 1:
				t = e[0].sequence
			default:
				for _, i := range e {
					if i.sequence == nil {
						continue
					}
					if t == nil {
						t = i.sequence.Copy()
						continue
					}
					t.accrueTimes(i.sequence)
				}

			}
			if t == nil {
				continue
			}
			mq := q.String()
			mt, err := json.Marshal(t)
			if err == nil {
				times = append(times, map[string]interface{}{"subquery": mq, "executionTimings": value.NewValue(mt)})
			}
		}
		trees.mutex.RUnlock()
		if len(times) == 0 {
			return nil
		}
		return times
	}
	return nil
}

func (this *Context) done() {
	trees := this.contextSubExecTrees()
	if trees != nil {
		trees.mutex.RLock()
		for _, e := range trees.entries {
			for _, i := range e {
				if i.sequence != nil {
					i.sequence.Done()
				}
			}
		}
		trees.mutex.RUnlock()
	}
}

func (this *Context) getSubExecTrees() *subqueryArrayMap {
	subExecTrees := this.contextSubExecTrees()
	if subExecTrees != nil {
		return subExecTrees
	}
	return this.initSubExecTrees()
}

func (this *Context) initSubExecTrees() *subqueryArrayMap {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.subExecTrees == nil {
		this.subExecTrees = newSubqueryArrayMap()
	}
	return this.subExecTrees
}

func (this *Context) contextSubExecTrees() *subqueryArrayMap {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.subExecTrees
}

// Context Subquery Plans
func (this *Context) GetSubqueryPlans(init bool) *algebra.SubqueryPlans {
	this.queryMutex.RLock()
	subqueryPlans := this.subqueryPlans
	this.queryMutex.RUnlock()

	if subqueryPlans == nil && init {
		this.queryMutex.Lock()
		subqueryPlans = this.subqueryPlans
		if subqueryPlans == nil {
			subqueryPlans = algebra.NewSubqueryPlans()
			this.subqueryPlans = subqueryPlans
		}
		this.queryMutex.Unlock()
	}
	return subqueryPlans
}

func (this *Context) getSubresults() *subqueryMap {
	subResults := this.contextSubresults()
	if subResults != nil {
		return subResults
	}
	return this.initSubresults()
}

func (this *Context) contextSubresults() *subqueryMap {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.subresults
}

func (this *Context) initSubresults() *subqueryMap {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.subresults == nil {
		this.subresults = newSubqueryMap(false)
	}
	return this.subresults
}

// Synchronized map
type subqueryMap struct {
	mutex   sync.RWMutex
	entries map[*algebra.Select]interface{}
}

func newSubqueryMap(plan bool) *subqueryMap {
	rv := &subqueryMap{}
	rv.entries = make(map[*algebra.Select]interface{})
	return rv
}

func (this *subqueryMap) get(key *algebra.Select) (interface{}, bool) {
	this.mutex.RLock()
	rv, ok := this.entries[key]
	this.mutex.RUnlock()
	return rv, ok
}

func (this *subqueryMap) set(key *algebra.Select, value interface{}) {
	this.mutex.Lock()
	this.entries[key] = value
	this.mutex.Unlock()
}

type subqueryEntry struct {
	sequence *Sequence
	collect  *Collect
}

// since the same copy of a subquery may be running in parallel
// we instantiate as many execution trees as required, and collate
// the times at the end
// while an execution tree is in use, it is removed from the map
type subqueryArrayMap struct {
	mutex   sync.RWMutex
	entries map[*algebra.Select][]subqueryEntry
}

func newSubqueryArrayMap() *subqueryArrayMap {
	rv := &subqueryArrayMap{}
	rv.entries = make(map[*algebra.Select][]subqueryEntry)
	return rv
}

func (this *subqueryArrayMap) get(key *algebra.Select) (*Sequence, *Collect, bool) {
	this.mutex.Lock()
	e, ok := this.entries[key]
	if ok {
		for i, l := range e {
			// if we didn't manage to reopen the execution tree, the previous copies are there for profiling only
			if l.collect.opState == _PAUSED || l.collect.opState == _COMPLETED {
				this.entries[key] = append(e[:i], e[i+1:]...)
				this.mutex.Unlock()
				return l.sequence, l.collect, ok
			}
		}
	}
	this.mutex.Unlock()
	return nil, nil, false
}

func (this *subqueryArrayMap) set(key *algebra.Select, sequence *Sequence, collect *Collect) {
	this.mutex.Lock()
	e := this.entries[key]
	if e == nil {
		e = []subqueryEntry{{sequence, collect}}
	} else {
		e = append(e, subqueryEntry{sequence, collect})
	}
	this.entries[key] = e
	this.mutex.Unlock()
}

// assertion checks

func (this *Context) assert(test bool, what string) bool {
	if test {
		return true
	}
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	s := string(buf[0:n])
	plan, _ := json.Marshal(this.prepared.Operator)
	logging.Severef("assert failure: %v ", what)
	logging.Severef("request text:<ud>%v</ud> ", this.prepared.Text())
	logging.Severef(" request plan: %s ", plan)
	logging.Severef("stack: %v", s)
	this.Abort(errors.NewExecutionInternalError(what))
	return false
}

func (this *Context) Recover(base *base) {
	err := recover()
	if err != nil {
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, false)
		s := string(buf[0:n])
		stmt := "<ud>" + this.prepared.Text() + "</ud>"
		qc := "<ud>" + this.queryContext + "</ud>"
		logging.Severef("panic: %v ", err)
		logging.Severef("request text: %v", stmt)
		logging.Severef("query context: %v", qc)
		logging.Severef("stack: %v", s)

		// TODO - this may very well be a duplicate, if the orchestrator is redirecting
		// the standard error to the same file as the log
		os.Stderr.WriteString(s)
		os.Stderr.Sync()

		event.Report(event.CRASH, event.ERROR, "error", err, "request-id", this.RequestId(),
			"statement", event.UpTo(stmt, 500), "query_context", event.UpTo(qc, 250), "stack", event.CompactStack(s, 2000))

		this.Abort(errors.NewExecutionPanicError(nil, fmt.Sprintf("Panic: %v", err)))

		// signal other operators that we are done, release resources
		if base != nil {
			base.release(this)
		}
	}
}

// contextless assert - for when we don't have a context!
// no statement text printend, but behaviour consistent with other asserts
func assert(test bool, what string) bool {
	if test {
		return true
	}
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	s := string(buf[0:n])
	logging.Severef("assert failure: %v ", what)
	logging.Severef("stack: %v", s)
	return false
}

/*
The map entry for hash list in the context can be shared among all parallel instances
of an operator (e.g. Filter), and the same hash table can be shared since all instances
should have the same expression for the IN-list and we should have already checked that
the elements of the IN-list are "static".
*/
func (this *Context) GetInlistHash(in *expression.In) *expression.InlistHash {
	this.inlistHashLock.RLock()
	defer this.inlistHashLock.RUnlock()
	if this.inlistHashMap != nil {
		return this.inlistHashMap[in]
	}
	return nil
}

func (this *Context) EnableInlistHash(in *expression.In) {
	if this.inlistHashMap == nil {
		this.inlistHashLock.Lock()
		if this.inlistHashMap == nil {
			this.inlistHashMap = make(map[*expression.In]*expression.InlistHash, 4)
		}
		this.inlistHashLock.Unlock()
	}
	this.inlistHashLock.RLock()
	ih := this.inlistHashMap[in]
	this.inlistHashLock.RUnlock()
	if ih == nil {
		this.inlistHashLock.Lock()
		ih = this.inlistHashMap[in]
		if ih == nil {
			ih = expression.NewInlistHash()
			this.inlistHashMap[in] = ih
		}
		this.inlistHashLock.Unlock()
	}
	ih.EnableHash()
}

func (this *Context) RemoveInlistHash(in *expression.In) {
	this.inlistHashLock.Lock()
	if this.inlistHashMap != nil {
		ih := this.inlistHashMap[in]
		if ih != nil {
			ih.DropHashTab()
			delete(this.inlistHashMap, in)
		}
	}
	this.inlistHashLock.Unlock()
}

func (this *Context) SetAdvisor() {
	this.flags |= CONTEXT_IS_ADVISOR
}

func (this *Context) IsAdvisor() bool {
	return (this.flags & CONTEXT_IS_ADVISOR) != 0
}

func (this *Context) SetTracked(t bool) {
	this.tracked = t
}

func (this *Context) IsTracked() bool {
	return this.tracked
}

// Return the cached regex for the input operator only if the like pattern is unchanged
func (this *Context) GetLikeRegex(in *expression.Like, s string) *regexp.Regexp {
	this.mutex.RLock()
	if this.likeRegexMap == nil {
		this.mutex.RUnlock()
		return nil
	}
	e, ok := this.likeRegexMap[in]
	this.mutex.RUnlock()

	if ok && e.Orig == s {
		return e.Re
	}
	return nil
}

func (this *Context) CacheLikeRegex(in *expression.Like, s string, re *regexp.Regexp) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.likeRegexMap == nil {
		this.likeRegexMap = make(map[*expression.Like]*expression.LikeRegex, 4)
	}
	if _, ok := this.likeRegexMap[in]; !ok {
		this.likeRegexMap[in] = new(expression.LikeRegex)
	}
	this.likeRegexMap[in].Orig = s
	this.likeRegexMap[in].Re = re
}

var phaseAggs = map[Phases]Phases{
	INDEX_SCAN_GSI:   INDEX_SCAN,
	INDEX_SCAN_FTS:   INDEX_SCAN,
	PRIMARY_SCAN_GSI: PRIMARY_SCAN,
	PRIMARY_SCAN_FTS: PRIMARY_SCAN,
}
