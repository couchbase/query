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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type CreatePrimaryIndex struct {
	base
	plan *plan.CreatePrimaryIndex
}

func NewCreatePrimaryIndex(plan *plan.CreatePrimaryIndex, context *Context) *CreatePrimaryIndex {
	rv := &CreatePrimaryIndex{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *CreatePrimaryIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreatePrimaryIndex(this)
}

func (this *CreatePrimaryIndex) Copy() Operator {
	rv := &CreatePrimaryIndex{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreatePrimaryIndex) PlanOp() plan.Operator {
	return this.plan
}

func (this *CreatePrimaryIndex) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		// Actually create primary index
		this.switchPhase(_SERVTIME)
		node := this.plan.Node()
		indexer, err := this.plan.Keyspace().Indexer(node.Using())
		if err != nil {
			context.Error(err)
			return
		}

		if indexer3, ok := indexer.(datastore.Indexer3); ok {
			var indexPartition *datastore.IndexPartition

			if node.Partition() != nil {
				indexPartition = &datastore.IndexPartition{Strategy: node.Partition().Strategy(),
					Exprs: node.Partition().Exprs()}
			}

			_, err = indexer3.CreatePrimaryIndex3(context.RequestId(), node.Name(), indexPartition, node.With())
			if err != nil {
				context.Error(err)
				return
			}
		} else {
			if node.Partition() != nil {
				context.Error(errors.NewPartitionIndexNotSupportedError())
				return
			}
			_, err = indexer.CreatePrimaryIndex(context.RequestId(), node.Name(), node.With())
			if err != nil {
				context.Error(err)
				return
			}
		}
	})
}

func (this *CreatePrimaryIndex) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
