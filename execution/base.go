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

// Execution operators have a complex life.
// Though the norm is that they are created, they run and they complete, some
// are not even designed to run (eg channel), some will be stopped half way, some
// will be delayed by an overloaded kernel and will only manage to start when the
// request has in fact completed, and some will not start at all (eg their parent
// hasn't managed to start).

// Transitioning states and freeing resources is tricky.
// For starters, cleaning resources can't wait for operators that haven't started to
// complete, because they may never get to start, leading to a fat deadly embrace.
// The Done() method therefore does not wait for latecomers.
// This  means that it is not safe to destruct what hasn't started, because they might
// come to life later and will need certain information to notify other operators:
// _KILLED operators will have to clean up after themselves and remove any residual
// references to other objects so as to help the GC.
// This also means that it is not safe to pool _KILLED operators, as they may later
// come to life
// The Done() method should only be called when it is known that no further actions
// are going to be sent, and the request as completed, either naturally, or via an
// OpStop(), as, as much as we try, it's difficult to control race conditions when
// SendAction() should take decisions based on structures that are being torn down.

// Conversely, dormant operators should never change state during request execution,
// because marking them as inactive will terminate early a result stream.

// It should be safe to pool an operator that has successfully been stopped, but
// our current policy is not to take chances.

// Finally, should a panic occur, it's not safe to clean up, but the operator that
// is terminating in error should still try to notify other operators, so that a stall
// can be avoided.

type opState int

const (
	// not yet active
	_CREATED = opState(iota)
	_DORMANT

	// operating
	_RUNNING
	_STOPPING

	// terminated
	_PANICKED
	_COMPLETED
	_STOPPED

	// paused - ready to reopen
	_PAUSED

	// disposed
	_DONE
	_ENDED
	_KILLED
)

// an operator action can be a STOP or a PAUSE
type opAction int

const (
	_ACTION_STOP = opAction(iota)
	_ACTION_PAUSE
)

