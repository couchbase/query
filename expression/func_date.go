//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	fmt := ""

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.ToString()
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
	fmt := ""

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
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// Get the timezone and the *Location.
	timeZone := tz.ToString()
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
	fmt := ""

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.ToString()
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
	date, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	n, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
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

	pa := part.ToString()
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
	date, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	n, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if date.Type() == value.MISSING || n.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date.Type() != value.STRING || n.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da := date.ToString()
	t, fmt, err := StrToTimeFormat(da)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	na := n.Actual().(float64)
	if na != math.Trunc(na) {
		return value.NULL_VALUE, nil
	}

	pa := part.ToString()
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
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if date1.Type() == value.MISSING || date2.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date1.Type() != value.NUMBER || date2.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := date1.Actual().(float64)
	da2 := date2.Actual().(float64)
	pa := part.ToString()
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
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if date1.Type() == value.MISSING || date2.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date1.Type() != value.STRING || date2.Type() != value.STRING || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := date1.ToString()
	t1, err := strToTime(da1, "")
	if err != nil {
		return value.NULL_VALUE, nil
	}

	da2 := date2.ToString()
	t2, err := strToTime(da2, "")
	if err != nil {
		return value.NULL_VALUE, nil
	}

	pa := part.ToString()
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
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if date1.Type() == value.MISSING || date2.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date1.Type() != value.NUMBER || date2.Type() != value.NUMBER || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := date1.Actual().(float64)
	da2 := date2.Actual().(float64)
	pa := part.ToString()
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
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if date1.Type() == value.MISSING || date2.Type() == value.MISSING || part.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if date1.Type() != value.STRING || date2.Type() != value.STRING || part.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := date1.ToString()
	t1, err := strToTime(da1, "")
	if err != nil {
		return value.NULL_VALUE, nil
	}

	da2 := date2.ToString()
	t2, err := strToTime(da2, "")
	if err != nil {
		return value.NULL_VALUE, nil
	}

	pa := part.ToString()
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

	str := first.ToString()
	t, err := strToTime(str, "")
	if err != nil {
		return value.NULL_VALUE, nil
	}

	format := second.ToString()

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
	part := second.ToString()

	// Convert the input millis to *Time
	timeVal := millisToTime(millis)

	if timeZone != _NIL_VALUE {
		// Process the timezone component as it isnt nil

		// Get the timezone and the *Location.
		tz := timeZone.ToString()
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

	str := first.ToString()
	t, err := strToTime(str, "")
	if err != nil {
		return value.NULL_VALUE, nil
	}

	part := second.ToString()
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
	da1 := startDate.ToString()
	t1, fmt1, err := StrToTimeFormat(da1)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	// Convert end date to time format.
	da2 := endDate.ToString()
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
	partStr := part.ToString()

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
	partStr := part.ToString()

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
	part := second.ToString()
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

	str := first.ToString()
	part := second.ToString()

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

	return value.NewValue(timeToStr(t, formatFromStr(str)) + tzComponent), nil
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
	fmt := ""
	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	millis := ev.Actual().(float64)
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
	fmt := ""

	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	millis := ev.Actual().(float64)
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
	fmt := ""

	if len(this.operands) > 2 {
		fv, err := this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			null = true
		}
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	millis := ev.Actual().(float64)
	tz := zv.ToString()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return value.NULL_VALUE, nil
	}

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
	fmt := ""

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		fmt = fv.ToString()
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
	fmt := ""
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
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// Get the timezone and the *Location.
	timeZone := tz.ToString()
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
	fmt := ""

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		fmt = fv.ToString()
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
	FunctionBase
}

