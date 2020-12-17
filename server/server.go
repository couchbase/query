//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package server

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/rewrite"
	"github.com/couchbase/query/semantics"
	queryMetakv "github.com/couchbase/query/server/settings/couchbase"
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

var _IPv6 = false

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
	servicers int
	size      int32
	runCnt    int32
	queueCnt  int32
	head      int32
	tail      int32
	queue     []waitEntry
	mutex     sync.RWMutex
}

type txRunQueues struct {
	mutex    sync.RWMutex
	queueCnt int32
	size     int32
	txQueues map[string]*runQueue
}

type Server struct {
	// due to alignment issues on x86 platforms these atomic
	// variables need to right at the beginning of the structure
	maxParallelism atomic.AlignedInt64
	keepAlive      atomic.AlignedInt64
	requestSize    atomic.AlignedInt64

	sync.RWMutex
	unboundQueue      runQueue
	plusQueue         runQueue
	transactionQueues txRunQueues
	datastore         datastore.Datastore
	systemstore       datastore.Systemstore
	configstore       clustering.ConfigurationStore
	acctstore         accounting.AccountingStore
	namespace         string
	readonly          bool
	timeout           time.Duration
	txTimeout         time.Duration
	signature         bool
	metrics           bool
	memprofile        string
	cpuprofile        string
	enterprise        bool
	pretty            bool
	srvprofile        Profile
	srvcontrols       bool
	whitelist         map[string]interface{}
	autoPrepare       bool
	memoryQuota       uint64
	atrCollection     string
	numAtrs           int
	settingsCallback  func(string, interface{})
}

// Default and min Keep Alive Length

const KEEP_ALIVE_DEFAULT = 1024 * 16
const KEEP_ALIVE_MIN = 1024

