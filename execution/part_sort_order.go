//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"container/heap"
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/sort"
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
	useHeap              bool
}

func NewPartSortOrder(order *plan.Order, context *Context) *PartSortOrder {
	var rv *PartSortOrder

	rv = &PartSortOrder{
		Order:                NewOrder(order, context),
		partialSortTermCount: order.PartialSortTermCount(),
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
	defer this.releaseValues()
	this.runConsumer(this, context, parent)
}

func (this *PartSortOrder) beforeItems(context *Context, parent value.Value) bool {
	context.AddPhaseOperator(SORT)
	this.setupTerms(context)
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
	this.useHeap = false
	if this.limit != nil {
		this.remainingLimit = uint64(this.limit.limit)
		this.useHeap = this.remainingOffset+this.remainingLimit < uint64(plan.OrderFallbackNum())
	}
	this.sortCount = 0
	this.numProcessedRows = 0
	logging.Debuga(func() string {
		return fmt.Sprintf("PartSortOrder: terms: %v, off: %v, lim: (%v) %v, heap: %v",
			this.partialSortTermCount, this.remainingOffset, this.limit != nil, this.remainingLimit, this.useHeap)
	})
	return true
}

func (this *PartSortOrder) processItem(item value.AnnotatedValue, context *Context) bool {
	this.numProcessedRows++

	if len(this.values) > 0 && !this.samePartialSortValues(item, this.values[len(this.values)-1]) {
		if this.remainingOffset >= uint64(len(this.values)) {
			// discard all accumulated values thus far as they'll not be part of the results
			this.remainingOffset -= uint64(len(this.values))
			this.releaseValuesSubset(context, 0, len(this.values))
			this.values = this.values[0:0:cap(this.values)]
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

	if len(this.values) == cap(this.values) {
		values := make(value.AnnotatedValues, len(this.values), len(this.values)<<1)
		copy(values, this.values)
		this.releaseValues()
		this.values = values
	}

	if this.useHeap {
		heap.Push(this, item)
		if uint64(len(this.values)) > this.remainingOffset+this.remainingLimit {
			heap.Pop(this)
		}
	} else {
		this.values = append(this.values, item)
	}

	return true
}

// this handles accounting for values that are not going to make it past this operator
func (this *PartSortOrder) releaseValuesSubset(context *Context, from int, to int) {
	useQuota := context.UseRequestQuota()
	for i := from; i < to && len(this.values) > i; i++ {
		if useQuota {
			context.ReleaseValueSize(this.values[i].Size())
		}
		this.values[i].Recycle()
	}
}

func (this *PartSortOrder) afterItems(context *Context) {
	defer this.releaseValues()
	defer func() {
		this.context = nil
		this.terms = nil
		context.SetSortCount(this.sortCount)
		context.AddPhaseCount(SORT, this.numProcessedRows)
	}()

	if this.stopped {
		return
	}

	if len(this.values) > 0 {
		this.setupTerms(context)
		this.sortAndStream(context)
	}
}

// called only once we've reached the point n the offset of returning items; caller handles prior to this
func (this *PartSortOrder) sortAndStream(context *Context) bool {

	this.sortCount += uint64(len(this.values))
	// also needed for heap; doesn't have to be all terms as we never have mixed pre-sorted term values
	sort.Sort(this)

	if this.useHeap {
		return this.sortAndStreamHeap(context)
	}

	// there is no following offset operator so we must fully implement it here
	if this.remainingOffset > 0 && uint64(len(this.values)) > this.remainingOffset {
		this.releaseValuesSubset(context, 0, int(this.remainingOffset))
		this.values = this.values[this.remainingOffset:]
		this.remainingOffset = 0
	}

	if this.limit != nil {
		if this.remainingLimit < uint64(len(this.values)) {
			this.releaseValuesSubset(context, int(this.remainingLimit), len(this.values))
			this.values = this.values[:this.remainingLimit]
		}
		this.remainingLimit -= uint64(len(this.values))
	}

	earlyOrder := this.plan.IsEarlyOrder()
	for n, av := range this.values {
		if earlyOrder {
			this.resetCachedValues(av)
		}
		if !this.sendItem(av) {
			this.releaseValuesSubset(context, n, len(this.values))
			return false
		}
	}

	this.values = this.values[:0]
	return this.limit == nil || this.remainingLimit != 0
}

// return the appropriate portion of the values list in reverse heap order (which is desired order)
func (this *PartSortOrder) sortAndStreamHeap(context *Context) bool {

	if this.remainingOffset > 0 && uint64(len(this.values)) > this.remainingOffset {
		this.releaseValuesSubset(context, len(this.values)-int(this.remainingOffset), len(this.values))
		this.values = this.values[:uint64(len(this.values))-this.remainingOffset]
		this.remainingOffset = 0
	}

	to := 0
	if this.limit != nil {
		if this.remainingLimit < uint64(len(this.values)) {
			this.releaseValuesSubset(context, 0, int(this.remainingLimit))
			to = int(this.remainingLimit)
			this.remainingLimit = 0
		} else {
			this.remainingLimit -= uint64(len(this.values))
		}
	}

	if to < len(this.values) {
		for i := len(this.values) - 1; i >= to; i-- {
			if !this.sendItem(this.values[i]) {
				this.releaseValuesSubset(context, to, i)
				return false
			}
		}
	}
	this.values = this.values[:0]
	return this.limit == nil || this.remainingLimit != 0
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

func (this *PartSortOrder) Less(i, j int) bool {
	if this.useHeap {
		i, j = j, i // invert for maximum heap
	}
	return this.remainingTermsLessThan(this.values[i], this.values[j])
}

func (this *PartSortOrder) Push(item interface{}) {
	this.values = append(this.values, item.(value.AnnotatedValue))
}

func (this *PartSortOrder) Pop() interface{} {
	index := len(this.values) - 1
	item := this.values[index]
	this.values = this.values[0:index:cap(this.values)]
	return item
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

		ev1, e = getOriginalCachedValue(v1, term.Expression(), s, this.context)
		if e != nil {
			return false
		}

		ev2, e = getOriginalCachedValue(v2, term.Expression(), s, this.context)
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

		ev1, e = getOriginalCachedValue(v1, term.Expression(), s, this.context)
		if e != nil {
			return false
		}

		ev2, e = getOriginalCachedValue(v2, term.Expression(), s, this.context)
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
