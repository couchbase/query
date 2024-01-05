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
This represents the Aggregate function AVG(expr). It returns
the arithmetic mean (average) of all the number values in the
group. Type Avg is a struct that inherits from AggregateBase.
*/
type Avg struct {
	AggregateBase
}

/*
The function NewAvg calls NewAggregateBase to
create an aggregate function named AVG with
one expression as input.
*/
func NewAvg(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Avg{
		*NewAggregateBase("avg", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Avg) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Avg) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Avg) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewAvg with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Avg) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewAvg(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Avg) Copy() expression.Expression {
	rv := &Avg{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the AVG function, then the default value
returned is a null.
*/
func (this *Avg) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands. For all
values other than Number, return the input value itself. Call
cumulatePart to compute the intermediate aggregate value
and return it.
*/
func (this *Avg) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {

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
		part := value.NewValue(map[string]interface{}{"sum": item, "count": value.ONE_VALUE})
		return this.cumulatePart(part, cumulative, context)
	}
}

/*
Aggregates intermediate results and return them.
*/
func (this *Avg) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return cumulateSets(part, cumulative)
	} else {
		return this.cumulatePart(part, cumulative, context)
	}
}

/*
Compute the Final. Compute the sum and the count. If these
arent numbers throw an error. Compute the avg as sum/count.
Check for divide by zero, and return a NULL value if true.
*/
func (this *Avg) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	count := float64(0)
	sum := value.ZERO_NUMBER

	if this.Distinct() {
		av := cumulative.(value.AnnotatedValue)
		set := av.GetAttachment(value.ATT_SET).(*value.Set)
		count = float64(set.Len())

		for _, v := range set.Values() {
			switch {
			case v.Type() == value.NUMBER:
				sum = sum.Add(value.AsNumberValue(v))
			default:
				return nil, fmt.Errorf("Invalid partial AVG %v of type %T.", v.Actual(), v.Actual())
			}
		}

	} else {
		sumv, _ := cumulative.Field("sum")
		countv, _ := cumulative.Field("count")

		if sumv.Type() != value.NUMBER || countv.Type() != value.NUMBER {
			return nil, fmt.Errorf("Missing or invalid sum or count in AVG: %v, %v.",
				sumv.Actual(), countv.Actual())
		}

		sum = value.AsNumberValue(sumv)
		count = countv.Actual().(float64)
	}

	if count > 0.0 {
		return value.NewValue(sum.Actual().(float64) / count), nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Used for Incremental Aggregation.
For Distinct aggregate this method will not be called.
Cumulative must be NUMBER because it has been added earlier.
Remove the Numbered input data by evaluating operands from Aggregate.
*/

func (this *Avg) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return nil, fmt.Errorf("Invalid %v.CumulateRemove() for DISTINCT values.", this.Name())
	}

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	if cumulative.Type() > value.NULL {
		csum, sok := cumulative.Field("sum")
		ccount, cok := cumulative.Field("count")
		if sok && cok && csum.Type() == value.NUMBER && ccount.Type() == value.NUMBER {
			cumulative.SetField("sum", value.AsNumberValue(csum).Sub(value.AsNumberValue(item)))
			cumulative.SetField("count", value.AsNumberValue(ccount).Sub(value.AsNumberValue(value.ONE_VALUE)))
			return cumulative, nil
		}
	}

	return nil, fmt.Errorf("Invalid %v.CumulateRemove() for %v value.", this.Name(), cumulative.Actual())
}

/*
Aggregate input partial values into cumulative result number value
for sum and count. If the partial results are not numbers, then
return an error.
*/
func (this *Avg) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	}

	psum, _ := part.Field("sum")
	pcount, _ := part.Field("count")
	csum, _ := cumulative.Field("sum")
	ccount, _ := cumulative.Field("count")

	if psum.Type() != value.NUMBER || pcount.Type() != value.NUMBER ||
		csum.Type() != value.NUMBER || ccount.Type() != value.NUMBER {
		return nil, fmt.Errorf("Missing or invalid partial sum or count in AVG: %v, %v, %v, %v.",
			psum.Actual(), pcount.Actual(), csum.Actual(), ccount.Actual())
	}

	cumulative.SetField("sum", value.AsNumberValue(csum).Add(value.AsNumberValue(psum)))
	cumulative.SetField("count", value.AsNumberValue(ccount).Add(value.AsNumberValue(pcount)))
	return cumulative, nil
}
