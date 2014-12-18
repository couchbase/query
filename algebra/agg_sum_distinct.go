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

	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

/*
This represents the Aggregate function SUM(DISTINCT expr).
It returns the arithmetic sum of all the distinct number
values in the group. Type SumDistinct is a struct that
inherits from DistinctAggregateBase.
*/
type SumDistinct struct {
	DistinctAggregateBase
}

/*
The function NewSumDistinct calls NewDistinctAggregateBase to
create an aggregate function named COUNT with one expression
as input.
*/
func NewSumDistinct(operand expression.Expression) Aggregate {
	rv := &SumDistinct{
		*NewDistinctAggregateBase("sum", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *SumDistinct) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *SumDistinct) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *SumDistinct) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewSumDistinct with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *SumDistinct) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSumDistinct(operands[0])
	}
}

/*
If no input to the SUM function with DISTINCT, then the default value
returned is a null value.
*/
func (this *SumDistinct) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands. For non number
values, return the cumulative value. Call setAdd to compute
the intermediate aggregate value and return it.
*/
func (this *SumDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return setAdd(item, cumulative)
}

/*
Aggregates distinct intermediate results and return them.
*/
func (this *SumDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateSets(part, cumulative)
}

/*
Compute the Final result. If input cumulative value is
null then return it. Retrieve the set, if it is empty
return a null value. Range over the values in the set
and sum all the float64 number values, and return it.
If a non number value is encountered in the set, throw
an error.
*/
func (this *SumDistinct) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	av := cumulative.(value.AnnotatedValue)
	set := av.GetAttachment("set").(*value.Set)
	if set.Len() == 0 {
		return value.NULL_VALUE, nil
	}

	sum := 0.0
	for _, v := range set.Values() {
		a := v.Actual()
		switch a := a.(type) {
		case float64:
			sum += a
		default:
			return nil, fmt.Errorf("Invalid partial SUM %v of type %T.", a, a)
		}
	}

	return value.NewValue(sum), nil
}
