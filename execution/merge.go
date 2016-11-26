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
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Merge struct {
	base
	plan         *plan.Merge
	update       Operator
	delete       Operator
	insert       Operator
	childChannel StopChannel
}

func NewMerge(plan *plan.Merge, update, delete, insert Operator) *Merge {
	rv := &Merge{
		base:         newBase(),
		plan:         plan,
		update:       update,
		delete:       delete,
		insert:       insert,
		childChannel: make(StopChannel, 3),
	}

	rv.output = rv
	return rv
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) Copy() Operator {
	return &Merge{
		base:         this.base.copy(),
		plan:         this.plan,
		update:       copyOperator(this.update),
		delete:       copyOperator(this.delete),
		insert:       copyOperator(this.insert),
		childChannel: make(StopChannel, 3),
	}
}

func (this *Merge) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		addTime := func() {
			context.AddPhaseTime(MERGE, this.duration)
			this.addTime(this.duration)
		}
		defer addTime()

		if context.Readonly() {
			return
		}

		go this.input.RunOnce(context, parent)

		update, updateInput := this.wrapChild(this.update)
		delete, deleteInput := this.wrapChild(this.delete)
		insert, insertInput := this.wrapChild(this.insert)

		children := _MERGE_OPERATOR_POOL.Get()
		defer _MERGE_OPERATOR_POOL.Put(children)
		inputs := _MERGE_CHANNEL_POOL.Get()
		defer _MERGE_CHANNEL_POOL.Put(inputs)

		if update != nil {
			children = append(children, update)
			inputs = append(inputs, updateInput)
		}

		if delete != nil {
			children = append(children, delete)
			inputs = append(inputs, deleteInput)
		}

		if insert != nil {
			children = append(children, insert)
			inputs = append(inputs, insertInput)
		}

		for _, child := range children {
			go child.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		ok := true
	loop:
		for ok {
			select {
			case <-this.stopChannel: // Never closed
				break loop
			default:
			}

			t := time.Now()
			select {
			case item, ok = <-this.input.ItemChannel():
				this.chanTime += time.Since(t)
				if ok {
					ok = this.processMatch(item, context, update, delete, insert)
				}
			case <-this.stopChannel: // Never closed
				this.chanTime += time.Since(t)
				break loop
			}
		}

		// Close child input Channels, which will signal children
		for _, input := range inputs {
			input.Close()
		}

		// Wait for all children
		n := len(children)
		for n > 0 {
			select {
			case <-this.childChannel: // Never closed
				n--
			}
		}
	})
}

func (this *Merge) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *Merge) mergeSendItem(op Operator, item value.AnnotatedValue) bool {
	t := time.Now()
	addTime := func() {
		this.chanTime += time.Since(t)
	}
	defer addTime()

	select {
	case <-this.stopChannel: // Never closed
		return false
	default:
	}

	select {
	case op.Input().ItemChannel() <- item:
		return true
	case <-this.stopChannel: // Never closed
		return false
	}
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

	timer := time.Now()

	ok = true
	bvs, errs := this.plan.Keyspace().Fetch([]string{k})

	this.duration += time.Since(timer)

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
			ok = this.mergeSendItem(update, item)
		}

		if ok && delete != nil {
			ok = this.mergeSendItem(delete, item)
		}
	} else {
		// Not matched; INSERT
		if insert != nil {
			ok = this.mergeSendItem(insert, item)
		}
	}

	return ok
}

func (this *Merge) wrapChild(op Operator) (Operator, *Channel) {
	if op == nil {
		return nil, nil
	}

	ch := NewChannel()
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

var _MERGE_OPERATOR_POOL = NewOperatorPool(3)
var _MERGE_CHANNEL_POOL = NewChannelPool(3)
