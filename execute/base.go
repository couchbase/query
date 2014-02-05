//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	_ "fmt"
	"sync"

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/value"
)

type base struct {
	itemChannel value.AnnotatedChannel
	stopChannel StopChannel // Never closed
	input       Operator
	output      Operator
	stop        Operator
	once        sync.Once
}

const _ITEM_CHAN_SIZE = 1024
const _STOP_CHAN_SIZE = 64

func newBase() base {
	return base{
		itemChannel: make(value.AnnotatedChannel, _ITEM_CHAN_SIZE),
		stopChannel: make(StopChannel, _STOP_CHAN_SIZE),
	}
}

func (this *base) ItemChannel() value.AnnotatedChannel {
	return this.itemChannel
}

func (this *base) StopChannel() StopChannel {
	return this.stopChannel
}

func (this *base) Input() Operator {
	return this.input
}

func (this *base) SetInput(op Operator) {
	this.input = op
}

func (this *base) Output() Operator {
	return this.output
}

func (this *base) SetOutput(op Operator) {
	this.output = op
}

func (this *base) Stop() Operator {
	return this.stop
}

func (this *base) SetStop(op Operator) {
	this.stop = op
}

func (this *base) copy() base {
	return base{
		itemChannel: make(value.AnnotatedChannel, _ITEM_CHAN_SIZE),
		stopChannel: make(StopChannel, _STOP_CHAN_SIZE),
		input:       this.input,
		output:      this.output,
		stop:        this.stop,
	}
}

func (this *base) runConsumer(cons consumer, context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel)                   // Broadcast that I have stopped
		defer func() { this.stop.StopChannel() <- 1 }() // Notify that I have stopped

		go this.input.RunOnce(context, parent)

		var item value.AnnotatedValue
		ok := cons.beforeItems(context, parent)

		for ok {
			select {
			case item, ok = <-this.input.ItemChannel():
				if ok {
					ok = cons.processItem(item, context)
				}
			case _, _ = <-this.stopChannel: // Never closed
				break
			}
		}

		cons.afterItems(context)
	})
}

func (this *base) sendItem(item value.AnnotatedValue) bool {
	ok := true
	for ok {
		select {
		case this.output.ItemChannel() <- item:
			return true
		case _, _ = <-this.stopChannel: // Never closed
			return false
		}
	}

	return ok
}

func (this *base) sendWarning(warning err.Error, context Context) bool {
	return this.sendState(warning, context.WarningChannel())
}

func (this *base) sendError(err err.Error, context Context) bool {
	return this.sendState(err, context.ErrorChannel())
}

func (this *base) sendState(err err.Error, channel err.ErrorChannel) bool {
	ok := true
	for ok {
		select {
		case channel <- err:
			return true
		case _, _ = <-this.stopChannel: // Never closed
			return false
		}
	}

	return ok
}

type consumer interface {
	beforeItems(context *Context, parent value.Value) bool
	processItem(item value.AnnotatedValue, context *Context) bool
	afterItems(context *Context)
}

func (this *base) beforeItems(context *Context, parent value.Value) bool {
	return true
}

func (this *base) afterItems(context *Context) {
}
