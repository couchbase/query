//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"fmt"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Window AI_RERANK(docs, options, query) function.
It reranks the documents collected over the window partition using an external AI API.

Arguments:
  - docs    (operand[0]): the expression whose values are collected into the input array.
  - options (operand[1]): a static object expression – { uri, model, cred_id, header }.
  - query   (operand[2]): a string expression – the rerank query text.

The action is implicitly "rerank"; it must not be specified in the options object.
ai_compute(docs, {action:"rerank", ...}, query) is equivalent but passes through an extra dispatch hop.
*/
type AiRerank struct {
	AggregateBase
}

func NewAiRerank(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &AiRerank{
		*NewAggregateBase("ai_rerank", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

func (this *AiRerank) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *AiRerank) Type() value.Type { return value.ARRAY }

func (this *AiRerank) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewAiRerank with the input operands
cast to a Function as the FunctionConstructor.
*/
func (this *AiRerank) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewAiRerank(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function.
*/
func (this *AiRerank) Copy() expression.Expression {
	rv := &AiRerank{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

func (this *AiRerank) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

// CumulateInitial collects values from operand[0] (docs) into an array.
// This is identical to AiCompute — both accumulate a partition's docs before
// the AI call is made.
func (this *AiRerank) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	// MISSING or BINARY use NULL_VALUE so that ARRAY position will be preserved.
	if item.Type() <= value.MISSING || item.Type() == value.BINARY {
		item = value.NULL_VALUE
	}

	return this.cumulatePart(value.NewTrackedValue([]interface{}{item}), cumulative, context)
}

/*
Aggregates intermediate results and returns them.
*/
func (this *AiRerank) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Compute the Final result after sorting (post processing).
*/
func (this *AiRerank) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	return cumulative, nil
}

func (this *AiRerank) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return part, nil
	}
	actual := part.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		if cumulative.Type() == value.ARRAY {
			var ok bool
			cumulative, ok = cumulative.Append(actual)
			if ok {
				return cumulative, nil
			}
		}

		array := cumulative.Actual()
		// This is an internal error: the cumulative accumulator should always be
		// an ARRAY at this point. If it is not, the aggregate state is corrupt.
		return nil, fmt.Errorf("AI_RERANK: internal error: expected cumulative value to be an array, got %T", array)
	default:
		// The partial value passed in was not a []interface{} slice; this
		// indicates the aggregate received an unexpected intermediate type.
		return nil, fmt.Errorf("AI_RERANK: internal error: unexpected partial value type %T", actual)
	}
}

// ai_rerank takes exactly 3 arguments: docs, options object, query string.
func (this *AiRerank) MaxArgs() int { return 3 }

func (this *AiRerank) MinArgs() int { return 3 }
