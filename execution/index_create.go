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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
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

	newRedirectBase(&rv.base, context)
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

func (this *CreateIndex) PlanOp() plan.Operator {
	return this.plan
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

		var ok3 bool
		var indexer3 datastore.Indexer3

		var ok5 bool
		var indexer5 datastore.Indexer5

		var ok6 bool
		var indexer6 datastore.Indexer6

		if indexer6, ok6 = indexer.(datastore.Indexer6); !ok6 {
			if indexer5, ok5 = indexer.(datastore.Indexer5); !ok5 {
				indexer3, ok3 = indexer.(datastore.Indexer3)
			}
		}

		var include expression.Expressions
		if node.Include() != nil {
			include = node.Include().Expressions()
		}
		isVector := node.Keys().HasVector() && (node.Using() == datastore.GSI || node.Using() == datastore.DEFAULT)

		if !ok6 && isVector {
			context.Error(errors.NewIndexerVersionError(datastore.INDEXER6_VERSION, "Index key has vector attribute"))
			return
		} else if !ok6 && include != nil {
			context.Error(errors.NewIndexerVersionError(datastore.INDEXER6_VERSION, "Include clasue present"))
			return
		}

		var idx datastore.Index
		if ok3 || ok5 || ok6 {
			var indexPartition *datastore.IndexPartition

			if node.Partition() != nil {
				indexPartition = &datastore.IndexPartition{Strategy: node.Partition().Strategy(),
					Exprs: node.Partition().Expressions()}
			}

			rangeKeys := this.getRangeKeys(node.Keys())

			if ok6 {
				conn, _ := datastore.NewSimpleIndexConnection(context)
				idx, err = indexer6.CreateIndex6(context.RequestId(), node.Name(),
					isVector && node.Vector(), rangeKeys,
					indexPartition, node.Where(), node.With(), include, conn)
			} else if ok5 {
				conn, _ := datastore.NewSimpleIndexConnection(context)
				idx, err = indexer5.CreateIndex5(context.RequestId(), node.Name(), rangeKeys,
					indexPartition, node.Where(), node.With(), conn)
			} else {
				idx, err = indexer3.CreateIndex3(context.RequestId(), node.Name(), rangeKeys,
					indexPartition, node.Where(), node.With())
			}

			if err != nil {
				if errors.IsIndexExistsError(err) {
					if this.plan.Node().FailIfExists() {
						err = errors.NewIndexAlreadyExistsError(node.Name())
					} else {
						err = nil
					}
				}
				if err != nil {
					context.Error(err)
					return
				}
			} else if context.useCBO && (node.Using() == datastore.GSI || node.Using() == datastore.DEFAULT) &&
				!deferred(node.With()) && !isVector {

				err = updateStats([]string{node.Name()}, "create_index", this.plan.Keyspace(), context)
				if err != nil {
					context.Error(err)
					return
				}
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
				idx, err = indexer2.CreateIndex2(context.RequestId(), node.Name(), node.SeekKeys(),
					rangeKeys, node.Where(), node.With())
				if err != nil {
					if errors.IsIndexExistsError(err) {
						if this.plan.Node().FailIfExists() {
							err = errors.NewIndexAlreadyExistsError(node.Name())
						} else {
							err = nil
						}
					}
					if err != nil {
						context.Error(err)
						return
					}
				}
			} else {
				if node.Keys().HasDescending() {
					context.Error(errors.NewIndexerDescCollationError())
					return
				}

				idx, err = indexer.CreateIndex(context.RequestId(), node.Name(), node.SeekKeys(),
					node.RangeKeys(), node.Where(), node.With())
				if err != nil {
					if errors.IsIndexExistsError(err) {
						if this.plan.Node().FailIfExists() {
							err = errors.NewIndexAlreadyExistsError(node.Name())
						} else {
							err = nil
						}
					}
					if err != nil {
						context.Error(err)
						return
					}
				}
			}
		}

		m := make(map[string]interface{}, 2)
		m["name"] = node.Name()
		if idx != nil {
			m["id"] = idx.Id()
			state, msg, err := idx.State()
			if err == nil {
				m["state"] = state.String()
				if msg != "" {
					m["message"] = msg
				}
			}
		}
		av := value.NewAnnotatedValue(m)
		if context.UseRequestQuota() {
			err := context.TrackValueSize(av.Size())
			if err != nil {
				context.Error(err)
				av.Recycle()
				return
			}
		}
		if !this.sendItem(av) {
			av.Recycle()
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
		if term.HasAttribute(algebra.IK_VECTOR) {
			attrs |= datastore.IK_VECTOR
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

func deferred(with value.Value) bool {
	if with != nil && with.Type() == value.OBJECT {
		if deferred, ok := with.Field("defer_build"); ok {
			return deferred.Truth()
		}
	}
	return false
}
