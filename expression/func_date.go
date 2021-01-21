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

	"github.com/couchbase/query/errors"
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

	rv.setVolatile()
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
	return value.NewValue(float64(nanos) / 1000000.0), nil
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

	rv.setVolatile()
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
	fmt := DEFAULT_FORMAT

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.Actual().(string)
	}

	return value.NewValue(timeToStr(time.Now(), fmt)), nil
}

func (this *ClockStr) Value() value.Value {
	return nil
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
// ClockTZ
//
///////////////////////////////////////////////////
/*
This represents the Date function CLOCK_TZ(timezone, [ fmt ]).
It returns the system clock at function evaluation time, as a
string in a supported format and input timezone and varies
during a query. There are a set of supported formats.
*/
type ClockTZ struct {
	FunctionBase
}

func NewClockTZ(operands ...Expression) Function {
	rv := &ClockTZ{
		*NewFunctionBase("clock_tz", operands...),
	}

	rv.setVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ClockTZ) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockTZ) Type() value.Type { return value.STRING }

func (this *ClockTZ) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	fmt := DEFAULT_FORMAT

	// Get current time
	timeVal := time.Now()

	tz, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if tz.Type() == value.MISSING {
		missing = true
	} else if tz.Type() != value.STRING {
		null = true
	}

	// Check format
	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
		fmt = fv.Actual().(string)
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// Get the timezone and the *Location.
	timeZone := tz.Actual().(string)
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// Use the timezone to get corresponding time component.
	timeVal = timeVal.In(loc)

	return value.NewValue(timeToStr(timeVal, fmt)), nil
}

func (this *ClockTZ) Value() value.Value {
	return nil
}

/*
Minimum input arguments required for the defined function
CLOCK_TZ is 1.
*/
func (this *ClockTZ) MinArgs() int { return 1 }

/*
Maximum input arguments allowable for the defined function
CLOCK_TZ is 2.
*/
func (this *ClockTZ) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *ClockTZ) Constructor() FunctionConstructor {
	return NewClockTZ
}

///////////////////////////////////////////////////
//
// ClockUTC
//
///////////////////////////////////////////////////
/*
This represents the Date function CLOCK_UTC([ fmt ]). It returns
the system clock at function evaluation time, as a string in a
supported format in UTC and varies during a query. There are a
set of supported formats.
*/
type ClockUTC struct {
	FunctionBase
}

func NewClockUTC(operands ...Expression) Function {
	rv := &ClockUTC{
		*NewFunctionBase("clock_utc", operands...),
	}

	rv.setVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ClockUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockUTC) Type() value.Type { return value.STRING }

func (this *ClockUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	fmt := DEFAULT_FORMAT

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.Actual().(string)
	}

	// Get current time in UTC
	t := time.Now().UTC()

	return value.NewValue(timeToStr(t, fmt)), nil
}

func (this *ClockUTC) Value() value.Value {
	return nil
}

/*
Minimum input arguments required for the defined function
CLOCK_UTC is 0.
*/
func (this *ClockUTC) MinArgs() int { return 0 }

/*
Maximum input arguments allowable for the defined function
CLOCK_UTC is 1.
*/
func (this *ClockUTC) MaxArgs() int { return 1 }

/*
Factory method pattern.
*/
func (this *ClockUTC) Constructor() FunctionConstructor {
	return NewClockUTC
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
		return value.NULL_VALUE, err
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
	t, fmt, err := StrToTimeFormat(da)
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
		return value.NULL_VALUE, err
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
		return value.NULL_VALUE, err
	}

	return value.NewValue(diff), nil
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
		return value.NULL_VALUE, err
	}

	return value.NewValue(diff), nil
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
// DateDiffAbsMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_DIFF_ABS_MILLIS(expr1,expr2,part).
It performs date arithmetic. It returns the absolute elapsed time between two
UNIX timestamps, as an integer whose unit is part. It is always a +ve int.
This is similar to Oracles date diff arithmetic.
*/
type DateDiffAbsMillis struct {
	TernaryFunctionBase
}

