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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// ArrayAppend
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_APPEND(expr, value ...).
It returns a new array with values appended.
*/
type ArrayAppend struct {
	FunctionBase
}

func NewArrayAppend(operands ...Expression) Function {
	rv := &ArrayAppend{
		*NewFunctionBase("array_append", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayAppend) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayAppend) Type() value.Type { return value.ARRAY }

func (this *ArrayAppend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayAppend) PropagatesNull() bool {
	return false
}

func (this *ArrayAppend) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}
	}

	first := args[0]
	if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().([]interface{})
	for _, arg := range args[1:] {
		f = append(f, arg)
	}

	return value.NewValue(f), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayAppend) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayAppend) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayAppend) Constructor() FunctionConstructor {
	return NewArrayAppend
}

///////////////////////////////////////////////////
//
// ArrayAvg
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_AVG(expr). It returns
the arithmetic mean (average) of all the non-NULL number values
in the array, or NULL if there are no such values.
*/
type ArrayAvg struct {
	UnaryFunctionBase
}

func NewArrayAvg(operand Expression) Function {
	rv := &ArrayAvg{
		*NewUnaryFunctionBase("array_avg", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayAvg) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayAvg) Type() value.Type { return value.NUMBER }

func (this *ArrayAvg) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method evaluates the avg value for the array. If the input
value is of type missing return a missing value, and for all
non array values return null. Calculate the average of the
values in the slice and return that value. If the array size
is 0 return a null value.
*/
func (this *ArrayAvg) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sum := value.ZERO_NUMBER
	count := 0
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() == value.NUMBER {
			sum = sum.Add(value.AsNumberValue(v))
			count++
		}
	}

	if count == 0 {
		return value.NULL_VALUE, nil
	} else {
		return value.NewValue(sum.Actual().(float64) / float64(count)), nil
	}
}

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_CONCAT(expr1, expr2 ...).
It returns a new array with the concatenation of the input
arrays.
*/
type ArrayConcat struct {
	FunctionBase
}

func NewArrayConcat(operands ...Expression) Function {
	rv := &ArrayConcat{
		*NewFunctionBase("array_concat", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayConcat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayConcat) Type() value.Type { return value.ARRAY }

func (this *ArrayConcat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayConcat) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.ARRAY {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	f := args[0].Actual().([]interface{})
	for _, arg := range args[1:] {
		a := arg.Actual().([]interface{})
		f = append(f, a...)
	}

	return value.NewValue(f), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayConcat) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayConcat) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayConcat) Constructor() FunctionConstructor {
	return NewArrayConcat
}

///////////////////////////////////////////////////
//
// ArrayContains
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_CONTAINS(expr, value).
It returns true if the array contains value.
*/
type ArrayContains struct {
	BinaryFunctionBase
}

