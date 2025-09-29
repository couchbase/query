//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/logging/event"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/natural"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/rewrite"
	"github.com/couchbase/query/semantics"
	queryMetakv "github.com/couchbase/query/server/settings/couchbase"
	"github.com/couchbase/query/system"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Profile int

const (
	ProfUnset = Profile(iota)
	ProfOff
	ProfPhases
	ProfOn
	ProfBench
)

var _PROFILE_MAP = map[string]Profile{
	"off":       ProfOff,
	"phases":    ProfPhases,
	"timings":   ProfOn,
	"benchmark": ProfBench,
}

var _PROFILE_DEFAULT = ProfOff

var _PROFILE_NAMES = []string{
	ProfUnset:  "",
	ProfOff:    "off",
	ProfPhases: "phases",
	ProfOn:     "timings",
	ProfBench:  "benchmark",
}

const (
	TCP_OFF = "off"
	TCP_REQ = "required"
	TCP_OPT = "optional"
)

// This will be set based on the input datastore value as opposed to the flag values.
// The flag values shall only be used for bringing up listeners.
var _IPv6 = false

var _IPv6val = TCP_OPT
var _IPv4val = TCP_REQ

func (profile Profile) String() string {
	return _PROFILE_NAMES[profile]
}

// we should have our own type - but then it would be casting galore in CompareAndSwapUint32...
const (
	_WAIT_EMPTY = uint32(iota)
	_WAIT_GO
	_WAIT_FULL
)

const (
	_TX_QUEUE_SIZE = 16
)

type waitEntry struct {
	request Request
	state   uint32
}

type runQueue struct {
	head      atomic.AlignedUint64
	tail      atomic.AlignedUint64
	servicers int
	size      int32
	runCnt    int32
	queueCnt  int32
	fullQueue int32
	queue     []waitEntry
	mutex     sync.RWMutex
	name      string
}

type txRunQueues struct {
	mutex    sync.RWMutex
	queueCnt int32
	size     int32
	txQueues map[string]*runQueue
}

const (
	_SERVER_RUNNING  = 0
	_REQUESTED       = 1
	_SERVER_SHUTDOWN = 2
)

type requestGate struct {
	counter        atomic.AlignedUint64
	waiting        atomic.AlignedInt64
	lock           sync.Mutex
	cond           *sync.Cond
	bypass         bool
	sleepInterrupt chan bool
	enabled        bool // indicates if the gate should perform its activities.
}

func (this *requestGate) init() {
	this.cond = sync.NewCond(&this.lock)
	this.sleepInterrupt = make(chan bool, 1)
}

func (this *requestGate) mustWait() bool {
	// Check without lock because this can be a hot path
	// The admission control featureis disabled by default and rarely enabled.
	// Race conditions during enablement toggling are handled safely in wait().
	// In the worst case, a few requests might skip waiting when the feature is toggled from disabled to enabled.
	// The check for this.enabled under lock in wait() ensures no requests are orphaned when the feature is
	// toggled from disabled to enabled.
	if !this.enabled {
		return false
	}

	return !this.bypass || this.waiting > 0
}

func (this *requestGate) wait(request Request, l logging.Log) {
	this.cond.L.Lock()
	// Whenever a new waiter joins the queue, a new check should be made if the head of the queue can be released to run
	select {
	case this.sleepInterrupt <- true:
	default:
	}

	// Check if the gate is enabled so that the request does not wait if the admission control has been disabled
	if this.enabled {
		logging.Infof("Pausing request %v due to memory pressure.", request.Id())
		start := time.Now()

		atomic.AddUint64(&this.counter, 1)
		atomic.AddInt64(&this.waiting, 1)
		this.cond.Wait()
		atomic.AddInt64(&this.waiting, -1)

		request.SetAdmissionWaitTime(time.Now().Sub(start))
		logging.Infof("Resuming paused request %v after %v.", request.Id(),
			util.FormatDuration(request.AdmissionWaitTime(), request.DurationStyle()))
		l.Infof("Request was paused for %v", util.FormatDuration(request.AdmissionWaitTime(), request.DurationStyle()))
	}

	this.cond.L.Unlock()
}

func (this *requestGate) releaseOne() {
	this.cond.L.Lock()
	this.cond.Signal()
	this.cond.L.Unlock()
}

func (this *requestGate) releaseAll() {
	// Release all waiters
	this.cond.L.Lock()
	this.cond.Broadcast()
	this.cond.L.Unlock()
}

func (this *requestGate) count() uint64 {
	return atomic.LoadUint64(&this.counter)
}

func (this *requestGate) waiters() uint64 {
	return uint64(atomic.LoadInt64(&this.waiting))
}

func (this *requestGate) changeState(enabled bool) {
	// Modify enabled flag under lock to:
	// 1. wait() checks this state under lock ensuring it sees the latest value. Prevents requests waiting when
	//  admission control is changed to disabled
	// 2. Broadcast/Signal/Wait must use the condional variable lock, to ensure that no waiters are orphaned.
	// Without this, a request might check if waiting is needed, then the gate gets disabled, and the request could wait
	// indefinitely for a signal that never comes.
	this.cond.L.Lock()
	this.enabled = enabled
	this.cond.L.Unlock()
}

type Server struct {
	// due to alignment issues on x86 platforms these atomic
	// variables need to right at the beginning of the structure
	maxParallelism atomic.AlignedInt64
	keepAlive      atomic.AlignedInt64
	requestSize    atomic.AlignedInt64

	sync.RWMutex
	unboundQueue           runQueue
	plusQueue              runQueue
	transactionQueues      txRunQueues
	datastore              datastore.Datastore
	systemstore            datastore.Systemstore
	configstore            clustering.ConfigurationStore
	acctstore              accounting.AccountingStore
	namespace              string
	readonly               bool
	timeout                time.Duration
	txTimeout              time.Duration
	signature              bool
	metrics                bool
	memprofile             string
	cpuprofile             string
	enterprise             bool
	pretty                 bool
	srvprofile             Profile
	srvcontrols            bool
	allowlist              map[string]interface{}
	autoPrepare            bool
	memoryQuota            uint64
	atrCollection          string
	numAtrs                int
	settingsCallback       func(string, interface{})
	gcpercent              int
	shutdown               int
	shutdownStart          time.Time
	requestErrorLimit      int
	memoryStats            runtime.MemStats
	lastTotalTime          int64
	lastNow                time.Time
	lastCpuPercent         float64
	useReplica             value.Tristate
	requestGate            requestGate
	admissionsLock         sync.Mutex
	admissionsRoutineState admissionsRoutineState
}

// Default and min Keep Alive Length

const KEEP_ALIVE_DEFAULT = 1024 * 16
const KEEP_ALIVE_MIN = 1024

const (
	SERVICERS_MULTIPLIER     = 4
	PLUSSERVICERS_MULTIPLIER = 16
)

type admissionsRoutineState int

const (
	_INACTIVE admissionsRoutineState = iota
	_ACTIVE
	_RESTART
)

func NewServer(store datastore.Datastore, sys datastore.Systemstore, config clustering.ConfigurationStore,
	acctng accounting.AccountingStore, namespace string, readonly bool,
	requestsCap, plusRequestsCap int, servicers, plusServicers, maxParallelism int,
	timeout time.Duration, signature, metrics, enterprise, pretty bool,
	srvprofile Profile, srvcontrols bool, initialCfg queryMetakv.Config) (*Server, errors.Error) {
	rv := &Server{
		datastore:        store,
		systemstore:      sys,
		configstore:      config,
		acctstore:        acctng,
		namespace:        namespace,
		readonly:         readonly,
		signature:        signature,
		timeout:          timeout,
		metrics:          metrics,
		enterprise:       enterprise,
		pretty:           pretty,
		srvcontrols:      srvcontrols,
		srvprofile:       srvprofile,
		settingsCallback: func(s string, v interface{}) {},
	}

	rv.unboundQueue.name = "unbound"
	rv.plusQueue.name = "plus"
	rv.SetServicers(servicers)
	rv.SetPlusServicers(plusServicers)
	newRunQueue("unbound", &rv.unboundQueue, requestsCap, false)
	newRunQueue("plus", &rv.plusQueue, plusRequestsCap, false)
	newTxRunQueues(&rv.transactionQueues, plusRequestsCap, _TX_QUEUE_SIZE)
	store.SetLogLevel(logging.LogLevel())
	rv.SetMaxParallelism(maxParallelism)
	rv.SetNumAtrs(datastore.DEF_NUMATRS)
	rv.SetUseReplica(value.NONE)

	// Must be initialized before N1QL feature control can be set
	rv.requestGate.init()
	rv.admissionsRoutineState = _INACTIVE

	// set default values
	rv.SetMaxIndexAPI(datastore.INDEX_API_MAX)
	if rv.enterprise {
		rv.SetN1qlFeatureControl(util.DEF_N1QL_FEAT_CTRL)
		util.SetUseCBO(util.DEF_USE_CBO)
	} else {
		rv.SetN1qlFeatureControl(util.DEF_N1QL_FEAT_CTRL | util.CE_N1QL_FEAT_CTRL)
		util.SetUseCBO(util.CE_USE_CBO)
	}

	// Setup callback function for metakv settings changes
	callb := func(cfg queryMetakv.Config) {
		logging.Infof("Settings notifier from metakv")

		// SetParamValuesForAll accepts a full-set or subset of global configuration
		// and updates those fields.
		SetParamValuesForAll(cfg, rv)
	}

	if initialCfg != nil {
		SetParamValuesForAll(initialCfg, rv)
	}
	queryMetakv.SetupSettingsNotifier(callb, make(chan struct{}))

	// set namespaces in parser
	ns, _ := store.NamespaceNames()
	ss, _ := sys.NamespaceNames()
	nsm := make(map[string]interface{}, len(ns)+len(ss))
	for i, _ := range ns {
		nsm[ns[i]] = true
	}
	for i, _ := range ss {
		nsm[ss[i]] = true
	}
	n1ql.SetNamespaces(nsm)
	rv.StartStatsCollector()
	return rv, nil
}

func MetakvSubscribe() {
	// Subscribe FTS Client Metakv information
	queryMetakv.Subscribe(N1ftyMetakvNotifier, queryMetakv.FTSMetaDir, make(chan struct{}))
}

func (this *Server) SetSettingsCallback(f func(string, interface{})) {
	this.settingsCallback = f
}

func (this *Server) SettingsCallback() func(string, interface{}) {
	return this.settingsCallback
}

