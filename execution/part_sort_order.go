//  Copyright 2022-Present Couchbase, Inc.
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

	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type PartSortOrder struct {
	*Order
	offset               *Offset
	limit                *Limit
	partialSortTermCount int
	remainingOffset      uint64
	remainingLimit       uint64
	sortCount            uint64
	numProcessedRows     uint64
	last                 value.AnnotatedValue
	fallbackNum          int
}

func NewPartSortOrder(order *plan.Order, context *Context) *PartSortOrder {
	var rv *PartSortOrder

	rv = &PartSortOrder{
		Order:                NewOrder(order, context),
		partialSortTermCount: order.PartialSortTermCount(),
		fallbackNum:          plan.OrderFallbackNum(),
	}

	if order.Offset() != nil {
		rv.offset = NewOffset(order.Offset(), context)
	}
	if order.Limit() != nil {
		rv.limit = NewLimit(order.Limit(), context)
	}

	rv.output = rv
	return rv
}

func (this *PartSortOrder) Copy() Operator {
	rv := &PartSortOrder{
		Order:                this.Order.Copy().(*Order),
		partialSortTermCount: this.partialSortTermCount,
		fallbackNum:          this.fallbackNum,
	}

	if this.offset != nil {
		rv.offset = this.offset.Copy().(*Offset)
	}
	if this.limit != nil {
		rv.limit = this.limit.Copy().(*Limit)
	}
	return rv
}

func (this *PartSortOrder) PlanOp() plan.Operator {
	return this.Order.plan
}

func (this *PartSortOrder) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, this.releaseValues)
}

func (this *PartSortOrder) beforeItems(context *Context, parent value.Value) bool {
	this.Order.setupTerms(context)
	context.AddPhaseOperator(SORT)
	if this.offset != nil && !this.offset.beforeItems(context, parent) {
		return false
	}
	if this.limit != nil && !this.limit.beforeItems(context, parent) {
		return false
	}
	this.remainingOffset = 0
	if this.offset != nil {
		this.remainingOffset = uint64(this.offset.offset)
	}
	this.remainingLimit = 0
	if this.limit != nil {
		this.remainingLimit = uint64(this.limit.limit)
	}
	heapSize := int(this.remainingOffset + this.remainingLimit)
	if this.fallbackNum < heapSize {
		heapSize = 0
	}
	this.values.SetHeapSize(heapSize)
	this.sortCount = 0
	this.numProcessedRows = 0
	logging.Debuga(func() string {
		return fmt.Sprintf("PartSortOrder: terms: %v, off: %v, lim: (%v) %v, heap: %v",
			this.partialSortTermCount, this.remainingOffset, this.limit != nil, this.remainingLimit, heapSize)
	})
	return true
}

func (this *PartSortOrder) processItem(item value.AnnotatedValue, context *Context) bool {
	this.numProcessedRows++
	if this.Order.plan.ClipValues() {
		this.Order.makeMinimal(item, context)
	}

	if this.last != nil && this.values.Length() > 0 && !this.samePartialSortValues(item, this.last) {
		if this.remainingOffset >= uint64(this.values.Length()) {
			// discard all accumulated values thus far as they'll not be part of the results
			this.remainingOffset -= uint64(this.values.Length())
			this.values.Truncate(
				func(v value.AnnotatedValue) {
					if context.UseRequestQuota() {
						context.ReleaseValueSize(v.Size())
					}
					v.Recycle()
				})
			heapSize := int(this.remainingOffset + this.remainingLimit)
			if this.fallbackNum < heapSize {
				heapSize = 0
			}
			this.values.ShrinkHeapSize(heapSize)
		} else {
			// sort and stream what we have, potentially stopping if we hit the limit
			if !this.sortAndStream(context) {
				if context.UseRequestQuota() {
					context.ReleaseValueSize(item.Size())
				}
				item.Recycle()
				return false
			}
		}
	}

	if this.last != nil {
		this.last.Recycle()
	}
	item.Track()
	err := this.values.Append(item)
	if err == nil {
		this.last = item
		return true
	}
	this.last = nil
	context.Error(err)
	return false
}

func (this *PartSortOrder) afterItems(context *Context) {
	defer this.releaseValues()
	defer func() {
		this.terms = nil
		context.SetSortCount(this.sortCount)
		context.AddPhaseCount(SORT, this.numProcessedRows)
	}()

	if this.stopped {
		return
	}

	if this.values.Length() > 0 {
		this.setupTerms(context)
		this.sortAndStream(context)
	}
}

// called only once we've reached the point n the offset of returning items; caller handles prior to this
func (this *PartSortOrder) sortAndStream(context *Context) bool {

	this.sortCount += uint64(this.values.Length())

	rv := true
	if this.limit == nil || this.remainingLimit > 0 {
		err := this.values.Foreach(func(av value.AnnotatedValue) bool {
			if this.remainingOffset == 0 {
				if !this.sendItem(av) {
					rv = false
					return false
				}
				if this.limit != nil {
					this.remainingLimit--
				}
			} else {
				if context.UseRequestQuota() {
					context.ReleaseValueSize(av.Size())
				}
				this.remainingOffset--
				av.Recycle()
			}
			if this.limit != nil && this.remainingLimit == 0 {
				return false
			}
			return true
		})
		if err != nil {
			context.Error(err)
			return false
		}
	}
	logging.Debuga(func() string { return this.values.Stats() })
	this.values.Truncate(nil)
	return rv && (this.limit == nil || this.remainingLimit != 0)
}

func (this *PartSortOrder) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *PartSortOrder) reopen(context *Context) bool {
	rv := this.Order.reopen(context)
	if this.limit != nil {
		this.limit.reopen(context)
	}
	if this.offset != nil {
		this.offset.reopen(context)
	}
	return rv
}

func (this *PartSortOrder) SendAction(action opAction) {
	this.Order.SendAction(action)
	limit := this.limit
	if limit != nil {
		limit.SendAction(action)
	}
	offset := this.offset
	if offset != nil {
		offset.SendAction(action)
	}
}

func (this *PartSortOrder) Done() {
	this.Order.Done()
	limit := this.limit
	if limit != nil {
		this.limit = nil
		limit.Done()
	}
	offset := this.offset
	if offset != nil {
		this.offset = nil
		offset.Done()
	}
}

// we are only completing the sort on partially sorted blocks so no need to check the already sorted terms
func (this *PartSortOrder) remainingTermsLessThan(v1 value.AnnotatedValue, v2 value.AnnotatedValue) bool {
	var ev1, ev2 value.Value
	var c int
	var e error

	remTerms := this.plan.Terms()[this.partialSortTermCount:]
	for i, term := range remTerms {
		i += this.partialSortTermCount
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

func (this *PartSortOrder) samePartialSortValues(v1 value.AnnotatedValue, v2 value.AnnotatedValue) bool {
	var ev1, ev2 value.Value
	var e error

	for i, term := range this.plan.Terms() {
		if i == this.partialSortTermCount {
			return true
		}
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
			if ev1.Collate(ev2) != 0 {
				return false
			}
		} else {
			return false
		}
	}

	return false
}
