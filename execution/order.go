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

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/system"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type orderTerm struct {
	term       string
	descending bool
	nullsLast  bool
}

type Order struct {
	base
	plan    *plan.Order
	values  *value.AnnotatedArray
	terms   []orderTerm
	spilled bool
}

const _ORDER_CAP = 1024
const _MIN_SIZE = 128 * util.MiB

var _ORDER_POOL = value.NewAnnotatedPool(_ORDER_CAP)

func NewOrder(plan *plan.Order, context *Context, less func(value.AnnotatedValue, value.AnnotatedValue) bool) *Order {
	rv := &Order{
		plan: plan,
	}
	// here only setting function to test for spilling when quota is in effect
	var shouldSpill func(uint64, uint64) bool
	if plan.CanSpill() && context.IsFeatureEnabled(util.N1QL_SPILL_TO_DISK) {
		if context.UseRequestQuota() && context.MemoryQuota() > 0 {
			shouldSpill = func(c uint64, n uint64) bool {
				if (c + n) <= context.ProducerThrottleQuota() {
					return false
				}
				f := util.RoundPlaces(system.GetMemActualFreePercent(), 1)
				if f < 0.1 {
					f = 0.1
				} else if f > 0.7 {
					f = 0.7
				}
				res := context.CurrentQuotaUsage() > f
				if res && !rv.spilled {
					rv.spilled = true
					accounting.UpdateCounter(accounting.SPILLS_ORDER)
				}
				return res
			}
		} else {
			maxSize := context.AvailableMemory()
			if maxSize > 0 {
				maxSize = uint64(float64(maxSize) / float64(util.NumCPU()) * 0.2) // 20% of per CPU free memory
			}
			if maxSize < _MIN_SIZE {
				maxSize = _MIN_SIZE
			}
			shouldSpill = func(c uint64, n uint64) bool {
				res := (c + n) > maxSize
				if res && !rv.spilled {
					rv.spilled = true
					accounting.UpdateCounter(accounting.SPILLS_ORDER)
				}
				return res
			}
		}
	}
	acquire := func(size int) value.AnnotatedValues {
		if size <= _ORDER_POOL.Size() {
			return _ORDER_POOL.Get()
		}
		return make(value.AnnotatedValues, 0, size)
	}
	trackMem := func(size int64) error {
		if context.UseRequestQuota() {
			if size < 0 {
				context.ReleaseValueSize(uint64(-size))
			} else {
				if err := context.TrackValueSize(uint64(size)); err != nil {
					context.Fatal(err)
					return err
				}
			}
		}
		return nil
	}
	if less == nil {
		less = rv.lessThan
	}
	rv.values = value.NewAnnotatedArray(
		acquire,
		func(p value.AnnotatedValues) { _ORDER_POOL.Put(p) },
		shouldSpill,
		trackMem,
		less,
		!plan.ClipValues(),
	)

	newBase(&rv.base, context)
	rv.execPhase = SORT
	rv.output = rv
	return rv
}

func (this *Order) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOrder(this)
}

func (this *Order) Copy() Operator {
	rv := &Order{
		plan:   this.plan,
		values: this.values.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Order) PlanOp() plan.Operator {
	return this.plan
}

func (this *Order) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, this.releaseValues)
}

func (this *Order) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.plan.ClipValues() {
		this.makeMinimal(item, context)
	}
	err := this.values.Append(item)
	if err != nil && !this.stopped && this.isRunning() {
		context.Error(err)
	}
	return err == nil
}

func (this *Order) setupTerms(item value.Value, context *Context) {
	if this.terms == nil {
		this.terms = make([]orderTerm, len(this.plan.Terms()))
		for i, term := range this.plan.Terms() {
			this.terms[i].term = term.Expression().String()
			this.terms[i].descending = term.Descending(item, &this.operatorCtx)
			this.terms[i].nullsLast = term.NullsLast(item, &this.operatorCtx)
		}
	}
}

func (this *Order) beforeItems(context *Context, item value.Value) bool {
	this.setupTerms(item, context)
	return true
}

func (this *Order) afterItems(context *Context) {
	// MB-25901 don't sort if we have been stopped
	if !this.stopped {
		context.SetSortCount(uint64(this.values.Length()))
		context.AddPhaseCount(SORT, uint64(this.values.Length()))

		earlyOrder := this.plan.IsEarlyOrder()
		err := this.values.Foreach(func(av value.AnnotatedValue) bool {
			if earlyOrder {
				this.resetCachedValues(av)
			}
			return this.sendItem(av)
		})
		if err != nil && !this.stopped && this.isRunning() {
			context.Error(err)
		}
		logging.Debuga(func() string { return this.values.Stats() })
	}
	this.releaseValues()
}

func (this *Order) releaseValues() {
	this.terms = nil
	this.values.Release()
}

func (this *Order) resetCachedValues(av value.AnnotatedValue) {
	for _, term := range this.terms {
		av.RemoveAttachment(getAttachmentIndexFor(av, term.term))
	}
}

func (this *Order) makeMinimal(item value.AnnotatedValue, context *Context) {
	var sz uint64
	useQuota := context.UseRequestQuota()
	if useQuota {
		sz = item.Size()
	}
	origAtt := item.Attachments()
	item.ResetAttachments()
	if aggs, ok := origAtt[value.ATT_AGGREGATES]; ok && aggs != nil {
		item.SetAttachment(value.ATT_AGGREGATES, aggs)
	}
	for i, term := range this.plan.Terms() {
		_, err := getOriginalCachedValue(item, term.Expression(), this.terms[i].term, &this.operatorCtx)
		if err != nil {
			for k, v := range origAtt {
				item.SetAttachment(k, v)
			}
			return
		}
	}
	origAtt = nil
	item.ResetCovers(nil)
	item.ResetMeta()
	item.ResetOriginal()
	if useQuota {
		asz := item.RecalculateSize()
		if sz < asz {
			// we could end up with growth if the evaluated term values are larger than the removed fields
			if err := context.TrackValueSize(asz - sz); err != nil {
				context.Error(err)
				return
			}
		} else {
			context.ReleaseValueSize(sz - asz)
		}
	}
}

func (this *Order) lessThan(v1 value.AnnotatedValue, v2 value.AnnotatedValue) bool {
	var ev1, ev2 value.Value
	var c int
	var e error

	for i, term := range this.plan.Terms() {
		s := this.terms[i].term

		ev1, e = getOriginalCachedValue(v1, term.Expression(), s, &this.operatorCtx)
		if e != nil {
			return false
		}

		ev2, e = getOriginalCachedValue(v2, term.Expression(), s, &this.operatorCtx)
		if e != nil {
			return false
		}

		if (this.terms[i].descending && this.terms[i].nullsLast) ||
			(!this.terms[i].descending && !this.terms[i].nullsLast) ||
			((ev1.Type() <= value.NULL && ev2.Type() <= value.NULL) ||
				(ev1.Type() > value.NULL && ev2.Type() > value.NULL)) {
			c = ev1.Collate(ev2)
		} else {
			if ev1.Type() <= value.NULL && ev2.Type() > value.NULL {
				c = 1
			} else {
				c = -1
			}
		}

		if c == 0 {
			continue
		} else if this.terms[i].descending {
			return c > 0
		} else {
			return c < 0
		}
	}

	return false
}

func (this *Order) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Order) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.spilled = false
	return rv
}

func (this *Order) SendAction(action opAction) {
	this.baseSendAction(action)

	if action == _ACTION_STOP && this.values != nil {
		this.values.Stop()
	}
}
