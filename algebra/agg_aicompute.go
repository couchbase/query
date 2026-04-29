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
This represents the Window AI_COMPUTE(expr, options, args...) function.
It returns the compute of each row returned from a query with respect to the other rows or same row based on action in options.
*/
type AiCompute struct {
	AggregateBase
}

func NewAiCompute(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &AiCompute{
		*NewAggregateBase("ai_compute", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

func (this *AiCompute) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *AiCompute) Type() value.Type { return value.ARRAY }

func (this *AiCompute) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewAiCompute with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *AiCompute) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewAiCompute(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *AiCompute) Copy() expression.Expression {
	rv := &AiCompute{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

func (this *AiCompute) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

func (this *AiCompute) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	// MISSING or BINARY use NULL_VALUE so that ARRAY position will be preserved
	if item.Type() <= value.MISSING || item.Type() == value.BINARY {
		item = value.NULL_VALUE
	}

	return this.cumulatePart(value.NewTrackedValue([]interface{}{item}), cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *AiCompute) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Compute the Final result after sorting(post processing).
*/
func (this *AiCompute) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	return cumulative, nil
}

/*
Aggregate input partial values into cumulative result slice of interfaces
and return. If no partial result exists(its value is a null) return the
cumulative value. If the cumulative input value is null, return the partial
value. Both values need to be slices. Append the partial result into the
cumulative value and return.
*/
func (this *AiCompute) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
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
		return nil, fmt.Errorf("AI_COMPUTE: internal error: expected cumulative value to be an array, got %T", array)
	default:
		// The partial value passed in was not a []interface{} slice; this
		// indicates the aggregate received an unexpected intermediate type.
		return nil, fmt.Errorf("AI_COMPUTE: internal error: unexpected partial value type %T", actual)
	}
}

// ai_compute takes 2 or more arguments: docs, options object, then optional action-specific args.
func (this *AiCompute) MaxArgs() int { return 3 }

func (this *AiCompute) MinArgs() int { return 2 }