func NewStrToMillis(operands ...Expression) Function {
	rv := &StrToMillis{
		*NewFunctionBase("str_to_millis", operands...),
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}
	str := arg.ToString()
	var fmt string
	var t time.Time
	if len(this.operands) == 2 {
		arg, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		fmt = arg.ToString()
	}

	t, err = strToTime(str, fmt)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

func (this *StrToMillis) MaxArgs() int { return 2 }
func (this *StrToMillis) MinArgs() int { return 1 }

/*
Factory method pattern.
*/
func (this *StrToMillis) Constructor() FunctionConstructor {
	return NewStrToMillis
}

///////////////////////////////////////////////////
//
// StrToUTC
//
///////////////////////////////////////////////////

/*
This represents the Date function STR_TO_UTC(expr). It
converts the input expression in the given format
to UTC.
*/
type StrToUTC struct {
	FunctionBase
}

func NewStrToUTC(operands ...Expression) Function {
	rv := &StrToUTC{
		*NewFunctionBase("str_to_utc", operands...),
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := arg.ToString()
	var format string
	var t time.Time
	if len(this.operands) == 2 {
		arg, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		format = arg.ToString()
		t, err = strToTime(str, format)
	} else {
		format = formatFromStr(str)
		t, err = strToTime(str, "")
	}

	if err != nil {
		return value.NULL_VALUE, nil
	}

	t = t.UTC()

	return value.NewValue(timeToStr(t, format)), nil
}

func (this *StrToUTC) MaxArgs() int { return 2 }
func (this *StrToUTC) MinArgs() int { return 1 }

func (this *StrToUTC) Constructor() FunctionConstructor {
	return NewStrToUTC
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
	FunctionBase
}

func NewStrToZoneName(operands ...Expression) Function {
	rv := &StrToZoneName{
		*NewFunctionBase("str_to_zone_name", operands...),
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

	str := first.ToString()

	tz := second.ToString()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	format := ""
	var t time.Time
	if len(this.operands) == 3 {
		var arg value.Value
		arg, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		format = arg.ToString()
		t, err = strToTime(str, format)
	} else {
		format = formatFromStr(str)
		t, err = strToTime(str, "")
	}

	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t.In(loc), format)), nil
}

func (this *StrToZoneName) MaxArgs() int { return 3 }
func (this *StrToZoneName) MinArgs() int { return 2 }

func (this *StrToZoneName) Constructor() FunctionConstructor {
	return NewStrToZoneName
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

/*
This method takes a duration and converts it to a string representation.
If the argument is missing, it returns missing.
If it's not a string or the conversion fails, it returns null.
*/
func (this *DurationToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
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

/*
This method takes a string representation of a duration and converts it to a
duration.
If the argument is missing, it returns missing.
If it's not a string or the conversion fails, it returns null.
*/
func (this *StrToDuration) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.ToString()
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
		tz := timeZone.ToString()
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.ToString()
	t, err := strToTime(str, "")
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

// Date string parsing:
// First we try to determine the type of the format string given by interrogating its contents.  The possible formats are example
// dates, common specification (YYYY,MM,DD,..), Unix date-style format (%Y,%m,%d,...) and Go-lang's native format type
// (2006.01,02,...).  Parsing tries to be flexible where possible to allow easy ingest of any date source.

type formatType int

const (
	percentFormat formatType = iota // e.g. %Y-%m-%d %H:%M:%S.%N %z
	commonFormat                    // e.g. YYYY-MM-DD HH:mm:ss.s TZD
	goFormat                        // e.g. 2006-01-02 15:04:05.999 -0700
	exampleFormat                   // e.g. 1111-11-11 11:11:11.111 +0000
	defaultFormat                   // = DEFAULT_FORMAT or try the list of default formats
)

func strToTime(s string, format string) (time.Time, error) {
	switch determineFormat(format) {
	case defaultFormat:
		return strToTimeTryAllDefaultFormats(s)
	case percentFormat:
		return strToTimePercentFormat(s, format)
	case commonFormat:
		return strToTimeCommonFormat(s, format)
	case goFormat:
		return strToTimeGoFormat(s, format)
	default:
		return strToTimeExampleFormat(s, format)
	}
}

// find one of the standard formats that parses the format string (which is an example) and use it
func strToTimeExampleFormat(s string, format string) (time.Time, error) {
	var t time.Time
	_, f, err := strToTimeFormatClosest(format, true)
	if err != nil {
		return t, err
	}
	return strToTimeGoFormat(s, f)
}

// Use go's standard formatting (e.g. 2006-01-02 03:04:05.000)
func strToTimeGoFormat(s string, format string) (time.Time, error) {
	return time.ParseInLocation(format, s, time.Local)
}

const (
	padZero int = iota
	padSpace
	noPad
)

/*
Date format *similar* to Unix 'date' command. Notable exceptions are locale-specific formats (we don't have locale internally
curently) and opposite case specification; width specification is ignored too.  Upper case preference modifier is sequestered
to mean case insensitive and a literal space means any literal character (useful when delimiters aren't consistent or there are
portions to be ignored).

Examples:
	format    ...      parses
	%F                 2021-06-25T04:00:00.000+05:30
	%D                 2021-06-25
	%Y %m %d           2021/06/25, 2021-06-25, 2021.06.25, etc.
	%T %N              14:24:37.001002003, 14:24:37,001002003, 14:24:37:001002003, 14:24:37.2, 14:24:37,345, etc.
	%d/%m/%y %-I %^p   25/06/21 4 am, 25/06/21 11 PM
*/
func strToTimePercentFormat(s string, format string) (time.Time, error) {
	var t time.Time
	var century, year, month, day, hour, minute, second, fraction, l, zoneh, zonem int
	var loc *time.Location

	century = -1
	yearSeen := false
	month = -1
	day = -1
	n := 0
	i := 0
	zoneh = -1
	pm := false
	h12 := false
	for i = 0; i < len(format) && n < len(s); i++ {
		if format[i] != '%' {
			if format[i] == ' ' {
				// space matches any character
				n++
			} else if format[i] != s[n] {
				return t, fmt.Errorf("Failed to parse '%c' in date string (found '%c')", format[i], s[n])
			} else {
				n++
			}
		} else if i+1 == len(format) {
			return t, fmt.Errorf("Invalid format: '%s'", format)
		} else {
			i++
			pad := padZero
			preferUpper := false
			if format[i] == '_' {
				pad = padSpace
				i++
			} else if format[i] == '-' {
				pad = noPad
				i++
			} else if format[i] == '0' {
				pad = padZero
				i++
			} else if format[i] == '^' {
				preferUpper = true
				i++
			}
			if i >= len(format) {
				return t, fmt.Errorf("Invalid format: '%s'", format)
			}
			st := i
			for ; unicode.IsDigit(rune(format[i])) && i < len(format); i++ {
			}
			if st < i {
				if i >= len(format) {
					return t, fmt.Errorf("Invalid format: '%s'", format)
				}
				// ignore the width specification
			}
			if format[i] == 'E' || format[i] == 'O' {
				i++
				if i >= len(format) {
					return t, fmt.Errorf("Invalid format: '%s'", format)
				}
			}
			switch format[i] {
			case '%':
				if s[n] != '%' {
					return t, fmt.Errorf("Failed to parse '%c' in date string (found '%c')", format[i], s[n])
				}
			case 'D':
				if n+len(DEFAULT_SHORT_DATE_FORMAT) <= len(s) {
					pt, err := time.ParseInLocation(DEFAULT_SHORT_DATE_FORMAT, s[n:n+len(DEFAULT_SHORT_DATE_FORMAT)], time.Local)
					if err != nil {
						return t, err
					}
					century = pt.Year() / 100
					year = pt.Year() % 100
					month = int(pt.Month())
					day = pt.Day()
					n += len(DEFAULT_SHORT_DATE_FORMAT)
				} else {
					return t, fmt.Errorf("Invalid date string")
				}
			case 'F':
				if n+len(DEFAULT_FORMAT) <= len(s) {
					pt, err := time.ParseInLocation(DEFAULT_FORMAT, s[n:n+len(DEFAULT_FORMAT)], time.Local)
					if err != nil {
						return t, err
					}
					century = pt.Year() / 100
					year = pt.Year() % 100
					month = int(pt.Month())
					day = pt.Day()
					hour = pt.Hour()
					h12 = false
					minute = pt.Minute()
					second = pt.Second()
					fraction = pt.Nanosecond()
					loc = pt.Location()
					n += len(DEFAULT_FORMAT)
				} else {
					return t, fmt.Errorf("Invalid date string")
				}
			case 'Y':
				year, l = gatherNumber(s[n:], 4, pad == padSpace)
				if (pad == noPad && l < 1) || (pad != noPad && l != 4) || year < 0 {
					return t, fmt.Errorf("Invalid year in date string")
				}
				century = year / 100
				year = year % 100
				n += l
			case 'C':
				century, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || century < 0 {
					return t, fmt.Errorf("Invalid century in date string")
				}
				n += l
			case 'y':
				year, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || year < 0 || year > 99 {
					return t, fmt.Errorf("Invalid year (in century) in date string")
				}
				yearSeen = true
				n += l
			case 'm':
				month, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || month < 1 || month > 12 {
					return t, fmt.Errorf("Invalid month in date string")
				}
				n += l
			case 'B':
				var m time.Month
				for m = time.January; m <= time.December; m++ {
					name := m.String()
					if n+len(name) <= len(s) && (s[n:n+len(name)] == name ||
						(preferUpper && strings.ToUpper(s[n:n+len(name)]) == strings.ToUpper(name))) {
						month = int(m)
						n += len(name)
						break
					}
				}
				if m > time.December {
					return t, fmt.Errorf("Invalid month in date string")
				}
			case 'b':
				var m time.Month
				for m = time.January; m <= time.December; m++ {
					name := m.String()[:3]
					if n+3 <= len(s) && (s[n:n+3] == name || (preferUpper && strings.ToUpper(s[n:n+3]) == strings.ToUpper(name))) {
						month = int(m)
						n += 3
						break
					}
				}
				if m > time.December {
					return t, fmt.Errorf("Invalid month in date string")
				}
			case 'd':
				day, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || day < 1 || day > 31 {
					return t, fmt.Errorf("Invalid day in date string")
				}
				n += l
			case 'e':
				day, l = gatherNumber(s[n:], 2, true)
				if l != 2 || day < 1 || day > 31 {
					return t, fmt.Errorf("Invalid day in date string")
				}
				n += l
			case 'A':
				// parse/validate but do nothing with it
				var d time.Weekday
				for d = time.Sunday; d <= time.Saturday; d++ {
					name := d.String()
					if n+len(name) <= len(s) && (s[n:n+len(name)] == name ||
						(preferUpper && strings.ToUpper(s[n:n+len(name)]) == strings.ToUpper(name))) {
						n += len(name)
						break
					}
				}
				if d > time.Saturday {
					return t, fmt.Errorf("Invalid day in date string")
				}
			case 'a':
				// parse/validate but do nothing with it
				var d time.Weekday
				for d = time.Sunday; d <= time.Saturday; d++ {
					name := d.String()[:3]
					if n+3 <= len(s) && (s[n:n+3] == name || (preferUpper && strings.ToUpper(s[n:n+3]) == strings.ToUpper(name))) {
						n += 3
						break
					}
				}
				if d > time.Saturday {
					return t, fmt.Errorf("Invalid day in date string")
				}
			case 'f':
				// gobble valid suffix
				if n+1 < len(s) {
					var suffix string
					if preferUpper {
						suffix = strings.ToLower(s[n : n+2])
					} else {
						suffix = s[n : n+2]
					}
					if suffix == "st" || suffix == "rd" || suffix == "th" || suffix == "nd" {
						n += 2
					}
				}
			case 'H':
				hour, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 0 || hour > 23 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = false
				n += l
			case 'I':
				hour, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 1 || hour > 12 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = true
				n += l
			case 'k':
				hour, l = gatherNumber(s[n:], 2, true)
				if l != 2 || hour < 0 || hour > 23 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = false
				n += l
			case 'l':
				hour, l = gatherNumber(s[n:], 2, true)
				if l != 2 || hour < 0 || hour > 11 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = true
				n += l
			case 'p':
				if n+1 < len(s) && ((s[n] == 'A' || s[n] == 'P') || (preferUpper && (s[n] == 'a' || s[n] == 'p'))) &&
					(s[n+1] == 'M' || (preferUpper && s[n+1] == 'm')) {
					if s[n] == 'P' || s[n] == 'p' {
						pm = true
					} else {
						pm = false
					}
					n += 2
				} else {
					return t, fmt.Errorf("Invalid 12-hour indicator date string")
				}
			case 'P':
				if n+1 < len(s) && ((s[n] == 'a' || s[n] == 'p') || (preferUpper && (s[n] == 'A' || s[n] == 'P'))) &&
					(s[n+1] == 'm' || (preferUpper && s[n+1] == 'M')) {
					if s[n] == 'p' || s[n] == 'P' {
						pm = true
					} else {
						pm = false
					}
					n += 2
				} else {
					return t, fmt.Errorf("Invalid 12-hour indicator date string")
				}
			case 'M':
				minute, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, fmt.Errorf("Invalid minute in date string")
				}
				n += l
			case 'S':
				second, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || second < 0 || second > 59 {
					return t, fmt.Errorf("Invalid second in date string")
				}
				n += l
			case 'R':
				hour, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 0 || hour > 23 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = false
				n += l
				if s[n] == ':' {
					n++
				}
				minute, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, fmt.Errorf("Invalid minute in date string")
				}
				n += l
			case 'T':
				hour, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 0 || hour > 23 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = false
				n += l
				if s[n] == ':' {
					n++
				}
				minute, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, fmt.Errorf("Invalid minute in date string")
				}
				n += l
				if s[n] == ':' {
					n++
				}
				second, l = gatherNumber(s[n:], 2, pad == padSpace)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || second < 0 || second > 59 {
					return t, fmt.Errorf("Invalid second in date string")
				}
				n += l
			case 'N':
				fraction, l = gatherNumber(s[n:], 9, pad == padSpace)
				if l == 0 {
					return t, fmt.Errorf("Invalid fraction in date string")
				}
				// convert to ns
				fraction *= int(math.Pow10(9 - l))
				n += l
			case 'Z':
				fallthrough
			case 'z':
				var err error
				n, zoneh, zonem, loc, err = gatherZone(s, n)
				if err != nil {
					return t, err
				}
			case 's':
				epoch := 0
				epoch, l = gatherNumber(s[n:], 10, pad == padSpace)
				if l == 0 {
					return t, fmt.Errorf("Invalid seconds since epoch")
				}
				e := time.Unix(int64(epoch), 0)
				century = e.Year() / 100
				year = e.Year() % 100
				month = int(e.Month())
				day = e.Day()
				hour = e.Hour()
				minute = e.Minute()
				second = e.Second()
				loc = nil
				zoneh = 0
				zonem = 0
				h12 = false
				n += l
			default:
				return t, fmt.Errorf("Invalid format '%c' (position %d)", format[i], i)
			}
		}
	}

	if i != len(format) || n != len(s) {
		return t, fmt.Errorf("Failed to completely parse date string")
	}

	// only default the century based on the final parsed year value
	if century == -1 && yearSeen {
		if year >= 69 {
			century = 19
		} else {
			century = 20
		}
	}
	if century != -1 {
		year = century*100 + year
	}
	if month == -1 {
		month = int(time.January)
	}
	if day == -1 {
		day = 1
	}
	err := validateMonthAndDay(year, month, day)
	if err != nil {
		return t, err
	}

	if h12 == true {
		if pm == true {
			if hour < 12 {
				hour += 12
			}
		} else if hour == 12 {
			hour = 0
		}
	}

	if loc == nil {
		loc = getLocation(zoneh, zonem)
	}

	t = time.Date(year, time.Month(month), day, hour, minute, second, fraction, loc)
	return t, nil
}

