//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

// IMPORTANT - please remember to override the opener, destructor and
// message methods in individual operators whenever there are actions
// to be taken on children operators, so as to ensure correct operation
// and avoid hangs destruction and contain memory consumption

import (
	"fmt"
	"reflect"
	"sync"
	go_atomic "sync/atomic"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type timePhases int

const (
	_NOTIME = timePhases(iota)
	_EXECTIME
	_CHANTIME
	_SERVTIME
)

var _PHASENAMES = []string{
	_NOTIME:   "",
	_EXECTIME: "running",
	_CHANTIME: "kernel",
	_SERVTIME: "services",
}

type annotatedChannel chan value.AnnotatedValue

type base struct {
	valueExchange
	stopChannel    stopChannel
	input          Operator
	output         Operator
	stop           Operator
	parent         Operator
	once           util.Once
	serializable   bool
	serialized     bool
	doSend         func(this *base, op Operator, item value.AnnotatedValue) bool
	batch          []value.AnnotatedValue
	timePhase      timePhases
	startTime      time.Time
	execPhase      Phases
	phaseTimes     func(time.Duration)
	execTime       time.Duration
	chanTime       time.Duration
	servTime       time.Duration
	inDocs         int64
	outDocs        int64
	phaseSwitches  int64
	stopped        bool
	isRoot         bool
	bit            uint8
	contextTracked *Context
	childrenLeft   int32
	activeCond     *sync.Cond
	activeLock     sync.Mutex
	primed         bool
	completed      bool
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

func SetPipelineCap(pcap int64) {
	if pcap < 1 {
		pcap = _ITEM_CAP
	}
	atomic.StoreInt64(&pipelineCap, pcap)
}

func GetPipelineCap() int64 {
	pcap := atomic.LoadInt64(&pipelineCap)
	if pcap > 0 {
		return pcap
	} else {
		return _ITEM_CAP
	}
}

func (this *base) getBase() *base {
	return this
}

// Constructor, (re)opener, closer, destructor

func newBase(base *base, context *Context) {
	newValueExchange(&base.valueExchange, context.GetPipelineCap())
	base.execPhase = PHASES
	base.phaseTimes = func(t time.Duration) {}
	base.activeCond = sync.NewCond(&base.activeLock)
	base.doSend = parallelSend
}

// The output of this operator will be redirected elsewhere, so we
// allocate a minimal itemChannel.
func newRedirectBase(base *base) {
	newValueExchange(&base.valueExchange, 1)
	base.execPhase = PHASES
	base.phaseTimes = func(t time.Duration) {}
	base.activeCond = sync.NewCond(&base.activeLock)
	base.doSend = parallelSend
}

func (this *base) copy(base *base) {
	newValueExchange(&base.valueExchange, int64(cap(this.valueExchange.items)))
	if this.valueExchange.children != nil {
		base.trackChildren(cap(this.valueExchange.children))
	}
	base.input = this.input
	base.output = this.output
	base.parent = this.parent
	base.execPhase = this.execPhase
	base.phaseTimes = this.phaseTimes
	base.activeCond = sync.NewCond(&base.activeLock)
	base.serializable = this.serializable
	base.serialized = false
	base.doSend = parallelSend
}

// reset the operator to an initial state
// it's the caller's responsability to make sure the operator has
// stopped, or, at least, will definitely stop: if not this method
// might wait indefinitely
func (this *base) reopen(context *Context) {
	this.baseReopen(context)
}

func (this *base) close(context *Context) {
	this.valueExchange.close()
	if this.output != nil {
		base := this.output.getBase()
		if base.serialized {
			serializedClose(this.output, base, context)
		}
	}
	this.inactive()
}

func (this *base) Done() {
	this.baseDone()
}

func (this *base) baseDone() {
	this.wait()
	this.valueExchange.dispose()
}

// reopen for the terminal operator case
func (this *base) baseReopen(context *Context) {
	this.wait()
	this.valueExchange.reset()
	this.once.Reset()
	this.contextTracked = nil
	this.childrenLeft = 0
	this.primed = false
	this.stopped = false
	this.serialized = false
	this.doSend = parallelSend
	this.activeCond.L.Lock()
	this.completed = false
	this.activeCond.L.Unlock()
}

// setUp

func (this *base) trackChildren(count int) {
	this.valueExchange.trackChildren(count)
}

func (this *base) ValueExchange() *valueExchange {
	return &this.valueExchange
}

// for those operators that really use channels
func (this *base) newStopChannel() {
	this.stopChannel = make(stopChannel, 1)
}

func (this *base) stopCh() stopChannel {
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
	base := op.getBase()

	// propagate inline operators
	if base != this && base.serialized {
		this.doSend = serializedSend
	} else {
		this.doSend = parallelSend
	}
}

func (this *base) Stop() Operator {
	return this.stop
}

func (this *base) SetStop(op Operator) {
	this.stop = op
}

func (this *base) Parent() Operator {
	return this.parent
}

func (this *base) SetParent(parent Operator) {
	this.parent = parent
}

func (this *base) Bit() uint8 {
	return this.bit
}

func (this *base) SetBit(b uint8) {
	this.bit = b
}

func (this *base) SetRoot() {
	this.isRoot = true
}

func (this *base) SetKeepAlive(children int, context *Context) {
	this.contextTracked = context
	this.childrenLeft = int32(children)
}

func (this *base) SetSerializable() {
	this.serializable = true
}

func (this *base) IsSerializable() bool {
	return this.serializable
}

func (this *base) SerializeOutput(op Operator, context *Context) {
	this.output = op
	this.doSend = serializedSend
	base := op.getBase()
	base.serialized = true
	base.contextTracked = context
}

// value and message exchange
//
// The rules are simple - we always receive from input and send onto output.
// Use SetInput() and SetOutput() as required.
// Output by default set to our own item channel.
// If you need to receive from a specific operator, set your input to that operator.
// If you need to fan out - set multiple inputs to the same producer operator
// If you need to fan in - create a channel operator, set the producer outputs to
// the channel, set the consumer input to the channel.
//
// The boolean return value is always true unless a stop signal has been received.
// The returned item is nil on no more data (usually, a channel close).
// The child return value is >=0 if a child message has been received.

// send a stop
func (this *base) SendStop() {
	this.baseSendStop()
}

// stop for the terminal operator case
func (this *base) baseSendStop() {
	if this.stopped || this.completed {
		return
	}
	this.switchPhase(_CHANTIME)
	this.valueExchange.sendStop()
	this.switchPhase(_EXECTIME)
}

func (this *base) chanSendStop() {
	if this.completed {
		return
	}
	this.switchPhase(_CHANTIME)
	this.valueExchange.sendStop()
	select {
	case this.stopChannel <- 0:
	default:
	}
	this.switchPhase(_EXECTIME)
}

func (this *base) sendItem(item value.AnnotatedValue) bool {
	return this.sendItemOp(this.output, item)
}

func (this *base) sendItemOp(op Operator, item value.AnnotatedValue) bool {
	if this.stopped {
		return false
	}
	ok := this.doSend(this, op, item)
	if ok {

		// sendItem tracks outgoing documents for most operators
		this.addOutDocs(1)
	} else {
		this.stopped = true
	}
	return ok
}

// send data down a channel
func parallelSend(this *base, op Operator, item value.AnnotatedValue) bool {
	this.switchPhase(_CHANTIME)
	ok := this.valueExchange.sendItem(op.ValueExchange(), item)
	this.switchPhase(_EXECTIME)
	return ok
}

func (this *base) getItem() (value.AnnotatedValue, bool) {
	return this.getItemOp(this.input)
}

func (this *base) getItemOp(op Operator) (value.AnnotatedValue, bool) {
	this.switchPhase(_CHANTIME)
	val, ok := this.ValueExchange().getItem(op.ValueExchange())
	this.switchPhase(_EXECTIME)
	if !ok {
		this.stopped = true
	}
	return val, ok
}

func (this *base) getItemValue(channel value.ValueChannel) (value.Value, bool) {
	this.switchPhase(_CHANTIME)
	defer this.switchPhase(_EXECTIME)

	select {
	case <-this.stopChannel: // Never closed
		this.stopped = true
		return nil, false
	default:
	}

	select {
	case item, ok := <-channel:
		if ok {

			// getItemValue does not keep track of
			// incoming documents
			return item, true
		}

		// no more data
		return nil, true
	case <-this.stopChannel: // Never closed
		this.stopped = true
		return nil, false
	}
}

func (this *base) getItemEntry(channel datastore.EntryChannel) (*datastore.IndexEntry, bool) {

	// this is used explictly to get keys from the indexer
	// so by definition we are tracking service time
	this.switchPhase(_SERVTIME)
	defer this.switchPhase(_EXECTIME)

	select {
	case <-this.stopChannel: // Never closed
		this.stopped = true
		return nil, false
	default:
	}

	select {
	case item, ok := <-channel:
		if ok {

			// getItemEntry does not keep track of
			// incoming documents
			return item, true
		}

		// no more data
		return nil, true
	case <-this.stopChannel: // Never closed
		this.stopped = true
		return nil, false
	}
}

func (this *base) getItemChildren() (value.AnnotatedValue, int, bool) {
	return this.getItemChildrenOp(this.input)
}

func (this *base) getItemChildrenOp(op Operator) (value.AnnotatedValue, int, bool) {
	this.switchPhase(_CHANTIME)
	val, child, ok := this.ValueExchange().getItemChildren(op.ValueExchange())
	this.switchPhase(_EXECTIME)
	if !ok {
		this.stopped = true
	}
	return val, child, ok
}

// wait for at least n children to complete
func (this *base) childrenWait(n int) bool {
	this.switchPhase(_CHANTIME)
	for n > 0 {

		// no values are actually coming
		child, ok := this.ValueExchange().retrieveChild()
		if !ok {
			this.stopped = true
			this.switchPhase(_EXECTIME)
			return false
		}
		if child >= 0 {
			n--
		}
	}

	this.switchPhase(_EXECTIME)
	return true
}

// wait for at least n children to complete ignoring stop messages
func (this *base) childrenWaitNoStop(n int) {
	this.switchPhase(_CHANTIME)
	for n > 0 {
		this.ValueExchange().retrieveChildNoStop()
		n--
	}
	this.switchPhase(_EXECTIME)
}

type consumer interface {
	beforeItems(context *Context, parent value.Value) bool
	processItem(item value.AnnotatedValue, context *Context) bool
	afterItems(context *Context)
	readonly() bool
}

func (this *base) runConsumer(cons consumer, context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		if this.execPhase != PHASES {
			this.setExecPhase(this.execPhase, context)
		}
		this.switchPhase(_EXECTIME)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		if this.serialized == true {
			ok := true
			if !active || (context.Readonly() && !cons.readonly()) {
				ok = false
			} else {
				ok = cons.beforeItems(context, parent)
			}

			if ok {
				go this.input.RunOnce(context, parent)
			}

			if !ok {
				this.notify()
				this.close(context)
			}
			return
		}
		defer this.close(context)
		defer this.notify() // Notify that I have stopped
		defer func() { this.batch = nil }()

		if !active || (context.Readonly() && !cons.readonly()) {
			return
		}

		ok := cons.beforeItems(context, parent)

		if ok {
			go this.input.RunOnce(context, parent)
		}

		for ok {
			item, ok := this.getItem()
			if !ok || item == nil {
				break
			}
			this.addInDocs(1)
			ok = cons.processItem(item, context)
		}

		this.notifyStop()
		cons.afterItems(context)
	})
}

