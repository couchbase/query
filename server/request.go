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
	"sync"
	"time"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type RequestChannel chan Request
type stopChannel chan bool

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
	RequestTime() time.Time
	ServiceTime() time.Time
	Timeout() time.Duration
	Namespace() string
	Command() string
	Plan() plan.Operator
	Arguments() map[string]value.Value
	Output() execution.Output
	Await()
	Servicing()
	Fail(err errors.Error)
	Execute()
	Expire()
	State() State
}

type BaseRequest struct {
	requestTime time.Time
	serviceTime time.Time
	timeout     time.Duration
	namespace   string
	command     string
	plan        plan.Operator
	arguments   map[string]value.Value
	state       State
	results     value.ValueChannel
	errors      errors.ErrorChannel
	warnings    errors.ErrorChannel
	stop        stopChannel
	once        sync.Once
}

func NewBaseRequest(timeout time.Duration, namespace, command string,
	plan plan.Operator, arguments map[string]value.Value) *BaseRequest {
	return &BaseRequest{
		requestTime: time.Now(),
		serviceTime: time.Now(),
		timeout:     timeout,
		namespace:   namespace,
		command:     command,
		plan:        plan,
		arguments:   arguments,
		state:       PENDING,
		results:     make(value.ValueChannel, RESULT_CAP),
		errors:      make(errors.ErrorChannel, ERROR_CAP),
		warnings:    make(errors.ErrorChannel, ERROR_CAP),
		stop:        make(stopChannel, 1),
	}
}

func (this *BaseRequest) RequestTime() time.Time {
	return this.requestTime
}

func (this *BaseRequest) ServiceTime() time.Time {
	return this.serviceTime
}

func (this *BaseRequest) Timeout() time.Duration {
	return this.timeout
}

func (this *BaseRequest) Namespace() string {
	return this.namespace
}

func (this *BaseRequest) Command() string {
	return this.command
}

func (this *BaseRequest) Plan() plan.Operator {
	return this.plan
}

func (this *BaseRequest) Arguments() map[string]value.Value {
	return this.arguments
}

func (this *BaseRequest) Await() {
	<-this.stop
}

func (this *BaseRequest) Servicing() {
	this.serviceTime = time.Now()
}

func (this *BaseRequest) State() State {
	return this.state
}

func (this *BaseRequest) Stop(state State) {
	this.state = state

	select {
	case this.stop <- false:
	default:
	}
}

func (this *BaseRequest) Result(item value.Value) bool {
	this.results <- item
	return true
}

func (this *BaseRequest) Fatal(err errors.Error) {
	select {
	case this.errors <- err:
	default:
	}

	this.Stop(FATAL)
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
