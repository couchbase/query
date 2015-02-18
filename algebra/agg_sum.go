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
func NewSum(operand expression.Expression) Aggregate {
	rv := &Sum{
		*NewAggregateBase("sum", operand),
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
		return NewSum(operands[0])
	}
}

/*
If no input to the SUM function, then the default value
returned is a null.
*/
func (this *Sum) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands. For all
values other than Number, return the input value itself. Call
cumulatePart to compute the intermediate aggregate value
and return it.
*/

func (this *Sum) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return this.cumulatePart(item, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *Sum) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Sum) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
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

	actual := part.Actual()
	switch actual := actual.(type) {
	case float64:
		sum := cumulative.Actual()
		switch sum := sum.(type) {
		case float64:
			return value.NewValue(sum + actual), nil
		default:
			return nil, fmt.Errorf("Invalid SUM %v of type %T.", sum, sum)
		}
	default:
		return nil, fmt.Errorf("Invalid partial SUM %v of type %T.", actual, actual)
	}
}
