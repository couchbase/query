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
	plan     *plan.Parallel
	child    Operator
	children []Operator
}

func NewParallel(plan *plan.Parallel, context *Context, child Operator) *Parallel {
	rv := &Parallel{
		plan:  plan,
		child: child,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Parallel) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParallel(this)
}

func (this *Parallel) Copy() Operator {
	rv := &Parallel{
		plan:  this.plan,
		child: this.child.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Parallel) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		n := util.MinInt(this.plan.MaxParallelism(), context.MaxParallelism())
		this.SetKeepAlive(n, context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)

		if !active || !context.assert(this.child != nil, "Parallel has no child") {
			return
		}
		this.children = _PARALLEL_POOL.Get()[0:n]

		for i := 1; i < n; i++ {
			this.children[i] = this.child.Copy()
			go this.runChild(this.children[i], context, parent)
		}

		this.children[0] = this.child
		go this.runChild(this.children[0], context, parent)
	})
}

func (this *Parallel) runChild(child Operator, context *Context, parent value.Value) {
	child.SetInput(this.input)
	child.SetOutput(this.output)
	child.SetParent(this)
	child.SetStop(nil)
	child.RunOnce(context, parent)
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

func (this *Parallel) SendStop() {
	this.baseSendStop()
	for _, child := range this.children {
		child.SendStop()
	}
}

func (this *Parallel) reopen(context *Context) {
	this.baseReopen(context)
	for _, child := range this.children {
		child.reopen(context)
	}
}

func (this *Parallel) Done() {
	this.baseDone()
	for c, child := range this.children {
		child.Done()
		this.children[c] = nil
	}
	_PARALLEL_POOL.Put(this.children)
	this.children = nil
}

var _PARALLEL_POOL = NewOperatorPool(runtime.NumCPU())