/*
Common date format, e.g. YYYY-MM-DD hh:mm:ss.s

Components are:
YYYY - 4 digit century+year
CC   - 2 digit century (00...99)
YY   - 2 digit year (00...99)
MM   - 2 digit month (01..12)
DD   - 2 digit day-of-month (01...31) (depending on month)
hh   - 2 digit 24-hour hour (00...23)
HH   - 2 digit 12-hour hour (01...12)
mm   - 2 digit minute (00...59)
ss   - 2 digit second (00...59)
s    - up to 9 digit fraction of a second
pp   - 2 character 12-hour cycle indicator (AM/PM)
TZD  - timezone specified as either: Z, +hh:mm:ss (seconds ignored), +hh:mm, +hhmm, +hh, <zone-name>

Spaces match any character else non format characters have to be matched exactly. There is no escape sequence to use components
listed above as literal content (individual parts can be, e.g. a single Y).
*/
func strToTimeCommonFormat(s string, format string) (time.Time, error) {
	var t time.Time
	var century, year, month, day, hour, minute, second, fraction, l, zoneh, zonem int
	var loc *time.Location

	century = -1
	yearSeen := false
	month = -1
	day = -1
	n := 0
	i := 0
	zoneh = -1
	pm := false
	h12 := false
	for i = 0; i < len(format) && n < len(s); i++ {
		if format[i] == ' ' {
			// space matches any character
			n++
		} else if format[i] == 'Y' {
			if i+4 <= len(format) && format[i:i+4] == "YYYY" {
				year, l = gatherNumber(s[n:], 4, false)
				if l != 4 || year < 0 {
					return t, fmt.Errorf("Invalid year in date string")
				}
				century = year / 100
				year = year % 100
				i += 3
			} else if i+1 < len(format) && format[+1] == 'Y' {
				year, l = gatherNumber(s[n:], 2, false)
				if l != 2 || year < 0 || year > 99 {
					return t, fmt.Errorf("Invalid year in date string")
				}
				i++
				yearSeen = true
			} else {
				return t, fmt.Errorf("Invalid format")
			}
			n += l
		} else if i+1 < len(format) && format[i] == format[i+1] && format[i] != 's' {
			i++
			switch format[i] {
			case 'C':
				century, l = gatherNumber(s[n:], 2, false)
				if l != 2 || century < 0 {
					return t, fmt.Errorf("Invalid century in date string")
				}
				n += 2
			case 'M':
				month, l = gatherNumber(s[n:], 2, false)
				if (l != 1 && l != 2) || month < 1 || month > 12 {
					return t, fmt.Errorf("Invalid month in date string")
				}
				n += l
			case 'D':
				day, l = gatherNumber(s[n:], 2, false)
				if (l != 1 && l != 2) || day < 1 || day > 31 {
					return t, fmt.Errorf("Invalid day in date string")
				}
				n += l
			case 'h':
				hour, l = gatherNumber(s[n:], 2, false)
				if (l != 1 && l != 2) || hour < 0 || hour > 23 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = false
				n += l
			case 'H':
				hour, l = gatherNumber(s[n:], 2, false)
				if (l != 1 && l != 2) || hour < 1 || hour > 12 {
					return t, fmt.Errorf("Invalid hour in date string")
				}
				h12 = true
				n += l
			case 'p':
				if n+1 < len(s) && (s[n] == 'p' || s[n] == 'P') {
					pm = true
				} else if n+1 < len(s) && (s[n] == 'a' || s[n] == 'A') {
					pm = false
				} else {
					return t, fmt.Errorf("Invalid 12-hour indicator date string")
				}
				n += 2
			case 'm':
				minute, l = gatherNumber(s[n:], 2, false)
				if (l != 1 && l != 2) || minute < 0 || minute > 59 {
					return t, fmt.Errorf("Invalid minute in date string")
				}
				n += l
			default:
				return t, fmt.Errorf("Invalid format")
			}
		} else if format[i] == 's' {
			j := 0
			for j = 0; i+j < len(format); j++ {
				if format[i+j] != 's' {
					break
				}
			}
			if j == 2 {
				second, l = gatherNumber(s[n:], 2, false)
				if (l != 1 && l != 2) || second < 0 || second > 59 {
					return t, fmt.Errorf("Invalid second in date string")
				}
			} else if j == 1 {
				fraction, l = gatherNumber(s[n:], 9, false)
				if l == 0 {
					return t, fmt.Errorf("Invalid fraction in date string")
				}
				// convert to ns
				fraction *= int(math.Pow10(9 - l))
			} else {
				return t, fmt.Errorf("Invalid format")
			}
			i += j - 1
			n += l
		} else if i+2 < len(format) && format[i:i+3] == "TZD" {
			var err error
			n, zoneh, zonem, loc, err = gatherZone(s, n)
			if err != nil {
				return t, err
			}
			i += 2
		} else {
			if format[i] != s[n] {
				return t, fmt.Errorf("Failed to parse '%c' in date string (found '%c')", format[i], s[n])
			}
			n++
		}
	}

	if i != len(format) || n != len(s) {
		return t, fmt.Errorf("Failed to completely parse date string")
	}

	// only default the century based on the final parsed year value
	if century == -1 && yearSeen {
		if year >= 69 {
			century = 19
		} else {
			century = 20
		}
	}
	if century != -1 {
		year = century*100 + year
	}
	if month == -1 {
		month = int(time.January)
	}
	if day == -1 {
		day = 1
	}
	err := validateMonthAndDay(year, month, day)
	if err != nil {
		return t, err
	}

	if h12 == true {
		if pm == true {
			if hour < 12 {
				hour += 12
			}
		} else if hour == 12 {
			hour = 0
		}
	}

	if loc == nil {
		loc = getLocation(zoneh, zonem)
	}

	t = time.Date(year, time.Month(month), day, hour, minute, second, fraction, loc)
	return t, nil
}

