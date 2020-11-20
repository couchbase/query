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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type InitialProject struct {
	base
	plan *plan.InitialProject
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

	if result.Star() && (expr == expression.SELF || expr == nil) {
		// Unprefixed star
		if item.Type() == value.OBJECT {
			item.SetSelf(true)
			return this.sendItem(item)
		} else {
			pv := value.EMPTY_ANNOTATED_OBJECT
			if context.UseRequestQuota() {
				pv := value.EMPTY_ANNOTATED_OBJECT
				iSz := item.Size()
				pSz := pv.Size()
				if pSz > iSz {
					if context.TrackValueSize(pSz - iSz) {
						context.Error(errors.NewMemoryQuotaExceededError())
						return false
					}
				} else {
					context.ReleaseValueSize(iSz - pSz)
				}
			}
			return this.sendItem(pv)
		}
	} else if this.plan.Projection().Raw() {
		// Raw projection of an expression
		v, err := expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "projection"))
			return false
		}

		sv := value.NewScopeValue(make(map[string]interface{}, 1), item)
		if result.As() != "" {
			sv.SetField(result.As(), v)
		}
		av := value.NewAnnotatedValue(sv)
		av.ShareAnnotations(item)
		av.SetProjection(v)
		if context.UseRequestQuota() {
			iSz := item.Size()
			aSz := av.Size()
			if aSz > iSz {
				if context.TrackValueSize(aSz - iSz) {
					context.Error(errors.NewMemoryQuotaExceededError())
					av.Recycle()
					return false
				}
			} else {
				context.ReleaseValueSize(iSz - aSz)
			}
		}
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

	for _, term := range this.plan.Terms() {
		if term.Result().Alias() != "" {
			v, err := term.Result().Expression().Evaluate(item, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "projection"))
				return false
			}

			p.SetField(term.Result().Alias(), v)

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
			switch sa := starval.Actual().(type) {
			case map[string]interface{}:
				for k, v := range sa {
					p.SetField(k, v)
				}
			}
		}
	}

	pv.SetProjection(p) //	pv.SetAttachment("projection", p)
	if context.UseRequestQuota() {
		iSz := item.Size()
		pSz := pv.Size()
		if pSz > iSz {
			if context.TrackValueSize(pSz - iSz) {
				context.Error(errors.NewMemoryQuotaExceededError())
				pv.Recycle()
				return false
			}
		} else {
			context.ReleaseValueSize(iSz - pSz)
		}
	}
	return this.sendItem(pv)
}

func (this *InitialProject) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *InitialProject) Done() {
	this.baseDone()
	if this.isComplete() {
		_INITPROJ_OP_POOL.Put(this)
	}
}