func (this *Server) Datastore() datastore.Datastore {
	return this.datastore
}

func (this *Server) Systemstore() datastore.Systemstore {
	return this.systemstore
}

func (this *Server) Namespace() string {
	return this.namespace
}

// Sets the Server Allowlist field for CURL()
// Transforms the allowed/disallowed URL strings into valid URL objects
func (this *Server) SetAllowlist(val map[string]interface{}) {
	if v, ok := val["allowed_urls"]; ok {
		aUrls, ok := v.([]interface{})
		if !ok {
			logging.Warnf("CURL allowed URLs list must be a list of strings.")
		}

		allowedUrlObjects := make([]*url.URL, 0, len(aUrls))
		allowedUrlStrings := make([]interface{}, 0, len(aUrls))
		for _, a := range aUrls {
			if aUrl, ok := a.(string); !ok {
				logging.Warnf("CURL allowed URLs list must be a list of strings.")
			} else {

				// Convert string URL to net/url object that is valid to be used in CURL()
				u, err := expression.CurlURLStringToObject(aUrl)
				if err != nil {
					logging.Warnf("URL in CURL allowed URLs list: %s - not in valid format."+
						" The URL must include a supported protocol, host and all other components of the URL.", aUrl)
				} else {
					allowedUrlObjects = append(allowedUrlObjects, u)
					allowedUrlStrings = append(allowedUrlStrings, aUrl)
				}
			}
		}

		val["allowed_transformed_urls"] = allowedUrlObjects // list of allowed URL objects
		val["allowed_urls"] = allowedUrlStrings             // list of the original allowed URL strings
	}

	if v, ok := val["disallowed_urls"]; ok {
		dUrls, ok := v.([]interface{})
		if !ok {
			logging.Warnf("CURL disallowed URLs list must be a list of strings.")
		}

		disallowedUrlObjects := make([]*url.URL, 0, len(dUrls))
		disallowedUrlStrings := make([]interface{}, 0, len(dUrls))

		for _, d := range dUrls {
			if dUrl, ok := d.(string); !ok {
				logging.Warnf("CURL disallowed URLs list must be a list of strings.")
			} else {

				// Convert string URL to net/url object that is valid to be used in CURL()
				u, err := expression.CurlURLStringToObject(dUrl)
				if err != nil {
					logging.Warnf("URL in CURL disallowed URLs list: %s - not in valid format."+
						" The URL must include a supported protocol, host and all other components of the URL.", dUrl)
				} else {
					disallowedUrlObjects = append(disallowedUrlObjects, u)
					disallowedUrlStrings = append(disallowedUrlStrings, dUrl)
				}
			}
		}
		val["disallowed_transformed_urls"] = disallowedUrlObjects // list of disallowed URL objects
		val["disallowed_urls"] = disallowedUrlStrings             // list of the original disallowed URL strings
	}

	this.allowlist = val
}

func (this *Server) GetAllowlist() map[string]interface{} {
	return this.allowlist
}

func (this *Server) ConfigurationStore() clustering.ConfigurationStore {
	return this.configstore
}

func (this *Server) AccountingStore() accounting.AccountingStore {
	return this.acctstore
}

func (this *Server) Signature() bool {
	return this.signature
}

func (this *Server) Metrics() bool {
	return this.metrics
}

func (this *Server) Pretty() bool {
	this.RLock()
	defer this.RUnlock()
	return this.pretty
}

func (this *Server) SetAutoPrepare(autoPrepare bool) {
	this.Lock()
	defer this.Unlock()
	this.autoPrepare = autoPrepare
}

func (this *Server) AutoPrepare() bool {
	this.RLock()
	defer this.RUnlock()
	return this.autoPrepare
}

func (this *Server) SetPretty(pretty bool) {
	this.Lock()
	defer this.Unlock()
	this.pretty = pretty
}

func (this *Server) KeepAlive() int {
	return int(atomic.LoadInt64(&this.keepAlive))
}

func (this *Server) SetKeepAlive(keepAlive int) {
	atomic.StoreInt64(&this.keepAlive, int64(keepAlive))
}

func (this *Server) MaxParallelism() int {
	return int(atomic.LoadInt64(&this.maxParallelism))
}

func (this *Server) SetMaxParallelism(maxParallelism int) {
	numProcs := util.NumCPU()

	// maxParallelism zero or negative or exceeds number of allowed procs limit to numProcs
	if maxParallelism <= 0 || maxParallelism > numProcs {
		maxParallelism = numProcs
	}

	atomic.StoreInt64(&this.maxParallelism, int64(maxParallelism))
}

func (this *Server) MemProfile() string {
	this.RLock()
	defer this.RUnlock()
	return this.memprofile
}

func (this *Server) SetMemProfile(memprofile string) {
	this.Lock()
	defer this.Unlock()
	this.memprofile = memprofile
}

func (this *Server) CpuProfile() string {
	this.RLock()
	defer this.RUnlock()
	return this.cpuprofile
}

func (this *Server) SetCpuProfile(cpuprofile string) {
	this.Lock()
	defer this.Unlock()
	this.cpuprofile = cpuprofile
	if this.cpuprofile == "" {
		return
	}
	f, err := os.Create(this.cpuprofile)
	if err != nil {
		logging.Errorf("Cannot start cpu profiler - error: %v", err)
		this.cpuprofile = ""
	} else {
		pprof.StartCPUProfile(f)
	}
}

func (this *Server) MutexProfile() bool {
	return runtime.SetMutexProfileFraction(-1) > 0
}

func (this *Server) SetMutexProfile(profile bool) {
	if profile {
		runtime.SetMutexProfileFraction(1)
	} else {
		runtime.SetMutexProfileFraction(0)
	}
}

func (this *Server) ScanCap() int64 {
	return datastore.GetScanCap()
}

func (this *Server) SetScanCap(scan_cap int64) {
	datastore.SetScanCap(scan_cap)
}

func (this *Server) PipelineCap() int64 {
	return execution.GetPipelineCap()
}

func (this *Server) SetPipelineCap(pipeline_cap int64) {
	execution.SetPipelineCap(pipeline_cap)
}

func (this *Server) PipelineBatch() int {
	return execution.PipelineBatchSize()
}

func (this *Server) SetPipelineBatch(pipeline_batch int) {
	execution.SetPipelineBatch(pipeline_batch)
}

func (this *Server) MaxIndexAPI() int {
	return util.GetMaxIndexAPI()
}

func (this *Server) SetMaxIndexAPI(apiVersion int) {
	if apiVersion < datastore.INDEX_API_MIN || apiVersion > datastore.INDEX_API_MAX {
		apiVersion = datastore.INDEX_API_MIN
	}
	util.SetMaxIndexAPI(apiVersion)
}

func (this *Server) Debug() bool {
	return logging.LogLevel() == logging.DEBUG
}

func (this *Server) SetDebug(debug bool) {
	if debug {
		this.SetLogLevel("debug")
	} else {
		this.SetLogLevel("info")
	}
}

func (this *Server) LogLevel() string {
	return logging.LogLevelString()
}

func (this *Server) SetLogLevel(level string) {
	lvl, ok, f := logging.ParseLevel(level)
	if !ok {
		logging.Errorf("SetLogLevel: unrecognized level %v", level)
		return
	}
	if this.datastore != nil {
		this.datastore.SetLogLevel(lvl)
	}
	logging.SetLevel(lvl)
	logging.SetDebugFilter(f)
}

const (
	MAX_REQUEST_SIZE = 64 * (1 << 20)
)

func (this *Server) RequestSizeCap() int {
	return int(atomic.LoadInt64(&this.requestSize))
}

func (this *Server) SetRequestSizeCap(requestSize int) {
	if requestSize <= 0 {
		requestSize = math.MaxInt32
	}
	atomic.StoreInt64(&this.requestSize, int64(requestSize))
}

func (this *Server) Servicers() int {
	return this.unboundQueue.servicers
}

func (this *Server) SetServicers(servicers int) {
	this.Lock()
	if servicers <= 0 {
		servicers = SERVICERS_MULTIPLIER * util.NumCPU()
	}
	this.unboundQueue.SetServicers(servicers)
	this.Unlock()
}

func (this *Server) PlusServicers() int {
	return this.plusQueue.servicers
}

func (this *Server) SetPlusServicers(plusServicers int) {
	this.Lock()
	if plusServicers <= 0 {
		plusServicers = PLUSSERVICERS_MULTIPLIER * util.NumCPU()
	}
	this.plusQueue.SetServicers(plusServicers)
	this.Unlock()
}

func (this *Server) Timeout() time.Duration {
	return this.timeout
}

func (this *Server) SetTimeout(timeout time.Duration) {
	this.timeout = timeout
}

func (this *Server) RequestTimeout(requestTimeout time.Duration) time.Duration {
	var timeout time.Duration
	if requestTimeout > 0 {
		timeout = requestTimeout
	}

	// never allow request side timeout to be higher than server side timeout
	if this.timeout > 0 && (timeout == 0 || this.timeout < timeout) {
		timeout = this.timeout
	}
	return timeout
}

func (this *Server) TxTimeout() time.Duration {
	return this.txTimeout
}

func (this *Server) SetTxTimeout(timeout time.Duration) {
	if timeout < 0 {
		timeout = 0
	}
	this.txTimeout = timeout
	datastore.GetTransactionSettings().SetTxTimeout(timeout)
}

func (this *Server) Profile() Profile {
	return this.srvprofile
}

func (this *Server) SetProfile(srvprofile Profile) {
	this.srvprofile = srvprofile
}

func (this *Server) Controls() bool {
	return this.srvcontrols
}

func (this *Server) SetControls(srvcontrols bool) {
	this.srvcontrols = srvcontrols
}

func ParseProfile(name string, bench bool) (Profile, bool) {
	prof, ok := _PROFILE_MAP[strings.ToLower(name)]
	if ok {
		if prof != ProfBench || bench {
			return prof, true
		}
	}
	return _PROFILE_DEFAULT, false
}

func (this *Server) MemoryQuota() uint64 {
	return this.memoryQuota
}

func (this *Server) SetMemoryQuota(memoryQuota uint64) {
	this.memoryQuota = memoryQuota
}

func (this *Server) AtrCollection() string {
	return this.atrCollection
}

func (this *Server) SetAtrCollection(s string) {
	this.atrCollection = s
	datastore.GetTransactionSettings().SetAtrCollection(s)
}

