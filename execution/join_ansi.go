//  Copyright (c) 2017 Couchbase, Inc.
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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

const (
	ANSI_REOPEN_CHILD   = 1 << iota // need to reopen child operator
	ANSI_ONCLAUSE_TRUE              // on-clause is TRUE
	ANSI_ONCLAUSE_FALSE             // on-clause is FALSE
)

type AnsiJoin struct {
	base
	plan      *plan.AnsiJoin
	child     Operator
	ansiFlags uint32
}

func NewAnsiJoin(plan *plan.AnsiJoin, context *Context, child Operator) *AnsiJoin {
	rv := &AnsiJoin{
		plan:  plan,
		child: child,
	}

	newBase(&rv.base, context)
	rv.trackChildren(1)
	rv.execPhase = ANSI_JOIN
	rv.output = rv
	return rv
}

func (this *AnsiJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAnsiJoin(this)
}

func (this *AnsiJoin) Copy() Operator {
	rv := &AnsiJoin{
		plan:  this.plan,
		child: this.child.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *AnsiJoin) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *AnsiJoin) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.child != nil, "ANSI JOIN has no child") {
		return false
	}
	if !context.assert(this.plan.Onclause() != nil, "ANSI JOIN does not have onclause") {
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
	}

	return true
}

func (this *AnsiJoin) processItem(item value.AnnotatedValue, context *Context) bool {
	defer this.switchPhase(_EXECTIME)

	if (this.ansiFlags & ANSI_REOPEN_CHILD) != 0 {
		if this.child != nil {
			this.child.SendStop()
			this.child.reopen(context)
		}
	} else {
		this.ansiFlags |= ANSI_REOPEN_CHILD
	}

	this.child.SetOutput(this.child)
	this.child.SetInput(nil)
	this.child.SetParent(this)
	this.child.SetStop(nil)

	go this.child.RunOnce(context, item)

	ok := true
	matched := false
	stopped := false
	n := 1

loop:
	for ok {
		right_item, child, cont := this.getItemChildrenOp(this.child)
		if cont {
			if right_item != nil {
				var match bool
				var joined value.AnnotatedValue
				match, ok, joined = processAnsiExec(item, right_item, this.plan.Onclause(),
					this.plan.Alias(), this.ansiFlags, context, "join")
				if ok && match {
					matched = true
					ok = this.sendItem(joined)
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
		notifyChildren(this.child)
		this.childrenWaitNoStop(n)
	}

	if stopped || !ok {
		return false
	}

	if this.plan.Outer() && !matched {
		return this.sendItem(item)
	}

	return true
}

func processAnsiExec(item value.AnnotatedValue, right_item value.AnnotatedValue,
	onclause expression.Expression, alias string, ansiFlags uint32, context *Context, op string) (
	bool, bool, value.AnnotatedValue) {

	var joined value.AnnotatedValue

	joined = item.Copy().(value.AnnotatedValue)

	// only interested in the value corresponding to "alias"
	val, ok := right_item.Field(alias)
	if !ok {
		context.Error(errors.NewExecutionInternalError(fmt.Sprintf("processAnsiExec: annotated value not found for %s", alias)))
		return false, false, nil
	}

	joined.SetField(alias, val)

	if op == "join" {
		covers := right_item.Covers()
		if covers != nil {
			for key, _ := range covers.Fields() {
				value, _ := covers.Field(key)
				joined.SetCover(key, value)
			}
		}
	}

	var match bool

	// evaluate ON-clause
	if (ansiFlags & ANSI_ONCLAUSE_TRUE) != 0 {
		match = true
	} else if (ansiFlags & ANSI_ONCLAUSE_FALSE) != 0 {
		match = false
	} else {
		result, err := onclause.Evaluate(joined, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "ON-clause"))
			return false, false, nil
		}

		if result.Truth() {
			match = true
		} else {
			match = false
		}
	}

	return match, true, joined
}

func (this *AnsiJoin) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~child"] = this.child
	})
	return json.Marshal(r)
}

func (this *AnsiJoin) SendStop() {
	this.baseSendStop()
	if this.child != nil {
		this.child.SendStop()
	}
}

func (this *AnsiJoin) reopen(context *Context) {
	this.baseReopen(context)
	this.ansiFlags &^= ANSI_REOPEN_CHILD
	if this.child != nil {
		this.child.reopen(context)
	}
}

func (this *AnsiJoin) Done() {
	this.baseDone()
	if this.child != nil {
		this.child.Done()
	}
	this.child = nil
}
