//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Window FIRST_VALUE() function.
It returns the first_value of sorted set of values.
nthItem will be 1.
*/
type FirstValue struct {
	AggregateBase
	nthItem int
}

/*
The function NewFirstValue calls NewAggregateBase to
create an aggregate function named FirstValue
*/
func NewFirstValue(operands expression.Expressions, flags uint32, filter expression.Expression, wTerm *WindowTerm) Aggregate {
	rv := &FirstValue{
		*NewAggregateBase("first_value", operands, flags, filter, wTerm), 1,
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *FirstValue) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type JSON.
*/
func (this *FirstValue) Type() value.Type { return value.JSON }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *FirstValue) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewFirstValue with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *FirstValue) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewFirstValue(operands, uint32(0), nil, nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *FirstValue) Copy() expression.Expression {
	rv := &FirstValue{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), expression.Copy(this.Filter()), CopyWindowTerm(this.WindowTerm())),
		this.nthItem,
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the FirstValue function, then the default value returned is a null.
The list attachment conatin the list of values. In this case 1.
startpos contains how many list values are finalized.
If there are duplicates in ORDER BY CLAUSE then lowest value of argument becomes result.
Also it need honor RESPECT | IGNORE NULLS CLAUSE
*/

func (this *FirstValue) Default(item value.Value, context Context) (value.Value, error) {
	av := value.NewAnnotatedValue(value.NULL_VALUE)
	av.SetAttachment(value.ATT_LIST, value.NewList(this.nthItem))
	av.SetAttachment(value.ATT_STARTPOS, value.ZERO_VALUE)
	return av, nil
}

/*
Aggregates input data by evaluating operands.
See the Default() section for details.
*/

func (this *FirstValue) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	e := compute_nth_value(item, cumulative, this.Operands()[0], this.nthItem, 1,
		true, this.HasFlags(AGGREGATE_IGNORENULLS), this.Name(), context)
	return cumulative, e
}

/*
Aggregates intermediate results and return them.
*/
func (this *FirstValue) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
Returns 0th value from the list attachment.
If there is no 0th element return Default value
*/
func (this *FirstValue) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	av, ok := cumulative.(value.AnnotatedValue)
	if !ok {
		return value.NULL_VALUE, nil
	}

	list, e := getList(cumulative)
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
Set attachment startpos value to number of items in the list attachment so that
items in the list until previous position are finalized.
When number of items in the list are same as nItems(i.e. 1). The aggregate is Done.
*/

func (this *FirstValue) IsCumulateDone(cumulative value.Value, context Context) (bool, error) {
	list, e := getList(cumulative)
	if e != nil {
		return false, e
	}

	values := list.Values()
	av := cumulative.(value.AnnotatedValue)
	av.SetAttachment(value.ATT_STARTPOS, value.NewValue(len(values)))

	return len(values) == this.nthItem, nil
}
