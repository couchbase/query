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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type HashJoin struct {
	base
	plan      *plan.HashJoin
	child     Operator
	aliasMap  map[string]string
	ansiFlags uint32
	hashTab   *util.HashTable
	buildVals value.Values
	probeVals value.Values
}

func NewHashJoin(plan *plan.HashJoin, context *Context, child Operator, aliasMap map[string]string) *HashJoin {
	rv := &HashJoin{
		plan:     plan,
		child:    child,
		aliasMap: aliasMap,
	}

	newBase(&rv.base, context)
	rv.trackChildren(1)
	rv.execPhase = HASH_JOIN
	rv.output = rv
	return rv
}

func (this *HashJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitHashJoin(this)
}

func (this *HashJoin) Copy() Operator {
	rv := &HashJoin{
		plan:     this.plan,
		child:    this.child.Copy(),
		aliasMap: this.aliasMap,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *HashJoin) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *HashJoin) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.child != nil, "HASH JOIN has no child") {
		return false
	}
	if !context.assert(this.plan.Onclause() != nil, "HASH JOIN does not have onclause") {
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

	go this.child.RunOnce(context, parent)

	return buildHashTab(&(this.base), this.child, this.hashTab,
		this.plan.BuildExprs(), this.buildVals, context)
}

func buildHashTab(base *base, buildOp Operator, hashTab *util.HashTable,
	buildExprs expression.Expressions, buildVals value.Values, context *Context) bool {
	var err error
	stopped := false
	n := 1

loop:
	for {
		build_item, child, cont := base.getItemChildrenOp(buildOp)
		if cont {
			if build_item != nil {
				for i, be := range buildExprs {
					buildVals[i], err = be.Evaluate(build_item, context)
					if err != nil {
						context.Error(errors.NewEvaluationError(err, "Hash Table Build Expression"))
						return false
					}
				}
				var buildVal value.Value
				if len(buildVals) == 1 {
					buildVal = buildVals[0]
				} else {
					buildVal = value.NewValue(buildVals)
				}
				err = hashTab.Put(buildVal, build_item, value.MarshalValue, value.EqualValue)
				if err != nil {
					context.Error(errors.NewHashTablePutError(err))
					return false
				}
			} else if child >= 0 {
				n--
			} else {
				break loop
			}
		} else {
			stopped = true
			break loop
		}
	}

	if n > 0 {
		notifyChildren(buildOp)
		base.childrenWaitNoStop(n)
	}

	if stopped {
		return false
	}

	return true
}

func getProbeVal(item value.AnnotatedValue, probeExprs expression.Expressions,
	probeVals value.Values, context *Context) value.Value {

	var err error
	for i, pe := range probeExprs {
		probeVals[i], err = pe.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "Hash Table Probe Expression"))
			return nil
		}
	}

	if len(probeVals) == 1 {
		return probeVals[0]
	} else {
		return value.NewValue(probeVals)
	}
}

func (this *HashJoin) processItem(item value.AnnotatedValue, context *Context) bool {
	defer this.switchPhase(_EXECTIME)

	var err error
	var outVal interface{}
	ok := true
	matched := false

	probeVal := getProbeVal(item, this.plan.ProbeExprs(), this.probeVals, context)
	if probeVal == nil {
		item.Recycle()
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
			var joined value.AnnotatedValue
			match, ok, joined = processAnsiExec(item, right_item, this.plan.Onclause(),
				this.plan.BuildAliases(), this.ansiFlags, context, "join")
			if match && ok {
				matched = true
				ok = this.sendItem(joined)
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

	if this.plan.Outer() && !matched {
		return this.sendItem(item)
	}

	return true
}

func (this *HashJoin) afterItems(context *Context) {
	this.dropHashTable()
	this.plan.Onclause().ResetMemory(context)
}

func (this *HashJoin) dropHashTable() {
	if this.hashTab != nil {
		this.hashTab.Drop()
		this.hashTab = nil
	}
}

func (this *HashJoin) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~child"] = this.child
	})
	return json.Marshal(r)
}

func (this *HashJoin) SendStop() {
	this.baseSendStop()
	child := this.child
	if child != nil {
		child.SendStop()
	}
}

func (this *HashJoin) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
}
