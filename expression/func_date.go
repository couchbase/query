//  Copyright 2014-Present Couchbase, Inc.

//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	_MIN_MILLIS = -377705073600000 // -9999-01-01 12:00:00.000000000 UTC (min TZ offset = -12h)
	_MAX_MILLIS = +253402250399999 //  9999-12-31 09:59:59.999000000 UTC (max TZ offset = +14h)
	_NO_ZONE    = -99              // an impossible time zone hour offset
)

///////////////////////////////////////////////////
//
// ClockMillis
//
///////////////////////////////////////////////////

// Return the system clock at function evaluation time as a UNIX milliseconds (varies during a query).
type ClockMillis struct {
	NullaryFunctionBase
}

var _CLOCK_MILLIS = NewClockMillis()

func NewClockMillis() Function {
	rv := &ClockMillis{}
	rv.Init("clock_millis")
	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *ClockMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockMillis) Type() value.Type { return value.NUMBER }

func (this *ClockMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	return value.NewValue(timeToMillis(time.Now())), nil
}

func (this *ClockMillis) Static() Expression {
	return this
}

func (this *ClockMillis) StaticNoVariable() Expression {
	return this
}

func (this *ClockMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _CLOCK_MILLIS }
}

///////////////////////////////////////////////////
//
// ClockStr
//
///////////////////////////////////////////////////

// Return the system clock at function evaluation time as a string (varies during a query).
type ClockStr struct {
	FunctionBase
}

func NewClockStr(operands ...Expression) Function {
	rv := &ClockStr{}
	rv.Init("clock_str", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

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
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, fv))
		}

		fmt = fv.ToString()
	}

	rv, err := timeToStr(time.Now(), fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *ClockStr) Value() value.Value {
	return nil
}

func (this *ClockStr) MinArgs() int { return 0 }

func (this *ClockStr) MaxArgs() int { return 1 }

func (this *ClockStr) Constructor() FunctionConstructor {
	return NewClockStr
}

///////////////////////////////////////////////////
//
// ClockTZ
//
///////////////////////////////////////////////////

// Return the system clock in the specified TZ at function evaluation time as a string (varies during a query).
type ClockTZ struct {
	FunctionBase
}

func NewClockTZ(operands ...Expression) Function {
	rv := &ClockTZ{}
	rv.Init("clock_tz", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *ClockTZ) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockTZ) Type() value.Type { return value.STRING }

func (this *ClockTZ) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	fmt := ""
	var info map[string]interface{}

	// Get current time
	timeVal := time.Now()

	tz, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if tz.Type() == value.MISSING {
		missing = true
	} else if tz.Type() != value.STRING {
		info = invalidArgInfo(0, tz)
	}

	// Check format
	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(1, fv)
			}
		} else {
			fmt = fv.ToString()
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	// Get the timezone and the *Location.
	timeZone := tz.ToString()
	loc, err := loadLocation(timeZone)
	if err != nil {
		return setWarning(context, errors.W_DATE_INVALID_TIMEZONE, timeZone)
	}

	// Use the timezone to get corresponding time component.
	timeVal = timeVal.In(loc)

	rv, err := timeToStr(timeVal, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *ClockTZ) Value() value.Value {
	return nil
}

func (this *ClockTZ) MinArgs() int { return 1 }

func (this *ClockTZ) MaxArgs() int { return 2 }

func (this *ClockTZ) Constructor() FunctionConstructor {
	return NewClockTZ
}

///////////////////////////////////////////////////
//
// ClockUTC
//
///////////////////////////////////////////////////

// Return the system clock in UTC at function evaluation time as a string (varies during a query).
type ClockUTC struct {
	FunctionBase
}

func NewClockUTC(operands ...Expression) Function {
	rv := &ClockUTC{}
	rv.Init("clock_utc", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

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
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, fv))
		}

		fmt = fv.ToString()
	}

	// Get current time in UTC
	t := time.Now().UTC()

	rv, err := timeToStr(t, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *ClockUTC) Value() value.Value {
	return nil
}

func (this *ClockUTC) MinArgs() int { return 0 }

func (this *ClockUTC) MaxArgs() int { return 1 }

func (this *ClockUTC) Constructor() FunctionConstructor {
	return NewClockUTC
}

///////////////////////////////////////////////////
//
// DateAddMillis
//
///////////////////////////////////////////////////

// Perform date arithmetic. n and part are used to define an interval or duration, which is then added (or subtracted) to
// the UNIX timestamp, returning the result.
type DateAddMillis struct {
	TernaryFunctionBase
}

func NewDateAddMillis(first, second, third Expression) Function {
	rv := &DateAddMillis{}
	rv.Init("date_add_millis", first, second, third)

	rv.expr = rv
	return rv
}

func (this *DateAddMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateAddMillis) Type() value.Type { return value.NUMBER }