// fire an operator's processItem
func serializedSend(this *base, op Operator, item value.AnnotatedValue) bool {
	rv := false
	if this.isStopped() {
		return rv
	}
	this.switchPhase(_NOTIME)
	opBase := op.getBase()
	opBase.switchPhase(_EXECTIME)
	if !opBase.stopped {
		if opBase.isStopped() {
			this.stopped = true
		} else {
			rv = op.processItem(item, opBase.contextTracked)
		}
		if !rv {
			serializedClose(op, opBase, opBase.contextTracked)
		}
	}
	opBase.switchPhase(_NOTIME)
	this.switchPhase(_EXECTIME)
	return rv
}

// mark a serialized operator as closed and inactive
func serializedClose(op Operator, opBase *base, context *Context) {
	op.afterItems(context)
	opBase.notify()
	op.close(context)
	opBase.inactive()
}

// Override if needed
func (this *base) beforeItems(context *Context, parent value.Value) bool {
	return true
}

// Override if needed
func (this *base) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.sendItem(item)
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

// release parent resources, if necessary
func (this *base) keepAlive(op Operator) bool {
	if this.childrenLeft == 0 {
		return false
	}
	if go_atomic.AddInt32(&this.childrenLeft, -1) == 0 {
		this.notify()
		op.close(this.contextTracked)
	}
	return true
}