// Determine the type of format string based on the content
func determineFormat(fmt string) formatType {
	tf := strings.TrimSpace(fmt)
	if len(tf) == 0 {
		return defaultFormat
	} else if strings.IndexAny(tf, "%") != -1 {
		return percentFormat
	} else if strings.IndexAny(tf, "0123456789") == -1 {
		return commonFormat
	} else if !unicode.IsDigit(rune(tf[0])) { // standard formats all start with a digit
		return goFormat
	}
	i := 0
	for i = 0; i < len(tf); i++ {
		if !unicode.IsDigit(rune(tf[i])) {
			break
		}
	}
	n := tf[0:i]
	if n == "2006" {
		return goFormat
	} else if len(n) < 3 {
		a := make([]rune, 2)
		a[0] = '0'
		for i := 1; i < 7; i++ {
			a[1] = rune('0' + i)
			if n == string(a) || n == string(a[1:]) {
				return goFormat
			}
		}
	}
	return exampleFormat
}

func gatherNumber(s string, max int, countLeadingSpaces bool) (int, int) {
	i := 0
	st := 0
	leading := true
	for i = 0; i < len(s) && i < max; i++ {
		if leading && countLeadingSpaces && unicode.IsSpace(rune(s[i])) {
			st++
			continue
		}
		if !unicode.IsDigit(rune(s[i])) {
			break
		}
		leading = false
	}
	if i == 0 || leading {
		return 0, 0
	}
	r, _ := strconv.Atoi(s[st:i])
	return r, i
}

