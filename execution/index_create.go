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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type CreateIndex struct {
	base
	plan *plan.CreateIndex
}

func NewCreateIndex(plan *plan.CreateIndex, context *Context) *CreateIndex {
	rv := &CreateIndex{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) Copy() Operator {
	rv := &CreateIndex{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreateIndex) RunOnce(context *Context, parent value.Value) {
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

		// Actually create index
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

			rangeKeys := this.getRangeKeys(node.Keys())
			_, err = indexer3.CreateIndex3(context.RequestId(), node.Name(), rangeKeys,
				indexPartition, node.Where(), node.With())
			if err != nil {
				context.Error(err)
				return
			}
		} else {
			if node.Keys().Missing() {
				context.Error(errors.NewIndexLeadingKeyMissingNotSupportedError())
				return
			}

			if node.Partition() != nil {
				context.Error(errors.NewPartitionIndexNotSupportedError())
				return
			}

			if indexer2, ok := indexer.(datastore.Indexer2); ok {
				rangeKeys := this.getRangeKeys(node.Keys())
				_, err = indexer2.CreateIndex2(context.RequestId(), node.Name(), node.SeekKeys(),
					rangeKeys, node.Where(), node.With())
				if err != nil {
					context.Error(err)
					return
				}
			} else {
				if node.Keys().HasDescending() {
					context.Error(errors.NewIndexerDescCollationError())
					return
				}

				_, err = indexer.CreateIndex(context.RequestId(), node.Name(), node.SeekKeys(),
					node.RangeKeys(), node.Where(), node.With())
				if err != nil {
					context.Error(err)
					return
				}
			}
		}
	})
}

func (this *CreateIndex) getRangeKeys(terms algebra.IndexKeyTerms) datastore.IndexKeys {
	rangeKeys := make(datastore.IndexKeys, 0, len(terms))
	for i, term := range terms {
		attrs := datastore.IK_NONE
		// non-leading IK_MISSING is always true
		if i > 0 || term.HasAttribute(algebra.IK_MISSING) {
			attrs = datastore.IK_MISSING
		}
		if term.HasAttribute(algebra.IK_DESC) {
			attrs |= datastore.IK_DESC
		}

		rk := &datastore.IndexKey{Expr: term.Expression(), Attributes: attrs}
		rangeKeys = append(rangeKeys, rk)
	}

	return rangeKeys
}

func (this *CreateIndex) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
