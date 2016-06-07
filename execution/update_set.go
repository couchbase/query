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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// Write to copy
type Set struct {
	base
	plan *plan.Set
}

func NewSet(plan *plan.Set) *Set {
	rv := &Set{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func (this *Set) Copy() Operator {
	return &Set{this.base.copy(), this.plan}
}

func (this *Set) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Set) processItem(item value.AnnotatedValue, context *Context) bool {
	atmt := item.GetAttachment("clone")
	if atmt == nil {
		context.Error(errors.NewUpdateMissingClone())
		return false
	}

	clone, ok := atmt.(value.AnnotatedValue)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid UPDATE clone of type %T.", clone)))
		return false
	}

	var err error
	for _, t := range this.plan.Node().Terms() {
		clone, err = setPath(t, clone, item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "SET clause"))
			return false
		}
	}

	item.SetAttachment("clone", clone)
	return this.sendItem(item)
}

func setPath(t *algebra.SetTerm, clone, item value.AnnotatedValue, context *Context) (
	value.AnnotatedValue, error) {
	if t.UpdateFor() != nil {
		return setFor(t, clone, item, context)
	}

	v, err := t.Value().Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	t.Path().Set(clone, v, context)
	return clone, nil
}

func setFor(t *algebra.SetTerm, clone, item value.AnnotatedValue, context *Context) (
	value.AnnotatedValue, error) {
	ivals, mismatch, err := buildFor(t.UpdateFor(), item, context)
	defer releaseValsFor(ivals)
	if err != nil {
		return nil, err
	}

	if mismatch {
		return clone, nil
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
	if len(ivals) != len(cvals) {
		return clone, nil
	}

	when := t.UpdateFor().When()
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

		v, err := t.Value().Evaluate(ivals[i], context)
		if err != nil {
			return nil, err
		}

		t.Path().Set(cvals[i], v, context)
	}

	return clone, nil
}
