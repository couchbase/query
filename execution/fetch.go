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
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
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
	context.SetSortCount(0)
}

func (this *Fetch) flushBatch(context *Context) bool {
	defer this.releaseBatch()

	if len(this.batch) == 0 {
		return true
	}

	// Build list of keys
	keys := allocateStringBatch()
	defer releaseStringBatch(keys)

	for _, av := range this.batch {
		meta := av.GetAttachment("meta")

		switch meta := meta.(type) {
		case map[string]interface{}:
			key := meta["id"]
			act := value.NewValue(key).Actual()
			switch act := act.(type) {
			case string:
				keys = append(keys, act)
			default:
				context.Error(errors.NewInvalidValueError(fmt.Sprintf(
					"Missing or invalid primary key %v of type %T.",
					act, act)))
				return false
			}
		default:
			context.Error(errors.NewInvalidValueError(
				"Missing or invalid meta for primary key."))
			return false
		}
	}

	timer := time.Now()

	// Fetch
	pairs, errs := this.plan.Keyspace().Fetch(keys)

	context.AddPhaseTime("fetch", time.Since(timer))

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	// Attach meta and send
	for _, pair := range pairs {
		pv, ok := pair.Value.(value.AnnotatedValue)
		if !ok {
			context.Fatal(errors.NewInvalidValueError(fmt.Sprintf(
				"Invalid fetch value %v of type %T", pair.Value)))
			return false
		}

		var fv value.AnnotatedValue

		// Apply projection, if any
		projection := this.plan.Term().Projection()
		if projection != nil {
			projectedItem, e := projection.Evaluate(pv, context)
			if e != nil {
				context.Error(errors.NewEvaluationError(e, "fetch path"))
				return false
			}

			if projectedItem.Type() == value.MISSING {
				continue
			}

			fv = value.NewAnnotatedValue(projectedItem)
			fv.SetAttachments(pv.Attachments())
		} else {
			fv = pv
		}

		item := value.NewAnnotatedValue(make(map[string]interface{}))
		item.SetField(this.plan.Term().Alias(), fv)

		if !this.sendItem(item) {
			return false
		}
	}

	return fetchOk
}
