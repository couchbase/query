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

	if pv.Type() == value.MISSING || nv.Type() == value.MISSING ||
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

	if pv.Type() == value.MISSING || nv.Type() == value.MISSING ||
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

type DateUTCStr struct {
	unaryBase
}

func NewDateUTCStr(operand Expression) Function {
	return &DateUTCStr{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *DateUTCStr) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := operand.Actual().(string)
	t, e := strToTime(str)
	if e != nil {
		return value.NULL_VALUE, nil
	}

	t = t.UTC()
	return value.NewValue(timeToStr(t, str)), nil
}

func (this *DateUTCStr) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDateUTCStr(args[0])
	}
}

type MillisToStr struct {
	unaryBase
}

func NewMillisToStr(operand Expression) Function {
	return &MillisToStr{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *MillisToStr) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	millis := operand.Actual().(float64)
	t := millisToTime(millis)
	return value.NewValue(timeToStr(t, "")), nil
}

func (this *MillisToStr) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewMillisToStr(args[0])
	}
}

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

func NewStrToMillis(operand Expression) Function {
	return &StrToMillis{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *StrToMillis) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str := operand.Actual().(string)
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
	if format == "" {
		format = _DEFAULT_FORMAT
	}

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
