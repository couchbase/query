//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// ClockMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function CLOCK_MILLIS(). It
returns the system clock at function evaluation time,
as UNIX milliseconds and varies during a query. Type
ClockMillis is a struct that implements NullaryFunctionBase.
*/
type ClockMillis struct {
	NullaryFunctionBase
}

var _CLOCK_MILLIS = NewClockMillis()

/*
The function NewClockMillis calls NewNullaryFunctionBase to
create a function named CLOCK_MILLIS.
*/
func NewClockMillis() Function {
	rv := &ClockMillis{
		*NewNullaryFunctionBase("clock_millis"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ClockMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *ClockMillis) Type() value.Type { return value.NUMBER }

/*
Get the current local time in Unix Nanoseconds. In
order to convert it to milliseconds, divide it by
10^6.
*/
func (this *ClockMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := time.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

/*
The constructor returns a FunctionConstructor by casting the receiver to a
Function as the FunctionConstructor.
*/
func (this *ClockMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _CLOCK_MILLIS }
}

///////////////////////////////////////////////////
//
// ClockStr
//
///////////////////////////////////////////////////
/*
This represents the Date function CLOCK_STR([ fmt ]). It returns
the system clock at function evaluation time, as a string in a
supported format and varies during a query. There are a set of
supported formats. Type ClockStr is a struct that implements
FunctionBase.
*/
type ClockStr struct {
	FunctionBase
}

/*
The function NewClockStr calls NewFunctionBase to
create a function named CLOCK_STR with input
arguments as the operands from the input expression.
*/
func NewClockStr(operands ...Expression) Function {
	rv := &ClockStr{
		*NewFunctionBase("clock_str", operands...),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ClockStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *ClockStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *ClockStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
Value() returns the static / constant value of this Expression, or
nil. Expressions that depend on data, clocks, or random numbers must
return nil.
*/
func (this *ClockStr) Value() value.Value {
	return nil
}

/*
Initialize format to default format. This is in the event the
function is called without input arguments. Then it uses the
default time format. If it has input args, and if it is a
missing type return a missing value. It the type is not a string
then return null value. In the event it is a string, convert the
format to a valid Go type, cast it to a string, and call timeToStr
function using the time package and return a NewValue.
*/
func (this *ClockStr) Apply(context Context, args ...value.Value) (value.Value, error) {
	fmt := _DEFAULT_FORMAT

	if len(args) > 0 {
		fv := args[0]
		if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.Actual().(string)
	}

	return value.NewValue(timeToStr(time.Now(), fmt)), nil
}

/*
Minimum input arguments required for the defined function
CLOCK_STR is 0.
*/
func (this *ClockStr) MinArgs() int { return 0 }

/*
Maximum input arguments allowable for the defined function
CLOCK_STR is 1.
*/
func (this *ClockStr) MaxArgs() int { return 1 }

/*
Returns receiver as FunctionConstructor.
*/
func (this *ClockStr) Constructor() FunctionConstructor { return NewClockStr }

///////////////////////////////////////////////////
//
// DateAddMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_ADD_MILLIS(expr,n,part).
It performs date arithmetic. n and part are used to define an
interval or duration, which is then added (or subtracted) to
the UNIX timestamp, returning the result. Type DateAddMillis
is a struct that implements TernaryFunctionBase since it has
3 input arguments.
*/
type DateAddMillis struct {
	TernaryFunctionBase
}

/*
The function NewDateAddMillis calls NewTernaryFunctionBase to
create a function named DATE_ADD_MILLIS with the three
expressions as input.
*/
func NewDateAddMillis(first, second, third Expression) Function {
	rv := &DateAddMillis{
		*NewTernaryFunctionBase("date_add_millis", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DateAddMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *DateAddMillis) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *DateAddMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method takes inputs date, n and part as values and returns a value.
If any of these are missing then return a missing value. If date and n
arent numbers or if part isnt a string then return a null value. Call
Actual for these values to convert into valid Go type and cast date,n to
float64(N1QL valid number type) and part to string. If n is a floating
point value return null value. Call the dateAdd method to a add n to
the time obtained by converting the date to time using the millisToTime
method. Convert the result back using timeToMillis and then return it in
value format.
*/
func (this *DateAddMillis) Apply(context Context, date, n, part value.Value) (value.Value, error) {
	if date.Type() == value.MISSING || n.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date.Type() != value.NUMBER || n.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da := date.Actual().(float64)
	na := n.Actual().(float64)
	if na != math.Trunc(na) {
		return value.NULL_VALUE, nil
	}

	pa := part.Actual().(string)
	t, err := dateAdd(millisToTime(da), int(na), pa)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

/*
The constructor returns a NewDateAddMillis with the three operands
cast to a Function as the FunctionConstructor.
*/
func (this *DateAddMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateAddMillis(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// DateAddStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_ADD_STR(expr,n,part).
It performs date arithmetic. n and part are used to define an
interval or duration, which is then added to the date string
in a supported format, returning the result. Type DateAddStr
is a struct that implements TernaryFunctionBase since it has
3 input arguments.
*/
type DateAddStr struct {
	TernaryFunctionBase
}

/*
The function NewDateAddStr calls NewTernaryFunctionBase to
create a function named DATE_ADD_STR with the three
expressions as input.
*/
func NewDateAddStr(first, second, third Expression) Function {
	rv := &DateAddStr{
		*NewTernaryFunctionBase("date_add_str", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DateAddStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *DateAddStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *DateAddStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method takes inputs date, n and part as values and returns a value.
If any of these are missing then return a missing value. If n isnt a
number or if date and part arent strings then return a null value. Call
Actual for these values to convert into valid Go type and cast n to
float64(N1QL valid number type) and date,part to string. If n is a floating
point value return null value. Call the dateAdd method to a add n to
the time obtained. Return it in value format.
*/
func (this *DateAddStr) Apply(context Context, date, n, part value.Value) (value.Value, error) {
	if date.Type() == value.MISSING || n.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date.Type() != value.STRING || n.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da := date.Actual().(string)
	t, fmt, err := strToTimeFormat(da)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	na := n.Actual().(float64)
	if na != math.Trunc(na) {
		return value.NULL_VALUE, nil
	}

	pa := part.Actual().(string)
	t, err = dateAdd(t, int(na), pa)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t, fmt)), nil
}

/*
The constructor returns a NewDateAddStr with the three operands
cast to a Function as the FunctionConstructor.
*/
func (this *DateAddStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateAddStr(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// DateDiffMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_DIFF_MILLIS(expr1,expr2,part).
It performs date arithmetic. It returns the elapsed time between two
UNIX timestamps, as an integer whose unit is part. Type DateDiffMillis
is a struct that implements TernaryFunctionBase since it has
3 input arguments.
*/
type DateDiffMillis struct {
	TernaryFunctionBase
}

/*
The function NewDateDiffMillis calls NewTernaryFunctionBase to
create a function named DATE_DIFF_MILLIS with the three
expressions as input.
*/
func NewDateDiffMillis(first, second, third Expression) Function {
	rv := &DateDiffMillis{
		*NewTernaryFunctionBase("date_diff_millis", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DateDiffMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *DateDiffMillis) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *DateDiffMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *DateDiffMillis) Apply(context Context, date1, date2, part value.Value) (value.Value, error) {
	if date1.Type() == value.MISSING || date2.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date1.Type() != value.NUMBER || date2.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := date1.Actual().(float64)
	da2 := date2.Actual().(float64)
	pa := part.Actual().(string)
	diff, err := dateDiff(millisToTime(da1), millisToTime(da2), pa)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(diff)), nil
}

/*
The constructor returns a NewDateDiffMillis with the three operands
cast to a Function as the FunctionConstructor.
*/
func (this *DateDiffMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateDiffMillis(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// DateDiffStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_DIFF_STR(expr1,expr2,part).
It performs date arithmetic and returns the elapsed time between two
date strings in a supported format, as an integer whose unit is
part.Type. DateAddStr is a struct that implements TernaryFunctionBase
since it has 3 input arguments.
*/
type DateDiffStr struct {
	TernaryFunctionBase
}

/*
The function NewDateDiffStr calls NewTernaryFunctionBase to
create a function named DATE_DIFF_STR with the three
expressions as input.
*/
func NewDateDiffStr(first, second, third Expression) Function {
	rv := &DateDiffStr{
		*NewTernaryFunctionBase("date_diff_str", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DateDiffStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *DateDiffStr) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *DateDiffStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method takes two dates and part as input values and returns a value.
If any of these are missing then return a missing value. If the dates
arent numbers or if part isnt a string then return a null value. Call
Actual for these values to convert into valid Go type and call strToTime
to convert the dates into valid format. dateDiff returns the difference,
that is cast to float64 and returned.
*/
func (this *DateDiffStr) Apply(context Context, date1, date2, part value.Value) (value.Value, error) {
	if date1.Type() == value.MISSING || date2.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date1.Type() != value.STRING || date2.Type() != value.STRING || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := date1.Actual().(string)
	t1, err := strToTime(da1)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	da2 := date2.Actual().(string)
	t2, err := strToTime(da2)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	pa := part.Actual().(string)
	diff, err := dateDiff(t1, t2, pa)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(diff)), nil
}

/*
The constructor returns a NewDateDiffMillis with the three operands
cast to a Function as the FunctionConstructor.
*/
func (this *DateDiffStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateDiffStr(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// DatePartMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_PART_MILLIS(expr, part).
It returns the date part as an integer. The date expr is a
number representing UNIX milliseconds, and part is one of the
date part strings. DatePartMillis is a struct that implements
BinaryFunctionBase.
*/
type DatePartMillis struct {
	BinaryFunctionBase
}

/*
The function NewDatePartMillis calls NewBinaryFunctionBase to
create a function named DATE_PART_MILLIS with the two
expressions as input.
*/
func NewDatePartMillis(first, second Expression) Function {
	rv := &DatePartMillis{
		*NewBinaryFunctionBase("date_part_millis", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DatePartMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *DatePartMillis) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *DatePartMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes inputs date and part as values and returns a value.
If either of these are missing then return a missing value. If date
isnt number or if part isnt a string then return a null value. Call
Actual for these values to convert into valid Go type and cast date to
float64(N1QL valid number type) and part to string. Call datePart function
with the date converted to time format and return it as a value.
*/
func (this *DatePartMillis) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)
	part := second.Actual().(string)
	rv, err := datePart(millisToTime(millis), part)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(rv)), nil
}

/*
The constructor returns a NewDatePartMillis with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *DatePartMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDatePartMillis(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// DatePartStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_PART_STR(expr, part).
It returns the date part as an integer. The date expr is a
string in a supported format, and part is one of the supported
date part strings. DatePartStr is a struct that implements
BinaryFunctionBase.
*/
type DatePartStr struct {
	BinaryFunctionBase
}

/*
The function NewDatePartStr calls NewBinaryFunctionBase to
create a function named DATE_PART_STR with the two
expressions as input.
*/
func NewDatePartStr(first, second Expression) Function {
	rv := &DatePartStr{
		*NewBinaryFunctionBase("date_part_str", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DatePartStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *DatePartStr) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *DatePartStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes inputs date and part as values and returns a value.
If either of these are missing then return a missing value. If date
isnt number or if part isnt a string then return a null value. Call
Actual for these values to convert into valid Go type and cast date to
float64(N1QL valid number type) and part to string. Call datePart function
with the date and return it as a value.
*/
func (this *DatePartStr) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	part := second.Actual().(string)
	t, err := strToTime(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	rv, err := datePart(t, part)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(rv)), nil
}

/*
The constructor returns a NewDatePartStr with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *DatePartStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDatePartStr(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// DateTruncMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_TRUNC_MILLIS(expr, part).
It truncates UNIX timestamp so that the given date part string
is the least significant. DateTruncMillis is a struct that
implements BinaryFunctionBase.
*/
type DateTruncMillis struct {
	BinaryFunctionBase
}

/*
The function NewDateTruncMillis calls NewBinaryFunctionBase to
create a function named DATE_TRUNC_MILLIS with the two
expressions as input.
*/
func NewDateTruncMillis(first, second Expression) Function {
	rv := &DateTruncMillis{
		*NewBinaryFunctionBase("date_trunc_millis", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DateTruncMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *DateTruncMillis) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *DateTruncMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes inputs date and part as values and returns a value.
If either of these are missing then return a missing value. If date
isnt number or if part isnt a string then return a null value. Call
Actual for these values to convert into valid Go type and cast date to
float64(N1QL valid number type) and part to string. Convert date to Time
format using millisToTime, and then call dateTrunc. Convert it back to
Milliseconds and return its Value.
*/
func (this *DateTruncMillis) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)
	part := second.Actual().(string)
	t := millisToTime(millis)

	var err error
	t, err = dateTrunc(t, part)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

/*
The constructor returns a NewDatePartMillis with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *DateTruncMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateTruncMillis(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// DateTruncStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_TRUNC_STR(expr, part).
It truncates ISO 8601 timestamp so that the given date part
string is the least significant. DateTruncStr is a struct that
implements BinaryFunctionBase.
*/
type DateTruncStr struct {
	BinaryFunctionBase
}

/*
The function NewDateTruncStr calls NewBinaryFunctionBase to
create a function named DATE_TRUNC_STR with the two
expressions as input.
*/
func NewDateTruncStr(first, second Expression) Function {
	rv := &DateTruncStr{
		*NewBinaryFunctionBase("date_trunc_str", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DateTruncStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *DateTruncStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *DateTruncStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes inputs date and part as values and returns a value.
If either of these are missing then return a missing value. If date
and part arent strings then return a null value. Call
Actual for these values to convert into valid Go type and cast both
date and expr into string. Use method strToTime to convert date to
Time format and call the dateTrunc method. Convert it back to a
string and return it in value format.
*/
func (this *DateTruncStr) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	part := second.Actual().(string)
	t, err := strToTime(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	t, err = dateTrunc(t, part)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t, str)), nil
}

/*
The constructor returns a NewDatePartMillis with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *DateTruncStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateTruncStr(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// MillisToStr
//
///////////////////////////////////////////////////

/*
This represents the Date function MILLIS_TO_STR(expr[,fmt]).
The date part as an integer. The date expr is a string in
a supported format, and part is one of the supported date
part strings. Type MillisToStr is a struct that implements
FunctionBase.
*/
type MillisToStr struct {
	FunctionBase
}

/*
The function MillisToStr calls NewFunctionBase to create a
function named MILLIS_TO_STR with input arguments as the
operands from the input expression.
*/
func NewMillisToStr(operands ...Expression) Function {
	rv := &MillisToStr{
		*NewFunctionBase("millis_to_str", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *MillisToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *MillisToStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *MillisToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
Initialize format to default format value. If there is
more than 1 argument then, that is the format. Either the
input argument or the format are missing then return missing.
If the expression is not a number and the format is not a
string then return a null value. Call Actual for these values
to convert into valid Go type and cast the expr to floar64
and the format to string. Convert it to Time format and return
a new stringvalue.
*/
func (this *MillisToStr) Apply(context Context, args ...value.Value) (value.Value, error) {
	ev := args[0]
	fv := _DEFAULT_FMT_VALUE

	if len(args) > 1 {
		fv = args[1]
	}

	if ev.Type() == value.MISSING || fv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if ev.Type() != value.NUMBER || fv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := ev.Actual().(float64)
	fmt := fv.Actual().(string)
	t := millisToTime(millis)
	return value.NewValue(timeToStr(t, fmt)), nil
}

/*
Minimum input arguments required for the defined function
is 1.
*/
func (this *MillisToStr) MinArgs() int { return 1 }

/*
Maximum input arguments required for the defined function
is 2.
*/
func (this *MillisToStr) MaxArgs() int { return 2 }

/*
Returns NewMillisToStr as FunctionConstructor.
*/
func (this *MillisToStr) Constructor() FunctionConstructor { return NewMillisToStr }

///////////////////////////////////////////////////
//
// MillisToUTC
//
///////////////////////////////////////////////////

/*
This represents the Date function MILLIS_TO_UTC(expr [, fmt ]).
It converts the UNIX timestamp to a UTC string in a supported format.
The type MillisToUTC is a struct that implements FunctionBase.
*/
type MillisToUTC struct {
	FunctionBase
}

/*
The function NewMillisToUTC calls NewFunctionBase to create a
function named MILLIS_TO_UTC with input arguments as the
operands from the input expression.
*/
func NewMillisToUTC(operands ...Expression) Function {
	rv := &MillisToUTC{
		*NewFunctionBase("millis_to_utc", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *MillisToUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *MillisToUTC) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *MillisToUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
Initialize format to default format value. If there is
more than 1 argument then, that is the format. Either the
input argument or the format are missing then return missing.
If the expression is not a number and the format is not a
string then return a null value. Call Actual for these values
to convert into valid Go type and cast the expr to floar64
and the format to string. Convert it to Time format, cast to UTC
and reconvert to string, return a new stringvalue.
*/
func (this *MillisToUTC) Apply(context Context, args ...value.Value) (value.Value, error) {
	ev := args[0]
	fv := _DEFAULT_FMT_VALUE

	if len(args) > 1 {
		fv = args[1]
	}

	if ev.Type() == value.MISSING || fv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if ev.Type() != value.NUMBER || fv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := ev.Actual().(float64)
	fmt := fv.Actual().(string)
	t := millisToTime(millis).UTC()
	return value.NewValue(timeToStr(t, fmt)), nil
}

/*
Minimum input arguments required for the defined function
is 1.
*/
func (this *MillisToUTC) MinArgs() int { return 1 }

/*
Maximum input arguments required for the defined function
is 2.
*/
func (this *MillisToUTC) MaxArgs() int { return 2 }

/*
Returns NewMillisToUTC as FunctionConstructor.
*/
func (this *MillisToUTC) Constructor() FunctionConstructor { return NewMillisToUTC }

///////////////////////////////////////////////////
//
// MillisToZoneName
//
///////////////////////////////////////////////////

/*
This represents the Date function
MILLIS_TO_ZONE_NAME(expr, tz_name [, fmt ]). It converts
the UNIX timestamp to a string in the named time zone.
Type MillisToZoneName is a struct that implements
FunctionBase.
*/
type MillisToZoneName struct {
	FunctionBase
}

/*
The function MillisToZoneName calls NewFunctionBase to create a
function named MILLIS_TO_ZONE_NAME with input arguments as the
operands from the input expression.
*/
func NewMillisToZoneName(operands ...Expression) Function {
	rv := &MillisToZoneName{
		*NewFunctionBase("millis_to_zone_name", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *MillisToZoneName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *MillisToZoneName) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *MillisToZoneName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
Initialize format to default format value. If there are
more than 2 arguments then, assign the 3rd args to format. Either the
input arguments or the format are missing then return missing.
If the expression is not a number and the zone and format are not
strings then return a null value. Call Actual for these values
to convert into valid Go type and cast the expr to floar64
and the zone name to string. Use the zone to call LoadLocation
method in the time package that returns the location. Convert it
to Time format in that location and return a new stringvalue by
reconverting it to a string.
*/
func (this *MillisToZoneName) Apply(context Context, args ...value.Value) (value.Value, error) {
	ev := args[0]
	zv := args[1]
	fv := _DEFAULT_FMT_VALUE

	if len(args) > 2 {
		fv = args[2]
	}

	if ev.Type() == value.MISSING || zv.Type() == value.MISSING || fv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if ev.Type() != value.NUMBER || zv.Type() != value.STRING || fv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := ev.Actual().(float64)
	tz := zv.Actual().(string)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	fmt := fv.Actual().(string)
	t := millisToTime(millis).In(loc)
	return value.NewValue(timeToStr(t, fmt)), nil
}

/*
Minimum input arguments required for the defined function
is 2.
*/
func (this *MillisToZoneName) MinArgs() int { return 2 }

/*
Maximum input arguments required for the defined function
is 3.
*/
func (this *MillisToZoneName) MaxArgs() int { return 3 }

/*
Returns NewMillisToZoneName as FunctionConstructor.
*/
func (this *MillisToZoneName) Constructor() FunctionConstructor { return NewMillisToZoneName }

///////////////////////////////////////////////////
//
// NowMillis
//
///////////////////////////////////////////////////
/*
This represents the Date function NOW_MILLIS(). It
returns a statement timestamp as UNIX milliseconds
and does not vary during a query. It is of type
struct and implements a NullaryFunctionBase.
*/
type NowMillis struct {
	NullaryFunctionBase
}

var _NOW_MILLIS = NewNowMillis()

/*
The function NewNowMillis() calls NewNullaryFunctionBase
to create a function named NOW_MILLIS.
*/
func NewNowMillis() Function {
	rv := &NowMillis{
		*NewNullaryFunctionBase("now_millis"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NowMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *NowMillis) Type() value.Type { return value.NUMBER }

/*
This method returns a value that represents the current timestamp
in milliseconds. It gets the current time in nanoseconds using
Now().UnixNano() and then divides it by 10^6 to convert it to
milliseconds. It then calls Newvalue to create a value and returns
it.
*/
func (this *NowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := context.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

/*
Returns the receiver cast to a function to return a FunctionConstructor.
*/
func (this *NowMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _NOW_MILLIS }
}

///////////////////////////////////////////////////
//
// NowStr
//
///////////////////////////////////////////////////

/*
This represents the Date function NOW_STR([fmt]).
It returns a statement timestamp as a string in
a supported format and does not vary during a query.
Type NowStr is a struct that implements FunctionBase.
*/
type NowStr struct {
	FunctionBase
}

/*
The function NewNowStr calls NewFunctionBase to create a
function named NOW_STR with input arguments as the
operands from the input expression.
*/
func NewNowStr(operands ...Expression) Function {
	rv := &NowStr{
		*NewFunctionBase("now_str", operands...),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NowStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *NowStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *NowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
Value() returns the static / constant value of this Expression, or
nil. Expressions that depend on data, clocks, or random numbers must
return nil.
*/
func (this *NowStr) Value() value.Value {
	return nil
}

/*
Initialize format to default format. If there is an agrs then
assign that to the format. If the format is missing, return a
missing value and if it isnt a string then return a null value.
Convert it into valid Go representation and cast to a string.
Get the current timestamp and convert it into the format string
by calling timeToStr using the time and format. Return this value.
*/
func (this *NowStr) Apply(context Context, args ...value.Value) (value.Value, error) {
	fmt := _DEFAULT_FORMAT

	if len(args) > 0 {
		fv := args[0]
		if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.Actual().(string)
	}

	now := context.Now()
	return value.NewValue(timeToStr(now, fmt)), nil
}

/*
Minimum input arguments required for the defined function
is 0.
*/
func (this *NowStr) MinArgs() int { return 0 }

/*
Maximum input arguments required for the defined function
is 1.
*/
func (this *NowStr) MaxArgs() int { return 1 }

/*
Returns NewNowStr as FunctionConstructor.
*/
func (this *NowStr) Constructor() FunctionConstructor { return NewNowStr }

///////////////////////////////////////////////////
//
// StrToMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function STR_TO_MILLIS(expr).
It converts date in a supported format to UNIX milliseconds.
It is of type struct that implements a UnaryFunctionBase.
*/
type StrToMillis struct {
	UnaryFunctionBase
}

/*
The function NewStrToMillis calls NewUnaryFunctionBase to
create a function named STR_TO_MILLIS with the an
expression as input.
*/
func NewStrToMillis(operand Expression) Function {
	rv := &StrToMillis{
		*NewUnaryFunctionBase("str_to_millis", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StrToMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns value type number.
*/
func (this *StrToMillis) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *StrToMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an input argument of type value, and returns a value that is
a timestamp.  If the input argument type is missing, then return missing, and
if it is not a string then return null value. Convert the value to a valid Go type
using Actual and cast it to string. Convert it into a valid time format using
strToTime. Use function timeToMillis to convert to milliseconds and then
return that value.
*/
func (this *StrToMillis) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := arg.Actual().(string)
	t, err := strToTime(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

/*
The constructor returns a NewStrToMillis with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *StrToMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewStrToMillis(operands[0])
	}
}

///////////////////////////////////////////////////
//
// StrToUTC
//
///////////////////////////////////////////////////

/*
This represents the Date function STR_TO_UTC(expr). It
converts the input expression in the ISO 8601 timestamp
to UTC. It is of type struct that implements a
UnaryFunctionBase.
*/
type StrToUTC struct {
	UnaryFunctionBase
}

/*
The function NewStrToUTC calls NewUnaryFunctionBase to
create a function named STR_TO_UTC with the an
expression as input.
*/
func NewStrToUTC(operand Expression) Function {
	rv := &StrToUTC{
		*NewUnaryFunctionBase("str_to_utc", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StrToUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns value type string.
*/
func (this *StrToUTC) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *StrToUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an input argument of type value, and returns a value that is
a timestamp in UTC. If the input argument type is missing, then return missing, and
if it is not a string then return null value. Convert the value to a valid Go type
using Actual and cast it to string. Convert it into a valid time format using
strToTime. Use function UTC() from the time package to set location to UTC,
convert it back to a string and return its N1QL value.
*/
func (this *StrToUTC) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := arg.Actual().(string)
	t, err := strToTime(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	t = t.UTC()
	return value.NewValue(timeToStr(t, str)), nil
}

/*
The constructor returns a NewStrToUTC with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *StrToUTC) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewStrToUTC(operands[0])
	}
}

///////////////////////////////////////////////////
//
// StrToZoneName
//
///////////////////////////////////////////////////

/*
This represents the Date function STR_TO_ZONE_NAME(expr, tz_name).
It converts the supported timestamp string to the named time zone.
StrToZoneName is a struct that implements BinaryFunctionBase.
*/
type StrToZoneName struct {
	BinaryFunctionBase
}

/*
The function NewStrToZoneName calls NewBinaryFunctionBase to
create a function named STR_TO_ZONE_NAME with the two
expressions as input.
*/
func NewStrToZoneName(first, second Expression) Function {
	rv := &StrToZoneName{
		*NewBinaryFunctionBase("str_to_zone_name", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StrToZoneName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *StrToZoneName) Type() value.Type { return value.STRING }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *StrToZoneName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes inputs date and part as string and timezone and returns a value.
If either of these are missing then return a missing value and if they are not string
then return a null value. After converting the string and time zone to valid Go type
convert the string to valid Time using the strToTime function. Use the LoadLocation
method from the time package to return the location with the given name. Convert
the time to string format and return the corresponding stringvalue.
*/
func (this *StrToZoneName) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	t, err := strToTime(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	tz := second.Actual().(string)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t.In(loc), str)), nil
}

/*
The constructor returns a NewStrToZoneName with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *StrToZoneName) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewStrToZoneName(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// DurationToStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DURATION_TO_STR(duration)
It converts a duration in nanoseconds to a string
DurationToStr is a struct that implements UnaryFunctionBase.
*/
type DurationToStr struct {
	UnaryFunctionBase
}

/*
The function NewDurationToStr calls NewUnaryFunctionBase to
create a function named DURATION_TO_STR with the expression
as input.
*/
func NewDurationToStr(first Expression) Function {
	rv := &DurationToStr{
		*NewUnaryFunctionBase("duration_to_str", first),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DurationToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *DurationToStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *DurationToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes a duration and converts it to its string
representation.
If the argument is missing, it returns missing.
If it's not an integer or the conversion fails, it returns null.
*/
func (this *DurationToStr) Apply(context Context, first value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	d := first.Actual().(float64)
	str := time.Duration(d).String()

	return value.NewValue(str), nil
}

/*
The constructor returns a NewDurationToStr with the operand
cast to a Function as the FunctionConstructor.
*/
func (this *DurationToStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDurationToStr(operands[0])
	}
}

///////////////////////////////////////////////////
//
// StrToDuration
//
///////////////////////////////////////////////////

/*
This represents the Date function STR_TO_DURATION(string)
It converts a string to a duration in nanoseconds.
StrToDuration is a struct that implements UnaryFunctionBase.
*/
type StrToDuration struct {
	UnaryFunctionBase
}

/*
The function NewStrToDuration calls NewUnaryFunctionBase to
create a function named STR_TO_DURATION with the expression
as input.
*/
func NewStrToDuration(first Expression) Function {
	rv := &StrToDuration{
		*NewUnaryFunctionBase("str_to_duration", first),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StrToDuration) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *StrToDuration) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *StrToDuration) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes a string representation of a duration and converts it to a
duration.
If the argument is missing, it returns missing.
If it's not a string or the conversion fails, it returns null.
*/
func (this *StrToDuration) Apply(context Context, first value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	d, err := time.ParseDuration(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(d), nil
}

/*
The constructor returns a NewStrToDuration with the operand
cast to a Function as the FunctionConstructor.
*/
func (this *StrToDuration) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewStrToDuration(operands[0])
	}
}

/*
Parse the input string using the defined formats for Date
and return the time value it represents, and error. The
Parse method is defined by the time package.
*/
func strToTime(s string) (time.Time, error) {
	var t time.Time
	var err error
	for _, f := range _DATE_FORMATS {
		t, err = time.ParseInLocation(f, s, time.Local)
		if err == nil {
			return t, nil
		}
	}

	return t, err
}

/*
Parse the input string using the defined formats for Date
and return the time value it represents, the format and an
error. The Parse method is defined by the time package.
*/
func strToTimeFormat(s string) (time.Time, string, error) {
	var t time.Time
	var err error
	for _, f := range _DATE_FORMATS {
		t, err = time.ParseInLocation(f, s, time.Local)
		if err == nil {
			return t, f, nil
		}
	}

	return t, _DEFAULT_FORMAT, err
}

/*
It returns a textual representation of the time value formatted
according to the Format string.
*/
func timeToStr(t time.Time, format string) string {
	_, fmt, _ := strToTimeFormat(format)
	return t.Format(fmt)
}

/*
Convert input milliseconds to time format by multiplying
with 10^6 and using the Unix method from the time package.
*/
func millisToTime(millis float64) time.Time {
	return time.Unix(0, int64(millis*1000000.0))
}

/*
Convert input time to milliseconds from nanoseconds returned
by UnixNano(). Cast it to float64 number and return it.
*/
func timeToMillis(t time.Time) float64 {
	return float64(t.UnixNano() / 1000000)
}

/*
Variable that represents different date formats.
*/
var _DATE_FORMATS = []string{
	"2006-01-02T15:04:05.999Z07:00", // time.RFC3339Milli
	"2006-01-02T15:04:05Z07:00",     // time.RFC3339
	"2006-01-02T15:04:05.999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05.999",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"15:04:05.999Z07:00",
	"15:04:05Z07:00",
	"15:04:05.999",
	"15:04:05",
}

/*
Represents the default format of the time string.
*/
const _DEFAULT_FORMAT = "2006-01-02T15:04:05.999Z07:00"

/*
Represents a value of the default format.
*/
var _DEFAULT_FMT_VALUE = value.NewValue(_DEFAULT_FORMAT)

/*
This function returns the part of the time string that is
depicted by part (for eg. the day, current quarter etc).
*/
func datePart(t time.Time, part string) (int, error) {
	p := strings.ToLower(part)

	switch p {
	case "millennium":
		return (t.Year() / 1000) + 1, nil
	case "century":
		return (t.Year() / 100) + 1, nil
	case "decade":
		return t.Year() / 10, nil
	case "year":
		return t.Year(), nil
	case "quarter":
		return (int(t.Month()) + 2) / 3, nil
	case "month":
		return int(t.Month()), nil
	case "day":
		return t.Day(), nil
	case "hour":
		return t.Hour(), nil
	case "minute":
		return t.Minute(), nil
	case "second":
		return t.Second(), nil
	case "millisecond":
		return t.Nanosecond() / int(time.Millisecond), nil
	case "week":
		return int(math.Ceil(float64(t.YearDay()) / 7.0)), nil
	case "day_of_year", "doy":
		return t.YearDay(), nil
	case "day_of_week", "dow":
		return int(t.Weekday()), nil
	case "iso_week":
		_, w := t.ISOWeek()
		return w, nil
	case "iso_year":
		y, _ := t.ISOWeek()
		return y, nil
	case "iso_dow":
		d := int(t.Weekday())
		if d == 0 {
			d = 7
		}
		return d, nil
	case "timezone":
		_, z := t.Zone()
		return z, nil
	case "timezone_hour":
		_, z := t.Zone()
		return z / (60 * 60), nil
	case "timezone_minute":
		_, z := t.Zone()
		zh := z / (60 * 60)
		z = z - (zh * (60 * 60))
		return z / 60, nil
	default:
		return 0, fmt.Errorf("Unsupported date part %s.", part)
	}
}

/*
Add part to the input time string using AddDate method from the
time package. n and part are used to define the interval or duration.
*/
func dateAdd(t time.Time, n int, part string) (time.Time, error) {
	p := strings.ToLower(part)

	switch p {
	case "millennium":
		return t.AddDate(n*1000, 0, 0), nil
	case "century":
		return t.AddDate(n*100, 0, 0), nil
	case "decade":
		return t.AddDate(n*10, 0, 0), nil
	case "year":
		return t.AddDate(n, 0, 0), nil
	case "quarter":
		return t.AddDate(0, n*3, 0), nil
	case "month":
		return t.AddDate(0, n, 0), nil
	case "week":
		return t.AddDate(0, 0, n*7), nil
	case "day":
		return t.AddDate(0, 0, n), nil
	case "hour":
		return t.Add(time.Duration(n) * time.Hour), nil
	case "minute":
		return t.Add(time.Duration(n) * time.Minute), nil
	case "second":
		return t.Add(time.Duration(n) * time.Second), nil
	case "millisecond":
		return t.Add(time.Duration(n) * time.Millisecond), nil
	default:
		return t, fmt.Errorf("Unsupported date add part %s.", part)
	}
}

/*
Truncate out the part of the date string from the output and return the
remaining time t.
*/
func dateTrunc(t time.Time, part string) (time.Time, error) {
	p := strings.ToLower(part)

	switch p {
	case "millennium":
		t = yearTrunc(t)
		return t.AddDate(-(t.Year() % 1000), 0, 0), nil
	case "century":
		t = yearTrunc(t)
		return t.AddDate(-(t.Year() % 100), 0, 0), nil
	case "decade":
		t = yearTrunc(t)
		return t.AddDate(-(t.Year() % 10), 0, 0), nil
	case "year":
		return yearTrunc(t), nil
	case "quarter":
		t = monthTrunc(t)
		return t.AddDate(0, -((int(t.Month()) - 1) % 3), 0), nil
	case "month":
		return monthTrunc(t), nil
	default:
		return timeTrunc(t, p)
	}
}

/*
This method returns time t that truncates the Day in the
week that year and returns the Time.
*/
func yearTrunc(t time.Time) time.Time {
	t, _ = timeTrunc(t, "day")
	return t.AddDate(0, 0, 1-t.YearDay())
}

/*
This method returns the time t with the day part truncated out. First
get Time part as day. Subtract that from the days and then Add the
given number of years, months and days to t using the AddDate method
from the time package and return.
*/
func monthTrunc(t time.Time) time.Time {
	t, _ = timeTrunc(t, "day")
	return t.AddDate(0, 0, 1-t.Day())
}

/*
Truncate the time string based on the value of the part string.
If type day convert to hours.
*/
func timeTrunc(t time.Time, part string) (time.Time, error) {
	switch part {
	case "day":
		return t.Truncate(time.Duration(24) * time.Hour), nil
	case "hour":
		return t.Truncate(time.Hour), nil
	case "minute":
		return t.Truncate(time.Minute), nil
	case "second":
		return t.Truncate(time.Second), nil
	case "millisecond":
		return t.Truncate(time.Millisecond), nil
	default:
		return t, fmt.Errorf("Unsupported date trunc part %s.", part)
	}
}

/*
This method returns the difference between the two times. Call
diffDates to calculate the difference between the 2 time strings
and then calls diffPart over these strings to unify it into part
format. In the event t2 is greater than t1, and the result returns
a negative value, return a negative result.
*/
func dateDiff(t1, t2 time.Time, part string) (int64, error) {
	var diff *date
	if t1.String() >= t2.String() {
		diff = diffDates(t1, t2)
		return diffPart(t1, t2, diff, part)
	} else {
		diff = diffDates(t2, t1)
		result, e := diffPart(t1, t2, diff, part)
		if result != 0 {
			return -result, e
		}
		return result, e
	}
}

func GetQuarter(t time.Time) int {
	return (int(t.Month()) + 2) / 3
}

/*
This method returns a value specifying a part of the dates specified
by part string. For each type of part (enumerated in the specs) it
computes the value in the type part(for eg. seconds) recursively and
returning it in format (int64) as per the part string.
*/
func diffPart(t1, t2 time.Time, diff *date, part string) (int64, error) {
	p := strings.ToLower(part)

	switch p {
	case "millisecond":
		sec, e := diffPart(t1, t2, diff, "second")
		if e != nil {
			return 0, e
		}
		return (sec * 1000) + int64(diff.millisecond), nil
	case "second":
		min, e := diffPart(t1, t2, diff, "minute")
		if e != nil {
			return 0, e
		}
		return (min * 60) + int64(diff.second), nil
	case "minute":
		hour, e := diffPart(t1, t2, diff, "hour")
		if e != nil {
			return 0, e
		}
		return (hour * 60) + int64(diff.minute), nil
	case "hour":
		day, e := diffPart(t1, t2, diff, "day")
		if e != nil {
			return 0, e
		}
		return (day * 24) + int64(diff.hour), nil
	case "day":
		days := (diff.year * 365) + diff.doy
		if diff.year != 0 {
			days += leapYearsBetween(t1.Year(), t2.Year())
		}
		return int64(days), nil
	case "week":
		day, e := diffPart(t1, t2, diff, "day")
		if e != nil {
			return 0, e
		}
		return day / 7, nil
	case "month":
		diff_month := (int64(t1.Year())*12 + int64(t1.Month())) - (int64(t2.Year())*12 + int64(t2.Month()))
		if diff_month < 0 {
			diff_month = -diff_month
		}
		return diff_month, nil
	case "quarter":
		diff_quarter := (int64(t1.Year())*4 + int64(GetQuarter(t1))) - (int64(t2.Year())*4 + int64(GetQuarter(t2)))

		if diff_quarter < 0 {
			diff_quarter = -diff_quarter
		}
		return diff_quarter, nil
	case "year":
		return int64(diff.year), nil
	case "decade":
		return int64(diff.year) / 10, nil
	case "century":
		return int64(diff.year) / 100, nil
	case "millenium":
		return int64(diff.year) / 1000, nil
	default:
		return 0, fmt.Errorf("Unsupported date diff part %s.", part)
	}
}

/*
This method returns the difference between two dates. The input
arguments to this function are of type Time. We use the setDate
to extract the dates from the time and then compute and return
the difference between the two dates.
*/
func diffDates(t1, t2 time.Time) *date {
	var d1, d2, diff date
	setDate(&d1, t1)
	setDate(&d2, t2)

	diff.millisecond = d1.millisecond - d2.millisecond
	diff.second = d1.second - d2.second
	diff.minute = d1.minute - d2.minute
	diff.hour = d1.hour - d2.hour
	diff.doy = d1.doy - d2.doy
	diff.year = d1.year - d2.year

	return &diff
}

/*
The type date is a structure containing fields year, day in that
year, hour, minute, second and millisecond (all integers).
*/
type date struct {
	year        int
	doy         int
	hour        int
	minute      int
	second      int
	millisecond int
}

/*
This method extracts a date from the input time and sets the
year (as t.year), Day (as t.YearDay), and time (hour, minute,
second using t.Clock, and millisecond as Nanosecond/10^6)
using the input time which is of type Time (defined in package
time).
*/
func setDate(d *date, t time.Time) {
	d.year = t.Year()
	d.doy = t.YearDay()
	d.hour, d.minute, d.second = t.Clock()
	d.millisecond = t.Nanosecond() / 1000000
}

/*
This method computes the number of leap years in
between start and end year, using the method
leapYearsWithin.
*/
func leapYearsBetween(end, start int) int {
	return leapYearsWithin(end) - leapYearsWithin(start)
}

/*
This method returns the number of leap years up until the
input year. This is done using the computation
(year/4) - (year/100) + (year/400).
*/
func leapYearsWithin(year int) int {
	if year > 0 {
		year--
	} else {
		year++
	}

	return (year / 4) - (year / 100) + (year / 400)
}

/*
This method is used to determine if the input year
is a leap year. Leap years can be evenly divided by 4,
and should not be evenly divided by 100, unless it can
be evenly divided by 400.
*/
func isLeapYear(year int) bool {
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}
