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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function Median(DISTINCT expr).
It returns an arithmetic median of all the distinct
number values in the group. Type MedianDistinct is a struct that
inherits from DistinctAggregateBase.
*/
type MedianDistinct struct {
	DistinctAggregateBase
}

/*
The function NewMedianDistinct calls NewDistinctAggregateBase to
create an aggregate function named Median with one expression
as input.
*/
func NewMedianDistinct(operand expression.Expression) Aggregate {
	rv := &MedianDistinct{
		*NewDistinctAggregateBase("median", operand),
	}
	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *MedianDistinct) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *MedianDistinct) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *MedianDistinct) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewMedianDistinct with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *MedianDistinct) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewMedianDistinct(operands[0])
	}
}

/*
If no input to the Median function with DISTINCT, then the default value
returned is a null.
*/
func (this *MedianDistinct) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands. For all
values other than Number, return the input value itself.
Call setAdd to compute the intermediate aggregate value
and return it.
*/
func (this *MedianDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return setAdd(item, cumulative, true), nil
}

/*
Aggregates distinct intermediate results and return them.
*/
func (this *MedianDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateSets(part, cumulative)
}

/*
Compute the Final. Return NULL if no values of type NUMBER exist
Compute the median with "median Of Medians algorithm"
as described in https://www.ics.uci.edu/~eppstein/161/960130.html
for set of values with odd/even length respectively.
*/
func (this *MedianDistinct) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	av := cumulative.(value.AnnotatedValue)
	medianSet := av.GetAttachment("set").(*value.Set)
	if medianSet.Len() == 0 {
		return value.NULL_VALUE, nil
	}

	length := medianSet.Len()
	return medianOfMedian(medianSet.Values(), (length+1)/2, length&1 == 0), nil

}
