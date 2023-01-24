//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"bytes"
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type Aggregates []Aggregate

/*
The Aggregate interface represents aggregate functions such as SUM(),
AVG(), COUNT, COUNT(DISTINCT), MIN(), and MAX().

Aggregate functions are computed in parallel. Each aggregate function
must supply the methods CumulateInitial(), CumulateIntermediate(), and
CumulateFinal(). CumulateInitial() aggregates input values and
produces an intermediate aggregate. CumulateIntermediate() aggregates
intermediate aggregates and produces a further intermediate
aggregate. CumulateFinal() takes a final aggregate and performs any
post-processing. For example, Avg.CumulateFinal() divides the final
sum by the final count.

CumulateInitial() and CumulateIntermediate() can be run across
parallel input streams. CumulateFinal() must be run in a single serial
stream. CumulateIntermediate() must be chainable, to provide cascading
aggregation.

If no input data is received, the Default() value is returned.
*/
type Aggregate interface {
	/*
	   Represents the aggregate function.
	*/
	expression.Function

	/*
	   Set aggregate modifers/flags and Window Term.
	*/
	SetAggregateModifiers(flags uint32, filter expression.Expression, wTerm *WindowTerm)

	/*
	   Return WindowTerm.
	*/
	WindowTerm() *WindowTerm

	/*
	   Return Flags
	*/
	Flags() uint32

	/*
	   Checks Any of the flags are set.
	*/
	HasFlags(flag uint32) bool

	/*
	   Filter expression
	*/

	Filter() expression.Expression

	/*
	   Aggregate allows incremental operation.
	*/
	Incremental() bool

	/*
	   Window Aggregate
	*/
	IsWindowAggregate() bool

	/*
	   Returned if there is no input data to the function.
	*/
	Default(item value.Value, context Context) (value.Value, error)

	/*
	   Aggregates input data.
	*/
	CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error)

	/*
	   Aggregates intermediate results.
	*/
	CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error)

	/*
	   Remove input data from Aggregate result. For Incremental Aggregation only.
	*/
	CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error)

	/*
	   Performs final post-processing, if any.
	*/
	ComputeFinal(cumulative value.Value, context Context) (value.Value, error)

	/*
	   Check if aggregate is done, if any.
	*/
	IsCumulateDone(cumulative value.Value, context Context) (bool, error)
}

/*
Base class for Aggregate functions.
It inherits from expressions FunctionBase, and has
     text           which represents the function name.
     flags          which represents the modifers/flags
                         DISTINCT, INCREMENTAL, RESPECT|IGNORE NULLS, FROM FIRST|LAST
     filter         include those objects that filter condition is true in aggregation
     windowTerm     which represents the Window information
*/

type AggregateBase struct {
	expression.FunctionBase
	text       string
	flags      uint32
	filter     expression.Expression
	windowTerm *WindowTerm
}

/*
This method creates a new function using the
input expression name and operands, and returns it
as a pointer to an AggregateBase struct.
*/
func NewAggregateBase(name string, operands expression.Expressions, flags uint32, filter expression.Expression,
	wTerm *WindowTerm) *AggregateBase {
	rv := &AggregateBase{
		FunctionBase: *expression.NewFunctionBase(name, operands...),
		text:         "",
		flags:        flags,
	}
	rv.SetAggregateModifiers(flags, filter, wTerm)
	return rv
}

/*
Adds new flags to aggregate
*/
func (this *AggregateBase) AddFlags(flags uint32) {
	this.flags |= flags
}

/*
Checks Any flags are set
*/
func (this *AggregateBase) HasFlags(flags uint32) bool {
	return (this.flags & flags) != 0
}

/*
Sets aggregate modifiers/flags and window information
*/
func (this *AggregateBase) SetAggregateModifiers(flags uint32, filter expression.Expression, wTerm *WindowTerm) {
	name := this.Name()
	if name == "min" || name == "max" { // NO-OP
		flags &^= AGGREGATE_DISTINCT
	}

	this.flags = flags
	this.filter = filter
	this.windowTerm = wTerm

	// Aggregate allows incremental operation set the flags
	if !this.Distinct() && AggregateHasProperty(name, AGGREGATE_ALLOWS_INCREMENTAL) {
		this.AddFlags(AGGREGATE_INCREMENTAL)
	}
}

/*
Helper functions
*/

func (this *AggregateBase) Distinct() bool                { return this.HasFlags(AGGREGATE_DISTINCT) }
func (this *AggregateBase) Aggregate() bool               { return true }
func (this *AggregateBase) WindowTerm() *WindowTerm       { return this.windowTerm }
func (this *AggregateBase) Flags() uint32                 { return this.flags }
func (this *AggregateBase) MinArgs() int                  { return 1 }
func (this *AggregateBase) MaxArgs() int                  { return 1 }
func (this *AggregateBase) Filter() expression.Expression { return this.filter }

/*
If Incremental aggregation is possible or not
*/
func (this *AggregateBase) Incremental() bool {
	return this.HasFlags(AGGREGATE_INCREMENTAL) && !this.Distinct()
}

/*
 Remove the item from aggregation. Not supported on base calss.
 When supported each derived function overwrites it.
*/

func (this *AggregateBase) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return nil, fmt.Errorf("There is no %v.CumulateRemove().", this.Name())
}

/*
 Aggregation is Done for early termination. Not supported on base calss.
 When supported each derived function overwrites it.
*/

func (this *AggregateBase) IsCumulateDone(cumulative value.Value, context Context) (bool, error) {
	return false, fmt.Errorf("There is no %v.IsCumulateDone().", this.Name())
}

