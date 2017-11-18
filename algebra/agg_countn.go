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
This represents the Aggregate function COUNTN(expr)
It returns the countn of all the non-NULL, non-MISSING  numeric values in
the group.  Type Countn is a struct that inherits from AggregateBase.
*/
type Countn struct {
	AggregateBase
}

/*
The function NewCountn calls NewAggregateBase to
create an aggregate function named COUNTN with
one expression as input.
*/
func NewCountn(operand expression.Expression) Aggregate {
	rv := &Countn{
		*NewAggregateBase("countn", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Countn) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Countn) Type() value.Type { return value.NUMBER }

/*
Directly call the evaluate method for aggregate functions and
pass in the receiver
*/
func (this *Countn) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewCountn with one input operand cast
to a Function as the FunctionConstructor.
*/
func (this *Countn) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewCountn(operands[0])
	}
}

/*
If no input to the COUNTN function, then the default value
returned is a zero value.
*/
func (this *Countn) Default() value.Value { return value.ZERO_VALUE }

/*
Aggregates input data by evaluating operands. For missing and
null values return the input value itself. Call cumulatePart
to compute the intermediate aggregate value and return it.
*/
func (this *Countn) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return this.cumulatePart(value.ONE_VALUE, cumulative, context)

}

/*
Aggregates intermediate results and return them.
*/
func (this *Countn) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Countn) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregate input partial values into cumulative result number value.
If the partial and current cumulative result are both float64
numbers, add them and return.
*/
func (this *Countn) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	switch part := part.(type) {
	case value.NumberValue:
		switch cumulative := cumulative.(type) {
		case value.NumberValue:
			return cumulative.Add(part), nil
		default:
			return nil, fmt.Errorf("Invalid COUNTN %v of type %T.", cumulative.Actual(), cumulative.Actual())
		}
	default:
		return nil, fmt.Errorf("Invalid partial COUNTN %v of type %T.", part.Actual(), part.Actual())
	}
}
