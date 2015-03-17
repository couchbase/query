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
	"sort"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function ARRAY_AGG(expr). It returns an
array of the non-MISSING values in the group, including NULLs. Type
ArrayAgg is a struct that inherits from AggregateBase.
*/
type ArrayAgg struct {
	AggregateBase
}

/*
The function NewArrayAgg calls NewAggregateBase to
create an aggregate function named ARRAY_AGG with
one expression as input.
*/
func NewArrayAgg(operand expression.Expression) Aggregate {
	rv := &ArrayAgg{
		*NewAggregateBase("array_agg", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayAgg) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type ARRAY.
*/
func (this *ArrayAgg) Type() value.Type { return value.ARRAY }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayAgg) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewArrayAgg with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *ArrayAgg) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewArrayAgg(operands[0])
	}
}

/*
If no input to the ARRAY_AGG function, then the default value
returned is a null.
*/
func (this *ArrayAgg) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands. For missing
item values, return the input value itself. Call
cumulatePart to compute the intermediate aggregate value
and return it.
*/
func (this *ArrayAgg) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.MISSING || item.Type() == value.BINARY {
		return cumulative, nil
	}

	return this.cumulatePart(value.NewValue([]interface{}{item}), cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *ArrayAgg) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Compute the Final result after sorting(post processing).
*/
func (this *ArrayAgg) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	sort.Sort(value.NewSorter(cumulative))
	return cumulative, nil
}

/*
Aggregate input partial values into cumulative result slice of interfaces
and return. If no partial result exists(its value is a null) return the
cumulative value. If the cumulative input value is null, return the partial
value. Both values need to be slices. Append the partial result into the
cumulative value and return.
*/
func (this *ArrayAgg) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	}

	actual := part.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		array := cumulative.Actual()
		switch array := array.(type) {
		case []interface{}:
			return value.NewValue(append(array, actual...)), nil
		default:
			return nil, fmt.Errorf("Invalid ARRAY_AGG %v of type %T.", array, array)
		}
	default:
		return nil, fmt.Errorf("Invalid partial ARRAY_AGG %v of type %T.", actual, actual)
	}
}