func NewDateDiffAbsMillis(first, second, third Expression) Function {
	rv := &DateDiffAbsMillis{
		*NewTernaryFunctionBase("date_diff_abs_millis", first, second, third),
	}

	rv.expr = rv
	return rv
}

func (this *DateDiffAbsMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffAbsMillis) Type() value.Type { return value.NUMBER }

func (this *DateDiffAbsMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *DateDiffAbsMillis) Apply(context Context, date1, date2, part value.Value) (value.Value, error) {
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
		return value.NULL_VALUE, err
	}

	return value.NewValue(math.Abs(float64(diff))), nil
}

func (this *DateDiffAbsMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateDiffAbsMillis(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// DateDiffAbsStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_DIFF_ABS_STR(expr1,expr2,part).
It performs date arithmetic and returns the absolute elapsed time between two
date strings in a supported format, as an integer whose unit is
part. It is always a +ve int.
This is similar to Oracles date diff arithmetic.
*/
type DateDiffAbsStr struct {
	TernaryFunctionBase
}

func NewDateDiffAbsStr(first, second, third Expression) Function {
	rv := &DateDiffAbsStr{
		*NewTernaryFunctionBase("date_diff_abs_str", first, second, third),
	}

	rv.expr = rv
	return rv
}

func (this *DateDiffAbsStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffAbsStr) Type() value.Type { return value.NUMBER }

func (this *DateDiffAbsStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *DateDiffAbsStr) Apply(context Context, date1, date2, part value.Value) (value.Value, error) {
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
		return value.NULL_VALUE, err
	}

	return value.NewValue(math.Abs(float64(diff))), nil
}

func (this *DateDiffAbsStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDateDiffAbsStr(operands[0], operands[1], operands[2])
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

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
This represents the Date function DATE_PART_MILLIS(expr, part, [ tz ]).
It returns the date part as an integer. The date expr is a number
representing UNIX milliseconds, and part is one of the date part
strings.
*/
type DatePartMillis struct {
	FunctionBase
}

func NewDatePartMillis(operands ...Expression) Function {
	rv := &DatePartMillis{
		*NewFunctionBase("date_part_millis", operands...),
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
	null := false
	missing := false

	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() != value.NUMBER {
		null = true
	}

	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		null = true
	}

	// Initialize timezone to nil to avoid processing if not specified.
	timeZone := _NIL_VALUE

	// Check if time zone is set
	if len(this.operands) > 2 {
		timeZone, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if timeZone.Type() == value.MISSING {
			missing = true
		} else if timeZone.Type() != value.STRING {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)
	part := second.Actual().(string)

	// Convert the input millis to *Time
	timeVal := millisToTime(millis)

	if timeZone != _NIL_VALUE {
		// Process the timezone component as it isnt nil

		// Get the timezone and the *Location.
		tz := timeZone.Actual().(string)
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return value.NULL_VALUE, nil
		}
		// Use the timezone to get corresponding time component.
		timeVal = timeVal.In(loc)
	}

	rv, err := datePart(timeVal, part)
	if err != nil {
		return value.NULL_VALUE, err
	}

	return value.NewValue(rv), nil
}

/*
Minimum input arguments required.
*/
func (this *DatePartMillis) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *DatePartMillis) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *DatePartMillis) Constructor() FunctionConstructor {
	return NewDatePartMillis
}

///////////////////////////////////////////////////
//
// DatePartStr
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_PART_STR(expr, part).  It
returns the date part as an integer. The date expr is a string in a
supported format, and part is one of the supported date part strings.
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

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

	part := second.Actual().(string)
	rv, err := datePart(t, part)
	if err != nil {
		return value.NULL_VALUE, err
	}

	return value.NewValue(rv), nil
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
// DateRangeStr
//
///////////////////////////////////////////////////