func NewArrayContains(first, second Expression) Function {
	rv := &ArrayContains{
		*NewBinaryFunctionBase("array_contains", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayContains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayContains) Type() value.Type { return value.BOOLEAN }

func (this *ArrayContains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *ArrayContains) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
This method checks if the first array value contains the second
value and returns true; else false. If either of the input
argument types are missing, then return a missing value. If the
first value is not an array return Null value. Range over the array
and call equals to check if the second value exists and retunr true
if it does.
*/
func (this *ArrayContains) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY || second.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	for _, f := range fa {
		fv := value.NewValue(f)
		if second.Equals(fv).Truth() {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
Factory method pattern.
*/
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
/*
This represents the array function ARRAY_COUNT(expr).
It resturns a count of all the non-NULL values in the
array, or zero if there are no such values.
*/
type ArrayCount struct {
	UnaryFunctionBase
}

func NewArrayCount(operand Expression) Function {
	rv := &ArrayCount{
		*NewUnaryFunctionBase("array_count", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayCount) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayCount) Type() value.Type { return value.NUMBER }

func (this *ArrayCount) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method calculates the number of non-NULL elements in the
array. If the input argument is missing return missing value, and if
it is not an array then return a null value.  Range through the array
and count the values that are not null and missing. Return this value.
*/
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

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_DISTINCT(expr).
It returns a new array with distinct elements of the input
array.
*/
type ArrayDistinct struct {
	UnaryFunctionBase
}

func NewArrayDistinct(operand Expression) Function {
	rv := &ArrayDistinct{
		*NewUnaryFunctionBase("array_distinct", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayDistinct) Type() value.Type { return value.ARRAY }

func (this *ArrayDistinct) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the input array with distinct elements.
If the input value is of type missing return a missing
value, and for all non array values return null. Create
a new set and add all distinct values to the set. Return it.
*/
func (this *ArrayDistinct) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	naa := make([]interface{}, 0, len(aa))
	set := value.NewSet(len(aa), false, false)
	for _, a := range aa {
		av := value.NewValue(a)
		if !set.Has(av) {
			set.Add(av)
			naa = append(naa, a)
		}
	}

	return value.NewValue(naa), nil
}

/*
Factory method pattern.
*/
func (this *ArrayDistinct) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayDistinct(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayFlatten
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_FLATTEN(expr, depth).  Nested
array elements are flattened into the top-level array, up to the given
depth.
*/
type ArrayFlatten struct {
	BinaryFunctionBase
}

func NewArrayFlatten(first, second Expression) Function {
	rv := &ArrayFlatten{
		*NewBinaryFunctionBase("array_flatten", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayFlatten) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayFlatten) Type() value.Type { return value.ARRAY }

func (this *ArrayFlatten) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayFlatten) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	arr := first.Actual().([]interface{})
	fdepth := second.Actual().(float64)

	// Second parameter must be an integer.
	if math.Trunc(fdepth) != fdepth {
		return value.NULL_VALUE, nil
	}
	depth := int(fdepth)

	destArr := make([]interface{}, 0, 4*len(arr))
	destArr = arrayFlattenInto(arr, destArr, depth)
	return value.NewValue(destArr), nil
}

func arrayFlattenInto(sourceArr, destArr []interface{}, depth int) []interface{} {
	// Just copy the contents of the source array into the destination array.
	if depth == 0 {
		return append(destArr, sourceArr...)
	}

	// Copy the elements into the destination array.
	// Recurse as necessary.
	for _, elem := range sourceArr {
		el := value.NewValue(elem)
		if el.Type() == value.ARRAY {
			subArr := el.Actual().([]interface{})
			destArr = arrayFlattenInto(subArr, destArr, depth-1)
		} else {
			destArr = append(destArr, elem)
		}
	}

	return destArr
}

/*
Factory method pattern.
*/
func (this *ArrayFlatten) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayFlatten(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayIfNull
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_IFNULL(expr).
It returns the first non-NULL value in the array, or
NULL.
*/
type ArrayIfNull struct {
	UnaryFunctionBase
}

func NewArrayIfNull(operand Expression) Function {
	rv := &ArrayIfNull{
		*NewUnaryFunctionBase("array_ifnull", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayIfNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayIfNull) Type() value.Type { return value.JSON }

func (this *ArrayIfNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method ranges through the array and returns the first
non null value in the array. It returns missing if input
type is missing and null for non array values.
*/
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

/*
Factory method pattern.
*/
func (this *ArrayIfNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayIfNull(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayInsert
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_INSERT(expr, pos, value ...).
It returns a new array with value inserted.
*/
type ArrayInsert struct {
	FunctionBase
}

func NewArrayInsert(operands ...Expression) Function {
	rv := &ArrayInsert{
		*NewFunctionBase("array_insert", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayInsert) Type() value.Type { return value.ARRAY }

func (this *ArrayInsert) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayInsert) PropagatesNull() bool {
	return false
}

/*
This method inserts the third and subsequent values into the first
array at the second position.
*/
func (this *ArrayInsert) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}
	}

	first := args[0]
	second := args[1]
	if first.Type() != value.ARRAY || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	/* the position needs to be an integer */
	f := second.Actual().(float64)
	if math.Trunc(f) != f {
		return value.NULL_VALUE, nil
	}

	s := first.Actual().([]interface{})
	n := int(f)

	/* count negative position from end of array */
	if n < 0 {
		n = len(s) + n
	}

	/* position goes from 0 to end of array */
	if n < 0 || n > len(s) {
		return value.NULL_VALUE, nil
	}

	ra := make([]interface{}, 0, len(s)+len(args)-2)

	/* corner case: append to the end */
	if n == len(s) {
		ra = append(ra, s...)
		for _, a := range args[2:] {
			ra = append(ra, a)
		}
	} else {
		ra = append(ra, s[:n]...)
		for _, a := range args[2:] {
			ra = append(ra, a)
		}
		ra = append(ra, s[n:]...)
	}

	return value.NewValue(ra), nil
}

/*
Minimum input arguments required is 3.
*/
func (this *ArrayInsert) MinArgs() int { return 3 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayInsert) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayInsert) Constructor() FunctionConstructor {
	return NewArrayInsert
}

///////////////////////////////////////////////////
//
// ArrayIntersect
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_INTERSECT(expr1, expr2 ...).
It returns a new array with the intersection of the input arrays.
*/
type ArrayIntersect struct {
	FunctionBase
}

func NewArrayIntersect(operands ...Expression) Function {
	rv := &ArrayIntersect{
		*NewFunctionBase("array_intersect", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayIntersect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayIntersect) Type() value.Type { return value.ARRAY }

func (this *ArrayIntersect) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayIntersect) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	maxLength := 0
	minLength := math.MaxInt32
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.ARRAY {
			null = true
		} else if !null {
			a := arg.Actual().([]interface{})
			n := len(a)
			if n > maxLength {
				maxLength = n
			}
			if n < minLength {
				minLength = n
			}
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	bag := value.NewBag(maxLength)
	for _, arg := range args {
		a := arg.Actual().([]interface{})
		// De-dup each array
		set := value.NewSet(len(a), true, false)
		for _, elem := range a {
			set.Add(value.NewValue(elem))
		}

		// Add to multi-set
		for _, elem := range set.Values() {
			if elem != nil && elem.Type() > value.NULL {
				bag.Add(elem)
			}
		}
	}

	ra := make([]interface{}, 0, minLength)
	n := len(args)
	for _, entry := range bag.Entries() {
		if entry.Count == n {
			ra = append(ra, entry.Value)
		}
	}

	return value.NewValue(ra), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayIntersect) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayIntersect) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayIntersect) Constructor() FunctionConstructor {
	return NewArrayIntersect
}

///////////////////////////////////////////////////
//
// ArrayLength
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_LENGTH(expr).
It returns the number of elements in the array.
*/
type ArrayLength struct {
	UnaryFunctionBase
}

func NewArrayLength(operand Expression) Function {
	rv := &ArrayLength{
		*NewUnaryFunctionBase("array_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayLength) Type() value.Type { return value.NUMBER }

func (this *ArrayLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the length of the input array. If the input value
is of type missing return a missing value, and for all non array
values return null.
*/
func (this *ArrayLength) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	return value.NewValue(len(aa)), nil
}

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_MAX(expr). It
returns the largest non-NULL, non-MISSING array element,
in N1QL collation order.
*/
type ArrayMax struct {
	UnaryFunctionBase
}

func NewArrayMax(operand Expression) Function {
	rv := &ArrayMax{
		*NewUnaryFunctionBase("array_max", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayMax) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayMax) Type() value.Type { return value.JSON }

func (this *ArrayMax) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the largest value in the array based
on N1QL's collation order. If the input value is of type
missing return a missing value, and for all non array
values return null.
*/
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

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_MIN(expr). It returns
the smallest non-NULL, non-MISSING array element, in N1QL
collation order.
*/
type ArrayMin struct {
	UnaryFunctionBase
}

func NewArrayMin(operand Expression) Function {
	rv := &ArrayMin{
		*NewUnaryFunctionBase("array_min", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayMin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayMin) Type() value.Type { return value.JSON }

func (this *ArrayMin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the smallest value in the array based
on N1QL's collation order. If the input value is of type
missing return a missing value, and for all non array
values return null.
*/
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

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_POSITION(expr, value).
It returns the first position of value within the array, or -1.
The position is 0-based.
*/
type ArrayPosition struct {
	BinaryFunctionBase
}

func NewArrayPosition(first, second Expression) Function {
	rv := &ArrayPosition{
		*NewBinaryFunctionBase("array_position", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayPosition) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayPosition) Type() value.Type { return value.NUMBER }

func (this *ArrayPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method ranges through the array and returns the position
of the second value in the array (first value). If either input
values is of type missing return a missing value, and for all
non array values return null. If not found then return -1.
*/
func (this *ArrayPosition) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY || second.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	for i, f := range fa {
		fv := value.NewValue(f)
		if second.Equals(fv).Truth() {
			return value.NewValue(i), nil
		}
	}

	return value.NEG_ONE_VALUE, nil
}

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_PREPEND(value ..., expr).
It returns a new array with values prepended.
*/
type ArrayPrepend struct {
	FunctionBase
}

func NewArrayPrepend(operands ...Expression) Function {
	rv := &ArrayPrepend{
		*NewFunctionBase("array_prepend", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayPrepend) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayPrepend) Type() value.Type { return value.ARRAY }

func (this *ArrayPrepend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayPrepend) PropagatesNull() bool {
	return false
}

func (this *ArrayPrepend) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}
	}

	n := len(args) - 1
	last := args[n]
	if last.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	s := last.Actual().([]interface{})
	ra := make([]interface{}, 0, len(s)+n)
	for _, arg := range args[:n] {
		ra = append(ra, arg)
	}

	ra = append(ra, s...)
	return value.NewValue(ra), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayPrepend) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayPrepend) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayPrepend) Constructor() FunctionConstructor {
	return NewArrayPrepend
}

///////////////////////////////////////////////////
//
// ArrayPut
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_PUT(expr, value ...).  It
returns a new array with each value appended, if value is not already
present; else unmodified input array.
*/
type ArrayPut struct {
	FunctionBase
}

func NewArrayPut(operands ...Expression) Function {
	rv := &ArrayPut{
		*NewFunctionBase("array_put", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayPut) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayPut) Type() value.Type { return value.ARRAY }

func (this *ArrayPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayPut) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() == value.NULL {
			null = true
		}
	}

	first := args[0]
	if null || first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	pa := args[1:]
	fa := first.Actual().([]interface{})
	ra := fa
ploop:
	for _, p := range pa {
		for _, f := range fa {
			fv := value.NewValue(f)
			if p.Equals(fv).Truth() {
				continue ploop
			}
		}

		ra = append(ra, p)
	}

	return value.NewValue(ra), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayPut) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayPut) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayPut) Constructor() FunctionConstructor {
	return NewArrayPut
}

///////////////////////////////////////////////////
//
// ArrayRange
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_RANGE(start, end [, step ]).
It returns a new array of numbers, from start until the largest number
less than end. Successive numbers are incremented by step. If step is
omitted, it defaults to 1. If step is negative, decrements until the
smallest number greater than end.
*/
type ArrayRange struct {
	FunctionBase
}

func NewArrayRange(operands ...Expression) Function {
	rv := &ArrayRange{
		*NewFunctionBase("array_range", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayRange) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayRange) Type() value.Type { return value.ARRAY }

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

	capacity := int(math.Abs(end-start) / math.Abs(step))
	if capacity > RANGE_LIMIT {
		return nil, errors.NewRangeError("ARRAY_RANGE()")
	}

	rv := make([]interface{}, 0, capacity)
	for v := start; (step > 0.0 && v < end) || (step < 0.0 && v > end); v += step {
		rv = append(rv, v)
	}

	return value.NewValue(rv), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayRange) MinArgs() int { return 2 }

/*
Maximum input arguments allowed is 3.
*/
func (this *ArrayRange) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *ArrayRange) Constructor() FunctionConstructor {
	return NewArrayRange
}

///////////////////////////////////////////////////
//
// ArrayRemove
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_REMOVE(expr, value ...).
It returns a new array with all occurences of values removed.
*/
type ArrayRemove struct {
	FunctionBase
}

func NewArrayRemove(operands ...Expression) Function {
	rv := &ArrayRemove{
		*NewFunctionBase("array_remove", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayRemove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayRemove) Type() value.Type { return value.ARRAY }

func (this *ArrayRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayRemove) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() == value.NULL {
			null = true
		}
	}

	first := args[0]
	if null || first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	if len(fa) == 0 {
		return first, nil
	}

	aa := args[1:]
	ra := make([]interface{}, 0, len(fa))
floop:
	for _, f := range fa {
		fv := value.NewValue(f)
		for _, a := range aa {
			if fv.Equals(a).Truth() {
				continue floop
			}
		}

		ra = append(ra, f)
	}

	return value.NewValue(ra), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayRemove) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayRemove) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayRemove) Constructor() FunctionConstructor {
	return NewArrayRemove
}

///////////////////////////////////////////////////
//
// ArrayRepeat
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_REPEAT(value, n).
It returns a new array with value repeated n times.
*/
type ArrayRepeat struct {
	BinaryFunctionBase
}

func NewArrayRepeat(first, second Expression) Function {
	rv := &ArrayRepeat{
		*NewBinaryFunctionBase("array_repeat", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayRepeat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayRepeat) Type() value.Type { return value.ARRAY }

func (this *ArrayRepeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayRepeat) PropagatesNull() bool {
	return false
}

func (this *ArrayRepeat) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	sf := second.Actual().(float64)
	if sf < 0 || sf != math.Trunc(sf) {
		return value.NULL_VALUE, nil
	}

	n := int(sf)
	if n > RANGE_LIMIT {
		return nil, errors.NewRangeError("ARRAY_REPEAT()")
	}

	ra := make([]interface{}, n)
	for i := 0; i < n; i++ {
		ra[i] = first
	}

	return value.NewValue(ra), nil
}

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_REPLACE(expr, value1, value2
[, n ]).  It returns a new array with all occurences of value1
replaced with value2.  If n is given, at most n replacements are
performed.
*/
type ArrayReplace struct {
	FunctionBase
}

func NewArrayReplace(operands ...Expression) Function {
	rv := &ArrayReplace{
		*NewFunctionBase("array_replace", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayReplace) Type() value.Type { return value.ARRAY }

func (this *ArrayReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayReplace) PropagatesNull() bool {
	return false
}

func (this *ArrayReplace) Apply(context Context, args ...value.Value) (value.Value, error) {
	av := args[0]
	v1 := args[1]
	v2 := args[2]

	if av.Type() == value.MISSING || v1.Type() == value.MISSING || v2.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if av.Type() != value.ARRAY || v1.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	aa := av.Actual().([]interface{})
	ra := make([]interface{}, 0, len(aa))
	for _, a := range aa {
		v := value.NewValue(a)
		if v1.Equals(v).Truth() {
			ra = append(ra, v2)
		} else {
			ra = append(ra, v)
		}
	}

	return value.NewValue(ra), nil
}

/*
Minimum input arguments required is 3.
*/
func (this *ArrayReplace) MinArgs() int { return 3 }

/*
Maximum input arguments allowed is 4.
*/
func (this *ArrayReplace) MaxArgs() int { return 4 }

/*
Factory method pattern.
*/
func (this *ArrayReplace) Constructor() FunctionConstructor {
	return NewArrayReplace
}

///////////////////////////////////////////////////
//
// ArrayReverse
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_REVERSE(expr).
It returns a new array with all elements in reverse order.
*/
type ArrayReverse struct {
	UnaryFunctionBase
}

func NewArrayReverse(operand Expression) Function {
	rv := &ArrayReverse{
		*NewUnaryFunctionBase("array_reverse", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayReverse) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayReverse) Type() value.Type { return value.ARRAY }

func (this *ArrayReverse) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method reverses the input array value and returns it.
If the input value is of type missing return a missing
value, and for all non array values return null.
*/
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

/*
Factory method pattern.
*/
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

/*
This represents the array function ARRAY_SORT(expr). It
returns a new array with elements sorted in N1QL collation
order.
*/
type ArraySort struct {
	UnaryFunctionBase
}

func NewArraySort(operand Expression) Function {
	rv := &ArraySort{
		*NewUnaryFunctionBase("array_sort", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArraySort) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySort) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ArraySort) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method sorts the input array value, in N1QL collation order. If
the input value is of type missing return a missing value, and for all
non array values return null.
*/
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

/*
Factory method pattern.
*/
func (this *ArraySort) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArraySort(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArrayStar
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_STAR(expr). It converts an
array of objects into an object of arrays.
*/
type ArrayStar struct {
	UnaryFunctionBase
}

func NewArrayStar(operand Expression) Function {
	rv := &ArrayStar{
		*NewUnaryFunctionBase("array_star", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayStar) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayStar) Type() value.Type { return value.OBJECT }

func (this *ArrayStar) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ArrayStar) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	actual := arg.Actual().([]interface{})
	if len(actual) == 0 {
		return _EMPTY_OBJECT, nil
	}

	// Wrap the elements in Values
	var dupBuf [256]value.Value
	var dup []value.Value
	if len(actual) <= len(dupBuf) {
		dup = dupBuf[0:len(actual)]
	} else {
		dup = _DUP_POOL.GetSized(len(actual))
		defer _DUP_POOL.Put(dup)
	}

	for i, a := range actual {
		dup[i] = value.NewValue(a)
	}

	// Collect all the names in the elements
	pairs := make(map[string]interface{}, len(dup[0].Fields()))
	for _, d := range dup {
		fields := d.Fields()
		for f, _ := range fields {
			pairs[f] = nil
		}
	}

	// Allocate and populate array for each name
	for name, _ := range pairs {
		vals := make([]interface{}, len(dup))
		pairs[name] = vals

		for i, d := range dup {
			vals[i], _ = d.Field(name)
		}
	}

	return value.NewValue(pairs), nil
}

var _DUP_POOL = value.NewValuePool(1024)

/*
Factory method pattern.
*/
func (this *ArrayStar) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayStar(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArraySum
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_SUM(expr).
It returns the sum of all the non-NULL number values
in the array, or zero if there are no such values.
*/
type ArraySum struct {
	UnaryFunctionBase
}

func NewArraySum(operand Expression) Function {
	rv := &ArraySum{
		*NewUnaryFunctionBase("array_sum", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArraySum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySum) Type() value.Type { return value.NUMBER }

func (this *ArraySum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the sum of all the elements in the array.
Range through the array and if the type of element is a number
then add it to the sum. Return 0 if no number value exists.
If the input value is of type missing return a missing value,
and for all non array values return null.
*/
func (this *ArraySum) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sum := value.ZERO_NUMBER
	aa := arg.Actual().([]interface{})
	for _, a := range aa {
		v := value.NewValue(a)
		if v.Type() == value.NUMBER {
			sum = sum.Add(value.AsNumberValue(v))
		}
	}

	return sum, nil
}

/*
Factory method pattern.
*/
func (this *ArraySum) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArraySum(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ArraySymdiff1
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_SYMDIFF1(expr1, expr2 ...).
It returns a new array based on the set symmetric difference, or
disjunctive union, of the input arrays. The new array contains only
those elements that appear in exactly one of the input arrays.
*/
type ArraySymdiff1 struct {
	FunctionBase
}

func NewArraySymdiff1(operands ...Expression) Function {
	rv := &ArraySymdiff1{
		*NewFunctionBase("array_symdiff1", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArraySymdiff1) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySymdiff1) Type() value.Type { return value.ARRAY }

func (this *ArraySymdiff1) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArraySymdiff1) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.ARRAY {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	bag := _ARRAY_BAG_POOL.Get()
	defer _ARRAY_BAG_POOL.Put(bag)
	for _, arg := range args {
		set := _ARRAY_SET_POOL.Get()
		defer _ARRAY_SET_POOL.Put(set)
		a := arg.Actual().([]interface{})
		set.AddAll(a)
		bag.AddAll(set.Items())
	}

	entries := bag.Entries()
	rv := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		if entry.Count == 1 {
			rv = append(rv, entry.Value)
		}
	}

	return value.NewValue(rv), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArraySymdiff1) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArraySymdiff1) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArraySymdiff1) Constructor() FunctionConstructor {
	return NewArraySymdiff1
}

///////////////////////////////////////////////////
//
// ArraySymdiffn
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_SYMDIFFN(expr1, expr2 ...).
It returns a new array based on the set symmetric difference, or
disjunctive union, of the input arrays. The new array contains only
those elements that appear in an odd number of the input arrays.
*/
type ArraySymdiffn struct {
	FunctionBase
}

func NewArraySymdiffn(operands ...Expression) Function {
	rv := &ArraySymdiffn{
		*NewFunctionBase("array_symdiffn", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArraySymdiffn) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySymdiffn) Type() value.Type { return value.ARRAY }

func (this *ArraySymdiffn) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArraySymdiffn) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.ARRAY {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	bag := _ARRAY_BAG_POOL.Get()
	defer _ARRAY_BAG_POOL.Put(bag)
	for _, arg := range args {
		set := _ARRAY_SET_POOL.Get()
		defer _ARRAY_SET_POOL.Put(set)
		a := arg.Actual().([]interface{})
		set.AddAll(a)
		bag.AddAll(set.Items())
	}

	entries := bag.Entries()
	rv := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		if (entry.Count & 1) == 1 {
			rv = append(rv, entry.Value)
		}
	}

	return value.NewValue(rv), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArraySymdiffn) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArraySymdiffn) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArraySymdiffn) Constructor() FunctionConstructor {
	return NewArraySymdiffn
}

///////////////////////////////////////////////////
//
// ArrayUnion
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_UNION(expr1, expr2 ...).  It
returns a new array with the set union of the input arrays.
*/
type ArrayUnion struct {
	FunctionBase
}

func NewArrayUnion(operands ...Expression) Function {
	rv := &ArrayUnion{
		*NewFunctionBase("array_union", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayUnion) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayUnion) Type() value.Type { return value.ARRAY }

func (this *ArrayUnion) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayUnion) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	for _, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.ARRAY {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	set := _ARRAY_SET_POOL.Get()
	defer _ARRAY_SET_POOL.Put(set)
	for _, arg := range args {
		a := arg.Actual().([]interface{})
		set.AddAll(a)
	}

	return value.NewValue(set.Items()), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ArrayUnion) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ArrayUnion) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayUnion) Constructor() FunctionConstructor {
	return NewArrayUnion
}

var _ARRAY_SET_POOL = value.NewSetPool(64, true, false)
var _ARRAY_BAG_POOL = value.NewBagPool(64)

///////////////////////////////////////////////////
//
// ArraySwap
//
///////////////////////////////////////////////////
/*
This represents the array function ARRAY_Swap(array,oldpos,newpos).  It
returns an array with the elements at oldpos and newpos switched positions with each other.
both oldpos and newpos are 0-based with negative reverse index accepted.
*/
type ArraySwap struct {
	TernaryFunctionBase
}

func NewArraySwap(first, second, third Expression) Function {
	rv := &ArraySwap{
		*NewTernaryFunctionBase("array_swap", first, second, third),
	}

	rv.expr = rv
	return rv
}

func (this *ArraySwap) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArraySwap) Type() value.Type { return value.ARRAY }

func (this *ArraySwap) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *ArraySwap) PropagatesNull() bool {
	return false
}

func (this *ArraySwap) Apply(context Context, first, second, third value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING || third.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() != value.ARRAY || second.Type() != value.NUMBER || third.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	oldPos := second.Actual().(float64)
	newPos := third.Actual().(float64)

	//both position should be integer.
	if math.Trunc(oldPos) != oldPos || math.Trunc(newPos) != newPos {
		return value.NULL_VALUE, nil
	}

	op := int(oldPos)
	np := int(newPos)

	a := first.Actual().([]interface{})
	length := len(a)

	//out of range check on the index.
	if op < -length || op > length-1 || np < -length || np > length-1 {
		return value.NULL_VALUE, nil
	}

	op = (op + length) % length
	np = (np + length) % length

	//do not switch with self.
	if op == np {
		return first, nil
	}

	a[op], a[np] = a[np], a[op]
	return value.NewValue(a), nil
}

func (this *ArraySwap) MinArgs() int { return 3 }

func (this *ArraySwap) MaxArgs() int { return 3 }

func (this *ArraySwap) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArraySwap(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ArrayMove
//
///////////////////////////////////////////////////
/*
This represents the array function ARRAY_Move(array,oldpos,newpos).  It
returns an array with the elements originally at oldpos moved to newpos.
both oldpos and newpos are 0-based with negative reverse index accepted.
*/

type ArrayMove struct {
	TernaryFunctionBase
}

func NewArrayMove(first, second, third Expression) Function {
	rv := &ArrayMove{
		*NewTernaryFunctionBase("array_move", first, second, third),
	}

	rv.expr = rv
	return rv
}

func (this *ArrayMove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayMove) Type() value.Type { return value.ARRAY }

func (this *ArrayMove) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *ArrayMove) PropagatesNull() bool {
	return false
}

func (this *ArrayMove) Apply(context Context, first, second, third value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING || third.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() != value.ARRAY || second.Type() != value.NUMBER || third.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	oldPos := second.Actual().(float64)
	newPos := third.Actual().(float64)

	//both position should be integer.
	if math.Trunc(oldPos) != oldPos || math.Trunc(newPos) != newPos {
		return value.NULL_VALUE, nil
	}

	op := int(oldPos)
	np := int(newPos)
	a := first.Actual().([]interface{})
	length := len(a)

	//out of range check on the index.
	if op < -length || op > length-1 || np < -length || np > length-1 {
		return value.NULL_VALUE, nil
	}

	op = (op + length) % length
	np = (np + length) % length

	//check if the old position and new position are same.
	if op == np {
		return first, nil
	}

	v := a[op]

	//remove the element at old position:
	for i := op; i < length-1; i++ {
		a[i] = a[i+1]
	}

	//insert the element at the new position:
	for i, _ := range a {
		if length-1-i <= np {
			break
		}
		a[length-i-1] = a[length-i-2]
	}
	a[np] = v

	return value.NewValue(a), nil
}

func (this *ArrayMove) MinArgs() int { return 3 }

func (this *ArrayMove) MaxArgs() int { return 3 }

func (this *ArrayMove) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayMove(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ArrayExcept
//
///////////////////////////////////////////////////
/*
This represents the array function ARRAY_EXCEPT(array A,array B).
It returns an array with the elements that belong to A and not to B.
*/
type ArrayExcept struct {
	BinaryFunctionBase
}

func NewArrayExcept(first, second Expression) Function {
	rv := &ArrayExcept{
		*NewBinaryFunctionBase("array_except", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *ArrayExcept) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayExcept) Type() value.Type { return value.ARRAY }

func (this *ArrayExcept) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayExcept) PropagatesNull() bool {
	return false
}

func (this *ArrayExcept) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() != value.ARRAY || second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	a := first.Actual().([]interface{})
	b := second.Actual().([]interface{})
	if len(a) == 0 || len(b) == 0 {
		return first, nil
	}

	set := _ARRAY_SET_POOL.Get()
	defer _ARRAY_SET_POOL.Put(set)
	set.AddAll(b)

	j := 0
	for i, _ := range a {
		v := value.NewValue(a[i])
		if !set.Has(v) {
			a[j] = a[i]
			j++
		}
	}

	res := value.NewValue(a[:j])

	//To avoid memory leakage
	for ; j < len(a); j++ {
		a[j] = nil
	}

	return res, nil
}

func (this *ArrayExcept) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayExcept(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// ArrayBinarySearch
//
///////////////////////////////////////////////////
/*
This represents the array function ARRAY_BINARY_SEARCH(array A, value B).
It returns the 1-based position of value B in array A
using binary search algorithm and suppose A is sorted.
If B is not found in A, returns -1.
*/

type ArrayBinarySearch struct {
	BinaryFunctionBase
}

func NewArrayBinarySearch(first, second Expression) Function {
	rv := &ArrayBinarySearch{
		*NewBinaryFunctionBase("array_binary_search", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *ArrayBinarySearch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ArrayBinarySearch) Type() value.Type { return value.NUMBER }

func (this *ArrayBinarySearch) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *ArrayBinarySearch) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	a := first.Actual().([]interface{})
	low := 0
	high := len(a) - 1

	for low <= high {
		mid := low + (high-low)/2
		v := value.NewValue(a[mid])

		if v.EquivalentTo(second) {
			return value.NewValue(mid), nil
		} else if v.Collate(second) < 0 {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return value.NEG_ONE_NUMBER, nil
}

func (this *ArrayBinarySearch) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayBinarySearch(operands[0], operands[1])
	}
}
