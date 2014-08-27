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
	"math"
	"sort"

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// ArrayAppend
//
///////////////////////////////////////////////////

type ArrayAppend struct {
	BinaryFunctionBase
}

func NewArrayAppend(first, second Expression) Function {
	return &ArrayAppend{
		*NewBinaryFunctionBase("array_append", first, second),
	}
}

func (this *ArrayAppend) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayAppend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayAppend) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().([]interface{})
	ra := append(f, second)
	return value.NewValue(ra), nil
}

func (this *ArrayAppend) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayAppend(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayAvg
//
///////////////////////////////////////////////////

type ArrayAvg struct {
	UnaryFunctionBase
}

func NewArrayAvg(operand Expression) Function {
	return &ArrayAvg{
		*NewUnaryFunctionBase("array_avg", operand),
	}
}

func (this *ArrayAvg) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayAvg) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayAvg) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sum := 0.0
	count := 0
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() == value.NUMBER {
			sum += v.Actual().(float64)
			count++
		}
	}

	if count == 0 {
		return value.NULL_VALUE, nil
	} else {
		return value.NewValue(sum / float64(count)), nil
	}
}

func (this *ArrayAvg) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayAvg(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayConcat
//
///////////////////////////////////////////////////

type ArrayConcat struct {
	BinaryFunctionBase
}

func NewArrayConcat(first, second Expression) Function {
	return &ArrayConcat{
		*NewBinaryFunctionBase("array_concat", first, second),
	}
}

func (this *ArrayConcat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayConcat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayConcat) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY || second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().([]interface{})
	s := second.Actual().([]interface{})
	ra := append(f, s...)
	return value.NewValue(ra), nil
}

func (this *ArrayConcat) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayConcat(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayContains
//
///////////////////////////////////////////////////

type ArrayContains struct {
	BinaryFunctionBase
}

func NewArrayContains(first, second Expression) Function {
	return &ArrayContains{
		*NewBinaryFunctionBase("array_contains", first, second),
	}
}

func (this *ArrayContains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayContains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayContains) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	for _, f := range fa {
		if second.Equals(value.NewValue(f)) {
			return value.NewValue(true), nil
		}
	}

	return value.NewValue(false), nil
}

func (this *ArrayContains) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayContains(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayCount
//
///////////////////////////////////////////////////

type ArrayCount struct {
	UnaryFunctionBase
}

func NewArrayCount(operand Expression) Function {
	return &ArrayCount{
		*NewUnaryFunctionBase("array_count", operand),
	}
}

func (this *ArrayCount) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayCount) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayCount) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	count := 0
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() > value.NULL {
			count++
		}
	}

	return value.NewValue(count), nil
}

func (this *ArrayCount) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayCount(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayDistinct
//
///////////////////////////////////////////////////

type ArrayDistinct struct {
	UnaryFunctionBase
}

func NewArrayDistinct(operand Expression) Function {
	return &ArrayDistinct{
		*NewUnaryFunctionBase("array_distinct", operand),
	}
}

func (this *ArrayDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayDistinct) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayDistinct) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	set := value.NewSet(len(aa))
	for _, a := range aa {
		set.Add(value.NewValue(a))
	}

	return value.NewValue(set.Actuals()), nil
}

func (this *ArrayDistinct) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayDistinct(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayIfNull
//
///////////////////////////////////////////////////

type ArrayIfNull struct {
	UnaryFunctionBase
}

func NewArrayIfNull(operand Expression) Function {
	return &ArrayIfNull{
		*NewUnaryFunctionBase("array_ifnull", operand),
	}
}

func (this *ArrayIfNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayIfNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayIfNull) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() > value.NULL {
			return v, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *ArrayIfNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayIfNull(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayLength
//
///////////////////////////////////////////////////

type ArrayLength struct {
	UnaryFunctionBase
}

func NewArrayLength(operand Expression) Function {
	return &ArrayLength{
		*NewUnaryFunctionBase("array_length", operand),
	}
}

func (this *ArrayLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayLength) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	return value.NewValue(float64(len(aa))), nil
}

func (this *ArrayLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayLength(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayMax
//
///////////////////////////////////////////////////

type ArrayMax struct {
	UnaryFunctionBase
}

func NewArrayMax(operand Expression) Function {
	return &ArrayMax{
		*NewUnaryFunctionBase("array_max", operand),
	}
}

func (this *ArrayMax) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayMax) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayMax) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	rv := value.NULL_VALUE
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Collate(rv) > 0 {
			rv = v
		}
	}

	return rv, nil
}

