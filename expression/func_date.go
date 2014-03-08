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
	"time"

	"github.com/couchbaselabs/query/value"
)

type ClockNowMillis struct {
	ExpressionBase
}

func NewClockNowMillis() Function {
	return &ClockNowMillis{}
}

func (this *ClockNowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := time.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *ClockNowMillis) MinArgs() int { return 0 }

func (this *ClockNowMillis) MaxArgs() int { return 0 }

func (this *ClockNowMillis) Constructor() FunctionConstructor {
	return func(Expressions) Function { return this }
}

type ClockNowStr struct {
	ExpressionBase
}

func NewClockNowStr() Function {
	return &ClockNowStr{}
}

func (this *ClockNowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	str := time.Now().String()
	return value.NewValue(str), nil
}

func (this *ClockNowStr) Constructor() FunctionConstructor {
	return func(Expressions) Function { return this }
}

type DateAddMillis struct {
	nAryBase
}

func NewDateAddMillis(args Expressions) Function {
	return &DateAddMillis{
		nAryBase{
			operands: args,
		},
	}
}

func (this *DateAddMillis) evaluate(args value.Values) (value.Value, error) {
	ev := args[0]
	nv := args[1]
	pv := args[2]

	if ev.Type() == value.MISSING || nv.Type() == value.MISSING ||
		pv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if ev.Type() != value.NUMBER || nv.Type() != value.NUMBER ||
		pv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	ea := ev.Actual().(float64)
	na := nv.Actual().(float64)
	if na != math.Trunc(na) {
		return value.NULL_VALUE, nil
	}

	pa := pv.Actual().(string)
	t, e := dateAdd(millisToTime(ea), int(na), pa)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

func (this *DateAddMillis) MinArgs() int { return 3 }

func (this *DateAddMillis) MaxArgs() int { return 3 }

func (this *DateAddMillis) Constructor() FunctionConstructor { return NewDateAddMillis }

type DateAddStr struct {
	nAryBase
}

func NewDateAddStr(args Expressions) Function {
	return &DateAddStr{
		nAryBase{
			operands: args,
		},
	}
}

func (this *DateAddStr) evaluate(args value.Values) (value.Value, error) {
	ev := args[0]
	nv := args[1]
	pv := args[2]

	if ev.Type() == value.MISSING || nv.Type() == value.MISSING ||
		pv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if ev.Type() != value.STRING || nv.Type() != value.NUMBER ||
		pv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	ea := ev.Actual().(string)
	t, e := strToTime(ea)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	na := nv.Actual().(float64)
	if na != math.Trunc(na) {
		return value.NULL_VALUE, nil
	}

	pa := pv.Actual().(string)
	t, e = dateAdd(t, int(na), pa)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t, ea)), nil
}

func (this *DateAddStr) MinArgs() int { return 3 }

func (this *DateAddStr) MaxArgs() int { return 3 }

func (this *DateAddStr) Constructor() FunctionConstructor { return NewDateAddStr }

type DateDiffMillis struct {
	nAryBase
}

func NewDateDiffMillis(args Expressions) Function {
	return &DateDiffMillis{
		nAryBase{
			operands: args,
		},
	}
}

func (this *DateDiffMillis) evaluate(args value.Values) (value.Value, error) {
	dv1 := args[0]
	dv2 := args[1]
	pv := args[2]

	if dv2.Type() == value.MISSING || dv2.Type() == value.MISSING ||
		pv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if dv1.Type() != value.NUMBER || dv2.Type() != value.NUMBER ||
		pv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := dv1.Actual().(float64)
	da2 := dv2.Actual().(float64)
	pa := pv.Actual().(string)
	diff, e := dateDiff(millisToTime(da1), millisToTime(da2), pa)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(diff)), nil
}

func (this *DateDiffMillis) MinArgs() int { return 3 }

func (this *DateDiffMillis) MaxArgs() int { return 3 }

func (this *DateDiffMillis) Constructor() FunctionConstructor { return NewDateDiffMillis }

type DateDiffStr struct {
	nAryBase
}

func NewDateDiffStr(args Expressions) Function {
	return &DateDiffStr{
		nAryBase{
			operands: args,
		},
	}
}

func (this *DateDiffStr) evaluate(args value.Values) (value.Value, error) {
	dv1 := args[0]
	dv2 := args[1]
	pv := args[2]

	if dv2.Type() == value.MISSING || dv2.Type() == value.MISSING ||
		pv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if dv1.Type() != value.STRING || dv2.Type() != value.STRING ||
		pv.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	da1 := dv1.Actual().(string)
	t1, e := strToTime(da1)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	da2 := dv2.Actual().(string)
	t2, e := strToTime(da2)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	pa := pv.Actual().(string)
	diff, e := dateDiff(t1, t2, pa)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(diff)), nil
}

