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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexCountProject struct {
	base
	plan *plan.IndexCountProject
}

func NewIndexCountProject(plan *plan.IndexCountProject) *IndexCountProject {
	rv := &IndexCountProject{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexCountProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountProject(this)
}

func (this *IndexCountProject) Copy() Operator {
	return &IndexCountProject{this.base.copy(), this.plan}
}

func (this *IndexCountProject) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IndexCountProject) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.plan.Projection().Raw() {
		return this.sendItem(item)
	} else {
		terms := this.plan.Terms()
		result := terms[0].Result()
		v := item.GetValue()
		sv := value.NewScopeValue(make(map[string]interface{}, 1), item)
		sv.SetField(result.Alias(), v)
		if result.As() != "" {
			sv.SetField(result.As(), v)
		}
		av := value.NewAnnotatedValue(sv)
		av.CopyCovers(item)
		return this.sendItem(av)
	}
}
