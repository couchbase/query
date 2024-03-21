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
This represents the Window NTH_VALUE() function.
It returns the NTH_VALUE() of sorted set of values.
direction represents [FROM FIRST|LAST]. Default is FROM FIRST
*/
type NthValue struct {
	AggregateBase
	nthItem   int
	direction int
}

/*
The function NewNthValue calls NewAggregateBase to
create an aggregate function named NthValue
*/
func NewNthValue(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &NthValue{
		*NewAggregateBase("nth_value", operands, flags, filter, wTerm), 1, 1,
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NthValue) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type JSON.
*/
func (this *NthValue) Type() value.Type { return value.JSON }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *NthValue) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewNthValue with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *NthValue) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewNthValue(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *NthValue) Copy() expression.Expression {
	rv := &NthValue{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
		this.nthItem, this.direction,
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the NthValue function, then the default value returned is a null.
The list attachment conatin the list of values.
startpos contains how many list values are finalized.
If there are duplicates in ORDER BY CLAUSE then lowest or highest value of argument becomes result based
on FROM FIRST|LAST CLAUSE
Also it need honor [FROM FIRST|LAST] [RESPECT | IGNORE NULLS]
The nth value is decided by Second argument. It must be non zero positive integer.
If second argument is expression depends on document it evalutes from the from current row.
*/

func (this *NthValue) Default(item value.Value, context Context) (value.Value, error) {
	nval, err := this.Operands()[1].Evaluate(item, context)
	if err == nil {
		if nval == nil || nval.Type() != value.NUMBER || nval.(value.NumberValue).Float64() <= 0.0 ||
			!value.IsInt(nval.(value.NumberValue).Float64()) {
			err = fmt.Errorf("%s() second argument%s must evaluate to a positive integer.", this.Name(),
				this.Operands()[1].ErrorContext())
		} else {
			this.nthItem = int(nval.(value.NumberValue).Int64())
		}
	}
	if this.HasFlags(AGGREGATE_FROMLAST) {
		this.direction = -1
	} else {
		this.direction = 1
	}
	av := value.NewAnnotatedValue(value.NULL_VALUE)
	av.SetAttachment("list", value.NewList(this.nthItem))
	av.SetAttachment("startpos", value.ZERO_VALUE)
	return av, err
}

/*
Minimum input arguments required is 2.
*/
func (this *NthValue) MinArgs() int { return 2 }

/*
Maximum number of input arguments allowed is 2.
*/
func (this *NthValue) MaxArgs() int { return 2 }

/*
Aggregates input data by evaluating operands.
See the Default() section for details.
FROM LAST, This function is called in revrse order. I.e. window frame end to start.
*/

func (this *NthValue) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	e := compute_nth_value(item, cumulative, this.Operands()[0], this.nthItem, this.direction,
		true, this.HasFlags(AGGREGATE_IGNORENULLS), this.Name(), context)
	return cumulative, e
}

/*
Aggregates intermediate results and return them.
*/
func (this *NthValue) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
Note: FROM LAST, This aggregate is called window frame end to start.
Returns nth value from the list attachment.
If there is no nth element return Default value
*/

func (this *NthValue) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	av, ok := cumulative.(value.AnnotatedValue)
	if !ok {
		return value.NULL_VALUE, nil
	}

	list, e := getList(av)
	if e != nil {
		return av.GetValue(), e
	}

	values := list.Values()
	if len(values) != this.nthItem {
		return av.GetValue(), nil
	}

	return values[this.nthItem-1], nil
}

/*
This is called when all the duplicates in ORDER BY CLAUSE are processed.
set attachment startpos value to number of items in the list attachment so that
items in the list until previous position are finalized.
When number of items in the list are same as nItems. The aggregate is Done.
*/

func (this *NthValue) IsCumulateDone(cumulative value.Value, context Context) (bool, error) {
	list, e := getList(cumulative)
	if e != nil {
		return false, e
	}

	values := list.Values()
	av := cumulative.(value.AnnotatedValue)
	av.SetAttachment("startpos", value.NewValue(len(values)))

	return len(values) == this.nthItem, nil
}
