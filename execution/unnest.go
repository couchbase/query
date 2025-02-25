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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Unnest struct {
	base
	buildBitFilterBase
	plan           *plan.Unnest
	timeSeries     bool
	timeSeriesData *expression.TimeSeriesData
}

func NewUnnest(plan *plan.Unnest, context *Context) *Unnest {
	rv := &Unnest{
		plan: plan,
	}
	newBase(&rv.base, context)
	_, rv.timeSeries = plan.Term().Expression().(*expression.TimeSeries)
	rv.output = rv
	return rv
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) Copy() Operator {
	rv := &Unnest{plan: this.plan}
	this.base.copy(&rv.base)
	rv.timeSeries = this.timeSeries
	return rv
}

func (this *Unnest) PlanOp() plan.Operator {
	return this.plan
}

func (this *Unnest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Unnest) beforeItems(context *Context, parent value.Value) bool {
	filter := this.plan.Filter()
	if filter != nil {
		filter.EnableInlistHash(&this.operatorCtx)
		aliasMap := make(map[string]string, 1)
		SetSearchInfo(aliasMap, parent, &this.operatorCtx, filter)
	}
	buildBitFilters := this.plan.GetBuildBitFilters()
	if len(buildBitFilters) > 0 {
		this.createLocalBuildFilters(buildBitFilters)
	}
	return true
}

func (this *Unnest) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.timeSeries {
		return this.processTimeSeriesItem(item, context)
	}

	ev, err := this.plan.Term().Expression().Evaluate(item, &this.operatorCtx)
	if err != nil {
		context.Error(errors.NewEvaluationError(err, "UNNEST path"))
		return false
	}

	// not an array, treat as outer unnest
	if ev.Type() != value.ARRAY {
		return !this.plan.Term().Outer() || this.sendItem(item)
	}

	idx := 0

	// empty, treat as outer unnest
	act, ok := ev.Index(idx)
	if act.Type() == value.MISSING && !ok {
		return !this.plan.Term().Outer() || this.sendItem(item)
	}

	filter := this.plan.Filter()
	buildBitFltr := this.hasBuildBitFilter()
	unnestAlias := this.plan.Alias()

	// Attach and send
	var av value.AnnotatedValue
	var actv value.AnnotatedValue
	useQuota := context.UseRequestQuota()
	for {
		// av and actv are reused if filter evaluation fails for an item, and reset to nil
		// if filter evaluation passes (see below when pass == true).
		if actv == nil {
			actv = value.NewAnnotatedValue(act)
		} else {
			actv.SetValue(act)
		}
		actv.SetAttachment(value.ATT_UNNEST_POSITION, idx)

		idx++
		nextAct, isValidIndex := ev.Index(idx)

		if !isValidIndex {
			if av != nil {
				av.Recycle()
			}
			av = item
			av.SetField(unnestAlias, actv)
		} else {
			if av == nil {
				av = value.NewAnnotatedValue(item.Copy())
			}
			av.SetField(unnestAlias, actv)
		}

		pass := true
		if filter != nil {
			result, err := filter.Evaluate(av, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "unnest filter"))
				return false
			}
			pass = result.Truth()
			result = nil
		}

		if pass {
			if buildBitFltr && !this.buildBitFilters(av, &this.operatorCtx) {
				av.Recycle()
				return false
			}

			if useQuota {
				trackSize := av.Size()
				if !isValidIndex {
					// for the last element, we reuse the original item (which is already tracked)
					trackSize -= item.Size()
				}
				err := context.TrackValueSize(trackSize)
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			}
			if !this.sendItem(av) {
				av.Recycle()
				return false
			} else {
				av = nil
				actv = nil
			}
		}

		// no more
		if !isValidIndex {
			if actv != nil {
				actv.Recycle()
			}
			if av != nil {
				if useQuota {
					context.ReleaseValueSize(av.Size())
				}
				av.Recycle()
			}
			break
		}
		act = nextAct
	}

	return true
}

