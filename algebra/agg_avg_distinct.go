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
This represents the Aggregate function AVG(DISTINCT expr).
It returns an arithmetic mean (average) of all the distinct
number values in the group. Type AvgDistinct is a struct that
inherits from DistinctAggregateBase.
*/
type AvgDistinct struct {
	DistinctAggregateBase
}

/*
The function NewAvgDistinct calls NewDistinctAggregateBase to
create an aggregate function named AVG with one expression
as input.
*/
func NewAvgDistinct(operand expression.Expression) Aggregate {
	rv := &AvgDistinct{
		*NewDistinctAggregateBase("avg", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *AvgDistinct) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *AvgDistinct) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *AvgDistinct) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewAvgDistinct with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *AvgDistinct) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewAvgDistinct(operands[0])
	}
}

/*
If no input to the AVG function with DISTINCT, then the default value
returned is a null.
*/
func (this *AvgDistinct) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands.For all
values other than Number, return the input value itself.
Call setAdd to compute the intermediate aggregate value
and return it.
*/
func (this *AvgDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
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
func (this *AvgDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateSets(part, cumulative)
}

/*
Compute the Final result. If input cumulative value is null return
it. Get the attachment, range over the values in the set and compute
the sum. Compute the avg by sum/length of set and return it.
(The values in the set are distinct).
*/
func (this *AvgDistinct) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
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
			return nil, fmt.Errorf("Invalid partial AVG %v of type %T.", a, a)
		}
	}

	return value.NewValue(sum / float64(set.Len())), nil
}
