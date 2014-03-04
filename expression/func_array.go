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

type ArrayAppend struct {
	binaryBase
}

func NewArrayAppend(first, second Expression) Function {
	return &ArrayAppend{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayAppend) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	} else if second.Type() == value.MISSING {
		return first, nil
	}

	f := first.Actual().([]interface{})
	ra := append(f, second)
	return value.NewValue(ra), nil
}

func (this *ArrayAppend) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayAppend(args[0], args[1])
	}
}

type ArrayConcat struct {
	binaryBase
}

func NewArrayConcat(first, second Expression) Function {
	return &ArrayConcat{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayConcat) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayConcat(args[0], args[1])
	}
}

type ArrayContains struct {
	binaryBase
}

func NewArrayContains(first, second Expression) Function {
	return &ArrayContains{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayContains) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayContains(args[0], args[1])
	}
}

type ArrayDistinct struct {
	unaryBase
}

func NewArrayDistinct(operand Expression) Function {
	return &ArrayDistinct{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArrayDistinct) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	set := value.NewSet(16)
	oa := operand.Actual().([]interface{})
	for _, a := range oa {
		set.Add(value.NewValue(a))
	}

	return value.NewValue(set.Actuals()), nil
}

func (this *ArrayDistinct) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayDistinct(args[0])
	}
}

type ArrayIfNull struct {
	unaryBase
}

func NewArrayIfNull(operand Expression) Function {
	return &ArrayIfNull{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArrayIfNull) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	oa := operand.Actual().([]interface{})
	for _, a := range oa {
		v := value.NewValue(a)
		if v.Type() > value.NULL {
			return v, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *ArrayIfNull) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayIfNull(args[0])
	}
}

type ArrayLength struct {
	unaryBase
}

func NewArrayLength(operand Expression) Function {
	return &ArrayLength{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArrayLength) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	oa := operand.Actual().([]interface{})
	return value.NewValue(float64(len(oa))), nil
}

func (this *ArrayLength) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayLength(args[0])
	}
}

type ArrayMax struct {
	unaryBase
}

func NewArrayMax(operand Expression) Function {
	return &ArrayMax{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArrayMax) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	rv := value.NULL_VALUE
	oa := operand.Actual().([]interface{})
	for _, a := range oa {
		v := value.NewValue(a)
		if v.Collate(rv) > 0 {
			rv = v
		}
	}

	return rv, nil
}

func (this *ArrayMax) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayMax(args[0])
	}
}

type ArrayMin struct {
	unaryBase
}

func NewArrayMin(operand Expression) Function {
	return &ArrayMin{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArrayMin) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	rv := value.NULL_VALUE
	oa := operand.Actual().([]interface{})
	for _, a := range oa {
		v := value.NewValue(a)
		if v.Type() > value.NULL &&
			(rv.Type() == value.NULL || v.Collate(rv) < 0) {
			rv = v
		}
	}

	return rv, nil
}

func (this *ArrayMin) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayMin(args[0])
	}
}

type ArrayPosition struct {
	binaryBase
}

func NewArrayPosition(first, second Expression) Function {
	return &ArrayPosition{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayPosition) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayPosition(args[0], args[1])
	}
}

type ArrayPrepend struct {
	binaryBase
}

func NewArrayPrepend(first, second Expression) Function {
	return &ArrayPrepend{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayPrepend) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayPrepend(args[0], args[1])
	}
}

type ArrayPut struct {
	binaryBase
}

func NewArrayPut(first, second Expression) Function {
	return &ArrayPut{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayPut) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayPut(args[0], args[1])
	}
}

type ArrayRemove struct {
	binaryBase
}

func NewArrayRemove(first, second Expression) Function {
	return &ArrayRemove{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayRemove) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayRemove(args[0], args[1])
	}
}

type ArrayRepeat struct {
	binaryBase
}

func NewArrayRepeat(first, second Expression) Function {
	return &ArrayRepeat{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *ArrayRepeat) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayRepeat(args[0], args[1])
	}
}

type ArrayReplace struct {
	nAryBase
}

func NewArrayReplace(args Expressions) Function {
	return &ArrayReplace{
		nAryBase{
			operands: args,
		},
	}
}

func (this *ArrayReplace) evaluate(args value.Values) (value.Value, error) {
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

type ArrayReverse struct {
	unaryBase
}

func NewArrayReverse(operand Expression) Function {
	return &ArrayReverse{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArrayReverse) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	oa := operand.Actual().([]interface{})
	n := len(oa)
	ra := make([]interface{}, n)
	n--
	for i, _ := range oa {
		ra[i] = oa[n-i]
	}

	return value.NewValue(ra), nil
}

func (this *ArrayReverse) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayReverse(args[0])
	}
}

type ArraySort struct {
	unaryBase
}

func NewArraySort(operand Expression) Function {
	return &ArraySort{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ArraySort) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	cv := operand.Copy()
	sorter := value.NewSorter(cv)
	sort.Sort(sorter)
	return cv, nil
}

func (this *ArraySort) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArraySort(args[0])
	}
}