func (this *DateAddMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	date, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date.Type() == value.MISSING {
		missing = true
	} else if date.Type() == value.NULL {
		null = true
	} else if date.Type() != value.NUMBER {
		info = invalidArgInfo(0, date)
	}
	n, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if n.Type() == value.MISSING {
		missing = true
	} else if n.Type() == value.NULL {
		null = true
	} else if n.Type() != value.NUMBER {
		if info == nil {
			info = invalidArgInfo(1, n)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	da := date.Actual().(float64)
	if da < _MIN_MILLIS || da > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	na := n.Actual().(float64)
	if na != math.Trunc(na) {
		return setWarning(context, errors.W_DATE_NON_INT_VALUE, n)
	}

	pa := part.ToString()
	t, err := dateAdd(millisToTime(da), int(na), pa)
	if err != nil {
		return setWarning(context, err)
	}

	return value.NewValue(timeToMillis(t)), nil
}

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

// Perform date arithmetic. n and part are used to define an interval or duration, which is then added to the date string
// in a supported format, returning the result.
type DateAddStr struct {
	TernaryFunctionBase
}

func NewDateAddStr(first, second, third Expression) Function {
	rv := &DateAddStr{}
	rv.Init("date_add_str", first, second, third)

	rv.expr = rv
	return rv
}

func (this *DateAddStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateAddStr) Type() value.Type { return value.STRING }

func (this *DateAddStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	date, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date.Type() == value.MISSING {
		missing = true
	} else if date.Type() == value.NULL {
		null = true
	} else if date.Type() != value.STRING {
		info = invalidArgInfo(0, date)
	}
	n, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if n.Type() == value.MISSING {
		missing = true
	} else if n.Type() == value.NULL {
		null = true
	} else if n.Type() != value.NUMBER {
		if info == nil {
			info = invalidArgInfo(1, n)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	da := date.ToString()
	t, fmt, err := StrToTimeFormat(da)
	if err != nil {
		return setWarning(context, err)
	}

	na := n.Actual().(float64)
	if na != math.Trunc(na) {
		return setWarning(context, errors.W_DATE_NON_INT_VALUE, n)
	}

	pa := part.ToString()
	t, err = dateAdd(t, int(na), pa)
	if err != nil {
		return setWarning(context, err)
	}
	ms := t.UTC().UnixMilli()
	if ms < _MIN_MILLIS || ms > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	rv, err := timeToStr(t, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

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

// Perform date arithmetic. It returns the elapsed time between two UNIX timestamps, as an integer whose unit is part.
type DateDiffMillis struct {
	TernaryFunctionBase
}

func NewDateDiffMillis(first, second, third Expression) Function {
	rv := &DateDiffMillis{}
	rv.Init("date_diff_millis", first, second, third)

	rv.expr = rv
	return rv
}

func (this *DateDiffMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffMillis) Type() value.Type { return value.NUMBER }

func (this *DateDiffMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date1.Type() == value.MISSING {
		missing = true
	} else if date1.Type() == value.NULL {
		null = true
	} else if date1.Type() != value.NUMBER {
		info = invalidArgInfo(0, date1)
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date2.Type() == value.MISSING {
		missing = true
	} else if date2.Type() == value.NULL {
		null = true
	} else if date2.Type() != value.NUMBER {
		if info == nil {
			info = invalidArgInfo(1, date2)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	da1 := date1.Actual().(float64)
	if da1 < _MIN_MILLIS || da1 > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	da2 := date2.Actual().(float64)
	if da2 < _MIN_MILLIS || da2 > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	pa := part.ToString()
	diff, err := dateDiff(millisToTime(da1), millisToTime(da2), pa)
	if err != nil {
		return setWarning(context, err)
	}

	return value.NewValue(diff), nil
}

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

// Perform date arithmetic and returns the elapsed time between two date strings in a supported format, as an integer whose unit
// is part.
type DateDiffStr struct {
	TernaryFunctionBase
}

func NewDateDiffStr(first, second, third Expression) Function {
	rv := &DateDiffStr{}
	rv.Init("date_diff_str", first, second, third)

	rv.expr = rv
	return rv
}

func (this *DateDiffStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffStr) Type() value.Type { return value.NUMBER }

func (this *DateDiffStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date1.Type() == value.MISSING {
		missing = true
	} else if date1.Type() == value.NULL {
		null = true
	} else if date1.Type() != value.STRING {
		info = invalidArgInfo(0, date1)
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date2.Type() == value.MISSING {
		missing = true
	} else if date2.Type() == value.NULL {
		null = true
	} else if date2.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, date2)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	da1 := date1.ToString()
	t1, err := strToTime(da1, "")
	if err != nil {
		return setWarning(context, err)
	}

	da2 := date2.ToString()
	t2, err := strToTime(da2, "")
	if err != nil {
		return setWarning(context, err)
	}

	pa := part.ToString()
	diff, err := dateDiff(t1, t2, pa)
	if err != nil {
		return setWarning(context, err)
	}

	return value.NewValue(diff), nil
}

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

// Perform date arithmetic. It returns the absolute (always +ve) elapsed time between two UNIX timestamps, as an integer whose
// unit is part.
type DateDiffAbsMillis struct {
	TernaryFunctionBase
}

func NewDateDiffAbsMillis(first, second, third Expression) Function {
	rv := &DateDiffAbsMillis{}
	rv.Init("date_diff_abs_millis", first, second, third)

	rv.expr = rv
	return rv
}

func (this *DateDiffAbsMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffAbsMillis) Type() value.Type { return value.NUMBER }

func (this *DateDiffAbsMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date1.Type() == value.MISSING {
		missing = true
	} else if date1.Type() == value.NULL {
		null = true
	} else if date1.Type() != value.NUMBER {
		info = invalidArgInfo(0, date1)
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date2.Type() == value.MISSING {
		missing = true
	} else if date2.Type() == value.NULL {
		null = true
	} else if date2.Type() != value.NUMBER {
		if info == nil {
			info = invalidArgInfo(1, date2)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	da1 := date1.Actual().(float64)
	if da1 < _MIN_MILLIS || da1 > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	da2 := date2.Actual().(float64)
	if da2 < _MIN_MILLIS || da2 > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	pa := part.ToString()
	diff, err := dateDiff(millisToTime(da1), millisToTime(da2), pa)
	if err != nil {
		return setWarning(context, err)
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

// Perform date arithmetic and returns the absolute (always +ve) elapsed time between two date strings in a supported format,
// as an integer whose unit is part.
type DateDiffAbsStr struct {
	TernaryFunctionBase
}

func NewDateDiffAbsStr(first, second, third Expression) Function {
	rv := &DateDiffAbsStr{}
	rv.Init("date_diff_abs_str", first, second, third)

	rv.expr = rv
	return rv
}

func (this *DateDiffAbsStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateDiffAbsStr) Type() value.Type { return value.NUMBER }

func (this *DateDiffAbsStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	date1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date1.Type() == value.MISSING {
		missing = true
	} else if date1.Type() == value.NULL {
		null = true
	} else if date1.Type() != value.STRING {
		info = invalidArgInfo(0, date1)
	}
	date2, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if date2.Type() == value.MISSING {
		missing = true
	} else if date2.Type() == value.NULL {
		null = true
	} else if date2.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, date2)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	da1 := date1.ToString()
	t1, err := strToTime(da1, "")
	if err != nil {
		return setWarning(context, err)
	}

	da2 := date2.ToString()
	t2, err := strToTime(da2, "")
	if err != nil {
		return setWarning(context, err)
	}

	pa := part.ToString()
	diff, err := dateDiff(t1, t2, pa)
	if err != nil {
		return setWarning(context, err)
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

// Return the input date in the expected format.
type DateFormatStr struct {
	FunctionBase
}

func NewDateFormatStr(operands ...Expression) Function {
	rv := &DateFormatStr{}
	rv.Init("date_format_str", operands...)

	rv.expr = rv
	return rv
}

func (this *DateFormatStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateFormatStr) Type() value.Type { return value.STRING }

func (this *DateFormatStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.STRING {
		info = invalidArgInfo(0, first)
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, second)
		}
	}
	third := second
	if len(this.operands) == 3 {
		third, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if third.Type() == value.MISSING {
			missing = true
		} else if third.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(2, third)
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	str := first.ToString()
	var t time.Time
	if len(this.operands) == 3 {
		t, err = strToTime(str, second.ToString())
	} else {
		t, err = strToTime(str, "")
	}
	if err != nil {
		return setWarning(context, err)
	}

	format := third.ToString()

	rv, err := timeToStr(t, format)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil

}

func (this *DateFormatStr) MinArgs() int { return 2 }

func (this *DateFormatStr) MaxArgs() int { return 3 }

func (this *DateFormatStr) Constructor() FunctionConstructor {
	return NewDateFormatStr
}

///////////////////////////////////////////////////
//
// DatePartMillis
//
///////////////////////////////////////////////////

// Return the date part as an integer. The date expr is a number representing UNIX milliseconds, and part is one of the date part
// strings.
type DatePartMillis struct {
	FunctionBase
}

func NewDatePartMillis(operands ...Expression) Function {
	rv := &DatePartMillis{}
	rv.Init("date_part_millis", operands...)

	rv.expr = rv
	return rv
}

func (this *DatePartMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DatePartMillis) Type() value.Type { return value.NUMBER }

func (this *DatePartMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}

	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.NUMBER {
		info = invalidArgInfo(0, first)
	}

	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, second)
		}
	}

	// Initialize timezone to nil to avoid processing if not specified.
	var timeZone value.Value

	// Check if time zone is set
	if len(this.operands) > 2 {
		timeZone, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if timeZone.Type() == value.MISSING {
			missing = true
		} else if timeZone.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(2, timeZone)
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	millis := first.Actual().(float64)
	if millis < _MIN_MILLIS || millis > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	part := second.ToString()

	// Convert the input millis to *Time
	timeVal := millisToTime(millis)

	if timeZone != nil {
		// Process the timezone component as it isnt nil

		// Get the timezone and the *Location.
		tz := timeZone.ToString()
		loc, err := loadLocation(tz)
		if err != nil {
			return setWarning(context, errors.W_DATE_INVALID_TIMEZONE, tz)
		}
		// Use the timezone to get corresponding time component.
		timeVal = timeVal.In(loc)
	}

	rv, err := datePart(timeVal, part)
	if err != nil {
		return setWarning(context, err)
	}

	return value.NewValue(rv), nil
}

func (this *DatePartMillis) MinArgs() int { return 2 }

func (this *DatePartMillis) MaxArgs() int { return 3 }

func (this *DatePartMillis) Constructor() FunctionConstructor {
	return NewDatePartMillis
}

///////////////////////////////////////////////////
//
// DatePartStr
//
///////////////////////////////////////////////////

// Return the date part as an integer. The date expr is a string in a supported format, and part is one of the supported date part
// strings.
type DatePartStr struct {
	BinaryFunctionBase
}

func NewDatePartStr(first, second Expression) Function {
	rv := &DatePartStr{}
	rv.Init("date_part_str", first, second)

	rv.expr = rv
	return rv
}

func (this *DatePartStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DatePartStr) Type() value.Type { return value.NUMBER }

func (this *DatePartStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.STRING {
		info = invalidArgInfo(0, first)
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, second)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	str := first.ToString()
	t, err := strToTime(str, "")
	if err != nil {
		return setWarning(context, err)
	}

	part := second.ToString()
	rv, err := datePart(t, part)
	if err != nil {
		return setWarning(context, err)
	}

	return value.NewValue(rv), nil
}

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

// Return a range of dates from expr1 to expr2. n and part are used to define an interval and duration.
type DateRangeStr struct {
	FunctionBase
}

func NewDateRangeStr(operands ...Expression) Function {
	rv := &DateRangeStr{}
	rv.Init("date_range_str", operands...)

	rv.expr = rv
	return rv
}

func (this *DateRangeStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateRangeStr) Type() value.Type { return value.ARRAY }

func (this *DateRangeStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	// Populate the args
	// If input arguments are missing then return missing, and if they arent valid types,
	// return null.
	startDate, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if startDate.Type() == value.MISSING {
		missing = true
	} else if startDate.Type() == value.NULL {
		null = true
	} else if startDate.Type() != value.STRING {
		info = invalidArgInfo(0, startDate)
	}
	endDate, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if endDate.Type() == value.MISSING {
		missing = true
	} else if endDate.Type() == value.NULL {
		null = true
	} else if endDate.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, endDate)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	// Default value for the increment is 1.
	n := value.ONE_VALUE
	defaultStep := true
	if len(this.operands) > 3 {
		n, err = this.operands[3].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if n.Type() == value.MISSING {
			missing = true
		} else if n.Type() != value.NUMBER {
			if info == nil {
				info = invalidArgInfo(3, n)
			}
		}
		defaultStep = false
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	// Convert start date to time format.
	da1 := startDate.ToString()
	t1, fmt1, err := StrToTimeFormat(da1)
	if err != nil {
		return setWarning(context, err)
	}

	// Convert end date to time format.
	da2 := endDate.ToString()
	t2, _, err := StrToTimeFormat(da2)
	if err != nil {
		return setWarning(context, err)
	}

	// Increment
	step := n.Actual().(float64)

	// Return null value for decimal increments.
	if step != math.Trunc(step) {
		return setWarning(context, errors.W_DATE_NON_INT_VALUE, n)
	}

	//  Return the start date when the step value is 0.
	var s = int64(step)
	if s == 0 {
		ts, err := timeToStr(t1, fmt1)
		if err != nil {
			return setWarning(context, err)
		}
		setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgValue(3, n))
		return value.NewValue([]interface{}{ts}), nil
	}

	// If the two dates are the same, return an empty array.
	if t1.Equal(t2) {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	// If the start date is after the end date
	if t1.After(t2) {
		if defaultStep {
			step *= -1
		}

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
	capacity = capacity / s

	if err != nil {
		return setWarning(context, err)
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
	if partStr != "calendar_month" {
		for (step > 0.0 && timeToMillis(start) < end) ||
			(step < 0.0 && timeToMillis(start) > end) {
			// Compute the new time
			ts, err := timeToStr(start, fmt1)
			if err != nil {
				return setWarning(context, err)
			}
			rv = append(rv, ts)
			t, err := dateAdd(start, int(step), partStr)
			if err != nil {
				// we could overflow on the last addition so consider that as reaching the end
				e, ok := err.(errors.Error)
				if ok && e.Code() == errors.W_DATE_OVERFLOW {
					break
				}
				return nil, err
			}

			start = t
		}
	} else {
		// Always compute relative to start so as to maintain relationship (last day of month)
		for i := 0; ; i++ {
			t, err := dateAdd(start, int(step)*i, partStr)
			if err != nil {
				// we could overflow on the last addition so consider that as reaching the end
				e, ok := err.(errors.Error)
				if ok && e.Code() == errors.W_DATE_OVERFLOW {
					break
				}
				return nil, err
			}
			if (step > 0.0 && timeToMillis(t) >= end) ||
				(step < 0.0 && timeToMillis(t) <= end) {
				break
			}
			ts, err := timeToStr(t, fmt1)
			if err != nil {
				return setWarning(context, err)
			}
			rv = append(rv, ts)
		}
	}

	return value.NewValue(rv), nil

}

func (this *DateRangeStr) MinArgs() int { return 3 }

func (this *DateRangeStr) MaxArgs() int { return 4 }

func (this *DateRangeStr) Constructor() FunctionConstructor {
	return NewDateRangeStr
}

///////////////////////////////////////////////////
//
// DateRangeMillis
//
///////////////////////////////////////////////////

// Return a range of dates from expr1 to expr2 in milliseconds. n and part are used to define an interval and duration.
type DateRangeMillis struct {
	FunctionBase
}

func NewDateRangeMillis(operands ...Expression) Function {
	rv := &DateRangeMillis{}
	rv.Init("date_range_millis", operands...)

	rv.expr = rv
	return rv
}

func (this *DateRangeMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateRangeMillis) Type() value.Type { return value.ARRAY }

func (this *DateRangeMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	// Populate the args
	// If input arguments are missing then return missing, and if they arent valid types,
	// return null.
	startDate, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if startDate.Type() == value.MISSING {
		missing = true
	} else if startDate.Type() == value.NULL {
		null = true
	} else if startDate.Type() != value.NUMBER {
		info = invalidArgInfo(0, startDate)
	}
	endDate, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if endDate.Type() == value.MISSING {
		missing = true
	} else if endDate.Type() == value.NULL {
		null = true
	} else if endDate.Type() != value.NUMBER {
		if info == nil {
			info = invalidArgInfo(1, endDate)
		}
	}
	part, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if part.Type() == value.MISSING {
		missing = true
	} else if part.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(2, part)
		}
	}
	// Default value for the increment is 1.
	n := value.ONE_VALUE
	defaultStep := true
	if len(this.operands) > 3 {
		n, err = this.operands[3].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if n.Type() == value.MISSING {
			missing = true
		} else if n.Type() != value.NUMBER {
			if info == nil {
				info = invalidArgInfo(3, n)
			}
		}
		defaultStep = false
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	// Convert start date to time format.
	da1 := startDate.Actual().(float64)
	if da1 < _MIN_MILLIS || da1 > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	t1 := millisToTime(da1)

	// Convert end date to time format.
	da2 := endDate.Actual().(float64)
	if da2 < _MIN_MILLIS || da2 > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	t2 := millisToTime(da2)

	// Increment
	step := n.Actual().(float64)

	// Return null value for decimal increments.
	if step != math.Trunc(step) {
		return setWarning(context, errors.W_DATE_NON_INT_VALUE, n)
	}

	//  Return the start date when the step value is 0.
	var s = int64(step)
	if s == 0 {
		ts := timeToMillis(t1)
		return value.NewValue([]interface{}{ts}), nil
	}

	// If the two dates are the same, return an empty array.
	if t1.Equal(t2) {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	// If the start date is after the end date
	if t1.After(t2) {
		if defaultStep {
			step *= -1
		}

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
	capacity = capacity / s
	if err != nil {
		return setWarning(context, err)
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
	if partStr != "calendar_month" {
		for (step > 0.0 && timeToMillis(start) < end) ||
			(step < 0.0 && timeToMillis(start) > end) {
			// Compute the new time
			rv = append(rv, timeToMillis(start))
			t, err := dateAdd(start, int(step), partStr)
			if err != nil {
				// we could overflow on the last addition so consider that as reaching the end
				e, ok := err.(errors.Error)
				if ok && e.Code() == errors.W_DATE_OVERFLOW {
					break
				}
				return nil, err
			}

			start = t
		}
	} else {
		// Always compute relative to start so as to maintain relationship (last day of month)
		for i := 0; ; i++ {
			t, err := dateAdd(start, int(step)*i, partStr)
			if err != nil {
				// we could overflow on the last addition so consider that as reaching the end
				e, ok := err.(errors.Error)
				if ok && e.Code() == errors.W_DATE_OVERFLOW {
					break
				}
				return setWarning(context, err)
			}
			if (step > 0.0 && timeToMillis(t) >= end) ||
				(step < 0.0 && timeToMillis(t) <= end) {
				break
			}
			rv = append(rv, timeToMillis(t))
		}
	}

	return value.NewValue(rv), nil

}

func (this *DateRangeMillis) MinArgs() int { return 3 }

func (this *DateRangeMillis) MaxArgs() int { return 4 }

func (this *DateRangeMillis) Constructor() FunctionConstructor {
	return NewDateRangeMillis
}

///////////////////////////////////////////////////
//
// DateTruncMillis
//
///////////////////////////////////////////////////

// Truncate a UNIX timestamp so that the given date part string is the least significant.
type DateTruncMillis struct {
	BinaryFunctionBase
}

func NewDateTruncMillis(first, second Expression) Function {
	rv := &DateTruncMillis{}
	rv.Init("date_trunc_millis", first, second)

	rv.expr = rv
	return rv
}

func (this *DateTruncMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateTruncMillis) Type() value.Type { return value.NUMBER }

func (this *DateTruncMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.NUMBER {
		info = invalidArgInfo(0, first)
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, second)
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	millis := first.Actual().(float64)
	if millis < _MIN_MILLIS || millis > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	part := second.ToString()
	t := millisToTime(millis).UTC()

	t, err = dateTrunc(t, part)
	if err != nil {
		return nil, err
	}

	return value.NewValue(timeToMillis(t)), nil
}

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

// Truncate an ISO 8601 timestamp so that the given date part string is the least significant.
type DateTruncStr struct {
	FunctionBase
}

func NewDateTruncStr(operands ...Expression) Function {
	rv := &DateTruncStr{}
	rv.Init("date_trunc_str", operands...)

	rv.expr = rv
	return rv
}

func (this *DateTruncStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DateTruncStr) Type() value.Type { return value.STRING }

func (this *DateTruncStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.STRING {
		info = invalidArgInfo(0, first)
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, second)
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	str := first.ToString()
	part := second.ToString()
	format := ""
	if len(this.operands) > 2 {
		arg, err := this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		format = arg.ToString()
	} else {
		format = formatFromStr(str)
	}

	t, err := strToTime(str, format)
	if err != nil {
		return setWarning(context, err)
	}

	t, err = dateTrunc(t, part)
	if err != nil {
		return nil, err
	}

	rv, err := timeToStr(t, format)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *DateTruncStr) Constructor() FunctionConstructor {
	return NewDateTruncStr
}

func (this *DateTruncStr) MinArgs() int { return 2 }
func (this *DateTruncStr) MaxArgs() int { return 3 }

///////////////////////////////////////////////////
//
// MillisToStr
//
///////////////////////////////////////////////////

// Convert a millisecond timestamp to a date string in a supported format.
type MillisToStr struct {
	FunctionBase
}

func NewMillisToStr(operands ...Expression) Function {
	rv := &MillisToStr{}
	rv.Init("millis_to_str", operands...)

	rv.expr = rv
	return rv
}

func (this *MillisToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MillisToStr) Type() value.Type { return value.STRING }

func (this *MillisToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	ev, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if ev.Type() == value.MISSING {
		missing = true
	} else if ev.Type() == value.NULL {
		null = true
	} else if ev.Type() != value.NUMBER {
		info = invalidArgInfo(0, ev)
	}
	fmt := ""
	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(1, fv)
			}
		}
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	millis := ev.Actual().(float64)
	if millis < _MIN_MILLIS || millis > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	rv, err := timeToStr(millisToTime(millis), fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *MillisToStr) MinArgs() int { return 1 }

func (this *MillisToStr) MaxArgs() int { return 2 }

func (this *MillisToStr) Constructor() FunctionConstructor {
	return NewMillisToStr
}

///////////////////////////////////////////////////
//
// MillisToUTC
//
///////////////////////////////////////////////////

// Convert the UNIX timestamp to a UTC string in a supported format.
type MillisToUTC struct {
	FunctionBase
}

func NewMillisToUTC(operands ...Expression) Function {
	rv := &MillisToUTC{}
	rv.Init("millis_to_utc", operands...)

	rv.expr = rv
	return rv
}

func (this *MillisToUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MillisToUTC) Type() value.Type { return value.STRING }

func (this *MillisToUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	ev, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if ev.Type() == value.MISSING {
		missing = true
	} else if ev.Type() == value.NULL {
		null = true
	} else if ev.Type() != value.NUMBER {
		info = invalidArgInfo(0, ev)
	}
	fmt := ""

	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(1, fv)
			}
		}
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	millis := ev.Actual().(float64)
	if millis < _MIN_MILLIS || millis > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	t := millisToTime(millis).UTC()
	rv, err := timeToStr(t, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *MillisToUTC) MinArgs() int { return 1 }

func (this *MillisToUTC) MaxArgs() int { return 2 }

func (this *MillisToUTC) Constructor() FunctionConstructor {
	return NewMillisToUTC
}

///////////////////////////////////////////////////
//
// MillisToZoneName
//
///////////////////////////////////////////////////

// Convert the UNIX timestamp to a string in the named time zone.
type MillisToZoneName struct {
	FunctionBase
}

func NewMillisToZoneName(operands ...Expression) Function {
	rv := &MillisToZoneName{}
	rv.Init("millis_to_zone_name", operands...)

	rv.expr = rv
	return rv
}

func (this *MillisToZoneName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MillisToZoneName) Type() value.Type { return value.STRING }

func (this *MillisToZoneName) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	ev, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if ev.Type() == value.MISSING {
		missing = true
	} else if ev.Type() == value.NULL {
		null = true
	} else if ev.Type() != value.NUMBER {
		info = invalidArgInfo(0, ev)
	}
	zv, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if zv.Type() == value.MISSING {
		missing = true
	} else if zv.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, zv)
		}
	}
	fmt := ""

	if len(this.operands) > 2 {
		fv, err := this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(2, fv)
			}
		}
		fmt = fv.ToString()
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	millis := ev.Actual().(float64)
	if millis < _MIN_MILLIS || millis > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}
	tz := zv.ToString()
	loc, err := loadLocation(tz)
	if err != nil {
		return setWarning(context, errors.W_DATE_INVALID_TIMEZONE, tz)
	}

	t := millisToTime(millis).In(loc)
	rv, err := timeToStr(t, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *MillisToZoneName) MinArgs() int { return 2 }

func (this *MillisToZoneName) MaxArgs() int { return 3 }

func (this *MillisToZoneName) Constructor() FunctionConstructor {
	return NewMillisToZoneName
}

///////////////////////////////////////////////////
//
// NowMillis
//
///////////////////////////////////////////////////

// Return a statement timestamp as UNIX milliseconds and does not vary during a query.
type NowMillis struct {
	NullaryFunctionBase
}

var _NOW_MILLIS = NewNowMillis()

func NewNowMillis() Function {
	rv := &NowMillis{}
	rv.Init("now_millis")
	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *NowMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowMillis) Type() value.Type { return value.NUMBER }

func (this *NowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	return value.NewValue(timeToMillis(context.Now())), nil
}

func (this *NowMillis) Static() Expression {
	return this
}

func (this *NowMillis) StaticNoVariable() Expression {
	return this
}

func (this *NowMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _NOW_MILLIS }
}

///////////////////////////////////////////////////
//
// NowStr
//
///////////////////////////////////////////////////

// Return a statement timestamp as a string in a supported format and does not vary during a query.
type NowStr struct {
	FunctionBase
}

func NewNowStr(operands ...Expression) Function {
	rv := &NowStr{}
	rv.Init("now_str", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *NowStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowStr) Type() value.Type { return value.STRING }

func (this *NowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	fmt := ""

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, fv))
		}

		fmt = fv.ToString()
	}

	now := context.Now()
	rv, err := timeToStr(now, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *NowStr) Value() value.Value {
	return nil
}

func (this *NowStr) MinArgs() int { return 0 }

func (this *NowStr) MaxArgs() int { return 1 }

func (this *NowStr) Constructor() FunctionConstructor {
	return NewNowStr
}

///////////////////////////////////////////////////
//
// NowTz
//
///////////////////////////////////////////////////

// Return a statement timestamp as a string in a supported format for input timezone and does not vary during a query.
type NowTZ struct {
	FunctionBase
}

func NewNowTZ(operands ...Expression) Function {
	rv := &NowTZ{}
	rv.Init("now_tz", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *NowTZ) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowTZ) Type() value.Type { return value.STRING }

func (this *NowTZ) Evaluate(item value.Value, context Context) (value.Value, error) {
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	missing := false
	fmt := ""
	now := context.Now()
	var info map[string]interface{}

	tz, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if tz.Type() == value.MISSING {
		missing = true
	} else if tz.Type() != value.STRING {
		info = invalidArgInfo(0, tz)
	}

	// Check format
	if len(this.operands) > 1 {
		fv, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			missing = true
		} else if fv.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(1, fv)
			}
		} else {
			fmt = fv.ToString()
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	// Get the timezone and the *Location.
	timeZone := tz.ToString()
	loc, err := loadLocation(timeZone)
	if err != nil {
		return setWarning(context, errors.W_DATE_INVALID_TIMEZONE, timeZone)
	}

	// Use the timezone to get corresponding time component.
	now = now.In(loc)

	rv, err := timeToStr(now, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *NowTZ) Value() value.Value {
	return nil
}

func (this *NowTZ) MinArgs() int { return 1 }

func (this *NowTZ) MaxArgs() int { return 2 }

func (this *NowTZ) Constructor() FunctionConstructor {
	return NewNowTZ
}

///////////////////////////////////////////////////
//
// NowUTC
//
///////////////////////////////////////////////////

// Return a statement timestamp as a string in a supported format and does not vary during a query.
type NowUTC struct {
	FunctionBase
}

func NewNowUTC(operands ...Expression) Function {
	rv := &NowUTC{}
	rv.Init("now_utc", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *NowUTC) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowUTC) Type() value.Type { return value.STRING }

func (this *NowUTC) Evaluate(item value.Value, context Context) (value.Value, error) {
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	fmt := ""

	if len(this.operands) > 0 {
		fv, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if fv.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if fv.Type() != value.STRING {
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, fv))
		}
		fmt = fv.ToString()
	}

	now := context.Now().UTC()
	rv, err := timeToStr(now, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *NowUTC) Value() value.Value {
	return nil
}

func (this *NowUTC) MinArgs() int { return 0 }

func (this *NowUTC) MaxArgs() int { return 1 }

func (this *NowUTC) Constructor() FunctionConstructor {
	return NewNowUTC
}

///////////////////////////////////////////////////
//
// StrToMillis
//
///////////////////////////////////////////////////

// Convert a date string in a supported format to UNIX milliseconds.
type StrToMillis struct {
	FunctionBase
}

func NewStrToMillis(operands ...Expression) Function {
	rv := &StrToMillis{}
	rv.Init("str_to_millis", operands...)

	rv.expr = rv
	return rv
}

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
	} else if arg.Type() == value.NULL {
		return value.NULL_VALUE, nil
	} else if arg.Type() != value.STRING {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, arg))
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
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(1, arg))
		}
		fmt = arg.ToString()
	}

	t, err = strToTime(str, fmt)
	if err != nil {
		return setWarning(context, err)
	}
	ms := t.UTC().UnixMilli()
	if ms < _MIN_MILLIS || ms > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	return value.NewValue(timeToMillis(t)), nil
}

