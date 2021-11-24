//  Copyright 2018-Present Couchbase, Inc.
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
This represents the Window RATIO_TO_REPORT() function.
It returns the ratio of a value to the sum of a set of values in the PARTITION.
*/
type RatioToReport struct {
	AggregateBase
}

/*
The function NewRatioToReport calls NewAggregateBase to
create an aggregate function named RatioToReport
*/
func NewRatioToReport(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &RatioToReport{
		*NewAggregateBase("ratio_to_report", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RatioToReport) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *RatioToReport) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *RatioToReport) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewRatioToReport with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *RatioToReport) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewRatioToReport(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *RatioToReport) Copy() expression.Expression {
	rv := &RatioToReport{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the RatioToReport function, then the default value returned is a null.
*/
func (this *RatioToReport) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
The part field in the attachment of item has sum of partition values
*/

func (this *RatioToReport) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	iv, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if iv.Type() != value.NUMBER {
		return cumulative, nil
	}

	part, err := getWindowAttachment(item, this.Name())
	if err != nil || part == nil {
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part, part)
	}

	return this.cumulatePart(iv, part, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *RatioToReport) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
*/
func (this *RatioToReport) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
The part field in the attachment has the sum of partition values
Returns item value divided by sum
*/
func (this *RatioToReport) cumulatePart(item, part value.Value, context Context) (value.Value, error) {
	sumv, _ := part.Field("part")
	if sumv.Type() == value.NULL || item.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	if sumv.Type() != value.NUMBER || item.Type() != value.NUMBER {
		return nil, fmt.Errorf("%s internal Missing or invalid values: %v, %v.",
			this.Name(), sumv.Actual(), item.Actual())
	}

	sum := sumv.Actual().(float64)
	val := item.Actual().(float64)
	if sum == 0.0 {
		return value.NULL_VALUE, nil
	} else {
		return value.NewValue(val / sum), nil
	}
}
