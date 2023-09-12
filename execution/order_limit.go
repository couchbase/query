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
	"github.com/couchbase/query/value"
)

type OrderLimit struct {
	*Order
	offset           *Offset // offset is optional
	limit            *Limit  // limit must present
	fallbackNum      int
	ignoreInput      bool
	numProcessedRows uint64
}

func NewOrderLimit(order *plan.Order, context *Context) *OrderLimit {
	var rv *OrderLimit
	if order.Offset() == nil {
		rv = &OrderLimit{
			Order:            NewOrder(order, context),
			offset:           nil,
			limit:            NewLimit(order.Limit(), context),
			fallbackNum:      plan.OrderFallbackNum(),
			ignoreInput:      false,
			numProcessedRows: 0,
		}
	} else {
		rv = &OrderLimit{
			Order:            NewOrder(order, context),
			offset:           NewOffset(order.Offset(), context),
			limit:            NewLimit(order.Limit(), context),
			fallbackNum:      plan.OrderFallbackNum(),
			ignoreInput:      false,
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
			ignoreInput:      this.ignoreInput,
			numProcessedRows: this.numProcessedRows,
		}
	} else {
		rv = &OrderLimit{
			Order:            this.Order.Copy().(*Order),
			offset:           this.offset.Copy().(*Offset),
			limit:            this.limit.Copy().(*Limit),
			ignoreInput:      this.ignoreInput,
			numProcessedRows: this.numProcessedRows,
		}
	}
	return rv
}

func (this *OrderLimit) PlanOp() plan.Operator {
	return this.Order.plan
}

func (this *OrderLimit) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, this.releaseValues)
}

func (this *OrderLimit) beforeItems(context *Context, parent value.Value) bool {
	this.Order.setupTerms(context)
	context.AddPhaseOperator(SORT)
	this.numProcessedRows = 0
	this.setupTerms(context)

	heapSize := 0
	if this.offset != nil {
		// There is an offset in the query.
		if !this.offset.beforeItems(context, parent) {
			return false
		}
		heapSize += int(this.offset.offset)
	}

	if !this.limit.beforeItems(context, parent) {
		return false
	}
	heapSize += int(this.limit.limit)

	this.ignoreInput = heapSize <= 0

	if !this.ignoreInput && heapSize < this.fallbackNum {
		this.values.SetHeapSize(heapSize)
	} else {
		this.values.SetHeapSize(-1)
	}
	return true
}

func (this *OrderLimit) processItem(item value.AnnotatedValue, context *Context) bool {
	this.numProcessedRows++
	if this.ignoreInput {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return true
	}
	return this.Order.processItem(item, context)
}

func (this *OrderLimit) afterItems(context *Context) {
	defer func() {
		if this.offset != nil {
			this.offset.afterItems(context)
		}
		this.limit.afterItems(context)
	}()

	offset := int64(0)
	if this.offset != nil {
		offset = this.offset.offset
	}

	if offset < int64(this.values.Length()) {
		this.Order.afterItems(context)
	} else {
		this.Order.values.Truncate(
			func(v value.AnnotatedValue) {
				if context.UseRequestQuota() {
					context.ReleaseValueSize(v.Size())
				}
				v.Recycle()
			})
	}

	// Set the sort count to the number of processed rows.
	context.AddPhaseCount(SORT, this.numProcessedRows)
	context.SetSortCount(this.numProcessedRows)
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
