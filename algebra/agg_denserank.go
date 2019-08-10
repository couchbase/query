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
	"fmt"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Window DENSE_RANK() function.
It returns the rank of each row returned from a query with
respect to the other rows, based on the values of ORDER BY CLAUSE.
Rows with equal values for the ranking criteria receive the same rank.
It starts with 1. The DENSE_RANK() will be consecutive numbers.
*/
type DenseRank struct {
	AggregateBase
}

/*
The function NewDenseRank calls NewAggregateBase to
create an aggregate function named DenseRank
*/
func NewDenseRank(operands expression.Expressions, flags uint32, wTerm *WindowTerm) Aggregate {
	rv := &DenseRank{
		*NewAggregateBase("dense_rank", operands, flags, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DenseRank) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *DenseRank) Type() value.Type { return value.NUMBER }

/*
Minimum input arguments required is 0.
*/
func (this *DenseRank) MinArgs() int { return 0 }

/*
Maximum number of input arguments allowed is 0.
*/
func (this *DenseRank) MaxArgs() int { return 0 }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *DenseRank) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewDenseRank with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *DenseRank) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewDenseRank(operands, uint32(0), nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *DenseRank) Copy() expression.Expression {
	rv := &DenseRank{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the DenseRank function, then the default value
returned is a 0.
*/
func (this *DenseRank) Default(item value.Value, context Context) (value.Value, error) {
	return value.ZERO_VALUE, nil
}

/*
The input to add is part of the window attachment in item.
*/

func (this *DenseRank) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	part, err := getWindowAttachment(item, this.Name())
	if err != nil {
		return nil, err
	}

	if part != nil {
		part, _ = part.Field("part")
	}

	if part == nil {
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part, part)
	}

	return this.cumulatePart(part, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *DenseRank) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregates Remove NOOP and return same input them.
*/

func (this *DenseRank) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
*/
func (this *DenseRank) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregate input part values into cumulative result number value.
*/
func (this *DenseRank) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	switch part := part.(type) {
	case value.NumberValue:
		switch cumulative := cumulative.(type) {
		case value.NumberValue:
			return cumulative.Add(part), nil
		default:
			return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), cumulative.Actual(), cumulative.Actual())
		}
	default:
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part.Actual(), part.Actual())
	}
}
