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

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/value"
)

// Distincting of input data.
type Distinct struct {
	base
	set *value.Set
}

const _DISTINCT_CAP = 1024

func NewDistinct() *Distinct {
	rv := &Distinct{
		base: newBase(),
		set:  value.NewSet(_DISTINCT_CAP),
	}

	rv.output = rv
	return rv
}

func (this *Distinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDistinct(this)
}

func (this *Distinct) Copy() Operator {
	return &Distinct{
		base: this.base.copy(),
		set:  value.NewSet(_DISTINCT_CAP),
	}
}

func (this *Distinct) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Distinct) processItem(item value.AnnotatedValue, context *Context) bool {
	project := item.GetAttachment("project")

	switch project := project.(type) {
	case value.AnnotatedValue:
		item = project
	case value.Value:
		item = value.NewAnnotatedValue(project)
	default:
		context.ErrorChannel() <- errors.NewError(nil,
			fmt.Sprintf("Invalid or missing projection %v.", project))
		return false
	}

	item.SetAttachment("project", item)
	e := this.set.Add(item)
	if e != nil {
		context.ErrorChannel() <- errors.NewError(e, "")
		return false
	}

	return true
}

func (this *Distinct) afterItems(context *Context) {
	for _, av := range this.set.Values() {
		if !this.sendItem(av.(value.AnnotatedValue)) {
			return
		}
	}
}