func (this *DateDiffStr) MinArgs() int { return 3 }

func (this *DateDiffStr) MaxArgs() int { return 3 }

func (this *DateDiffStr) Constructor() FunctionConstructor { return NewDateDiffStr }

type DatePartMillis struct {
	binaryBase
}

func NewDatePartMillis(first, second Expression) Function {
	return &DatePartMillis{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *DatePartMillis) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)
	part := second.Actual().(string)
	rv, e := datePart(millisToTime(millis), part)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(rv)), nil
}

func (this *DatePartMillis) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDatePartMillis(args[0], args[1])
	}
}

type DatePartStr struct {
	binaryBase
}

func NewDatePartStr(first, second Expression) Function {
	return &DatePartStr{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *DatePartStr) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	part := second.Actual().(string)
	t, e := strToTime(str)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	rv, e := datePart(t, part)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(float64(rv)), nil
}

func (this *DatePartStr) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDatePartStr(args[0], args[1])
	}
}

type DateTruncMillis struct {
	binaryBase
}

func NewDateTruncMillis(first, second Expression) Function {
	return &DateTruncMillis{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *DateTruncMillis) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	millis := first.Actual().(float64)
	part := second.Actual().(string)
	t := millisToTime(millis)

	var e error
	t, e = dateTrunc(t, part)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

func (this *DateTruncMillis) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDateTruncMillis(args[0], args[1])
	}
}

type DateTruncStr struct {
	binaryBase
}

func NewDateTruncStr(first, second Expression) Function {
	return &DateTruncStr{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *DateTruncStr) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	part := second.Actual().(string)
	t, e := strToTime(str)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	t, e = dateTrunc(t, part)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t, str)), nil
}

func (this *DateTruncStr) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDateTruncStr(args[0], args[1])
	}
}

type MillisToStr struct {
	nAryBase
}

func NewMillisToStr(args Expressions) Function {
	return &MillisToStr{
		nAryBase{
			operands: args,
		},
	}
}

func (this *MillisToStr) evaluate(args value.Values) (value.Value, error) {
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

func (this *MillisToStr) MinArgs() int { return 1 }

func (this *MillisToStr) MaxArgs() int { return 2 }

func (this *MillisToStr) Constructor() FunctionConstructor { return NewMillisToStr }

type MillisToUTC struct {
	nAryBase
}

func NewMillisToUTC(args Expressions) Function {
	return &MillisToUTC{
		nAryBase{
			operands: args,
		},
	}
}

func (this *MillisToUTC) evaluate(args value.Values) (value.Value, error) {
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

func (this *MillisToUTC) MinArgs() int { return 1 }

func (this *MillisToUTC) MaxArgs() int { return 2 }

func (this *MillisToUTC) Constructor() FunctionConstructor { return NewMillisToUTC }

type MillisToZoneName struct {
	nAryBase
}

func NewMillisToZoneName(args Expressions) Function {
	return &MillisToZoneName{
		nAryBase{
			operands: args,
		},
	}
}

func (this *MillisToZoneName) evaluate(args value.Values) (value.Value, error) {
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
	loc, e := time.LoadLocation(tz)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	fmt := fv.Actual().(string)
	t := millisToTime(millis).In(loc)
	return value.NewValue(timeToStr(t, fmt)), nil
}

func (this *MillisToZoneName) MinArgs() int { return 2 }

func (this *MillisToZoneName) MaxArgs() int { return 3 }

func (this *MillisToZoneName) Constructor() FunctionConstructor { return NewMillisToZoneName }

type NowMillis struct {
	ExpressionBase
}

func NewNowMillis() Function {
	return &NowMillis{}
}

func (this *NowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := context.(Context).Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *NowMillis) MinArgs() int { return 0 }

func (this *NowMillis) MaxArgs() int { return 0 }

func (this *NowMillis) Constructor() FunctionConstructor {
	return func(Expressions) Function { return this }
}

type NowStr struct {
	ExpressionBase
}

func NewNowStr() Function {
	return &NowStr{}
}

func (this *NowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	str := context.(Context).Now().String()
	return value.NewValue(str), nil
}

func (this *NowStr) Constructor() FunctionConstructor {
	return func(Expressions) Function { return this }
}

type StrToMillis struct {
	unaryBase
}

func NewStrToMillis(arg Expression) Function {
	return &StrToMillis{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *StrToMillis) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := arg.Actual().(string)
	t, e := strToTime(str)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToMillis(t)), nil
}

func (this *StrToMillis) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewStrToMillis(args[0])
	}
}

