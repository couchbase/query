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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

// Write to copy
type Unset struct {
	base
	plan *plan.Unset
}

func NewUnset(plan *plan.Unset) *Unset {
	rv := &Unset{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) Copy() Operator {
	return &Unset{this.base.copy(), this.plan}
}

func (this *Unset) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Unset) processItem(item value.AnnotatedValue, context *Context) bool {
	clone, ok := item.GetAttachment("clone").(value.AnnotatedValue)
	if !ok {
		context.ErrorChannel() <- err.NewError(nil,
			fmt.Sprintf("Invalid UPDATE clone of type %T.", clone))
		return false
	}

	for _, up := range this.plan.Node().Paths() {
		unsetPath(up, clone, context)
	}

	return this.sendItem(item)
}

func unsetPath(up *algebra.UnsetPath, clone value.AnnotatedValue, context *Context) error {
	if up.PathFor() != nil {
		return unsetPathFor(up, clone, context)
	}

	up.Path().Unset(clone)
	return nil
}

func unsetPathFor(up *algebra.UnsetPath, clone value.AnnotatedValue, context *Context) error {
	cvals, e := buildFor(up.PathFor(), clone, context)
	if e != nil {
		return e
	}

	for i := 0; i < len(cvals); i++ {
		up.Path().Unset(cvals[i])
	}

	return nil
}