/*
This represents the Date function ARRAY_DATE_RANGE(expr,expr,part,[n]).
It returns a range of dates from expr1 to expr2. n and part are used to
define an interval and duration.
*/
type DateRangeStr struct {
	FunctionBase
}

func NewDateRangeStr(operands ...Expression) Function {
	rv := &DateRangeStr{
		*NewFunctionBase("date_range_str", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateRangeStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateRangeStr) Type() value.Type { return value.ARRAY }

func (this *DateRangeStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	// Populate the args
	// If input arguments are missing then return missing, and if they arent valid types,
	// return null.
	startDate, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if startDate.Type() == value.MISSING {
		missing = true
	} else if startDate.Type() != value.STRING {
		null = true
	}
	endDate, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if endDate.Type() == value.MISSING {
		missing = true
	} else if endDate.Type() != value.STRING {
		null = true
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		null = true
	}
	// Default value for the increment is 1.
	n := value.ONE_VALUE
	if len(this.operands) > 3 {
		n, err = this.operands[3].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if n.Type() == value.MISSING {
			missing = true
		} else if n.Type() != value.NUMBER {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// Convert start date to time format.
	da1 := startDate.Actual().(string)
	t1, fmt1, err := StrToTimeFormat(da1)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// Convert end date to time format.
	da2 := endDate.Actual().(string)
	t2, _, err := StrToTimeFormat(da2)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// Increment
	step := n.Actual().(float64)

	// Return null value for decimal increments.
	if step != math.Trunc(step) {
		return value.NULL_VALUE, nil
	}

	// If the two dates are the same, return an empty array.
	if t1.Equal(t2) {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	// If the start date is after the end date
	if t1.After(t2) {

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

	// Date Part
	partStr := part.Actual().(string)

	//Define capacity of the slice using dateDiff
	capacity, err := dateDiff(t1, t2, partStr)
	if err != nil {
		return value.NULL_VALUE, err
	}
	if capacity < 0 {
		capacity = -capacity
	}
	if capacity > RANGE_LIMIT {
		return nil, errors.NewRangeError("DATE_RANGE_STR()")
	}

	rv := make([]interface{}, 0, capacity)

	// Max date value is end date/ t2.
	// Keep incrementing start date by step for part, and add it to
	// the array to be returned.
	start := t1
	end := timeToMillis(t2)

	// Populate the array now
	// Until you reach the end date
	for (step > 0.0 && timeToMillis(start) < end) ||
		(step < 0.0 && timeToMillis(start) > end) {
		// Compute the new time
		rv = append(rv, timeToStr(start, fmt1))
		t, err := dateAdd(start, int(step), partStr)
		if err != nil {
			return value.NULL_VALUE, err
		}

		start = t
	}

	return value.NewValue(rv), nil

}

/*
Minimum input arguments required is 3.
*/
func (this *DateRangeStr) MinArgs() int { return 3 }

/*
Maximum input arguments allowed is 4.
*/
func (this *DateRangeStr) MaxArgs() int { return 4 }

/*
Factory method pattern.
*/
func (this *DateRangeStr) Constructor() FunctionConstructor {
	return NewDateRangeStr
}

///////////////////////////////////////////////////
//
// DateRangeMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function DATE_RANGE_MILLIS(expr,expr,part,[n]).
It returns a range of dates from expr1 to expr2 in milliseconds. n and part are used to
define an interval and duration.
*/
type DateRangeMillis struct {
	FunctionBase
}

func NewDateRangeMillis(operands ...Expression) Function {
	rv := &DateRangeMillis{
		*NewFunctionBase("date_range_millis", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DateRangeMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateRangeMillis) Type() value.Type { return value.ARRAY }

func (this *DateRangeMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	// Populate the args
	// If input arguments are missing then return missing, and if they arent valid types,
	// return null.
	startDate, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if startDate.Type() == value.MISSING {
		missing = true
	} else if startDate.Type() != value.NUMBER {
		null = true
	}
	endDate, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if endDate.Type() == value.MISSING {
		missing = true
	} else if endDate.Type() != value.NUMBER {
		null = true
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		null = true
	}
	// Default value for the increment is 1.
	n := value.ONE_VALUE
	if len(this.operands) > 3 {
		n, err = this.operands[3].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if n.Type() == value.MISSING {
			missing = true
		} else if n.Type() != value.NUMBER {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// Convert start date to time format.
	da1 := startDate.Actual().(float64)
	t1 := millisToTime(da1)

	// Convert end date to time format.
	da2 := endDate.Actual().(float64)
	t2 := millisToTime(da2)

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

	// Date Part
	partStr := part.Actual().(string)

	//Define capacity of the slice using dateDiff
	capacity, err := dateDiff(t1, t2, partStr)
	if err != nil {
		return value.NULL_VALUE, err
	}
	if capacity < 0 {
		capacity = -capacity
	}
	if capacity > RANGE_LIMIT {
		return nil, errors.NewRangeError("DATE_RANGE_MILLIS()")
	}

	rv := make([]interface{}, 0, capacity)

	// Max date value is end date/ t2.
	// Keep incrementing start date by step for part, and add it to
	// the array to be returned.
	start := t1
	end := timeToMillis(t2)
	// Populate the array now
	// Until you reach the end date
	for (step > 0.0 && timeToMillis(start) < end) ||
		(step < 0.0 && timeToMillis(start) > end) {
		// Compute the new time
		rv = append(rv, timeToMillis(start))
		t, err := dateAdd(start, int(step), partStr)
		if err != nil {
			return value.NULL_VALUE, err
		}

		start = t
	}

	return value.NewValue(rv), nil

}

/*
Minimum input arguments required is 3.
*/
func (this *DateRangeMillis) MinArgs() int { return 3 }

/*
Maximum input arguments allowed is 4.
*/
func (this *DateRangeMillis) MaxArgs() int { return 4 }

/*
Factory method pattern.
*/
func (this *DateRangeMillis) Constructor() FunctionConstructor {
	return NewDateRangeMillis
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)
	part := second.Actual().(string)
	t := millisToTime(millis)

	t, err = dateTrunc(t, part)
	if err != nil {
		return value.NULL_VALUE, err
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	part := second.Actual().(string)

	// For date trunc we do not consider the timezone.
	// This messes up the result of the golang time functions.
	// To avoid this remove it before processing
	_, tzComponent, _ := strToTimeforTrunc(str)

	if tzComponent != "" {
		str = str[:len(str)-len(tzComponent)]
	}

	t, _, err := strToTimeforTrunc(str)

	if err != nil {
		return value.NULL_VALUE, nil
	}

	t, err = dateTrunc(t, part)
	if err != nil {
		return value.NULL_VALUE, err
	}

	return value.NewValue(timeToStr(t, str) + tzComponent), nil
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
	null := false
	missing := false
	// Populate the args
	// If input arguments are missing then return missing, and if they arent valid types,
	// return null.
	ev, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if ev.Type() == value.MISSING {
		missing = true
	} else if ev.Type() != value.NUMBER {
		null = true
	}
	// Default value for the increment is 1.
	fv := _DEFAULT_FMT_VALUE
	if len(this.operands) > 1 {
		fv, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
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
	null := false
	missing := false
	ev, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if ev.Type() == value.MISSING {
		missing = true
	} else if ev.Type() != value.NUMBER {
		null = true
	}
	fv := _DEFAULT_FMT_VALUE

	if len(this.operands) > 1 {
		fv, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
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
	null := false
	missing := false
	ev, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if ev.Type() == value.MISSING {
		missing = true
	} else if ev.Type() != value.NUMBER {
		null = true
	}
	zv, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if zv.Type() == value.MISSING {
		missing = true
	} else if zv.Type() != value.STRING {
		null = true
	}
	fv := _DEFAULT_FMT_VALUE

	if len(this.operands) > 2 {
		fv, err := this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
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

	rv.setVolatile()
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
	return value.NewValue(float64(nanos) / 1000000.0), nil
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

	rv.setVolatile()
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
	fmt := DEFAULT_FORMAT

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.Actual().(string)
	}

	now := context.Now()
	return value.NewValue(timeToStr(now, fmt)), nil
}

func (this *NowStr) Value() value.Value {
	return nil
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
// NowTz
//
///////////////////////////////////////////////////

/*
This represents the Date function NOW_TZ(timezone, [fmt]).
It returns a statement timestamp as a string in
a supported format for input timezone and does not vary
during a query.
*/
type NowTZ struct {
	FunctionBase
}

func NewNowTZ(operands ...Expression) Function {
	rv := &NowTZ{
		*NewFunctionBase("now_tz", operands...),
	}

	rv.setVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NowTZ) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowTZ) Type() value.Type { return value.STRING }

func (this *NowTZ) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	fmt := DEFAULT_FORMAT
	now := context.Now()

	tz, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if tz.Type() == value.MISSING {
		missing = true
	} else if tz.Type() != value.STRING {
		null = true
	}

	// Check format
	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
		fmt = fv.Actual().(string)
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// Get the timezone and the *Location.
	timeZone := tz.Actual().(string)
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// Use the timezone to get corresponding time component.
	now = now.In(loc)

	return value.NewValue(timeToStr(now, fmt)), nil
}

func (this *NowTZ) Value() value.Value {
	return nil
}

/*
Minimum input arguments required for the defined function
is 1.
*/
func (this *NowTZ) MinArgs() int { return 1 }

/*
Maximum input arguments required for the defined function
is 2.
*/
func (this *NowTZ) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *NowTZ) Constructor() FunctionConstructor {
	return NewNowTZ
}

///////////////////////////////////////////////////
//
// NowUTC
//
///////////////////////////////////////////////////

/*
This represents the Date function NOW_STR([fmt]).
It returns a statement timestamp as a string in
a supported format and does not vary during a query.
*/
type NowUTC struct {
	FunctionBase
}

func NewNowUTC(operands ...Expression) Function {
	rv := &NowUTC{
		*NewFunctionBase("now_utc", operands...),
	}

	rv.setVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NowUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowUTC) Type() value.Type { return value.STRING }

func (this *NowUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	fmt := DEFAULT_FORMAT

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		fmt = fv.Actual().(string)
	}

	now := context.Now().UTC()
	return value.NewValue(timeToStr(now, fmt)), nil
}

func (this *NowUTC) Value() value.Value {
	return nil
}

/*
Minimum input arguments required for the defined function
is 0.
*/
func (this *NowUTC) MinArgs() int { return 0 }

/*
Maximum input arguments required for the defined function
is 1.
*/
func (this *NowUTC) MaxArgs() int { return 1 }

/*
Factory method pattern.
*/
func (this *NowUTC) Constructor() FunctionConstructor {
	return NewNowUTC
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

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

///////////////////////////////////////////////////
//
// WeekdayMillis
//
///////////////////////////////////////////////////

/*
This represents the Date function WEEKDAY_MILLIS(expr, [ tz ]).  It
returns the English name of the weekday as a string. The date expr is
a number representing UNIX milliseconds.
*/
type WeekdayMillis struct {
	FunctionBase
}

func NewWeekdayMillis(operands ...Expression) Function {
	rv := &WeekdayMillis{
		*NewFunctionBase("weekday_millis", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *WeekdayMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *WeekdayMillis) Type() value.Type { return value.STRING }

func (this *WeekdayMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() != value.NUMBER {
		null = true
	}

	// Initialize timezone to nil to avoid processing if not specified.
	timeZone := _NIL_VALUE

	// Check if time zone is set
	if len(this.operands) > 1 {
		timeZone, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if timeZone.Type() == value.MISSING {
			missing = true
		} else if timeZone.Type() != value.STRING {
			null = true
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)

	// Convert the input millis to *Time
	timeVal := millisToTime(millis)

	if timeZone != _NIL_VALUE {
		// Process the timezone component as it isnt nil
		// Get the timezone and the *Location.
		tz := timeZone.Actual().(string)
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return value.NULL_VALUE, nil
		}
		// Use the timezone to get corresponding time component.
		timeVal = timeVal.In(loc)
	}

	dow, err := datePart(timeVal, "day_of_week")
	if err != nil {
		return value.NULL_VALUE, err
	}

	rv := time.Weekday(dow).String()
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required.
*/
func (this *WeekdayMillis) MinArgs() int { return 1 }

/*
Maximum input arguments allowed.
*/
func (this *WeekdayMillis) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *WeekdayMillis) Constructor() FunctionConstructor {
	return NewWeekdayMillis
}

///////////////////////////////////////////////////
//
// WeekdayStr
//
///////////////////////////////////////////////////

/*
This represents the Date function WEEKDAY_STR(expr).  It returns the
English name of the weekday as a string. The date expr is a string in
a supported format.
*/
type WeekdayStr struct {
	UnaryFunctionBase
}

func NewWeekdayStr(first Expression) Function {
	rv := &WeekdayStr{
		*NewUnaryFunctionBase("weekday_str", first),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *WeekdayStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *WeekdayStr) Type() value.Type { return value.STRING }

func (this *WeekdayStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *WeekdayStr) Apply(context Context, first value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	t, err := strToTime(str)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	dow, err := datePart(t, "day_of_week")
	if err != nil {
		return value.NULL_VALUE, err
	}

	rv := time.Weekday(dow).String()
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *WeekdayStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewWeekdayStr(operands[0])
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

func strToTimeforTrunc(s string) (time.Time, string, error) {
	var t time.Time
	var err error
	newloc, _ := time.LoadLocation("UTC")
	for _, f := range _DATE_FORMATS {
		// Check if the format has a timezone
		t, err = time.ParseInLocation(f, s, newloc)
		if err == nil {
			// Calculate the timezone component for input string
			pos := strings.Index(f, "Z")
			tz := ""
			spos := strings.LastIndexAny(s, "Z+-")
			if pos > 0 && spos > 0 {
				tz = s[spos:]
			}
			return t, tz, nil
		}
	}

	return t, "", err
}

/*
Parse the input string using the defined formats for Date
and return the time value it represents, the format and an
error. The Parse method is defined by the time package.
*/
func StrToTimeFormat(s string) (time.Time, string, error) {
	var t time.Time
	var err error
	for _, f := range _DATE_FORMATS {
		t, err = time.ParseInLocation(f, s, time.Local)
		if err == nil {
			return t, f, nil
		}
	}

	return t, DEFAULT_FORMAT, err
}

/*
It returns a textual representation of the time value formatted
according to the Format string.
*/
func timeToStr(t time.Time, format string) string {
	_, fmt, _ := StrToTimeFormat(format)
	return t.Format(fmt)
}

/*
Convert input milliseconds to time format by multiplying
with 10^6 and using the Unix method from the time package.
*/
func millisToTime(millis float64) time.Time {
	return time.Unix(int64(millis/1000), int64(math.Mod(millis, 1000)*1000000.0))
}

/*
Convert input time to milliseconds from nanoseconds returned
by UnixNano().
*/
func timeToMillis(t time.Time) float64 {
	return float64(t.Unix()*1000) + float64(t.Round(time.Millisecond).Nanosecond())/1000000
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
const DEFAULT_FORMAT = "2006-01-02T15:04:05.999Z07:00"

/*
Represents a value of the default format.
*/
var _DEFAULT_FMT_VALUE = value.NewValue(DEFAULT_FORMAT)

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
	sign := 1
	if t1.String() < t2.String() {
		t1, t2 = t2, t1
		sign = -1
	}

	diff := diffDates(t1, t2)
	d, err := diffPart(t1, t2, diff, part)
	return d * int64(sign), err
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
	case "millennium":
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
