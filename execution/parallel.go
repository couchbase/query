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
	"runtime"

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Parallel struct {
	base
	plan         *plan.Parallel
	child        Operator
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
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		n := util.MinInt(this.plan.MaxParallelism(), context.MaxParallelism())
		children := _PARALLEL_POOL.Get()[0:n]
		defer _PARALLEL_POOL.Put(children)

		for i := 1; i < n; i++ {
			children[i] = this.child.Copy()
			go this.runChild(children[i], context, parent)
		}

		children[0] = this.child
		go this.runChild(children[0], context, parent)

		for n > 0 {
			select {
			case <-this.childChannel: // Never closed
				// Wait for all children
				n--
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				notifyChildren(children...)
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

var _PARALLEL_POOL = NewOperatorPool(runtime.NumCPU())
