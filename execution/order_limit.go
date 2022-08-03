//  Copyright 2014-Present Couchbase, Inc.
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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type OrderLimit struct {
	*Order
	offset           *Offset // offset is optional
	limit            *Limit  // limit must present
	numReturnedRows  int
	fallbackNum      int
	ignoreInput      bool
	fallback         bool
	numProcessedRows uint64
}

func NewOrderLimit(order *plan.Order, context *Context) *OrderLimit {
	var rv *OrderLimit
	if order.Offset() == nil {
		rv = &OrderLimit{
			Order:            NewOrder(order, context),
			offset:           nil,
			limit:            NewLimit(order.Limit(), context),
			numReturnedRows:  0,
			fallbackNum:      plan.OrderFallbackNum(),
			ignoreInput:      false,
			fallback:         false,
			numProcessedRows: 0,
		}
	} else {
		rv = &OrderLimit{
			Order:            NewOrder(order, context),
			offset:           NewOffset(order.Offset(), context),
			limit:            NewLimit(order.Limit(), context),
			numReturnedRows:  0,
			fallbackNum:      plan.OrderFallbackNum(),
			ignoreInput:      false,
			fallback:         false,
			numProcessedRows: 0,
		}
	}

	rv.output = rv
	return rv
}

func (this *OrderLimit) Copy() Operator {
	var rv *OrderLimit

	if this.offset == nil {
		rv = &OrderLimit{
			Order:            this.Order.Copy().(*Order),
			offset:           nil,
			limit:            this.limit.Copy().(*Limit),
			numReturnedRows:  this.numReturnedRows,
			ignoreInput:      this.ignoreInput,
			fallback:         this.fallback,
			numProcessedRows: this.numProcessedRows,
		}
	} else {
		rv = &OrderLimit{
			Order:            this.Order.Copy().(*Order),
			offset:           this.offset.Copy().(*Offset),
			limit:            this.limit.Copy().(*Limit),
			numReturnedRows:  this.numReturnedRows,
			ignoreInput:      this.ignoreInput,
			fallback:         this.fallback,
			numProcessedRows: this.numProcessedRows,
		}
	}
	return rv
}

func (this *OrderLimit) PlanOp() plan.Operator {
	return this.Order.plan
}

func (this *OrderLimit) RunOnce(context *Context, parent value.Value) {
	defer this.releaseValues()
	this.runConsumer(this, context, parent)
}

func (this *OrderLimit) beforeItems(context *Context, parent value.Value) bool {
	context.AddPhaseOperator(SORT)
	this.numReturnedRows = 0
	this.fallback = false
	this.numProcessedRows = 0
	this.setupTerms(context)
	res := true

	if this.offset != nil {
		// There is an offset in the query.
		res = this.offset.beforeItems(context, parent)
		if !res {
			return res
		}
		offset := this.offset.offset
		if offset > int64(this.fallbackNum) {
			// Fall back to the standard sort.
			this.fallback = true
		} else {
			this.numReturnedRows += int(offset)
		}
	}

	res = res && this.limit.beforeItems(context, parent)
	if !res {
		return res
	}
	limit := this.limit.limit
	this.ignoreInput = limit <= 0
	if !this.ignoreInput && !this.fallback && limit > int64(this.fallbackNum-this.numReturnedRows) {
		// Fallback to the standard sort.
		this.fallback = true
	}

	if !this.ignoreInput && !this.fallback {
		this.numReturnedRows += int(limit)
	}

	// Will ignore input rows if numReturnedRows is not positive.
	this.ignoreInput = this.ignoreInput || this.numReturnedRows <= 0

	// Allocate more space if necessary.
	if this.numReturnedRows > cap(this.values) {
		values := make(value.AnnotatedValues, len(this.values), this.numReturnedRows)
		copy(values, this.values)
		this.releaseValues()
		this.values = values
	}
	return res
}

func (this *OrderLimit) processItem(item value.AnnotatedValue, context *Context) bool {
	this.numProcessedRows++
	if this.fallback {
		return this.Order.processItem(item, context)
	}
	if this.ignoreInput {
		return true
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
		if this.offset != nil {
			this.offset.afterItems(context)
		}
		this.limit.afterItems(context)
	}()

	// Deal with the case no data item is needed at all:
	// when offset is too large.
	len := len(this.values)
	offset := int64(0)
	if this.offset != nil {
		offset = this.offset.offset
	}
	if offset >= int64(len) {
		this.values = this.values[0:0]
	}

	this.Order.afterItems(context)

	// Set the sort count to the number of processed rows.
	context.AddPhaseCount(SORT, this.numProcessedRows)
	context.SetSortCount(this.numProcessedRows)
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
	this.values = this.values[0:index:cap(this.values)]
	return item
}

func (this *OrderLimit) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *OrderLimit) SendAction(action opAction) {
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

func (this *OrderLimit) reopen(context *Context) bool {
	rv := this.Order.reopen(context)
	this.limit.baseReopen(context)
	if this.offset != nil {
		this.offset.reopen(context)
	}
	return rv
}

func (this *OrderLimit) Done() {
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
