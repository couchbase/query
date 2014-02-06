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

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/value"
)

// Collect subquery results
type Collect struct {
	base
	values []value.Value
	length int
}

const _COLLECT_BUF_CAP = 64

func NewCollect() *Collect {
	rv := &Collect{
		base:   newBase(),
		values: make([]value.Value, _COLLECT_BUF_CAP),
	}

	rv.output = rv
	return rv
}

func (this *Collect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCollect(this)
}

func (this *Collect) Copy() Operator {
	return &Collect{
		base:   this.base.copy(),
		values: make([]value.Value, _COLLECT_BUF_CAP),
	}
}

func (this *Collect) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Collect) processItem(item value.AnnotatedValue, context *Context) bool {
	mv := item.GetAttachment("meta")
	if mv == nil {
		context.ErrorChannel() <- err.NewError(nil, "Unable to find meta.")
		return false
	}

	meta := mv.(map[string]interface{})
	project, ok := meta["project"]
	if !ok {
		context.ErrorChannel() <- err.NewError(nil, "Unable to find projection.")
		return false
	}

	switch project := project.(type) {
	case value.Value:
		if project.Type() != value.MISSING {
			// Ensure room
			if len(this.values) == this.length {
				values := make([]value.Value, this.length<<1)
				copy(values, this.values)
				this.values = values
			}

			this.values[this.length] = project
			this.length++
		}

		return true
	default:
		context.ErrorChannel() <- err.NewError(nil,
			fmt.Sprintf("Unable to project value %v of type %T.", project, project))
		return false
	}
}

func (this *Collect) afterItems(context *Context) {
	this.values = this.values[0:this.length]
}

func (this *Collect) Values() []value.Value {
	return this.values
}
