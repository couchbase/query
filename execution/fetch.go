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

type Fetch struct {
	base
	plan *plan.Fetch
}

func NewFetch(plan *plan.Fetch) *Fetch {
	rv := &Fetch{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Fetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFetch(this)
}

func (this *Fetch) Copy() Operator {
	return &Fetch{this.base.copy(), this.plan}
}

func (this *Fetch) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Fetch) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.enbatch(item, this, context)
}

func (this *Fetch) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *Fetch) flushBatch(context *Context) bool {
	if len(this.batch) == 0 {
		return true
	}

	// Build list of keys
	keys := make([]string, len(this.batch))
	for i, av := range this.batch {
		meta := av.GetAttachment("meta")

		switch meta := meta.(type) {
		case map[string]interface{}:
			key := meta["id"]

			switch key := key.(type) {
			case string:
				keys[i] = key
			default:
				context.ErrorChannel() <- errors.NewError(nil, fmt.Sprintf(
					"Missing or invalid primary key %v of type %T.",
					key, key))
				return false
			}
		default:
			context.ErrorChannel() <- errors.NewError(nil,
				"Missing or invalid meta for primary key.")
			return false
		}
	}

	// Fetch
	pairs, er := this.plan.Bucket().Fetch(keys)
	if er != nil {
		context.ErrorChannel() <- er
		return false
	}

	// Attach meta and send
	for i, pair := range pairs {
		item := pair.Value

		// Apply projection, if any
		project := this.plan.Term().Project()
		if project != nil {
			var e error
			item, e = project.Evaluate(item, context)
			if e != nil {
				context.ErrorChannel() <- errors.NewError(e,
					"Error evaluating fetch path.")
				return false
			}

			if item.Type() == value.MISSING {
				continue
			}
		}

		av := this.batch[i]
		fv := value.NewAnnotatedValue(item)
		fv.SetAttachment("meta", av.GetAttachment("meta"))
		av.SetField(this.plan.Alias(), fv)

		if !this.sendItem(av) {
			return false
		}
	}

	return true
}