// Notify parent, if any.
func (this *base) notifyParent() {
	parent := this.parent
	if parent != nil && !parent.keepAlive(parent) {

		// Block on parent
		this.switchPhase(_CHANTIME)
		parent.ValueExchange().sendChild(int(this.bit))
		this.switchPhase(_EXECTIME)
	}
	this.parent = nil
}

// Notify upstream to stop.
func (this *base) notifyStop() {
	stop := this.stop
	if stop != nil {
		stop.SendStop()
		this.stop = nil
	}
}

type batcher interface {
	allocateBatch(context *Context)
	enbatch(item value.AnnotatedValue, b batcher, context *Context) bool
	enbatchSize(item value.AnnotatedValue, b batcher, batchSize int, context *Context) bool
	flushBatch(context *Context) bool
	releaseBatch(context *Context)
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

func (this *base) allocateBatch(context *Context) {
	if context.PipelineBatch() == 0 {
		this.batch = getBatchPool().Get()
	} else {
		this.batch = make(value.AnnotatedValues, 0, context.PipelineBatch())
	}
}

func (this *base) releaseBatch(context *Context) {
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
		this.allocateBatch(context)
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

// operator state handling
func (this *base) active() bool {
	this.activeCond.L.Lock()
	defer this.activeCond.L.Unlock()

	// we have been killed before we started!
	if this.completed {
		return false
	}

	// we are good to go
	this.primed = true

	return true
}

func (this *base) inactive() {
	this.activeCond.L.Lock()

	// we are done
	this.completed = true
	this.activeCond.L.Unlock()

	// wake up whoever wants to free us
	this.activeCond.Signal()
}

func (this *base) wait() {
	this.activeCond.L.Lock()

	// still running, just wait
	if this.primed && !this.completed {

		// technically this should be in a loop testing for completed
		// but there's ever going to be one other actor, and all it
		// does is releases us, so this suffices
		this.activeCond.Wait()
	}

	// signal that no go routine should touch this operator
	this.completed = true
	this.activeCond.L.Unlock()
}

func (this *base) waitComplete() {
	this.activeCond.L.Lock()

	// still running, just wait
	if !this.completed {

		// technically this should be in a loop testing for completed
		// but there's ever going to be one other actor, and all it
		// does is releases us, so this suffices
		this.activeCond.Wait()
	}

	// signal that no go routine should touch this operator
	this.completed = true
	this.activeCond.L.Unlock()
}

// profiling

// phase switching
func (this *base) switchPhase(p timePhases) {
	oldPhase := this.timePhase
	this.timePhase = p

	// not switching phases
	if oldPhase == p {
		return
	}
	oldTime := this.startTime
	this.startTime = time.Now()

	// starting or restarting after a stop
	// either way, no time to accrue as of yet
	if oldPhase == _NOTIME {
		return
	}

	// keep track of phase switching
	go_atomic.AddInt64((*int64)(&this.phaseSwitches), 1)
	d := this.startTime.Sub(oldTime)
	switch oldPhase {
	case _EXECTIME:
		this.addExecTime(d)
		this.phaseTimes(d)
	case _SERVTIME:
		this.addServTime(d)
		this.phaseTimes(d)
	case _CHANTIME:
		this.addChanTime(d)
	}
}

// accrues operators and phase times
func (this *base) setExecPhase(phase Phases, context *Context) {
	context.AddPhaseOperator(phase)
	this.addExecPhase(phase, context)
}

// accrues phase times (useful where we don't want to count operators)
func (this *base) addExecPhase(phase Phases, context *Context) {
	this.phaseTimes = func(t time.Duration) { context.AddPhaseTime(phase, t) }
}

// operator times and items accrual
func (this *base) addExecTime(t time.Duration) {
	go_atomic.AddInt64((*int64)(&this.execTime), int64(t))
}

func (this *base) addChanTime(t time.Duration) {
	go_atomic.AddInt64((*int64)(&this.chanTime), int64(t))
}

func (this *base) addServTime(t time.Duration) {
	go_atomic.AddInt64((*int64)(&this.servTime), int64(t))
}

func (this *base) addInDocs(d int64) {
	go_atomic.AddInt64((*int64)(&this.inDocs), d)
}

func (this *base) addOutDocs(d int64) {
	go_atomic.AddInt64((*int64)(&this.outDocs), d)
}

// profile marshaller
func (this *base) marshalTimes(r map[string]interface{}) {
	var d time.Duration
	stats := make(map[string]interface{}, 6)

	if this.inDocs != 0 {
		stats["#itemsIn"] = this.inDocs
	}
	if this.outDocs != 0 {
		stats["#itemsOut"] = this.outDocs
	}
	if this.phaseSwitches != 0 {
		stats["#phaseSwitches"] = this.phaseSwitches
	}

	execTime := this.execTime
	chanTime := this.chanTime
	servTime := this.servTime
	if this.timePhase != _NOTIME {
		d = time.Since(this.startTime)
		switch this.timePhase {
		case _EXECTIME:
			execTime += d
		case _SERVTIME:
			servTime += d
		case _CHANTIME:
			chanTime += d
		}
		stats["state"] = _PHASENAMES[this.timePhase]
	}

	if execTime != 0 {
		stats["execTime"] = execTime.String()
	}
	if chanTime != 0 {
		stats["kernTime"] = chanTime.String()
	}
	if servTime != 0 {
		stats["servTime"] = servTime.String()
	}

	// chosen to follow "#operator" in the subdocument
	if len(stats) > 0 {
		r["#stats"] = stats
	}

	// chosen to go at the end of the plan
	if this.isRoot {
		var versions []interface{}

		versions = append(versions, util.VERSION)
		versions = append(versions, datastore.GetDatastore().Info().Version())
		r["~versions"] = versions
	}
}

// the following functions are used to sum execution
// times of children of the parallel operator
// 1- tot up times
func (this *base) accrueTime(copy *base) {
	this.inDocs += copy.inDocs
	this.outDocs += copy.outDocs
	this.phaseSwitches += copy.phaseSwitches
	this.execTime += copy.execTime
	this.chanTime += copy.chanTime
	this.servTime += copy.servTime
}

// 2- descend children: default for childless operators
func (this *base) accrueTimes(copy Operator) {
	this.accrueTime(copy.time())
}

// 3- times to be copied
func (this *base) time() *base {
	return this
}

// 4- check and add operator times
func baseAccrueTimes(o1, o2 Operator) bool {
	t1 := reflect.TypeOf(o1)
	t2 := reflect.TypeOf(o2)
	if !assert(t1 == t2, "mismatching operators detected") {
		return true
	}
	o1.accrueTime(o2.time())
	return false
}

// 5- check and add children
func childrenAccrueTimes(o1, o2 []Operator) bool {
	l1 := len(o1)
	l2 := len(o2)
	if !assert(l1 == l2, "mismatching operator lengths detected") {
		return true
	}
	for i, c := range o1 {
		c.accrueTimes(o2[i])
	}
	return false
}
