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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _SET_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_SET_OP_POOL, func() interface{} {
		return &Set{}
	})
}

// Write to copy
type Set struct {
	base
	plan *plan.Set
}

func NewSet(plan *plan.Set, context *Context) *Set {
	rv := _SET_OP_POOL.Get().(*Set)
	rv.plan = plan

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func (this *Set) Copy() Operator {
	rv := _SET_OP_POOL.Get().(*Set)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *Set) PlanOp() plan.Operator {
	return this.plan
}

func (this *Set) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Set) processItem(item value.AnnotatedValue, context *Context) bool {
	atmt := item.GetAttachment(value.ATT_CLONE)
	if atmt == nil {
		context.Error(errors.NewUpdateMissingClone())
		return false
	}

	clone, ok := atmt.(value.AnnotatedValue)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid UPDATE clone of type %T.", clone)))
		return false
	}

	var err error
	for _, t := range this.plan.Node().Terms() {
		clone, err = setPath(t, clone, item, &this.operatorCtx)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "SET clause"))
			return false
		}
	}

	item.SetAttachment(value.ATT_CLONE, clone)
	return this.sendItem(item)
}

func setPath(t *algebra.SetTerm, clone, item value.AnnotatedValue, context *opContext) (value.AnnotatedValue, error) {

	// make sure we don't used a possibly stale cached value
	t.Value().ResetValue()

	if t.UpdateFor() != nil {
		return setFor(t, clone, item, context)
	}

	v, err := t.Value().Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if t.Value().Static() != nil {
		v = v.CopyForUpdate()
	}

	if t.Meta() != nil {
		if opVal, ok := clone.GetAttachment(value.ATT_OPTIONS).(value.Value); ok && opVal.Type() != value.MISSING {
			if v.Type() == value.MISSING {
				if f, ok := t.Path().(*expression.Field); ok && f.First().Alias() == "xattrs" {
					// convert MISSING to nil so it is processed as a deletion of the top-level xattr
					v = nil
				}
			}
			if !t.Path().Set(opVal, v, context) {
				// if we're trying to set an xattr and none were retrieved previously, we'll need to create the empty value first
				if f, ok := t.Path().(*expression.Field); ok && f.First().Alias() == "xattrs" {
					if xattrs, ok := opVal.Field("xattrs"); ok && xattrs == nil || xattrs.Type() == value.NULL {
						opVal.SetField("xattrs", value.NewValue(map[string]interface{}{}))
						// try again
						t.Path().Set(opVal, v, context)
					}
				}
			}
		}
	} else {
		t.Path().Set(clone, v, context)
	}

	return clone, err
}

func setFor(t *algebra.SetTerm, clone, item value.AnnotatedValue, context *opContext) (value.AnnotatedValue, error) {
	ivals, mismatch, err := buildFor(t.UpdateFor(), item, context)
	defer releaseValsFor(ivals)
	if err != nil {
		return nil, err
	}

	if mismatch {
		return clone, nil
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
	if len(ivals) != len(cvals) {
		return clone, nil
	}
	when := t.UpdateFor().When()
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

		v, err := t.Value().Evaluate(ivals[i], context)
		if err != nil {
			return nil, err
		}
		if t.Value().Static() != nil {
			v = v.CopyForUpdate()
		}
		t.Path().Set(cvals[i], v, context)
	}

	return clone, nil
}

func (this *Set) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Set) Done() {
	this.baseDone()
	if this.isComplete() {
		_SET_OP_POOL.Put(this)
	}
}
