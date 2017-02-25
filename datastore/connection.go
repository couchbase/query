//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type StopChannel chan bool

type ValueConnection struct {
	valueChannel value.ValueChannel // Closed by the generator when the scan is completed or aborted.
	stopChannel  StopChannel        // Notifies generator  to stop generating. Never closed, just garbage-collected.
	context      Context            // Context.
	timeout      bool               // True if timed out.
}

const _VALUE_CAP = 256 // Buffer size

func NewValueConnection(context Context) *ValueConnection {
	return NewSizedValueConnection(_VALUE_CAP, context)
}

func NewSizedValueConnection(size int, context Context) *ValueConnection {
	return &ValueConnection{
		valueChannel: make(value.ValueChannel, size),
		stopChannel:  make(StopChannel, 1),
		context:      context,
	}
}

func (this *ValueConnection) ValueChannel() value.ValueChannel {
	return this.valueChannel
}

func (this *ValueConnection) StopChannel() StopChannel {
	return this.stopChannel
}

func (this *ValueConnection) Fatal(err errors.Error) {
	this.context.Fatal(err)
}

func (this *ValueConnection) Error(err errors.Error) {
	this.context.Error(err)
}

func (this *ValueConnection) Warning(wrn errors.Error) {
	this.context.Warning(wrn)
}

func (this *ValueConnection) Timeout() bool {
	return this.timeout
}

func (this *ValueConnection) SetTimeout(timeout bool) {
	this.timeout = timeout
}