func NewServer(store datastore.Datastore, sys datastore.Systemstore, config clustering.ConfigurationStore,
	acctng accounting.AccountingStore, namespace string, readonly bool,
	requestsCap, plusRequestsCap int, servicers, plusServicers, maxParallelism int,
	timeout time.Duration, signature, metrics, enterprise, pretty bool,
	srvprofile Profile, srvcontrols bool) (*Server, errors.Error) {
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

	rv.unboundQueue.servicers = servicers
	rv.plusQueue.servicers = plusServicers
	newRunQueue(&rv.unboundQueue, requestsCap, false)
	newRunQueue(&rv.plusQueue, plusRequestsCap, false)
	newTxRunQueues(&rv.transactionQueues, plusRequestsCap, _TX_QUEUE_SIZE)
	store.SetLogLevel(logging.LogLevel())
	rv.SetMaxParallelism(maxParallelism)
	rv.SetNumAtrs(datastore.DEF_NUMATRS)

	// set default values
	rv.SetMaxIndexAPI(datastore.INDEX_API_MAX)
	if rv.enterprise {
		util.SetN1qlFeatureControl(util.DEF_N1QL_FEAT_CTRL)
		util.SetUseCBO(util.DEF_USE_CBO)
	} else {
		util.SetN1qlFeatureControl(util.DEF_N1QL_FEAT_CTRL | util.CE_N1QL_FEAT_CTRL)
		util.SetUseCBO(util.CE_USE_CBO)
	}

	// Setup callback function for metakv settings changes
	callb := func(cfg queryMetakv.Config) {
		logging.Infof("Settings notifier from metakv")

		// SetParamValuesForAll accepts a full-set or subset of global configuration
		// and updates those fields.
		SetParamValuesForAll(cfg, rv)
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

func (this *Server) SetWhitelist(val map[string]interface{}) {
	this.whitelist = val
}

func (this *Server) GetWhitelist() map[string]interface{} {
	return this.whitelist
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
	numProcs := runtime.NumCPU()
	if os.Getenv("GOMAXPROCS") != "" {
		numProcs = runtime.GOMAXPROCS(0)
	}

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
		logging.Errorp("Cannot start cpu profiler", logging.Pair{"error", err})
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
	return logging.LogLevel().String()
}

func (this *Server) SetLogLevel(level string) {
	lvl, ok := logging.ParseLevel(level)
	if !ok {
		logging.Errorp("SetLogLevel: unrecognized level", logging.Pair{"level", level})
		return
	}
	if this.datastore != nil {
		this.datastore.SetLogLevel(lvl)
	}
	logging.SetLevel(lvl)
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
	this.unboundQueue.servicers = servicers
	this.Unlock()
}

func (this *Server) PlusServicers() int {
	return this.plusQueue.servicers
}

func (this *Server) SetPlusServicers(plusServicers int) {
	this.Lock()
	this.plusQueue.servicers = plusServicers
	this.Unlock()
}

func (this *Server) Timeout() time.Duration {
	return this.timeout
}

func (this *Server) SetTimeout(timeout time.Duration) {
	this.timeout = timeout
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

func (this *Server) ServiceRequest(request Request) bool {
	if !this.setupRequestContext(request) {
		request.Failed(this)
		return true // so that StatusServiceUnavailable will not return
	}

	return this.handleRequest(request, &this.unboundQueue)
}

func (this *Server) PlusServiceRequest(request Request) bool {
	if !this.setupRequestContext(request) {
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

	optimizer := getNewOptimizer()
	maxParallelism := request.MaxParallelism()
	if maxParallelism <= 0 || maxParallelism > this.MaxParallelism() {
		maxParallelism = this.MaxParallelism()
	}

	context := execution.NewContext(request.Id().String(), this.datastore, this.systemstore, namespace,
		this.readonly, maxParallelism, request.ScanCap(), request.PipelineCap(), request.PipelineBatch(),
		request.NamedArgs(), request.PositionalArgs(), request.Credentials(), request.ScanConsistency(),
		request.ScanVectorSource(), request.Output(), nil, request.IndexApiVersion(), request.FeatureControls(),
		request.QueryContext(), request.UseFts(), request.UseCBO(), optimizer, request.KvTimeout())
	context.SetWhitelist(this.whitelist)
	context.SetDurability(request.DurabilityLevel(), request.DurabilityTimeout())
	context.SetScanConsistency(request.ScanConsistency(), request.OriginalScanConsistency())

	if request.TxId() != "" {
		err := context.SetTransactionInfo(request.TxId(), request.TxStmtNum())
		if err == nil {
			err = context.TxContext().TxValid()
		}

		if err != nil {
			request.Fail(err)
			return false
		}

		if request.OriginalScanConsistency() == datastore.NOT_SET {
			request.SetScanConsistency(context.TxContext().TxScanConsistency())
		}
		request.SetDurabilityLevel(context.TxContext().TxDurabilityLevel())
		request.SetDurabilityTimeout(context.TxContext().TxDurabilityTimeout())
	}
	request.SetExecutionContext(context)
	return true
}

func (this *Server) handleRequest(request Request, queue *runQueue) bool {
	if !queue.enqueue(request) {
		return false
	}

	this.serviceRequest(request) // service

	queue.dequeue()

	return true
}

func (this *Server) handlePlusRequest(request Request, queue *runQueue, transactionQueues *txRunQueues) bool {
	if !queue.enqueue(request) {
		return false
	}

	dequeue := true
	if request.TxId() != "" {
		err := this.handlePreTxRequest(request, queue, transactionQueues)
		if err != nil {
			request.Fail(err)
			request.Failed(this) // don't return
		} else {
			this.serviceRequest(request) // service
			dequeue = this.handlePostTxRequest(request, queue, transactionQueues)
		}
	} else {
		this.serviceRequest(request) // service
	}

	if dequeue {
		queue.dequeue()
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
			newRunQueue(txQueue, int(transactionQueues.size), true)
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

func newRunQueue(q *runQueue, num int, txflag bool) {
	q.size = int32(num)
	q.queue = make([]waitEntry, num)
	if !txflag {
		go q.checkWaiters()
	}
}

func newTxRunQueues(q *txRunQueues, nqueues, num int) {
	q.size = int32(num)
	q.txQueues = make(map[string]*runQueue, nqueues)
}

func (this *runQueue) enqueue(request Request) bool {
	runCnt := int(atomic.AddInt32(&this.runCnt, 1))

	// if servicers exceeded, reserve a spot in the queue
	if runCnt > this.servicers {

		atomic.AddInt32(&this.runCnt, -1)
		queueCnt := atomic.AddInt32(&this.queueCnt, 1)

		// rats! queue full, can't handle this request
		if queueCnt >= this.size {
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
	entry := atomic.AddInt32(&this.tail, 1) % this.size

	// set up the entry and the request
	request.setSleep()
	this.queue[entry].request = request

	if txQueueMutex != nil {
		txQueueMutex.Unlock()
	}

	if txQueuesMutex != nil {
		txQueuesMutex.Unlock()
	}

	// atomically set the state to full
	// if we succeed, sleep
	if atomic.CompareAndSwapUint32(&this.queue[entry].state, _WAIT_EMPTY, _WAIT_FULL) {
		request.sleep()
	} else {

		// if the waker got there first, continue
		this.queue[entry].request = nil
		this.queue[entry].state = _WAIT_EMPTY
		request.release()
	}
}

func (this *runQueue) releaseRequest(txQueueMutex, txQueuesMutex *sync.RWMutex) {

	// get the next available entry
	entry := atomic.AddInt32(&this.head, 1) % this.size

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
		this.queue[entry].state = _WAIT_EMPTY
		request.release()
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
			if runCnt > 0 || queueCnt == 0 {
				break
			}
			logging.Infof("request scheduler emptying queue: %v requests", queueCnt)

			// can we pull one?
			queueCnt = atomic.AddInt32(&this.queueCnt, -1)
			if this.queueCnt < 0 {

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
	return 100 * (int(this.runCnt) + int(this.queueCnt) + txqueueCnt) / this.servicers
}

func (this *Server) Load() int {
	return this.plusQueue.load(this.txQueueCount()) + this.unboundQueue.load(0)
}

func (this *Server) txQueueCount() int {
	return int(this.transactionQueues.queueCnt)
}

func (this *Server) serviceRequest(request Request) {
	defer func() {
		err := recover()
		if err != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("panic: %v ", err)
			logging.Severef("request text: <ud>%v</ud>", request.Statement())
			logging.Severef("query context: <ud>%v</ud>", request.QueryContext())
			logging.Severef("stack: %v", s)
			os.Stderr.WriteString(s)
			os.Stderr.Sync()
			request.Fail(errors.NewExecutionPanicError(nil, fmt.Sprintf("Panic: %v", err)))
			request.Failed(this)
		}
	}()

	request.Servicing()

	context := request.ExecutionContext()
	if request.TxId() != "" {
		err := context.TxContext().TxValid()
		if err == nil {
			atrCollection := this.AtrCollection()
			if request.AtrCollection() != "" {
				atrCollection = request.AtrCollection()
			}
			numAtrs := this.NumAtrs()
			if request.NumAtrs() > 0 {
				numAtrs = request.NumAtrs()
			}
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
	}

	prepared, err := this.getPrepared(request, context)
	if err != nil {
		request.Fail(err)
	} else {
		context.SetPrepared(prepared)
		if (this.readonly || value.ToBool(request.Readonly())) &&
			(prepared != nil && !prepared.Readonly()) {
			request.Fail(errors.NewServiceErrorReadonly("The server or request is read-only" +
				" and cannot accept this write statement."))
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
	}

	memoryQuota := request.MemoryQuota()

	// never allow request side quota to be higher than
	// server side quota
	if this.memoryQuota > 0 && (this.memoryQuota < memoryQuota || memoryQuota == 0) {
		memoryQuota = this.memoryQuota
	}
	context.SetMemoryQuota(memoryQuota)

	context.SetIsPrepared(request.Prepared() != nil)
	build := time.Now()
	operator, er := execution.Build(prepared, context)
	if er != nil {
		error, ok := er.(errors.Error)
		if ok {
			request.Fail(error)
		} else {
			request.Fail(errors.NewError(er, ""))
		}
	}

	operator.SetRoot()
	request.SetTimings(operator)
	request.Output().AddPhaseTime(execution.INSTANTIATE, time.Since(build))

	if request.State() == FATAL {
		request.Failed(this)
		return
	}

	timeout := request.Timeout()

	// never allow request side timeout to be higher than
	// server side timeout
	if this.timeout > 0 && (this.timeout < timeout || timeout <= 0) {
		timeout = this.timeout
	}

	timeout = context.AdjustTimeout(timeout, request.Type(), request.IsPrepare())
	if timeout != request.Timeout() {
		request.SetTimeout(timeout)
	}

	if timeout > 0 {
		request.SetTimer(time.AfterFunc(timeout, func() { request.Expire(TIMEOUT, timeout) }))
		context.SetReqDeadline(time.Now().Add(timeout))
	} else {
		context.SetReqDeadline(time.Time{})
	}

	request.NotifyStop(operator)
	request.SetExecTime(time.Now())
	operator.RunOnce(context, nil)

	request.Execute(this, context, request.Type(), prepared.Signature())
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
	autoExecute := request.AutoExecute() == value.TRUE
	if len(namedArgs) > 0 || len(positionalArgs) > 0 || autoExecute {
		autoPrepare = false
	}

	if prepared == nil && autoPrepare {
		var prepContext planner.PrepareContext
		planner.NewPrepareContext(&prepContext, request.Id().String(), request.QueryContext(), nil, nil,
			request.IndexApiVersion(), request.FeatureControls(), request.UseFts(),
			request.UseCBO(), context.Optimizer(), context.DeltaKeyspaces())

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
		parse := time.Now()
		stmt, err := n1ql.ParseStatement2(request.Statement(), context.Namespace(), request.QueryContext())
		request.Output().AddPhaseTime(execution.PARSE, time.Since(parse))
		if err != nil {
			return nil, errors.NewParseSyntaxError(err, "")
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

		if ok, msg := IsValidStatement(request.TxId(), stype, request.TxImplicit(), allow); !ok {
			return nil, errors.NewTranStatementNotSupportedError(stype, msg)
		}

		if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
			return nil, errors.NewRewriteError(err, "")
		}

		semChecker := semantics.NewSemChecker(this.Enterprise(), stmt.Type(), request.TxId() != "")
		_, err = stmt.Accept(semChecker)
		if err != nil {
			return nil, errors.NewSemanticsError(err, "")
		}

		prep := time.Now()

		// MB-24871: do not replace named/positional parameters with value for prepare statement
		if isPrepare {
			namedArgs = nil
			positionalArgs = nil
		}

		var prepContext planner.PrepareContext
		planner.NewPrepareContext(&prepContext, request.Id().String(), request.QueryContext(), namedArgs,
			positionalArgs, request.IndexApiVersion(), request.FeatureControls(), request.UseFts(),
			request.UseCBO(), context.Optimizer(), context.DeltaKeyspaces())
		if stmt, ok := stmt.(*algebra.Advise); ok {
			stmt.SetContext(context)
		}

		prepared, err = planner.BuildPrepared(stmt, this.datastore, this.systemstore, context.Namespace(),
			autoExecute, !autoExecute, &prepContext)
		request.Output().AddPhaseTime(execution.PLAN, time.Since(prep))
		if err != nil {
			return nil, errors.NewPlanError(err, "")
		}

		// EXECUTE doesn't get a plan. Get the plan from the cache.
		switch stmt.Type() {
		case "EXECUTE":
			var err errors.Error
			exec, _ := stmt.(*algebra.Execute)
			if prepared, err = this.getPreparedByName(exec.Prepared(), request, context); err != nil {
				return nil, err
			}

			request.SetType(prepared.Type())
			if ok, msg := IsValidStatement(request.TxId(), request.Type(), request.TxImplicit(), false); !ok {
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

						// trigger prepare metrics recording
						if prepareds.AddAutoPreparePlan(stmt, prepared) {
							request.SetPrepared(prepared)
						}
					}
				}
			}
		}
	} else {
		if request.TxId() != "" && !autoPrepare {
			var err errors.Error
			if prepared, err = this.getPreparedByName(prepared.Name(), request, context); err != nil {
				return nil, err
			}
		}

		// ditto
		request.SetType(prepared.Type())
		if ok, msg := IsValidStatement(request.TxId(), request.Type(), request.TxImplicit(), true); !ok {
			return nil, errors.NewTranStatementNotSupportedError(request.Type(), msg)
		}
	}

	if logging.LogLevel() >= logging.DEBUG {
		// log EXPLAIN for the request
		logExplain(prepared)
	}

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
	res, _, er := context.ExecutePrepared(prepared, false, request.NamedArgs(), request.PositionalArgs())
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
				&reprepTime)
			if reprepTime > 0 {
				request.Output().AddPhaseTime(execution.REPREPARE, reprepTime)
			}
		} else {
			er = errors.NewUnrecognizedPreparedError(fmt.Errorf("auto_execute did not produce a prepared statement"))
		}
	}

	if er != nil {
		if err, ok := er.(errors.Error); ok {
			return prepared, err
		}
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
		&reprepTime)
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

func logExplain(prepared *plan.Prepared) {
	var pl plan.Operator = prepared
	explain, err := json.MarshalIndent(pl, "", "    ")
	if err != nil {
		logging.Tracep("Error logging explain", logging.Pair{"error", err})
		return
	}

	logging.Tracep("Explain ", logging.Pair{"explain", fmt.Sprintf("<ud>%v</ud>", string(explain))})
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
func IsIPv6() bool {
	return _IPv6
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

func SetIP(val bool) {
	_IPv6 = val
	util.IPv6 = val
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

func IsValidStatement(txId, stmtType string, tximplicit, allow bool) (bool, string) {
	switch stmtType {
	case "SELECT", "UPDATE", "INSERT", "UPSERT", "DELETE", "MERGE":
		return true, ""
	case "EXECUTE", "PREPARE":
		return true, ""
	case "COMMIT", "ROLLBACK", "ROLLBACK_SAVEPOINT", "SET_TRANSACTION_ISOLATION", "SAVEPOINT":
		return allow || txId != "", "outside the"
	case "START_TRANSACTION":
		return allow || txId == "", "within the"
	default:
		if txId != "" {
			return false, "within the"
		}
		return !tximplicit, "in implicit"
	}
}
