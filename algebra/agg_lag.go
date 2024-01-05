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
This represents the Window LAG() function.
It returns the LAG access to a row at a given physical offset prior to that position.
nthItem represents physical offset.
direction represents LAG (-1)
*/
type Lag struct {
	AggregateBase
	nthItem   int
	direction int
}

/*
The function NewLag calls NewAggregateBase to
create an aggregate function named Lag
*/
func NewLag(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &Lag{
		*NewAggregateBase("lag", operands, flags, filter, wTerm), 1, -1,
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Lag) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type JSON.
*/
func (this *Lag) Type() value.Type { return value.JSON }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Lag) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewLag with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Lag) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewLag(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Lag) Copy() expression.Expression {
	rv := &Lag{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
		this.nthItem, this.direction,
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the Lag function, then the default value returned is a null.
     The list attachment conatin the list of values.
     It need honor [RESPECT | IGNORE NULLS]

The physical offset value is decided by Second argument. It must be non zero positive integer.
     If second argument is expression depends on document it evalutes from the from current row.
     If no second argument physical offset is 1.

The third argument represnts default value when physical offset is out of bounds.
     If no third argument the default value will be NULL.
*/

func (this *Lag) Default(item value.Value, context Context) (value.Value, error) {
	av := value.NewAnnotatedValue(value.NULL_VALUE)
	this.nthItem = 1
	ops := this.Operands()

	if len(ops) > 2 {
		val, err := ops[2].Evaluate(item, context)
		if err != nil {
			return av, err
		}
		av = value.NewAnnotatedValue(val)
	}

	if len(ops) > 1 {
		nval, err := ops[1].Evaluate(item, context)
		if err == nil {
			if nval == nil || nval.Type() != value.NUMBER || nval.(value.NumberValue).Float64() <= 0.0 ||
				!value.IsInt(nval.(value.NumberValue).Float64()) {
				err = fmt.Errorf("%s() second argument%s must evaluate to a positive integer.", this.Name(), ops[1].ErrorContext())
			} else {
				this.nthItem = int(nval.(value.NumberValue).Int64())
			}
		}
		if err != nil {
			return av, err
		}
	}

	av.SetAttachment(value.ATT_LIST, value.NewList(this.nthItem))
	return av, nil
}

/*
Minimum input arguments required is 1.
*/
func (this *Lag) MinArgs() int { return 1 }

/*
Maximum number of input arguments allowed is 3.
*/
func (this *Lag) MaxArgs() int { return 3 }

/*
Aggregates input data by evaluating operands.
See the Default() section for details.
For LAG, This function is called from current row to start row (reverse order).
For LEAD, This function is called from current row to end row.
*/

func (this *Lag) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	e := compute_nth_value(item, cumulative, this.Operands()[0], this.nthItem, this.direction,
		false, this.HasFlags(AGGREGATE_IGNORENULLS), this.Name(), context)
	return cumulative, e
}

/*
Aggregates intermediate results and return them.
*/
func (this *Lag) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
Returns nth value from the list attachment.
If there is no nth element return Default value
*/

func (this *Lag) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
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
This is called when each row is processed to check if the aggregate is done.
When number of items in the list are same as nItems. The aggregate is Done.
*/

func (this *Lag) IsCumulateDone(cumulative value.Value, context Context) (bool, error) {
	list, e := getList(cumulative)
	if e != nil {
		return false, e
	}

	return list.Len() == this.nthItem, nil
}
