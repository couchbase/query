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
	"sort"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function ARRAY_AGG(DISTINCT expr).
It returns an array of the distinct non-MISSING values in the
group, including NULLs. Type ArrayAggDistinct is a struct that
inherits from DistinctAggregateBase.
*/
type ArrayAggDistinct struct {
	DistinctAggregateBase
}

/*
The function NewArrayAggDistinct calls NewDistinctAggregateBase to
create an aggregate function named ARRAY_AGG with one expression
as input.
*/
func NewArrayAggDistinct(operand expression.Expression) Aggregate {
	rv := &ArrayAggDistinct{
		*NewDistinctAggregateBase("array_agg", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayAggDistinct) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type ARRAY.
*/
func (this *ArrayAggDistinct) Type() value.Type { return value.ARRAY }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayAggDistinct) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewArrayAggDistinct with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *ArrayAggDistinct) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewArrayAggDistinct(operands[0])
	}
}

/*
If no input to the ARRAY_AGG function with DISTINCT, then the default value
returned is a null.
*/
func (this *ArrayAggDistinct) Default() value.Value { return value.NULL_VALUE }

/*
Aggregates input data by evaluating operands. For missing
item values, return the input value itself. Call
setAdd to compute the intermediate aggregate value
and return it.
*/
func (this *ArrayAggDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.MISSING {
		return cumulative, nil
	}

	return setAdd(item, cumulative)
}

/*
Aggregates distinct intermediate results and return them.
*/
func (this *ArrayAggDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateSets(part, cumulative)
}

/*
Compute the Final result. If input cumulative value is null return
it. Get the attachment, create a new value and add it to the set
in a sorted manner. (The values in the set are distinct).
*/
func (this *ArrayAggDistinct) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	av := cumulative.(value.AnnotatedValue)
	set := av.GetAttachment("set").(*value.Set)
	if set.Len() == 0 {
		return value.NULL_VALUE, nil
	}

	actuals := set.Actuals()
	c = value.NewValue(actuals)
	sorter := value.NewSorter(c)
	sort.Sort(sorter)
	return c, nil
}
