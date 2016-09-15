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
// ArrayDateRange
//
///////////////////////////////////////////////////

/*
This represents the Date function ARRAY_DATE_RANGE(expr,expr,part,[n]).
It returns a range of dates from expr1 to expr2. n and part are used to
define an interval and duration.
*/
type ArrayDateRange struct {
	FunctionBase
}

func NewArrayDateRange(operands ...Expression) Function {
	rv := &ArrayDateRange{
		*NewFunctionBase("array_date_range", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayDateRange) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayDateRange) Type() value.Type { return value.ARRAY }

func (this *ArrayDateRange) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayDateRange) Apply(context Context, args ...value.Value) (value.Value, error) {

	// Populate the args
	startDate := args[0]
	endDate := args[1]
	part := args[2]

	// Default value for the increment is 1.
	n := value.ONE_VALUE
	if len(args) > 3 {
		n = args[3]
	}

	// If input arguments are missing then return missing, and if they arent valid types,
	// return null.
	if startDate.Type() == value.MISSING || endDate.Type() == value.MISSING ||
		n.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil

	} else if startDate.Type() != value.STRING || endDate.Type() != value.STRING ||
		n.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	// Convert start date to time format.
	da1 := startDate.Actual().(string)
	t1, fmt1, err := strToTimeFormat(da1)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// Convert end date to time format.
	da2 := endDate.Actual().(string)
	t2, fmt2, err := strToTimeFormat(da2)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// The dates need to be the same format, if not, return null.
	if fmt1 != fmt2 {
		return value.NULL_VALUE, nil
	}

	// Increment
	step := n.Actual().(float64)

	// Return null value for decimal increments.
	if step != math.Trunc(step) {
		return value.NULL_VALUE, nil
	}

	// If the two dates are the same, return an empty array.
	if t1.String() == t2.String() {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	// Date Part
	partStr := part.Actual().(string)

	//Define capacity of the slice using dateDiff
	val, err := dateDiff(t1, t2, partStr)
	if val < 0 {
		val = -val
	}
	if err != nil {
		return value.NULL_VALUE, nil
	}
	rv := make([]interface{}, 0, val)

	// If the start date is after the end date
	if t1.String() > t2.String() {

		// And the increment is positive return empty array. If
		// the increment is negative, so populate the array with
		// decresing dates.
		if step >= 0.0 {
			return value.EMPTY_ARRAY_VALUE, nil
		}
	} else {
		// If end date is after start date but the increment is negative.
		if step < 0.0 {
			return value.EMPTY_ARRAY_VALUE, nil
		}
	}

	// Max date value is end date/ t2.
	// Keep incrementing start date by step for part, and add it to
	// the array to be returned.
	start := t1

	// Populate the array now
	// Until you reach the end date
	for (step > 0.0 && start.String() <= t2.String()) ||
		(step < 0.0 && start.String() >= t2.String()) {
		// Compute the new time
		rv = append(rv, timeToStr(start, fmt1))
		t, err := dateAdd(start, int(step), partStr)
		if err != nil {
			return value.NULL_VALUE, nil
		}

		start = t
	}

	return value.NewValue(rv), nil

}

/*
Minimum input arguments required is 3.
*/
func (this *ArrayDateRange) MinArgs() int { return 3 }

/*
Maximum input arguments allowed is 4.
*/
func (this *ArrayDateRange) MaxArgs() int { return 4 }

/*
Factory method pattern.
*/
func (this *ArrayDateRange) Constructor() FunctionConstructor {
	return NewArrayDateRange
}

///////////////////////////////////////////////////
//
// ClockMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function CLOCK_MILLIS(). It
returns the system clock at function evaluation time,
as UNIX milliseconds and varies during a query.
*/
type ClockMillis struct {
	NullaryFunctionBase
}

var _CLOCK_MILLIS = NewClockMillis()