func (this *Server) NumAtrs() int {
	return this.numAtrs
}

func (this *Server) SetNumAtrs(i int) {
	this.numAtrs = i
	datastore.GetTransactionSettings().SetNumAtrs(i)
}

func (this *Server) Enterprise() bool {
	return this.enterprise
}

func (this *Server) UseCBO() bool {
	return util.GetUseCBO()
}

func (this *Server) SetUseCBO(useCBO bool) {
	util.SetUseCBO(useCBO)
}

func (this *Server) RequestErrorLimit() int {
	return this.requestErrorLimit
}

func (this *Server) SetRequestErrorLimit(limit int) error {
	if limit < 0 {
		limit = 0
	}
	this.requestErrorLimit = limit
	return nil
}

func (this *Server) IsHealthy() bool {
	return !this.unboundQueue.isFull() && !this.plusQueue.isFull()
}

func (this *Server) serviceNaturalRequest(request Request) (bool, bool) {
	var err errors.Error
	var elems []*algebra.Path

	nlquery := request.Natural()
	if nlquery == "" {
		return false, true
	}
	oldState := request.State()
	request.SetState(PREPROCESSING)
	defer request.SetState(oldState)

	if !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_NATURAL_LANG_REQ) {
		request.Fail(errors.NewNaturalLanguageRequestError(errors.E_NL_REQ_FEAT_DISABLED))
		request.Failed(this)
		return true, false
	}

	elems, err = algebra.ParseAndValidatePathList(request.NaturalContext(), "default", request.QueryContext())
	if err == nil && len(elems) > natural.MAX_KEYSPACES {
		err = errors.NewNaturalLanguageRequestError(errors.E_NL_TOO_MANY_KEYSPACES)
	}
	if err != nil {
		request.Fail(errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, err))
		request.Failed(this)
		return true, false
	}

	if !this.setupRequestContext(request) {
		request.Failed(this)
		return true, false
	}

	nloutputOpt := natural.SQL
	if nloutput := request.NaturalOutput(); nloutput != "" {
		nloutputOpt = natural.NewNaturalOutput(nloutput)
		switch nloutputOpt {
		case natural.UNDEFINED_NATURAL_OUTPUT:
			err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutput)
		case natural.SQL:
			nloutputOpt = natural.SQL
		case natural.JSUDF:
			nloutputOpt = natural.JSUDF
		case natural.FTSSQL:
			nloutputOpt = natural.FTSSQL
		}
	}
	if err != nil {
		request.Fail(err)
		request.Failed(this)
		return true, false
	}
	request.SetNaturalOutput(nloutputOpt.String())

	var nlAlgebraStmt algebra.Statement
	var stmt string
	stmt, nlAlgebraStmt, err = natural.ProcessRequest(request.NaturalCred(), request.NaturalOrganizationId(),
		nlquery, elems, nloutputOpt, request.NaturalExplain(), request.NaturalAdvise(),
		request.ExecutionContext(), request.Output().AddPhaseTime)
	if err != nil {
		request.Fail(err)
		request.Failed(this)
		return true, false
	}

	request.SetStatement(stmt)
	request.SetQueryContext("")
	request.IncrementStatementCount()
	request.SetNaturalStatement(nlAlgebraStmt)

	if (nlAlgebraStmt.Type() != "SELECT" && nlAlgebraStmt.Type() != "ADVISE" &&
		nlAlgebraStmt.Type() != "EXPLAIN") || request.NaturalShowOnly() {
		request.CompletedNaturalRequest(this)
		return true, false
	}

	return false, false
}

func (this *Server) ServiceRequest(request Request) bool {

	stop, needSetup := this.serviceNaturalRequest(request)

	if stop {
		return true // so that StatusServiceUnavailable will not return
	} else if needSetup && !this.setupRequestContext(request) {
		request.Failed(this)
		return true // so that StatusServiceUnavailable will not return
	}

	return this.handleRequest(request, &this.unboundQueue)
}

func (this *Server) PlusServiceRequest(request Request) bool {

	stop, needSetup := this.serviceNaturalRequest(request)

	if stop {
		return true // so that StatusServiceUnavailable will not return
	} else if needSetup && !this.setupRequestContext(request) {
		request.Failed(this)
		return true // so that StatusServiceUnavailable will not return
	}

	return this.handlePlusRequest(request, &this.plusQueue, &this.transactionQueues)
}

func (this *Server) setupRequestContext(request Request) bool {
	namespace := request.Namespace()
	if namespace == "" {
		namespace = this.namespace
	}

	optimizer := GetNewOptimizer()
	maxParallelism := request.MaxParallelism()
	if maxParallelism <= 0 || maxParallelism > this.MaxParallelism() {
		maxParallelism = this.MaxParallelism()
	}

	context := execution.NewContext(request.Id().String(), this.datastore, this.systemstore, namespace,
		this.readonly || request.Readonly() == value.TRUE,
		maxParallelism, request.ScanCap(), request.PipelineCap(), request.PipelineBatch(),
		request.NamedArgs(), request.PositionalArgs(), request.Credentials(), request.ScanConsistency(),
		request.ScanVectorSource(), request.Output(), nil, request.IndexApiVersion(), request.FeatureControls(),
		request.QueryContext(), request.UseFts(), request.UseCBO(), optimizer, request.KvTimeout(), request.Timeout(),
		request.LogLevel())
	context.SetAllowlist(this.allowlist)
	context.SetDurability(request.DurabilityLevel(), request.DurabilityTimeout())
	context.SetScanConsistency(request.ScanConsistency(), request.OriginalScanConsistency())
	context.SetPreserveExpiry(request.PreserveExpiry())
	context.SetTracked(request.Tracked())
	context.SetTenantCtx(request.TenantCtx())
	context.SetPreserveProjectionOrder(!request.SortProjection())
	context.SetDurationStyle(request.DurationStyle())
	context.SetUserAgent(request.UserAgent())
	context.SetUsers(datastore.CredsString(request.Credentials()))
	context.SetRemoteAddr(request.RemoteAddr())

	if request.TxId() != "" {
		err := context.SetTransactionInfo(request.TxId(), request.TxStmtNum())
		if err == nil {
			err = context.TxContext().TxValid()
		}

		if err != nil {
			if this.ShuttingDown() && !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_PART_GRACEFUL) {
				if this.ShutDown() {
					err = errors.NewServiceShutDownError()
				} else {
					err = errors.NewServiceShuttingDownError()
				}
			}
			request.Fail(err)
			return false
		}

		if request.OriginalScanConsistency() == datastore.NOT_SET {
			request.SetScanConsistency(context.TxContext().TxScanConsistency())
		}
		request.SetDurabilityLevel(context.TxContext().TxDurabilityLevel())
		request.SetDurabilityTimeout(context.TxContext().TxDurabilityTimeout())
		request.SetTransactionStartTime(context.TxContext().TxStartTime())
		request.SetTxTimeout(context.TxContext().TxTimeout())
	}
	request.SetExecutionContext(context)
	return true
}

func (this *Server) handleRequest(request Request, queue *runQueue) bool {
	mark := util.Now()
	if !queue.enqueue(request) {
		ffdc.Capture(ffdc.RequestQueueFull)
		request.Fail(errors.NewServiceErrorRequestQueueFull())
		request.Failed(this)
		return false
	} else {
		ffdc.Reset(ffdc.RequestQueueFull)
	}
	request.Output().AddPhaseTime(execution.QUEUED, util.Since(mark))

	defer queue.dequeue()

	if !request.Alive() {
		request.Fail(errors.NewServiceNoClientError())
		request.Failed(this)
		return true
	}

	this.serviceRequest(request) // service

	return true
}

func (this *Server) handlePlusRequest(request Request, queue *runQueue, transactionQueues *txRunQueues) bool {
	mark := util.Now()
	if !queue.enqueue(request) {
		ffdc.Capture(ffdc.PlusQueueFull)
		request.Fail(errors.NewServiceErrorRequestQueueFull())
		request.Failed(this)
		return false
	} else {
		ffdc.Reset(ffdc.PlusQueueFull)
	}
	request.Output().AddPhaseTime(execution.QUEUED, util.Since(mark))

	dequeue := true
	defer func() {
		if dequeue {
			queue.dequeue()
		}
	}()

	if !request.Alive() {
		request.Fail(errors.NewServiceNoClientError())
		request.Failed(this)
		return true
	}

	if request.TxId() != "" {
		mark := util.Now()
		err := this.handlePreTxRequest(request, queue, transactionQueues)
		request.Output().AddPhaseTime(execution.QUEUED, util.Since(mark))
		if err != nil {
			request.Fail(err)
			request.Failed(this) // don't return
		} else {
			if !request.Alive() {
				request.Fail(errors.NewServiceNoClientError())
				request.Failed(this)
				return true
			}
			this.serviceRequest(request) // service
			dequeue = this.handlePostTxRequest(request, queue, transactionQueues)
		}
	} else {
		this.serviceRequest(request) // service
	}

	return true
}

func (this *Server) handlePreTxRequest(request Request, queue *runQueue, transactionQueues *txRunQueues) errors.Error {
	txId := request.TxId()
	txContext := request.ExecutionContext().TxContext()
	if err := txContext.TxValid(); err != nil {
		return err
	}

	transactionQueues.mutex.Lock()

	if txContext.TxInUse() {
		txQueue, ok := transactionQueues.txQueues[txId]
		if !ok {
			txQueue = &runQueue{servicers: 1}
			newRunQueue(txId, txQueue, int(transactionQueues.size), true)
			transactionQueues.txQueues[txId] = txQueue
		}
		txQueue.mutex.Lock()
		if txQueue.runCnt >= txQueue.size {
			txQueue.mutex.Unlock()
			transactionQueues.mutex.Unlock()
			return errors.NewTransactionQueueFull()
		}
		txQueue.runCnt++
		transactionQueues.queueCnt++
		queue.dequeue()                                                       // release the servicer
		txQueue.addRequest(request, &txQueue.mutex, &transactionQueues.mutex) // unlock done inside
		return nil
	}

	defer transactionQueues.mutex.Unlock()
	return txContext.SetTxInUse(true)
}

