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
type Unset struct {
	base
	plan *plan.Unset
}

func NewUnset(plan *plan.Unset) *Unset {
	rv := &Unset{
		base: newBase(),
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

	for _, t := range this.plan.Node().Terms() {
		unsetPath(t, clone, context)
	}

	return this.sendItem(item)
}

func unsetPath(t *algebra.UnsetTerm, clone value.AnnotatedValue, context *Context) error {
	if t.UpdateFor() != nil {
		return unsetFor(t, clone, context)
	}

	t.Path().Unset(clone, context)
	return nil
}

func unsetFor(t *algebra.UnsetTerm, clone value.AnnotatedValue, context *Context) error {
	arrays, e := arraysFor(t.UpdateFor(), clone, context)
	if e != nil {
		return e
	}

	cvals, e := buildFor(t.UpdateFor(), clone, arrays, context)
	if e != nil {
		return e
	}

	when := t.UpdateFor().When()
	for i := 0; i < len(cvals); i++ {
		if when != nil {
			w, e := when.Evaluate(cvals[i], context)
			if e != nil {
				return e
			}

			if !w.Truth() {
				continue
			}
		}

		t.Path().Unset(cvals[i], context)
	}

	return nil
}
