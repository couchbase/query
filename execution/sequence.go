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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Sequence struct {
	base
	isParallel bool
	plan       *plan.Sequence
	children   []Operator
}

var _SEQUENCE_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_SEQUENCE_OP_POOL, func() interface{} {
		return &Sequence{}
	})
}

func NewSequence(plan *plan.Sequence, context *Context, children ...Operator) *Sequence {
	return NewParallelSequence(plan, false, context, children...)
}

func NewParallelSequence(plan *plan.Sequence, isParallel bool, context *Context, children ...Operator) *Sequence {
	rv := _SEQUENCE_OP_POOL.Get().(*Sequence)
	rv.plan = plan
	rv.isParallel = isParallel
	rv.children = children

	// allocate value exchanges for serialized children if required
	// if not the first operator and the sender has already an operator
	// we'll just use that, since the sender has no use for it
	prevBase := children[0].getBase()
	for i := 1; i < len(children); i++ {
		thisBase := children[i].getBase()
		if thisBase.IsSerializable() && !prevBase.IsParallel() {
			prevBase.exchangeMove(thisBase)
		}
		prevBase = thisBase
	}

	// we'll even use the last child's value exchange and save allocating
	// an unused one for ourselves
	newRedirectBase(&rv.base, context)
	rv.base.setInline()
	prevBase.exchangeMove(&rv.base)
	rv.output = rv
	return rv
}

func (this *Sequence) SetParallel() {
	this.isParallel = true
}

func (this *Sequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSequence(this)
}

func (this *Sequence) Copy() Operator {
	return this.CustomizedCopy(false)
}

// forTimingAccrual: set this for special handling when making a copy of the Operator
// currently set during the accrual of execution trees for subquery timings
func (this *Sequence) CustomizedCopy(forTimingAccrual bool) Operator {
	rv := _SEQUENCE_OP_POOL.Get().(*Sequence)
	children := _SEQUENCE_POOL.Get()

	for _, child := range this.children {

		custom := false
		var copy Operator

		if forTimingAccrual {
			if cOp, ok := child.(interface{ CustomizedCopy(bool) Operator }); ok {
				copy = cOp.CustomizedCopy(forTimingAccrual)
				custom = true
			}
		}

		if !custom {
			copy = child.Copy()
		}

		children = append(children, copy)
	}

	rv.plan = this.plan
	rv.isParallel = this.isParallel
	rv.children = children
	this.base.copy(&rv.base)
	return rv
}

func (this *Sequence) PlanOp() plan.Operator {
	return this.plan
}

func (this *Sequence) IsParallel() bool {
	return this.isParallel
}

func (this *Sequence) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		this.SetKeepAlive(1, context)

		n := len(this.children)
		if !active || !context.assert(n > 0, "Sequence has no children") {
			this.notify()
			this.fail(context)
			return
		}

		curr := this.children[0]
		curr.SetInput(this.input)
		curr.SetStop(this.stop)

		// Define all Inputs and Outputs
		var next Operator
		for i := 1; i < n; i++ {
			next = this.children[i]

			// run the consumer inline, if feasible
			if next.IsSerializable() && !curr.IsParallel() {
				next.SetInput(curr)
				curr.SerializeOutput(next, context)
			} else {
				curr.SetOutput(curr)
				next.SetInput(curr.Output())
			}
			next.SetStop(curr)
			curr = next
		}

		next.SetOutput(this.output)
		next.SetParent(this)
		this.stashOutput()

		// Run last child
		this.fork(next, context, parent)
	})
}

func (this *Sequence) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["~children"] = this.children
	})
	return json.Marshal(r)
}

func (this *Sequence) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Sequence)
	childrenAccrueTimes(this.children, copy.children)
}

func (this *Sequence) SendAction(action opAction) {
	this.baseSendAction(action)
	for _, child := range this.children {
		if child != nil {
			child.SendAction(action)
		}
	}
}

func (this *Sequence) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv {
		for _, child := range this.children {
			if !child.reopen(context) {
				return false
			}
		}
	}
	return rv
}

func (this *Sequence) Done() {
	this.baseDone()
	for i := len(this.children) - 1; i >= 0; i-- {
		child := this.children[i]
		this.children[i] = nil
		child.Done()
	}
	_SEQUENCE_POOL.Put(this.children)
	this.children = nil
	if this.isComplete() {
		_SEQUENCE_OP_POOL.Put(this)
	}
}

var _SEQUENCE_POOL = NewOperatorPool(32)
