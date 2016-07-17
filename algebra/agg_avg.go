//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
func NewAvg(operand expression.Expression) Aggregate {
	rv := &Avg{
		*NewAggregateBase("avg", operand),
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
		return NewAvg(operands[0])
	}
}

/*
If no input to the AVG function, then the default value
returned is a null.
*/
func (this *Avg) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands. For all
values other than Number, return the input value itself. Call
cumulatePart to compute the intermediate aggregate value
and return it.
*/
func (this *Avg) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	part := value.NewValue(map[string]interface{}{"sum": item, "count": value.ONE_VALUE})
	return this.cumulatePart(part, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *Avg) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
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

	sum, _ := cumulative.Field("sum")
	count, _ := cumulative.Field("count")

	if sum.Type() != value.NUMBER || count.Type() != value.NUMBER {
		return nil, fmt.Errorf("Missing or invalid sum or count in AVG: %v, %v.",
			sum.Actual(), count.Actual())
	}

	if count.Actual().(float64) > 0.0 {
		return value.NewValue(sum.Actual().(float64) / count.Actual().(float64)), nil
	} else {
		return value.NULL_VALUE, nil
	}
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
		return nil, fmt.Errorf("Missing or invalid partial sum or count in AVG: %v, %v, %v, v.",
			psum.Actual(), pcount.Actual(), csum.Actual(), ccount.Actual())
	}

	cumulative.SetField("sum", csum.(value.NumberValue).Add(psum.(value.NumberValue)))
	cumulative.SetField("count", ccount.(value.NumberValue).Add(pcount.(value.NumberValue)))
	return cumulative, nil
}
