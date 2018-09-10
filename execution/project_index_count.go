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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexCountProject struct {
	base
	plan *plan.IndexCountProject
}

func NewIndexCountProject(plan *plan.IndexCountProject, context *Context) *IndexCountProject {
	rv := &IndexCountProject{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexCountProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountProject(this)
}

func (this *IndexCountProject) Copy() Operator {
	rv := &IndexCountProject{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexCountProject) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IndexCountProject) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.plan.Projection().Raw() {
		return this.sendItem(item)
	} else {
		var v value.Value
		var err error

		sv := value.NewScopeValue(make(map[string]interface{}, len(this.plan.Terms())), item)
		for _, term := range this.plan.Terms() {
			switch term.Result().Expression().(type) {
			case *algebra.Count:
				v = item.GetValue()
			default:
				v, err = term.Result().Expression().Evaluate(item, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "projection"))
					return false
				}
			}

			sv.SetField(term.Result().Alias(), v)
			if term.Result().As() != "" {
				sv.SetField(term.Result().As(), v)
			}
		}
		av := value.NewAnnotatedValue(sv)
		return this.sendItem(av)
	}
}

func (this *IndexCountProject) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
