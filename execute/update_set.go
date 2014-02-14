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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
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
	clone, ok := item.GetAttachment("clone").(value.AnnotatedValue)
	if !ok {
		context.ErrorChannel() <- err.NewError(nil,
			fmt.Sprintf("Invalid UPDATE clone of type %T.", clone))
		return false
	}

	var e error
	for _, t := range this.plan.Node().Terms() {
		clone, e = setPath(t, clone, item, context)
		if e != nil {
			context.ErrorChannel() <- err.NewError(e, "Error evaluating SET clause.")
			return false
		}
	}

	item.SetAttachment("clone", clone)
	return this.sendItem(item)
}

func setPath(t *algebra.SetTerm, clone, item value.AnnotatedValue, context *Context) (value.AnnotatedValue, error) {
	if t.UpdateFor() != nil {
		return setFor(t, clone, item, context)
	}

	v, e := t.Value().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if t.Path() != nil {
		t.Path().Set(clone, v)
		return clone, nil
	} else {
		av := value.NewAnnotatedValue(v)
		av.SetAttachments(clone.Attachments())
		return av, nil
	}
}

func setFor(t *algebra.SetTerm, clone, item value.AnnotatedValue, context *Context) (value.AnnotatedValue, error) {
	carrays, e := arraysFor(t.UpdateFor(), clone, context)
	if e != nil {
		return nil, e
	}

	cvals, e := buildFor(t.UpdateFor(), clone, carrays, context)
	if e != nil {
		return nil, e
	}

	iarrays, e := arraysFor(t.UpdateFor(), item, context)
	if e != nil {
		return nil, e
	}

	ivals, e := buildFor(t.UpdateFor(), item, iarrays, context)
	if e != nil {
		return nil, e
	}

	// Clone may have been shortened by previous SET term
	n := len(ivals)
	if len(cvals) < n {
		n = len(cvals)
	}

	when := t.UpdateFor().When()
	for i := 0; i < n; i++ {
		v, e := t.Value().Evaluate(ivals[i], context)
		if e != nil {
			return nil, e
		}

		if when != nil {
			w, e := when.Evaluate(ivals[i], context)
			if e != nil {
				return nil, e
			}

			if !w.Truth() {
				continue
			}
		}

		t.Path().Set(cvals[i], v)
	}

	// Set array elements
	f := t.UpdateFor()
	for a, b := range f.Bindings() {
		switch ca := carrays[a].Actual().(type) {
		case []interface{}:
			for i := 0; i < n; i++ {
				ca[i], _ = cvals[i].Field(b.Variable())
			}
		}
	}

	return clone, nil
}
