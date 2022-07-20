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
	_ "fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type ExpressionScan struct {
	base
	buildBitFilterBase
	plan    *plan.ExpressionScan
	results value.AnnotatedValues
}

func NewExpressionScan(plan *plan.ExpressionScan, context *Context) *ExpressionScan {
	rv := &ExpressionScan{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *ExpressionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExpressionScan(this)
}

func (this *ExpressionScan) Copy() Operator {
	rv := &ExpressionScan{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *ExpressionScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *ExpressionScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.cleanup(context)
		if !active {
			return
		}

		correlated := this.plan.IsCorrelated()

		// use cached results if available
		if !correlated && this.results != nil {
			for _, av := range this.results {
				av.Track()
				if context.UseRequestQuota() && context.TrackValueSize(av.Size()) {
					context.Error(errors.NewMemoryQuotaExceededError())
					av.Recycle()
					return
				}
				if !this.sendItem(av) {
					av.Recycle()
					break
				}
			}
			return
		}

		filter := this.plan.Filter()
		if filter != nil {
			filter.EnableInlistHash(context)
			defer filter.ResetMemory(context)
		}

		alias := this.plan.Alias()

		var buildBitFltr bool
		buildBitFilters := this.plan.GetBuildBitFilters()
		if len(buildBitFilters) > 0 {
			this.createLocalBuildFilters(buildBitFilters)
			buildBitFltr = this.hasBuildBitFilter()
			defer this.setBuildBitFilters(alias, context)
		}

		ev, e := this.plan.FromExpr().Evaluate(parent, context)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "ExpressionScan"))
			return
		}
		if ev == nil {
			return
		}

		actuals := ev.Actual()
		switch actuals.(type) {
		case []interface{}:
		case nil:
			if ev.Type() == value.NULL {
				actuals = _ARRAY_NULL_VALUE
			} else {
				actuals = _ARRAY_MISSING_VALUE
			}
		default:
			actuals = []interface{}{actuals}
		}

		acts := actuals.([]interface{})
		var results value.AnnotatedValues
		if !correlated {
			this.results = nil
			results = make(value.AnnotatedValues, 0, len(acts))
			defer func() {
				for i := range results {
					results[i].Recycle()
				}
			}()
		}
		for _, act := range acts {
			actv := value.NewScopeValue(make(map[string]interface{}), parent)
			actv.SetField(this.plan.Alias(), act)
			av := value.NewAnnotatedValue(actv)
			av.SetId("")

			if filter != nil {
				result, err := filter.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "expression scan filter"))
					return
				}
				if !result.Truth() {
					av.Recycle()
					continue
				}
			}

			if buildBitFltr && !this.buildBitFilters(av, context) {
				return
			}

			if !correlated {
				av.Track()
				if context.UseRequestQuota() && context.TrackValueSize(av.Size()) {
					context.Error(errors.NewMemoryQuotaExceededError())
					av.Recycle()
					return
				}
				results = append(results, av)
			}
			if context.UseRequestQuota() && context.TrackValueSize(av.Size()) {
				context.Error(errors.NewMemoryQuotaExceededError())
				av.Recycle()
				return
			}
			if !this.sendItem(av) {
				av.Recycle()
				return
			}
		}
		this.results, results = results, nil
	})
}

func (this *ExpressionScan) Done() {
	this.baseDone()
	for i := range this.results {
		this.results[i].Recycle()
	}
	this.results = nil
}

func (this *ExpressionScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

var _ARRAY_NULL_VALUE []interface{} = []interface{}{value.NULL_VALUE}
var _ARRAY_MISSING_VALUE []interface{} = []interface{}(nil)
