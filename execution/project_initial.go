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
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type InitialProject struct {
	base
	plan *plan.InitialProject
}

func NewInitialProject(plan *plan.InitialProject) *InitialProject {
	rv := &InitialProject{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *InitialProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialProject(this)
}

func (this *InitialProject) Copy() Operator {
	return &InitialProject{this.base.copy(), this.plan}
}

func (this *InitialProject) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

var _EMPTY_ANNOTATED_VALUE = value.NewAnnotatedValue(map[string]interface{}{})

func (this *InitialProject) processItem(item value.AnnotatedValue, context *Context) bool {
	terms := this.plan.Terms()
	n := len(terms)

	if n > 1 {
		return this.processTerms(item, context)
	}

	if n == 0 {
		return this.sendItem(item)
	}

	// n == 1

	result := terms[0].Result()
	expr := result.Expression()

	if expr == nil {
		// Unprefixed star
		if item.Type() == value.OBJECT {
			return this.sendItem(item)
		} else {
			return this.sendItem(_EMPTY_ANNOTATED_VALUE)
		}
	} else if this.plan.Projection().Raw() {
		// Raw projection of an expression
		v, err := expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewError(err, "Error evaluating projection."))
			return false
		}

		if v.Type() == value.MISSING {
			return true
		}

		if result.As() == "" {
			return this.sendItem(value.NewAnnotatedValue(v))
		}

		sv := value.NewScopeValue(make(map[string]interface{}, 1), item)
		sv.SetField(result.As(), v)
		av := value.NewAnnotatedValue(sv)
		av.SetAttachment("projection", v)
		return this.sendItem(av)
	} else {
		// Any other projection
		return this.processTerms(item, context)
	}
}

func (this *InitialProject) processTerms(item value.AnnotatedValue, context *Context) bool {
	pv := value.NewAnnotatedValue(item.Copy().Fields())
	pv.SetAttachments(item.Attachments())

	n := len(this.plan.Terms())
	p := value.NewValue(make(map[string]interface{}, n+32))
	pv.SetAttachment("projection", p)

	for _, term := range this.plan.Terms() {
		if term.Result().Alias() != "" {
			v, err := term.Result().Expression().Evaluate(item, context)
			if err != nil {
				context.Error(errors.NewError(err, "Error evaluating projection."))
				return false
			}

			p.SetField(term.Result().Alias(), v)

			// Explicit aliases override data
			if term.Result().As() != "" {
				pv.SetField(term.Result().As(), v)
			}
		} else {
			// Star
			starval := item.GetValue()
			if term.Result().Expression() != nil {
				var err error
				starval, err = term.Result().Expression().Evaluate(item, context)
				if err != nil {
					context.Error(errors.NewError(err, "Error evaluating projection."))
					return false
				}
			}

			// Latest star overwrites previous star
			switch sa := starval.Actual().(type) {
			case map[string]interface{}:
				for k, v := range sa {
					p.SetField(k, v)
				}
			}
		}
	}

	return this.sendItem(pv)
}
