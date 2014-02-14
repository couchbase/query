//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	"fmt"

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Merge struct {
	base
	plan          *plan.Merge
	update        Operator
	delete        Operator
	insert        Operator
	updateChannel value.AnnotatedChannel
	deleteChannel value.AnnotatedChannel
	insertChannel value.AnnotatedChannel
	childChannel  StopChannel
	childCount    int
}

func NewMerge(plan *plan.Merge, update, delete, insert Operator) *Merge {
	rv := &Merge{
		base:          newBase(),
		plan:          plan,
		update:        update,
		delete:        delete,
		insert:        insert,
		updateChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		deleteChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		insertChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		childChannel:  make(StopChannel, 3),
	}

	rv.output = rv
	return rv
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) Copy() Operator {
	return &Merge{
		base:          this.base.copy(),
		plan:          this.plan,
		update:        copyOperator(this.update),
		delete:        copyOperator(this.delete),
		insert:        copyOperator(this.insert),
		updateChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		deleteChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		insertChannel: make(value.AnnotatedChannel, _ITEM_CHAN_CAP),
		childChannel:  make(StopChannel, 3),
	}
}

func (this *Merge) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Merge) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *Merge) beforeItems(context *Context, parent value.Value) bool {
	if this.update != nil {
		this.update.SetParent(this)
		this.update.SetOutput(this.output)
		this.childCount++
	}

	if this.delete != nil {
		this.delete.SetParent(this)
		this.delete.SetOutput(this.output)
		this.childCount++
	}

	if this.insert != nil {
		this.insert.SetParent(this)
		this.insert.SetOutput(this.output)
		this.childCount++
	}

	return true
}

func (this *Merge) processItem(item value.AnnotatedValue, context *Context) bool {
	kv, e := this.plan.Key().Evaluate(item, context)
	if e != nil {
		context.ErrorChannel() <- err.NewError(e, "Error evaluatating MERGE key.")
		return false
	}

	ka := kv.Actual()
	k, ok := ka.(string)
	if !ok {
		context.ErrorChannel() <- err.NewError(nil,
			fmt.Sprintf("Invalid MERGE key %v of type %T.", ka, ka))
		return false
	}

	tv, er := this.plan.Bucket().FetchOne(k)
	if er != nil {
		context.ErrorChannel() <- er
		return false
	}

	if tv != nil {
		av := value.NewAnnotatedValue(tv)

		// Matched; UPDATE and/or DELETE
		if this.updateChannel != nil {
			this.updateChannel <- av
		}
		if this.deleteChannel != nil {
			this.deleteChannel <- av
		}
	} else {
		// Not matched; INSERT
		if this.insertChannel != nil {
			this.insertChannel <- item
		}
	}

	return true
}

func (this *Merge) afterItems(context *Context) {
}

func copyOperator(op Operator) Operator {
	if op == nil {
		return nil
	} else {
		return op.Copy()
	}
}
