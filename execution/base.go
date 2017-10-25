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

type base struct {
	itemChannel   value.AnnotatedChannel
	stopChannel   StopChannel // Never closed
	input         Operator
	output        Operator
	stop          Operator
	parent        Parent
	once          util.Once
	batch         []value.AnnotatedValue
	timePhase     timePhases
	startTime     time.Time
	execPhase     Phases
	phaseTimes    func(time.Duration)
	execTime      time.Duration
	chanTime      time.Duration
	servTime      time.Duration
	inDocs        int64
	outDocs       int64
	phaseSwitches int64
	stopped       bool
	isRoot        bool
	bit           uint8
	activeCond    *sync.Cond
	activeLock    sync.Mutex
	primed        bool
	completed     bool
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

func newBase(base *base, context *Context) {
	base.itemChannel = make(value.AnnotatedChannel, context.GetPipelineCap())
	base.stopChannel = make(StopChannel, 1)
	base.execPhase = PHASES
	base.phaseTimes = func(t time.Duration) {}
	base.activeCond = sync.NewCond(&base.activeLock)
}

// The output of this operator will be redirected elsewhere, so we
// allocate a minimal itemChannel.
func newRedirectBase(base *base) {
	base.itemChannel = make(value.AnnotatedChannel, 1)
	base.stopChannel = make(StopChannel, 1)
	base.execPhase = PHASES
	base.phaseTimes = func(t time.Duration) {}
	base.activeCond = sync.NewCond(&base.activeLock)
}

// IMPORTANT - please remember to override the next three methods
// in individual operators whenever there are actions to be taken on
// children operators, so as to ensure correct operation and avoid
// hangs destruction and contain memory consumption

// send a stop
func (this *base) SendStop() {
	this.baseSendStop()
}

// reset the operator to an initial state
// it's the caller's responsability to make sure the operator has
// stopped, or, at least, will definitely stop: if not this method
// might wait indefinitely
func (this *base) reopen(context *Context) {
	this.baseReopen(context)
}

// execution destructor is empty by default
func (this *base) Done() {
}

// stop for the terminal operator case
func (this *base) baseSendStop() {
	if this.completed {
		return
	}
	this.switchPhase(_CHANTIME)
	select {
	case this.stopChannel <- 0:
	default:
	}
	this.switchPhase(_EXECTIME)
}

// reopen for the terminal operator case
func (this *base) baseReopen(context *Context) {
	this.wait()
	this.itemChannel = make(value.AnnotatedChannel, context.GetPipelineCap())
	this.stopChannel = make(StopChannel, 1)
	this.once.Reset()
	this.primed = false
	this.stopped = false
	this.activeCond.L.Lock()
	this.completed = false
	this.activeCond.L.Unlock()
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

func (this *base) SetRoot() {
	this.isRoot = true
}

func (this *base) copy(base *base) {
	base.itemChannel = make(value.AnnotatedChannel, cap(this.itemChannel))
	base.stopChannel = make(StopChannel, 1)
	base.input = this.input
	base.output = this.output
	base.parent = this.parent
	base.execPhase = this.execPhase
	base.phaseTimes = this.phaseTimes
	base.activeCond = sync.NewCond(&base.activeLock)
}

func (this *base) sendItem(item value.AnnotatedValue) bool {
	this.switchPhase(_CHANTIME)
	defer this.switchPhase(_EXECTIME)

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

		// sendItem keeps track of outgoing
		// documengs for most operators
		this.addOutDocs(1)
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
		defer context.Recover() // Recover from any panic
		if this.execPhase != PHASES {
			this.setExecPhase(this.execPhase, context)
		}
		this.switchPhase(_EXECTIME)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer close(this.itemChannel)                // Broadcast that I have stopped
		defer this.notify()                          // Notify that I have stopped
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
			this.switchPhase(_CHANTIME)

			select {
			case <-this.stopChannel: // Never closed
				this.switchPhase(_EXECTIME)
				this.stopped = true
				break loop
			default:
			}

			select {
			case item, ok = <-this.input.ItemChannel():
				this.switchPhase(_EXECTIME)
				if ok {

					// runConsumer keeps track of incoming
					// documents for most operators
					this.addInDocs(1)
					ok = cons.processItem(item, context)
				}
			case <-this.stopChannel: // Never closed
				this.switchPhase(_EXECTIME)
				this.stopped = true
				break loop
			}
		}

		this.notifyStop()
		cons.afterItems(context)
	})
}

// actions to be taken if runConsumer() doesn't get to run
func (this *base) releaseConsumer() {
	defer close(this.itemChannel) // Broadcast that I have stopped
	defer this.notify()           // Notify that I have stopped
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
		this.switchPhase(_CHANTIME)
		parent.ChildChannel() <- int(this.bit)
		this.switchPhase(_EXECTIME)
		this.parent = nil
	}
}

// Notify upstream to stop.
func (this *base) notifyStop() {
	stop := this.stop
	if stop != nil {
		this.switchPhase(_CHANTIME)
		select {
		case stop.StopChannel() <- 0:
		default:
			// Already notified.
		}
		this.switchPhase(_EXECTIME)

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
