//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
func NewCountn(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Countn{
		*NewAggregateBase("countn", operands, flags, filter, wTerm),
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
		return NewCountn(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Countn) Copy() expression.Expression {
	rv := &Countn{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the COUNTN function, then the default value
returned is a zero value.
*/
func (this *Countn) Default(item value.Value, context Context) (value.Value, error) {
	return value.ZERO_VALUE, nil
}

/*
Aggregates input data by evaluating operands. For missing and
null values return the input value itself. Call cumulatePart
to compute the intermediate aggregate value and return it.
*/
func (this *Countn) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {

	// apply filter if any
	if ok, e := this.evaluateFilter(item, context); e != nil || !ok {
		return cumulative, e
	}

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	if this.Distinct() {
		return setAdd(item, cumulative, true), nil
	} else {
		return this.cumulatePart(value.ONE_VALUE, cumulative, context)
	}
}

/*
Aggregates intermediate results and return them.
*/
func (this *Countn) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		if part == value.ZERO_VALUE {
			return cumulative, nil
		} else if cumulative == value.ZERO_VALUE {
			return part, nil
		}

		return cumulateSets(part, cumulative)
	} else {
		return this.cumulatePart(part, cumulative, context)
	}
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Countn) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		if cumulative == value.ZERO_VALUE {
			return cumulative, nil
		}

		av := cumulative.(value.AnnotatedValue)
		set := av.GetAttachment(value.ATT_SET).(*value.Set)
		return value.NewValue(set.Len()), nil
	} else {
		return cumulative, nil
	}
}

/*
Used for Incremental Aggregation.
For Distinct aggregate this method will not be called.
Cumulative must be NUMBER because it has been added earlier.
Remove the Numbered input data by evaluating operands from Aggregate.
*/

func (this *Countn) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	if this.Distinct() {
		return nil, fmt.Errorf("Invalid %v.CumulateRemove() for DISTINCT values.", this.Name())
	}

	item, e := this.Operands()[0].Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	if cumulative.Type() == value.NUMBER && value.AsNumberValue(cumulative).Int64() > 0 {
		return value.AsNumberValue(cumulative).Sub(value.AsNumberValue(value.ONE_VALUE)), nil
	}

	return nil, fmt.Errorf("Invalid %v.CumulateRemove() for %v value.", this.Name(), cumulative.Actual())
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