func (this *Unnest) processTimeSeriesItem(item value.AnnotatedValue, context *Context) bool {
	texpr, ok := this.plan.Term().Expression().(*expression.TimeSeries)
	if !ok {
		return false
	}
	if this.timeSeriesData == nil {
		operands := texpr.Operands()
		var tsKeep bool
		var tsRanges, tsProject value.Value
		var err error
		// Construct timeseries data once per document
		if len(operands) > 1 {
			_, tsKeep, tsRanges, tsProject, err = texpr.GetOptionFields(operands[1], item, &this.operatorCtx)
		}
		if err == nil {
			this.timeSeriesData, err = expression.NewTimeSeriesData(texpr.AliasName(), texpr.TsPaths(),
				tsKeep, tsRanges, tsProject, &this.operatorCtx)
		}
		if err != nil {
			context.Error(errors.NewEvaluationWithCauseError(err, "timeseries expression"))
			return false
		}
	}

	// Evaluate paths against document
	rv, err := this.timeSeriesData.Evaluate(item, context)
	defer this.timeSeriesData.ResetTsData()
	if err != nil {
		context.Error(errors.NewEvaluationWithCauseError(err, "timeseries expression"))
		return false
	}

	if !this.plan.Term().Outer() || !this.timeSeriesData.AllData() {
		if rv != nil && rv.Type() <= value.NULL {
			return true
		}
	}
	if !this.plan.Term().Outer() {
		if qok, _ := this.timeSeriesData.Qualified(false); !qok {
			return true
		}
	}

	var nitem value.AnnotatedValue
	if path, ok := this.timeSeriesData.TsDataExpr().(expression.Path); ok && !this.timeSeriesData.TsKeep() {
		// strip of the tsdata path from original document
		nitem = value.NewAnnotatedValue(item.Copy())
		path.Unset(nitem, &this.operatorCtx)
	} else {
		nitem = item
	}

	if this.plan.Term().Outer() && this.timeSeriesData.AllData() {
		if rv != nil && rv.Type() <= value.NULL {
			return this.sendItem(nitem)
		}
	}

	if this.plan.Term().Outer() {
		if qok, qokOuter := this.timeSeriesData.Qualified(true); !qok {
			return !qokOuter || this.sendItem(nitem)
		}
	}

	// empty, treat as outer unnest
	act, idx, ok := this.timeSeriesData.GetNextValue(0)
	if !ok {
		return !this.plan.Term().Outer() || (idx != 0) || !this.timeSeriesData.AllData() || this.sendItem(nitem)
	}

	filter := this.plan.Filter()
	buildBitFltr := this.hasBuildBitFilter()
	unnestAlias := this.plan.Alias()

	var nextAct value.Value
	var isValidIndex bool
	var av, actv value.AnnotatedValue
	useQuota := context.UseRequestQuota()

	// iterate of over timeseries data points
	for {
		if actv == nil {
			actv = value.NewAnnotatedValue(act)
		} else {
			actv.SetValue(act)
		}
		actv.SetAttachment(value.ATT_UNNEST_POSITION, idx-1)

		nextAct, idx, isValidIndex = this.timeSeriesData.GetNextValue(idx)

		if !isValidIndex {
			if av != nil {
				av.Recycle()
			}
			av = nitem
			av.SetField(unnestAlias, actv)
		} else {
			if av == nil {
				av = value.NewAnnotatedValue(nitem.Copy())
			}
			av.SetField(unnestAlias, actv)
		}

		pass := true
		if filter != nil {
			result, err := filter.Evaluate(av, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationWithCauseError(err, "unnest filter"))
				return false
			}
			pass = result.Truth()
			result = nil
		}

		if pass {
			if buildBitFltr && !this.buildBitFilters(av, &this.operatorCtx) {
				av.Recycle()
				return false
			}

			if useQuota {
				trackSize := av.Size()
				if !isValidIndex {
					// for the last element, we reuse the original item (which is already tracked)
					trackSize -= item.Size()
				}
				err := context.TrackValueSize(trackSize)
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			}
			if !this.sendItem(av) {
				av.Recycle()
				return false
			} else {
				av = nil
				actv = nil
			}
		}

		// no more
		if !isValidIndex {
			if actv != nil {
				actv.Recycle()
			}
			if av != nil {
				if useQuota {
					context.ReleaseValueSize(av.Size())
				}
				av.Recycle()
			}
			break
		}
		act = nextAct
	}

	return true
}

func (this *Unnest) afterItems(context *Context) {
	filter := this.plan.Filter()
	if filter != nil {
		filter.ResetMemory(&this.operatorCtx)
	}
	if this.hasBuildBitFilter() {
		this.setBuildBitFilters(this.plan.Alias(), context)
	}
}

func (this *Unnest) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

var _EMPTY_ACTUALS []interface{}
