//  Copyright 2018-Present Couchbase, Inc.
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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type HashNest struct {
	base
	plan      *plan.HashNest
	child     Operator
	aliasMap  map[string]string
	ansiFlags uint32
	hashTab   *util.HashTable
	buildVals []interface{}
	probeVals []interface{}
}

func NewHashNest(plan *plan.HashNest, context *Context, child Operator, aliasMap map[string]string) *HashNest {
	rv := &HashNest{
		plan:     plan,
		child:    child,
		aliasMap: aliasMap,
	}

	newBase(&rv.base, context)
	rv.trackChildren(1)
	rv.execPhase = HASH_NEST
	rv.output = rv
	return rv
}

func (this *HashNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitHashNest(this)
}

func (this *HashNest) Copy() Operator {
	rv := &HashNest{
		plan:     this.plan,
		child:    this.child.Copy(),
		aliasMap: this.aliasMap,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *HashNest) PlanOp() plan.Operator {
	return this.plan
}

func (this *HashNest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *HashNest) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.child != nil, "HASH NEST has no child") {
		return false
	}
	if !context.assert(this.plan.Onclause() != nil, "HASH NEST does not have onclause") {
		return false
	}

	// check for constant TRUE or FALSE onclause
	cpred := this.plan.Onclause().Value()
	if cpred != nil {
		if cpred.Truth() {
			this.ansiFlags |= ANSI_ONCLAUSE_TRUE
		} else {
			this.ansiFlags |= ANSI_ONCLAUSE_FALSE
		}
	} else {
		this.plan.Onclause().EnableInlistHash(&this.operatorCtx)
		SetSearchInfo(this.aliasMap, parent, &this.operatorCtx, this.plan.Onclause())
	}

	filter := this.plan.Filter()
	if filter != nil {
		filter.EnableInlistHash(&this.operatorCtx)
	}

	// build hash table
	this.hashTab = util.NewHashTable(util.HASH_TABLE_FOR_HASH_JOIN, this.child.PlanOp().Cardinality(), len(this.plan.BuildExprs()))

	this.buildVals = make([]interface{}, len(this.plan.BuildExprs()))
	this.probeVals = make([]interface{}, len(this.plan.ProbeExprs()))

	this.child.SetOutput(this.child)
	this.child.SetInput(nil)
	this.child.SetParent(this)
	this.child.SetStop(nil)

	this.fork(this.child, context, parent)

	return buildHashTab(&(this.base), this.child, this.hashTab,
		this.plan.BuildExprs(), this.buildVals, &this.operatorCtx)
}

func (this *HashNest) processItem(item value.AnnotatedValue, context *Context) bool {
	defer this.switchPhase(_EXECTIME)

	var err error
	var outVal interface{}
	var right_items value.AnnotatedValues
	ok := true

	err1 := getProbeVal(item, this.plan.ProbeExprs(), this.probeVals, &this.operatorCtx)
	if err1 != nil {
		context.Error(err1)
		return false
	}

	var probeVal interface{}
	var marshal func(interface{}) ([]byte, error)
	var equal func(interface{}, interface{}) bool
	if len(this.probeVals) == 1 {
		probeVal = this.probeVals[0]
		marshal = value.MarshalValue
		equal = value.EqualValue
	} else {
		probeVal = this.probeVals
		marshal = value.MarshalArray
		equal = value.EqualArray
	}

	outVal, err = this.hashTab.Get(probeVal, marshal, equal)
	if err != nil {
		context.Error(errors.NewHashTableGetError(err))
		return false
	}
	for outVal != nil {
		if right_item, ok1 := outVal.(value.AnnotatedValue); ok1 {
			var match bool
			aliases := []string{this.plan.BuildAlias()}
			match, ok, _ = processAnsiExec(item, right_item, this.plan.Onclause(),
				aliases, this.ansiFlags, &this.operatorCtx, "nest")
			if match && ok {
				right_items = append(right_items, right_item)
			}
		} else {
			context.Error(errors.NewExecutionInternalError("Hash Table Get produced non-Annotated value"))
			return false
		}

		outVal, err = this.hashTab.GetNext()
		if err != nil {
			context.Error(errors.NewHashTableGetError(err))
			return false
		}
	}

	var joined value.AnnotatedValue
	joined, ok = processAnsiNest(item, right_items, this.plan.BuildAlias(), this.plan.Outer(), context)
	if !ok {
		return false
	}
	if joined != nil {
		if this.plan.Filter() != nil {
			result, err := this.plan.Filter().Evaluate(joined, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "hash nest filter"))
				return false
			}
			if !result.Truth() {
				return true
			}
		}
		if context.UseRequestQuota() {
			iSz := item.Size()
			jSz := joined.Size()
			if jSz > iSz {
				err := context.TrackValueSize(jSz - iSz)
				if err != nil {
					context.Error(err)
					return false
				}
			} else {
				context.ReleaseValueSize(iSz - jSz)
			}
		}
		return this.sendItem(joined)
	}
	// TODO Recycle

	return true
}

func (this *HashNest) afterItems(context *Context) {
	this.dropHashTable(context)
	if (this.ansiFlags & (ANSI_ONCLAUSE_TRUE | ANSI_ONCLAUSE_FALSE)) == 0 {
		this.plan.Onclause().ResetMemory(&this.operatorCtx)
	}
	filter := this.plan.Filter()
	if filter != nil {
		filter.ResetMemory(&this.operatorCtx)
	}
}

func (this *HashNest) dropHashTable(context *Context) {
	if this.hashTab != nil {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(this.hashTab.Size())
		}
		this.hashTab.Drop()
		this.hashTab = nil
	}
}

func (this *HashNest) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~child"] = this.child
	})
	return json.Marshal(r)
}

func (this *HashNest) SendAction(action opAction) {
	this.baseSendAction(action)
	child := this.child
	if child != nil {
		child.SendAction(action)
	}
}

func (this *HashNest) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.child != nil {
		rv = this.child.reopen(context)
	}
	return rv
}

func (this *HashNest) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
}
