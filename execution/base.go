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
	go_atomic "sync/atomic"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
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
	execTime    time.Duration
	duration    time.Duration
	chanTime    time.Duration
	stopped     bool
	bit         uint8
}

const _ITEM_CAP = 512
const _MAP_POOL_CAP = 512

var pipelineCap atomic.AlignedInt64

func init() {
	atomic.StoreInt64(&pipelineCap, int64(_ITEM_CAP))
	p := value.NewAnnotatedPool(_BATCH_SIZE)
	_BATCH_POOL.Store(p)
	j := value.NewAnnotatedJoinPairPool(_BATCH_SIZE)
	_JOIN_BATCH_POOL.Store(j)
}

func SetPipelineCap(cap int) {
	if cap < 1 {
		cap = _ITEM_CAP
	}
	atomic.StoreInt64(&pipelineCap, int64(cap))
}

func GetPipelineCap() int64 {
	return atomic.LoadInt64(&pipelineCap)
}

func newBase() base {
	return base{
		itemChannel: make(value.AnnotatedChannel, GetPipelineCap()),
		stopChannel: make(StopChannel, 1),
	}
}

// The output of this operator will be redirected elsewhere, so we
// allocate a minimal itemChannel.
func newRedirectBase() base {
	return base{
		itemChannel: make(value.AnnotatedChannel),
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

func (this *base) Bit() uint8 {
	return this.bit
}

func (this *base) SetBit(b uint8) {
	this.bit = b
}

func (this *base) copy() base {
	return base{
		itemChannel: make(value.AnnotatedChannel, GetPipelineCap()),
		stopChannel: make(StopChannel, 1),
		input:       this.input,
		output:      this.output,
		parent:      this.parent,
	}
}

func (this *base) sendItem(item value.AnnotatedValue) bool {
	t := time.Now()
	addTime := func() {
		this.chanTime += time.Since(t)
	}
	defer addTime()

	if this.stopped {
		return false
	}

	select {
	case <-this.stopChannel: // Never closed
		return false
	default:
	}

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
	readonly() bool
}

func (this *base) runConsumer(cons consumer, context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped
		defer func() { this.batch = nil }()

		if context.Readonly() && !cons.readonly() {
			return
		}

		ok := cons.beforeItems(context, parent)

		if ok {
			go this.input.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
	loop:
		for ok {
			t := time.Now()

			select {
			case <-this.stopChannel: // Never closed
				this.chanTime += time.Since(t)
				this.stopped = true
				break loop
			default:
			}

			select {
			case item, ok = <-this.input.ItemChannel():
				if ok {
					ok = cons.processItem(item, context)
				}
			case <-this.stopChannel: // Never closed
				this.chanTime += time.Since(t)
				this.stopped = true
				break loop
			}

			this.chanTime += time.Since(t)
		}

		this.notifyStop()
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

// Override if needed
func (this *base) readonly() bool {
	return true
}

// Unblock all dependencies.
func (this *base) notify() {
	this.notifyStop()
	this.notifyParent()
}

// Notify parent, if any.
func (this *base) notifyParent() {
	parent := this.parent
	if parent != nil {
		// Block on parent
		parent.ChildChannel() <- int(this.bit)
		this.parent = nil
	}
}

// Notify upstream to stop.
func (this *base) notifyStop() {
	stop := this.stop
	if stop != nil {
		select {
		case stop.StopChannel() <- 0:
		default:
			// Already notified.
		}

		this.stop = nil
	}
}

type batcher interface {
	allocateBatch()
	enbatch(item value.AnnotatedValue, b batcher, context *Context) bool
	enbatchSize(item value.AnnotatedValue, b batcher, batchSize int, context *Context) bool
	flushBatch(context *Context) bool
	releaseBatch()
}

var _BATCH_SIZE = 64

var _BATCH_POOL go_atomic.Value
var _JOIN_BATCH_POOL go_atomic.Value

func SetPipelineBatch(size int) {
	if size < 1 {
		size = _BATCH_SIZE
	}

	p := value.NewAnnotatedPool(size)
	_BATCH_POOL.Store(p)
	j := value.NewAnnotatedJoinPairPool(size)
	_JOIN_BATCH_POOL.Store(j)
}

func PipelineBatchSize() int {
	return _BATCH_POOL.Load().(*value.AnnotatedPool).Size()
}

func getBatchPool() *value.AnnotatedPool {
	return _BATCH_POOL.Load().(*value.AnnotatedPool)
}

func getJoinBatchPool() *value.AnnotatedJoinPairPool {
	return _JOIN_BATCH_POOL.Load().(*value.AnnotatedJoinPairPool)
}

func (this *base) allocateBatch() {
	this.batch = getBatchPool().Get()
}

func (this *base) releaseBatch() {
	getBatchPool().Put(this.batch)
	this.batch = nil
}

func (this *base) enbatchSize(item value.AnnotatedValue, b batcher, batchSize int, context *Context) bool {
	if len(this.batch) >= batchSize {
		if !b.flushBatch(context) {
			return false
		}
	}

	if this.batch == nil {
		this.allocateBatch()
	}

	this.batch = append(this.batch, item)
	return true
}

func (this *base) enbatch(item value.AnnotatedValue, b batcher, context *Context) bool {
	return this.enbatchSize(item, b, cap(this.batch), context)
}

func (this *base) requireKey(item value.AnnotatedValue, context *Context) (string, bool) {
	mv := item.GetAttachment("meta")
	if mv == nil {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Value does not contain META: %v", item)))
		return "", false
	}

	meta := mv.(map[string]interface{})
	key, ok := meta["id"]
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("META does not contain ID: %v", item)))
		return "", false
	}

	act := value.NewValue(key).Actual()
	switch act := act.(type) {
	case string:
		return act, true
	default:
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("ID %v of type %T is not a string in value %v", act, act, item)))
		return "", false
	}
}

func (this *base) evaluateKey(keyExpr expression.Expression, item value.AnnotatedValue, context *Context) ([]string, bool) {
	kv, e := keyExpr.Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "keys"))
		return nil, false
	}

	actuals := kv.Actual()
	switch actuals.(type) {
	case []interface{}:
		// Do nothing
	case nil:
		actuals = []interface{}(nil)
	default:
		actuals = []interface{}{actuals}
	}

	acts := actuals.([]interface{})
	keys := make([]string, 0, len(acts))
	for _, key := range acts {
		k := value.NewValue(key).Actual()
		switch k := k.(type) {
		case string:
			keys = append(keys, k)
		}
	}

	return keys, true
}

func (this *base) addTime(t time.Duration) {
	go_atomic.AddInt64((*int64)(&this.execTime), int64(t))
}

func (this *base) marshalTimes(r map[string]interface{}) {
	if this.execTime != 0 {
		r["#time"] = this.execTime.String()
	}
}

// execution destructor is empty by default
// IMPORTANT - please remember to override this method
// in individual operators whenever there are actions
// to be taken on destruction, so as to contain memory
// consumption
func (this *base) Done() {
}
