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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type InitialProject struct {
	base
	plan  *plan.InitialProject
	order []string
}

var _INITPROJ_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_INITPROJ_OP_POOL, func() interface{} {
		return &InitialProject{}
	})
}

func NewInitialProject(plan *plan.InitialProject, context *Context) *InitialProject {
	rv := _INITPROJ_OP_POOL.Get().(*InitialProject)
	rv.plan = plan
	newBase(&rv.base, context)
	rv.output = rv
	rv.execPhase = PROJECT
	return rv
}

func (this *InitialProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialProject(this)
}

func (this *InitialProject) Copy() Operator {
	rv := _INITPROJ_OP_POOL.Get().(*InitialProject)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *InitialProject) PlanOp() plan.Operator {
	return this.plan
}

func (this *InitialProject) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *InitialProject) beforeItems(context *Context, parent value.Value) bool {
	this.order = nil
	return true
}

func (this *InitialProject) processItem(item value.AnnotatedValue, context *Context) bool {
	terms := this.plan.Terms()
	n := len(terms)

	if n > 1 {
		return this.processTerms(item, context)
	}

	if n == 0 {
		return this.sendItem(item)
	}

	// n == 1

	result := terms[0].Result()
	expr := result.Expression()

	if result.Star() && result.Self() {
		// Unprefixed star
		if item.Type() == value.OBJECT {
			item.SetValue(value.NewValue(item.Actual()))
			item.SetSelf(true)
			return this.sendItem(item)
		} else {
			pv := value.EMPTY_ANNOTATED_OBJECT
			if context.UseRequestQuota() {
				iSz := item.Size()
				pSz := pv.Size()
				if pSz > iSz {
					err := context.TrackValueSize(pSz - iSz)
					if err != nil {
						context.Error(err)
						item.Recycle()
						return false
					}
				} else {
					context.ReleaseValueSize(iSz - pSz)
				}
			}
			item.Recycle()
			return this.sendItem(pv)
		}
	} else if this.plan.Projection().Raw() {
		// Raw projection of an expression
		v, err := expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "projection"))
			return false
		}
		if av, ok := v.(value.AnnotatedValue); ok && av.Seen() {
			av.Track()
		}

		if result.Self() {
			v = value.NewValue(v.Actual())
		}
		sv := value.NewScopeValue(make(map[string]interface{}, 1), item)
		if result.As() != "" {
			sv.SetField(result.As(), v)
		}
		av := value.NewAnnotatedValue(sv)
		av.ShareAnnotations(item)
		av.SetProjection(v, nil)
		if context.UseRequestQuota() {
			iSz := item.Size()
			aSz := av.Size()
			if aSz > iSz {
				err := context.TrackValueSize(aSz - iSz)
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			} else {
				context.ReleaseValueSize(iSz - aSz)
			}
		}
		item.Recycle()
		return this.sendItem(av)
	} else {
		// Any other projection
		return this.processTerms(item, context)
	}
}

func (this *InitialProject) afterItems(context *Context) {
	if context.IsAdvisor() {
		context.AddPhaseOperator(ADVISOR)
	}
}

func (this *InitialProject) processTerms(item value.AnnotatedValue, context *Context) bool {
	n := len(this.plan.Terms())
	sv := value.NewScopeValue(make(map[string]interface{}, n), item)
	pv := value.NewAnnotatedValue(sv)
	pv.ShareAnnotations(item)

	p := value.NewValue(make(map[string]interface{}, n+(this.plan.StarTermCount()*7)))

	var order map[string]int
	nextOrder := 0
	doOrder := context.PreserveProjectionOrder() && this.plan.PreserveOrder() && this.order == nil
	if doOrder {
		order = _PROJECTION_ORDER_POOL.Get()
	}
	for _, term := range this.plan.Terms() {
		alias := term.Result().Alias()
		if alias != "" {
			v, err := term.Result().Expression().Evaluate(item, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "projection"))
				return false
			}
			if term.Result().Self() {
				v = value.NewValue(v.Actual())
			}

			if doOrder {
				order[alias] = nextOrder
				nextOrder++
			}
			p.SetField(alias, v)

			// Explicit aliases override data
			if term.Result().As() != "" {
				pv.SetField(term.Result().As(), v)
			}
		} else {
			// Star
			starval := item.GetValue()
			if term.Result().Expression() != nil {
				var err error
				starval, err = term.Result().Expression().Evaluate(item, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "projection"))
					return false
				}
			}

			// Latest star overwrites previous star
			if sa, ok := starval.Actual().(map[string]interface{}); ok {
				if doOrder {
					for k, v := range sa {
						order[k] = nextOrder
						p.SetField(k, v)
					}
					nextOrder++
				} else {
					for k, v := range sa {
						p.SetField(k, v)
					}
				}
			}
		}
	}
	if this.order != nil {
		pv.SetProjection(p, this.order)
	} else if order != nil {
		sl := &sortableList{}
		sl.list = make([]string, len(order))
		sl.mp = &order
		if nextOrder == len(order) && this.plan.StarTermCount() == 0 {
			for k, v := range order {
				sl.list[v] = k
			}
		} else {
			i := 0
			for k, _ := range order {
				sl.list[i] = k
				i++
			}
			sort.Sort(sl)
		}
		sl.mp = nil
		_PROJECTION_ORDER_POOL.Put(order)
		if this.plan.StarTermCount() == 0 {
			this.order = sl.list
		}
		pv.SetProjection(p, sl.list)
	} else {
		pv.SetProjection(p, nil)
	}
	if context.UseRequestQuota() {
		iSz := item.Size()
		pSz := pv.Size()
		if pSz > iSz {
			err := context.TrackValueSize(pSz - iSz)
			if err != nil {
				context.Error(err)
				pv.Recycle()
				return false
			}
		} else {
			context.ReleaseValueSize(iSz - pSz)
		}
	}
	item.Recycle()
	if !this.sendItem(pv) {
		pv.Recycle()
		return false
	}
	return true
}

func (this *InitialProject) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *InitialProject) Done() {
	this.order = nil
	this.baseDone()
	if this.isComplete() {
		_INITPROJ_OP_POOL.Put(this)
	}
}

var _PROJECTION_ORDER_POOL = util.NewStringIntPool(256)

type sortableList struct {
	list []string
	mp   *map[string]int
}

func (this sortableList) Len() int { return len(this.list) }
func (this sortableList) Less(i int, j int) bool {
	a := (*this.mp)[this.list[i]]
	b := (*this.mp)[this.list[j]]
	if a == b {
		return this.list[i] < this.list[j]
	}
	return a < b
}
func (this sortableList) Swap(i int, j int) {
	this.list[i], this.list[j] = this.list[j], this.list[i]
}
