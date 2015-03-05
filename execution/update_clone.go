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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	base
	plan *plan.Clone
}

func NewClone(plan *plan.Clone) *Clone {
	rv := &Clone{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) Copy() Operator {
	return &Clone{
		base: this.base.copy(),
		plan: this.plan,
	}
}

func (this *Clone) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Clone) processItem(item value.AnnotatedValue, context *Context) bool {
	t, ok := item.Field(this.plan.Alias())
	if !ok {
		context.Fatal(errors.NewError(nil, fmt.Sprintf("UPDATE alias %s not found in item.", this.plan.Alias())))
		return false
	}

	target, ok := t.(value.AnnotatedValue)
	if !ok {
		context.Fatal(errors.NewError(nil, fmt.Sprintf("UPDATE alias %s has no metadata in item.", this.plan.Alias())))
		return false
	}

	clone := target.CopyForUpdate()
	item.SetAttachment("clone", clone)
	return this.sendItem(item)
}
