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

	for _, sp := range this.plan.Node().Paths() {
		setPath(sp, clone, item, context)
	}

	return this.sendItem(item)
}

func setPath(sp *algebra.SetPath, clone, item value.AnnotatedValue, context *Context) error {
	if sp.PathFor() != nil {
		return setPathFor(sp, clone, item, context)
	}

	v, e := sp.Value().Evaluate(item, context)
	if e != nil {
		return e
	}

	sp.Path().Set(clone, v)
	return nil
}

func setPathFor(sp *algebra.SetPath, clone, item value.AnnotatedValue, context *Context) error {
	cvals, e := buildFor(sp.PathFor(), clone, context)
	if e != nil {
		return e
	}

	ivals, e := buildFor(sp.PathFor(), item, context)
	if e != nil {
		return e
	}

	n := len(cvals)
	if len(ivals) < n {
		n = len(ivals)
	}

	for i := 0; i < n; i++ {
		v, e := sp.Value().Evaluate(ivals[i], context)
		if e != nil {
			return e
		}
		sp.Path().Set(cvals[i], v)
	}

	return nil
}

func buildFor(pf *algebra.PathFor, val value.Value, context *Context) ([]value.Value, error) {
	var e error
	arrays := make([]value.Value, len(pf.Bindings()))
	for i, b := range pf.Bindings() {
		arrays[i], e = b.Expression().Evaluate(val, context)
		if e != nil {
			return nil, e
		}
	}

	n := 0
	for _, a := range arrays {
		act := a.Actual()
		switch act := act.(type) {
		case []interface{}:
			if len(act) > n {
				n = len(act)
			}
		}
	}

	rv := make([]value.Value, n)
	for i, _ := range rv {
		rv[i] = value.NewCorrelatedValue(val)
		for j, b := range pf.Bindings() {
			v := arrays[j].Index(i)
			if v.Type() != value.MISSING {
				rv[i].SetField(b.Variable(), v)
			}
		}
	}

	return rv, nil
}
