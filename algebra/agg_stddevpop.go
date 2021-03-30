//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"math"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function STDDEV_POP(expr). It returns
the arithmetic population standard deviation of all the number values in the
group. Type StddevPop is a struct that inherits from AggregateBase.
*/
type StddevPop struct {
	AggregateBase
}

/*
The function NewStddevPop calls NewAggregateBase to
create an aggregate function named stddev_pop with
one expression as input.
*/
func NewStddevPop(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &StddevPop{
		*NewAggregateBase("stddev_pop", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StddevPop) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *StddevPop) Type() value.Type {
	return value.NUMBER
}

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *StddevPop) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewStddevPop with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *StddevPop) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewStddevPop(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *StddevPop) Copy() expression.Expression {
	rv := &StddevPop{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the StddevPop function, then the default value
returned is a null.
*/
func (this *StddevPop) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands.
For all values other than Number, return the input value itself.
Maintain two variables for sum and
list of all the values of type NUMBER.
Call stddevAdd to compute the intermediate aggregate value and return it.
*/
func (this *StddevPop) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	// apply filter if any
	if ok, e := this.evaluateFilter(item, context); e != nil || !ok {
		return cumulative, e
	}

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return addStddevVariance(item, cumulative, this.Distinct())
}

/*
Aggregates intermediate results and return them.
*/
func (this *StddevPop) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateStddevVariance(part, cumulative, this.Distinct())
}

/*
Compute the population standard deviation as the final.
Return NULL if no values of type NUMBER exist.
Return zero if only one value exists.
calculate variance according to definition
and return the square root of it as the standard deviation.
*/
func (this *StddevPop) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	variance, e := computeVariance(cumulative, this.Distinct(), false, 0.0)
	if e != nil {
		return nil, e
	}

	if variance == value.NULL_VALUE {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sqrt(variance.(value.NumberValue).Float64())), nil
}
