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
This represents the Window RANK() function.
It returns the rank of each row returned from a query with respect to the other rows, based on the values of order by CLAUSE.
Rows with equal values for the ranking criteria receive the same rank.  The number of tied rows to the tied rank to calculate
the next rank.
It starts with 1. The ranks may not be consecutive numbers.
*/
type Rank struct {
	AggregateBase
}

/*
The function NewRank calls NewAggregateBase to
create an aggregate function named Rank
*/
func NewRank(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Rank{
		*NewAggregateBase("rank", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Rank) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Rank) Type() value.Type { return value.NUMBER }

/*
Minimum input arguments required is 0.
*/
func (this *Rank) MinArgs() int { return 0 }

/*
Maximum number of input arguments allowed is 0.
*/
func (this *Rank) MaxArgs() int { return 0 }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Rank) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewRank with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Rank) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewRank(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Rank) Copy() expression.Expression {
	rv := &Rank{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the Rank function, then the default value returned is a 0.
*/

func (this *Rank) Default(item value.Value, context Context) (value.Value, error) {
	return value.ZERO_VALUE, nil
}

/*
The part field in the attachment of item has how much value to increase to get RANK()
*/

func (this *Rank) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	part, err := getWindowAttachment(item, this.Name())
	if err != nil {
		return nil, err
	}

	if part != nil {
		part, _ = part.Field("part")
	}

	if part == nil {
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part, part)
	}

	return this.cumulatePart(part, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *Rank) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregates Remove NOOP and return same input them.
*/

func (this *Rank) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Rank) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
The part field in the attachment has how much value to increase to get RANK()
The cumulative value represent the RANK() of the current row (add the previous rank by value from part)
*/
func (this *Rank) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	switch part := part.(type) {
	case value.NumberValue:
		switch cumulative := cumulative.(type) {
		case value.NumberValue:
			return cumulative.Add(part), nil
		default:
			return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), cumulative.Actual(), cumulative.Actual())
		}
	default:
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part.Actual(), part.Actual())
	}
}
