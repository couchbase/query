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

func (this *ArrayAppend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayAppend) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayAppend) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayAppend) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayAppend) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayAppend) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayAppend) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayAvg
//
///////////////////////////////////////////////////

type ArrayAvg struct {
	unaryBase
}

func NewArrayAvg(arg Expression) Function {
	return &ArrayAvg{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayAvg) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayAvg) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayAvg) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayAvg) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayAvg) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayAvg) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayAvg) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayAvg(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayConcat
//
///////////////////////////////////////////////////

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

func (this *ArrayConcat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayConcat) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayConcat) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayConcat) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayConcat) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayConcat) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayConcat) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayContains
//
///////////////////////////////////////////////////

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

func (this *ArrayContains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayContains) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayContains) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayContains) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayContains) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayContains) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayContains) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayCount
//
///////////////////////////////////////////////////

type ArrayCount struct {
	unaryBase
}

func NewArrayCount(arg Expression) Function {
	return &ArrayCount{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayCount) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayCount) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayCount) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayCount) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayCount) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayCount) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayCount) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayCount(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayDistinct
//
///////////////////////////////////////////////////

type ArrayDistinct struct {
	unaryBase
}

func NewArrayDistinct(arg Expression) Function {
	return &ArrayDistinct{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayDistinct) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayDistinct) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayDistinct) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayDistinct) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayDistinct) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayDistinct) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayDistinct) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	set := value.NewSet(16)
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		set.Add(value.NewValue(a))
	}

	return value.NewValue(set.Actuals()), nil
}

func (this *ArrayDistinct) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayDistinct(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayIfNull
//
///////////////////////////////////////////////////

type ArrayIfNull struct {
	unaryBase
}

func NewArrayIfNull(arg Expression) Function {
	return &ArrayIfNull{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayIfNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayIfNull) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayIfNull) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayIfNull) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayIfNull) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayIfNull) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayIfNull) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayIfNull(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayLength
//
///////////////////////////////////////////////////

type ArrayLength struct {
	unaryBase
}

func NewArrayLength(arg Expression) Function {
	return &ArrayLength{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayLength) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayLength) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayLength) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayLength) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayLength) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayLength) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	return value.NewValue(float64(len(aa))), nil
}

func (this *ArrayLength) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewArrayLength(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayMax
//
///////////////////////////////////////////////////

type ArrayMax struct {
	unaryBase
}

func NewArrayMax(arg Expression) Function {
	return &ArrayMax{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayMax) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayMax) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayMax) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayMax) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayMax) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayMax) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayMax) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayMax(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayMin
//
///////////////////////////////////////////////////

type ArrayMin struct {
	unaryBase
}

func NewArrayMin(arg Expression) Function {
	return &ArrayMin{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayMin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayMin) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayMin) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayMin) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayMin) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayMin) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayMin) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayMin(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayPosition
//
///////////////////////////////////////////////////

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

func (this *ArrayPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayPosition) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayPosition) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayPosition) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayPosition) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayPosition) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayPosition) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayPrepend
//
///////////////////////////////////////////////////

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

func (this *ArrayPrepend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayPrepend) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayPrepend) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayPrepend) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayPrepend) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayPrepend) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayPrepend) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayPut
//
///////////////////////////////////////////////////

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

func (this *ArrayPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayPut) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayPut) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayPut) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayPut) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayPut) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayPut) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayRange
//
///////////////////////////////////////////////////

type ArrayRange struct {
	nAryBase
}

func NewArrayRange(args Expressions) Function {
	return &ArrayRange{
		nAryBase{
			operands: args,
		},
	}
}

func (this *ArrayRange) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayRange) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayRange) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayRange) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayRange) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayRange) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayRange) eval(args value.Values) (value.Value, error) {
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

func (this *ArrayRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayRemove) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayRemove) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayRemove) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayRemove) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayRemove) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayRemove) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayRepeat
//
///////////////////////////////////////////////////

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

func (this *ArrayRepeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayRepeat) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayRepeat) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayRepeat) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayRepeat) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayRepeat) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayRepeat) eval(first, second value.Value) (value.Value, error) {
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

///////////////////////////////////////////////////
//
// ArrayReplace
//
///////////////////////////////////////////////////

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

func (this *ArrayReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayReplace) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayReplace) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayReplace) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayReplace) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayReplace) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayReplace) eval(args value.Values) (value.Value, error) {
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
	unaryBase
}

func NewArrayReverse(arg Expression) Function {
	return &ArrayReverse{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArrayReverse) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayReverse) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayReverse) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayReverse) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayReverse) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayReverse) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayReverse) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArrayReverse(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArraySort
//
///////////////////////////////////////////////////

type ArraySort struct {
	unaryBase
}

func NewArraySort(arg Expression) Function {
	return &ArraySort{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArraySort) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArraySort) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArraySort) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArraySort) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArraySort) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArraySort) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArraySort) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArraySort(args[0])
	}
}

///////////////////////////////////////////////////
//
// ArraySum
//
///////////////////////////////////////////////////

type ArraySum struct {
	unaryBase
}

func NewArraySum(arg Expression) Function {
	return &ArraySum{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ArraySum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArraySum) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArraySum) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArraySum) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArraySum) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArraySum) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArraySum) eval(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewArraySum(args[0])
	}
}