func (this *StrToMillis) MaxArgs() int { return 2 }
func (this *StrToMillis) MinArgs() int { return 1 }

func (this *StrToMillis) Constructor() FunctionConstructor {
	return NewStrToMillis
}

///////////////////////////////////////////////////
//
// StrToUTC
//
///////////////////////////////////////////////////

// Convert the input expression in the given format to UTC.
type StrToUTC struct {
	FunctionBase
}

func NewStrToUTC(operands ...Expression) Function {
	rv := &StrToUTC{}
	rv.Init("str_to_utc", operands...)

	rv.expr = rv
	return rv
}

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
	} else if arg.Type() == value.NULL {
		return value.NULL_VALUE, nil
	} else if arg.Type() != value.STRING {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, arg))
	}

	str := arg.ToString()
	var format string
	var outputFormat string
	var t time.Time
	if len(this.operands) > 2 {
		arg, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		outputFormat = arg.ToString()
	}
	if len(this.operands) > 1 {
		arg, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(1, arg))
		}
		format = arg.ToString()
		if len(this.operands) == 2 {
			outputFormat = format
		}
	} else {
		outputFormat = formatFromStr(str)
	}
	t, err = strToTime(str, format)
	if err != nil {
		return setWarning(context, err)
	}

	t = t.UTC()
	ms := t.UnixMilli()
	if ms < _MIN_MILLIS || ms > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	rv, err := timeToStr(t, outputFormat)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *StrToUTC) MaxArgs() int { return 3 }
func (this *StrToUTC) MinArgs() int { return 1 }

func (this *StrToUTC) Constructor() FunctionConstructor {
	return NewStrToUTC
}

///////////////////////////////////////////////////
//
// StrToZoneName
//
///////////////////////////////////////////////////

// Convert the supported timestamp string to the named time zone.
type StrToZoneName struct {
	FunctionBase
}

func NewStrToZoneName(operands ...Expression) Function {
	rv := &StrToZoneName{}
	rv.Init("str_to_zone_name", operands...)

	rv.expr = rv
	return rv
}

func (this *StrToZoneName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *StrToZoneName) Type() value.Type { return value.STRING }

func (this *StrToZoneName) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	var info map[string]interface{}
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.STRING {
		info = invalidArgInfo(0, first)
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if second.Type() == value.MISSING {
		missing = true
	} else if second.Type() != value.STRING {
		if info == nil {
			info = invalidArgInfo(1, second)
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	str := first.ToString()

	tz := second.ToString()
	loc, err := loadLocation(tz)
	if err != nil {
		return setWarning(context, errors.W_DATE_INVALID_TIMEZONE, tz)
	}

	var format string
	var outputFormat string
	var t time.Time
	if len(this.operands) > 3 {
		var arg value.Value
		arg, err = this.operands[3].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		outputFormat = arg.ToString()
	}
	if len(this.operands) > 2 {
		var arg value.Value
		arg, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(2, arg))
		}
		format = arg.ToString()
		if len(this.operands) == 3 {
			outputFormat = format
		}
	} else {
		format = formatFromStr(str)
		outputFormat = format
	}

	t, err = strToTime(str, format)
	if err != nil {
		return setWarning(context, err)
	}
	ms := t.UTC().UnixMilli()
	if ms < _MIN_MILLIS || ms > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	rv, err := timeToStr(t.In(loc), outputFormat)
	if err != nil {
		return setWarning(context, err)
	}
	return value.NewValue(rv), nil
}

func (this *StrToZoneName) MaxArgs() int { return 4 }
func (this *StrToZoneName) MinArgs() int { return 2 }

func (this *StrToZoneName) Constructor() FunctionConstructor {
	return NewStrToZoneName
}

///////////////////////////////////////////////////
//
// DurationToStr
//
///////////////////////////////////////////////////

// Convert a duration in nanoseconds to a string
type DurationToStr struct {
	FunctionBase
}

func NewDurationToStr(operands ...Expression) Function {
	rv := &DurationToStr{}
	rv.Init("duration_to_str", operands...)

	rv.expr = rv
	return rv
}

func (this *DurationToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DurationToStr) Type() value.Type { return value.STRING }

func (this *DurationToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
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
	var styleStr string
	if len(this.operands) == 2 {
		second, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if second.Type() == value.MISSING {
			missing = true
		} else if second.Type() != value.STRING {
			null = true
		}
		styleStr = second.ToString()
	}
	if missing {
		return value.MISSING_VALUE, nil
	}
	if null {
		return value.NULL_VALUE, nil
	} else if first.Type() != value.NUMBER {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, first))
	}

	var style util.DurationStyle
	if styleStr != "" {
		var ok bool
		style, ok = util.IsDurationStyle(styleStr)
		if !ok {
			return value.NULL_VALUE, nil
		}
	} else if dc, ok := context.(interface{ DurationStyle() util.DurationStyle }); ok {
		style = dc.DurationStyle()
	} else {
		style = util.GetDurationStyle()
	}

	d := first.Actual().(float64)
	str := util.FormatDuration(time.Duration(d), style)

	return value.NewValue(str), nil
}

func (this *DurationToStr) Constructor() FunctionConstructor {
	return NewDurationToStr
}

func (this *DurationToStr) MinArgs() int { return 1 }
func (this *DurationToStr) MaxArgs() int { return 2 }

///////////////////////////////////////////////////
//
// StrToDuration
//
///////////////////////////////////////////////////

// Convert a string representation of a duration to a nanosecond duration value.
type StrToDuration struct {
	FunctionBase
}

func NewStrToDuration(operands ...Expression) Function {
	rv := &StrToDuration{}
	rv.Init("str_to_duration", operands...)

	rv.expr = rv
	return rv
}

func (this *StrToDuration) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *StrToDuration) Type() value.Type { return value.NUMBER }

func (this *StrToDuration) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() != value.STRING {
		null = true
	}
	var styleStr string
	if len(this.operands) == 2 {
		second, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if second.Type() == value.MISSING {
			missing = true
		} else if second.Type() != value.STRING {
			null = true
		}
		styleStr = second.ToString()
	}
	if missing {
		return value.MISSING_VALUE, nil
	}
	if null {
		return value.NULL_VALUE, nil
	} else if first.Type() != value.STRING {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, first))
	}

	str := first.ToString()
	style := util.DEFAULT
	if styleStr != "" {
		var ok bool
		if style, ok = util.IsDurationStyle(styleStr); !ok {
			return value.NULL_VALUE, nil
		}
	}
	d, err := util.ParseDurationStyle(str, style)
	if err != nil {
		return setWarning(context, errors.W_DATE, err)
	}

	return value.NewValue(d), nil
}

func (this *StrToDuration) Constructor() FunctionConstructor {
	return NewStrToDuration
}

func (this *StrToDuration) MinArgs() int { return 1 }
func (this *StrToDuration) MaxArgs() int { return 2 }

///////////////////////////////////////////////////
//
// WeekdayMillis
//
///////////////////////////////////////////////////

// Return the English name of the weekday as a string. The date expr is a number representing UNIX milliseconds.
type WeekdayMillis struct {
	FunctionBase
}

func NewWeekdayMillis(operands ...Expression) Function {
	rv := &WeekdayMillis{}
	rv.Init("weekday_millis", operands...)

	rv.expr = rv
	return rv
}

func (this *WeekdayMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *WeekdayMillis) Type() value.Type { return value.STRING }