// try parse ISO-8601 time zone formats, or load location by name
func gatherZone(s string, n int) (int, int, int, *time.Location, error) {
	var err error
	var zoneh, zonem int
	var loc *time.Location
	if s[n] == 'Z' {
		zoneh = 0
		zonem = 0
		loc = nil
		n++
	} else if n+8 < len(s) && s[n+3] == ':' && s[n+6] == ':' && (s[n] == '+' || s[n] == '-') && // +00:00:00
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+4])) && unicode.IsDigit(rune(s[n+5])) &&
		unicode.IsDigit(rune(s[n+7])) && unicode.IsDigit(rune(s[n+8])) {
		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+4 : n+6])
		// seconds are ignored as aren't ISO-8601
		loc = nil
		n += 9
	} else if n+5 < len(s) && s[n+3] == ':' && (s[n] == '+' || s[n] == '-') && // +00:00
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+4])) && unicode.IsDigit(rune(s[n+5])) {
		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+4 : n+6])
		loc = nil
		n += 6
	} else if n+4 < len(s) && (s[n] == '+' || s[n] == '-') && // +0000
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+3])) && unicode.IsDigit(rune(s[n+4])) {
		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+3 : n+5])
		loc = nil
		n += 5
	} else if n+2 < len(s) && (s[n] == '+' || s[n] == '-') && // +00
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) {
		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem = 0
		loc = nil
		n += 3
	} else {
		f := strings.FieldsFunc(s[n:], nonIANATZDBRune)
		loc, err = time.LoadLocation(f[0])
		if err != nil {
			err = fmt.Errorf("Invalid time zone in date string")
		} else {
			n += len(f[0])
		}
	}
	return n, zoneh, zonem, loc, err
}

func nonIANATZDBRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == '_' || r == '+' || r == '-' {
		return false
	}
	return true
}

// Make sure YMD specification makes sense
func validateMonthAndDay(year int, month int, day int) error {
	if month < 1 || month > 12 {
		return fmt.Errorf("Invalid month in date string")
	}
	if day < 1 {
		return fmt.Errorf("Invalid day in date string")
	}
	if month == int(time.February) {
		if isLeapYear(year) {
			if day > 29 {
				return fmt.Errorf("Invalid day in date string")
			}
		} else if day > 28 {
			return fmt.Errorf("Invalid day in date string")
		}
	} else if month == int(time.April) || month == int(time.June) || month == int(time.September) || month == int(time.November) {
		if day > 30 {
			return fmt.Errorf("Invalid day in date string")
		}
	} else {
		if day > 31 {
			return fmt.Errorf("Invalid day in date string")
		}
	}
	return nil
}

// wrapper for loading a fixed location from a seconds-offset
func getLocation(h int, m int) *time.Location {
	if h == -1 {
		return time.Local
	}
	return time.FixedZone(fmt.Sprintf("%+03d%02d", h, m), h*60*60+m*60)
}

/*
Parse the input string using the defined formats for Date and return the time value it represents, and error.
Pick the first one that successfully parses preferring formats that exactly match the length over those with optional components.
(Optional components are handled by the time package API.)
*/

func strToTimeTryAllDefaultFormats(s string) (time.Time, error) {
	var t time.Time
	var err error
	// first pass try formats that match length before encountering the overhead of parsing all
	for _, f := range _DATE_FORMATS {
		if len(f) == len(s) {
			t, err = time.ParseInLocation(f, s, time.Local)
			if err == nil {
				return t, nil
			}
		}
	}
	// only check formats we've not checked above
	for _, f := range _DATE_FORMATS {
		if len(f) != len(s) {
			t, err = time.ParseInLocation(f, s, time.Local)
			if err == nil {
				return t, nil
			}
		}
	}

	return t, err
}

