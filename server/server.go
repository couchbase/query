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
	"runtime"
	"sync"
	"time"

	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/datastore/system"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/parser/n1ql"
	"github.com/couchbaselabs/query/plan"
)

type Server struct {
	datastore   datastore.Datastore
	systemstore datastore.Datastore
	configstore clustering.ConfigurationStore
	namespace   string
	readonly    bool
	channel     RequestChannel
	threadCount int
	timeout     time.Duration
	signature   bool
	metrics     bool
	once        sync.Once
}

func NewServer(store datastore.Datastore, config clustering.ConfigurationStore,
	namespace string, readonly bool, channel RequestChannel, threadCount int,
	timeout time.Duration, signature, metrics bool) (*Server, errors.Error) {
	rv := &Server{
		datastore:   store,
		configstore: config,
		namespace:   namespace,
		readonly:    readonly,
		channel:     channel,
		threadCount: threadCount,
		timeout:     timeout,
		signature:   signature,
		metrics:     metrics,
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

func (this *Server) Channel() RequestChannel {
	return this.channel
}

func (this *Server) Signature() bool {
	return this.signature
}

func (this *Server) Metrics() bool {
	return this.metrics
}

func (this *Server) Serve() {
	this.once.Do(func() {
		// Use a threading model. Do not spawn a separate
		// goroutine for each request, as that would be
		// unbounded and could degrade performance of already
		// executing queries.
		for i := 0; i < this.threadCount; i++ {
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
			logging.Severep("", logging.Pair{"panic", err},
				logging.Pair{"stack", runtime.Stack(buf, false)})
		}
	}()

	// The request may have been failed - e.g. http request missing required params
	// do not proceed if so
	if request.State() == FATAL {
		return
	}

	request.Servicing()

	namespace := request.Namespace()
	if namespace == "" {
		namespace = this.namespace
	}

	prepared, err := this.getPrepared(request, namespace)
	if err != nil {
		request.Fail(errors.NewError(err, ""))
		return
	}

	if (this.readonly || request.Readonly()) && !prepared.Readonly() {
		request.Fail(errors.NewError(nil, "The server or request is read-only"+
			" and cannot accept this write statement."))
		return
	}

	operator, err := execution.Build(prepared)
	if err != nil {
		request.Fail(errors.NewError(err, ""))
		return
	}

	// Apply server execution timeout
	if this.timeout > 0 {
		timer := time.AfterFunc(this.timeout, func() { request.Expire() })
		defer timer.Stop()
	}

	go request.Execute(this, prepared.Signature(), operator.StopChannel())

	context := execution.NewContext(this.datastore, this.systemstore, namespace,
		this.readonly, request.NamedArgs(), request.PositionalArgs(), request.Output())
	operator.RunOnce(context, nil)
}

func (this *Server) getPrepared(request Request, namespace string) (*plan.Prepared, error) {
	prepared := request.Prepared()
	if prepared == nil {
		stmt, err := n1ql.ParseStatement(request.Statement())
		if err != nil {
			return nil, err
		}

		prepared, err = plan.Prepare(stmt, this.datastore, this.systemstore, namespace, false)
		if err != nil {
			return nil, err
		}
	}

	return prepared, nil
}