func (this *WeekdayMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false
	var info map[string]interface{}
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if first.Type() == value.MISSING {
		missing = true
	} else if first.Type() == value.NULL {
		null = true
	} else if first.Type() != value.NUMBER {
		info = invalidArgInfo(0, first)
	}

	// Initialize timezone to nil to avoid processing if not specified.
	var timeZone value.Value

	// Check if time zone is set
	if len(this.operands) > 1 {
		timeZone, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if timeZone.Type() == value.MISSING {
			missing = true
		} else if timeZone.Type() != value.STRING {
			if info == nil {
				info = invalidArgInfo(1, timeZone)
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else if info != nil {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, info)
	}

	millis := first.Actual().(float64)
	if millis < _MIN_MILLIS || millis > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	// Convert the input millis to *Time
	timeVal := millisToTime(millis)

	if timeZone != nil {
		// Process the timezone component as it isnt nil
		// Get the timezone and the *Location.
		tz := timeZone.ToString()
		loc, err := loadLocation(tz)
		if err != nil {
			return setWarning(context, errors.W_DATE_INVALID_TIMEZONE, tz)
		}
		// Use the timezone to get corresponding time component.
		timeVal = timeVal.In(loc)
	}

	dow, err := datePart(timeVal, "day_of_week")
	if err != nil {
		return setWarning(context, err)
	}

	rv := time.Weekday(dow).String()
	return value.NewValue(rv), nil
}

func (this *WeekdayMillis) MinArgs() int { return 1 }

func (this *WeekdayMillis) MaxArgs() int { return 2 }

func (this *WeekdayMillis) Constructor() FunctionConstructor {
	return NewWeekdayMillis
}

///////////////////////////////////////////////////
//
// WeekdayStr
//
///////////////////////////////////////////////////

// Return the English name of the weekday as a string. The date expr is a string in a supported format.
type WeekdayStr struct {
	UnaryFunctionBase
}

func NewWeekdayStr(first Expression) Function {
	rv := &WeekdayStr{}
	rv.Init("weekday_str", first)

	rv.expr = rv
	return rv
}

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
	} else if first.Type() == value.NULL {
		return value.NULL_VALUE, nil
	} else if first.Type() != value.STRING {
		return setWarning(context, errors.W_DATE_INVALID_ARGUMENT, invalidArgInfo(0, first))
	}

	str := first.ToString()
	t, err := strToTime(str, "")
	if err != nil {
		return setWarning(context, err)
	}
	ms := t.UTC().UnixMilli()
	if ms < _MIN_MILLIS || ms > _MAX_MILLIS {
		return setWarning(context, errors.W_DATE_OVERFLOW, nil)
	}

	dow, err := datePart(t, "day_of_week")
	if err != nil {
		return setWarning(context, err)
	}

	rv := time.Weekday(dow).String()
	return value.NewValue(rv), nil
}

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
	_, f, err := strToTimeFormatClosest(format)
	if err != nil {
		return t, err
	}
	return strToTimeGoFormat(s, f)
}

// Use go's standard formatting (e.g. 2006-01-02 03:04:05.000)
func strToTimeGoFormat(s string, format string) (time.Time, error) {
	// Go's formatting is inconsistent for our needs.  e.g. the specifier "MST" will parse "EST" and "EDT" but not "EST5EDT",
	// despite this being an IANA zone name and "EDT" not being one.
	// To avoid a performance penalty with the format, we implement our own Go format parser here so we can handle time zone parsing
	// in a way that suits us.  (Conversion to another format would incur a performance penalty, hence avoiding it.)

	var t time.Time
	var century, year, month, yday, day, hour, minute, second, fraction, l, zoneh, zonem int
	var loc *time.Location

	century = -1
	yearSeen := false
	yday = -1
	month = -1
	day = -1
	n := 0
	i := 0
	zoneh = _NO_ZONE
	pm := false
	h12 := false
	sign := false
	for i = 0; i < len(format) && n < len(s); i++ {
		if format[i] == ' ' && s[n] == ' ' {
			// space matches any number of spaces, and any number of consecutive spaces in the format is the same as a single space
			for i < len(format) && format[i] == ' ' {
				i++
			}
			i--
			for n < len(s) && s[n] == ' ' {
				n++
			}
			continue
		}
		if i+7 <= len(format) && format[i:i+7] == "January" {
			j := 0
			for j = 1; j < 13; j++ {
				m := time.Month(j).String()
				if strings.HasPrefix(s[n:], m) {
					month = j
					n += len(m)
					break
				}
			}
			if j > 12 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
			}
			i += 6
			continue
		}
		if i+6 <= len(format) && format[i:i+6] == "Monday" {
			j := 0
			for j = 0; j < 7; j++ {
				w := time.Weekday(j).String()
				if strings.HasPrefix(s[n:], w) {
					// parse & validate but do nothing with it
					n += len(w)
					break
				}
			}
			if j > 6 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day of week"))
			}
			i += 5
			continue
		}
		if i+4 <= len(format) && format[i:i+4] == "2006" {
			year, l, sign = gatherNumber(s[n:], 4, false, true)
			if (!sign && l != 4) || (sign && l != 5) {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "year"))
			}
			century = year / 100
			year = year % 100
			n += l
			i += 3
			continue
		}
		if i+3 <= len(format) {
			if format[i:i+3] == "MST" {
				var err error
				n, zoneh, zonem, loc, err = gatherZone(s, n, _TZ_FORMAT_NAME)
				if err != nil {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i, s, n, "time zone"))
				}
				i += 2
				continue
			}
			if format[i:i+3] == "Jan" {
				j := 0
				for j = 1; j < 13; j++ {
					m := time.Month(j).String()[:3] // Jan, Feb, Mar ...
					if strings.HasPrefix(s[n:], m) {
						month = j
						n += len(m)
						break
					}
				}
				if j > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
				}
				i += 2
				continue
			}
			if format[i:i+3] == "Mon" {
				j := 0
				for j = 0; j < 7; j++ {
					w := time.Weekday(j).String()[:3] // Sun, Mon, Tue...
					if strings.HasPrefix(s[n:], w) {
						// parse & validate but do nothing with it
						n += len(w)
						break
					}
				}
				if j > 6 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day of week"))
				}
				i += 2
				continue
			}
			if format[i:i+3] == "__2" {
				yday, l, _ = gatherNumber(s[n:], 3, true, false)
				if l < 1 || l > 3 || yday < 1 || yday > 366 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "year"))
				}
				n += l
				i += 2
				continue
			}
			if format[i:i+3] == "002" {
				yday, l, _ = gatherNumber(s[n:], 3, true, false)
				if l != 3 || yday < 1 || yday > 366 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day of year"))
				}
				n += l
				i += 2
				continue
			}
		}
		if i+2 <= len(format) {
			switch format[i : i+2] {
			case "01":
				month, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || month < 1 || month > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
				}
				n += l
				i++
				continue
			case "02":
				day, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || day < 1 || day > 31 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
				}
				n += l
				i++
				continue
			case "_2":
				day, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 2 && l != 1) || day < 1 || day > 31 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
				}
				n += l
				i++
				continue
			case "15":
				hour, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 2 && l != 1) || hour < 0 || hour > 23 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
				}
				h12 = false
				n += l
				i++
				continue
			case "03":
				hour, l, _ = gatherNumber(s[n:], 2, true, false)
				h12 = true
				if l != 2 || hour < 1 || hour > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
				}
				n += l
				i++
				continue
			case "PM":
				if n+1 < len(s) && s[n] == 'P' {
					pm = true
				} else if n+1 < len(s) && s[n] == 'A' {
					pm = false
				} else {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "12-hour indicator"))
				}
				n += 2
				i++
				continue
			case "pm":
				if n+1 < len(s) && s[n] == 'p' {
					pm = true
				} else if n+1 < len(s) && s[n] == 'a' {
					pm = false
				} else {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "12-hour indicator"))
				}
				n += 2
				i++
				continue
			case "04":
				minute, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || minute < 0 || minute > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
				}
				n += l
				i++
				continue
			case "05":
				second, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || second < 0 || second > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "second"))
				}
				n += l
				i++
				continue
			case "06":
				year, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || year < 0 || year > 99 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "year"))
				}
				yearSeen = true
				n += l
				i++
				continue
			case ".0":
				if s[n] != '.' {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
				}
				n++
				i++
				j := 0
				for ; i+j < len(format) && format[i] == format[i+j]; j++ {
				}
				i += j - 1
				fraction, l, _ = gatherNumber(s[n:], j, false, false)
				if l == 0 || l != j {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
				}
				n += l
				if l > 9 {
					l = 9
				}
				// convert to ns
				fraction *= int(math.Pow10(9 - l))
				continue
			case ".9":
				if s[n] != '.' {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
				}
				n++
				i++
				j := 0
				for ; i+j < len(format) && format[i] == format[i+j]; j++ {
				}
				i += j - 1
				fraction, l, _ = gatherNumber(s[n:], 9, true, false)
				if l == 0 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
				}
				n += l
				// convert to ns
				fraction *= int(math.Pow10(9 - l))
				continue
			}
		}
		switch format[i] {
		case '1':
			month, l, _ = gatherNumber(s[n:], 2, false, false)
			if (l != 1 && l != 2) || month < 1 || month > 12 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
			}
			n += l
			continue
		case '2':
			day, l, _ = gatherNumber(s[n:], 2, false, false)
			if (l != 1 && l != 2) || day < 1 || day > 31 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
			}
			n += l
			continue
		case '3':
			hour, l, _ = gatherNumber(s[n:], 2, false, false)
			h12 = true
			if (l != 1 && l != 2) || hour < 1 || hour > 12 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
			}
			n += l
			continue
		case '4':
			minute, l, _ = gatherNumber(s[n:], 2, false, false)
			if (l != 1 && l != 2) || minute < 0 || minute > 59 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
			}
			n += l
			continue
		case '5':
			second, l, _ = gatherNumber(s[n:], 2, false, false)
			if (l != 1 && l != 2) || second < 0 || second > 59 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "second"))
			}
			n += l
			continue
		}

		if format[i] == 'Z' || format[i] == '-' {
			var j int
			var tzf uint32
			if i+9 <= len(format) && format[i+1:i+9] == "07:00:00" {
				tzf = _TZ_FORMAT_2COLON
				j = 8
			} else if i+7 <= len(format) && format[i+1:i+7] == "070000" {
				tzf = _TZ_FORMAT_3PART
				j = 6
			} else if i+6 <= len(format) && format[i+1:i+6] == "07:00" {
				tzf = _TZ_FORMAT_1COLON
				j = 5
			} else if i+5 <= len(format) && format[i+1:i+5] == "0700" {
				tzf = _TZ_FORMAT_2PART
				j = 4
			} else if i+3 <= len(format) && format[i+1:i+3] == "07" {
				tzf = _TZ_FORMAT_1PART
				j = 2
			}
			if j > 0 || format[i] == 'Z' {
				if format[i] == 'Z' {
					tzf |= _TZ_FORMAT_ALLOW_Z
				}
				var err error
				n, zoneh, zonem, loc, err = gatherZone(s, n, tzf)
				if err != nil {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i, s, n, "time zone"))
				}
				i += j
				continue
			}
		}

		if format[i] != s[n] {
			return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, s, n, format[i]))
		}
		n++
	}

	if i != len(format) {
		return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(0x0)))
	}

	if n != len(s) {
		return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(0x3)))
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
	if yday != -1 {
		m, d := yearDay(year, yday)
		if (month != -1 && month != m) || (day != -1 && d != day) {
			return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day of year"))
		}
		month = m
		day = d
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
	if t.UnixMilli() > _MAX_MILLIS || t.UnixMilli() < _MIN_MILLIS {
		return t, errors.NewDateWarning(errors.W_DATE_OVERFLOW, nil)
	}
	return t, nil
}

const (
	padZero int = iota
	padSpace
	noPad
)

func errorInfo(format string, fpos int, input string, ipos int, extra ...interface{}) map[string]interface{} {
	info := make(map[string]interface{})
	if len(input) > 0 || len(format) == 0 {
		m := make(map[string]interface{}, 3)
		m["value"] = input
		if ipos >= 0 {
			m["~~~~~"] = strings.Repeat("-", ipos) + "^"
			m["pos"] = ipos
		}
		info["input"] = m
	}
	if len(format) > 0 {
		m := make(map[string]interface{}, 3)
		m["value"] = format
		if fpos >= 0 {
			m["~~~~~"] = strings.Repeat("-", fpos) + "^"
			m["pos"] = fpos
		}
		info["format"] = m
	}
	if len(extra) > 0 {
		switch ev := extra[0].(type) {
		case rune:
			if ev == rune(0x3) {
				info["expected"] = "<END>"
			} else if ev == rune(0x0) {
				info["expected"] = "<FURTHER INPUT>"
			} else {
				info["expected"] = fmt.Sprintf("%c", ev)
			}
		default:
			info["field"] = extra[0]
		}
	}
	return info
}

func processParseError(code errors.ErrorCode, e interface{}) errors.Error {
	if pe, ok := e.(*time.ParseError); ok {
		m := strings.TrimPrefix(pe.Message, ": ")
		off := 0
		field := ""
		if m != "" {
			if strings.HasSuffix(m, " out of range") {
				code = errors.W_DATE_INVALID_DATE_STRING
				field = strings.TrimSpace(m[:len(m)-13])
				if field == "year" {
					off = -4
				} else {
					off = -2
				}
			} else {
				field = m
			}
		}
		fpos := strings.Index(pe.Layout, pe.LayoutElem)
		ipos := strings.Index(pe.Value, pe.ValueElem)
		ipos += off
		if field != "" {
			return errors.NewDateWarning(code, errorInfo(pe.Layout, fpos, pe.Value, ipos, field))
		} else {
			return errors.NewDateWarning(code, errorInfo(pe.Layout, fpos, pe.Value, ipos))
		}
	}
	return errors.NewDateWarning(code, e)
}

