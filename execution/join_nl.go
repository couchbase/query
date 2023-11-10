//  Copyright 2017-Present Couchbase, Inc.
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

const (
	ANSI_REOPEN_CHILD   = 1 << iota // need to reopen child operator
	ANSI_ONCLAUSE_TRUE              // on-clause is TRUE
	ANSI_ONCLAUSE_FALSE             // on-clause is FALSE
)

type NLJoin struct {
	base
	plan      *plan.NLJoin
	child     Operator
	aliasMap  map[string]string
	ansiFlags uint32
}

func NewNLJoin(plan *plan.NLJoin, context *Context, child Operator, aliasMap map[string]string) *NLJoin {
	rv := &NLJoin{
		plan:     plan,
		child:    child,
		aliasMap: aliasMap,
	}

	newBase(&rv.base, context)
	rv.trackChildren(1)
	rv.execPhase = NL_JOIN
	rv.output = rv
	return rv
}

func (this *NLJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNLJoin(this)
}

func (this *NLJoin) Copy() Operator {
	rv := &NLJoin{
		plan:     this.plan,
		child:    this.child.Copy(),
		aliasMap: this.aliasMap,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *NLJoin) PlanOp() plan.Operator {
	return this.plan
}

func (this *NLJoin) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *NLJoin) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.child != nil, "Nested Loop Join has no child") {
		return false
	}

	// check for constant TRUE or FALSE onclause
	onclause := this.plan.Onclause()
	if onclause != nil {
		cpred := onclause.Value()
		if cpred != nil {
			if cpred.Truth() {
				this.ansiFlags |= ANSI_ONCLAUSE_TRUE
			} else {
				this.ansiFlags |= ANSI_ONCLAUSE_FALSE
			}
		} else {
			onclause.EnableInlistHash(&this.operatorCtx)
			SetSearchInfo(this.aliasMap, parent, &this.operatorCtx, onclause)
		}
	} else {
		// for comma-separated join, treat it as having a TRUE onclause
		this.ansiFlags |= ANSI_ONCLAUSE_TRUE
	}

	filter := this.plan.Filter()
	if filter != nil {
		filter.EnableInlistHash(&this.operatorCtx)
	}

	return true
}

func (this *NLJoin) processItem(item value.AnnotatedValue, context *Context) bool {
	defer this.switchPhase(_EXECTIME)

	if (this.ansiFlags&ANSI_REOPEN_CHILD) != 0 && this.child != nil && !this.child.reopen(context) {

		// If the reopen failed, we should propagate the stop signal to the inner scan again
		// to terminate any operator that we had successfully restarted
		this.child.SendAction(_ACTION_STOP)
		return false
	}

	this.child.SetOutput(this.child)
	this.child.SetInput(nil)
	this.child.SetParent(this)
	this.child.SetStop(nil)

	this.fork(this.child, context, item)
	this.ansiFlags |= ANSI_REOPEN_CHILD

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
				aliases := []string{this.plan.Alias()}
				match, ok, joined = processAnsiExec(item, right_item, this.plan.Onclause(),
					aliases, this.ansiFlags, &this.operatorCtx, "join")
				if ok && match {
					matched = true
					ok = this.checkSendItem(joined, func() uint64 {
						return joined.Size()
					}, true, this.plan.Filter(), context)
				} else if joined != nil {
					joined.Recycle()
				}

				// TODO break out and child.SendAction(_ACTION_STOP) here for semin-scans
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

	// There is no need to terminate the inner scan under normal completion
	if stopped || !ok {
		if n > 0 {
			this.child.SendAction(_ACTION_STOP)
			this.childrenWaitNoStop(this.child)
		}

		return false
	}

	if this.plan.Outer() && !matched {
		return this.checkSendItem(item, func() uint64 {
			return 0
		}, false, this.plan.Filter(), context)
	} else if context.UseRequestQuota() {
		context.ReleaseValueSize(item.Size())
	}
	// TODO Recycle

	return true
}

func (this *NLJoin) afterItems(context *Context) {
	if (this.ansiFlags & (ANSI_ONCLAUSE_TRUE | ANSI_ONCLAUSE_FALSE)) == 0 {
		onclause := this.plan.Onclause()
		if onclause != nil {
			onclause.ResetMemory(&this.operatorCtx)
		}
	}
	filter := this.plan.Filter()
	if filter != nil {
		filter.ResetMemory(&this.operatorCtx)
	}
}

func processAnsiExec(item value.AnnotatedValue, right_item value.AnnotatedValue,
	onclause expression.Expression, aliases []string, ansiFlags uint32, context *opContext, op string) (
	bool, bool, value.AnnotatedValue) {

	var joined value.AnnotatedValue

	joined = item.Copy().(value.AnnotatedValue)

	// only interested in the value corresponding to "aliases"
	for _, alias := range aliases {
		val, ok := right_item.Field(alias)
		if !ok {
			// it's possible to have a MISSING value for an alias, e.g.,
			// in case of HASH JOIN, when there are multiple aliases on the
			// build side, value could be MISSING for an alias, if right_item
			// contains the result of an outer join;
			// or if a MISSING value is specified explicitly in an expression term
			continue
		}

		joined.SetField(alias, val)
	}

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

func (this *NLJoin) checkSendItem(av value.AnnotatedValue, quotaFunc func() uint64, recycle bool, filter expression.Expression, context *Context) bool {
	if filter != nil {
		result, err := filter.Evaluate(av, &this.operatorCtx)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "nested-loop join filter"))
			if recycle {
				av.Recycle()
			}
			return false
		}
		if !result.Truth() {
			if recycle {
				av.Recycle()
			}
			return true
		}
	}
	if context.UseRequestQuota() && context.TrackValueSize(quotaFunc()) {
		context.Error(errors.NewMemoryQuotaExceededError())
		if recycle {
			av.Recycle()
		}
		return false

	}
	return this.sendItem(av)
}

func (this *NLJoin) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~child"] = this.child
	})
	return json.Marshal(r)
}

func (this *NLJoin) SendAction(action opAction) {
	this.baseSendAction(action)
	child := this.child
	if child != nil {
		child.SendAction(action)
	}
}

func (this *NLJoin) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.ansiFlags &^= ANSI_REOPEN_CHILD
	if rv && this.child != nil {
		this.child.reopen(context)
	}
	return rv
}

func (this *NLJoin) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
}