func (this *ArrayMax) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayMax(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayMin
//
///////////////////////////////////////////////////

type ArrayMin struct {
	UnaryFunctionBase
}

func NewArrayMin(operand Expression) Function {
	return &ArrayMin{
		*NewUnaryFunctionBase("array_min", operand),
	}
}

func (this *ArrayMin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayMin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayMin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	rv := value.NULL_VALUE
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() > value.NULL &&
			(rv == value.NULL_VALUE || v.Collate(rv) < 0) {
			rv = v
		}
	}

	return rv, nil
}

func (this *ArrayMin) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayMin(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayPosition
//
///////////////////////////////////////////////////

type ArrayPosition struct {
	BinaryFunctionBase
}

func NewArrayPosition(first, second Expression) Function {
	return &ArrayPosition{
		*NewBinaryFunctionBase("array_position", first, second),
	}
}

func (this *ArrayPosition) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayPosition) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	for i, f := range fa {
		if second.Equals(value.NewValue(f)) {
			return value.NewValue(float64(i)), nil
		}
	}

	return value.NewValue(float64(-1)), nil
}

func (this *ArrayPosition) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayPosition(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayPrepend
//
///////////////////////////////////////////////////

type ArrayPrepend struct {
	BinaryFunctionBase
}

func NewArrayPrepend(first, second Expression) Function {
	return &ArrayPrepend{
		*NewBinaryFunctionBase("array_prepend", first, second),
	}
}

func (this *ArrayPrepend) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayPrepend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayPrepend) Apply(context Context, first, second value.Value) (value.Value, error) {
	if second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	} else if first.Type() == value.MISSING {
		return second, nil
	}

	s := second.Actual().([]interface{})
	ra := make([]interface{}, 1, len(s)+1)
	ra[0] = first
	ra = append(ra, s...)
	return value.NewValue(ra), nil
}

func (this *ArrayPrepend) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayPrepend(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayPut
//
///////////////////////////////////////////////////

type ArrayPut struct {
	BinaryFunctionBase
}

func NewArrayPut(first, second Expression) Function {
	return &ArrayPut{
		*NewBinaryFunctionBase("array_put", first, second),
	}
}

func (this *ArrayPut) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayPut) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	} else if second.Type() == value.MISSING {
		return first, nil
	}

	f := first.Actual().([]interface{})
	for _, a := range f {
		v := value.NewValue(a)
		if second.Equals(v) {
			return first, nil
		}
	}

	ra := append(f, second)
	return value.NewValue(ra), nil
}

func (this *ArrayPut) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayPut(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayRange
//
///////////////////////////////////////////////////

type ArrayRange struct {
	FunctionBase
}

func NewArrayRange(operands ...Expression) Function {
	return &ArrayRange{
		*NewFunctionBase("array_range", operands...),
	}
}

func (this *ArrayRange) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayRange) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayRange) Apply(context Context, args ...value.Value) (value.Value, error) {
	startv := args[0]
	endv := args[1]
	stepv := value.ONE_VALUE
	if len(args) > 2 {
		stepv = args[2]
	}

	if startv.Type() == value.MISSING ||
		endv.Type() == value.MISSING ||
		stepv.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if startv.Type() != value.NUMBER ||
		endv.Type() != value.NUMBER ||
		stepv.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	start := startv.Actual().(float64)
	end := endv.Actual().(float64)
	step := stepv.Actual().(float64)

	if step == 0.0 ||
		start == end ||
		(step > 0.0 && start > end) ||
		(step < 0.0 && start < end) {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	rv := make([]interface{}, 0, int(math.Abs(end-start)/math.Abs(step)))
	for v := start; (step > 0.0 && v < end) || v > end; v += step {
		rv = append(rv, v)
	}

	return value.NewValue(rv), nil
}

func (this *ArrayRange) MinArgs() int { return 2 }

func (this *ArrayRange) MaxArgs() int { return 3 }

func (this *ArrayRange) Constructor() FunctionConstructor { return NewArrayRange }

///////////////////////////////////////////////////
//
// ArrayRemove
//
///////////////////////////////////////////////////

type ArrayRemove struct {
	BinaryFunctionBase
}

func NewArrayRemove(first, second Expression) Function {
	return &ArrayRemove{
		*NewBinaryFunctionBase("array_remove", first, second),
	}
}

func (this *ArrayRemove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayRemove) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	} else if second.Type() == value.MISSING {
		return first, nil
	}

	fa := first.Actual().([]interface{})
	ra := make([]interface{}, 0, len(fa))
	for _, f := range fa {
		if !second.Equals(value.NewValue(f)) {
			ra = append(ra, f)
		}
	}

	return value.NewValue(ra), nil
}

