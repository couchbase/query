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
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type InitialProject struct {
	base
	plan       *plan.InitialProject
	order      []string
	exclusions [][]string
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
	this.runConsumer(this, context, parent, nil)
}

func (this *InitialProject) beforeItems(context *Context, parent value.Value) bool {
	this.order = nil
	this.exclusions = nil
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
			item.SetSelf(true)
			// remove WITH bindings
			if sv, ok := item.GetValue().(*value.ScopeValue); ok {
				sv.ResetParent()
			}
			// remove LET bindings
			if len(this.plan.BindingNames()) > 0 {
				for k, _ := range this.plan.BindingNames() {
					item.UnsetField(k)
				}
			}
			exclusions, err := this.getExclusions(true, item, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "projection"))
				return false
			}
			return this.excludeAndSend(exclusions, item, context)
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
		v, err := expr.Evaluate(item, &this.operatorCtx)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "projection"))
			e, ok := err.(errors.Error)
			if v == nil || !ok || e.Level() != errors.WARNING {
				return false
			}
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
	this.exclusions = nil
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
	exclusions, err := this.getExclusions(false, item, context)
	if err != nil {
		context.Error(errors.NewEvaluationError(err, "projection"))
		return false
	}

	bindingNames := this.plan.BindingNames()

	for _, term := range this.plan.Terms() {
		alias := term.Result().Alias()
		if alias != "" {
			v, err := term.Result().Expression().Evaluate(item, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "projection"))
				e, ok := err.(errors.Error)
				if v == nil || !ok || e.Level() != errors.WARNING {
					return false
				}
			}
			if term.Result().Self() {
				v = value.NewValue(v.Actual())
			}

			if doOrder {
				order[alias] = nextOrder
				nextOrder++
			}
			// check if we must copy to support the EXCLUDE clause; check if term is referenced in an exclusion (first element)
			// as alias is constant we can cache the result
			if term.MustCopy() == value.NONE {
				if len(exclusions) > 0 {
					found := false
					for i := range exclusions {
						if exclusions[i][0][0] == 'i' && strings.ToLower(exclusions[i][0][1:]) == strings.ToLower(alias) {
							found = true
							break
						} else if exclusions[i][0][1:] == alias {
							found = true
							break
						}
					}
					if found {
						v = v.CopyForUpdate()
						if this.exclusions != nil {
							term.SetMustCopy(value.TRUE)
						}
					} else if this.exclusions != nil {
						term.SetMustCopy(value.FALSE)
					}
				}
			} else if term.MustCopy() == value.TRUE {
				v = v.CopyForUpdate()
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
				starval, err = term.Result().Expression().Evaluate(item, &this.operatorCtx)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "projection"))
					e, ok := err.(errors.Error)
					if starval == nil || !ok || e.Level() != errors.WARNING {
						return false
					}
				}
			}
			// remove bindings
			if len(bindingNames) > 0 {
				starval = starval.Copy()
				for k, _ := range bindingNames {
					starval.UnsetField(k)
				}
			}

			// check if we must copy to support the EXCLUDE clause; check if term is referenced in an exclusion (first element)
			// as each item may have a different schema, always check
			if len(exclusions) > 0 {
				if sa, ok := starval.Actual().(map[string]interface{}); ok {
					found := false
				excl:
					for i := range exclusions {
						if exclusions[i][0][0] == 'i' {
							exclKey := strings.ToLower(exclusions[i][0][1:])
							for k, _ := range sa {
								if strings.ToLower(k) == exclKey {
									found = true
									break excl
								}
							}
						} else if _, ok := sa[exclusions[i][0][1:]]; ok {
							found = true
							break excl
						}
					}
					if found {
						starval = starval.CopyForUpdate()
					}
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
	if this.plan.DiscardOriginal() {
		pv.ResetOriginal()
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
	if !this.excludeAndSend(exclusions, pv, context) {
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

func (this *InitialProject) getExclusions(singlequalification bool, item value.AnnotatedValue, context *Context) (
	[][]string, error) {

	if len(this.plan.Projection().Exclude()) == 0 {
		return nil, nil
	}

	var exclusions [][]string

	if this.exclusions != nil {
		exclusions = this.exclusions
	} else {
		var cache bool
		var err error
		exclusions, cache, err = expression.GetReferences(this.plan.Projection().Exclude(), item, &this.operatorCtx,
			singlequalification)
		if err != nil {
			return nil, nil
		}
		if cache {
			this.exclusions = exclusions
		}
	}
	return exclusions, nil
}

func (this *InitialProject) excludeAndSend(exclusions [][]string, item value.AnnotatedValue, context *Context) bool {
	if len(exclusions) != 0 {
		if ia, ok := item.Actual().(map[string]interface{}); ok {
			before := item.Size()
			for _, e := range exclusions {
				expression.DeleteFromObject(ia, e)
			}
			after := item.RecalculateSize()
			if before != after && context.UseRequestQuota() {
				context.ReleaseValueSize(before - after)
			}
		}
	}
	return this.sendItem(item)
}