func strToTimeforTrunc(s string) (time.Time, string, error) {
	var t time.Time
	var err error
	var f string
	newloc, _ := time.LoadLocation("UTC")
	// first pass try formats that match length before encountering the overhead of parsing all
	for _, f = range _DATE_FORMATS {
		if len(f) == len(s) {
			// Check if the format has a timezone
			t, err = time.ParseInLocation(f, s, newloc)
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		// only check formats we've not checked above
		for _, f = range _DATE_FORMATS {
			if len(f) != len(s) {
				// Check if the format has a timezone
				t, err = time.ParseInLocation(f, s, newloc)
				if err == nil {
					break
				}
			}
		}
	}

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
	return t, "", err
}

/*
Parse the input string using the defined formats for Date and return the time value it represents, and error.
Pick the first one that successfully parses preferring formats that exactly match the length over those with optional components.
(Optional components are handled by the time package API.)

If an exact-length format can't be founf, This tries all remaining and the one closest in length to the input string is picked in
an effort to improve the selection especially when using an example to identify a format to use (some formats have components
which are optional when parsing but present when formatting).
*/

func StrToTimeFormat(s string) (time.Time, string, error) {
	return strToTimeFormatClosest(s, false)
}

func strToTimeFormatClosest(s string, nearestFormat bool) (time.Time, string, error) {
	var t, rt time.Time
	var rf string
	var closest int
	var err error

	// first pass try formats that match length before encountering the overhead of parsing all
	for _, f := range _DATE_FORMATS {
		if len(f) == len(s) {
			t, err = time.ParseInLocation(f, s, time.Local)
			if err == nil {
				return t, f, nil
			}
		}
	}
	// only check formats we've not checked above
	closest = math.MaxInt32
	for _, f := range _DATE_FORMATS {
		if len(f) != len(s) {
			t, err = time.ParseInLocation(f, s, time.Local)
			if err == nil {
				if !nearestFormat {
					return t, f, nil
				}
				l := len(s) - len(f)
				if l < 0 {
					l = l * -1
				}
				if l < closest {
					rf = f
					rt = t
					closest = l
				}
			}
		}
	}

	if closest < math.MaxInt32 {
		return rt, rf, nil
	}

	return t, DEFAULT_FORMAT, err
}

// Date string formatting:
// Returns a textual representation of the time value formatted according to the format string.
func timeToStr(t time.Time, format string) string {
	switch determineFormat(format) {
	case defaultFormat:
		return timeToStrGoFormat(t, DEFAULT_FORMAT)
	case percentFormat:
		return timeToStrPercentFormat(t, format)
	case commonFormat:
		return timeToStrCommonFormat(t, format)
	case goFormat:
		return timeToStrGoFormat(t, format)
	default:
		return timeToStrExampleFormat(t, format)
	}
}

// format string is go standard (e.g. 2006-01-02 15:04:05)
func timeToStrGoFormat(t time.Time, format string) string {
	return t.Format(format)
}

// find a default format that parses the example given in the format string and use that to format the result
func timeToStrExampleFormat(t time.Time, format string) string {
	_, f, _ := strToTimeFormatClosest(format, true)
	return timeToStrGoFormat(t, f)
}

/*
format using Unix date-like format string (e.g. %Y-%m-%d %H:%M:%S.%N)

Examples:
	format    ...      produces
	%F                 2021-06-25T04:00:00.000+05:30
	%D                 2021-06-25
	%Y.%m.%d           2021.06.25
	%T,%n              14:24:37,345
	%d/%m/%y %-I %^p   25/06/21 4 AM
	[%_3S]             [ 37]
	%Z                 BST
*/

func timeToStrPercentFormat(t time.Time, format string) string {
	res := make([]rune, 0, len(format)*3)
	i := 0
	for i = 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			i++
			pad := padZero
			preferUpper := false
			if format[i] == '_' {
				pad = padSpace
				i++
			} else if format[i] == '-' {
				pad = noPad
				i++
			} else if format[i] == '0' {
				pad = padZero
				i++
			} else if format[i] == '^' {
				preferUpper = true
				i++
			}
			if i >= len(format) {
				return fmt.Sprintf("!(Invalid format: '%s')", format)
			}
			width := 0
			st := i
			for ; unicode.IsDigit(rune(format[i])) && i < len(format); i++ {
			}
			if st < i {
				if i >= len(format) {
					return fmt.Sprintf("!(Invalid format: '%s')", format)
				}
				width, _ = strconv.Atoi(format[st:i])
			}
			if format[i] == 'E' || format[i] == 'O' {
				i++
				if i >= len(format) {
					return fmt.Sprintf("!(Invalid format: '%s')", format)
				}
			}
			switch format[i] {
			case 'D':
				res = append(res, []rune(t.Format(DEFAULT_SHORT_DATE_FORMAT))...)
			case 'F':
				res = append(res, []rune(t.Format(DEFAULT_FORMAT))...)
			case 'Y':
				res = append(res, formatInt(width, 4, pad, t.Year())...)
			case 'C':
				res = append(res, formatInt(width, 2, pad, t.Year()/100)...)
			case 'y':
				res = append(res, formatInt(width, 2, pad, t.Year()%100)...)
			case 'm':
				res = append(res, formatInt(width, 2, pad, int(t.Month()))...)
			case 'B':
				if preferUpper {
					res = append(res, []rune(strings.ToUpper(t.Month().String()))...)
				} else {
					res = append(res, []rune(t.Month().String())...)
				}
			case 'b':
				if preferUpper {
					res = append(res, []rune(strings.ToUpper(t.Month().String()[:3]))...)
				} else {
					res = append(res, []rune(t.Month().String()[:3])...)
				}
			case 'd':
				res = append(res, formatInt(width, 2, pad, t.Day())...)
			case 'f':
				if t.Day() == 3 || t.Day() == 23 {
					if preferUpper {
						res = append(res, []rune("RD")...)
					} else {
						res = append(res, []rune("rd")...)
					}
				} else if t.Day() == 1 || t.Day() == 21 {
					if preferUpper {
						res = append(res, []rune("ST")...)
					} else {
						res = append(res, []rune("st")...)
					}
				} else if t.Day() == 2 || t.Day() == 22 {
					if preferUpper {
						res = append(res, []rune("ND")...)
					} else {
						res = append(res, []rune("nd")...)
					}
				} else {
					if preferUpper {
						res = append(res, []rune("TH")...)
					} else {
						res = append(res, []rune("th")...)
					}
				}
			case 'A':
				if preferUpper {
					res = append(res, []rune(strings.ToUpper(t.Weekday().String()))...)
				} else {
					res = append(res, []rune(t.Weekday().String())...)
				}
			case 'a':
				if preferUpper {
					res = append(res, []rune(strings.ToUpper(t.Weekday().String()[:3]))...)
				} else {
					res = append(res, []rune(t.Weekday().String()[:3])...)
				}
			case 'H':
				res = append(res, formatInt(width, 2, pad, t.Hour())...)
			case 'I':
				h := t.Hour()
				if h == 0 {
					h = 12
				} else if h > 12 {
					h -= 12
				}
				res = append(res, formatInt(width, 2, pad, h)...)
			case 'p':
				h := t.Hour()
				if h < 12 {
					res = append(res, []rune("AM")...)
				} else {
					res = append(res, []rune("PM")...)
				}
			case 'P':
				h := t.Hour()
				if h < 12 {
					if preferUpper {
						res = append(res, []rune("AM")...)
					} else {
						res = append(res, []rune("am")...)
					}
				} else {
					if preferUpper {
						res = append(res, []rune("PM")...)
					} else {
						res = append(res, []rune("pm")...)
					}
				}
			case 'M':
				res = append(res, formatInt(width, 2, pad, t.Minute())...)
			case 'S':
				res = append(res, formatInt(width, 2, pad, t.Second())...)
			case 's':
				res = append(res, formatInt(width, 0, pad, int(t.Unix()))...)
			case 'R':
				res = append(res, formatInt(width, 2, pad, t.Hour())...)
				res = append(res, rune(':'))
				res = append(res, formatInt(width, 2, pad, t.Minute())...)
			case 'T':
				res = append(res, formatInt(width, 2, pad, t.Hour())...)
				res = append(res, rune(':'))
				res = append(res, formatInt(width, 2, pad, t.Minute())...)
				res = append(res, rune(':'))
				res = append(res, formatInt(width, 2, pad, t.Second())...)
			case 'N':
				res = append(res, formatInt(width, 9, pad, t.Nanosecond())...)
			case 'n':
				res = append(res, formatInt(width, 3, pad, int(t.Round(time.Millisecond).Nanosecond()/1000000))...)
			case 'z':
				_, off := t.Zone()
				h := off / (60 * 60)
				m := off - (h * 60 * 60)
				m /= 60
				if m < 0 {
					m = m * -1
				}
				res = append(res, []rune(fmt.Sprintf("%+03d%02d", h, m))...)
			case 'Z':
				zone, _ := t.Zone()
				res = append(res, []rune(zone)...)
			case '%':
				res = append(res, rune('%'))
			default:
				res = append(res, []rune(fmt.Sprintf("!(Unknown format: '%c')", format[i]))...)
			}
		} else {
			res = append(res, rune(format[i]))
		}
	}
	return string(res)
}