func NewClockMillis() Function {
	rv := &ClockMillis{
		*NewNullaryFunctionBase("clock_millis"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ClockMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

func (this *ClockMillis) Static() Expression {
	return this
}

/*
Factory method pattern.
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
supported formats.
*/
type ClockStr struct {
	FunctionBase
}

func NewClockStr(operands ...Expression) Function {
	rv := &ClockStr{
		*NewFunctionBase("clock_str", operands...),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ClockStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockStr) Type() value.Type { return value.STRING }

func (this *ClockStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ClockStr) Value() value.Value {
	return nil
}

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
Factory method pattern.
*/
func (this *ClockStr) Constructor() FunctionConstructor {
	return NewClockStr
}

///////////////////////////////////////////////////
//
// DateAddMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_ADD_MILLIS(expr,n,part).
It performs date arithmetic. n and part are used to define an
interval or duration, which is then added (or subtracted) to
the UNIX timestamp, returning the result.
*/
type DateAddMillis struct {
	TernaryFunctionBase
}

func NewDateAddMillis(first, second, third Expression) Function {
	rv := &DateAddMillis{
		*NewTernaryFunctionBase("date_add_millis", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateAddMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateAddMillis) Type() value.Type { return value.NUMBER }

func (this *DateAddMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

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
Factory method pattern.
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
in a supported format, returning the result.
*/
type DateAddStr struct {
	TernaryFunctionBase
}

func NewDateAddStr(first, second, third Expression) Function {
	rv := &DateAddStr{
		*NewTernaryFunctionBase("date_add_str", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateAddStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateAddStr) Type() value.Type { return value.STRING }

func (this *DateAddStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

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
Factory method pattern.
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
UNIX timestamps, as an integer whose unit is part.
*/
type DateDiffMillis struct {
	TernaryFunctionBase
}

func NewDateDiffMillis(first, second, third Expression) Function {
	rv := &DateDiffMillis{
		*NewTernaryFunctionBase("date_diff_millis", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateDiffMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffMillis) Type() value.Type { return value.NUMBER }

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
Factory method pattern.
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
part.
*/
type DateDiffStr struct {
	TernaryFunctionBase
}

func NewDateDiffStr(first, second, third Expression) Function {
	rv := &DateDiffStr{
		*NewTernaryFunctionBase("date_diff_str", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateDiffStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffStr) Type() value.Type { return value.NUMBER }

func (this *DateDiffStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

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
Factory method pattern.
*/
func (this *DateDiffStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateDiffStr(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// DateFormatStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_FORMAT_STR(expr, format).
It returns the input date in the expected format.
*/
type DateFormatStr struct {
	BinaryFunctionBase
}

func NewDateFormatStr(first, second Expression) Function {
	rv := &DateFormatStr{
		*NewBinaryFunctionBase("date_format_str", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateFormatStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateFormatStr) Type() value.Type { return value.STRING }

func (this *DateFormatStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *DateFormatStr) Apply(context Context, first, second value.Value) (value.Value, error) {

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

	format := second.Actual().(string)

	return value.NewValue(timeToStr(t, format)), nil

}

/*
Factory method pattern.
*/
func (this *DateFormatStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateFormatStr(operands[0], operands[1])
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
date part strings.
*/
type DatePartMillis struct {
	BinaryFunctionBase
}

func NewDatePartMillis(first, second Expression) Function {
	rv := &DatePartMillis{
		*NewBinaryFunctionBase("date_part_millis", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DatePartMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DatePartMillis) Type() value.Type { return value.NUMBER }

func (this *DatePartMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

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
Factory method pattern.
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
date part strings.
*/
type DatePartStr struct {
	BinaryFunctionBase
}

func NewDatePartStr(first, second Expression) Function {
	rv := &DatePartStr{
		*NewBinaryFunctionBase("date_part_str", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DatePartStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DatePartStr) Type() value.Type { return value.NUMBER }

func (this *DatePartStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

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
Factory method pattern.
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
is the least significant.
*/
type DateTruncMillis struct {
	BinaryFunctionBase
}

func NewDateTruncMillis(first, second Expression) Function {
	rv := &DateTruncMillis{
		*NewBinaryFunctionBase("date_trunc_millis", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateTruncMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateTruncMillis) Type() value.Type { return value.NUMBER }

func (this *DateTruncMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

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
Factory method pattern.
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
string is the least significant.
*/
type DateTruncStr struct {
	BinaryFunctionBase
}

func NewDateTruncStr(first, second Expression) Function {
	rv := &DateTruncStr{
		*NewBinaryFunctionBase("date_trunc_str", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateTruncStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateTruncStr) Type() value.Type { return value.STRING }

func (this *DateTruncStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

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
Factory method pattern.
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
part strings.
*/
type MillisToStr struct {
	FunctionBase
}

func NewMillisToStr(operands ...Expression) Function {
	rv := &MillisToStr{
		*NewFunctionBase("millis_to_str", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *MillisToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MillisToStr) Type() value.Type { return value.STRING }

func (this *MillisToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

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
Factory method pattern.
*/
func (this *MillisToStr) Constructor() FunctionConstructor {
	return NewMillisToStr
}

///////////////////////////////////////////////////
//
// MillisToUTC
//
///////////////////////////////////////////////////

/*
This represents the Date function MILLIS_TO_UTC(expr [, fmt ]).
It converts the UNIX timestamp to a UTC string in a supported format.
*/
type MillisToUTC struct {
	FunctionBase
}

func NewMillisToUTC(operands ...Expression) Function {
	rv := &MillisToUTC{
		*NewFunctionBase("millis_to_utc", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *MillisToUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MillisToUTC) Type() value.Type { return value.STRING }

func (this *MillisToUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

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
Factory method pattern.
*/
func (this *MillisToUTC) Constructor() FunctionConstructor {
	return NewMillisToUTC
}

///////////////////////////////////////////////////
//
// MillisToZoneName
//
///////////////////////////////////////////////////

/*
This represents the Date function
MILLIS_TO_ZONE_NAME(expr, tz_name [, fmt ]). It converts
the UNIX timestamp to a string in the named time zone.
*/
type MillisToZoneName struct {
	FunctionBase
}

func NewMillisToZoneName(operands ...Expression) Function {
	rv := &MillisToZoneName{
		*NewFunctionBase("millis_to_zone_name", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *MillisToZoneName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MillisToZoneName) Type() value.Type { return value.STRING }

func (this *MillisToZoneName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

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
Factory method pattern.
*/
func (this *MillisToZoneName) Constructor() FunctionConstructor {
	return NewMillisToZoneName
}

///////////////////////////////////////////////////
//
// NowMillis
//
///////////////////////////////////////////////////
/*
This represents the Date function NOW_MILLIS(). It
returns a statement timestamp as UNIX milliseconds
and does not vary during a query.
*/
type NowMillis struct {
	NullaryFunctionBase
}

var _NOW_MILLIS = NewNowMillis()

func NewNowMillis() Function {
	rv := &NowMillis{
		*NewNullaryFunctionBase("now_millis"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NowMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowMillis) Type() value.Type { return value.NUMBER }

func (this *NowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := context.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *NowMillis) Static() Expression {
	return this
}

/*
Factory method pattern.
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
*/
type NowStr struct {
	FunctionBase
}

func NewNowStr(operands ...Expression) Function {
	rv := &NowStr{
		*NewFunctionBase("now_str", operands...),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NowStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowStr) Type() value.Type { return value.STRING }

func (this *NowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *NowStr) Value() value.Value {
	return nil
}

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
Factory method pattern.
*/
func (this *NowStr) Constructor() FunctionConstructor {
	return NewNowStr
}

///////////////////////////////////////////////////
//
// StrToMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function STR_TO_MILLIS(expr).
It converts date in a supported format to UNIX milliseconds.
*/
type StrToMillis struct {
	UnaryFunctionBase
}

func NewStrToMillis(operand Expression) Function {
	rv := &StrToMillis{
		*NewUnaryFunctionBase("str_to_millis", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *StrToMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *StrToMillis) Type() value.Type { return value.NUMBER }

func (this *StrToMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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
Factory method pattern.
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
to UTC.
*/
type StrToUTC struct {
	UnaryFunctionBase
}

func NewStrToUTC(operand Expression) Function {
	rv := &StrToUTC{
		*NewUnaryFunctionBase("str_to_utc", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *StrToUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *StrToUTC) Type() value.Type { return value.STRING }

func (this *StrToUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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
Factory method pattern.
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
*/
type StrToZoneName struct {
	BinaryFunctionBase
}

func NewStrToZoneName(first, second Expression) Function {
	rv := &StrToZoneName{
		*NewBinaryFunctionBase("str_to_zone_name", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *StrToZoneName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *StrToZoneName) Type() value.Type { return value.STRING }

func (this *StrToZoneName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

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
Factory method pattern.
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
*/
type DurationToStr struct {
	UnaryFunctionBase
}

func NewDurationToStr(first Expression) Function {
	rv := &DurationToStr{
		*NewUnaryFunctionBase("duration_to_str", first),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DurationToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DurationToStr) Type() value.Type { return value.STRING }

func (this *DurationToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes a duration and converts it to a string representation.
If the argument is missing, it returns missing.
If it's not a string or the conversion fails, it returns null.
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
Factory method pattern.
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
*/
type StrToDuration struct {
	UnaryFunctionBase
}

func NewStrToDuration(first Expression) Function {
	rv := &StrToDuration{
		*NewUnaryFunctionBase("str_to_duration", first),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *StrToDuration) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *StrToDuration) Type() value.Type { return value.NUMBER }

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
Factory method pattern.
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
	d.millisecond = round(float64(t.Nanosecond()) / 1000000.0)
}

/*
Round input float64 value to int.
*/
func round(f float64) int {
	if math.Abs(f) < 0.5 {
		return 0
	}
	return int(f + math.Copysign(0.5, f))
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
