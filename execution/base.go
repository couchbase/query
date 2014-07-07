//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"fmt"
	"sync"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/value"
)

type base struct {
	itemChannel value.AnnotatedChannel
	stopChannel StopChannel // Never closed
	input       Operator
	output      Operator
	stop        Operator
	parent      Parent
	once        sync.Once
	batch       []value.AnnotatedValue
}

const _ITEM_CHAN_CAP = 1024

func newBase() base {
	return base{
		itemChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		stopChannel: make(StopChannel, 1),
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

func (this *base) Parent() Parent {
	return this.parent
}

func (this *base) SetParent(parent Parent) {
	this.parent = parent
}

func (this *base) copy() base {
	return base{
		itemChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		stopChannel: make(StopChannel, 1),
		input:       this.input,
		output:      this.output,
		parent:      this.parent,
	}
}

func (this *base) sendItem(item value.AnnotatedValue) bool {
	select {
	case this.output.ItemChannel() <- item:
		return true
	case <-this.stopChannel: // Never closed
		return false
	}
}

type consumer interface {
	beforeItems(context *Context, parent value.Value) bool
	processItem(item value.AnnotatedValue, context *Context) bool
	afterItems(context *Context)
}

func (this *base) runConsumer(cons consumer, context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped
		defer func() { this.batch = nil }()

		go this.input.RunOnce(context, parent)

		var item value.AnnotatedValue
		ok := cons.beforeItems(context, parent)

		for ok {
			select {
			case item, ok = <-this.input.ItemChannel():
				if ok {
					ok = cons.processItem(item, context)
				}
			case <-this.stopChannel: // Never closed
				break
			}
		}

		cons.afterItems(context)
	})
}

// Override if needed
func (this *base) beforeItems(context *Context, parent value.Value) bool {
	return true
}

// Override if needed
func (this *base) afterItems(context *Context) {
}

// Unblock all dependencies.
func (this *base) notify() {
	this.notifyParent()
	this.notifyStop()
}

// Notify parent, if any.
func (this *base) notifyParent() {
	parent := this.parent
	if parent != nil {
		// Blocks on parent
		parent.ChildChannel() <- false
		this.parent = nil
	}
}

// Notify upstream to stop.
func (this *base) notifyStop() {
	stop := this.stop
	if stop != nil {
		select {
		case stop.StopChannel() <- false:
			this.stop = nil
		default:
			// Already notified.
		}
	}
}

type batcher interface {
	allocateBatch(n int)
	enbatch(item value.AnnotatedValue, b batcher, context *Context) bool
	flushBatch(context *Context) bool
}

func (this *base) allocateBatch(n int) {
	this.batch = make([]value.AnnotatedValue, 0, n)
}

func (this *base) enbatch(item value.AnnotatedValue, b batcher, context *Context) bool {
	if this.batch == nil {
		this.allocateBatch(1024)
	} else if len(this.batch) == cap(this.batch) {
		if !b.flushBatch(context) {
			return false
		}

		if len(this.batch) == cap(this.batch) {
			this.allocateBatch(1024)
		}
	}

	this.batch = append(this.batch, item)
	return true
}

func (this *base) requireKey(item value.AnnotatedValue, context *Context) (string, bool) {
	mv := item.GetAttachment("meta")
	if mv == nil {
		context.ErrorChannel() <- errors.NewError(nil, "Unable to find meta.")
		return "", false
	}

	meta := mv.(map[string]interface{})
	key, ok := meta["id"]
	if !ok {
		context.ErrorChannel() <- errors.NewError(nil, "Unable to find key.")
		return "", false
	}

	switch key := key.(type) {
	case string:
		return key, true
	default:
		e := errors.NewError(nil, fmt.Sprintf("Unable to process non-string key %v of type %T.", key, key))
		context.ErrorChannel() <- e
		return "", false
	}
}
