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
	"time"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type RequestChannel chan Request

const RESULT_CAP = 1 << 14
const ERROR_CAP = 1 << 10

type State string

const (
	PENDING   State = "pending"
	COMPLETED State = "completed"
	TIMEOUT   State = "timeout"
	FATAL     State = "fatal"
)

type Request interface {
	Statement() string
	Prepared() *plan.Prepared
	Arguments() map[string]value.Value
	Namespace() string
	Timeout() time.Duration
	Readonly() bool
	Metrics() value.Tristate
	RequestTime() time.Time
	ServiceTime() time.Time
	Output() execution.Output
	CloseNotify() chan bool
	Servicing()
	Fail(err errors.Error)
	Execute(server *Server, signature value.Value, notifyStop chan bool)
	Expire()
	State() State
}

type BaseRequest struct {
	statement   string
	prepared    *plan.Prepared
	arguments   map[string]value.Value
	namespace   string
	timeout     time.Duration
	readonly    bool
	metrics     value.Tristate
	requestTime time.Time
	serviceTime time.Time
	state       State
	results     value.ValueChannel
	errors      errors.ErrorChannel
	warnings    errors.ErrorChannel
	closeNotify chan bool
	stopNotify  chan bool
	stopResult  chan bool
	stopExecute chan bool
}

func NewBaseRequest(statement string, prepared *plan.Prepared, arguments map[string]value.Value,
	namespace string, readonly bool, metrics value.Tristate) *BaseRequest {
	rv := &BaseRequest{
		statement:   statement,
		prepared:    prepared,
		arguments:   arguments,
		namespace:   namespace,
		readonly:    readonly,
		metrics:     metrics,
		requestTime: time.Now(),
		serviceTime: time.Now(),
		state:       PENDING,
		results:     make(value.ValueChannel, RESULT_CAP),
		errors:      make(errors.ErrorChannel, ERROR_CAP),
		warnings:    make(errors.ErrorChannel, ERROR_CAP),
		closeNotify: make(chan bool, 1),
		stopResult:  make(chan bool, 1),
		stopExecute: make(chan bool, 1),
	}

	return rv
}

func (this *BaseRequest) SetTimeout(request Request, timeout time.Duration) {
	this.timeout = timeout

	// Apply request timeout
	if timeout > 0 {
		time.AfterFunc(timeout, func() { request.Expire() })
	}
}

func (this *BaseRequest) Statement() string {
	return this.statement
}

func (this *BaseRequest) Prepared() *plan.Prepared {
	return this.prepared
}

func (this *BaseRequest) Arguments() map[string]value.Value {
	return this.arguments
}

func (this *BaseRequest) Namespace() string {
	return this.namespace
}

func (this *BaseRequest) Timeout() time.Duration {
	return this.timeout
}

func (this *BaseRequest) Readonly() bool {
	return this.readonly
}

func (this *BaseRequest) Metrics() value.Tristate {
	return this.metrics
}

func (this *BaseRequest) RequestTime() time.Time {
	return this.requestTime
}

func (this *BaseRequest) ServiceTime() time.Time {
	return this.serviceTime
}

func (this *BaseRequest) State() State {
	return this.state
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
