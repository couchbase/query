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

func (this *InitialProject) processItem(item value.AnnotatedValue, context *Context) bool {
	terms := this.plan.Terms()
	n := len(terms)

	if n > 1 {
		return this.processTerms(item, context)
	}

	if n == 0 {
		// No terms; send raw value
		item.SetAttachment("project", item.GetValue())
		return this.sendItem(item)
	}

	// n == 1
	result := terms[0].Result()
	expr := result.Expression()

	// Special cases
	if expr == nil {
		// Unprefixed star or raw projection of item
		if item.Type() == value.OBJECT || this.plan.Projection().Raw() {
			item.SetAttachment("project", item.GetValue())
		} else {
			item.SetAttachment("project",
				value.NewValue(map[string]interface{}{}))
		}
		return this.sendItem(item)
	} else if this.plan.Projection().Raw() {
		// Raw projection of an expression
		val, err := expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewError(err, "Error evaluating projection."))
			return false
		}
		item.SetAttachment("project", val)
		return this.sendItem(item)
	} else {
		// Default
		return this.processTerms(item, context)
	}
}

func (this *InitialProject) processTerms(item value.AnnotatedValue, context *Context) bool {
	n := len(this.plan.Terms())
	sv := value.NewScopeValue(make(map[string]interface{}, n), item)

	pv := value.NewAnnotatedValue(sv)
	pv.SetAttachments(item.Attachments())

	project := value.NewValue(make(map[string]interface{}))
	pv.SetAttachment("project", project)

	for _, term := range this.plan.Terms() {
		if term.Result().Alias() != "" {
			val, err := term.Result().Expression().Evaluate(item, context)
			if err != nil {
				context.Error(errors.NewError(err, "Error evaluating projection."))
				return false
			}

			project.SetField(term.Result().Alias(), val)

			// Explicit aliases overshadow data
			if term.Result().As() != "" {
				pv.SetField(term.Result().Alias(), val)
			}
		} else {
			// Star
			starValue := item.(value.Value)
			if term.Result().Expression() != nil {
				var err error
				starValue, err = term.Result().Expression().Evaluate(item, context)
				if err != nil {
					context.Error(errors.NewError(err, "Error evaluating projection."))
					return false
				}
			}

			// Latest star overwrites previous star
			switch starActual := starValue.Actual().(type) {
			case map[string]interface{}:
				for k, v := range starActual {
					project.SetField(k, value.NewValue(v))
				}
			}
		}
	}

	return this.sendItem(pv)
}
