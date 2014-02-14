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
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Alias struct {
	base
	plan *plan.Alias
}

func NewAlias(plan *plan.Alias) *Alias {
	rv := &Alias{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Alias) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlias(this)
}

func (this *Alias) Copy() Operator {
	return &Alias{this.base.copy(), this.plan}
}

func (this *Alias) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Alias) processItem(item value.AnnotatedValue, context *Context) bool {
	av := value.NewAnnotatedValue(make(map[string]interface{}))
	av.SetAttachments(item.Attachments())
	av.SetField(this.plan.Alias(), item)
	return this.sendItem(av)
}