// Date format *similar* to Unix 'date' command. Notable exceptions are locale-specific formats (we don't have locale internally
// curently) and opposite case specification; width specification is ignored too.  Upper case preference modifier is sequestered
// to mean case insensitive and a literal space means any literal character (useful when delimiters aren't consistent or there are
// portions to be ignored).
//
// Examples:
//
//	format    ...      parses
//	%F                 2021-06-25T04:00:00.000+05:30
//	%D                 2021-06-25
//	%Y %m %d           2021/06/25, 2021-06-25, 2021.06.25, etc.
//	%T %N              14:24:37.001002003, 14:24:37,001002003, 14:24:37:001002003, 14:24:37.2, 14:24:37,345, etc.
//	%d/%m/%y %-I %^p   25/06/21 4 am, 25/06/21 11 PM
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
	zoneh = _NO_ZONE
	pm := false
	h12 := false
	sign := false
	for i = 0; i < len(format) && n < len(s); i++ {
		if format[i] != '%' {
			if format[i] == ' ' {
				// space matches any character
				n++
			} else if format[i] != s[n] {
				return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, format[i]))
			} else {
				n++
			}
		} else if i+1 == len(format) {
			return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
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
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
			}
			st := i
			for ; unicode.IsDigit(rune(format[i])) && i < len(format); i++ {
			}
			if st < i {
				if i >= len(format) {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
				}
				// ignore the width specification
			}
			if format[i] == 'E' || format[i] == 'O' {
				i++
				if i >= len(format) {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
				}
			}
			colons := 0
			for format[i] == ':' {
				colons++
				i++
			}
			if colons > 0 && format[i] != 'z' {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
			}
			switch format[i] {
			case '%':
				if s[n] != '%' {
					return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, format[i]))
				}
			case 'x':
				fallthrough
			case 'D':
				// expand and reprocess
				format = format[:i-1] + _SHORT_DATE_FORMAT + format[i+1:]
				i -= 2
			case 'F':
				// expand and reprocess
				format = format[:i-1] + _FULL_DATE_FORMAT + format[i+1:]
				i -= 2
			case 'Y':
				year, l, sign = gatherNumber(s[n:], 4, pad == padSpace, true)
				if (pad == noPad && l < 1) || (pad != noPad && ((!sign && l != 4) || (sign && l != 5))) || year < -9999 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "year"))
				}
				century = year / 100
				year = year % 100
				n += l
			case 'C':
				century, l, sign = gatherNumber(s[n:], 2, pad == padSpace, true)
				if (pad == noPad && l == 0) || (pad != noPad && ((!sign && l != 2) || (sign && l != 3))) || century < -99 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "century"))
				}
				n += l
			case 'y':
				year, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || year < 0 || year > 99 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "year (in century)"))
				}
				yearSeen = true
				n += l
			case 'm':
				month, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || month < 1 || month > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
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
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
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
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
				}
			case 'd':
				day, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || day < 1 || day > 31 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
				}
				n += l
			case 'e':
				day, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || day < 1 || day > 31 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
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
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
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
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
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
				hour, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 0 || hour > 23 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour (00-23)"))
				}
				h12 = false
				n += l
			case 'I':
				hour, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 1 || hour > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour (01-12)"))
				}
				h12 = true
				n += l
			case 'k':
				hour, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || hour < 0 || hour > 23 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour (0-23)"))
				}
				h12 = false
				n += l
			case 'l':
				hour, l, _ = gatherNumber(s[n:], 2, true, false)
				if l != 2 || hour < 0 || hour > 11 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour (0-11)"))
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
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "12-hour indicator"))
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
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "12-hour indicator"))
				}
			case 'M':
				minute, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
				}
				n += l
			case 'S':
				second, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || second < 0 || second > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "second"))
				}
				n += l
			case 'R':
				hour, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 0 || hour > 23 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
				}
				h12 = false
				n += l
				if s[n] == ':' {
					n++
				}
				minute, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
				}
				n += l
			case 'X':
				fallthrough
			case 'T':
				hour, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 0 || hour > 23 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
				}
				h12 = false
				n += l
				if s[n] == ':' {
					n++
				}
				minute, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
				}
				n += l
				if s[n] == ':' {
					n++
				}
				second, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || second < 0 || second > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "second"))
				}
				n += l
			case 'n':
				fallthrough
			case 'N':
				fraction, l, _ = gatherNumber(s[n:], 9, pad == padSpace, false)
				if l == 0 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
				}
				// convert to ns
				fraction *= int(math.Pow10(9 - l))
				n += l
			case 'Z':
				fallthrough
			case 'z':
				var err error
				n, zoneh, zonem, loc, err = gatherZone(s, n, _TZ_FORMAT_ALL)
				if err != nil {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i, s, n, "time zone"))
				}
			case 's':
				var e time.Time
				if preferUpper {
					epoch := 0
					epoch, l, _ = gatherNumber(s[n:], 19, pad == padSpace, false)
					if l == 0 {
						return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
							errorInfo(format, i-1, s, n, "seconds since epoch"))
					}
					s := int64(epoch / 1000000000)
					n := int64(epoch % 1000000000)
					e = time.Unix(s, n)
				} else {
					epoch := 0
					epoch, l, _ = gatherNumber(s[n:], 10, pad == padSpace, false)
					if l == 0 {
						return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
							errorInfo(format, i-1, s, n, "seconds since epoch"))
					}
					e = time.Unix(int64(epoch), 0)
				}
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
			case 'r':
				hour, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || hour < 1 || hour > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
				}
				h12 = false
				n += l
				if s[n] == ':' {
					n++
				}
				minute, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || minute < 0 || minute > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
				}
				n += l
				if s[n] == ':' {
					n++
				}
				second, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l == 0) || (pad != noPad && l != 2) || second < 0 || second > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "second"))
				}
				n += l
				if n+1 < len(s) && ((s[n] == 'a' || s[n] == 'p') || (preferUpper && (s[n] == 'A' || s[n] == 'P'))) &&
					(s[n+1] == 'm' || (preferUpper && s[n+1] == 'M')) {
					if s[n] == 'p' || s[n] == 'P' {
						pm = true
					} else {
						pm = false
					}
					n += 2
				} else {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "12-hour indicator"))
				}
			case 'V':
				// parse but do nothing with it
				isoWeek, l, _ := gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || isoWeek < 1 || isoWeek > 53 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "ISO week number"))
				}
				n += l
			case 'G':
				// parse but do nothing with it
				isoYear, l, _ := gatherNumber(s[n:], 4, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 4) || isoYear < 0 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "ISO year"))
				}
				n += l
			case 'j':
				// parse but do nothing with it
				dayOfYear, l, _ := gatherNumber(s[n:], 3, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 3) || dayOfYear < 1 || dayOfYear > 366 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "day of year"))
				}
				n += l
			case 'q':
				// parse but do nothing with it
				quarter, l, _ := gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || quarter < 1 || quarter > 4 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "quarter"))
				}
				n += l
			case 'u':
				// parse but do nothing with it
				dow, l, _ := gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || dow < 1 || dow > 7 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "day of week"))
				}
				n += l
			case 'w':
				// parse but do nothing with it
				dow, l, _ := gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || dow < 0 || dow > 6 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "day of week"))
				}
				n += l
			case 'U':
				fallthrough
			case 'W':
				// parse but do nothing with it
				week, l, _ := gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || week < 0 || week > 53 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "day of week"))
				}
				n += l
			case '@':
				fallthrough
			case '#':
				var hh, mm, ss, ff int
				hh, l, _ = gatherNumber(s[n:], 10, pad == padSpace, false)
				if (l < 1) || hh < 0 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hours"))
				}
				n += l
				if s[n] == ':' {
					n++
				}
				mm, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || mm < 0 || mm > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minutes"))
				}
				n += l
				if s[n] == ':' {
					n++
				}
				ss, l, _ = gatherNumber(s[n:], 2, pad == padSpace, false)
				if (pad == noPad && l < 1) || (pad != noPad && l != 2) || ss < 0 || ss > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "seconds"))
				}
				n += l
				if format[i] == '@' {
					if s[n] == '.' {
						n++
					}
					ff, l, _ = gatherNumber(s[n:], 3, pad == padSpace, false)
					if (pad == noPad && l < 1) || (pad != noPad && l != 3) || ff < 0 || ff > 999 {
						return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
					}
					n += l
				} else {
					ff = 0
				}
				ms := (hh * 3600000) + (mm * 60000) + (ss * 1000) + ff
				t = millisToTime(float64(ms))
				if i+1 == len(format) && n == len(s) {
					return t, nil
				}
				year = t.Year()
				century = year / 100
				year %= 100
				month = int(t.Month())
				day = t.Day()
				hour = t.Hour()
				minute = t.Minute()
				second = t.Second()
				fraction = t.Nanosecond()
				h12 = false
			default:
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
			}
		}
	}

	if i != len(format) {
		return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(0x0)))
	}

	if n != len(s) {
		return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(0x3)))
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
	if t.UnixMilli() > _MAX_MILLIS || t.UnixMilli() < _MIN_MILLIS {
		return t, errors.NewDateWarning(errors.W_DATE_OVERFLOW, nil)
	}
	return t, nil
}

// Common date format, e.g. YYYY-MM-DD hh:mm:ss.s
//
// Components are:
// YYYY - 4 digit century+year
// CC   - 2 digit century (00...99)
// YY   - 2 digit year (00...99)
// MM   - 2 digit month (01..12)
// DD   - 2 digit day-of-month (01...31) (depending on month)
// hh   - 2 digit 24-hour hour (00...23)
// HH24 - synonym
// HH   - 2 digit 12-hour hour (01...12)
// HH12 - synonym
// mm   - 2 digit minute (00...59)
// MI   - synonym
// ss   - 2 digit second (00...59)
// s    - up to 9 digit fraction of a second
// pp   - 2 character 12-hour cycle indicator (am/pm)
// PP   - 2 character 12-hour cycle indicator (AM/PM)
// AM   - 2 character 12-hour cycle indicator UPPERCASE
// PM   - synonym
// am   - 2 character 12-hour cycle indicator LOWERCASE
// pm   - synonym
// TZD  - timezone specified as either: Z, +hh:mm:ss (seconds ignored), +hh:mm, +hhmm, +hh, <zone-name>
// TZN  - timezone specified as either: Z, +hh:mm:ss (seconds ignored), +hh:mm, +hhmm, +hh, <zone-name>
// MONTH - English month name (uppercase)
// Month - English month name (capitalised)
// month - English month name (lowercase)
// MON  - English month name abbreviated
// Mon  - English month name abbreviated
// mon  - English month name abbreviated
// DAY  - English day name
// Day  - English day name
// day  - English day name
// DY   - English day name abbreviated
// Dy   - English day name abbreviated
// dy   - English day name abbreviated
//
// Spaces match any character else non format characters have to be matched exactly. There is no escape sequence to use components
// listed above as literal content (individual parts can be, e.g. a single Y).

var _COMMON_FORMATS = map[rune][]string{ // descending length order is important in each array!
	'A': {"AM"},
	'a': {"am"},
	'C': {"CC"},
	'D': {"DAY", "Day", "Dy", "DY", "DD"},
	'd': {"day", "dy"},
	'H': {"HH12", "HH24", "HH"},
	'h': {"hh"},
	'M': {"MONTH", "Month", "MON", "Mon", "MM", "MI"},
	'm': {"month", "mon", "mm"},
	'P': {"PP", "PM"},
	'p': {"pp", "pm"},
	'S': {"SS"},
	's': {"ss", "s"},
	'T': {"TZD", "TZN", "TZF", "T"},
	'Y': {"YYYY", "YY"},
}

