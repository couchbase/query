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
This represents the Window CUME_DIST() function.
It returns the relative position of a specified value in a group of values.
The number of rows including equal rows in the order divide by total rows.
*/
type CumeDist struct {
	AggregateBase
}

/*
The function NewCumeDist calls NewAggregateBase to
create an aggregate function named CumeDist
*/
func NewCumeDist(operands expression.Expressions, flags uint32, wTerm *WindowTerm) Aggregate {
	rv := &CumeDist{
		*NewAggregateBase("cume_dist", operands, flags, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *CumeDist) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *CumeDist) Type() value.Type { return value.NUMBER }

/*
Minimum input arguments required is 0.
*/
func (this *CumeDist) MinArgs() int { return 0 }

/*
Maximum number of input arguments allowed is 0.
*/
func (this *CumeDist) MaxArgs() int { return 0 }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *CumeDist) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewCumeDist with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *CumeDist) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewCumeDist(operands, uint32(0), nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *CumeDist) Copy() expression.Expression {
	rv := &CumeDist{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), CopyWindowTerm(this.WindowTerm())),
	}

	rv.SetExpr(rv)
	return rv
}

/*
If no input to the CumeDist function, then the default value
returned is a 0.
*/
func (this *CumeDist) Default(item value.Value, context Context) (value.Value, error) {
	return value.ZERO_VALUE, nil
}

/*
Divide the number of rows including equal rows in the order divide by total rows.
The input is part of the window attachment in item.
*/

func (this *CumeDist) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	part, err := getWindowAttachment(item, this.Name())
	if err != nil || part == nil {
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part, part)
	}

	return this.cumulatePart(part, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *CumeDist) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregates Remove NOOP and return same input.
*/

func (this *CumeDist) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
*/
func (this *CumeDist) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
The number of rows including equal rows in the order  is avilable in the field part.
The total number of rows in the parttion is avilable in the field nrows
Return number of rows by total rows.
*/

func (this *CumeDist) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	nrowsv, _ := part.Field("nrows")
	partv, _ := part.Field("part")

	if partv.Type() != value.NUMBER || nrowsv.Type() != value.NUMBER {
		return nil, fmt.Errorf("%s internal Missing or invalid values: %v, %v.", this.Name(), partv.Actual(), nrowsv.Actual())
	}

	nrows := nrowsv.Actual().(float64)
	fpart := partv.Actual().(float64)

	if nrows > 0.0 {
		return value.NewValue(fpart / nrows), nil
	}

	return value.NULL_VALUE, nil
}