func (this *ArrayRemove) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayRemove(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayRepeat
//
///////////////////////////////////////////////////

type ArrayRepeat struct {
	BinaryFunctionBase
}

func NewArrayRepeat(first, second Expression) Function {
	return &ArrayRepeat{
		*NewBinaryFunctionBase("array_repeat", first, second),
	}
}

func (this *ArrayRepeat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayRepeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayRepeat) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	sf := second.Actual().(float64)
	if sf != math.Trunc(sf) {
		return value.NULL_VALUE, nil
	}

	n := int(sf)
	ra := make([]interface{}, n)
	for i := 0; i < n; i++ {
		ra[i] = first
	}

	return value.NewValue(ra), nil
}

func (this *ArrayRepeat) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayRepeat(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayReplace
//
///////////////////////////////////////////////////

type ArrayReplace struct {
	FunctionBase
}

func NewArrayReplace(operands ...Expression) Function {
	return &ArrayReplace{
		*NewFunctionBase("array_replace", operands...),
	}
}

func (this *ArrayReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayReplace) Apply(context Context, args ...value.Value) (value.Value, error) {
	av := args[0]
	v1 := args[1]
	v2 := args[2]

	if av.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if av.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	} else if v1.Type() == value.MISSING {
		return av, nil
	}

	aa := av.Actual().([]interface{})
	ra := make([]interface{}, 0, len(aa))
	for _, a := range aa {
		v := value.NewValue(a)
		if v1.Equals(v) {
			if v2.Type() != value.MISSING {
				ra = append(ra, v2)
			}
		} else {
			ra = append(ra, v)
		}
	}

	return value.NewValue(ra), nil
}

func (this *ArrayReplace) MinArgs() int { return 3 }

func (this *ArrayReplace) MaxArgs() int { return 4 }

func (this *ArrayReplace) Constructor() FunctionConstructor { return NewArrayReplace }

///////////////////////////////////////////////////
//
// ArrayReverse
//
///////////////////////////////////////////////////

type ArrayReverse struct {
	UnaryFunctionBase
}

func NewArrayReverse(operand Expression) Function {
	return &ArrayReverse{
		*NewUnaryFunctionBase("array_reverse", operand),
	}
}

func (this *ArrayReverse) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayReverse) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayReverse) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	n := len(aa)
	ra := make([]interface{}, n)
	n--
	for i, _ := range aa {
		ra[i] = aa[n-i]
	}

	return value.NewValue(ra), nil
}

func (this *ArrayReverse) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayReverse(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArraySort
//
///////////////////////////////////////////////////

type ArraySort struct {
	UnaryFunctionBase
}

func NewArraySort(operand Expression) Function {
	return &ArraySort{
		*NewUnaryFunctionBase("array_sort", operand),
	}
}

func (this *ArraySort) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySort) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArraySort) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	cv := arg.Copy()
	sorter := value.NewSorter(cv)
	sort.Sort(sorter)
	return cv, nil
}

func (this *ArraySort) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArraySort(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArraySum
//
///////////////////////////////////////////////////

type ArraySum struct {
	UnaryFunctionBase
}

func NewArraySum(operand Expression) Function {
	return &ArraySum{
		*NewUnaryFunctionBase("array_sum", operand),
	}
}

func (this *ArraySum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArraySum) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sum := 0.0
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() == value.NUMBER {
			sum += v.Actual().(float64)
		}
	}

	return value.NewValue(sum), nil
}

func (this *ArraySum) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArraySum(operands[0])
	}
}