func isCommonFormat(format string) bool {
outer:
	for i := 0; len(format) > i; {
		if list, ok := _COMMON_FORMATS[rune(format[i])]; ok {
			for _, f := range list {
				if len(format) >= i+len(f) && format[i:i+len(f)] == f {
					i += len(f)
					continue outer
				}
			}
		}
		if unicode.IsSpace(rune(format[i])) || unicode.IsPunct(rune(format[i])) {
			i++
			continue outer
		}
		// not a valid format nor punctuation (or space), so not common format
		return false
	}
	return true
}

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
	zoneh = _NO_ZONE
	pm := false
	h12 := false
	sign := true
	for i = 0; i < len(format) && n < len(s); i++ {
		if format[i] == ' ' {
			// space matches any character
			n++
		} else if format[i] == 'Y' {
			if i+4 <= len(format) && format[i:i+4] == "YYYY" {
				year, l, sign = gatherNumber(s[n:], 4, false, true)
				if (!sign && l != 4) || (sign && l != 5) {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i, s, n, "year"))
				}
				century = year / 100
				year = year % 100
				i += 3
			} else if i+1 < len(format) && format[i+1] == 'Y' {
				year, l, _ = gatherNumber(s[n:], 2, false, false)
				if l != 2 || year < 0 || year > 99 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i, s, n, "year"))
				}
				i++
				yearSeen = true
			} else {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, "")
			}
			n += l
		} else if i+1 < len(format) && format[i] == format[i+1] && format[i] != 's' && format[i] != 'S' {
			i++
			switch format[i] {
			case 'C':
				century, l, sign = gatherNumber(s[n:], 2, false, true)
				if (!sign && l != 2) || (sign && l != 3) || century < -99 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "century"))
				}
				n += l
			case 'M':
				month, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 1 && l != 2) || month < 1 || month > 12 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "month"))
				}
				n += l
			case 'D':
				day, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 1 && l != 2) || day < 1 || day > 31 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day"))
				}
				n += l
			case 'h':
				hour, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 1 && l != 2) || hour < 0 || hour > 23 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour (00-23)"))
				}
				h12 = false
				n += l
			case 'H':
				hour, l, _ = gatherNumber(s[n:], 2, false, false)
				h12 = true
				min := 1
				max := 12
				if i+2 < len(format) {
					if format[i+1] == '1' {
						if format[i+2] != '2' {
							return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
						}
						i += 2
					} else if format[i+1] == '2' {
						if format[i+2] != '4' {
							return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
						}
						h12 = false
						min = 0
						max = 23
						i += 2
					}
				}
				if (l != 1 && l != 2) || hour < min || hour > max {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "hour"))
				}
				n += l
			case 'p':
				if n+1 < len(s) && (s[n] == 'p' || s[n] == 'P') {
					pm = true
				} else if n+1 < len(s) && (s[n] == 'a' || s[n] == 'A') {
					pm = false
				} else {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
						errorInfo(format, i-1, s, n, "12-hour indicator"))
				}
				n += 2
			case 'm':
				minute, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 1 && l != 2) || minute < 0 || minute > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
				}
				n += l
			default:
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, format)
			}
		} else if i+1 < len(format) && format[i] == 'M' && format[i+1] == 'I' {
			i++
			minute, l, _ = gatherNumber(s[n:], 2, false, false)
			if (l != 1 && l != 2) || minute < 0 || minute > 59 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "minute"))
			}
			n += l
		} else if i+1 < len(format) && (format[i] == 'A' || format[i] == 'P') && format[i+1] == 'M' {
			i++
			if n+1 < len(s) && s[n] == 'P' && s[n+1] == 'M' {
				pm = true
			} else if n+1 < len(s) && s[n] == 'A' && s[n+1] == 'M' {
				pm = false
			} else {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
					errorInfo(format, i-1, s, n, "12-hour indicator"))
			}
			n += 2
		} else if i+1 < len(format) && (format[i] == 'a' || format[i] == 'p') && format[i+1] == 'm' {
			i++
			if n+1 < len(s) && s[n] == 'p' && s[n+1] == 'm' {
				pm = true
			} else if n+1 < len(s) && s[n] == 'a' && s[n+1] == 'm' {
				pm = false
			} else {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
					errorInfo(format, i-1, s, n, "12-hour indicator"))
			}
			n += 2
		} else if format[i] == 's' || format[i] == 'S' {
			j := 0
			for j = 0; i+j < len(format); j++ {
				if format[i+j] != format[i] {
					break
				}
			}
			if j == 2 {
				second, l, _ = gatherNumber(s[n:], 2, false, false)
				if (l != 1 && l != 2) || second < 0 || second > 59 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "second"))
				}
			} else if j == 1 || j == 3 {
				fraction, l, _ = gatherNumber(s[n:], 9, false, false)
				if l == 0 {
					return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "fraction"))
				}
				// convert to ns
				fraction *= int(math.Pow10(9 - l))
			} else {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
			}
			i += j - 1
			n += l
		} else if i+2 < len(format) && (format[i:i+3] == "TZD" || format[i:i+3] == "TZN" || format[i:i+3] == "TZF") {
			var err error
			n, zoneh, zonem, loc, err = gatherZone(s, n, _TZ_FORMAT_ALL)
			if err != nil {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i, s, n, "time zone"))
			}
			i += 2
		} else if i+4 < len(format) && (format[i:i+5] == "MONTH" || format[i:i+5] == "Month" || format[i:i+5] == "month") {
			j := 0
			for j = 1; j < 13; j++ {
				m := time.Month(j).String()
				if format[i] == 'm' {
					m = strings.ToLower(m)
				} else if format[i+1] == 'O' {
					m = strings.ToUpper(m)
				}
				if strings.HasPrefix(s[n:], m) {
					month = j
					n += len(m)
					break
				}
			}
			i += 4
			if j > 12 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "full month name"))
			}
		} else if i+2 < len(format) && (format[i:i+3] == "MON" || format[i:i+3] == "Mon" || format[i:i+3] == "mon") {
			j := 0
			for j = 1; j < 13; j++ {
				m := time.Month(j).String()[:3] // Jan, Feb, Mar ...
				if format[i] == 'm' {
					m = strings.ToLower(m)
				} else if format[i+1] == 'O' {
					m = strings.ToUpper(m)
				}
				if strings.HasPrefix(s[n:], m) {
					month = j
					n += len(m)
					break
				}
			}
			i += 2
			if j > 12 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "short month name"))
			}
		} else if i+2 < len(format) && (format[i:i+3] == "DAY" || format[i:i+3] == "Day" || format[i:i+3] == "day") {
			j := 0
			for j = 0; j < 7; j++ {
				w := time.Weekday(j).String()
				if format[i] == 'd' {
					w = strings.ToLower(w)
				} else if format[i+1] == 'A' {
					w = strings.ToUpper(w)
				}
				if strings.HasPrefix(s[n:], w) {
					// parse & validate but do nothing with it
					n += len(w)
					break
				}
			}
			i += 2
			if j > 6 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, errorInfo(format, i-1, s, n, "day of week"))
			}
		} else if i+1 < len(format) && (format[i:i+2] == "DY" || format[i:i+2] == "Dy" || format[i:i+2] == "dy") {
			j := 0
			for j = 0; j < 7; j++ {
				w := time.Weekday(j).String()[:3] // Sun, Mon, Tue...
				if format[i] == 'd' {
					w = strings.ToLower(w)
				} else if format[i+1] == 'Y' {
					w = strings.ToUpper(w)
				}
				if strings.HasPrefix(s[n:], w) {
					// parse & validate but do nothing with it
					n += len(w)
					break
				}
			}
			i++
			if j > 6 {
				return t, errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING,
					errorInfo(format, i-1, s, n, "short day of week"))
			}
		} else if !unicode.IsPunct(rune(format[i])) && format[i] != 'T' {
			return t, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, errorInfo(format, i, "", i))
		} else {
			if format[i] != 'T' && format[i] != s[n] {
				return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(format[i])))
			}
			n++
		}
	}

	if i != len(format) {
		return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(0x0)))
	}

	if n != len(s) {
		return t, errors.NewDateWarning(errors.W_DATE_PARSE_FAILED, errorInfo(format, i, s, n, rune(0x3)))
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
		if century < 0 && year > 0 {
			year *= -1
		}
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
	if t.UnixMilli() > _MAX_MILLIS || t.UnixMilli() < _MIN_MILLIS {
		return t, errors.NewDateWarning(errors.W_DATE_OVERFLOW, nil)
	}
	return t, nil
}

// Determine the type of format string based on the content
type formatCache struct {
	sync.Mutex
	format string
	fType  formatType
}

var dateFormatCache formatCache = formatCache{sync.Mutex{}, "", defaultFormat}

func updateCache(fmt string, t formatType) formatType {
	dateFormatCache.Lock()
	dateFormatCache.format = fmt
	dateFormatCache.fType = t
	dateFormatCache.Unlock()
	return t
}

func determineFormat(fmt string) formatType {
	dateFormatCache.Lock()
	if fmt == dateFormatCache.format {
		rv := dateFormatCache.fType
		dateFormatCache.Unlock()
		return rv
	}
	dateFormatCache.Unlock()
	if len(fmt) == 0 {
		return updateCache(fmt, defaultFormat)
	} else if strings.IndexAny(fmt, "%") != -1 {
		return updateCache(fmt, percentFormat)
	} else if isCommonFormat(fmt) {
		return updateCache(fmt, commonFormat)
	} else if !unicode.IsDigit(rune(fmt[0])) { // standard formats all start with a digit
		return updateCache(fmt, goFormat)
	}
	i := 0
	for i = 0; i < len(fmt); i++ {
		if !unicode.IsDigit(rune(fmt[i])) {
			break
		}
	}
	n := fmt[0:i]
	if n == "2006" {
		return updateCache(fmt, goFormat)
	} else if len(n) < 3 {
		a := make([]rune, 2)
		a[0] = '0'
		for i := 1; i < 7; i++ {
			a[1] = rune('0' + i)
			if n == string(a) || n == string(a[1:]) {
				return updateCache(fmt, goFormat)
			}
		}
	}
	return updateCache(fmt, exampleFormat)
}

func gatherNumber(s string, max int, countLeadingSpaces bool, allowSign bool) (int, int, bool) {
	i := 0
	st := 0
	leading := true
	sign := false
	for i = 0; i < len(s) && i < max; i++ {
		if leading && countLeadingSpaces && unicode.IsSpace(rune(s[i])) {
			st++
			continue
		}
		if !unicode.IsDigit(rune(s[i])) && !(allowSign && leading && (s[i] == '-' || s[i] == '+')) {
			break
		}
		if allowSign && (s[i] == '-' || s[i] == '+') {
			max++
			sign = true
		}
		leading = false
	}
	if i == 0 || leading {
		return 0, 0, false
	}
	en := i
	if i > 9 && !countLeadingSpaces {
		en = 9
	}
	r, _ := strconv.Atoi(s[st:en])
	return r, i, sign
}

const (
	_TZ_FORMAT_2COLON = uint32(1) << iota
	_TZ_FORMAT_1COLON
	_TZ_FORMAT_3PART
	_TZ_FORMAT_2PART
	_TZ_FORMAT_1PART
	_TZ_FORMAT_NAME
	_TZ_FORMAT_ALLOW_Z
)
const _TZ_FORMAT_ALL = uint32(0xffffffff)

// try parse ISO-8601 time zone formats, or load location by name
func gatherZone(s string, n int, allowedFormats uint32) (int, int, int, *time.Location, error) {
	var err error
	var zoneh, zonem, sec int
	var loc *time.Location

	sn := n
	if allowedFormats&_TZ_FORMAT_2COLON != 0 && n+8 < len(s) &&
		s[n+3] == ':' && s[n+6] == ':' && (s[n] == '+' || s[n] == '-') && // +00:00:00

		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+4])) && unicode.IsDigit(rune(s[n+5])) &&
		unicode.IsDigit(rune(s[n+7])) && unicode.IsDigit(rune(s[n+8])) {
		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+4 : n+6])
		sec, _ = strconv.Atoi(s[n+7 : n+9]) // seconds are validated but ignored as aren't ISO-8601
		n += 9
	} else if allowedFormats&_TZ_FORMAT_1COLON != 0 && n+5 < len(s) &&
		s[n+3] == ':' && (s[n] == '+' || s[n] == '-') && // +00:00

		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+4])) && unicode.IsDigit(rune(s[n+5])) {
		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+4 : n+6])
		n += 6
	} else if allowedFormats&_TZ_FORMAT_3PART != 0 && n+6 < len(s) && (s[n] == '+' || s[n] == '-') && // +000000
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+3])) && unicode.IsDigit(rune(s[n+4])) &&
		unicode.IsDigit(rune(s[n+5])) && unicode.IsDigit(rune(s[n+6])) {

		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+3 : n+5])
		sec, _ = strconv.Atoi(s[n+5 : n+7]) // seconds are validated but ignored
		n += 7
	} else if allowedFormats&_TZ_FORMAT_2PART != 0 && n+4 < len(s) && (s[n] == '+' || s[n] == '-') && // +0000
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) &&
		unicode.IsDigit(rune(s[n+3])) && unicode.IsDigit(rune(s[n+4])) {

		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem, _ = strconv.Atoi(s[n+3 : n+5])
		n += 5
	} else if allowedFormats&_TZ_FORMAT_1PART != 0 && n+2 < len(s) && (s[n] == '+' || s[n] == '-') && // +00
		unicode.IsDigit(rune(s[n+1])) && unicode.IsDigit(rune(s[n+2])) {

		zoneh, _ = strconv.Atoi(s[n : n+3])
		zonem = 0
		n += 3
	} else if allowedFormats&_TZ_FORMAT_ALLOW_Z != 0 && n < len(s) && s[n] == 'Z' {
		zoneh = 0
		zonem = 0
		n++
	} else if allowedFormats&_TZ_FORMAT_NAME != 0 {
		var name string
		l := 0
		if n < len(s) {
			f := strings.FieldsFunc(s[n:], nonIANATZDBRune)
			if len(f) > 0 {
				f[0] = strings.TrimRight(f[0], "+-/_")
				l = len(f[0])
				// perform mapping before attempting to load so we can redirect (for example) EST to EST5EDT, the more commonly
				// used zone (for our purposes).
				if long, ok := shortToLong[f[0]]; ok {
					name = long
				} else {
					name = f[0]
				}
			}
		}
		if len(name) > 4 && (name[3] == '+' || name[3] == '-') && name[:3] == "GMT" && unicode.IsDigit(rune(name[4])) {
			e := 5
			// any number of digits are allowed; gatherNumber parses at most 9 so directly process here
			for e < len(name) && unicode.IsDigit(rune(name[e])) {
				e++
			}
			zoneh, _ = strconv.Atoi(name[3:e])
			zonem = 0
			if zoneh > 23 || zoneh < -23 {
				err = errors.NewDateWarning(errors.W_DATE_INVALID_TIMEZONE, name[:e])
			} else {
				loc = time.FixedZone(name[:e], zoneh*60*60)
				n += e
			}
		} else {
			loc, err = time.LoadLocation(name)
			if err != nil {
				err = errors.NewDateWarning(errors.W_DATE_INVALID_TIMEZONE, name)
			} else {
				n += l
			}
		}
	} else {
		err = errors.NewDateWarning(errors.W_DATE_INVALID_TIMEZONE, s[n:])
	}
	if err == nil && loc == nil && (sec > 59 || zoneh > 14 || zoneh < -12 || zonem > 59) {
		err = errors.NewDateWarning(errors.W_DATE_INVALID_TIMEZONE, s[sn:])
	}
	return n, zoneh, zonem, loc, err
}

func nonIANATZDBRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == '_' || r == '+' || r == '-' {
		return false
	}
	return true
}

// short zone monikers are not unique (e.g. CST = China Standard Time as well as (North American) Central Standard Time) so really
// are best avoided
// See: https://www.timeanddate.com/time/zones/
//
//	https://www.iana.org/time-zones
//
// a lingering problem for us is that Go's time package produces the non-unique short form in common formatting
var shortToLong = map[string]string{
	"ACST": "Australia/Darwin",
	"AEDT": "Australia/Sydney",
	"AEST": "Australia/Sydney",
	"AET":  "Australia/Sydney",
	"AWDT": "Australia/Perth",
	"AWST": "Australia/Perth",
	"BST":  "Europe/London",
	"CAT":  "Africa/Windhoek",
	"CDT":  "CST6CDT",
	"CEDT": "Europe/Paris",
	"CEST": "Europe/Paris",
	"CST":  "CST6CDT",
	"EDT":  "EST5EDT",
	"EEST": "Europe/Kiev",
	"EST":  "EST5EDT",
	"HMT":  "Europe/Helsinki",
	"JST":  "Asia/Tokyo",
	"KST":  "Asia/Seoul",
	"MDT":  "MST7MDT",
	"MEDT": "Europe/Paris",
	"MEST": "Europe/Paris",
	"MMT":  "Indian/Maldives",
	"MST":  "MST7MDT",
	"PDT":  "PST8PDT",
	"PST":  "PST8PDT",
	"SAST": "Africa/Johannesburg",
	"SGT":  "Asia/Singapore",
	"SMT":  "Asia/Singapore",
	"WAT":  "Africa/Lagos",
	"WEST": "Europe/Lisbon",
}

// Make sure YMD specification makes sense
func validateMonthAndDay(year int, month int, day int) error {
	if month < 1 || month > 12 {
		return errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, "month")
	}
	if day < 1 {
		return errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, "day")
	}
	if month == int(time.February) {
		if isLeapYear(year) {
			if day > 29 {
				return errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, "day")
			}
		} else if day > 28 {
			return errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, "day")
		}
	} else if month == int(time.April) || month == int(time.June) || month == int(time.September) || month == int(time.November) {
		if day > 30 {
			return errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, "day")
		}
	} else {
		if day > 31 {
			return errors.NewDateWarning(errors.W_DATE_INVALID_DATE_STRING, "day")
		}
	}
	return nil
}

