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

	"github.com/couchbaselabs/query/value"
)

// Distincting of input data.
type InitialDistinct struct {
	base
}

// Distincting of distincts. Recursable.
type SubsequentDistinct struct {
	base
}

func NewInitialDistinct() *InitialDistinct {
	rv := &InitialDistinct{
		base: newBase(),
	}

	rv.output = rv
	return rv
}

func (this *InitialDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialDistinct(this)
}

func (this *InitialDistinct) Copy() Operator {
	return &InitialDistinct{this.base.copy()}
}

func (this *InitialDistinct) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *InitialDistinct) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *InitialDistinct) afterItems(context *Context, parent value.Value) {
}

func NewSubsequentDistinct() *SubsequentDistinct {
	rv := &SubsequentDistinct{
		base: newBase(),
	}

	rv.output = rv
	return rv
}

func (this *SubsequentDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSubsequentDistinct(this)
}

func (this *SubsequentDistinct) Copy() Operator {
	return &SubsequentDistinct{this.base.copy()}
}

func (this *SubsequentDistinct) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SubsequentDistinct) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *SubsequentDistinct) afterItems(context *Context, parent value.Value) {
}