func (this *Server) handlePostTxRequest(request Request, queue *runQueue, transactionQueues *txRunQueues) bool {
	txId := request.TxId()
	transactionQueues.mutex.Lock()
	txContext := request.ExecutionContext().TxContext()

	txQueue, ok := transactionQueues.txQueues[txId]
	if ok {
		txQueue.mutex.Lock()
		if txQueue.runCnt > 0 {
			txQueue.runCnt--
			transactionQueues.queueCnt--
			txQueue.releaseRequest(&txQueue.mutex, &transactionQueues.mutex) // unlock done inside
			return false
		} else {
			delete(transactionQueues.txQueues, txId)
		}
		txQueue.mutex.Unlock()
	}

	if txContext != nil {
		txContext.SetTxInUse(false)
	}
	transactionQueues.mutex.Unlock()

	return true
}

// _LAG_MULTIPLIER defines how many full queue cycles occur before wrapping the underlying array
// this is needed to accomodate routines that may be suspended for extended periods (due to load) and therefore lag behind the
// "current" queue
// this buffer allows us to avoid concurrent operations on the same array entry due to thread scheduling
// this applies only to non-transaction queues (currently 2 of)
//
// it also accomodates the small timing hole in dequeue() between reducing the queueCnt and actually releasing a slot
// - the number of servicers should never be larger than the requests cap times the number of CPUs times _LAG_MULTIPLIER, but no
//   checks are made to ensure this
//
// waitEntry is 24 bytes so even with 64 CPUs and a (default) queue length of 16384, a multiplier of 10 doesn't mean the space
// required is significant relative to the overall service memory requirements (3.75 MiB) (per queue)

const _LAG_MULTIPLIER = 10

func newRunQueue(n string, q *runQueue, num int, txflag bool) {
	q.name = n
	if txflag {
		q.size = int32(num)
	} else {
		q.size = int32(num * _LAG_MULTIPLIER)
	}
	q.fullQueue = int32(num)
	q.queue = make([]waitEntry, q.size)
	if !txflag {
		go q.checkWaiters()
	}
}

func newTxRunQueues(q *txRunQueues, nqueues, num int) {
	q.size = int32(num)
	q.txQueues = make(map[string]*runQueue, nqueues)
}

func (this *runQueue) SetServicers(num int) {
	this.servicers = num
}

func (this *runQueue) isFull() bool {
	return atomic.LoadInt32(&this.queueCnt) >= this.fullQueue
}

func (this *runQueue) enqueue(request Request) bool {
	runCnt := int(atomic.AddInt32(&this.runCnt, 1))

	// if servicers exceeded, reserve a spot in the queue
	if runCnt > this.servicers {

		atomic.AddInt32(&this.runCnt, -1)
		queueCnt := atomic.AddInt32(&this.queueCnt, 1)

		// rats! queue full, can't handle this request
		if queueCnt > this.fullQueue {
			atomic.AddInt32(&this.queueCnt, -1)
			return false
		}

		// (RC #1) wait
		this.addRequest(request, nil, nil)
	}
	return true
}

func (this *runQueue) dequeue() {
	// anyone to release?
	queueCnt := atomic.LoadInt32(&this.queueCnt)
	wakeUp := queueCnt > 0
	if wakeUp {
		queueCnt = atomic.AddInt32(&this.queueCnt, -1)
		wakeUp = queueCnt >= 0

		// (RC #2) rats! waker already gone
		if !wakeUp {
			atomic.AddInt32(&this.queueCnt, 1)
		}
	}

	if wakeUp {
		this.releaseRequest(nil, nil)
	} else {
		atomic.AddInt32(&this.runCnt, -1)
	}
}

// in this next couple of methods, the waiter populates the wait entry and then tries the CAS
// the waker tries the CAS and then empties
func (this *runQueue) addRequest(request Request, txQueueMutex, txQueuesMutex *sync.RWMutex) {

	// get the next available entry
	entry := int32(atomic.AddUint64(&this.tail, 1) % uint64(this.size))

	var r Request
	if atomic.LoadUint32(&this.queue[entry].state) == _WAIT_FULL {
		// this is a safety-net and shouldn't happen unless a routine has been suspended mid queue operations for
		// long enough for the queue to wrap around the underlying array.
		r = this.queue[entry].request
		atomic.StoreUint32(&this.queue[entry].state, _WAIT_EMPTY)
	}

	// set up the entry and the request
	request.setSleep()
	this.queue[entry].request = request

	// if the safety-net was hit and we found a request, run it
	// we do this after setting up the slot with our request to minimise the window for a concurrent release to interfere
	// with the slot
	if r != nil {
		// we may temporarily have an additional servicer until the queue drains (normal decrement will resolve this)
		atomic.AddInt32(&this.runCnt, 1)
		r.release()
		logging.Errorf("Found full slot (%v) when adding to queue. Releasing.", entry)
	}

	if txQueueMutex != nil {
		txQueueMutex.Unlock()
	}

	if txQueuesMutex != nil {
		txQueuesMutex.Unlock()
	}

	// atomically set the state to full and sleep if successful
	if atomic.CompareAndSwapUint32(&this.queue[entry].state, _WAIT_EMPTY, _WAIT_FULL) {
		request.sleep()
	} else {
		// if the waker got there first, continue
		this.queue[entry].request = nil
		atomic.StoreUint32(&this.queue[entry].state, _WAIT_EMPTY)
		request.release()
	}
}

func (this *runQueue) releaseRequest(txQueueMutex, txQueuesMutex *sync.RWMutex) {

	// get the next available entry
	entry := int32(atomic.AddUint64(&this.head, 1) % uint64(this.size))

	if txQueueMutex != nil {
		txQueueMutex.Unlock()
	}

	if txQueuesMutex != nil {
		txQueuesMutex.Unlock()
	}

	// can we mark the entry as ready to go?
	if !atomic.CompareAndSwapUint32(&this.queue[entry].state, _WAIT_EMPTY, _WAIT_GO) {
		// nope, the waiter got there first, wake it up
		request := this.queue[entry].request
		this.queue[entry].request = nil
		atomic.StoreUint32(&this.queue[entry].state, _WAIT_EMPTY)

		if request != nil {
			request.release()
		} else {
			// this case indicates we have sufficient load that a routine was suspended for long enough for the queue to
			// cycle its underlying array before it got to run
			logging.Errorf("Releasing slot (%v) with nil request.", entry)
			// since we didn't release a request we must relinquish our runCnt
			atomic.AddInt32(&this.runCnt, -1)
		}
	}
}

/*
MB-31848
Ideally we would like to modify runCnt and queueCnt atomically and within
the queues boundaries, however having a lock on the counters does have
substantial performance implications on NUMA architectures, and we hit
a throughput ceiling which we wish to avoid.

Sharding the queues would address the ceiling, but it means that we could
end up with sub-utilised servicers, or conversely no waiters in the current
queue but plenty in others, which would then have to wait for other runners
to be woken up.
In order to fully utilise the servicers, we would have to go round queues
and push ourselves in, or steal waiters to make sure that they are woken up
timely - which kind of counters the advantages of the sharding.

Changing the two counters non atomically and correcting out of bounds increments
does wonders for throughput, but it does open us up to two race conditions,
marked as RC #1 and RC #2 above.
In RC #1, by the time that the waiter has queued itself up, all the runners
might have gone, and the waiter would never be woken up.
In RC #2, we could have so many concurrent queueCnt decrements, that the runner
might mistakenly think that there are no waiters.

What we do here is have a cleanup goroutine that wakes up any left behind waiter.
To the best of my knowledge, this goroutine has never had to correct anythings.
*/
func (this *runQueue) checkWaiters() {
	for {

		// check every tenth of a second
		time.Sleep(100 * time.Millisecond)
		runCnt := atomic.LoadInt32(&this.runCnt)
		queueCnt := atomic.LoadInt32(&this.queueCnt)
		for {
			// no left behind requests
			if runCnt > 0 {
				break
			}
			if queueCnt == 0 {
				for i := int32(0); i < this.size; i++ {
					request := this.queue[i].request
					if request != nil &&
						atomic.LoadInt32(&this.runCnt) == 0 &&
						atomic.CompareAndSwapUint32(&this.queue[i].state, _WAIT_FULL, _WAIT_EMPTY) {

						// found a queued request that isn't scheduled to run; let it run
						this.queue[i].request = nil
						atomic.AddInt32(&this.runCnt, 1)
						request.release()
						logging.Infof("request scheduler released queued request")
						break
					}
				}
				break
			}
			logging.Infof("request scheduler emptying queue: %v requests", queueCnt)

			// can we pull one?
			queueCnt = atomic.AddInt32(&this.queueCnt, -1)
			if queueCnt < 0 {

				// another runner came, went, and emptied the queue
				atomic.AddInt32(&this.queueCnt, 1)
				break
			} else {

				// let it free
				runCnt = atomic.AddInt32(&this.runCnt, 1)
				this.releaseRequest(nil, nil)
			}
		}
	}
}

func (this *runQueue) load(txqueueCnt int) int {
	return 100 * (this.activeRequests() + txqueueCnt) / this.servicers
}

func (this *runQueue) activeRequests() int {
	return int(this.runCnt) + this.queuedRequests()
}

func (this *runQueue) queuedRequests() int {
	return int(this.queueCnt)
}

func (this *Server) Load() int {
	return this.plusQueue.load(this.txQueueCount()) + this.unboundQueue.load(0)
}

func (this *Server) txQueueCount() int {
	return int(this.transactionQueues.queueCnt)
}

func (this *Server) ActiveRequests() int {
	return this.plusQueue.activeRequests() + this.unboundQueue.activeRequests()
}

func (this *Server) QueuedRequests() int {
	return this.unboundQueue.queuedRequests() + this.plusQueue.queuedRequests() + this.txQueueCount()
}

func (this *Server) ServicersPaused() uint64 {
	return this.requestGate.waiters()
}

func (this *Server) ServicerPauses() uint64 {
	return this.requestGate.count()
}

func (this *Server) admit(request Request, l logging.Log) {
	if this.requestGate.mustWait() {
		this.requestGate.wait(request, l)
	}
}

