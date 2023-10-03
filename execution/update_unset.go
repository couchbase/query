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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _UNSET_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_UNSET_OP_POOL, func() interface{} {
		return &Unset{}
	})
}

// Write to copy
type Unset struct {
	base
	plan *plan.Unset
}

func NewUnset(plan *plan.Unset, context *Context) *Unset {
	rv := _UNSET_OP_POOL.Get().(*Unset)
	rv.plan = plan

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) Copy() Operator {
	rv := _UNSET_OP_POOL.Get().(*Unset)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *Unset) PlanOp() plan.Operator {
	return this.plan
}

func (this *Unset) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Unset) processItem(item value.AnnotatedValue, context *Context) bool {
	clone, ok := item.GetAttachment("clone").(value.AnnotatedValue)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid UPDATE clone of type %T.", clone)))
		return false
	}

	var err error
	for _, t := range this.plan.Node().Terms() {
		clone, err = unsetPath(t, clone, item, &this.operatorCtx)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "UNSET clause"))
			return false
		}
	}

	item.SetAttachment("clone", clone)
	return this.sendItem(item)
}

func unsetPath(t *algebra.UnsetTerm, clone, item value.AnnotatedValue, context *opContext) (
	value.AnnotatedValue, error) {
	if t.UpdateFor() != nil {
		return unsetFor(t, clone, item, context)
	}

	t.Path().Unset(clone, context)
	return clone, nil
}

func unsetFor(t *algebra.UnsetTerm, clone, item value.AnnotatedValue, context *opContext) (
	value.AnnotatedValue, error) {
	var ivals []value.Value

	when := t.UpdateFor().When()
	if when != nil {
		vals, mismatch, err := buildFor(t.UpdateFor(), item, context)
		ivals = vals
		defer releaseValsFor(ivals)
		if err != nil {
			return nil, err
		}

		if mismatch {
			return clone, nil
		}
	}

	cvals, mismatch, err := buildFor(t.UpdateFor(), clone, context)
	defer releaseValsFor(cvals)
	if err != nil {
		return nil, err
	}

	if mismatch {
		return clone, nil
	}

	// Clone may have been mutated by another term
	if ivals != nil && len(ivals) != len(cvals) {
		return clone, nil
	}

	for i := 0; i < len(cvals); i++ {
		if when != nil {
			w, err := when.Evaluate(ivals[i], context)
			if err != nil {
				return nil, err
			}

			if !w.Truth() {
				continue
			}
		}

		t.Path().Unset(cvals[i], context)
	}

	return clone, nil
}

func (this *Unset) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Unset) Done() {
	this.baseDone()
	if this.isComplete() {
		_UNSET_OP_POOL.Put(this)
	}
}
