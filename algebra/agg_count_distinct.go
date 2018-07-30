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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function COUNT(DISTINCT expr).
It returns a count of all the distinct non-NULL, non-MISSING
values in the group. Type CountDistinct is a struct that
inherits from DistinctAggregateBase.
*/
type CountDistinct struct {
	DistinctAggregateBase
}

/*
The function NewCountDistinct calls NewDistinctAggregateBase to
create an aggregate function named COUNT with one expression
as input.
*/
func NewCountDistinct(operand expression.Expression) Aggregate {
	rv := &CountDistinct{
		*NewDistinctAggregateBase("count", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *CountDistinct) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *CountDistinct) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *CountDistinct) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewCountDistinct with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *CountDistinct) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewCountDistinct(operands[0])
	}
}

/*
If no input to the COUNT DISTINCT function, then the default value
returned is a zero value.
*/
func (this *CountDistinct) Default() value.Value { return value.ZERO_VALUE }

/*
Aggregates input data by evaluating operands. For null
and missing values , return the input value itself.
Call setAdd to compute the intermediate aggregate value
and return it.
*/
func (this *CountDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	return setAdd(item, cumulative, false), nil
}

/*
Aggregates distinct intermediate results and return them.
If the partial value is a zero value return the cumulative
value, and if the cumulative value is zero then return the
partial value.
*/
func (this *CountDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.ZERO_VALUE {
		return cumulative, nil
	} else if cumulative == value.ZERO_VALUE {
		return part, nil
	}

	return cumulateSets(part, cumulative)
}

/*
Compute the Final result. If input cumulative value is
a zero value return it. Return the length of the set
as the count (number of elements in the set).
*/
func (this *CountDistinct) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == value.ZERO_VALUE {
		return cumulative, nil
	}

	av := cumulative.(value.AnnotatedValue)
	set := av.GetAttachment("set").(*value.Set)
	return value.NewValue(set.Len()), nil
}
