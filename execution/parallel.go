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
	"encoding/json"
	"runtime"

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Parallel struct {
	base
	plan         *plan.Parallel
	child        Operator
	children     []Operator
	childChannel StopChannel
}

func NewParallel(plan *plan.Parallel, child Operator) *Parallel {
	rv := &Parallel{
		base:         newBase(),
		plan:         plan,
		child:        child,
		childChannel: make(StopChannel, runtime.NumCPU()),
	}

	rv.output = rv
	return rv
}

func (this *Parallel) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParallel(this)
}

func (this *Parallel) Copy() Operator {
	return &Parallel{
		base:         this.base.copy(),
		plan:         this.plan,
		child:        this.child.Copy(),
		childChannel: make(StopChannel, runtime.NumCPU()),
	}
}

func (this *Parallel) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		n := util.MinInt(this.plan.MaxParallelism(), context.MaxParallelism())
		this.children = _PARALLEL_POOL.Get()[0:n]

		for i := 1; i < n; i++ {
			this.children[i] = this.child.Copy()
			go this.runChild(this.children[i], context, parent)
		}

		this.children[0] = this.child
		go this.runChild(this.children[0], context, parent)

		this.switchPhase(_CHANTIME)
		for n > 0 {
			select {
			case <-this.childChannel: // Never closed
				// Wait for all children
				n--
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				notifyChildren(this.children...)
			}
		}
	})
}

func (this *Parallel) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *Parallel) runChild(child Operator, context *Context, parent value.Value) {
	child.SetInput(this.input)
	child.SetOutput(this.output)
	child.SetParent(this)
	child.SetStop(nil)
	child.RunOnce(context, parent)
}

func (this *Parallel) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)

		// TODO this only marshals times for the first child
		r["~child"] = this.child
	})
	return json.Marshal(r)
}

func (this *Parallel) Done() {
	for c, child := range this.children {
		child.Done()
		this.children[c] = nil
	}
	_PARALLEL_POOL.Put(this.children)
	this.children = nil
}

var _PARALLEL_POOL = NewOperatorPool(runtime.NumCPU())
