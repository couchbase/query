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
	"github.com/couchbase/query/value"
)

type Sequence struct {
	base
	plan     *plan.Sequence
	children []Operator
}

func NewSequence(plan *plan.Sequence, context *Context, children ...Operator) *Sequence {
	rv := &Sequence{
		plan:     plan,
		children: children,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Sequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSequence(this)
}

func (this *Sequence) Copy() Operator {
	children := _SEQUENCE_POOL.Get()

	for _, child := range this.children {
		children = append(children, child.Copy())
	}

	rv := &Sequence{
		plan:     this.plan,
		children: children,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Sequence) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		this.SetKeepAlive(1, context)

		n := len(this.children)
		if !active || !context.assert(n > 0, "Sequence has no children") {
			this.close(context)
			return
		}

		first_child := this.children[0]
		first_child.SetInput(this.input)
		first_child.SetStop(this.stop)

		// Define all Inputs and Outputs
		for i := 0; i < n-1; i++ {
			curr := this.children[i]
			next := this.children[i+1]

			// run the consumer inline, if feasible
			if next.IsSerializable() {
				next.SetInput(curr)
				curr.SerializeOutput(next, context)
			} else {
				curr.SetOutput(curr)
				next.SetInput(curr.Output())
			}
			next.SetStop(curr)
		}

		last_child := this.children[n-1]

		last_child.SetOutput(this.output)
		last_child.SetParent(this)

		// Run last child
		go last_child.RunOnce(context, parent)
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

func (this *Sequence) SendStop() {
	this.baseSendStop()
	for _, child := range this.children {
		if child != nil {
			child.SendStop()
		}
	}
}

func (this *Sequence) reopen(context *Context) {
	this.baseReopen(context)
	for _, child := range this.children {
		child.reopen(context)
	}
}

func (this *Sequence) Done() {
	this.baseDone()
	for c, child := range this.children {
		child.Done()
		this.children[c] = nil
	}
	_SEQUENCE_POOL.Put(this.children)
	this.children = nil
}

var _SEQUENCE_POOL = NewOperatorPool(32)