// wrapper for loading a fixed location from a seconds-offset
func getLocation(h int, m int) *time.Location {
	if h == _NO_ZONE {
		return time.Local
	}
	off := h * 60 * 60
	if off >= 0 {
		off += m * 60
	} else {
		off -= m * 60
	}
	return time.FixedZone(fmt.Sprintf("%+03d%02d", h, m), off)
}

func loadLocation(tz string) (*time.Location, error) {
	_, zoneh, zonem, loc, err := gatherZone(tz, 0, _TZ_FORMAT_ALL)
	if err == nil && loc == nil {
		loc = getLocation(zoneh, zonem)
	}
	return loc, err
}

// Parse the input string using the defined formats for Date and return the time value it represents, and error.
// Pick the first one that successfully parses preferring formats that exactly match the length over those with optional components.
// (Optional components are handled by the time package API.)
func strToTimeTryAllDefaultFormats(s string) (time.Time, error) {
	var t time.Time
	var err error
	// first pass try formats that match length before encountering the overhead of parsing all
	for _, f := range _DATE_FORMATS {
		if len(f) == len(s) {
			t, err = strToTimeGoFormat(s, f)
			if err == nil {
				return t, nil
			}
		}
	}

	format := determineKnownFormat(s)
	if format == "" {
		err = errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, "unable to determine format")
	} else {
		t, err = strToTimeGoFormat(s, format)
		if err == nil {
			return t, nil
		}
	}

	return t, processParseError(errors.W_DATE_INVALID_DATE_STRING, err)
}

// Parse the input string using the defined formats for Date and return the time value it represents, and error.
// Pick the first one that successfully parses preferring formats that exactly match the length over those with optional components.
// (Optional components are handled by the time package API.)
//
// If an exact-length format can't be found, This tries all remaining and the one closest in length to the input string is picked in
// an effort to improve the selection especially when using an example to identify a format to use (some formats have components
// which are optional when parsing but present when formatting).
func StrToTimeFormat(s string) (time.Time, string, error) {
	return strToTimeFormatClosest(s)
}

func strToTimeFormatClosest(s string) (time.Time, string, error) {
	var t time.Time
	var err error

	// first pass try formats that match length before encountering the overhead of parsing all
	for _, f := range _DATE_FORMATS {
		if len(f) == len(s) {
			t, err = strToTimeGoFormat(s, f)
			if err == nil {
				return t, f, nil
			}
		}
	}

	format := determineKnownFormat(s)
	if format == "" {
		return t, DEFAULT_FORMAT, errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, "unable to determine format")
	} else {
		t, err = strToTimeGoFormat(s, format)
		if err == nil {
			return t, format, nil
		}
	}

	return t, DEFAULT_FORMAT, processParseError(errors.W_DATE_INVALID_DATE_STRING, err)
}

// Date string formatting: Returns a text representation of the time value formatted according to the format string.
func timeToStr(t time.Time, format string) (string, error) {
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
func timeToStrGoFormat(t time.Time, format string) (string, error) {
	return t.Format(format), nil
}

// find a format that parses the example given in the format string and use that to format the result
func timeToStrExampleFormat(t time.Time, format string) (string, error) {
	_, f, _ := strToTimeFormatClosest(format)

	return timeToStrGoFormat(t, f)
}

// format using Unix date-like format string (e.g. %Y-%m-%d %H:%M:%S.%N)
//
// Examples:
//
//	format    ...      produces
//	%F                 2021-06-25T04:00:00.000+05:30
//	%D                 2021-06-25
//	%Y.%m.%d           2021.06.25
//	%T,%n              14:24:37,345
//	%d/%m/%y %-I %^p   25/06/21 4 AM
//	[%_3S]             [ 37]
//	%Z                 BST
func timeToStrPercentFormat(t time.Time, format string) (string, error) {
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
				return "", errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, format)
			}
			width := 0
			st := i
			for ; i < len(format) && unicode.IsDigit(rune(format[i])); i++ {
			}
			if st < i {
				if i >= len(format) {
					return "", errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, format)
				}
				width, _ = strconv.Atoi(format[st:i])
			}
			if format[i] == 'E' || format[i] == 'O' {
				i++
				if i >= len(format) {
					return "", errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, format)
				}
			}
			colons := 0
			for format[i] == ':' {
				colons++
				i++
			}
			if colons > 0 && format[i] != 'z' {
				return "", errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, format)
			}
			switch format[i] {
			case 'x':
				fallthrough
			case 'D':
				// expand and reprocess
				format = format[:i-1] + _SHORT_DATE_FORMAT + format[i+1:]
				i -= 2
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
				if preferUpper {
					res = append(res, formatInt(width, 0, pad, int(t.UnixNano()))...)
				} else {
					res = append(res, formatInt(width, 0, pad, int(t.Unix()))...)
				}
			case 'r':
				p := false
				h := t.Hour()
				if h == 0 {
					h = 12
				} else if h > 12 {
					p = true
					h -= 12
				}
				res = append(res, formatInt(width, 2, pad, h)...)
				res = append(res, rune(':'))
				res = append(res, formatInt(width, 2, pad, t.Minute())...)
				res = append(res, rune(':'))
				res = append(res, formatInt(width, 2, pad, t.Second())...)
				res = append(res, rune(' '))
				if !p {
					res = append(res, []rune("AM")...)
				} else {
					res = append(res, []rune("PM")...)
				}
			case 'R':
				res = append(res, formatInt(width, 2, pad, t.Hour())...)
				res = append(res, rune(':'))
				res = append(res, formatInt(width, 2, pad, t.Minute())...)
			case 'X':
				fallthrough
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
				s := m % 60
				m /= 60
				if m < 0 {
					m = m * -1
					s = s * -1
				}
				switch colons {
				case 0:
					res = append(res, []rune(fmt.Sprintf("%+03d%02d", h, m))...)
				case 1:
					res = append(res, []rune(fmt.Sprintf("%+03d:%02d", h, m))...)
				case 2:
					res = append(res, []rune(fmt.Sprintf("%+03d:%02d:%02d", h, m, s))...)
				case 3:
					if m == 0 && s == 0 {
						res = append(res, []rune(fmt.Sprintf("%+03d", h))...)
					} else if s == 0 {
						res = append(res, []rune(fmt.Sprintf("%+03d:%02d", h, m))...)
					} else {
						res = append(res, []rune(fmt.Sprintf("%+03d:%02d:%02d", h, m, s))...)
					}
				default:
					return "", errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, format)
				}
			case 'Z':
				zone, off := t.Zone()
				if pad == noPad && off == 0 {
					res = append(res, rune('Z'))
				} else {
					// Uses the ^ modifier to mean "prefer location name" (and not uppercase)
					// In cases where the location has been constructed from an IANA zone name, the name will be reported
					// e.g. "Europe/Berlin" instead of "CET"
					if preferUpper {
						res = append(res, []rune(t.Location().String())...)
					} else {
						res = append(res, []rune(zone)...)
					}
				}
			case '%':
				res = append(res, rune('%'))
			case 'V':
				_, w := t.ISOWeek()
				res = append(res, formatInt(width, 1, pad, w)...)
			case 'G':
				y, _ := t.ISOWeek()
				res = append(res, formatInt(width, 1, pad, y)...)
			case 'j':
				res = append(res, formatInt(width, 1, pad, t.YearDay())...)
			case 'q':
				q := (int(t.Month()) - 1) / 3
				res = append(res, formatInt(width, 1, pad, q+1)...)
			case 'u':
				w := int(t.Weekday())
				if w == 0 {
					w = 7
				}
				res = append(res, formatInt(width, 1, pad, w)...)
			case 'w':
				res = append(res, formatInt(width, 1, pad, int(t.Weekday()))...)
			case 'U':
				first := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
				fd := int(first.Weekday())
				d := t.YearDay() - 1 + fd
				w := d / 7
				res = append(res, formatInt(width, 1, pad, w)...)
			case 'W':
				first := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
				fd := int(first.Weekday())
				fd -= 1
				if fd < 0 {
					fd += 7
				}
				d := t.YearDay() - 1 + fd
				w := d / 7
				res = append(res, formatInt(width, 1, pad, w)...)
			case '#':
				n := t.Unix()
				if n < 0 {
					n = 0
				}
				h := n / (60 * 60)
				n -= h * (60 * 60)
				m := n / 60
				s := n % 60
				res = append(res, []rune(fmt.Sprintf("%d:%02d:%02d", h, m, s))...)
			case '@':
				n := t.UnixMilli()
				if n < 0 {
					n = 0
				}
				h := n / (60 * 60 * 1000)
				n -= h * (60 * 60 * 1000)
				m := n / (60 * 1000)
				n -= m * (60 * 1000)
				s := n / 1000
				n %= 1000
				res = append(res, []rune(fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, n))...)
			default:
				return "", errors.NewDateWarning(errors.W_DATE_INVALID_FORMAT, fmt.Sprintf("%%%c", format[i]))
			}
		} else {
			res = append(res, rune(format[i]))
		}
	}
	return string(res), nil
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

