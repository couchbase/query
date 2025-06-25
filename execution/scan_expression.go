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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type ExpressionScan struct {
	base
	buildBitFilterBase
	plan    *plan.ExpressionScan
	results []interface{}
	context *Context
}

func NewExpressionScan(plan *plan.ExpressionScan, context *Context) *ExpressionScan {
	rv := &ExpressionScan{
		plan:    plan,
		context: context,
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

		useCache := false
		subq, isSubq := this.plan.FromExpr().(*algebra.Subquery)
		var subqIsCorrelated bool
		if isSubq {
			subqIsCorrelated = subq.IsCorrelated()
		}
		if !this.plan.IsCorrelated() {
			if isSubq {
				// if the subquery evaluation is caching result already, no need
				// to cache result here
				useCache = !useSubqCachedResult(subq.Select())
			} else {
				useCache = true
			}
		}

		alias := this.plan.Alias()

		var buildBitFltr bool
		buildBitFilters := this.plan.GetBuildBitFilters()
		if len(buildBitFilters) > 0 {
			this.createLocalBuildFilters(buildBitFilters)
			buildBitFltr = this.hasBuildBitFilter()
			defer this.setBuildBitFilters(alias, context)
		}

		// use cached results if available
		if useCache && this.results != nil {
			for _, act := range this.results {
				actv := value.NewScopeValue(make(map[string]interface{}), parent)
				actv.SetField(alias, act)
				av := value.NewAnnotatedValue(actv)
				av.SetId("")

				if buildBitFltr && !this.buildBitFilters(av, &this.operatorCtx) {
					return
				}

				if this.plan.IsUnderNL() {
					// Reset Covers (inherited from parent) if under nested-loop join
					av.ResetCovers(nil)
				}

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
					break
				}
			}
			return
		}

		filter := this.plan.Filter()
		if filter != nil {
			filter.EnableInlistHash(&this.operatorCtx)
			defer filter.ResetMemory(&this.operatorCtx)
		}

		ev, e := this.plan.FromExpr().Evaluate(parent, &this.operatorCtx)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "ExpressionScan"))
			return
		}
		if ev == nil {
			return
		}
		var freeSize uint64
		if context.UseRequestQuota() {
			freeSize = ev.Size()
			// Track output of expression evaluation as necessary:
			// 1. For correlated subqueries:
			//    The values are already tracked as part of (repeated) execution of the subquery.
			// 2. For non-correlated subqueries:
			//    The values come from the subquery result cache and are not tracked (except for first execution).
			// 3. for non-subqueries, the values are not tracked
			if !isSubq || !subqIsCorrelated {
				err := context.TrackValueSize(freeSize)
				if err != nil {
					context.Error(err)
					return
				}
			}
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
		var results []interface{}
		if useCache {
			this.results = nil
			results = make([]interface{}, 0, len(acts))
		}
		for _, act := range acts {
			actv := value.NewScopeValue(make(map[string]interface{}), parent)
			actv.SetField(alias, act)
			av := value.NewAnnotatedValue(actv)
			av.SetId("")

			if filter != nil {
				result, err := filter.Evaluate(av, &this.operatorCtx)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "expression scan filter"))
					return
				}
				if !result.Truth() {
					av.Recycle()
					continue
				}
			}

			if buildBitFltr && !this.buildBitFilters(av, &this.operatorCtx) {
				return
			}

			if this.plan.IsUnderNL() {
				// Reset Covers (inherited from parent) if under nested-loop join
				// (this needs to be done after the expression evaluations above)
				av.ResetCovers(nil)
			}

			if useCache {
				results = append(results, act)
			}
			if context.UseRequestQuota() {
				actSz := value.AnySize(act)
				if !useCache {
					freeSize -= actSz
				}
				err := context.TrackValueSize(av.Size() - actSz)
				if err != nil {
					context.Error(err)
					av.Recycle()
					return
				}
			}
			if !this.sendItem(av) {
				av.Recycle()
				return
			}
		}
		if !useCache && freeSize > 0 && context.UseRequestQuota() {
			context.ReleaseValueSize(freeSize)
		}
		this.results, results = results, nil
	})
}

func (this *ExpressionScan) Done() {
	this.baseDone()
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
