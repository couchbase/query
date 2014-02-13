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
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Project struct {
	base
	plan *plan.Project
}

func NewProject(plan *plan.Project) *Project {
	rv := &Project{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Project) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitProject(this)
}

func (this *Project) Copy() Operator {
	return &Project{this.base.copy(), this.plan}
}

func (this *Project) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Project) processItem(item value.AnnotatedValue, context *Context) bool {
	terms := this.plan.Terms()
	n := len(terms)

	if n > 1 {
		return this.processTerms(item, context)
	}

	// Raw value
	if n == 0 {
		item.SetAttachment("project", item.GetValue())
		return this.sendItem(item)
	}

	// n == 1
	result := terms[0].Result()
	expr := result.Expression()
	alias := terms[0].Alias()

	// Special cases
	if expr == nil {
		// Unprefixed star or raw value
		if item.Type() == value.OBJECT || !result.Star() {
			item.SetAttachment("project", item.GetValue())
		} else {
			item.SetAttachment("project",
				value.NewValue(map[string]interface{}{}))
		}
		return this.sendItem(item)
	} else if alias == "" && !result.Star() {
		// Raw projection of single expression
		val, e := expr.Evaluate(item, context)
		if e != nil {
			context.ErrorChannel() <- err.NewError(e, "Error evaluating projection.")
			return false
		}
		item.SetAttachment("project", val)
		return this.sendItem(item)
	}

	// Default
	return this.processTerms(item, context)
}

func (this *Project) processTerms(item value.AnnotatedValue, context *Context) bool {
	n := len(this.plan.Terms())
	cv := value.NewCorrelatedValue(make(map[string]interface{}, n), item)

	pv := value.NewAnnotatedValue(cv)
	pv.SetAttachments(item.Attachments())

	project := value.NewValue(make(map[string]interface{}))
	pv.SetAttachment("project", project)

	for _, term := range this.plan.Terms() {
		if term.Alias() != "" {
			val, e := term.Result().Expression().Evaluate(item, context)
			if e != nil {
				context.ErrorChannel() <- err.NewError(e, "Error evaluating projection.")
				return false
			}

			project.SetField(term.Alias(), val)

			// Explicit aliases overshadow data
			if term.Result().As() != "" {
				pv.SetField(term.Alias(), val)
			}
		} else {
			// Star
			starValue := item.(value.Value)
			if term.Result().Expression() != nil {
				var e error
				starValue, e = term.Result().Expression().Evaluate(item, context)
				if e != nil {
					context.ErrorChannel() <- err.NewError(e, "Error evaluating projection.")
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
