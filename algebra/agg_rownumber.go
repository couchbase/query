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
This represents the Window ROW_NUMBER() function.
It returns a unique number to each row each row in the partition,
in the ordered sequence of rows specified in the ORDER BY clause.
It starts with 1.
*/
type RowNumber struct {
	AggregateBase
}

/*
The function NewRowNumber calls NewAggregateBase to
create an aggregate function named RowNumber
*/
func NewRowNumber(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &RowNumber{
		*NewAggregateBase("row_number", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RowNumber) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *RowNumber) Type() value.Type { return value.NUMBER }

/*
Minimum input arguments required is 0.
*/
func (this *RowNumber) MinArgs() int { return 0 }

/*
Maximum number of input arguments allowed is 0.
*/
func (this *RowNumber) MaxArgs() int { return 0 }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *RowNumber) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewRowNumber with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *RowNumber) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewRowNumber(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *RowNumber) Copy() expression.Expression {
	rv := &RowNumber{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the RowNumber function, then the default value returned is a 0.
*/
func (this *RowNumber) Default(item value.Value, context Context) (value.Value, error) {
	return value.ZERO_VALUE, nil
}

/*
The part field in the attachment of item has how much value to increase to get ROW_NUMBER()
i.e. 1
*/

func (this *RowNumber) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
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
func (this *RowNumber) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregates Remove NOOP and return same input them.
*/

func (this *RowNumber) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
*/
func (this *RowNumber) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
The part field in the attachment has how much value to increase to get ROW_NUMBER()
i.e. 1
*/
func (this *RowNumber) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
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
