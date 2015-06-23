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
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Server struct {
	datastore      datastore.Datastore
	systemstore    datastore.Datastore
	configstore    clustering.ConfigurationStore
	acctstore      accounting.AccountingStore
	namespace      string
	readonly       bool
	channel        RequestChannel
	servicerCount  int
	maxParallelism int
	timeout        time.Duration
	signature      bool
	metrics        bool
	keepAlive      int
	once           sync.Once
}

// Default Keep Alive Length

const KEEP_ALIVE_DEFAULT = 1024 * 16

func NewServer(store datastore.Datastore, config clustering.ConfigurationStore,
	acctng accounting.AccountingStore, namespace string, readonly bool,
	channel RequestChannel, servicerCount, maxParallelism int, timeout time.Duration,
	signature, metrics bool, keepAlive int) (*Server, errors.Error) {
	rv := &Server{
		datastore:      store,
		configstore:    config,
		acctstore:      acctng,
		namespace:      namespace,
		readonly:       readonly,
		channel:        channel,
		servicerCount:  servicerCount,
		maxParallelism: maxParallelism,
		timeout:        timeout,
		signature:      signature,
		metrics:        metrics,
		keepAlive:      keepAlive,
	}

	store.SetLogLevel(logging.LogLevel())

	if rv.maxParallelism <= 0 {
		rv.maxParallelism = runtime.NumCPU()
	}

	sys, err := system.NewDatastore(store)
	if err != nil {
		return nil, err
	}

	rv.systemstore = sys
	return rv, nil
}

func (this *Server) Datastore() datastore.Datastore {
	return this.datastore
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

func (this *Server) Signature() bool {
	return this.signature
}

func (this *Server) Metrics() bool {
	return this.metrics
}

func (this *Server) KeepAlive() int {
	return this.keepAlive
}

func (this *Server) Serve() {
	this.once.Do(func() {
		// Use a threading model. Do not spawn a separate
		// goroutine for each request, as that would be
		// unbounded and could degrade performance of already
		// executing queries.
		for i := 0; i < this.servicerCount; i++ {
			go this.doServe()
		}
	})
}

func (this *Server) doServe() {
	for request := range this.channel {
		this.serviceRequest(request)
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

	var operator execution.Operator
	if request.State() != FATAL {
		var err error
		build := time.Now()
		operator, err = execution.Build(prepared)
		if err != nil {
			request.Fail(errors.NewError(err, ""))
		}

		if logging.LogLevel() >= logging.TRACE {
			request.Output().AddPhaseTime("instantiate", time.Since(build))
		}
	}

	if request.State() == FATAL {
		// Fail the request - Write out response - and return
		request.Failed(this)
		return
	}

	// Apply server execution timeout
	if this.timeout > 0 {
		timer := time.AfterFunc(this.timeout, func() { request.Expire() })
		defer timer.Stop()
	}

	go request.Execute(this, prepared.Signature(), operator.StopChannel())

	maxParallelism := util.MinInt(this.maxParallelism, request.MaxParallelism())

	context := execution.NewContext(this.datastore, this.systemstore, namespace,
		this.readonly, maxParallelism, request.NamedArgs(), request.PositionalArgs(),
		request.Credentials(), request.ScanConsistency(), request.ScanVector(), request.Output())

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

		prep := time.Now()
		prepared, err = planner.BuildPrepared(stmt, this.datastore, this.systemstore, namespace, false)
		if err != nil {
			return nil, errors.NewPlanError(err, "")
		}

		if logging.LogLevel() >= logging.TRACE {
			request.Output().AddPhaseTime("plan", time.Since(prep))
			request.Output().AddPhaseTime("parse", prep.Sub(parse))
		}
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
