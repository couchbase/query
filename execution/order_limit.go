//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"container/heap"

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type OrderLimit struct {
	Order
	offset          Offset
	limit           Limit
	numReturnedRows int
	ignoreInput     bool
	fallback        bool
}

const _FALLBACK_NUM = 8192

func NewOrderLimit(plan *plan.Order) *OrderLimit {
	rv := &OrderLimit{
		Order:           *NewOrder(plan),
		offset:          *NewOffset(plan.Offset()), // offset is optional
		limit:           *NewLimit(plan.Limit()),   // limit must present
		numReturnedRows: 0,
		ignoreInput:     false,
		fallback:        false,
	}

	rv.output = rv
	return rv
}

func (this *OrderLimit) Copy() Operator {
	return &OrderLimit{
		Order: Order{
			base:   this.base.copy(),
			plan:   this.plan,
			values: _ORDER_POOL.Get(),
		},
		offset: Offset{
			base: this.offset.base.copy(),
			plan: this.offset.plan,
		},
		limit: Limit{
			base: this.limit.base.copy(),
			plan: this.limit.plan,
		},
		numReturnedRows: this.numReturnedRows,
		ignoreInput:     this.ignoreInput,
		fallback:        this.fallback,
	}
}

func (this *OrderLimit) RunOnce(context *Context, parent value.Value) {
	defer this.releaseValues()
	this.runConsumer(this, context, parent)
}

func (this *OrderLimit) beforeItems(context *Context, parent value.Value) bool {
	this.numReturnedRows = 0
	this.fallback = false
	this.setupTerms(context)
	res := true

	if this.offset.plan != nil {
		// There is an offset in the query.
		res = this.offset.beforeItems(context, parent)
		offset := this.offset.offset
		if offset > _FALLBACK_NUM {
			// Fall back to the standard sort.
			this.fallback = true
		} else {
			this.numReturnedRows += int(offset)
		}
	}

	res = res && this.limit.beforeItems(context, parent)
	limit := this.limit.limit
	this.ignoreInput = limit <= 0
	if !this.ignoreInput && !this.fallback && limit > int64(_FALLBACK_NUM-this.numReturnedRows) {
		// Fallback to the standard sort.
		this.fallback = true
	}

	if !this.ignoreInput && !this.fallback {
		this.numReturnedRows += int(limit)
	}

	// Will ignore input rows if numReturnedRows is not positive.
	this.ignoreInput = this.ignoreInput || this.numReturnedRows <= 0
	return res
}

func (this *OrderLimit) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.fallback {
		return this.Order.processItem(item, context)
	}
	if this.ignoreInput {
		return true
	}
	if len(this.values) == cap(this.values) {
		values := make(value.AnnotatedValues, len(this.values), len(this.values)<<1)
		copy(values, this.values)
		this.releaseValues()
		this.values = values
	}

	// Prune the item that does not need to enter the heap.
	if len(this.values) == this.numReturnedRows && !this.lessThan(item, this.values[0]) {
		return true
	}

	// Push the current item into the maximum heap.
	heap.Push(this, item)
	if len(this.values) > this.numReturnedRows {
		// Pop and discard the largest item out of the maximum heap.
		heap.Pop(this)
	}
	return true
}

func (this *OrderLimit) afterItems(context *Context) {
	defer func() {
		if this.offset.plan != nil {
			this.offset.afterItems(context)
		}
		this.limit.afterItems(context)
	}()

	// Deal with the case no data item is needed at all:
	// when offset is too large.
	len := len(this.values)
	offset := uint64(0)
	if this.offset.plan != nil {
		offset = this.offset.offset
	}
	if offset >= uint64(len) {
		this.values = this.values[0:0]
	}

	this.Order.afterItems(context)
}

func (this *OrderLimit) Less(i, j int) bool {
	// Since the heap is a maximum heap, it needs to returns the reversal of Less in Order.
	return this.Order.Less(j, i)
}

func (this *OrderLimit) Push(item interface{}) {
	this.values = append(this.values, item.(value.AnnotatedValue))
}

func (this *OrderLimit) Pop() interface{} {
	index := len(this.values) - 1
	item := this.values[index]
	this.values = this.values[0:index]
	return item
}
