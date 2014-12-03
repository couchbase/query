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

type Authenticate struct {
	base
	plan *plan.Authenticate
}

func NewAuthenticate(plan *plan.Authenticate) *Authenticate {
	rv := &Authenticate{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Authenticate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAuthenticate(this)
}

func (this *Authenticate) Copy() Operator {
	return &Authenticate{this.base.copy(), this.plan}
}

func (this *Authenticate) RunOnce(context *Context, parent value.Value) {

	this.runConsumer(this, context, parent)
}

func (this *Authenticate) beforeItems(context *Context, parent value.Value) bool {

	credentials := this.plan.Credentials()
	privilege := this.plan.Privilege()
	err := this.plan.Keyspace().Authenticate(credentials, privilege)
	if err != nil {
		context.Error(errors.NewError(err, "Authentication Failed"))
		return false
	}

	return true

}

func (this *Authenticate) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.sendItem(item)
}
