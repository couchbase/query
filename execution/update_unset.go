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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// Write to copy
type Unset struct {
	base
	plan *plan.Unset
}

func NewUnset(plan *plan.Unset, context *Context) *Unset {
	rv := &Unset{
		base: newBase(context),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) Copy() Operator {
	return &Unset{this.base.copy(), this.plan}
}

func (this *Unset) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Unset) processItem(item value.AnnotatedValue, context *Context) bool {
	clone, ok := item.GetAttachment("clone").(value.AnnotatedValue)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid UPDATE clone of type %T.", clone)))
		return false
	}

	var err error
	for _, t := range this.plan.Node().Terms() {
		clone, err = unsetPath(t, clone, item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "UNSET clause"))
			return false
		}
	}

	item.SetAttachment("clone", clone)
	return this.sendItem(item)
}

func unsetPath(t *algebra.UnsetTerm, clone, item value.AnnotatedValue, context *Context) (
	value.AnnotatedValue, error) {
	if t.UpdateFor() != nil {
		return unsetFor(t, clone, item, context)
	}

	t.Path().Unset(clone, context)
	return clone, nil
}

func unsetFor(t *algebra.UnsetTerm, clone, item value.AnnotatedValue, context *Context) (
	value.AnnotatedValue, error) {
	var ivals []value.Value

	when := t.UpdateFor().When()
	if when != nil {
		vals, mismatch, err := buildFor(t.UpdateFor(), item, context)
		ivals = vals
		defer releaseValsFor(ivals)
		if err != nil {
			return nil, err
		}

		if mismatch {
			return clone, nil
		}
	}

	cvals, mismatch, err := buildFor(t.UpdateFor(), clone, context)
	defer releaseValsFor(cvals)
	if err != nil {
		return nil, err
	}

	if mismatch {
		return clone, nil
	}

	// Clone may have been mutated by another term
	if ivals != nil && len(ivals) != len(cvals) {
		return clone, nil
	}

	for i := 0; i < len(cvals); i++ {
		if when != nil {
			w, err := when.Evaluate(ivals[i], context)
			if err != nil {
				return nil, err
			}

			if !w.Truth() {
				continue
			}
		}

		t.Path().Unset(cvals[i], context)
	}

	return clone, nil
}

func (this *Unset) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