// format using common-style format string (e.g. YYYY-MM-DD HH:mm:ss.s)
//
// Components are:
// YYYY - 4 digit century+year
// CC   - 2 digit century (00...99)
// YY   - 2 digit year (00...99)
// MM   - 2 digit month (01..12)
// MON/Mon/mon - 3 character month matching case
// MONTH/Month/month - month name matching case
// DD   - 2 digit day-of-month (01...31) (depending on month)
// DAY/Day/day - day-of-week matching case
// DY/Dy/dy - 3 character day-of-week matching case
// hh   - 2 digit 24-hour hour (00...23)
// HH24 - synonym
// HH   - 2 digit 12-hour hour (01...12)
// HH12 - synonym
// mm   - 2 digit minute (00...59)
// MI   - synonym
// ss   - 2 digit second (00...59)
// SS   - synonym
// s    - 3 digit zero-padded milliseconds
// sss  - 9 digit zero-padded nanoseconds
// PP   - 2 character upper case 12-hour cycle indicator (AM/PM)
// AM   - synonym
// PM   - synonym
// pp   - 2 character lower case 12-hour cycle indicator (am/pm)
// am   - synonym
// pm   - synonym
// TZD  - timezone Z or +hh:mm
// TZN  - timezone name
//
// Other characters/sequences are produced literally in the output.
func timeToStrCommonFormat(t time.Time, format string) (string, error) {
	res := make([]rune, 0, len(format)*3)
	i := 0
	for i = 0; i < len(format); i++ {
		if i+3 < len(format) && format[i:i+4] == "YYYY" {
			res = append(res, []rune(fmt.Sprintf("%04d", t.Year()))...)
			i += 3
		} else if i+1 < len(format) && format[i] == format[i+1] && format[i] != 's' && format[i] != 'S' {
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
				if i+2 < len(format) {
					if format[i+1] == '1' && format[i+2] == '2' {
						i += 2
					} else if format[i+1] == '2' && format[i+2] == '4' {
						res = append(res, []rune(fmt.Sprintf("%02d", t.Hour()))...)
						i += 2
						break
					}
				}
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
		} else if i+1 < len(format) && format[i] == 'M' && format[i+1] == 'I' {
			res = append(res, []rune(fmt.Sprintf("%02d", t.Minute()))...)
			i++
		} else if i+1 < len(format) && (format[i] == 'A' || format[i] == 'P') && format[i+1] == 'M' {
			i++
			h := t.Hour()
			if h < 12 {
				res = append(res, []rune("AM")...)
			} else {
				res = append(res, []rune("PM")...)
			}
		} else if i+1 < len(format) && (format[i] == 'a' || format[i] == 'p') && format[i+1] == 'm' {
			i++
			h := t.Hour()
			if h < 12 {
				res = append(res, []rune("am")...)
			} else {
				res = append(res, []rune("pm")...)
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
		} else if i+2 < len(format) && format[i:i+3] == "TZN" {
			zone, _ := t.Zone()
			res = append(res, []rune(zone)...)
			i += 2
		} else if i+2 < len(format) && format[i:i+3] == "TZF" {
			zone := t.Location().String()
			res = append(res, []rune(zone)...)
			i += 2
		} else if format[i] == 's' || format[i] == 'S' {
			n := 0
			for n = 0; n+i < len(format); n++ {
				if format[i+n] != format[i] {
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
		} else if i+4 < len(format) && (format[i:i+5] == "MONTH" || format[i:i+5] == "Month" || format[i:i+5] == "month") {
			m := t.Month().String()
			if format[i] == 'm' {
				m = strings.ToLower(m)
			} else if format[i+1] == 'O' {
				m = strings.ToUpper(m)
			}
			res = append(res, []rune(m)...)
			i += 4
		} else if i+2 < len(format) && (format[i:i+3] == "MON" || format[i:i+3] == "Mon" || format[i:i+3] == "mon") {
			m := t.Month().String()[:3]
			if format[i] == 'm' {
				m = strings.ToLower(m)
			} else if format[i+1] == 'O' {
				m = strings.ToUpper(m)
			}
			res = append(res, []rune(m)...)
			i += 2
		} else if i+2 < len(format) && (format[i:i+3] == "DAY" || format[i:i+3] == "Day" || format[i:i+3] == "day") {
			w := t.Weekday().String()
			if format[i] == 'd' {
				w = strings.ToLower(w)
			} else if format[i+1] == 'A' {
				w = strings.ToUpper(w)
			}
			res = append(res, []rune(w)...)
			i += 2
		} else if i+1 < len(format) && (format[i:i+2] == "DY" || format[i:i+2] == "Dy" || format[i:i+2] == "dy") {
			w := t.Weekday().String()[:3]
			if format[i] == 'd' {
				w = strings.ToLower(w)
			} else if format[i+1] == 'Y' {
				w = strings.ToUpper(w)
			}
			res = append(res, []rune(w)...)
			i++
		} else {
			res = append(res, rune(format[i]))
		}
	}
	return string(res), nil
}

// Convert input milliseconds to time format by multiplying with 10^6 and using the Unix method from the time package.
func millisToTime(millis float64) time.Time {
	return time.Unix(int64(millis/1000), int64(math.Mod(millis, 1000)*1000000.0))
}

// Convert input time to milliseconds from nanoseconds returned by UnixNano().
func timeToMillis(t time.Time) float64 {
	return float64(t.Unix()*1000) + float64(t.Round(time.Millisecond).Nanosecond())/1000000
}

// Default date formats
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

func isDateSeparator(r rune) bool {
	return r == '/' || r == '.' || r == '-'
}

// When the input date's length doesn't exactly match the length of a _DATE_FORMATS entry that successfully parses it, it is more
// efficient to try analyse what fields exist than to try parsing with all other entries.
// (Better still would be to just parse the date string directly)
func determineKnownFormat(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 8 {
		return ""
	}
	dt := ""
	// date part
	if (s[0] == '-' || s[0] == '+') && isDateSeparator(rune(s[5])) && len(s) >= 11 {
		s = s[1:]
	}
	if isDateSeparator(rune(s[4])) && len(s) >= 10 {
		if s[7] != s[4] {
			return ""
		}
		for i := 0; i < 10; i++ {
			if !(i == 4 || i == 7 || isdigit(s[i])) {
				return ""
			}
		}
		if len(s) == 10 {
			return fmt.Sprintf("2006%c01%c02", s[4], s[4])
		}
		n := 10
		if s[n] == ' ' {
			for len(s) > n+1 && s[n+1] == ' ' {
				n++
			}
		}
		if s[n] != 'T' && s[n] != ' ' {
			return ""
		}
		dt = fmt.Sprintf("2006%c01%c02%c", s[4], s[4], s[n])
		s = s[n+1:]
	}

	if s == "" {
		return dt
	}

	if len(s) < 5 {
		return ""
	}

	// time part
	if s[2] == ':' && ((len(s) > 5 && s[5] == ':') || len(s) == 5) {
		for i := 0; i < 8 && i < len(s); i++ {
			if i == 2 || i == 5 {
				if s[i] != ':' {
					return ""
				}
			} else if !isdigit(s[i]) {
				return ""
			}
		}
		if len(s) == 8 {
			return dt + "15:04:05"
		} else if len(s) == 5 {
			return dt + "15:04"
		} else if len(s) < 8 {
			return ""
		}
		n := 8
		frac := ""
		// fractions
		if s[n] == '.' {
			n++
			i := 0
			for n+i < len(s) && isdigit(s[n+i]) {
				i++
			}
			if i == 0 {
				return ""
			} else if i <= 3 {
				frac = ".999"
			} else if i <= 6 {
				frac = ".999999"
			} else if i <= 9 {
				frac = ".999999999"
			} else {
				return ""
			}
			if len(s) == n+i {
				return dt + "15:04:05" + frac
			}
			n += i
		}

		sep := frac + "Z"
		// TZ
		if s[n] == ' ' {
			for n < len(s) && s[n] == ' ' {
				n++
			}
			sep = frac + " Z"
		}
		if len(s) > n && s[n] == 'Z' || s[n] == '+' || s[n] == '-' {
			if s[n] == 'Z' {
				if len(s) == n+1 {
					return dt + "15:04:05" + sep + "07:00"
				}
				return ""
			}
			if len(s) < n+3 || !isdigit(s[n+1]) || !isdigit(s[n+2]) {
				return ""
			}
			if len(s) == n+3 {
				return dt + "15:04:05" + sep + "07"
			}
			if len(s) == n+5 {
				if isdigit(s[n+3]) || isdigit(s[n+4]) {
					return dt + "15:04:05" + sep + "0700"
				}
				return ""
			}
			if len(s) != n+6 || s[n+3] != ':' || !isdigit(s[n+4]) || !isdigit(s[n+5]) {
				return ""
			}
			return dt + "15:04:05" + sep + "07:00"
		}
	}
	return ""
}

func isdigit(b byte) bool {
	// should really be unicode.IsDigit but this ought to be faster
	return b >= '0' && b <= '9'
}

const DEFAULT_FORMAT = util.DEFAULT_FORMAT                // Represents the default format of the time string.
const _SHORT_DATE_FORMAT = "%Y-%m-%d"                     // only used for '%D' expansion
const _FULL_DATE_FORMAT = _SHORT_DATE_FORMAT + "T%T.%N%z" // only used for '%F' expansion

// Return the part of the time string that is depicted by part (for eg. the day, current quarter etc).
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
	case "calendar_month":
		fallthrough
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
		return 0, errors.NewDateWarning(errors.W_DATE_INVALID_COMPONENT, part)
	}
}

// Add part to the input time string using AddDate method from the time package. n and part are used to define the interval or
// duration.
func dateAdd(t time.Time, n int, part string) (time.Time, error) {
	p := strings.ToLower(part)
	var res time.Time

	if n == 0 {
		return t, nil
	} else if n >= math.MaxInt64 || n <= math.MinInt64 {
		return t, errors.NewDateWarning(errors.W_DATE_OVERFLOW, nil)
	}

	switch p {
	case "millennium":
		res = t.AddDate(n*1000, 0, 0)
	case "century":
		res = t.AddDate(n*100, 0, 0)
	case "decade":
		res = t.AddDate(n*10, 0, 0)
	case "year":
		res = t.AddDate(n, 0, 0)
	case "quarter":
		res = t.AddDate(0, n*3, 0)
	case "month":
		res = t.AddDate(0, n, 0)
	case "calendar_month":
		// adds months but if the original was the last day of the start month, the result is the last day of the new month
		// if the new day would be beyond the end of the new month, round it down to the end of the new month (as opposed to
		// advancing the months; e.g. 2021-01-31 + 1 calendar_month = 2021-02-28). This mimics the behaviour of some RDBMSes.
		om := t.Month()
		od := t.Day()
		last := false
		switch {
		case om == time.January || om == time.March || om == time.May || om == time.July ||
			om == time.August || om == time.October || om == time.December:
			if od == 31 {
				last = true
			}
		case om == time.February:
			ly := isLeapYear(t.Year())
			if ly && od == 29 {
				last = true
			} else if !ly && od == 28 {
				last = true
			}
		default:
			if od == 30 {
				last = true
			}
		}
		ny := t.Year() + (n / 12)
		nm := time.January
		if n > 0 {
			t := int(om-1) + (n % 12)
			if t >= 12 {
				t -= 12
				ny++
			}
			nm = time.Month(t + 1)
		} else {
			t := int(om-1) + (n % 12)
			if t < 0 {
				t += 12
				ny--
			}
			nm = time.Month(t + 1)
		}
		nd := od
		if last {
			switch {
			case nm == time.January || nm == time.March || nm == time.May || nm == time.July ||
				nm == time.August || nm == time.October || nm == time.December:
				nd = 31
			case nm == time.February:
				nd = 28
				if isLeapYear(ny) {
					nd = 29
				}
			default:
				nd = 30
			}
		} else {
			switch {
			case nm == time.January || nm == time.March || nm == time.May || nm == time.July ||
				nm == time.August || nm == time.October || nm == time.December:
				if nd > 31 {
					nd = 31
				}
			case nm == time.February:
				max := 28
				if isLeapYear(ny) {
					max = 29
				}
				if nd > max {
					nd = max
				}
			default:
				if nd > 30 {
					nd = 30
				}
			}
		}
		res = time.Date(ny, nm, nd, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	case "week":
		res = t.AddDate(0, 0, n*7)
	case "day":
		res = t.AddDate(0, 0, n)
	case "hour":
		res = t.Add(time.Duration(n) * time.Hour)
	case "minute":
		res = t.Add(time.Duration(n) * time.Minute)
	case "second":
		res = t.Add(time.Duration(n) * time.Second)
	case "millisecond":
		res = t.Add(time.Duration(n) * time.Millisecond)
	default:
		return t, errors.NewDateWarning(errors.W_DATE_INVALID_COMPONENT, part)
	}
	t1 := timeToMillis(t)
	t2 := timeToMillis(res)
	if (n > 0 && t1 > t2) || (n < 0 && t1 < t2) || t2 < _MIN_MILLIS || t2 > _MAX_MILLIS {
		return t, errors.NewDateWarning(errors.W_DATE_OVERFLOW, nil)
	}
	return res, nil
}

// Truncate out the part of the date string from the output and return the remaining time t.
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
	case "calendar_month":
		fallthrough
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

// Return time t that truncates the Day in the week that year and returns the Time.
func yearTrunc(t time.Time) time.Time {
	t, _ = timeTrunc(t, "day")
	return t.AddDate(0, 0, 1-t.YearDay())
}

// Return the time t with the day part truncated out. First get Time part as day. Subtract that from the days and
// then Add the given number of years, months and days to t using the AddDate method from the time package and return.
func monthTrunc(t time.Time) time.Time {
	t, _ = timeTrunc(t, "day")
	return t.AddDate(0, 0, 1-t.Day())
}

// Truncate the time string based on the value of the part string.  If type day convert to hours.
func timeTrunc(t time.Time, part string) (time.Time, error) {
	switch part {
	case "day":
		// add the zone offset effectively negating the zone so zone doesn't
		// interfere with the truncation
		_, off := t.Zone()
		t = t.Add(time.Duration(off) * time.Second)

		t = t.Truncate(time.Duration(24) * time.Hour)

		// revert the zone negation
		t = t.Add(time.Duration(off*-1) * time.Second)

		return t, nil
	case "hour":
		return t.Truncate(time.Hour), nil
	case "minute":
		return t.Truncate(time.Minute), nil
	case "second":
		return t.Truncate(time.Second), nil
	case "millisecond":
		return t.Truncate(time.Millisecond), nil
	default:
		return t, errors.NewDateWarning(errors.W_DATE_INVALID_COMPONENT, part)
	}
}

// Return the difference between the two times. Call diffDates to calculate the difference between the 2 time strings
// and then calls diffPart over these strings to unify it into part format. In the event t2 is greater than t1, and the result
// returns a negative value, return a negative result.
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

// Return a value specifying a part of the dates specified by part string. For each type of part (enumerated in the specs) it
// computes the value in the type part(for eg. seconds) recursively and returning it in format (int64) as per the part string.
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
	case "calendar_month":
		fallthrough
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
		return 0, errors.NewDateWarning(errors.W_DATE_INVALID_COMPONENT, part)
	}
}

// Return the difference between two dates. The input arguments to this function are of type Time. We use the setDate
// to extract the dates from the time and then compute and return the difference between the two dates.
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

// The type date is a structure containing fields year, day in that year, hour, minute, second and millisecond (all integers).
type date struct {
	year        int
	doy         int
	hour        int
	minute      int
	second      int
	millisecond int
}

// Extract a date from the input time and sets the year (as t.year), Day (as t.YearDay), and time (hour, minute,
// second using t.Clock, and millisecond as Nanosecond/10^6) using the input time which is of type Time (defined in package time).
func setDate(d *date, t time.Time) {
	d.year = t.Year()
	d.doy = t.YearDay()
	d.hour, d.minute, d.second = t.Clock()
	d.millisecond = t.Nanosecond() / 1000000
}

// Round input float64 value to int.
func round(f float64) int {
	if math.Abs(f) < 0.5 {
		return 0
	}
	return int(f + math.Copysign(0.5, f))
}

// Compute the number of leap years in between start and end year, using the method leapYearsWithin.
func leapYearsBetween(end, start int) int {
	return leapYearsWithin(end) - leapYearsWithin(start)
}

// return the number of leap years up until the input year. This is done using the computation (year/4) - (year/100) + (year/400).
func leapYearsWithin(year int) int {
	if year > 0 {
		year--
	} else {
		year++
	}

	return (year / 4) - (year / 100) + (year / 400)
}

// Determine if the input year is a leap year. Leap years can be evenly divided by 4, and should not be evenly divided by 100,
// unless it can be evenly divided by 400.
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

// convert day of year (1-366) to month and day
func yearDay(year int, yday int) (int, int) {
	if yday <= 31 {
		return 1, yday
	} else if yday <= 59 {
		return 2, yday - 31
	} else if yday == 60 {
		if isLeapYear(year) {
			return 2, 29
		}
		return 3, 1
	} else if isLeapYear(year) {
		yday--
	}
	yday -= 59
	switch {
	case yday <= 31:
		return 3, yday
	case yday <= 31+30:
		return 4, yday - 31
	case yday <= 31+30+31:
		return 5, yday - 31 - 30
	case yday <= 31+30+31+30:
		return 6, yday - 31 - 30 - 31
	case yday <= 31+30+31+30+31:
		return 7, yday - 31 - 30 - 31 - 30
	case yday <= 31+30+31+30+31+31:
		return 8, yday - 31 - 30 - 31 - 30 - 31
	case yday <= 31+30+31+30+31+31+30:
		return 9, yday - 31 - 30 - 31 - 30 - 31 - 31
	case yday <= 31+30+31+30+31+31+30+31:
		return 10, yday - 31 - 30 - 31 - 30 - 31 - 31 - 30
	case yday <= 31+30+31+30+31+31+30+31+30:
		return 11, yday - 31 - 30 - 31 - 30 - 31 - 31 - 30 - 31
	default:
		return 12, yday - 31 - 30 - 31 - 30 - 31 - 31 - 30 - 31 - 30
	}
}

func invalidArgInfo(arg int, v value.Value) map[string]interface{} {
	info := make(map[string]interface{})
	info["argument"] = arg
	if v != nil {
		info["type"] = v.Type().String()
	}
	return info
}

func invalidArgValue(arg int, v value.Value) map[string]interface{} {
	info := make(map[string]interface{})
	info["argument"] = arg
	if v != nil {
		info["value"] = v.Actual()
	}
	return info
}

func setWarning(context Context, other ...interface{}) (value.Value, errors.Error) {
	if c, ok := context.(interface {
		Warning(errors.Error)
		IsFeatureEnabled(uint64) bool
	}); ok && len(other) > 0 {
		if c.IsFeatureEnabled(util.N1QL_NO_DATE_WARNINGS) {
			return value.NULL_VALUE, nil
		}
		switch o := other[0].(type) {
		case errors.ErrorCode:
			if len(other) == 2 {
				c.Warning(errors.NewDateWarning(o, other[1]))
			} else {
				c.Warning(errors.NewDateWarning(o, nil))
			}
		case errors.Error:
			c.Warning(o)
		}
	}
	return value.NULL_VALUE, nil
}
