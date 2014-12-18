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
This represents the Aggregate function COUNT(expr) and COUNT(*).
It returns the count of all the non-NULL, non-MISSING values in
the group. If no input arguments then it returns a count of all
the input rows for the group, regardless of value. Type Count
is a struct that inherits from AggregateBase.
*/
type Count struct {
	AggregateBase
}

/*
The function NewCount calls NewAggregateBase to
create an aggregate function named COUNT with
one expression as input.
*/
func NewCount(operand expression.Expression) Aggregate {
	rv := &Count{
		*NewAggregateBase("count", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Count) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Count) Type() value.Type { return value.NUMBER }

/*
Directly call the evaluate method for aggregate functions and
passe in the receiver, current item and current context, for
count with an input expression operand. For a count with no
operands (count (*)), get the count from the attachment and
then evaluate.
*/
func (this *Count) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	if this.Operand() != nil {
		return this.evaluate(this, item, context)
	}

	// Full keyspace count is short-circuited
	switch item := item.(type) {
	case value.AnnotatedValue:
		count := item.GetAttachment("count")
		if count != nil {
			return value.NewValue(count), nil
		}
	}

	return this.evaluate(this, item, context)
}

/*
Minimum number of arguments to the count function is 0.
*/
func (this *Count) MinArgs() int { return 0 }

/*
The constructor returns a NewCount with either nil or one
input operand cast to a Function as the FunctionConstructor.
*/
func (this *Count) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		if len(operands) > 0 {
			return NewCount(operands[0])
		} else {
			return NewCount(nil)
		}
	}
}

/*
If no input to the COUNT function, then the default value
returned is a zero value.
*/
func (this *Count) Default() value.Value { return value.ZERO_VALUE }

/*
Aggregates input data by evaluating operands. For missing and
null values return the input value itself. Call cumulatePart
to compute the intermediate aggregate value and return it.
*/
func (this *Count) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	if this.Operand() != nil {
		item, e := this.Operand().Evaluate(item, context)
		if e != nil {
			return nil, e
		}

		if item.Type() <= value.NULL {
			return cumulative, nil
		}
	}

	return this.cumulatePart(value.ONE_VALUE, cumulative, context)

}

/*
Aggregates intermediate results and return them.
*/
func (this *Count) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Count) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregate input partial values into cumulative result number value.
If the partial and current cumulative result are both float64
numbers, add them and return.
*/
func (this *Count) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	actual := part.Actual()
	switch actual := actual.(type) {
	case float64:
		count := cumulative.Actual()
		switch count := count.(type) {
		case float64:
			return value.NewValue(count + actual), nil
		default:
			return nil, fmt.Errorf("Invalid COUNT %v of type %T.", count, count)
		}
	default:
		return nil, fmt.Errorf("Invalid partial COUNT %v of type %T.", actual, actual)
	}
}
