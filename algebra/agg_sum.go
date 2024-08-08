//  Copyright 2014-Present Couchbase, Inc.
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
This represents the Aggregate function SUM(expr). It returns
the sum of all the number values in the group. Type Sum
is a struct that inherits from AggregateBase.
*/
type Sum struct {
	AggregateBase
}

/*
The function NewSum calls NewAggregateBase to
create an aggregate function named SUM with
one expression as input.
*/
func NewSum(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Sum{
		*NewAggregateBase("sum", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Sum) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Sum) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Sum) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewSum with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Sum) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSum(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Sum) Copy() expression.Expression {
	rv := &Sum{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the SUM function, then the default value
returned is a null.
*/
func (this *Sum) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands. For all
values other than Number, return the input value itself. Call
cumulatePart to compute the intermediate aggregate value
and return it.
*/

func (this *Sum) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
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

	if this.Distinct() {
		return setAdd(item, cumulative, true), nil
	} else {
		return this.cumulatePart(item, cumulative, context)
	}
}

/*
Aggregates intermediate results and return them.
*/
func (this *Sum) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return cumulateSets(part, cumulative)
	} else {
		return this.cumulatePart(part, cumulative, context)
	}
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Sum) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return this.computeDistinctFinal(cumulative, context)
	} else {
		return cumulative, nil
	}
}

func (this *Sum) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return nil, fmt.Errorf("Invalid %v.CumulateRemove() for DISTINCT values.", this.Name())
	}

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	} else if cumulative.Type() == value.NUMBER {
		return value.AsNumberValue(cumulative).Sub(value.AsNumberValue(item)), nil
	}

	return nil, fmt.Errorf("Invalid %v.CumulateRemove() for %v value.", this.Name(), cumulative.Actual())
}

/*
Aggregate input partial values into cumulative result number value.
If the partial and current cumulative result are both float64
numbers, add them and return.
*/

func (this *Sum) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	}

	switch {
	case part.Type() == value.NUMBER:
		switch {
		case cumulative.Type() == value.NUMBER:
			return value.AsNumberValue(cumulative).Add(value.AsNumberValue(part)), nil
		default:
			return nil, fmt.Errorf("Invalid SUM %v of type %T.", cumulative.Actual(), cumulative.Actual())
		}
	default:
		return nil, fmt.Errorf("Invalid partial SUM %v of type %T.", part.Actual(), part.Actual())
	}
}

func (this *Sum) computeDistinctFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	av := cumulative.(value.AnnotatedValue)
	set := av.GetAttachment(value.ATT_SET).(*value.Set)
	if set.Len() == 0 {
		return value.NULL_VALUE, nil
	}

	sum := value.ZERO_NUMBER
	for _, v := range set.Values() {
		switch {
		case v.Type() == value.NUMBER:
			sum = sum.Add(value.AsNumberValue(v))
		default:
			return nil, fmt.Errorf("Invalid partial SUM %v of type %T.", v.Actual(), v.Actual())
		}
	}

	return sum, nil
}

func (this *Sum) SetRewriteIndexAggs() {
	this.AddFlags(AGGREGATE_REWRITE_INDEX_AGGS)
}
