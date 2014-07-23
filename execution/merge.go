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

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Merge struct {
	base
	plan         *plan.Merge
	update       Operator
	delete       Operator
	insert       Operator
	childChannel StopChannel
	childCount   int
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
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		update := this.wrapChild(this.update)
		delete := this.wrapChild(this.delete)
		insert := this.wrapChild(this.insert)

		go this.input.RunOnce(context, parent)

		var item value.AnnotatedValue
		n := this.childCount
		ok := true

		for {
			select {
			case item, ok = <-this.input.ItemChannel():
				if ok {
					ok = this.processMatch(item, context, update, delete, insert)
				}

				if !ok {
					notifyChildren(update, delete, insert)
				}
			case <-this.childChannel: // Never closed
				// Wait for all children
				if n--; n <= 0 {
					return
				}
			case <-this.stopChannel: // Never closed
				notifyChildren(update, delete, insert)
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
		context.Error(errors.NewError(e, "Error evaluatating MERGE key."))
		return false
	}

	ka := kv.Actual()
	k, ok := ka.(string)
	if !ok {
		context.Error(errors.NewError(nil,
			fmt.Sprintf("Invalid MERGE key %v of type %T.", ka, ka)))
		return false
	}

	bv, err := this.plan.Keyspace().FetchOne(k)
	if err != nil {
		context.Error(err)
		return false
	}

	if bv != nil {
		// Matched; join source and target
		if update != nil {
			item.SetAttachment("target", item.Copy())
		}

		abv := value.NewAnnotatedValue(bv)
		item.SetField(this.plan.Alias(), abv)

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

	return true
}

func copyOperator(op Operator) Operator {
	if op == nil {
		return nil
	} else {
		return op.Copy()
	}
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
	this.childCount++
	return seq
}

func notifyChildren(children ...Operator) {
	for _, child := range children {
		if child == nil {
			continue
		}

		select {
		case child.Input().StopChannel() <- false:
		default:
		}

		select {
		case child.StopChannel() <- false:
		default:
		}
	}
}
