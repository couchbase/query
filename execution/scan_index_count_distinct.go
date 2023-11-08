//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexCountDistinctScan2 struct {
	base
	plan *plan.IndexCountDistinctScan2
}

func NewIndexCountDistinctScan2(plan *plan.IndexCountDistinctScan2, context *Context) *IndexCountDistinctScan2 {
	rv := &IndexCountDistinctScan2{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexCountDistinctScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountDistinctScan2(this)
}

func (this *IndexCountDistinctScan2) Copy() Operator {
	rv := &IndexCountDistinctScan2{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexCountDistinctScan2) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexCountDistinctScan2) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_COUNT, context)
		defer this.cleanup(context)
		if !active {
			return
		}

		var count int64

		keyspaceTerm := this.plan.Term()
		scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Path().Bucket())
		dspans, empty, err := evalSpan2(this.plan.Spans(), nil, &this.operatorCtx)
		if err == nil && !empty {
			count, err = this.plan.Index().CountDistinct(context.RequestId(), dspans, context.ScanConsistency(), scanVector)
		}

		if err != nil {
			context.Error(errors.NewEvaluationError(err, "scanCountDistinct()"))
		}

		av := value.NewAnnotatedValue(count)
		av.InheritCovers(parent)
		this.sendItem(av)
	})
}

func (this *IndexCountDistinctScan2) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