const (
	_ADMISSION_CHECK_TIME_UNIT  = time.Millisecond * 50 // how frequently the ticker fires; minimum resolution
	_ADMISSION_CHECK_INTERVAL   = time.Second
	_ADMISSION_CHECK_INTERVAL_1 = time.Millisecond * 500
	_ADMISSION_CHECK_INTERVAL_2 = time.Millisecond * 200
	_ADMISSION_CHECK_INTERVAL_3 = time.Millisecond * 50
	_ADMISSION_CHECK_INTERVAL_4 = time.Millisecond * 100
	_ADMISSION_ERROR_INTERVAL   = time.Second * 5

	_ADMISSION_CHECK_GC_THRESHOLD = time.Millisecond * 250 // if usage is high and the GC hasn't run in this interval, run it
	_ADMISSION_MIN_FFDC_INTERVAL  = time.Second * 60       // minimum interval between FFDC runs for memory threshold

	_ADMISSION_BYPASS_MEMORY_PERCENT = 80 // below this level we don't perfom gating
	_ADMISSION_MEMORY_THRESHOLD_1    = 60 // after gating, if it drops below this level release an additional waiter
	_ADMISSION_MEMORY_THRESHOLD_2    = 40 // after gating, if it drops below this level release an additional waiter
	_ADMISSION_MEMORY_THRESHOLD_3    = 20 // after gating, if it drops below this level release an additional waiter
	_ADMISSION_MIN_FREE_MEM_PERCENT  = 10 // active request control starts when we drop below this
	_ADMISSION_MAX_RELEASE           = 5  // maximum number of paused requests to resume at a time

	_ADMISSION_MSG_PREFIX = "RAC: "
)

func (this *Server) admissions() {

	this.admissionsLock.Lock()
	// Exit if this routine is already running. Only one admissions routine can run.
	if this.admissionsRoutineState == _ACTIVE {
		this.admissionsLock.Unlock()
		return
	}
	this.admissionsRoutineState = _ACTIVE
	this.admissionsLock.Unlock()

	logging.Infof(_ADMISSION_MSG_PREFIX + "Admission control routine started.")
	triggered := false
	lastFFDC := int64(0)
	lastError := int64(0)
	ticker := time.NewTicker(_ADMISSION_CHECK_TIME_UNIT)
	waitInterval := _ADMISSION_CHECK_INTERVAL
	this.requestGate.bypass = false
	this.requestGate.changeState(true)

	defer func() {

		ticker.Stop()

		// drain the channel
		select {
		case <-this.requestGate.sleepInterrupt:
		default:
		}

		this.admissionsLock.Lock()

		e := recover()
		if e != nil {
			this.admissionsRoutineState = _INACTIVE
			this.admissionsLock.Unlock()
			go this.admissions()
			return
		}

		if this.admissionsRoutineState == _RESTART {
			// Restart the routine if required
			this.admissionsRoutineState = _INACTIVE
			this.admissionsLock.Unlock()
			go this.admissions()
			return
		} else {
			// Routine should no longer be active
			this.admissionsRoutineState = _INACTIVE
		}

		this.admissionsLock.Unlock()
	}()

	for {
		select {
		case <-this.requestGate.sleepInterrupt:
			waitInterval = 0
		case <-ticker.C:
			waitInterval -= _ADMISSION_CHECK_TIME_UNIT
		}
		if waitInterval > 0 {
			continue
		}
		waitInterval = _ADMISSION_CHECK_INTERVAL

		// If admission control is disabled, release all waiting servicers and un-pause all paused active requestss
		if util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_UNRESTRICTED_ADMISSION_AND_ACTIVITY) {
			logging.Infof(_ADMISSION_MSG_PREFIX + "Admission control has been disabled. Shutting down admission control routine.")

			this.requestGate.bypass = true
			this.requestGate.changeState(false)

			// Release all waiting servicers
			for this.requestGate.waiters() > 0 {
				this.requestGate.releaseAll()
			}

			// Un-pause all paused active requests
			released := 0
			var reqs []Request
			ActiveRequestsForEach(func(id string, req Request) bool {
				if req.State() == RUNNING {
					reqs = append(reqs, req)
				}
				return true
			}, nil)

			for _, req := range reqs {
				if ctx := req.ExecutionContext(); ctx != nil {
					if ctx.Pause(false) {
						released++
					}
				}
			}

			if released > 0 {
				logging.Infof(_ADMISSION_MSG_PREFIX+"Request pipeline(s) resumed: %d", released)
			} else {
				logging.Infof(_ADMISSION_MSG_PREFIX + "No request pipelines to resume.")
			}

			return
		}

		mu, mf, lastGC := this.MemoryUsage(true)
		if triggered || mf < _ADMISSION_MIN_FREE_MEM_PERCENT {
			logging.Debugf(_ADMISSION_MSG_PREFIX+"triggered: %v mf: %v", triggered, mf)
			this.requestGate.bypass = false
			if !triggered {
				// don't flood the log with errors
				if n := util.Now().UnixNano(); n-int64(lastError) >= int64(_ADMISSION_ERROR_INTERVAL) {
					logging.Errorf(_ADMISSION_MSG_PREFIX+"Free memory (%d%%) below limit", mf)
					lastError = n
				}
				ffdc.Capture(ffdc.MemoryLimit)
			} else if lastError != 0 {
				logging.Infof(_ADMISSION_MSG_PREFIX+"Free memory: %d%%", mf)
				lastError = 0
			}
			triggered = this.controlActiveProcessing(mf)
			waitInterval = _ADMISSION_CHECK_INTERVAL_4
			logging.Debugf(_ADMISSION_MSG_PREFIX+"triggered: %v", triggered)
			continue
		} else {
			if lastError != 0 {
				logging.Infof(_ADMISSION_MSG_PREFIX+"Free memory: %d%%", mf)
				lastError = 0
			}
			ffdc.Reset(ffdc.MemoryLimit)
		}
		this.requestGate.bypass = (mu < _ADMISSION_BYPASS_MEMORY_PERCENT)
		if !this.requestGate.bypass {
			logging.Debugf(_ADMISSION_MSG_PREFIX+"mu: %v", mu)
			ffdc.Capture(ffdc.MemoryThreshold)
			lastFFDC = util.Now().UnixNano()
		} else {
			if lastFFDC > 0 && util.Now().UnixNano()-lastFFDC > int64(_ADMISSION_MIN_FFDC_INTERVAL) {
				ffdc.Reset(ffdc.MemoryThreshold)
				lastFFDC = 0
			}
		}
		w := this.requestGate.waiters()
		if this.requestGate.bypass && w > 0 {
			logging.Debugf(_ADMISSION_MSG_PREFIX+"w: %v", w)
			this.requestGate.releaseOne()
			if mu < _ADMISSION_MEMORY_THRESHOLD_1 && w > 1 {
				this.requestGate.releaseOne()
				waitInterval = _ADMISSION_CHECK_INTERVAL_1
			}
			if mu < _ADMISSION_MEMORY_THRESHOLD_2 && w > 2 {
				this.requestGate.releaseOne()
				waitInterval = _ADMISSION_CHECK_INTERVAL_2
			}
			if mu < _ADMISSION_MEMORY_THRESHOLD_3 && w > 3 {
				this.requestGate.releaseOne()
				waitInterval = _ADMISSION_CHECK_INTERVAL_3
			}
		} else if !this.requestGate.bypass && util.Now().UnixNano()-int64(lastGC) > int64(_ADMISSION_CHECK_GC_THRESHOLD) {
			logging.Debugf(_ADMISSION_MSG_PREFIX+"w: %v, lastGC: %v", w, lastGC)
			runtime.GC()
		}
	}
}

// This is triggered when we drop below 10% free system memory.  New request processing has already been halted.  This will continue
// to be called until there are no requests left to deal with.  The idea is to pause the pipelines for all active requests and
// trigger a return of memory to the OS.  If this doesn't succeed in returning to above the threshold, then halt the largest memory
// consuming request (this only works as intended for requests with a memory quota in effect).  If we have returned to above the
// threshold, then release some requests - based on how far above the threshold we are.  One request for every 10%.  We'll wait
// a further cycle (1/10th second) before attempting to release any more requests.  This is effectively to drip-feed the active
// requests back in to the system in order to try avoid immediately returning to the condition that triggered this.  Only once all
// the paused active requests have been released (+1/10th second cycle) and we have enough free memory (strictly our usage is below
// 80% of permitted) will queued new requests begin to be released.  This sequence can cycle repeatedly.
func (this *Server) controlActiveProcessing(perc uint64) bool {
	defer func() {
		err := recover()
		if err != nil {
			logging.Severef(_ADMISSION_MSG_PREFIX+"Active processing control failed: %v", err)
		}
	}()

	var reqs []Request
	paused := 0
	ActiveRequestsForEach(func(id string, req Request) bool {
		if req.State() == RUNNING {
			if perc < _ADMISSION_MIN_FREE_MEM_PERCENT {
				if ctx := req.ExecutionContext(); ctx != nil {
					if ctx.Pause(true) {
						paused++
					}
				}
			}
			reqs = append(reqs, req)
		}
		return true
	}, nil)
	if paused > 0 {
		logging.Infof(_ADMISSION_MSG_PREFIX+"Request pipeline(s) paused: %d", paused)
	}

	if len(reqs) == 0 {
		logging.Debugf(_ADMISSION_MSG_PREFIX + "No requests stopped. Running GC.")
		debug.FreeOSMemory()
		return false
	} else if len(reqs) > 1 {
		sort.Slice(reqs, func(i int, j int) bool {
			// This is not a strict order as UsedMemory() isn't guaranteed to be constant for the duration of the sort
			umi := reqs[i].UsedMemory()
			umj := reqs[j].UsedMemory()
			if umi == umj {
				return reqs[i].RequestTime().Before(reqs[j].RequestTime())
			}
			return umi < umj
		})
	}

	rv := false
	if perc < _ADMISSION_MIN_FREE_MEM_PERCENT {
		if paused == 0 {
			// stop the largest memory consumer
			req := reqs[len(reqs)-1]
			req.Halt(errors.NewLowMemory(_ADMISSION_MIN_FREE_MEM_PERCENT))
			logging.Infof(_ADMISSION_MSG_PREFIX+"%v: halted.", req.Id())
		}
		rv = true
		debug.FreeOSMemory()
	} else {
		// release requests depending on how much free memory we have
		released := 0
		for n := range reqs {
			if perc >= _ADMISSION_MIN_FREE_MEM_PERCENT && released < _ADMISSION_MAX_RELEASE {
				if ctx := reqs[n].ExecutionContext(); ctx != nil {
					if ctx.Pause(false) {
						perc -= _ADMISSION_MIN_FREE_MEM_PERCENT
						released++
					}
				}
			} else {
				break
			}
		}
		rv = released != 0
		if !rv {
			debug.FreeOSMemory()
			logging.Debugf(_ADMISSION_MSG_PREFIX + "No request pipelines to resume.")
		} else {
			logging.Infof(_ADMISSION_MSG_PREFIX+"Request pipeline(s) resumed: %d", released)
		}
	}
	return rv
}

