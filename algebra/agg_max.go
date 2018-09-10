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
This represents the Aggregate function MAX(expr). It returns
the maximum non-NULL, non-MISSING value in the group, in N1QL
collation order. Type Max is a struct that inherits from
AggregateBase.
*/
type Max struct {
	AggregateBase
}

/*
The function NewMax calls NewAggregateBase to
create an aggregate function named MAX with
one expression as input.
*/
func NewMax(operands expression.Expressions, flags uint32, wTerm *WindowTerm) Aggregate {
	rv := &Max{
		*NewAggregateBase("max", operands, flags, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Max) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type JSON.
*/
func (this *Max) Type() value.Type { return value.JSON }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Max) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewMax with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Max) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewMax(operands, uint32(0), nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Max) Copy() expression.Expression {
	rv := &Max{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), CopyWindowTerm(this.WindowTerm())),
	}

	rv.SetExpr(rv)
	return rv
}

/*
If no input to the MAX function, then the default value
returned is a null.
*/
func (this *Max) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
Aggregates input data by evaluating operands. For missing and
null values return the input value itself. Call cumulatePart
to compute the intermediate aggregate value and return it.
*/
func (this *Max) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	return this.cumulatePart(item, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *Max) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Max) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregate input partial values into cumulative result value.
If partial result is null return the current cumulative value,
and if the cumulative result is null, return the partial value.
For non null partial and cumulative values, call Collate and
return the larger value depending on the N1QL collation order.
*/
func (this *Max) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	} else if part.Collate(cumulative) > 0 {
		return part, nil
	} else {
		return cumulative, nil
	}
}
