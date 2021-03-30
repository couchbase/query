//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function  VARIANCE_POP(expr). It returns
the arithmetic population variance of all the number values in the
group. Type VarPop is a struct that inherits from AggregateBase.
*/
type VarPop struct {
	AggregateBase
}

/*
The function NewVarPop calls NewAggregateBase to
create an aggregate function named var_pop with
one expression as input.
*/
func NewVarPop(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &VarPop{
		*NewAggregateBase("var_pop", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *VarPop) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *VarPop) Type() value.Type {
	return value.NUMBER
}

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *VarPop) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewVarPop with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *VarPop) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewVarPop(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *VarPop) Copy() expression.Expression {
	rv := &VarPop{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the VarPop function, then the default value
returned is a null.
*/
func (this *VarPop) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands.
For all values other than Number, return the input value itself.
Maintain two variables for sum and
list of all the values of type NUMBER.
Call addStddevVariance to compute the intermediate aggregate value and return it.
*/
func (this *VarPop) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
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
func (this *VarPop) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateStddevVariance(part, cumulative, this.Distinct())
}

/*
Compute the population variance as the final.
Return NULL if no values of type NUMBER exist.
Return zero if only one value exists.
calculate population variance according to definition
and return it.
*/
func (this *VarPop) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	variance, e := computeVariance(cumulative, this.Distinct(), false, 0.0)
	if e != nil {
		return nil, e
	}

	return variance, nil
}