func (this *Server) serviceRequest(request Request) {
	defer func() {
		err := recover()
		if err != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			stmt := "<ud>" + request.Statement() + "</ud>"
			qc := "<ud>" + request.QueryContext() + "</ud>"
			logging.Severef("panic: %v ", err, request.ExecutionContext())
			logging.Severef("request text: %v", stmt, request.ExecutionContext())
			logging.Severef("query context: %v", qc, request.ExecutionContext())
			logging.Severef("stack: %v", s, request.ExecutionContext())
			os.Stderr.WriteString(s)
			os.Stderr.Sync()
			event.Report(event.CRASH, event.ERROR, "error", err, "request-id", request.Id().String(),
				"statement", event.UpTo(stmt, 500), "query_context", event.UpTo(qc, 250), "stack", event.CompactStack(s, 2000))

			request.Fail(errors.NewExecutionPanicError(nil, fmt.Sprintf("Panic: %v", err)))
			request.Failed(this)
		}
	}()

	context := request.ExecutionContext()

	// If the system is under memory pressure we'll delay starting processing the request whilst holding the servicer in order
	// to reduce concurrent load.  This effectively grants us "a dynamic number of servicers" without complex restructuring
	this.admit(request, context)

	context.Infof("Servicing request")
	request.Servicing()

	atrCollection := this.AtrCollection()
	if request.AtrCollection() != "" {
		atrCollection = request.AtrCollection()
	}
	numAtrs := this.NumAtrs()
	if request.NumAtrs() > 0 {
		numAtrs = request.NumAtrs()
	}
	if request.TxId() != "" {
		err := context.TxContext().TxValid()
		if err == nil {
			err = context.SetTransactionContext(request.Type(), request.TxImplicit(),
				request.TxTimeout(), this.TxTimeout(), atrCollection, numAtrs,
				request.TxData())
		}

		if err != nil {
			request.Fail(err)
			request.Failed(this)
			return
		}
	} else if request.TxImplicit() {
		context.SetDeltaKeyspaces(make(map[string]bool, 1))
	} else {

		// set tup atr collection for possible UDF execution
		context.SetAtrCollection(atrCollection, numAtrs)

		// only enable error limits for non-transactional requests
		if request.GetErrorLimit() == -1 {
			request.SetErrorLimit(this.RequestErrorLimit())
		}
	}

	prepared, err := this.getPrepared(request, context)
	if err != nil {
		request.Fail(err)
	} else {
		context.SetPrepared(prepared)
		context.SetPlanPreparedTime(prepared.PreparedTime())
		if (this.readonly || value.ToBool(request.Readonly())) &&
			(prepared != nil && !prepared.Readonly()) {
			request.Fail(errors.NewServiceErrorReadonly())
		} else if request.TxId() == "" && !request.IsPrepare() {
			atrCollection := this.AtrCollection()
			if request.AtrCollection() != "" {
				atrCollection = request.AtrCollection()
			}
			numAtrs := this.NumAtrs()
			if request.NumAtrs() > 0 {
				numAtrs = request.NumAtrs()
			}
			if err = context.SetTransactionContext(request.Type(), request.TxImplicit(),
				request.TxTimeout(), this.TxTimeout(), atrCollection, numAtrs,
				request.TxData()); err != nil {
				request.Fail(err)
				request.Failed(this)
				return
			}
			if request.OriginalScanConsistency() == datastore.NOT_SET && context.TxContext() != nil {
				request.SetScanConsistency(context.TxContext().TxScanConsistency())
			}
		}
	}

	if request.State() == FATAL {
		request.Failed(this)
		return
	}

	if request.AutoExecute() == value.TRUE {
		prepared, err = this.getAutoExecutePrepared(request, prepared, context)

		if err != nil {
			request.Fail(err)
			request.Failed(this)
			return
		}

		// make a read-only check for the prepared statement to be executed
		if (this.readonly || value.ToBool(request.Readonly())) && (prepared != nil && !prepared.Readonly()) {
			request.Fail(errors.NewServiceErrorReadonly())
			request.Failed(this)
			return
		}
	}

	memoryQuota := request.MemoryQuota()

	// never allow request side quota to be higher than
	// server side quota
	if this.memoryQuota > 0 && (this.memoryQuota < memoryQuota || memoryQuota == 0) {
		memoryQuota = this.memoryQuota
	}
	context.SetMemoryQuota(memoryQuota)

	if tenant.IsServerless() {
		context.SetMemorySession(tenant.Register(context.TenantCtx()))
	} else if memoryQuota > 0 || memory.Quota() > 0 {
		context.SetMemorySession(memory.Register())
	}

	context.SetIsPrepared(request.Prepared() != nil)
	build := util.Now()
	operator, er := execution.Build(prepared, context)
	if er != nil {
		// NewError returns its error argument if it is an Error object
		request.Fail(errors.NewError(er, ""))
	}

	operator.SetRoot(context)
	request.SetTimings(operator)
	request.Output().AddPhaseTime(execution.INSTANTIATE, util.Now().Sub(build))

	if request.State() == FATAL {
		request.Failed(this)
		return
	}

	if !request.Alive() {
		request.Fail(errors.NewServiceNoClientError())
		request.Failed(this)
		return
	}

	timeout := this.RequestTimeout(request.Timeout())
	timeout = context.AdjustTimeout(timeout, request.Type(), request.IsPrepare())
	if timeout != request.Timeout() {
		request.SetTimeout(timeout)
	}

	now := time.Now()
	if timeout > 0 {
		request.SetTimer(time.AfterFunc(timeout, func() { request.Expire(TIMEOUT, timeout) }))
		context.SetReqDeadline(now.Add(timeout))
	} else {
		context.SetReqDeadline(time.Time{})
	}

	context.Infof("Executing request")
	request.NotifyStop(operator)
	request.SetExecTime(now)
	operator.RunOnce(context, nil)

	request.Execute(this, context, request.Type(), prepared.Signature(), request.Type() == "START_TRANSACTION")
}

func (this *Server) getPrepared(request Request, context *execution.Context) (*plan.Prepared, errors.Error) {
	var autoPrepare bool
	var name string

	prepared := request.Prepared()

	// if Auto Prepare is on, see if we have it already
	if request.AutoPrepare() == value.NONE {
		autoPrepare = this.autoPrepare
	} else {
		autoPrepare = request.AutoPrepare() == value.TRUE
	}

	namedArgs := request.NamedArgs()
	positionalArgs := request.PositionalArgs()
	dsContext := context
	autoExecute := request.AutoExecute() == value.TRUE
	if len(namedArgs) > 0 || len(positionalArgs) > 0 || autoExecute {
		autoPrepare = false
	}

	if prepared == nil && autoPrepare {

		// no datastore context for autoprepare
		var prepContext planner.PrepareContext
		planner.NewPrepareContext(&prepContext, request.Id().String(), request.QueryContext(), nil, nil,
			request.IndexApiVersion(), request.FeatureControls(), request.UseFts(),
			request.UseCBO(), context.Optimizer(), context.DeltaKeyspaces(), dsContext, true)

		name = prepareds.GetAutoPrepareName(request.Statement(), &prepContext)
		if name != "" {
			prepared = prepareds.GetAutoPreparePlan(name, request.Statement(),
				request.Namespace(), &prepContext)
			request.SetPrepared(prepared)
		} else {
			autoPrepare = false
		}
	}

	if prepared == nil {

		var stmt algebra.Statement
		var err error
		if nlstmt := request.NaturalStatement(); nlstmt != nil {
			stmt = nlstmt
		} else {
			parse := util.Now()
			stmt, err = n1ql.ParseStatement2(request.Statement(), context.Namespace(), request.QueryContext(), context)
			request.Output().AddPhaseTime(execution.PARSE, util.Now().Sub(parse))
			if err != nil {
				return nil, errors.NewParseSyntaxError(err, "")
			}
		}

		isPrepare := false
		if _, ok := stmt.(*algebra.Prepare); ok {
			isPrepare = true
		} else {
			autoExecute = false
			request.SetAutoExecute(value.FALSE)
		}

		var stype string
		var allow bool
		switch estmt := stmt.(type) {
		case *algebra.Explain:
			stype = estmt.Statement().Type()
			allow = true
		case *algebra.Advise:
			stype = estmt.Statement().Type()
			allow = true
		default:
			stype = stmt.Type()
			allow = isPrepare && !autoExecute
		}

		if ok, msg := transactions.IsValidStatement(request.TxId(), stype, request.TxImplicit(), allow); !ok {
			return nil, errors.NewTranStatementNotSupportedError(stype, msg)
		}

		if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
			return nil, errors.NewRewriteError(err, "")
		}

		semChecker := semantics.GetSemChecker(stmt.Type(), request.TxId() != "")
		_, err = stmt.Accept(semChecker)
		if err != nil {
			return nil, errors.NewSemanticsError(err, "")
		}

		prep := util.Now()

		// MB-24871: do not replace named/positional parameters with value for prepare statement
		// no credentials for prepared statements
		if isPrepare {
			namedArgs = nil
			positionalArgs = nil
		}

		var prepContext planner.PrepareContext
		planner.NewPrepareContext(&prepContext, request.Id().String(), request.QueryContext(), namedArgs,
			positionalArgs, request.IndexApiVersion(), request.FeatureControls(), request.UseFts(),
			request.UseCBO(), context.Optimizer(), context.DeltaKeyspaces(), dsContext, isPrepare)
		if stmt, ok := stmt.(*algebra.Advise); ok {
			stmt.SetContext(execution.NewOpContext(context))
		}

		var subTimes map[string]time.Duration
		prepared, err, subTimes = planner.BuildPrepared(stmt, this.datastore, this.systemstore, context.Namespace(),
			autoExecute, !autoExecute, &prepContext)
		if subTimes != nil {
			for k, v := range subTimes {
				p := execution.PhaseByName("plan." + k)
				if p != execution.PHASES {
					request.Output().AddPhaseTime(p, v)
				}
			}
		}
		request.Output().AddPhaseTime(execution.PLAN, util.Now().Sub(prep))
		if err != nil {
			return nil, errors.NewPlanError(err, "")
		}

		// set the time the plan was generated
		prepared.SetPreparedTime(prep.ToTime())

		// EXECUTE doesn't get a plan. Get the plan from the cache.
		switch stmt.Type() {
		case "EXECUTE":
			var err errors.Error
			exec, _ := stmt.(*algebra.Execute)
			if prepared, err = this.getPreparedByName(exec.Prepared(), request, context); err != nil {
				return nil, err
			}

			request.SetType(prepared.Type())
			if ok, msg := transactions.IsValidStatement(request.TxId(), request.Type(), request.TxImplicit(), false); !ok {
				return nil, errors.NewTranStatementNotSupportedError(request.Type(), msg)
			}

			if err = this.setUsingArgs(exec, positionalArgs, namedArgs, request, context); err != nil {
				return nil, err
			}

		default:

			if isPrepare && autoExecute {
				request.SetIsPrepare(false)
				request.SetType(stmt.Type())
			} else {

				// even though this is not a prepared statement, add the
				// text for the benefit of context.Recover(): we can
				// output the text in case of crashes
				prepared.SetText(request.Statement())

				// set the type for all statements bar prepare
				// (doing otherwise would have accounting track prepares
				// as if they were executions)
				if isPrepare {
					request.SetIsPrepare(true)
				} else {
					request.SetType(stmt.Type())

					// if autoPrepare is on and this statement is eligible
					// save it for the benefit of others
					if autoPrepare {
						prepared.SetName(name)
						prepared.SetIndexApiVersion(request.IndexApiVersion())
						prepared.SetFeatureControls(request.FeatureControls())
						prepared.SetNamespace(request.Namespace())
						prepared.SetQueryContext(request.QueryContext())
						prepared.SetUseFts(request.UseFts())
						prepared.SetUseCBO(request.UseCBO())
						prepared.SetUsers(datastore.CredsString(request.Credentials()))
						prepared.SetUserAgent(request.UserAgent())
						prepared.SetRemoteAddr(request.RemoteAddr())

						// trigger prepare metrics recording
						if prepareds.AddAutoPreparePlan(stmt, prepared) {
							request.SetPrepared(prepared)
						}
					}
				}
			}
		}
	} else {
		if (request.TxId() != "" || request.TxImplicit()) && !autoPrepare {
			var err errors.Error
			if prepared, err = this.getPreparedByName(prepared.Name(), request, context); err != nil {
				return nil, err
			}
		}

		// ditto
		request.SetType(prepared.Type())
		if ok, msg := transactions.IsValidStatement(request.TxId(), request.Type(), request.TxImplicit(), true); !ok {
			return nil, errors.NewTranStatementNotSupportedError(request.Type(), msg)
		}
	}

	// Check if query should allow read from replica
	// Read From Replica can only be allowed if the all query nodes in the cluster possess this capability

	// Read from replica is by default what the Node Level Param is. But..
	// If Node Level Param is False - cannot be overridden at request level
	// If Node Level Param is True -  can be set to False at request level
	// If Node Level Param is None / Unset - can be set to True or False at request level
	// But if both Node Level and Request Level Params are Unset - read from replica is set to False
	useReplica := value.FALSE

	if (request.Type() == "SELECT" || request.Type() == "EXECUTE_FUNCTION") && this.useReplica != value.FALSE {

		if request.UseReplica() == value.NONE {
			useReplica = this.useReplica
		} else {
			useReplica = request.UseReplica()
		}

		if useReplica == value.NONE {
			useReplica = value.FALSE
		} else if useReplica == value.TRUE {
			// check if cluster has readFromReplica enabled
			if !distributed.RemoteAccess().Enabled(distributed.READ_FROM_REPLICA) {
				useReplica = value.FALSE
			}
		}
	}

	request.SetUseReplica(useReplica)
	context.SetStmtType(request.Type())
	context.SetUseReplica(value.ToBool(useReplica))

	context.Infof("Read from replicas permitted: %v", context.UseReplica())

	logging.Tracea(func() string {
		var pl plan.Operator = prepared
		explain, err := json.Marshal(pl)
		if err != nil {
			return fmt.Sprintf("Explain: %s error: %v", request.Id().String(), err)
		}
		return fmt.Sprintf("Explain: %s <ud>%v</ud>", request.Id().String(), string(explain))
	}, context)

	return prepared, nil
}

