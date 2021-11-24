//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function MIN(expr). It returns
the minimum non-NULL, non-MISSING value in the group, in N1QL
collation order. Type Min is a struct that inherits from
AggregateBase.
*/
type Min struct {
	AggregateBase
}

/*
The function NewMin calls NewAggregateBase to
create an aggregate function named MIN with
one expression as input.
*/
func NewMin(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Min{
		*NewAggregateBase("min", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Min) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type JSON.
*/
func (this *Min) Type() value.Type { return value.JSON }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Min) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewMin with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Min) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewMin(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Min) Copy() expression.Expression {
	rv := &Min{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the MIN function, then the default value
returned is a null.
*/
func (this *Min) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands. For missing and
null values return the input value itself. Call cumulatePart
to compute the intermediate aggregate value and return it.
*/
func (this *Min) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	// apply filter if any
	if ok, e := this.evaluateFilter(item, context); e != nil || !ok {
		return cumulative, e
	}

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	return this.cumulatePart(item, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *Min) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Min) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregate input partial values into cumulative result value.
If partial result is null return the current cumulative value,
and if the cumulative result is null, return the partial value.
For non null partial and cumulative values, call Collate and
return the smaller value depending on the N1QL collation order.
*/
func (this *Min) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	} else if part.Collate(cumulative) < 0 {
		return part, nil
	} else {
		return cumulative, nil
	}
}