/*
Returns string representation of aggregate
*/
func (this *AggregateBase) String() string {
	var buf bytes.Buffer
	stringer := expression.NewStringer()

	buf.WriteString(this.Name())
	buf.WriteString("(")

	if this.Distinct() {
		buf.WriteString("DISTINCT ")
	}

	for i, op := range this.Operands() {
		if i > 0 {
			buf.WriteString(", ")
		}

		// special case: convert count() to count(*)
		if op == nil && this.Name() == "count" {
			buf.WriteString("*")
		} else {
			buf.WriteString(stringer.Visit(op))
		}
	}

	buf.WriteString(")")

	if this.Filter() != nil {
		buf.WriteString(" FILTER (WHERE ")
		buf.WriteString(stringer.Visit(this.Filter()))
		buf.WriteString(")")
	}

	// Handle [FROM FIRST|LAST]. FROM FIRST is default of ""
	if this.HasFlags(AGGREGATE_FROMLAST) {
		buf.WriteString(" FROM LAST")
	}

	// Handle [RESPECT|IGNORE NULLS]. RESPECT NULLS is default of ""
	if this.HasFlags(AGGREGATE_IGNORENULLS) {
		buf.WriteString(" IGNORE NULLS")
	}

	// Handle window term
	wTerm := this.WindowTerm()
	if wTerm != nil {
		buf.WriteString(wTerm.String())
	}

	return buf.String()
}

/*
Window Aggregate
*/
func (this *AggregateBase) IsWindowAggregate() bool {
	return this.windowTerm != nil
}

/*
This method evaluates the input aggregate, by retrieving the
aggregates map from the attachments and performing a lookup
using the input agg string value. If a result value(name) is
found, return it. If not throw an error stating that the
aggregate string is not found.
*/
func (this *AggregateBase) evaluate(agg Aggregate, item value.Value,
	context expression.Context) (result value.Value, err error) {
	if item == nil {
		return nil, errors.NewNilEvaluateParamError("item")
	}
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("Error evaluating aggregate: %v.", r)
		}
	}()

	av := item.(value.AnnotatedValue)
	aggregates := av.GetAttachment("aggregates")
	if aggregates != nil {
		aggs := aggregates.(map[string]value.Value)
		result = aggs[agg.String()]
	}

	if result == nil {
		err = fmt.Errorf("Aggregate %s not found.", agg.String())
	}

	return
}

/*
Not constant.
*/
func (this *AggregateBase) Value() value.Value {
	return nil
}

/*
Not static.
*/
func (this *AggregateBase) Static() expression.Expression {
	return nil
}

/*
Not indexable.
*/
func (this *AggregateBase) Indexable() bool {
	return false
}

/*
Aggregates are same Return true otherwise Return false.
*/

func (this *AggregateBase) EquivalentTo(other expression.Expression) bool {
	otherAggregate, ok := other.(Aggregate)
	return ok && this.String() == otherAggregate.String()
}

/*
Return when aggregate are aggregate are same except names
*/
func EqualAggregateModifiers(agg1, agg2 Aggregate) bool {
	wTerm1 := agg1.WindowTerm()
	wTerm2 := agg2.WindowTerm()

	return agg1.Flags() == agg2.Flags() &&
		expression.Equivalent(agg1.Filter(), agg2.Filter()) &&
		expression.Equivalents(agg1.Operands(), agg2.Operands()) &&
		((wTerm1 == wTerm2) || (wTerm1 != nil && wTerm2 != nil && wTerm1.String() == wTerm2.String()))
}

/*
Return False.
*/
func (this *AggregateBase) SubsetOf(other expression.Expression) bool {
	return false
}

/*
Return the operands of the Aggregate function.
*/
func (this *AggregateBase) Children() expression.Expressions {
	ops := this.Operands()
	rv := make(expression.Expressions, 0, len(ops))
	for _, op := range ops {
		if op != nil {
			rv = append(rv, op)
		}
	}

	if this.Filter() != nil {
		rv = append(rv, this.Filter())
	}

	wTerm := this.WindowTerm()
	if wTerm != nil {
		exprs := wTerm.Expressions()
		if len(exprs) > 0 {
			rv = append(rv, exprs...)
		}
	}

	return rv
}

/*
It is a utility function that takes in as input parameter
a mapper and maps the involved expressions to an expression.
If there is an error during the mapping, an error is returned.
*/
func (this *AggregateBase) MapChildren(mapper expression.Mapper) error {
	children := this.Operands()

	for i, c := range children {
		if c != nil {
			expr, err := mapper.Map(c)
			if err != nil {
				return err
			}

			children[i] = expr
		}
	}

	if this.Filter() != nil {
		expr, err := mapper.Map(this.Filter())
		if err != nil {
			return err
		}

		this.filter = expr
	}

	wTerm := this.WindowTerm()
	if wTerm != nil {
		return wTerm.MapExpressions(mapper)
	}

	return nil
}

/*
Check if the expressions are depends only on group by or aggregates.
Regular aggregate this is NO-OP.
window aggregate the expression in arguments, PARTITION BY, ORDER BY, WINDOWING clause
must be part of group by or regular aggregates (If there is GORUP BY)
*/
func (this *AggregateBase) SurvivesGrouping(groupKeys expression.Expressions,
	allowed *value.ScopeValue) (bool, expression.Expression) {
	if this.WindowTerm() != nil {
		for _, child := range this.Children() {
			ok, _ := child.SurvivesGrouping(groupKeys, allowed)
			if !ok {
				return ok, child
			}
		}
	}
	return true, nil
}

func (this *AggregateBase) evaluateFilter(item value.Value, context Context) (bool, error) {
	if this.Filter() == nil {
		return true, nil
	}

	val, e := this.Filter().Evaluate(item, context)
	if e != nil {
		return false, e
	}

	return val.Truth(), nil

}