func (this *Server) setUsingArgs(exec *algebra.Execute, positionalArgs value.Values, namedArgs map[string]value.Value,
	request Request, context *execution.Context) errors.Error {

	usingArgs := exec.Using()
	if usingArgs == nil {
		return nil
	}

	// USING clause and REST API parameters can't go together
	if namedArgs != nil || positionalArgs != nil {
		return errors.NewExecutionParameterError("cannot have both USING clause and request parameters")
	}

	argsValue := usingArgs.Value()
	if argsValue == nil {
		return errors.NewExecutionParameterError("USING clause does not evaluate to static values")
	}

	actualValue := argsValue.Actual()
	switch actualValue := actualValue.(type) {
	case map[string]interface{}:
		newArgs := make(map[string]value.Value, len(actualValue))
		for n, v := range actualValue {
			newArgs[n] = value.NewValue(v)
		}
		request.SetNamedArgs(newArgs)
		context.SetNamedArgs(newArgs)
	case []interface{}:
		newArgs := make([]value.Value, len(actualValue))
		for n, v := range actualValue {
			newArgs[n] = value.NewValue(v)
		}
		request.SetPositionalArgs(newArgs)
		context.SetPositionalArgs(newArgs)
	default:

		// this never happens, but for completeness
		return errors.NewExecutionParameterError("unexpected value type")
	}

	return nil
}

func (this *Server) getAutoExecutePrepared(request Request, prepared *plan.Prepared,
	context *execution.Context) (*plan.Prepared, errors.Error) {
	res, _, er := context.ExecutePrepared(prepared, false, request.NamedArgs(), request.PositionalArgs(), "", false, "")
	if er == nil {
		var name string
		actual, ok := res.Actual().([]interface{})
		if ok && len(actual) > 0 {
			if fields, ok := actual[0].(map[string]interface{}); ok {
				name, _ = fields["name"].(string)
			}
		}

		if name != "" {
			var reprepTime time.Duration

			prepared, er = prepareds.GetPreparedWithContext(name, request.QueryContext(),
				context.DeltaKeyspaces(), prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY,
				&reprepTime, context)
			if reprepTime > 0 {
				request.Output().AddPhaseTime(execution.REPREPARE, reprepTime)
			}
		} else {
			er = errors.NewUnrecognizedPreparedError(fmt.Errorf("auto_execute did not produce a prepared statement"))
		}
	}

	if er != nil {
		// NewError returns its error argument if it is an Error object
		return prepared, errors.NewError(er, "")
	}

	request.SetPrepared(prepared)
	context.SetPrepared(prepared)
	return prepared, nil
}

func (this *Server) getPreparedByName(prepareName string, request Request, context *execution.Context) (
	*plan.Prepared, errors.Error) {

	var reprepTime time.Duration

	prepared, err := prepareds.GetPreparedWithContext(prepareName, request.QueryContext(),
		context.DeltaKeyspaces(), prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY,
		&reprepTime, context)
	if reprepTime > 0 {
		request.Output().AddPhaseTime(execution.REPREPARE, reprepTime)
	}
	if err != nil {
		return nil, err
	}

	request.SetPrepared(prepared)

	// when executing prepared statements, we set the type to that of the prepared statement
	request.SetType(prepared.Type())

	return prepared, nil
}

func (this *Server) GCPercent() int {
	return this.gcpercent
}

func (this *Server) SetGCPercent(gcpercent int) error {
	if gcpercent < 25 || gcpercent > 300 {
		return fmt.Errorf("gcpercent (%v) outside permitted range (25-300)", gcpercent)
	}
	if this.gcpercent != gcpercent {
		logging.Infof("Changing GC percent from %d to %d", this.gcpercent, gcpercent)
		this.gcpercent = gcpercent
	}
	debug.SetGCPercent(this.gcpercent)
	return nil
}

func (this *Server) ShuttingDown() bool {
	this.RLock()
	rv := this.shutdown != _SERVER_RUNNING
	this.RUnlock()
	return rv
}

func (this *Server) ShutDown() bool {
	this.RLock()
	rv := this.shutdown == _SERVER_SHUTDOWN
	this.RUnlock()
	return rv
}

func (this *Server) InitiateShutdown(timeout time.Duration, reason string) {
	var what string
	this.Lock()
	if this.shutdown == _SERVER_RUNNING {
		this.shutdown = _REQUESTED
		if util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_PART_GRACEFUL) {
			this.shutdownStart = time.Now()
			what = "Partial graceful"
		} else {
			this.shutdownStart = time.Time{}
			what = "Graceful"
		}
		this.Unlock()
		logging.Infof("%s shutdown initiated in response to %s.", what, reason)
		go this.monitorShutdown(timeout)
	} else {
		this.Unlock()
	}
}

func (this *Server) CancelShutdown() {
	var what string
	log := false
	this.Lock()
	if this.shutdown != _SERVER_RUNNING {
		this.shutdown = _SERVER_RUNNING
		if this.shutdownStart.IsZero() {
			what = "Graceful"
		} else {
			what = "Partial graceful"
			this.shutdownStart = time.Time{}
		}
		log = true
	}
	this.Unlock()
	if log {
		logging.Infof("%s shutdown cancelled.", what)
	}
}

const _SHUTDOWN_WAIT_LIMIT = 10 * time.Minute

func (this *Server) InitiateShutdownAndWait(reason string) {
	this.InitiateShutdown(_SHUTDOWN_WAIT_LIMIT, reason)
	for this.ShuttingDown() && !this.ShutDown() {
		time.Sleep(time.Second)
	}
}

const (
	_CHECK_INTERVAL  = 100 * time.Millisecond
	_REPORT_INTERVAL = 10 * time.Second
	_FFDC_THRESHOLD  = 30 * time.Minute
)

func RunningRequests(before time.Time) int {
	count := 0
	ActiveRequestsForEach(func(id string, request Request) bool {
		if request.State() == RUNNING || request.State() == SUBMITTED || request.State() == PREPROCESSING {
			if before.IsZero() || request.RequestTime().Before(before) {
				count++
			}
		}
		return true
	}, nil)
	return count
}

