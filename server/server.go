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
	"fmt"
	"sync"
	"time"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/datastore/system"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/parser/n1ql"
	"github.com/couchbaselabs/query/plan"
)

type Server struct {
	datastore   datastore.Datastore
	systemstore datastore.Datastore
	namespace   string
	readonly    bool
	channel     RequestChannel
	threadCount int
	timeout     time.Duration
	once        sync.Once
}

func NewServer(store datastore.Datastore, namespace string, readonly bool,
	channel RequestChannel, threadCount int, timeout time.Duration) (*Server, errors.Error) {
	rv := &Server{
		datastore:   store,
		namespace:   namespace,
		readonly:    readonly,
		channel:     channel,
		threadCount: threadCount,
		timeout:     timeout,
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
	var request Request
	ok := true

	for ok {
		request, ok = <-this.channel
		if request != nil {
			this.serveRequest(request)
		}
	}
}

func (this *Server) serveRequest(request Request) {
	namespace := request.Namespace()
	if namespace == "" {
		namespace = this.namespace
	}

	plann := request.Plan()
	if plann == nil {
		node, err := n1ql.Parse(request.Command())
		if err != nil {
			request.Fail(err)
			return
		}

		plann, err = plan.Build(node, this.datastore, namespace, true)
		if err != nil {
			request.Fail(err)
			return
		}
	}

	if this.readonly && !plann.Readonly() {
		request.Fail(fmt.Errorf("The server is read-only and cannot execute this write request."))
		return
	}

	operator, err := execution.Build(plann)
	if err != nil {
		request.Fail(err)
		return
	}

	expire := func() {
		operator.StopChannel() <- false
		request.Expire()
	}

	request.Start()

	// Apply request timeout
	var requestTimer *time.Timer
	if request.Timeout() > 0 {
		delay := request.Timeout() - time.Now().Sub(request.RequestTime())
		if delay <= 0 {
			request.Expire()
			return
		}

		requestTimer = time.AfterFunc(delay, expire)
	}

	// Apply server timeout
	var serverTimer *time.Timer
	if this.timeout > 0 {
		serverTimer = time.AfterFunc(this.timeout, expire)
	}

	context := execution.NewContext(this.datastore, this.systemstore,
		namespace, this.readonly, request.Arguments(), request.Output())
	operator.RunOnce(context, nil)

	if requestTimer != nil {
		requestTimer.Stop()
	}

	if serverTimer != nil {
		serverTimer.Stop()
	}

	request.Finish()
}
