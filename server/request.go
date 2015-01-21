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
	"sync/atomic"
	"time"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/util"
	"github.com/couchbaselabs/query/value"
)

type RequestChannel chan Request

const RESULT_CAP = 1 << 14
const ERROR_CAP = 1 << 10

type State string

const (
	RUNNING   State = "running"
	SUCCESS   State = "success"
	ERRORS    State = "errors"
	COMPLETED State = "completed"
	STOPPED   State = "stopped"
	TIMEOUT   State = "timeout"
	FATAL     State = "fatal"
)

type Request interface {
	Id() RequestID
	ClientID() ClientContextID
	Statement() string
	Prepared() *plan.Prepared
	NamedArgs() map[string]value.Value
	PositionalArgs() value.Values
	Namespace() string
	Timeout() time.Duration
	Readonly() value.Tristate
	Metrics() value.Tristate
	Signature() value.Tristate
	ScanConfiguration() ScanConfiguration
	RequestTime() time.Time
	ServiceTime() time.Time
	Output() execution.Output
	CloseNotify() chan bool
	Servicing()
	Fail(err errors.Error)
	Execute(server *Server, signature value.Value, notifyStop chan bool)
	Failed(server *Server)
	Expire()
	State() State
	Credentials() datastore.Credentials
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
	NOT_BOUNDED ScanConsistency = iota
	REQUEST_PLUS
	STATEMENT_PLUS
	AT_PLUS
	UNDEFINED_CONSISTENCY
)

type ScanConfiguration interface {
	ScanConsistency() ScanConsistency
	ScanWait() time.Duration
	ScanVectorFull() []int
	ScanVectorSparse() map[string]int
}

type BaseRequest struct {
	id             *requestIDImpl
	client_id      *clientContextIDImpl
	statement      string
	prepared       *plan.Prepared
	namedArgs      map[string]value.Value
	positionalArgs value.Values
	namespace      string
	timeout        time.Duration
	readonly       value.Tristate
	signature      value.Tristate
	metrics        value.Tristate
	consistency    ScanConfiguration
	mutationCount  uint64
	requestTime    time.Time
	serviceTime    time.Time
	state          State
	credentials    datastore.Credentials
	results        value.ValueChannel
	errors         errors.ErrorChannel
	warnings       errors.ErrorChannel
	closeNotify    chan bool // implement http.CloseNotifier
	stopNotify     chan bool // notified when request execution stops
	stopResult     chan bool // stop consuming results
	stopExecute    chan bool // stop executing request
}

type requestIDImpl struct {
	id string
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

const MAX_CLIENTID = 64

func newClientContextIDImpl(id string) *clientContextIDImpl {
	if len(id) > MAX_CLIENTID {
		id_cut := make([]byte, MAX_CLIENTID)
		copy(id_cut[:], id)
		return &clientContextIDImpl{id: string(id_cut)}
	}
	return &clientContextIDImpl{id: id}
}

func NewBaseRequest(statement string, prepared *plan.Prepared, namedArgs map[string]value.Value, positionalArgs value.Values,
	namespace string, readonly, metrics, signature value.Tristate, consistency ScanConfiguration,
	client_id string, creds datastore.Credentials) *BaseRequest {
	rv := &BaseRequest{
		statement:      statement,
		prepared:       prepared,
		namedArgs:      namedArgs,
		positionalArgs: positionalArgs,
		namespace:      namespace,
		readonly:       readonly,
		signature:      signature,
		metrics:        metrics,
		consistency:    consistency,
		credentials:    creds,
		requestTime:    time.Now(),
		serviceTime:    time.Now(),
		state:          RUNNING,
		results:        make(value.ValueChannel, RESULT_CAP),
		errors:         make(errors.ErrorChannel, ERROR_CAP),
		warnings:       make(errors.ErrorChannel, ERROR_CAP),
		closeNotify:    make(chan bool, 1),
		stopResult:     make(chan bool, 1),
		stopExecute:    make(chan bool, 1),
	}
	uuid, _ := util.UUID()
	rv.id = &requestIDImpl{id: uuid}
	rv.client_id = newClientContextIDImpl(client_id)
	return rv
}

func (this *BaseRequest) SetTimeout(request Request, timeout time.Duration) {
	this.timeout = timeout

	// Apply request timeout
	if timeout > 0 {
		time.AfterFunc(timeout, func() { request.Expire() })
	}
}

func (this *BaseRequest) Id() RequestID {
	return this.id
}

func (this *BaseRequest) ClientID() ClientContextID {
	return this.client_id
}

func (this *BaseRequest) Statement() string {
	return this.statement
}

func (this *BaseRequest) Prepared() *plan.Prepared {
	return this.prepared
}

func (this *BaseRequest) NamedArgs() map[string]value.Value {
	return this.namedArgs
}

func (this *BaseRequest) PositionalArgs() value.Values {
	return this.positionalArgs
}

func (this *BaseRequest) Namespace() string {
	return this.namespace
}

func (this *BaseRequest) Timeout() time.Duration {
	return this.timeout
}

func (this *BaseRequest) Readonly() value.Tristate {
	return this.readonly
}

func (this *BaseRequest) Signature() value.Tristate {
	return this.signature
}

func (this *BaseRequest) Metrics() value.Tristate {
	return this.metrics
}

func (this *BaseRequest) ScanConfiguration() ScanConfiguration {
	return this.consistency
}

func (this *BaseRequest) RequestTime() time.Time {
	return this.requestTime
}

func (this *BaseRequest) ServiceTime() time.Time {
	return this.serviceTime
}

func (this *BaseRequest) SetState(state State) {
	this.state = state
}

func (this *BaseRequest) State() State {
	return this.state
}

func (this *BaseRequest) Credentials() datastore.Credentials {
	return this.credentials
}

func (this *BaseRequest) CloseNotify() chan bool {
	return this.closeNotify
}

func (this *BaseRequest) Servicing() {
	this.serviceTime = time.Now()
}

func (this *BaseRequest) Result(item value.Value) bool {
	select {
	case <-this.stopResult:
		return false
	default:
	}

	select {
	case this.results <- item:
		return true
	case <-this.stopResult:
		return false
	}
}

func (this *BaseRequest) CloseResults() {
	close(this.results)
}

func (this *BaseRequest) Fatal(err errors.Error) {
	defer this.Stop(FATAL)

	this.Error(err)
}

func (this *BaseRequest) Error(err errors.Error) {
	select {
	case this.errors <- err:
	default:
	}
}

func (this *BaseRequest) Warning(wrn errors.Error) {
	select {
	case this.warnings <- wrn:
	default:
	}
}

func (this *BaseRequest) AddMutationCount(i uint64) {
	atomic.AddUint64(&this.mutationCount, i)
}

func (this *BaseRequest) MutationCount() uint64 {
	return atomic.LoadUint64(&this.mutationCount)
}

func (this *BaseRequest) Results() value.ValueChannel {
	return this.results
}

func (this *BaseRequest) Errors() errors.ErrorChannel {
	return this.errors
}

func (this *BaseRequest) Warnings() errors.ErrorChannel {
	return this.warnings
}

func (this *BaseRequest) NotifyStop(ch chan bool) {
	this.stopNotify = ch
}

func (this *BaseRequest) StopExecute() chan bool {
	return this.stopExecute
}

func (this *BaseRequest) Stop(state State) {
	defer sendStop(this.closeNotify)
	defer sendStop(this.stopNotify)
	defer sendStop(this.stopResult)
	defer sendStop(this.stopExecute)

	this.state = state
}

func sendStop(ch chan bool) {
	select {
	case ch <- false:
	default:
	}
}