func (this *Server) monitorShutdown(timeout time.Duration) {
	// wait for existing requests to complete
	ar := RunningRequests(this.shutdownStart)
	at := transactions.CountValidTransContextBefore(this.shutdownStart)
	if ar > 0 || at > 0 {
		logging.Infof("Shutdown: Waiting for %v active request(s) and %v active transaction(s) to complete.", ar, at)
		start := time.Now()
		reportStart := start
		ffdcStart := start
		for this.ShuttingDown() {
			ar = RunningRequests(this.shutdownStart)
			at = transactions.CountValidTransContextBefore(this.shutdownStart)
			if ar == 0 && at == 0 {
				logging.Infof("Shutdown: All monitored requests and transactions completed.")
				break
			}
			now := time.Now()
			if now.Sub(reportStart) > _REPORT_INTERVAL {
				logging.Infof("Shutdown: Waiting for %v active request(s) and %v active transaction(s) to complete.", ar, at)
				reportStart = now
			}
			if now.Sub(ffdcStart) > _FFDC_THRESHOLD {
				ffdc.Capture(ffdc.Shutdown)
				ffdcStart = now
			}
			if timeout > 0 && now.Sub(start) > timeout {
				logging.Infof("Shutdown: Timeout (%v) exceeded.", timeout)
				break
			}
			time.Sleep(_CHECK_INTERVAL)
		}
	} else {
		logging.Infof("Shutdown: No active requests or transactions to monitor.")
	}

	if this.ShuttingDown() {
		// only mark the server as now down; the master manager will use this state to report that this node is no longer active
		this.Lock()
		this.shutdown = _SERVER_SHUTDOWN
		this.Unlock()
		// after this point we have to trust the external monitoring will shut the process down eventually.  We cannot exit
		// ourselves if it is still monitoring us as it will cause issues with running monitoring operations.  If the shutdown
		// isn't initiated by something that will kill us off eventually, we will end up just sitting there unable to do anything.
		logging.Infof("Shutdown: Monitor complete.")
	} else {
		logging.Infof("Shutdown: Monitor detected shutdown was cancelled.")
	}
}

// API for tracking server options
type ServerOptions interface {
	Controls() bool
	Profile() Profile
}

var options ServerOptions

func SetOptions(o ServerOptions) {
	options = o
}

func GetControls() bool {
	return options.Controls()
}

func GetProfile() Profile {
	return options.Profile()
}

// FIXME should the IPv6 / host name and port code be in util?
func IsIPv6() string {
	return _IPv6val
}

func IsIPv4() string {
	return _IPv4val
}

// Return the correct address for localhost depending on if
// IPv4 or IPv6
func GetIP(is_url bool) string {
	if _IPv6 {
		if is_url {
			return "[::1]"
		} else {
			return "::1"
		}
	}
	return "127.0.0.1"
}

func checkIPVals(val, vtype string) error {
	if val != TCP_OFF && val != TCP_REQ && val != TCP_OPT {
		return fmt.Errorf("%v flag values take required optional or off values. Current value %v is invalid.", vtype, val)
	}
	return nil
}

func SetIP(vIpv4, vIpv6 string, localhost6, listener bool) error {

	err := checkIPVals(vIpv4, "IPv4")
	if err != nil {
		return err
	}

	err = checkIPVals(vIpv6, "IPv6")
	if err != nil {
		return err
	}

	// Set Ipv6 or Ipv4

	//ns_server will pass both command line options always to the service. In case, through
	// a bug in code, both options are "off"/"optional" the service should fail to start.
	// We never expect both address families to be "off"/"optional",
	// we expect at least one to be required.
	if vIpv4 != TCP_REQ && vIpv6 != TCP_REQ {
		return fmt.Errorf("Incorrect IPv4 and IPv6 flag value. Atleast one value must be required.")
	}

	_IPv6val = vIpv6
	_IPv4val = vIpv4

	if listener {
		util.IPv6 = true
	}

	if localhost6 {
		_IPv6 = true
	}

	return nil
}

// return true, nil for IPv6 address and false for IPv4.
func CheckURL(store, urltype string) (bool, error) {
	ipUrl, err := url.Parse(store)
	if err != nil {
		return false, fmt.Errorf("Incorrect input url format for %v.", urltype)
	}

	ip := net.ParseIP(ipUrl.Hostname())
	if ip == nil {
		return false, fmt.Errorf("Incorrect input url format for %v.", urltype)
	}

	if ip.To4() == nil {
		return true, nil
	} else {
		return false, nil
	}
}

// This needs to support both IPv4 and IPv6
// The prev version of impl for this function assumed
// that node is always ip:port. It should not have a protocol component.
func HostNameandPort(node string) (host, port string) {
	if len(node) == 0 {
		return "", ""
	}
	tokens := []string{}

	// it's an IPv6 with port
	if _IPv6 && node[0] == '[' {

		// Then the url should be of the form [::1]:8091
		tokens = strings.Split(node, "]:")
		host = strings.Replace(tokens[0], "[", "", 1)
	} else {

		// IPv4 with or without port
		// FQDN with or without port
		// IPv6 without port
		tokens = strings.Split(node, ":")

		// if we have more than two tokens, it was IPv6 without port
		if len(tokens) > 2 {
			tokens = []string{node}
		}
		host = tokens[0]
	}

	if len(tokens) == 2 {
		port = tokens[1]
	} else {
		port = ""
	}

	return
}

const (
	_SERVICE_PERCENT_LIMIT   = 100
	_SERVICE_OVERHEAD_FACTOR = 2 // 2 requests per servicer is normal
	_MEMORY_PERCENT          = 100
	_MEMORY_QUOTA            = float64(0.5) // 50% of system memory
	_DEF_MEMORY_USAGE        = uint64(1)
)

func (this *Server) LoadFactor() int {
	return getQsLoadFactor()
}

func (this *Server) loadFactor(refresh bool) int {
	// consider one request per servicer in queue is normal
	servicers := util.MinInt(int(this.ServicerUsage()/_SERVICE_OVERHEAD_FACTOR), _SERVICE_PERCENT_LIMIT)
	cpu := int(this.CpuUsage(refresh))
	memory, _, _ := this.MemoryUsage(refresh)
	// max of all of them
	return util.MaxInt(util.MaxInt(servicers, cpu), util.MinInt(int(memory), _MEMORY_PERCENT))

}

func (this *Server) ServicerUsage() int {
	reqplus := this.plusQueue.load(this.txQueueCount())
	unbound := this.unboundQueue.load(0)
	if int(reqplus/_SERVICE_OVERHEAD_FACTOR) >= _SERVICE_PERCENT_LIMIT || reqplus > unbound {
		// request_plus is higher than unbounded, so request plus intensive
		return reqplus
	} else {
		// unbounded intensive
		return unbound
	}
}

func (this *Server) CpuUsage(refresh bool) float64 {
	// get process cpu usage
	cpu, _, _, _, _, _ := system.GetSystemStats(nil, refresh, false)
	if cpu == 0 {
		// no data, calculate alternative approach
		newUtime, newStime := util.CpuTimes()
		totalTime := newUtime + newStime
		now := time.Now()
		dur := now.Sub(this.lastNow)
		if dur > time.Second {
			cpu = 100 * (float64(totalTime-this.lastTotalTime) / float64(dur))
			this.lastTotalTime = totalTime
			this.lastNow = now
			this.lastCpuPercent = cpu
		} else {
			cpu = this.lastCpuPercent
		}
	} else {
		this.lastCpuPercent = cpu
	}

	// cpu percent per core
	return util.RoundPlaces(cpu/float64(util.NumCPU()), 2)
}

func (this *Server) MemoryUsage(refresh bool) (uint64, uint64, uint64) {
	// get go runtime memory stats
	ms := this.MemoryStats(refresh)
	mem_used := ms.HeapInuse + ms.HeapIdle - ms.HeapReleased + ms.GCSys
	mem_quota := memory.NodeQuota() * util.MiB
	mup := uint64(_DEF_MEMORY_USAGE)
	musp := uint64(_DEF_MEMORY_USAGE)
	if mem_quota > 0 {
		mup = uint64((mem_used * 100) / mem_quota)
		if !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_USE_SYS_FREE_MEM) {
			// report remaining quota relative to process system memory use
			sys := ms.HeapSys - ms.HeapReleased
			if sys < mem_quota {
				musp = ((mem_quota - sys) * 100) / mem_quota
			}
			return mup, musp, ms.LastGC
		}
	}

	// get system memory info
	_, _, total, _, free, _ := system.GetSystemStats(nil, refresh, false)
	if total > 0 {
		if mem_quota == 0 {
			// no node quota; use 50% of the system memory as the Query memory quota
			quota := uint64(float64(total) * _MEMORY_QUOTA)
			mup = uint64((mem_used * 100) / quota)
			if mup < _DEF_MEMORY_USAGE {
				mup = _DEF_MEMORY_USAGE
			}
		}
		// report overall system free memory
		musp = (free * 100) / total
	}

	return mup, musp, ms.LastGC
}

// extract go runtime stats
func (this *Server) MemoryStats(refresh bool) (ms runtime.MemStats) {
	if refresh {
		runtime.ReadMemStats(&ms)
		this.Lock()
		this.memoryStats = ms
		this.Unlock()
	} else {
		this.RLock()
		ms = this.memoryStats
		this.RUnlock()
	}
	return
}

func (this *Server) UseReplica() value.Tristate {
	return this.useReplica
}

func (this *Server) SetUseReplica(useReplica value.Tristate) {
	this.useReplica = useReplica
}

func (this *Server) UseReplicaToString() string {
	return value.TRISTATE_NAMES[this.useReplica]
}

func (this *Server) SetN1qlFeatureControl(control uint64) {
	this.admissionsLock.Lock()
	prev := util.SetN1qlFeatureControl(control)
	prevAdmissionControlEnabled := !util.IsFeatureEnabled(prev, util.N1QL_UNRESTRICTED_ADMISSION_AND_ACTIVITY)
	nowAdmissionControlEnabled := !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_UNRESTRICTED_ADMISSION_AND_ACTIVITY)

	// Admission control was previously disabled, now it is enabled
	if !prevAdmissionControlEnabled && nowAdmissionControlEnabled {
		// If no admissions routine is running, start it.
		if this.admissionsRoutineState == _INACTIVE {
			this.admissionsLock.Unlock()
			go this.admissions()
			return
		} else if this.admissionsRoutineState == _ACTIVE {
			// If the routine is running, it could be in the process of shutting down. Indicate that the routine must be restarted
			this.admissionsRoutineState = _RESTART
		}
	}

	// If admission control is disabled, the admissions routine will stop itself.

	this.admissionsLock.Unlock()

}
