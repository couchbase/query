//  Copyright (c) 2018 Couchbase, Inc.
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
	buildVals value.Values
	probeVals value.Values
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
	this.runConsumer(this, context, parent)
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
		this.plan.Onclause().EnableInlistHash(context)
		SetSearchInfo(this.aliasMap, parent, context, this.plan.Onclause())
	}

	// build hash table
	this.hashTab = util.NewHashTable(util.HASH_TABLE_FOR_HASH_JOIN)

	this.buildVals = make(value.Values, len(this.plan.BuildExprs()))
	this.probeVals = make(value.Values, len(this.plan.ProbeExprs()))

	this.child.SetOutput(this.child)
	this.child.SetInput(nil)
	this.child.SetParent(this)
	this.child.SetStop(nil)

	this.fork(this.child, context, parent)

	return buildHashTab(&(this.base), this.child, this.hashTab,
		this.plan.BuildExprs(), this.buildVals, context)
}

func (this *HashNest) processItem(item value.AnnotatedValue, context *Context) bool {
	defer this.switchPhase(_EXECTIME)

	var err error
	var outVal interface{}
	var right_items value.AnnotatedValues
	ok := true

	probeVal := getProbeVal(item, this.plan.ProbeExprs(), this.probeVals, context)
	if probeVal == nil {
		return false
	}
	outVal, err = this.hashTab.Get(probeVal, value.MarshalValue, value.EqualValue)
	if err != nil {
		context.Error(errors.NewHashTableGetError(err))
		return false
	}
	for outVal != nil {
		if right_item, ok1 := outVal.(value.AnnotatedValue); ok1 {
			var match bool
			aliases := []string{this.plan.BuildAlias()}
			match, ok, _ = processAnsiExec(item, right_item, this.plan.Onclause(),
				aliases, this.ansiFlags, context, "nest")
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
			result, err := this.plan.Filter().Evaluate(joined, context)
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
				if context.TrackValueSize(jSz - iSz) {
					context.Error(errors.NewMemoryQuotaExceededError())
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
	this.plan.Onclause().ResetMemory(context)
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
	rv := this.baseSendAction(action)
	child := this.child
	if rv && child != nil {
		child.SendAction(action)
	}
}

func (this *HashNest) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
}
