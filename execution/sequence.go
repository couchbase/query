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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Sequence struct {
	base
	plan     *plan.Sequence
	children []Operator
}

var _SEQUENCE_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_SEQUENCE_OP_POOL, func() interface{} {
		return &Sequence{}
	})
}

func NewSequence(plan *plan.Sequence, context *Context, children ...Operator) *Sequence {
	rv := _SEQUENCE_OP_POOL.Get().(*Sequence)
	rv.plan = plan
	rv.children = children

	// allocate value exchanges for serialized children if required
	// if not the first operator and the sender has already an operator
	// we'll just use that, since the sender has no use for it
	prevBase := children[0].getBase()
	for i := 1; i < len(children); i++ {
		thisBase := children[i].getBase()
		if thisBase.IsSerializable() {
			prevBase.exchangeMove(thisBase)
		}
		prevBase = thisBase
	}

	// we'll even use the last child's value exchange and save allocating
	// an unused one for ourselves
	newRedirectBase(&rv.base)
	rv.base.setInline()
	prevBase.exchangeMove(&rv.base)
	rv.output = rv
	return rv
}

func (this *Sequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSequence(this)
}

func (this *Sequence) Copy() Operator {
	rv := _SEQUENCE_OP_POOL.Get().(*Sequence)
	children := _SEQUENCE_POOL.Get()

	for _, child := range this.children {
		children = append(children, child.Copy())
	}

	rv.plan = this.plan
	rv.children = children
	this.base.copy(&rv.base)
	return rv
}

func (this *Sequence) PlanOp() plan.Operator {
	return this.plan
}

func (this *Sequence) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		this.SetKeepAlive(1, context)

		n := len(this.children)
		if !active || !context.assert(n > 0, "Sequence has no children") {
			this.notify()
			this.fail(context)
			return
		}

		curr := this.children[0]
		curr.SetInput(this.input)
		curr.SetStop(this.stop)

		// Define all Inputs and Outputs
		var next Operator
		for i := 1; i < n; i++ {
			next = this.children[i]

			// run the consumer inline, if feasible
			if next.IsSerializable() {
				next.SetInput(curr)
				curr.SerializeOutput(next, context)
			} else {
				curr.SetOutput(curr)
				next.SetInput(curr.Output())
			}
			next.SetStop(curr)
			curr = next
		}

		next.SetOutput(this.output)
		next.SetParent(this)

		// Run last child
		this.fork(next, context, parent)
	})
}

func (this *Sequence) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~children"] = this.children
	})
	return json.Marshal(r)
}

func (this *Sequence) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Sequence)
	childrenAccrueTimes(this.children, copy.children)
}

func (this *Sequence) SendAction(action opAction) {
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

func (this *Sequence) reopen(context *Context) bool {
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

func (this *Sequence) Done() {
	this.baseDone()
	for c, child := range this.children {
		this.children[c] = nil
		child.Done()
	}
	_SEQUENCE_POOL.Put(this.children)
	this.children = nil
	if this.isComplete() {
		_SEQUENCE_OP_POOL.Put(this)
	}
}

var _SEQUENCE_POOL = NewOperatorPool(32)
