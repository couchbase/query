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
)

var _PROFILE_MAP = map[string]Profile{
	"off":     ProfOff,
	"phases":  ProfPhases,
	"timings": ProfOn,
}

var _PROFILE_DEFAULT = ProfOff

var _PROFILE_NAMES = []string{
	ProfUnset:  "",
	ProfOff:    "off",
	ProfPhases: "phases",
	ProfOn:     "timings",
}

var _IPv6 = false

func (profile Profile) String() string {
	return _PROFILE_NAMES[profile]
}

type Server struct {
	// due to alignment issues on x86 platforms these atomic
	// variables need to right at the beginning of the structure
	servicers      atomic.AlignedInt64
	plusServicers  atomic.AlignedInt64
	maxParallelism atomic.AlignedInt64
	keepAlive      atomic.AlignedInt64
	requestSize    atomic.AlignedInt64

	sync.RWMutex
	datastore   datastore.Datastore
	systemstore datastore.Datastore
	configstore clustering.ConfigurationStore
	acctstore   accounting.AccountingStore
	namespace   string
	readonly    bool
	channel     RequestChannel
	plusChannel RequestChannel
	done        chan bool
	plusDone    chan bool
	timeout     time.Duration
	signature   bool
	metrics     bool
	wg          sync.WaitGroup
	plusWg      sync.WaitGroup
	memprofile  string
	cpuprofile  string
	enterprise  bool
	pretty      bool
	srvprofile  Profile
	srvcontrols bool
	whitelist   map[string]interface{}
}

// Default Keep Alive Length

const KEEP_ALIVE_DEFAULT = 1024 * 16

func NewServer(store datastore.Datastore, sys datastore.Datastore, config clustering.ConfigurationStore,
	acctng accounting.AccountingStore, namespace string, readonly bool,
	channel, plusChannel RequestChannel, servicers, plusServicers, maxParallelism int,
	timeout time.Duration, signature, metrics, enterprise, pretty bool,
	srvprofile Profile, srvcontrols bool) (*Server, errors.Error) {
	rv := &Server{
		datastore:   store,
		systemstore: sys,
		configstore: config,
		acctstore:   acctng,
		namespace:   namespace,
		readonly:    readonly,
		channel:     channel,
		plusChannel: plusChannel,
		signature:   signature,
		timeout:     timeout,
		metrics:     metrics,
		done:        make(chan bool),
		plusDone:    make(chan bool),
		enterprise:  enterprise,
		pretty:      pretty,
		srvcontrols: srvcontrols,
		srvprofile:  srvprofile,
	}

	// special case handling for the atomic specfic stuff
	atomic.StoreInt64(&rv.servicers, int64(servicers))
	atomic.StoreInt64(&rv.plusServicers, int64(plusServicers))

	store.SetLogLevel(logging.LogLevel())
	rv.SetMaxParallelism(maxParallelism)

	// set default values
	rv.SetMaxIndexAPI(datastore.INDEX_API_MAX)
	if rv.enterprise {
		util.SetN1qlFeatureControl(util.DEF_N1QL_FEAT_CTRL)
	} else {
		util.SetN1qlFeatureControl(util.DEF_N1QL_FEAT_CTRL | util.CE_N1QL_FEAT_CTRL)
	}

	//	sys, err := system.NewDatastore(store)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	rv.systemstore = sys

	// Setup callback function for metakv settings changes
	callb := func(cfg queryMetakv.Config) {
		logging.Infof("Settings notifier from metakv\n")

		// SetParamValuesForAll accepts a full-set or subset of global configuration
		// and updates those fields.
		SetParamValuesForAll(cfg, rv)
	}

	queryMetakv.SetupSettingsNotifier(callb, make(chan struct{}))

	return rv, nil
}

func (this *Server) Datastore() datastore.Datastore {
	return this.datastore
}