type StrToUTC struct {
	unaryBase
}

func NewStrToUTC(arg Expression) Function {
	return &StrToUTC{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *StrToUTC) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := arg.Actual().(string)
	t, e := strToTime(str)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	t = t.UTC()
	return value.NewValue(timeToStr(t, str)), nil
}

func (this *StrToUTC) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewStrToUTC(args[0])
	}
}

type StrToZoneName struct {
	binaryBase
}

func NewStrToZoneName(first, second Expression) Function {
	return &StrToZoneName{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *StrToZoneName) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := first.Actual().(string)
	t, e := strToTime(str)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	tz := second.Actual().(string)
	loc, e := time.LoadLocation(tz)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(timeToStr(t.In(loc), str)), nil
}

func (this *StrToZoneName) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewStrToZoneName(args[0], args[1])
	}
}

func strToTime(s string) (time.Time, error) {
	var t time.Time
	var err error
	for _, f := range _DATE_FORMATS {
		t, err = time.Parse(f, s)
		if err == nil {
			return t, nil
		}
	}

	return t, err
}

func timeToStr(t time.Time, format string) string {
	return t.Format(format)
}

func millisToTime(millis float64) time.Time {
	return time.Unix(0, int64(millis*1000000.0))
}

func timeToMillis(t time.Time) float64 {
	return float64(t.UnixNano() / 1000000)
}

var _DATE_FORMATS = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999Z07:00",
	time.RFC3339,
	"2006-01-02 15:04:05.999Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02",
	"15:04:05.999999999Z07:00",
	"15:04:05Z07:00",
	"15:04:05.999999999",
	"15:04:05",
}

const _DEFAULT_FORMAT = "2006-01-02 15:04:05.999Z07:00"

var _DEFAULT_FMT_VALUE = value.NewValue(_DEFAULT_FORMAT)

func datePart(t time.Time, part string) (int, error) {
	switch part {
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

func dateAdd(t time.Time, n int, part string) (time.Time, error) {
	switch part {
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

func dateTrunc(t time.Time, part string) (time.Time, error) {
	switch part {
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
		return timeTrunc(t, part)
	}
}

func yearTrunc(t time.Time) time.Time {
	t, _ = timeTrunc(t, "day")
	return t.AddDate(0, 0, 1-t.YearDay())
}

func monthTrunc(t time.Time) time.Time {
	t, _ = timeTrunc(t, "day")
	return t.AddDate(0, 0, 1-t.Day())
}

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

func dateDiff(t1, t2 time.Time, part string) (int64, error) {
	diff := diffDates(t1, t2)
	return diffPart(t1, t2, diff, part)
}

func diffPart(t1, t2 time.Time, diff *date, part string) (int64, error) {
	switch part {
	case "millisecond":
		sec, e := diffPart(t1, t2, diff, "second")
		if e != nil {
			return 0, e
		}
		return (sec * 1000) + int64(diff.millisecond), nil
	case "second":
		min, e := diffPart(t1, t2, diff, "min")
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

func diffDates(t1, t2 time.Time) *date {
	var d1, d2, diff date
	setDate(&d1, t1)
	setDate(&d2, t2)

	if d1.millisecond < d2.millisecond {
		d1.millisecond += 1000
		d1.second--
	}
	diff.millisecond = d1.millisecond - d2.millisecond

	if d1.second < d2.second {
		d1.second += 60
		d1.minute--
	}
	diff.second = d1.second - d2.second

	if d1.minute < d2.minute {
		d1.minute += 60
		d1.hour--
	}
	diff.minute = d1.minute - d2.minute

	if d1.hour < d2.hour {
		d1.hour += 24
		d1.doy--
	}
	diff.hour = d1.hour - d2.hour

	if d1.doy < d2.doy {
		if isLeapYear(d2.year) {
			d2.doy -= 366
		} else {
			d2.doy -= 365
		}
		d2.year++
	}
	diff.doy = d1.doy - d2.doy

	diff.year = d1.year - d2.year
	return &diff
}

type date struct {
	year        int
	doy         int
	hour        int
	minute      int
	second      int
	millisecond int
}

func setDate(d *date, t time.Time) {
	d.year = t.Year()
	d.doy = t.YearDay()
	d.hour, d.minute, d.second = t.Clock()
	d.millisecond = t.Nanosecond() / 1000000
}

func leapYearsBetween(end, start int) int {
	return leapYearsWithin(end) - leapYearsWithin(start)
}

func leapYearsWithin(year int) int {
	if year > 0 {
		year--
	} else {
		year++
	}

	return (year / 4) - (year / 100) + (year / 400)
}

func isLeapYear(year int) bool {
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}
