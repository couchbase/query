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

type UnionAll struct {
	base
	plan         *plan.UnionAll
	children     []Operator
	childChannel StopChannel
}

func NewUnionAll(plan *plan.UnionAll, context *Context, children ...Operator) *UnionAll {
	rv := &UnionAll{
		plan:         plan,
		children:     children,
		childChannel: make(StopChannel, len(children)),
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *UnionAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionAll(this)
}

func (this *UnionAll) Copy() Operator {
	rv := &UnionAll{
		plan:         this.plan,
		childChannel: make(StopChannel, len(this.children)),
	}
	this.base.copy(&rv.base)

	children := _UNION_POOL.Get()

	for _, c := range this.children {
		children = append(children, c.Copy())
	}

	rv.children = children
	return rv
}

func (this *UnionAll) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		defer this.inactive() // signal that resources can be freed
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		n := len(this.children)
		if !active || !context.assert(n > 0, "Union has no children") {
			return
		}

		// Run children in parallel
		for _, child := range this.children {
			child.SetOutput(this.output)
			child.SetStop(nil)
			child.SetParent(this)
			go child.RunOnce(context, parent)
		}

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

		context.SetSortCount(0)
	})
}

func (this *UnionAll) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *UnionAll) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~children"] = this.children
	})
	return json.Marshal(r)
}

func (this *UnionAll) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*UnionAll)
	childrenAccrueTimes(this.children, copy.children)
}

func (this *UnionAll) SendStop() {
	this.baseSendStop()
	for _, child := range this.children {
		child.SendStop()
	}
}

func (this *UnionAll) reopen(context *Context) {
	this.baseReopen(context)
	this.childChannel = make(StopChannel, len(this.children))
	for _, child := range this.children {
		child.reopen(context)
	}
}

func (this *UnionAll) Done() {
	this.wait()
	for c, child := range this.children {
		child.Done()
		this.children[c] = nil
	}
	_UNION_POOL.Put(this.children)
	this.children = nil
}

var _UNION_POOL = NewOperatorPool(4)
