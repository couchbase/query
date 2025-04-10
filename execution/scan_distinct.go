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

type DistinctScan struct {
	base
	plan *plan.DistinctScan
	scan Operator
	keys map[string]bool
}

func NewDistinctScan(plan *plan.DistinctScan, context *Context, scan Operator) *DistinctScan {
	rv := &DistinctScan{
		plan: plan,
		scan: scan,
	}

	newBase(&rv.base, context)
	rv.trackChildren(1)
	rv.output = rv
	return rv
}

func (this *DistinctScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDistinctScan(this)
}

func (this *DistinctScan) Copy() Operator {
	rv := &DistinctScan{
		plan: this.plan,
		scan: this.scan.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *DistinctScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *DistinctScan) Scan() Operator {
	return this.scan
}

func (this *DistinctScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.cleanup(context)
		if !active {
			return
		}

		defer func() {
			this.keys = nil
		}()

		pipelineCap := int(context.GetPipelineCap())
		if pipelineCap <= _STRING_BOOL_POOL.Size() {
			this.keys = _STRING_BOOL_POOL.Get()
			defer func() {
				_STRING_BOOL_POOL.Put(this.keys)
			}()
		} else {
			this.keys = make(map[string]bool, pipelineCap)
		}

		if !context.assert(this.scan != nil, "Distinct has no scan") {
			return
		}

		this.scan.SetParent(this)
		this.scan.SetOutput(this.scan)
		this.SetInput(this.scan)
		this.fork(this.scan, context, parent)

		limit := evalLimitOffset(this.plan.Limit(), parent, int64(-1), this.plan.Covering(), &this.operatorCtx)
		offset := evalLimitOffset(this.plan.Offset(), parent, int64(0), this.plan.Covering(), &this.operatorCtx)
		n := 1
		ok := true

	loop:
		for ok {
			item, child, cont := this.getItemChildren()
			if cont {
				if item != nil {
					this.addInDocs(1)
					ok = this.processKey(item, context, limit, offset)
				} else if child >= 0 {
					n--
				} else {
					break loop
				}
			} else {
				break loop
			}
		}

		// Await child scan
		if n > 0 {
			sendChildren(this.plan, this.scan)
			this.childrenWaitNoStop(this.scan)
		}
	})
}

func (this *DistinctScan) processKey(item value.AnnotatedValue,
	context *Context, limit, offset int64) bool {

	key, ok := this.getDocumentKey(item, context)
	if !ok {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return false
	}

	if _, ok = this.keys[key]; ok {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return true
	}

	this.keys[key] = true

	length := int64(len(this.keys))
	if offset > 0 && length <= offset {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return true
	}

	if limit > 0 && length > (limit+offset) {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return false
	}

	item.SetBit(this.bit)
	return this.sendItem(item)
}

func (this *DistinctScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	r["scan"] = this.scan
	return json.Marshal(r)
}

func (this *DistinctScan) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*DistinctScan)
	this.scan.accrueTimes(copy.scan)
}

func (this *DistinctScan) SendAction(action opAction) {
	this.baseSendAction(action)
	scan := this.scan
	if scan != nil {
		scan.SendAction(action)
	}
}

func (this *DistinctScan) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.scan != nil {
		rv = this.scan.reopen(context)
	}
	return rv
}

func (this *DistinctScan) Done() {
	this.baseDone()
	if this.scan != nil {
		scan := this.scan
		this.scan = nil
		scan.Done()
	}
}

var _STRING_BOOL_POOL = util.NewStringBoolPool(1024)