type base struct {
	valueExchange
	conn           *datastore.IndexConnection
	stopChannel    stopChannel
	quota          uint64
	input          Operator
	output         Operator
	stop           Operator
	parent         Operator
	once           util.Once
	serializable   bool
	serialized     bool
	inline         bool
	doSend         func(this *base, op Operator, item value.AnnotatedValue) bool
	closeConsumer  bool
	batch          []value.AnnotatedValue
	timePhase      timePhases
	startTime      util.Time
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
	activeCond     sync.Cond
	activeLock     sync.Mutex
	opState        opState
	panicErr       interface{}
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

func newBase(dest *base, context *Context) {
	*dest = base{}
	newValueExchange(&dest.valueExchange, context.GetPipelineCap())
	dest.execPhase = PHASES
	dest.phaseTimes = func(t time.Duration) {}
	dest.activeCond.L = &dest.activeLock
	dest.doSend = parallelSend
	dest.closeConsumer = false
	dest.quota = context.ProducerThrottleQuota()
}

// The output of this operator will be redirected elsewhere, so we
// allocate a minimal itemChannel.
func newRedirectBase(dest *base) {
	*dest = base{}
	newValueExchange(&dest.valueExchange, 1)
	dest.execPhase = PHASES
	dest.phaseTimes = func(t time.Duration) {}
	dest.activeCond.L = &dest.activeLock
	dest.doSend = parallelSend
	dest.closeConsumer = false
}

// This operator will be serialised - allocate valueExchange dynamically
//
// A few ground rules for serializable operators:
// - must always be called in a sequence
// - must follow a producer in a sequence
func newSerializedBase(dest *base, context *Context) {
	*dest = base{}
	newValueExchange(&dest.valueExchange, 1)
	dest.execPhase = PHASES
	dest.phaseTimes = func(t time.Duration) {}
	dest.activeCond.L = &dest.activeLock
	dest.doSend = parallelSend
	dest.closeConsumer = false
	dest.serializable = true
	dest.quota = context.ProducerThrottleQuota()
}

func (this *base) setInline() {
	this.inline = true
}

func (this *base) copy(dest *base) {
	*dest = base{}
	newValueExchange(&dest.valueExchange, int64(cap(this.valueExchange.items)))
	if this.valueExchange.children != nil {
		dest.trackChildren(cap(this.valueExchange.children))
	}
	dest.input = this.input
	dest.output = this.output
	dest.parent = this.parent
	dest.execPhase = this.execPhase
	dest.phaseTimes = this.phaseTimes
	dest.activeCond.L = &dest.activeLock
	dest.serializable = this.serializable
	dest.inline = this.inline
	dest.serialized = false
	dest.doSend = parallelSend
	dest.closeConsumer = false
	dest.quota = this.quota
}

// reset the operator to an initial state
// it's the caller's responsability to make sure the operator has
// stopped, or, at least, will definitely stop: if not this method
// might wait indefinitely
func (this *base) reopen(context *Context) bool {
	return this.baseReopen(context)
}

func (this *base) close(context *Context) {
	err := recover()
	if err != nil {
		this.panicErr = err
		return
	} else if this.panicErr != nil {
		return
	}

	this.valueExchange.close()

	if this.output != nil {

		// MB-27362 avoid serialized close recursion
		if this.closeConsumer {
			base := this.output.getBase()
			serializedClose(this.output, base, context)
		}
	}
	this.inactive()

	// operators that never enter a _RUNNING state have to clean after themselves when they finally go
	if this.opState == _KILLED || this.opState == _PAUSED {
		this.valueExchange.dispose()
		this.stopChannel = nil
		this.input = nil
		this.output = nil
		this.parent = nil
		this.stop = nil
		this.contextTracked = nil
	}
}

// flag terminal early failure (when children don't get to start)
func (this *base) fail(context *Context) {
	this.close(context)
	if this.isRoot {
		context.CloseResults()
	}
}

func (this *base) Done() {
	this.baseDone()
}

func (this *base) baseDone() {
	this.activeCond.L.Lock()

	// if it hasn't started, kill it
	switch this.opState {
	case _CREATED, _PAUSED:
		this.opState = _KILLED
	case _DORMANT:
		this.opState = _DONE

	// otherwise wait
	case _RUNNING, _STOPPING:
		this.activeCond.Wait()
	}

	// from now on, this operator can't be touched
	switch this.opState {
	case _COMPLETED:
		this.opState = _DONE
	case _STOPPED:
		this.opState = _ENDED
	}

	if this.opState == _DONE || this.opState == _ENDED {
		this.valueExchange.dispose()
		this.stopChannel = nil
		this.input = nil
		this.output = nil
		this.parent = nil
		this.stop = nil
		this.contextTracked = nil
	}
	this.activeCond.L.Unlock()
}

// reopen for the terminal operator case
func (this *base) baseReopen(context *Context) bool {
	this.activeCond.L.Lock()

	// still running, just wait
	if this.opState == _CREATED || this.opState == _RUNNING || this.opState == _STOPPING {
		this.activeCond.Wait()
	}

	// the request terminated, a stop was sent, or something catastrophic happened
	// cannot reopen, bail out
	if this.opState == _STOPPED || this.opState == _DONE || this.opState == _ENDED || this.opState == _KILLED || this.opState == _PANICKED {
		this.activeCond.L.Unlock()
		return false
	}

	// opState of _PAUSED is safe to reopen, just follow through and set to _CREATED state

	if this.stopChannel != nil {
		// drain the stop channel
		select {
		case <-this.stopChannel:
		default:
		}
	}
	if this.conn != nil {
		this.conn = nil
	}
	this.contextTracked = nil
	this.childrenLeft = 0
	this.stopped = false
	this.serialized = false
	this.doSend = parallelSend
	this.closeConsumer = false
	this.opState = _CREATED
	this.valueExchange.reset()
	this.once.Reset()
	this.activeCond.L.Unlock()
	return true
}

// setUp

func (this *base) trackChildren(count int) {
	this.valueExchange.trackChildren(count)
}

func (this *base) ValueExchange() *valueExchange {
	return &this.valueExchange
}

func (this *base) exchangeMove(dest *base) {
	this.valueExchange.move(&dest.valueExchange)
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
		this.closeConsumer = true
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

func (this *base) IsSerializable() bool {
	return this.serializable
}

func (this *base) SerializeOutput(op Operator, context *Context) {
	this.output = op
	this.doSend = serializedSend
	this.closeConsumer = true
	base := op.getBase()
	base.serialized = true
	base.contextTracked = context
}

// MB-38469 / go issue 18138 initial goroutine stack too small
//go:noinline
func primeStack() {
	const _STACK_BUF_SIZE = 512 // 128 multiples, tuned for likely stack usage!
	var buf [_STACK_BUF_SIZE]int64

	// force the compiler to allocate buf
	for i := 127; i < _STACK_BUF_SIZE; i += 128 {
		buf[i] = int64(i)
	}

	_ = stackTop(buf[_STACK_BUF_SIZE-1])
}

//go:noinline
func stackTop(v int64) int64 {
	return v
}

func execOp(op Operator, context *Context, parent value.Value) {
	primeStack()
	op.RunOnce(context, parent)
}

// fork operator
func (this *base) fork(op Operator, context *Context, parent value.Value) {
	if op.getBase().inline {
		this.switchPhase(_NOTIME)
		op.RunOnce(context, parent)
		this.switchPhase(_EXECTIME)
	} else {
		go execOp(op, context, parent)
		// go op.RunOnce(context, parent)
	}
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

// stop the operator
func OpStop(op Operator) {
	op.SendAction(_ACTION_STOP)
}

// send an action
func (this *base) SendAction(action opAction) {
	this.baseSendAction(action)
}

// action for the terminal operator case
func (this *base) baseSendAction(action opAction) bool {

	// CREATED and DORMANT cannot apply, as they have neither sent or received
	// PANICKED, COMPLETED and STOPPED have already sent a notifyStop
	// DONE, ENDED and KILLED can no longer be operated upon
	if this.stopped && !this.valueExchange.isWaiting() {
		switch this.opState {
		case _PAUSED:
			if action == _ACTION_PAUSE {
				return true
			}
			// _ACTION_STOP has to take the slow route
		case _RUNNING, _STOPPING:
			return true
		default:
			return false
		}
	}

	// STOPPED, COMPLETED, DONE, ENDED, KILLED have already sent signals or stopped operating
	rv := false
	this.activeCond.L.Lock()
	switch this.opState {
	case _CREATED:
		if action == _ACTION_PAUSE {
			this.opState = _PAUSED
			rv = true
		} else { // _ACTION_STOP
			this.kill()
		}
		this.activeCond.L.Unlock()

	case _PAUSED:
		if action == _ACTION_STOP {
			this.kill()
		} else { // action == _ACTION_PAUSE, no-op
			rv = true
		}
		this.activeCond.L.Unlock()
	case _RUNNING:
		this.opState = _STOPPING
		this.activeCond.L.Unlock()
		rv = true
		this.switchPhase(_CHANTIME)
		this.valueExchange.sendStop()
		this.switchPhase(_EXECTIME)
	case _STOPPING:
		this.activeCond.L.Unlock()
		rv = true
	default:
		this.activeCond.L.Unlock()
	}
	return rv
}

func (this *base) chanSendAction(action opAction) {
	this.activeCond.L.Lock()
	if this.opState == _CREATED {
		if action == _ACTION_PAUSE {
			this.opState = _PAUSED
		} else { // _ACTION_STOP
			this.kill()
		}
		this.activeCond.L.Unlock()
	} else if this.opState == _PAUSED {
		if action == _ACTION_STOP {
			this.kill()
		} // else action == _ACTION_PAUSE, no-op
		this.activeCond.L.Unlock()
	} else if this.opState == _RUNNING {
		this.opState = _STOPPING
		this.activeCond.L.Unlock()
		this.switchPhase(_CHANTIME)
		this.valueExchange.sendStop()
		select {
		case this.stopChannel <- 0:
		default:
		}
		this.switchPhase(_EXECTIME)
	} else {
		this.activeCond.L.Unlock()
	}
}

func (this *base) connSendAction(conn *datastore.IndexConnection, action opAction) {
	this.activeCond.L.Lock()
	if this.opState == _CREATED {
		if action == _ACTION_PAUSE {
			this.opState = _PAUSED
		} else { // _ACTION_STOP
			this.kill()
		}
		this.activeCond.L.Unlock()
	} else if this.opState == _PAUSED {
		if action == _ACTION_STOP {
			this.kill()
		} // else action == _ACTION_PAUSE, no-op
		this.activeCond.L.Unlock()
	} else if this.opState == _RUNNING {
		this.opState = _STOPPING
		this.activeCond.L.Unlock()
		this.switchPhase(_CHANTIME)
		this.valueExchange.sendStop()
		if conn != nil {
			conn.SendStop()
		}
		this.switchPhase(_EXECTIME)
	} else {
		this.activeCond.L.Unlock()
	}
}

func (this *base) kill() {
	this.opState = _KILLED

	// This operator is being killed before it started as part of a request wide OpStop() or Done()
	// it doesn't need to warn anyone else anymore
	this.stop = nil
	this.parent = nil
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
	ok := this.valueExchange.sendItem(op.ValueExchange(), item, this.quota)
	this.switchPhase(_EXECTIME)
	return ok
}

func (this *base) getItem() (value.AnnotatedValue, bool) {
	return this.getItemOp(this.input)
}

func (this *base) getItemOp(op Operator) (value.AnnotatedValue, bool) {
	if this.stopped {
		return nil, false
	}
	this.switchPhase(_CHANTIME)
	val, ok := this.ValueExchange().getItem(op.ValueExchange())
	this.switchPhase(_EXECTIME)
	if !ok {
		this.stopped = true
	}
	return val, ok
}

func (this *base) queuedItems() int {
	return this.ValueExchange().queuedItems(this.input.ValueExchange())
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

func (this *base) getItemEntry(conn *datastore.IndexConnection) (*datastore.IndexEntry, bool) {
	this.conn = conn
	if this.stopped {
		return nil, false
	}

	// this is used explictly to get keys from the indexer
	// so by definition we are tracking service time
	this.switchPhase(_SERVTIME)
	item, ok := conn.Sender().GetEntry()
	this.switchPhase(_EXECTIME)
	if !ok {
		this.stopped = true
		return nil, false
	}
	return item, ok
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
func (this *base) childrenWaitNoStop(ops ...Operator) {
	this.switchPhase(_CHANTIME)
	for _, o := range ops {
		b := o.getBase()
		b.activeCond.L.Lock()
		state := b.opState
		b.activeCond.L.Unlock()
		switch state {
		case _RUNNING, _STOPPING, _COMPLETED, _STOPPED:
			// signal reliably sent
			this.ValueExchange().retrieveChildNoStop()
		case _CREATED, _PAUSED, _KILLED, _PANICKED:
			// signal reliably not sent
		default:

			// we are waiting after we've sent a stop but before we have terminated
			// flag bad states
			assert(false, fmt.Sprintf("child has unexpected state %v", state))
		}
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
		defer context.Recover(this) // Recover from any panic
		active := this.active()
		if this.execPhase != PHASES {
			this.setExecPhase(this.execPhase, context)
		}
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME) // accrue current phase's time
		if this.serialized == true {
			ok := true
			if !active || (context.Readonly() && !cons.readonly()) {
				ok = false
			} else {
				ok = cons.beforeItems(context, parent)
			}

			if ok {
				this.fork(this.input, context, parent)
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
		if !active {
			return
		}

		if context.Readonly() && !cons.readonly() {

			// TODO reinstate assertion on inputs: except all seems to run without one?
			// || !context.assert(this.input != nil, "consumer input is nil") {
			return
		}

		ok := cons.beforeItems(context, parent)

		if ok {
			this.switchPhase(_NOTIME)
			this.fork(this.input, context, parent)
			this.switchPhase(_EXECTIME)
		}

		var item value.AnnotatedValue
		for ok {
			item, ok = this.getItem()
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
			if rv {
				opBase.addInDocs(1)
			} else {
				opBase.stopped = true
				opBase.notifyStop()
			}
		}

		// closing channels and after items in the consumer
		// will be done after the producer has stopped
	}
	opBase.switchPhase(_NOTIME)
	this.switchPhase(_EXECTIME)
	return rv
}

// mark a serialized operator as closed and inactive
func serializedClose(op Operator, opBase *base, context *Context) {
	if !opBase.stopped {
		opBase.stopped = true
		opBase.notifyStop()
	}
	op.afterItems(context)
	opBase.notifyParent()
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
	err := recover()
	if err != nil {
		this.panicErr = err
		return
	}
	if this.panicErr == nil {
		this.notifyStop()
		this.notifyParent()
	}
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
		var action opAction
		this.activeCond.L.Lock()

		// if we stopped normally, flag that a reopen is possible
		// if not, just stop for good
		if this.opState == _RUNNING || this.opState == _COMPLETED {
			action = _ACTION_PAUSE
		} else {
			action = _ACTION_STOP
		}
		this.activeCond.L.Unlock()
		stop.SendAction(action)
		this.stop = nil
	}
}

func (this *base) scanDeltaKeyspace(keyspace datastore.Keyspace, parent value.Value,
	phase Phases, context *Context, covers expression.Covers) (keys map[string]bool, pool bool) {

	pipelineCap := int(context.GetPipelineCap())
	if pipelineCap <= _STRING_BOOL_POOL.Size() {
		keys = _STRING_BOOL_POOL.Get()
		pool = true
	} else {
		keys = make(map[string]bool, pipelineCap)
	}

	conn := datastore.NewIndexConnection(context)
	defer conn.Dispose()
	defer conn.SendStop()

	go context.datastore.TransactionDeltaKeyScan(keyspace.QualifiedName(), conn)

	var docs uint64
	defer func() {
		if docs > 0 {
			context.AddPhaseCount(phase, docs)
		}
	}()

	for {
		entry, ok := this.getItemEntry(conn)
		if ok {
			if entry != nil {
				if entry.MetaData == nil {
					av := this.newEmptyDocumentWithKey(entry.PrimaryKey, parent, context)
					av.SetBit(this.bit)
					if len(covers) > 0 { // only primary key
						av.SetCover(covers[len(covers)-1].Text(), value.NewValue(entry.PrimaryKey))
					}
					ok = this.sendItem(av)
					docs++
					if docs > _PHASE_UPDATE_COUNT {
						context.AddPhaseCount(phase, docs)
						docs = 0
					}
				}
				keys[entry.PrimaryKey] = true
			} else {
				break
			}
		} else {
			return
		}
	}
	return
}

func (this *base) deltaKeyspaceDone(keys map[string]bool, pool bool) (map[string]bool, bool) {
	if pool {
		_STRING_BOOL_POOL.Put(keys)
	}
	return nil, false
}

type batcher interface {
	allocateBatch(context *Context, size int)
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

func (this *base) allocateBatch(context *Context, size int) {
	if size <= PipelineBatchSize() {
		this.batch = getBatchPool().Get()
	} else {
		this.batch = make(value.AnnotatedValues, 0, size)
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
		this.allocateBatch(context, batchSize)
	}

	this.batch = append(this.batch, item)
	return true
}

func (this *base) enbatch(item value.AnnotatedValue, b batcher, context *Context) bool {
	return this.enbatchSize(item, b, cap(this.batch), context)
}

func (this *base) newEmptyDocumentWithKey(key interface{}, parent value.Value, context *Context) value.AnnotatedValue {
	cv := value.NewNestedScopeValue(parent)
	av := value.NewAnnotatedValue(cv)
	av.SetId(key)
	return av
}

func (this *base) setDocumentKey(key interface{}, item value.AnnotatedValue,
	expiration uint32, context *Context) value.AnnotatedValue {
	item.NewMeta()["expiration"] = expiration
	item.SetId(key)
	return item
}

func (this *base) getDocumentKey(item value.AnnotatedValue, context *Context) (string, bool) {

	// fast path for where value Id is in use
	key := item.GetId()
	if key != nil {
		switch key := key.(type) {
		case string:
			return key, true
		case value.Value:
			if key.Type() == value.STRING {
				return key.Actual().(string), true
			}
		}
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("ID %v of type %T is not a string in value %v", key, key, item)))
		return "", false
	} else {

		// slow path (to be deprecated)
		meta := item.GetMeta()
		if meta == nil {
			context.Error(errors.NewInvalidValueError(
				fmt.Sprintf("Value does not contain META: %v", item)))
			return "", false
		}

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

	// we are good to go
	if this.opState == _CREATED {
		this.opState = _RUNNING
		return true
	}

	// we have been killed before we started!
	return false
}

func (this *base) dormant() {
	this.activeCond.L.Lock()
	this.opState = _DORMANT
	this.activeCond.L.Unlock()
}

func (this *base) inactive() {
	this.activeCond.L.Lock()

	// we are done
	switch this.opState {
	case _RUNNING:
		this.opState = _COMPLETED
	case _STOPPING, _PAUSED:
		this.opState = _STOPPED
	}
	this.activeCond.L.Unlock()

	// wake up whoever wants to free us
	this.activeCond.Broadcast()
}

// do any op require to release request in case of a panic
func (this *base) release(context *Context) {

	// signal that we are not in a good place
	this.activeCond.L.Lock()
	this.opState = _PANICKED
	this.activeCond.L.Unlock()

	// release any consumer attached to us
	if this.output != nil && this.closeConsumer {
		base := this.output.getBase()
		serializedClose(this.output, base, context)
	}

	// release any waiter
	this.notify()

	// remove any reference we have about anyone else
	this.stopChannel = nil
	this.input = nil
	this.output = nil
	this.parent = nil
	this.stop = nil
	this.contextTracked = nil
}

func (this *base) waitComplete() {
	this.activeCond.L.Lock()

	// still running, just wait
	if this.opState == _CREATED || this.opState == _PAUSED || this.opState == _RUNNING || this.opState == _STOPPING {
		this.activeCond.Wait()

		// signal that no go routine should touch this operator
		switch this.opState {
		case _COMPLETED:
			this.opState = _DONE
		case _STOPPED:
			this.opState = _ENDED
		}
	}

	this.activeCond.L.Unlock()
}

func (this *base) isComplete() bool {
	return this.opState == _DONE
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
	this.startTime = util.Now()

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
		d = util.Since(this.startTime)
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

	if this.valueExchange.beatYields > 0 {
		stats["#heartbeatYields"] = this.valueExchange.beatYields
	}
	if this.valueExchange.maxSize > 0 {
		stats["usedMemory"] = this.valueExchange.maxSize
		if this.valueExchange.memYields > 0 {
			stats["#memYields"] = this.valueExchange.memYields
		}
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