func formatInt(width int, defWidth int, pad int, val int) []rune {
	if width <= 0 {
		width = defWidth
	}
	f := ""
	switch pad {
	case padSpace:
		f = fmt.Sprintf("%%%dd", width)
	case noPad:
		f = "%d"
	default:
		f = fmt.Sprintf("%%0%dd", width)
	}
	return []rune(fmt.Sprintf(f, val))
}

/*
format using common-style format string (e.g. YYYY-MM-DD HH:mm:ss.s)

Components are:
YYYY - 4 digit century+year
CC   - 2 digit century (00...99)
YY   - 2 digit year (00...99)
MM   - 2 digit month (01..12)
DD   - 2 digit day-of-month (01...31) (depending on month)
hh   - 2 digit 24-hour hour (00...23)
HH   - 2 digit 12-hour hour (01...12)
mm   - 2 digit minute (00...59)
ss   - 2 digit second (00...59)
s    - 3 digit zero-padded milliseconds
sss  - 9 digit zero-padded nanoseconds
PP   - 2 character upper case 12-hour cycle indicator (AM/PM)
pp   - 2 character lower case 12-hour cycle indicator (am/pm)
TZD  - timezone Z or +hh:mm

Other characters/sequences are produced literally in the output.
*/
func timeToStrCommonFormat(t time.Time, format string) string {
	res := make([]rune, 0, len(format)*3)
	i := 0
	for i = 0; i < len(format); i++ {
		if i+3 < len(format) && format[i:i+4] == "YYYY" {
			res = append(res, []rune(fmt.Sprintf("%04d", t.Year()))...)
			i += 3
		} else if i+1 < len(format) && format[i] == format[i+1] && format[i] != 's' {
			i++
			switch format[i] {
			case 'C':
				res = append(res, []rune(fmt.Sprintf("%02d", t.Year()/100))...)
			case 'Y':
				res = append(res, []rune(fmt.Sprintf("%02d", t.Year()%100))...)
			case 'M':
				res = append(res, []rune(fmt.Sprintf("%02d", t.Month()))...)
			case 'D':
				res = append(res, []rune(fmt.Sprintf("%02d", t.Day()))...)
			case 'h':
				res = append(res, []rune(fmt.Sprintf("%02d", t.Hour()))...)
			case 'H':
				h := t.Hour()
				if h == 0 {
					h = 12
				} else if h > 12 {
					h -= 12
				}
				res = append(res, []rune(fmt.Sprintf("%02d", h))...)
			case 'P':
				h := t.Hour()
				if h < 12 {
					res = append(res, []rune("AM")...)
				} else {
					res = append(res, []rune("PM")...)
				}
			case 'p':
				h := t.Hour()
				if h < 12 {
					res = append(res, []rune("am")...)
				} else {
					res = append(res, []rune("pm")...)
				}
			case 'm':
				res = append(res, []rune(fmt.Sprintf("%02d", t.Minute()))...)
			default:
				res = append(res, []rune(fmt.Sprintf("%c%c", format[i], format[i]))...)
			}
		} else if i+2 < len(format) && format[i:i+3] == "TZD" {
			_, off := t.Zone()
			if off == 0 {
				res = append(res, rune('Z'))
			} else {
				h := off / (60 * 60)
				m := (off - (h * 60 * 60))
				m /= 60
				if m < 0 {
					m = m * -1
				}
				res = append(res, []rune(fmt.Sprintf("%+03d:%02d", h, m))...)
			}
			i += 2
		} else if format[i] == 's' {
			n := 0
			for n = 0; n+i < len(format); n++ {
				if format[i+n] != 's' {
					break
				}
			}
			if n == 2 {
				res = append(res, []rune(fmt.Sprintf("%02d", t.Second()))...)
			} else if n == 1 {
				res = append(res, []rune(fmt.Sprintf("%03d", int(t.Round(time.Millisecond).Nanosecond()/1000000)))...)
			} else if n == 3 {
				res = append(res, []rune(fmt.Sprintf("%09d", t.Nanosecond()))...)
			}
			i += n - 1
		} else {
			res = append(res, rune(format[i]))
		}
	}
	return string(res)
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
	"2006-01-02 15:04:05.999Z07:00",
	"2006-01-02T15:04:05.999Z0700",
	"2006-01-02 15:04:05.999Z0700",
	"2006-01-02T15:04:05.999Z07",
	"2006-01-02 15:04:05.999Z07",

	"2006-01-02T15:04:05Z07:00", // time.RFC3339
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02T15:04:05Z0700",
	"2006-01-02 15:04:05Z0700",
	"2006-01-02T15:04:05Z07",
	"2006-01-02 15:04:05Z07",

	"2006-01-02T15:04:05.999",
	"2006-01-02 15:04:05.999",

	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",

	"2006-01-02",
	"15:04:05.999Z07:00",
	"15:04:05Z07:00",
	"15:04:05.000000000",
	"15:04:05.000000",
	"15:04:05.999",
	"15:04:05",
}

/*
Represents the default format of the time string.
*/
const DEFAULT_FORMAT = "2006-01-02T15:04:05.999Z07:00"

// Used solely for the %D format specifier
const DEFAULT_SHORT_DATE_FORMAT = "2006-01-02"

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
	case "week":
		t, _ = timeTrunc(t, "day")
		return t.AddDate(0, 0, -int(t.Weekday())), nil
	case "iso_week": // ISO-8601:  Monday is the first day of the week
		t, _ = timeTrunc(t, "day")
		wd := int(t.Weekday()) - 1
		if wd < 0 {
			wd += 7
		}
		return t.AddDate(0, 0, -wd), nil
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

// Generate a format from the input date that can't be confused with a go-lang native style format
func formatFromStr(str string) string {
	f := append([]rune(nil), []rune(str)...)
	for i, r := range f {
		if unicode.IsDigit(r) {
			f[i] = rune('1')
		}
	}
	return string(f)
}
