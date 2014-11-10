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

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// ClockMillis
//
///////////////////////////////////////////////////

type ClockMillis struct {
	NullaryFunctionBase
}

func NewClockMillis() Function {
	return &ClockMillis{
		*NewNullaryFunctionBase("clock_millis"),
	}
}

func (this *ClockMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockMillis) Type() value.Type { return value.NUMBER }

func (this *ClockMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := time.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *ClockMillis) Indexable() bool {
	return false
}

func (this *ClockMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return this }
}

///////////////////////////////////////////////////
//
// ClockStr
//
///////////////////////////////////////////////////

type ClockStr struct {
	FunctionBase
}

func NewClockStr(operands ...Expression) Function {
	return &ClockStr{
		*NewFunctionBase("clock_str", operands...),
	}
}

func (this *ClockStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ClockStr) Type() value.Type { return value.STRING }

func (this *ClockStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
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

func (this *ClockStr) Indexable() bool {
	return false
}

func (this *ClockStr) MinArgs() int { return 0 }

func (this *ClockStr) MaxArgs() int { return 1 }

func (this *ClockStr) Constructor() FunctionConstructor { return NewClockStr }

///////////////////////////////////////////////////
//
// DateAddMillis
//
///////////////////////////////////////////////////

type DateAddMillis struct {
	TernaryFunctionBase
}

func NewDateAddMillis(first, second, third Expression) Function {
	return &DateAddMillis{
		*NewTernaryFunctionBase("date_add_millis", first, second, third),
	}
}

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

type DateAddStr struct {
	TernaryFunctionBase
}

func NewDateAddStr(first, second, third Expression) Function {
	return &DateAddStr{
		*NewTernaryFunctionBase("date_add_str", first, second, third),
	}
}

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
	t, err := strToTime(da)
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

	return value.NewValue(t.String()), nil
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

type DateDiffMillis struct {
	TernaryFunctionBase
}

func NewDateDiffMillis(first, second, third Expression) Function {
	return &DateDiffMillis{
		*NewTernaryFunctionBase("date_diff_millis", first, second, third),
	}
}

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

type DateDiffStr struct {
	TernaryFunctionBase
}

func NewDateDiffStr(first, second, third Expression) Function {
	return &DateDiffStr{
		*NewTernaryFunctionBase("date_diff_str", first, second, third),
	}
}

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

type DatePartMillis struct {
	BinaryFunctionBase
}

func NewDatePartMillis(first, second Expression) Function {
	return &DatePartMillis{
		*NewBinaryFunctionBase("date_part_millis", first, second),
	}
}

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

type DatePartStr struct {
	BinaryFunctionBase
}

func NewDatePartStr(first, second Expression) Function {
	return &DatePartStr{
		*NewBinaryFunctionBase("date_part_str", first, second),
	}
}

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

type DateTruncMillis struct {
	BinaryFunctionBase
}

func NewDateTruncMillis(first, second Expression) Function {
	return &DateTruncMillis{
		*NewBinaryFunctionBase("date_trunc_millis", first, second),
	}
}

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

type DateTruncStr struct {
	BinaryFunctionBase
}

func NewDateTruncStr(first, second Expression) Function {
	return &DateTruncStr{
		*NewBinaryFunctionBase("date_trunc_str", first, second),
	}
}

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

type MillisToStr struct {
	FunctionBase
}

func NewMillisToStr(operands ...Expression) Function {
	return &MillisToStr{
		*NewFunctionBase("millis_to_str", operands...),
	}
}

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

func (this *MillisToStr) MinArgs() int { return 1 }

func (this *MillisToStr) MaxArgs() int { return 2 }

func (this *MillisToStr) Constructor() FunctionConstructor { return NewMillisToStr }

///////////////////////////////////////////////////
//
// MillisToUTC
//
///////////////////////////////////////////////////

type MillisToUTC struct {
	FunctionBase
}

func NewMillisToUTC(operands ...Expression) Function {
	return &MillisToUTC{
		*NewFunctionBase("millis_to_utc", operands...),
	}
}

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

func (this *MillisToUTC) MinArgs() int { return 1 }

func (this *MillisToUTC) MaxArgs() int { return 2 }

func (this *MillisToUTC) Constructor() FunctionConstructor { return NewMillisToUTC }

///////////////////////////////////////////////////
//
// MillisToZoneName
//
///////////////////////////////////////////////////

type MillisToZoneName struct {
	FunctionBase
}

func NewMillisToZoneName(operands ...Expression) Function {
	return &MillisToZoneName{
		*NewFunctionBase("millis_to_zone_name", operands...),
	}
}

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

func (this *MillisToZoneName) MinArgs() int { return 2 }

func (this *MillisToZoneName) MaxArgs() int { return 3 }

func (this *MillisToZoneName) Constructor() FunctionConstructor { return NewMillisToZoneName }

///////////////////////////////////////////////////
//
// MowMillis
//
///////////////////////////////////////////////////

type NowMillis struct {
	NullaryFunctionBase
}

func NewNowMillis() Function {
	return &NowMillis{
		*NewNullaryFunctionBase("now_millis"),
	}
}

func (this *NowMillis) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowMillis) Type() value.Type { return value.NUMBER }

func (this *NowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := context.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *NowMillis) Indexable() bool {
	return false
}

func (this *NowMillis) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return this }
}

///////////////////////////////////////////////////
//
// NowStr
//
///////////////////////////////////////////////////

type NowStr struct {
	FunctionBase
}

func NewNowStr(operands ...Expression) Function {
	return &NowStr{
		*NewFunctionBase("now_str", operands...),
	}
}

func (this *NowStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NowStr) Type() value.Type { return value.STRING }

func (this *NowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
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

func (this *NowStr) Indexable() bool {
	return false
}

func (this *NowStr) MinArgs() int { return 0 }

func (this *NowStr) MaxArgs() int { return 1 }

func (this *NowStr) Constructor() FunctionConstructor { return NewNowStr }

///////////////////////////////////////////////////
//
// StrToMillis
//
///////////////////////////////////////////////////

type StrToMillis struct {
	UnaryFunctionBase
}

func NewStrToMillis(operand Expression) Function {
	return &StrToMillis{
		*NewUnaryFunctionBase("str_to_millis", operand),
	}
}

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

type StrToUTC struct {
	UnaryFunctionBase
}

func NewStrToUTC(operand Expression) Function {
	return &StrToUTC{
		*NewUnaryFunctionBase("str_to_utc", operand),
	}
}

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

type StrToZoneName struct {
	BinaryFunctionBase
}

func NewStrToZoneName(first, second Expression) Function {
	return &StrToZoneName{
		*NewBinaryFunctionBase("str_to_zone_name", first, second),
	}
}

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

func (this *StrToZoneName) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewStrToZoneName(operands[0], operands[1])
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

const _DEFAULT_FORMAT = "2006-01-02T15:04:05.999Z07:00"

var _DEFAULT_FMT_VALUE = value.NewValue(_DEFAULT_FORMAT)

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
	var diff *date
	if t1.String() > t2.String() {
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

	diff.millisecond = d1.millisecond - d2.millisecond
	diff.second = d1.second - d2.second
	diff.minute = d1.minute - d2.minute
	diff.hour = d1.hour - d2.hour
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
