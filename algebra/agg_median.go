//  Copyright 2018-Present Couchbase, Inc.
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
This represents the Aggregate function Median(expr). It returns
the arithmetic median of all the number values in the
group. Type Median is a struct that inherits from AggregateBase.
*/

type Median struct {
	AggregateBase
}

/*
The function NewMedian calls NewAggregateBase to
create an aggregate function named Median with
one expression as input.
*/
func NewMedian(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Median{
		*NewAggregateBase("median", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Median) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Median) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Median) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewMedian with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Median) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewMedian(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Median) Copy() expression.Expression {
	rv := &Median{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the Median function, then the default value
returned is a null.
*/
func (this *Median) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands. For all
values other than Number, return the input value itself. Call
arrayAdd to collect all the values of type NUMBER as the intermediate aggregate value
and return it.
*/
func (this *Median) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
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
		return listAdd(item, cumulative), nil
	}
}

/*
Aggregates intermediate results and return them.
*/
func (this *Median) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return cumulateSets(part, cumulative)
	} else {
		return cumulateLists(part, cumulative)
	}
}

/*
Compute the Final. Return NULL if no values of type NUMBER exist
Compute the median with "median Of Medians algorithm"
as described in https://www.ics.uci.edu/~eppstein/161/960130.html
for array of values with odd/even length respectively.
*/
func (this *Median) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	var length int
	var vals value.Values
	av := cumulative.(value.AnnotatedValue)

	if this.Distinct() {
		medianSet := av.GetAttachment(value.ATT_SET).(*value.Set)
		length = medianSet.Len()
		vals = medianSet.Values()
	} else {
		medianList := av.GetAttachment(value.ATT_LIST).(*value.List)
		length = medianList.Len()
		vals = medianList.Values()
	}

	if length == 0 {
		return value.NULL_VALUE, nil
	}
	return medianOfMedian(vals, (length+1)/2, length&1 == 0), nil
}