func (this *Server) Systemstore() datastore.Datastore {
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

func (this *Server) Channel() RequestChannel {
	return this.channel
}

func (this *Server) PlusChannel() RequestChannel {
	return this.plusChannel
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

func (this *Server) SetPretty(pretty bool) {
	this.Lock()
	defer this.Unlock()
	this.pretty = pretty
}

func (this *Server) KeepAlive() int {
	return int(atomic.LoadInt64(&this.keepAlive))
}

func (this *Server) SetKeepAlive(keepAlive int) {
	if keepAlive <= 0 {
		keepAlive = KEEP_ALIVE_DEFAULT
	}
	atomic.StoreInt64(&this.keepAlive, int64(keepAlive))
}

func (this *Server) MaxParallelism() int {
	return int(atomic.LoadInt64(&this.maxParallelism))
}

func (this *Server) SetMaxParallelism(maxParallelism int) {
	if maxParallelism <= 0 {
		maxParallelism = runtime.NumCPU()
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
	return int(atomic.LoadInt64(&this.servicers))
}

func (this *Server) SetServicers(servicers int) {
	this.Lock()
	defer this.Unlock()

	// MB-19683 - don't restart if no change
	if int(atomic.LoadInt64(&this.servicers)) == servicers {
		return
	}

	// Stop the current set of servicers
	close(this.done)
	logging.Infop("SetServicers - waiting for current servicers to finish")
	this.wg.Wait()
	// Set servicer count and recreate servicers
	atomic.StoreInt64(&this.servicers, int64(servicers))
	logging.Infop("SetServicers - starting new servicers")
	// Start new set of servicers
	this.done = make(chan bool)
	go this.Serve()
}

func (this *Server) PlusServicers() int {
	return int(atomic.LoadInt64(&this.plusServicers))
}

func (this *Server) SetPlusServicers(plusServicers int) {
	this.Lock()
	defer this.Unlock()

	// MB-19683 - don't restart if no change
	if int(atomic.LoadInt64(&this.plusServicers)) == plusServicers {
		return
	}

	// Stop the current set of servicers
	close(this.plusDone)
	logging.Infop("SetPlusServicers - waiting for current plusServicers to finish")
	this.plusWg.Wait()
	// Set plus servicer count and recreate plus servicers
	atomic.StoreInt64(&this.plusServicers, int64(plusServicers))
	logging.Infop("SetPlusServicers - starting new plusServicers")
	// Start new set of servicers
	this.plusDone = make(chan bool)
	go this.PlusServe()
}

func (this *Server) Timeout() time.Duration {
	return this.timeout
}

func (this *Server) SetTimeout(timeout time.Duration) {
	this.timeout = timeout
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

func ParseProfile(name string) (Profile, bool) {
	prof, ok := _PROFILE_MAP[strings.ToLower(name)]
	if ok {
		return prof, ok
	} else {
		return _PROFILE_DEFAULT, ok
	}
}

func (this *Server) Enterprise() bool {
	return this.enterprise
}

func (this *Server) Serve() {
	// Use a threading model. Do not spawn a separate
	// goroutine for each request, as that would be
	// unbounded and could degrade performance of already
	// executing queries.
	servicers := this.Servicers()
	this.wg.Add(servicers)
	for i := 0; i < servicers; i++ {
		go this.doServe()
	}
}

func (this *Server) doServe() {
	defer this.wg.Done()
	ok := true
	for ok {
		select {
		case request := <-this.channel:
			this.serviceRequest(request)
		case <-this.done:
			ok = false
		}
	}
}

func (this *Server) PlusServe() {
	// Use a threading model. Do not spawn a separate
	// goroutine for each request, as that would be
	// unbounded and could degrade performance of already
	// executing queries.
	plusServicers := this.PlusServicers()
	this.plusWg.Add(plusServicers)
	for i := 0; i < plusServicers; i++ {
		go this.doPlusServe()
	}
}

func (this *Server) doPlusServe() {
	defer this.plusWg.Done()
	ok := true
	for ok {
		select {
		case request := <-this.plusChannel:
			this.serviceRequest(request)
		case <-this.plusDone:
			ok = false
		}
	}
}

func (this *Server) serviceRequest(request Request) {
	defer func() {
		err := recover()
		if err != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severep("", logging.Pair{"panic", err},
				logging.Pair{"stack", s})
			os.Stderr.WriteString(s)
			os.Stderr.Sync()
		}
	}()

	request.Servicing()

	namespace := request.Namespace()
	if namespace == "" {
		namespace = this.namespace
	}

	prepared, err := this.getPrepared(request, namespace)
	if err != nil {
		request.Fail(err)
	}

	if (this.readonly || value.ToBool(request.Readonly())) &&
		(prepared != nil && !prepared.Readonly()) {
		request.Fail(errors.NewServiceErrorReadonly("The server or request is read-only" +
			" and cannot accept this write statement."))
	}

	if request.State() == FATAL {
		request.Failed(this)
		return
	}

	maxParallelism := request.MaxParallelism()
	if maxParallelism <= 0 {
		maxParallelism = this.MaxParallelism()
	}

	context := execution.NewContext(request.Id().String(), this.datastore, this.systemstore, namespace,
		this.readonly, maxParallelism, request.ScanCap(), request.PipelineCap(), request.PipelineBatch(),
		request.NamedArgs(), request.PositionalArgs(), request.Credentials(), request.ScanConsistency(),
		request.ScanVectorSource(), request.Output(), request.OriginalHttpRequest(),
		prepared, request.IndexApiVersion(), request.FeatureControls())

	context.SetWhitelist(this.whitelist)

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
	if timeout > 0 {
		request.SetTimer(time.AfterFunc(timeout, func() { request.Expire(TIMEOUT, timeout) }))
		context.SetReqDeadline(time.Now().Add(timeout))
	} else {
		context.SetReqDeadline(time.Time{})
	}

	go request.Execute(this, prepared.Signature(), operator)

	run := time.Now()
	operator.RunOnce(context, nil)

	request.Output().AddPhaseTime(execution.RUN, time.Since(run))
}

func (this *Server) getPrepared(request Request, namespace string) (*plan.Prepared, errors.Error) {
	prepared := request.Prepared()
	if prepared == nil {
		parse := time.Now()
		stmt, err := n1ql.ParseStatement(request.Statement())
		request.Output().AddPhaseTime(execution.PARSE, time.Since(parse))
		if err != nil {
			return nil, errors.NewParseSyntaxError(err, "")
		}

		isprepare := false
		if _, ok := stmt.(*algebra.Prepare); ok {
			isprepare = true
		}

		prep := time.Now()
		namedArgs := request.NamedArgs()
		positionalArgs := request.PositionalArgs()

		// No args for a prepared statement - should we throw an error?
		if isprepare {
			namedArgs = nil
			positionalArgs = nil
		}

		prepared, err = planner.BuildPrepared(stmt, this.datastore, this.systemstore, namespace, false,
			namedArgs, positionalArgs, request.IndexApiVersion(), request.FeatureControls())
		request.Output().AddPhaseTime(execution.PLAN, time.Since(prep))
		if err != nil {
			return nil, errors.NewPlanError(err, "")
		}

		// EXECUTE doesn't get a plan. Get the plan from the cache.
		switch stmt.Type() {
		case "EXECUTE":
			var reprepTime time.Duration
			var err errors.Error

			exec, _ := stmt.(*algebra.Execute)
			if exec.Prepared() != nil {

				prepared, err = prepareds.GetPrepared(exec.Prepared(), prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY, &reprepTime)
				if reprepTime > 0 {
					request.Output().AddPhaseTime(execution.REPREPARE, reprepTime)
				}
				if err != nil {
					return nil, err
				}
				request.SetPrepared(prepared)

				// when executing prepared statements, we set the type to that
				// of the prepared statement
				request.SetType(prepared.Type())
			} else {

				// this never happens, but for completeness
				errors.NewPlanError(nil, "prepared not specified")
			}
		default:

			// set the type for all statements bar prepare
			// (doing otherwise would have accounting track prepares
			// as if they were executions)
			if isprepare {
				request.SetIsPrepare(true)
			} else {
				request.SetType(stmt.Type())
			}

			// even though this is not a prepared statement, add the
			// text for the benefit of context.Recover(): we can
			// output the text in case of crashes
			prepared.SetText(request.Statement())
		}
	} else {

		// ditto
		request.SetType(prepared.Type())
	}

	if logging.LogLevel() >= logging.DEBUG {
		// log EXPLAIN for the request
		logExplain(prepared)
	}

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
	tokens := []string{}
	// For IPv6
	if _IPv6 {
		// Then the url should be of the form [::1]:8091
		tokens = strings.Split(node, "]:")
		host = strings.Replace(tokens[0], "[", "", 1)

	} else {
		// For IPv4
		tokens = strings.Split(node, ":")
		host = tokens[0]
	}

	if len(tokens) == 2 {
		port = tokens[1]
	} else {
		port = ""
	}

	return
}
