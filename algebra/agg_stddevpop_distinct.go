//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"math"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function STDDEV_POP(DISTINCT expr).
It returns an arithmetic population standard deviation of all the distinct
number values in the group. Type StddevPopDistinct is a struct that
inherits from DistinctAggregateBase.
*/
type StddevPopDistinct struct {
	DistinctAggregateBase
}

/*
The function NewStddevPopDistinct calls NewDistinctAggregateBase to
create an aggregate function named stddev_pop with one expression
as input.
*/
func NewStddevPopDistinct(operand expression.Expression) Aggregate {
	rv := &StddevPopDistinct{
		*NewDistinctAggregateBase("stddev_pop", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StddevPopDistinct) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *StddevPopDistinct) Type() value.Type {
	return value.NUMBER
}

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *StddevPopDistinct) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewStddevPopDistinct with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *StddevPopDistinct) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewStddevPopDistinct(operands[0])
	}
}

/*
If no input to the STDDEV_POP function with DISTINCT, then the default value
returned is a null.
*/
func (this *StddevPopDistinct) Default() value.Value {
	return value.NULL_VALUE
}

/*
Aggregates input data by evaluating operands.
For all values other than Number, return the input value itself.
Maintain two variables for sum and
set of all the values of type NUMBER.
Call addStddevVariance to compute the intermediate aggregate value and return it.
*/
func (this *StddevPopDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return addStddevVariance(item, cumulative, true)
}

/*
Aggregates distinct intermediate results and return them.
*/
func (this *StddevPopDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateStddevVariance(part, cumulative, true)
}

/*
Compute the Final. Return NULL if no values of type NUMBER exist.
Return zero if only one value exists.
calculate variance and return the square root of it as the population standard deviation.
*/
func (this *StddevPopDistinct) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	variance, e := computeVariance(cumulative, true, false, 0.0)
	if e != nil {
		return nil, e
	}

	if variance == value.NULL_VALUE {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sqrt(variance.(value.NumberValue).Float64())), nil
}
