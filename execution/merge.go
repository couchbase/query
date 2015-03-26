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
	"fmt"

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

		if context.Readonly() {
			return
		}

		go this.input.RunOnce(context, parent)

		update := this.wrapChild(this.update)
		delete := this.wrapChild(this.delete)
		insert := this.wrapChild(this.insert)

		children := make([]Operator, 0, 3)

		if update != nil {
			children = append(children, update)
		}

		if delete != nil {
			children = append(children, delete)
		}

		if insert != nil {
			children = append(children, insert)
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
				this.notifyStop()
				notifyChildren(children...)
				break loop
			default:
			}

			select {
			case item, ok = <-this.input.ItemChannel():
				if ok {
					ok = this.processMatch(item, context, update, delete, insert)
				}
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				notifyChildren(children...)
				break loop
			}
		}

		for _, child := range children {
			// Signal end of input data
			select {
			case child.Input().StopChannel() <- false:
			default:
			}
		}

		n := len(children)
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

func (this *Merge) ChildChannel() StopChannel {
	return this.childChannel
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

	fetchOk := true
	bvs, errs := this.plan.Keyspace().Fetch([]string{k})
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	if len(bvs) > 0 {
		bv := bvs[0]
		item.SetField(this.plan.KeyspaceRef().Alias(), bv.Value)

		// Perform UPDATE and/or DELETE
		if update != nil {
			update.Input().ItemChannel() <- item
		}

		if delete != nil {
			delete.Input().ItemChannel() <- item
		}
	} else {
		// Not matched; INSERT
		if insert != nil {
			insert.Input().ItemChannel() <- item
		}
	}

	return fetchOk
}

func (this *Merge) wrapChild(op Operator) Operator {
	if op == nil {
		return nil
	}

	ch := NewChannel()
	seq := NewSequence(ch, op)
	seq.SetInput(ch)
	seq.SetParent(this)
	seq.SetOutput(this.output)
	return seq
}
