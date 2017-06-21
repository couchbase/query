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
	"math"
	"os"
	"runtime"
	"runtime/pprof"
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
	"github.com/couchbase/query/value"
)

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
}

// Default Keep Alive Length

const KEEP_ALIVE_DEFAULT = 1024 * 16

func NewServer(store datastore.Datastore, sys datastore.Datastore, config clustering.ConfigurationStore,
	acctng accounting.AccountingStore, namespace string, readonly bool,
	channel, plusChannel RequestChannel, servicers, plusServicers, maxParallelism int,
	timeout time.Duration, signature, metrics, enterprise, pretty bool) (*Server, errors.Error) {
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
	}

	// special case handling for the atomic specfic stuff
	atomic.StoreInt64(&rv.servicers, int64(servicers))
	atomic.StoreInt64(&rv.plusServicers, int64(plusServicers))

	store.SetLogLevel(logging.LogLevel())
	rv.SetMaxParallelism(maxParallelism)

	//	sys, err := system.NewDatastore(store)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	rv.systemstore = sys
	return rv, nil
}

func (this *Server) Datastore() datastore.Datastore {
	return this.datastore
}

func (this *Server) Systemstore() datastore.Datastore {
	return this.systemstore
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

func (this *Server) PipelineCap() int {
	return int(execution.GetPipelineCap())
}

func (this *Server) SetPipelineCap(pipeline_cap int) {
	execution.SetPipelineCap(pipeline_cap)
}

func (this *Server) SetPipelineBatch(pipeline_batch int) {
	execution.SetPipelineBatch(pipeline_batch)
}

func (this *Server) PipelineBatch() int {
	return execution.PipelineBatchSize()
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

func (this *Server) ScanCap() int {
	return int(datastore.GetScanCap())
}

func (this *Server) SetScanCap(size int) {
	datastore.SetScanCap(int64(size))
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
		this.readonly, maxParallelism, request.NamedArgs(), request.PositionalArgs(),
		request.Credentials(), request.ScanConsistency(), request.ScanVectorSource(), request.Output())

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

	if logging.LogLevel() >= logging.TRACE {
		request.Output().AddPhaseTime("instantiate", time.Since(build))
	}

	if request.State() == FATAL {
		request.Failed(this)
		return
	}

	// Apply server execution timeout
	if this.Timeout() > 0 {
		timer := time.AfterFunc(this.Timeout(), func() { request.Expire(TIMEOUT, this.Timeout()) })
		defer timer.Stop()
	}

	go request.Execute(this, prepared.Signature(), operator.StopChannel())

	run := time.Now()
	operator.RunOnce(context, nil)

	if logging.LogLevel() >= logging.TRACE {
		request.Output().AddPhaseTime("run", time.Since(run))
		logPhases(request)
	}
}

func (this *Server) getPrepared(request Request, namespace string) (*plan.Prepared, errors.Error) {
	prepared := request.Prepared()
	if prepared == nil {
		parse := time.Now()
		stmt, err := n1ql.ParseStatement(request.Statement())
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
		if isprepare {
			namedArgs = nil
			positionalArgs = nil
		}
		prepared, err = planner.BuildPrepared(stmt, this.datastore, this.systemstore, namespace, false, namedArgs, positionalArgs)
		if err != nil {
			return nil, errors.NewPlanError(err, "")
		}

		// In order to allow monitoring to track prepared statement executed through
		// N1QL "EXECUTE", set request.prepared - because, as of yet, it isn't!
		//
		// HACK ALERT - request does not currently track the request type
		// and even if it did, and prepared.(*plan.Prepared) is set, it
		// does not carry a name or text.
		// This should probably done in build.go and / or build_execute.go,
		// but for now this will do.
		exec, ok := stmt.(*algebra.Execute)
		if ok && exec.Prepared() != nil {
			prep, _ := plan.TrackPrepared(exec.Prepared())
			request.SetPrepared(prep)
		}

		if logging.LogLevel() >= logging.TRACE {
			request.Output().AddPhaseTime("plan", time.Since(prep))
			request.Output().AddPhaseTime("parse", prep.Sub(parse))
		}
	}

	request.SetTimings(prepared.Operator)
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

	logging.Tracep("Explain ", logging.Pair{"explain", string(explain)})
}

func logPhases(request Request) {
	phaseTimes := request.Output().PhaseTimes()
	if len(phaseTimes) == 0 {
		return
	}

	pairs := make([]logging.Pair, 0, len(phaseTimes)+1)
	pairs = append(pairs, logging.Pair{"_id", request.Id()})
	for k, v := range phaseTimes {
		pairs = append(pairs, logging.Pair{k, v})
	}

	logging.Tracep("Phase aggregates", pairs...)
}
