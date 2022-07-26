//  Copyright 2016-Present Couchbase, Inc.
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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type IndexCountScan struct {
	base
	plan *plan.IndexCountScan
}

func NewIndexCountScan(plan *plan.IndexCountScan, context *Context) *IndexCountScan {
	rv := &IndexCountScan{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexCountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountScan(this)
}

func (this *IndexCountScan) Copy() Operator {
	rv := &IndexCountScan{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexCountScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexCountScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_COUNT, context)
		defer this.cleanup(context)
		if !active {
			return
		}

		spans := this.plan.Spans()
		n := len(spans)

		// ideally we should use this.itemChannel
		// for this to work properly, this channel must be never closed
		// ideally we should stop the scanCount goroutines
		countChannel := make(value.ValueChannel, n)

		keyspaceTerm := this.plan.Term()
		scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Path().Bucket())

		var count int64

		for _, span := range spans {
			go func() {
				primeStack()
				this.scanCount(span, scanVector, countChannel, context)
			}()
		}

		for n > 0 {
			val, ok := this.getItemValue(countChannel)
			if !ok {
				return
			}

			subcount := int64(0)
			if val.Type() == value.NUMBER {
				subcount = val.(value.NumberValue).Int64()
			}

			// current policy is to only count 'in' documents
			// from operators, not kv
			// add this.addInDocs(1) if this changes
			// this could be used for diagnostic purposes:
			// if docsIn != spans, something has gone wrong
			// somewhere
			count += subcount
			n--
		}

		av := value.NewAnnotatedValue(count)
		av.InheritCovers(parent)
		this.sendItem(av)
	})
}

func (this *IndexCountScan) scanCount(span *plan.Span, scanVector timestamp.Vector, countChannel value.ValueChannel, context *Context) {
	dspan, empty, err := evalSpan(span, nil, context)

	var count int64
	if err == nil && !empty {
		count, err = this.plan.Index().Count(dspan, context.ScanConsistency(), scanVector)
	}

	if err != nil {
		context.Error(errors.NewEvaluationError(err, "scanCount()"))
	}

	countChannel <- value.NewValue(count)
}

func (this *IndexCountScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
