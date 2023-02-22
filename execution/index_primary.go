//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

	newRedirectBase(&rv.base, context)
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

		var ok3 bool
		var indexer3 datastore.Indexer3

		var ok5 bool
		var indexer5 datastore.Indexer5

		if indexer5, ok5 = indexer.(datastore.Indexer5); !ok5 {
			indexer3, ok3 = indexer.(datastore.Indexer3)
		}

		if ok3 || ok5 {
			var indexPartition *datastore.IndexPartition

			if node.Partition() != nil {
				indexPartition = &datastore.IndexPartition{Strategy: node.Partition().Strategy(),
					Exprs: node.Partition().Exprs()}
			}

			if ok5 {
				conn, _ := datastore.NewSimpleIndexConnection(context)
				_, err = indexer5.CreatePrimaryIndex5(context.RequestId(), node.Name(), indexPartition, node.With(), conn)
			} else {
				_, err = indexer3.CreatePrimaryIndex3(context.RequestId(), node.Name(), indexPartition, node.With())
			}

			if err != nil {
				if !errors.IsIndexExistsError(err) || this.plan.Node().FailIfExists() {
					context.Error(err)
					return
				} else {
					err = nil
				}
			}
			if context.useCBO && (node.Using() == datastore.GSI || node.Using() == datastore.DEFAULT) && !deferred(node.With()) {
				err = updateStats([]string{node.Name()}, "create_primary_index", this.plan.Keyspace(), context)
				if err != nil {
					context.Error(err)
					return
				}
			}
		} else {
			if node.Partition() != nil {
				context.Error(errors.NewPartitionIndexNotSupportedError())
				return
			}
			_, err = indexer.CreatePrimaryIndex(context.RequestId(), node.Name(), node.With())
			if err != nil {
				if !errors.IsIndexExistsError(err) || this.plan.Node().FailIfExists() {
					context.Error(err)
				} else {
					err = nil
				}
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
