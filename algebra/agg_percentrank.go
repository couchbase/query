//  Copyright 2018-Present Couchbase, Inc.
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
This represents the Window PERCENT_RANK() function.
It returns the RANK() minus 1 divided by number of rows minus 1.
The range of values returned is 0 to 1, inclusive.
*/
type PercentRank struct {
	AggregateBase
}

/*
The function NewPercentRank calls NewAggregateBase to
create an aggregate function named PercentRank
*/
func NewPercentRank(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &PercentRank{
		*NewAggregateBase("percent_rank", operands, flags, filter, wTerm),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *PercentRank) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *PercentRank) Type() value.Type { return value.NUMBER }

/*
Minimum input arguments required is 0.
*/
func (this *PercentRank) MinArgs() int { return 0 }

/*
Maximum number of input arguments allowed is 0.
*/
func (this *PercentRank) MaxArgs() int { return 0 }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *PercentRank) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewPercentRank with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *PercentRank) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewPercentRank(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *PercentRank) Copy() expression.Expression {
	rv := &PercentRank{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the PercentRank function, then the default value
returned is a null.
*/
func (this *PercentRank) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
The input to add is part of the window attachment in item.
     The part field in the attachment has how much value to increase to get RANK()
     The nrows field in the attachment has total rows in the PARTITION
*/

func (this *PercentRank) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	part, err := getWindowAttachment(item, this.Name())
	if err != nil || part == nil {
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part, part)
	}

	return this.cumulatePart(part, cumulative, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *PercentRank) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregates Remove NOOP and return same input them.
*/

func (this *PercentRank) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
The cumulative value

	part represent the RANK() of the current row
	nrows represent the total row in the PARTITION

Returns Final result as rank minus 1 divided by total rows minus 1
*/
func (this *PercentRank) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}
	cnrows, _ := cumulative.Field("nrows")
	cpart, _ := cumulative.Field("part")

	if cnrows.Type() != value.NUMBER || cpart.Type() != value.NUMBER {
		return nil, fmt.Errorf("%s internal Missing or invalid values: %v, %v.",
			this.Name(), cnrows.Actual(), cpart.Actual())
	}

	nrows := cnrows.Actual().(float64)
	fpart := cpart.Actual().(float64)
	if nrows <= 1.0 {
		return value.ZERO_VALUE, nil
	} else {
		return value.NewValue((fpart - 1) / (nrows - 1)), nil
	}
}

/*
The part field in the attachment has how much value to increase to get RANK()
The nrows field in the attachment has total rows in the PARTITION
The cumulative value

	part represent the RANK() of the current row (add the previous rank by value from part)
	nrows represent the total row in the PARTITION
*/
func (this *PercentRank) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		cumulative = value.NewValue(map[string]interface{}{"part": value.ZERO_VALUE, "nrows": value.ONE_VALUE})
	}

	pnrows, _ := part.Field("nrows")
	ppart, _ := part.Field("part")
	cpart, _ := cumulative.Field("part")

	if pnrows.Type() != value.NUMBER || ppart.Type() != value.NUMBER || cpart.Type() != value.NUMBER {
		return nil, fmt.Errorf("%s internal Missing or invalid values: %v, %v, %v.",
			this.Name(), pnrows.Actual(), ppart.Actual(), cpart.Actual())
	}

	cumulative.SetField("nrows", value.AsNumberValue(pnrows))
	cumulative.SetField("part", value.AsNumberValue(cpart).Add(value.AsNumberValue(ppart)))

	return cumulative, nil
}
