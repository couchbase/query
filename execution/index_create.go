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
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type CreateIndex struct {
	base
	plan *plan.CreateIndex
}

func NewCreateIndex(plan *plan.CreateIndex) *CreateIndex {
	rv := &CreateIndex{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) Copy() Operator {
	return &CreateIndex{this.base.copy(), this.plan}
}

func (this *CreateIndex) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		if context.Readonly() {
			return
		}

		// Actually create index
		node := this.plan.Node()
		var equalKey expression.Expressions
		if node.Partition() != nil {
			equalKey = expression.Expressions{node.Partition()}
		}

		indexer, err := this.plan.Keyspace().Indexer(node.Using())
		if err != nil {
			context.Error(err)
			return
		}

		_, err = indexer.CreateIndex(node.Name(), equalKey,
			node.Expressions(), node.Where())
		if err != nil {
			context.Error(err)
		}
	})
}
