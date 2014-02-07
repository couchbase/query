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
	_ "fmt"

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
	// Single unprefixed star, or project raw value
	if len(this.plan.Terms()) == 1 && this.plan.Terms()[0].Result().Expression() == nil {
		if item.Type() == value.OBJECT || !this.plan.Terms()[0].Result().Star() {
			item.SetAttachment("project", item.GetValue())
		} else {
			item.SetAttachment("project", value.NewValue(map[string]interface{}{}))
		}

		return this.sendItem(item)
	}

	project := value.NewValue(make(map[string]interface{}))
	as := make(map[string]value.Value)

	for _, term := range this.plan.Terms() {
		if term.Alias() != "" {
			val, e := term.Result().Expression().Evaluate(item, context)
			if e != nil {
				context.ErrorChannel() <- err.NewError(e, "Error evaluating projection.")
				return false
			}

			if term.Result().As() != "" {
				as[term.Alias()] = val
			} else {
				project.SetField(term.Alias(), val)
			}
		} else {
			// Star
			var starActual interface{}
			if term.Result().Expression() == nil {
				starActual = item.Actual()
			} else {
				val, e := term.Result().Expression().Evaluate(item, context)
				if e != nil {
					context.ErrorChannel() <- err.NewError(e, "Error evaluating projection.")
					return false
				}
				starActual = val.Actual()
			}

			// Latest star overwrites previous star
			switch starActual := starActual.(type) {
			case map[string]interface{}:
				for k, v := range starActual {
					project.SetField(k, value.NewValue(v))
				}
			}
		}
	}

	// Explicit aliases overwrite everything, including data
	for a, v := range as {
		project.SetField(a, v)
		item.SetField(a, v)
	}

	item.SetAttachment("project", project)
	return this.sendItem(item)
}
