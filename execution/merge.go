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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Merge struct {
	base
	plan     *plan.Merge
	update   Operator
	delete   Operator
	insert   Operator
	children []Operator
}

func NewMerge(plan *plan.Merge, context *Context, update, delete, insert Operator) *Merge {
	rv := &Merge{
		plan:   plan,
		update: update,
		delete: delete,
		insert: insert,
	}

	newBase(&rv.base, context)
	rv.trackChildren(3)
	rv.output = rv
	return rv
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) Copy() Operator {
	rv := &Merge{
		plan:   this.plan,
		update: copyOperator(this.update),
		delete: copyOperator(this.delete),
		insert: copyOperator(this.insert),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Merge) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(MERGE, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		go this.input.RunOnce(context, parent)

		update, updateInput := this.wrapChild(this.update, context)
		delete, deleteInput := this.wrapChild(this.delete, context)
		insert, insertInput := this.wrapChild(this.insert, context)

		this.children = _MERGE_OPERATOR_POOL.Get()
		inputs := _MERGE_CHANNEL_POOL.Get()
		defer _MERGE_CHANNEL_POOL.Put(inputs)

		if update != nil {
			this.children = append(this.children, update)
			inputs = append(inputs, updateInput)
		}

		if delete != nil {
			this.children = append(this.children, delete)
			inputs = append(inputs, deleteInput)
		}

		if insert != nil {
			this.children = append(this.children, insert)
			inputs = append(inputs, insertInput)
		}

		for _, child := range this.children {
			go child.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		ok := true

		for ok {
			item, ok = this.getItem()
			if !ok || item == nil {
				break
			}
			this.addInDocs(1)
			ok = this.processMatch(item, context, update, delete, insert)
		}

		// Close child input Channels, which will signal children
		for _, input := range inputs {
			input.close(context)
		}

		// Wait for all children
		this.childrenWaitNoStop(len(this.children))
	})
}

func (this *Merge) processMatch(item value.AnnotatedValue,
	context *Context, update, delete, insert Operator) bool {
	kv, e := this.plan.Key().Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "MERGE key"))
		return false
	}

	ka := kv.Actual()
	k, ok := ka.(string)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid MERGE key %v of type %T.", ka, ka)))
		return false
	}

	this.switchPhase(_SERVTIME)

	ok = true
	bvs, errs := this.plan.Keyspace().Fetch([]string{k}, context)

	this.switchPhase(_EXECTIME)

	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			ok = false
		}
	}

	if !ok {
		return false
	}

	if len(bvs) > 0 {
		bv := bvs[0]
		item.SetField(this.plan.KeyspaceRef().Alias(), bv.Value)

		// Perform UPDATE and/or DELETE
		if update != nil {
			ok = this.sendItemOp(update.Input(), item)
		}

		if ok && delete != nil {
			ok = this.sendItemOp(delete.Input(), item)
		}
	} else {
		// Not matched; INSERT
		if insert != nil {
			ok = this.sendItemOp(insert.Input(), item)
		}
	}

	return ok
}

func (this *Merge) wrapChild(op Operator, context *Context) (Operator, *Channel) {
	if op == nil {
		return nil, nil
	}

	ch := NewChannel(context)
	op.SetInput(ch)
	op.SetOutput(this.output)
	op.SetParent(this)
	op.SetStop(this)
	return op, ch
}

func (this *Merge) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		if this.update != nil {
			r["update"] = this.update
		}
		if this.delete != nil {
			r["delete"] = this.delete
		}
		if this.insert != nil {
			r["insert"] = this.insert
		}
	})
	return json.Marshal(r)
}

func (this *Merge) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Merge)
	if this.update != nil {
		this.insert.accrueTimes(copy.insert)
	}
	if this.delete != nil {
		this.update.accrueTimes(copy.update)
	}
	if this.insert != nil {
		this.insert.accrueTimes(copy.insert)
	}
}

func (this *Merge) SendStop() {
	this.baseSendStop()
	if this.update != nil {
		this.update.SendStop()
	}
	if this.delete != nil {
		this.delete.SendStop()
	}
	if this.insert != nil {
		this.insert.SendStop()
	}
}

func (this *Merge) reopen(context *Context) {
	this.baseReopen(context)
	if this.update != nil {
		this.update.reopen(context)
	}
	if this.delete != nil {
		this.delete.reopen(context)
	}
	if this.insert != nil {
		this.insert.reopen(context)
	}
}

func (this *Merge) Done() {
	this.wait()
	if this.update != nil {
		this.update.Done()
		this.update = nil
	}
	if this.delete != nil {
		this.delete.Done()
		this.delete = nil
	}
	if this.insert != nil {
		this.insert.Done()
		this.insert = nil
	}
	_MERGE_OPERATOR_POOL.Put(this.children)
	this.children = nil
}

var _MERGE_OPERATOR_POOL = NewOperatorPool(3)
var _MERGE_CHANNEL_POOL = NewChannelPool(3)
