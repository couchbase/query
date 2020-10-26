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

var _PARALLEL_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_PARALLEL_OP_POOL, func() interface{} {
		return &Parallel{}
	})
}

type Parallel struct {
	base
	plan     *plan.Parallel
	child    Operator
	children []Operator
}

func NewParallel(plan *plan.Parallel, context *Context, child Operator) *Parallel {
	rv := _PARALLEL_OP_POOL.Get().(*Parallel)
	rv.plan = plan
	rv.child = child

	// all the children will be using the same value exchange,
	// which is the parallel's output anyway
	// if the child already has a value exchange allocated, we'll use
	// that and avoid needlessly allocating more
	childBase := child.getBase()
	if childBase.cap() > 1 {
		newRedirectBase(&rv.base)
		childBase.exchangeMove(&rv.base)
	} else {
		newBase(&rv.base, context)
	}
	rv.base.setInline()
	rv.output = rv
	return rv
}

func (this *Parallel) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParallel(this)
}

func (this *Parallel) Copy() Operator {
	rv := _PARALLEL_OP_POOL.Get().(*Parallel)
	rv.plan = this.plan
	rv.child = this.child.Copy()
	this.base.copy(&rv.base)
	return rv
}

func (this *Parallel) PlanOp() plan.Operator {
	return this.plan
}

func (this *Parallel) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		n := util.MinInt(this.plan.MaxParallelism(), context.MaxParallelism())
		this.SetKeepAlive(n, context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)

		if !active || !context.assert(this.child != nil, "Parallel has no child") {
			this.notify()
			this.fail(context)
			return
		}
		this.children = _PARALLEL_POOL.Get()[0:n]

		for i := 1; i < n; i++ {
			this.children[i] = this.child.Copy()
			this.runChild(this.children[i], context, parent)
		}

		this.children[0] = this.child
		this.runChild(this.children[0], context, parent)
	})
}

func (this *Parallel) runChild(child Operator, context *Context, parent value.Value) {
	child.SetInput(this.input)
	child.SetOutput(this.output)
	child.SetParent(this)
	child.SetStop(nil)
	this.fork(child, context, parent)
}

func (this *Parallel) MarshalJSON() ([]byte, error) {
	var outChild Operator

	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)

		childCount := len(this.children)
		r["copies"] = childCount

		// when we have multiple copies, we create a temporary child
		// for the purpose of adding up all of the times (remember, the
		// actual query might be running and we are accessing the times
		// via system:active_requests)
		if childCount > 1 {
			outChild = this.child.Copy()
			for _, c := range this.children {
				outChild.accrueTimes(c)
			}
			r["~child"] = outChild
		} else {
			r["~child"] = this.child
		}
	})
	val, err := json.Marshal(r)

	// free up resources of temporary child
	if outChild != nil {
		outChild.Done()
	}
	return val, err
}

func (this *Parallel) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Parallel)
	childrenAccrueTimes(this.children, copy.children)
}

func (this *Parallel) SendAction(action opAction) {
	if this.baseSendAction(action) {
		children := this.children
		for _, child := range children {
			if child != nil {
				child.SendAction(action)
			}
			if this.children == nil {
				break
			}
		}
	}
}

func (this *Parallel) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv {
		for _, child := range this.children {
			if !child.reopen(context) {
				return false
			}
		}
	}
	return rv
}

func (this *Parallel) Done() {
	this.baseDone()
	for c, child := range this.children {
		this.children[c] = nil
		child.Done()
	}
	_PARALLEL_POOL.Put(this.children)
	this.children = nil
	this.child = nil
	if this.isComplete() {
		_PARALLEL_OP_POOL.Put(this)
	}
}

var _PARALLEL_POOL = NewOperatorPool(runtime.NumCPU())
