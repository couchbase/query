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

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// ArrayAppend
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_APPEND(expr, value).
It returns a new array with value appended. Type ArrayAppend
is a struct that implements BinaryFunctionBase.
*/
type ArrayAppend struct {
	BinaryFunctionBase
}

/*
The function NewArrayAppend calls NewBinaryFunctionBase to
create a function named ARRAY_APPEND with the two
expressions as input.
*/
func NewArrayAppend(first, second Expression) Function {
	rv := &ArrayAppend{
		*NewBinaryFunctionBase("array_append", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayAppend) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayAppend) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayAppend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the array append function. If either
of the input argument types are missing, or not an array return
a missing and null value respectively. Use the append method
to append the second expression to the first expression. Return
the new array.
*/
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

/*
The constructor returns a NewArrayAppend with the two operands
cast to a Function as the FunctionConstructor.
*/
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

/*
This represents the array function ARRAY_AVG(expr). It returns
the arithmetic mean (average) of all the non-NULL number values
in the array, or NULL if there are no such values. Type ArrayAvg
is a struct that implements UnaryFunctionBase.
*/
type ArrayAvg struct {
	UnaryFunctionBase
}

/*
The function NewArrayAvg takes as input an expression and returns
a pointer to the ArrayAvg struct that calls NewUnaryFunctionBase to
create a function named ARRAY_AVG with an input operand as the
expression.
*/
func NewArrayAvg(operand Expression) Function {
	rv := &ArrayAvg{
		*NewUnaryFunctionBase("array_avg", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayAvg) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *ArrayAvg) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
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

/*
The constructor returns a NewArrayAvg with an operand cast to a
Function as the FunctionConstructor.
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
This represents the array function ARRAY_CONCAT(expr1, expr2).
It returns a new array with the concatenation of the input
arrays. Type ArrayConcat is a struct that implements
BinaryFunctionBase.
*/
type ArrayConcat struct {
	BinaryFunctionBase
}

/*
The function NewArrayConcat calls NewBinaryFunctionBase to
create a function named ARRAY_CONCAT with the two
expressions as input.
*/
func NewArrayConcat(first, second Expression) Function {
	rv := &ArrayConcat{
		*NewBinaryFunctionBase("array_concat", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayConcat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayConcat) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayConcat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
The method concatenates two arrays and returns this value. If either
of the input values are missing, return a missing value. For all
non array values, return a null value. Use the append method for
this purpose.
*/
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

/*
The constructor returns a NewArrayConcat with the two operands
cast to a Function as the FunctionConstructor.
*/
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

/*
This represents the array function ARRAY_CONTAINS(expr, value).
It returns true if the array contains value. Type ArrayContains
is a struct that implements BinaryFunctionBase.
*/
type ArrayContains struct {
	BinaryFunctionBase
}

/*
The function NewArrayContains calls NewBinaryFunctionBase to
create a function named ARRAY_CONTAINS with the two
expressions as input.
*/
func NewArrayContains(first, second Expression) Function {
	rv := &ArrayContains{
		*NewBinaryFunctionBase("array_contains", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayContains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *ArrayContains) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
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
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	for _, f := range fa {
		if second.Equals(value.NewValue(f)).Truth() {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
The constructor returns a NewArrayContains with the two operands
cast to a Function as the FunctionConstructor.
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
array, or zero if there are no such values. Type ArrayCount
is a struct that implements UnaryFunctionBase.
*/
type ArrayCount struct {
	UnaryFunctionBase
}

/*
The function NewArrayCount takes as input an expression
and returns a pointer to the ArrayCount struct that calls
NewUnaryFunctionBase to create a function named ARRAY_COUNT
with an input operand as the expression.
*/
func NewArrayCount(operand Expression) Function {
	rv := &ArrayCount{
		*NewUnaryFunctionBase("array_count", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayCount) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *ArrayCount) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayCount) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method calculates the number of elements in the
array. If the input argument is missing return missing
value, and if it isnt an array then return a null value.
Range through the array and count the values that are'nt
null and missing. Return this value.
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
The constructor returns a NewArrayCount with an operand cast to a
Function as the FunctionConstructor.
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
It returns a new array with distinct elements of input
array. Type ArrayDistinct is a struct that implements
UnaryFunctionBase.
*/
type ArrayDistinct struct {
	UnaryFunctionBase
}

/*
The function NewArrayDistinct takes as input an expression
and returns a pointer to the ArrayDistinct struct that
calls NewUnaryFunctionBase to create a function named
ARRAY_DISTINCT with an input operand as the expression.
*/
func NewArrayDistinct(operand Expression) Function {
	rv := &ArrayDistinct{
		*NewUnaryFunctionBase("array_distinct", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayDistinct) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
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
	set := value.NewSet(len(aa))
	for _, a := range aa {
		set.Add(value.NewValue(a))
	}

	return value.NewValue(set.Actuals()), nil
}

/*
The constructor returns a NewArrayDistinct with an operand cast to a
Function as the FunctionConstructor.
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
This represents the array function ARRAY_FLATTEN(expr, value).
It returns a new array with value appended. Type ArrayFlatten
is a struct that implements BinaryFunctionBase.
*/
type ArrayFlatten struct {
	BinaryFunctionBase
}

/*
The function NewArrayFlatten calls NewBinaryFunctionBase to
create a function named ARRAY_FLATTEN with the two
expressions as input.
*/
func NewArrayFlatten(first, second Expression) Function {
	rv := &ArrayFlatten{
		*NewBinaryFunctionBase("array_flatten", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayFlatten) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayFlatten) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayFlatten) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the array flatten function. If either
of the input argument types are missing, or not an array return
a missing and null value respectively. Use the append method
to append the second expression to the first expression. Return
the new array.
*/
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

	destArr := make([]interface{}, 0)
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
		el := elem.(value.Value)
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
The constructor returns a NewArrayFlatten with the two operands
cast to a Function as the FunctionConstructor.
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
NULL. Type ArrayIfNull is a struct that implements
UnaryFunctionBase.
*/
type ArrayIfNull struct {
	UnaryFunctionBase
}

/*
The function NewArrayIfNull takes as input an expression
and returns a pointer to the ArrayIfNull struct that calls
NewUnaryFunctionBase to create a function named ARRAY_IFNULL
with an input operand as the expression.
*/
func NewArrayIfNull(operand Expression) Function {
	rv := &ArrayIfNull{
		*NewUnaryFunctionBase("array_ifnull", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayIfNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a JSON value.
*/
func (this *ArrayIfNull) Type() value.Type { return value.JSON }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
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
The constructor returns a NewArrayIfNull with an operand cast to a
Function as the FunctionConstructor.
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
This represents the array function ARRAY_INSERT(value, expr, expr).
It returns a new array with value inserted. Type ArrayInsert
is a struct that implements TernaryFunctionBase.
*/
type ArrayInsert struct {
	TernaryFunctionBase
}

/*
The function NewArrayInsert calls NewTernaryFunctionBase to
create a function named ARRAY_INSERT with the three
expressions as input.
*/
func NewArrayInsert(first, second, third Expression) Function {
	rv := &ArrayInsert{
		*NewTernaryFunctionBase("array_insert", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayInsert) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayInsert) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method inserts the third value to the first value at the second position.
*/
func (this *ArrayInsert) Apply(context Context, first, second, third value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	} else if third.Type() == value.MISSING {
		return first, nil
	}

	/* the position needs to be an integer */
	f := second.Actual().(float64)
	if math.Trunc(f) != f {
		return value.NULL_VALUE, nil
	}

	s := first.Actual().([]interface{})

	n := int(f)

	/* position goes from 0 to end of array */
	if n < 0 || n > len(s) {
		return value.NULL_VALUE, nil
	}

	ra := make([]interface{}, 0, len(s)+1)

	/* corner case: append to the end */
	if n == len(s) {
		ra = append(ra, s...)
		ra = append(ra, third)
	} else {
		ra = append(ra, s[:n]...)
		ra = append(ra, third)
		ra = append(ra, s[n:]...)
	}

	return value.NewValue(ra), nil
}

/*
The constructor returns a NewArrayInsert with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *ArrayInsert) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArrayInsert(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ArrayLength
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_LENGTH(expr).
It returns the number of elements in the array. Type
ArrayLength is a struct that implements UnaryFunctionBase.
*/
type ArrayLength struct {
	UnaryFunctionBase
}

/*
The function NewArrayLength takes as input an expression
and returns a pointer to the ArrayLength struct that
calls NewUnaryFunctionBase to create a function named
ARRAY_LENGTH with an input operand as the expression.
*/
func NewArrayLength(operand Expression) Function {
	rv := &ArrayLength{
		*NewUnaryFunctionBase("array_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *ArrayLength) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the length of the input array using
the len method. If the input value is of type missing
return a missing value, and for all non array values
return null.
*/
func (this *ArrayLength) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	aa := arg.Actual().([]interface{})
	return value.NewValue(float64(len(aa))), nil
}

/*
The constructor returns a NewArrayLength with an operand cast to a
Function as the FunctionConstructor.
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
in N1QL collation order. Type ArrayMax is a struct that
implements UnaryFunctionBase.
*/
type ArrayMax struct {
	UnaryFunctionBase
}

/*
The function NewArrayMax takes as input an expression
and returns a pointer to the ArrayMax struct that calls
NewUnaryFunctionBase to create a function named ARRAY_MAX
with an input operand as the expression.
*/
func NewArrayMax(operand Expression) Function {
	rv := &ArrayMax{
		*NewUnaryFunctionBase("array_max", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayMax) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a JSON value.
*/
func (this *ArrayMax) Type() value.Type { return value.JSON }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
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
The constructor returns a NewArrayMax with an operand cast to a
Function as the FunctionConstructor.
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
collation order. Type ArrayMin is a struct that implements
UnaryFunctionBase.
*/
type ArrayMin struct {
	UnaryFunctionBase
}

/*
The function NewArrayMin takes as input an expression
and returns a pointer to the ArrayMin struct that calls
NewUnaryFunctionBase to create a function named ARRAY_MIN
with an input operand as the expression.
*/
func NewArrayMin(operand Expression) Function {
	rv := &ArrayMin{
		*NewUnaryFunctionBase("array_min", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayMin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a JSON value.
*/
func (this *ArrayMin) Type() value.Type { return value.JSON }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
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
The constructor returns a NewArrayMin with an operand cast to a
Function as the FunctionConstructor.
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
The position is 0-based. Type ArrayPosition is a struct that
implements UnaryFunctionBase.
*/
type ArrayPosition struct {
	BinaryFunctionBase
}

/*
The function NewArrayPosition takes as input two expressions
and returns a pointer to the ArrayPosition struct that calls
NewBinaryFunctionBase to create a function named ARRAY_POSITION
with an input operands as the expressions.
*/
func NewArrayPosition(first, second Expression) Function {
	rv := &ArrayPosition{
		*NewBinaryFunctionBase("array_position", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayPosition) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *ArrayPosition) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method ranges through the array and returns the position
of the second value in the array (first value). If the input
values are of type missing return a missing value, and for all
non array values return null. If not found then return -1.
*/
func (this *ArrayPosition) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	fa := first.Actual().([]interface{})
	for i, f := range fa {
		if second.Equals(value.NewValue(f)).Truth() {
			return value.NewValue(float64(i)), nil
		}
	}

	return value.NewValue(float64(-1)), nil
}

/*
The constructor returns a NewArrayPosition with the operands cast
to a Function as the FunctionConstructor.
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
This represents the array function ARRAY_PREPEND(value, expr).
It returns a new array with value prepended. Type ArrayPrepend
is a struct that implements BinaryFunctionBase.
*/
type ArrayPrepend struct {
	BinaryFunctionBase
}

/*
The function NewArrayPrepend calls NewBinaryFunctionBase to
create a function named ARRAY_PREPEND with the two
expressions as input.
*/
func NewArrayPrepend(first, second Expression) Function {
	rv := &ArrayPrepend{
		*NewBinaryFunctionBase("array_prepend", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayPrepend) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayPrepend) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayPrepend) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method prepends the first value to the second value, by
reversing the input to the append method. If either
of the input argument types are missing, or not an array return
a missing and null value respectively.
*/
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

/*
The constructor returns a NewArrayPrepend with the two operands
cast to a Function as the FunctionConstructor.
*/
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

/*
This represents the array function ARRAY_PUT(expr, value).
It returns a new array with value appended, if value is not
already present; else unmodified input array. Type ArrayPut
is a struct that implements BinaryFunctionBase.
*/
type ArrayPut struct {
	BinaryFunctionBase
}

/*
The function NewArrayPut calls NewBinaryFunctionBase to
create a function named ARRAY_PUT with the two
expressions as input.
*/
func NewArrayPut(first, second Expression) Function {
	rv := &ArrayPut{
		*NewBinaryFunctionBase("array_put", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayPut) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayPut) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method appends the value into the array if it isnt
present. Range over the array and check if the value exists.
If it does return the array as is. If either of the input
argument types are missing, or not an array return a missing
and null value respectively.
*/
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
		if second.Equals(v).Truth() {
			return first, nil
		}
	}

	ra := append(f, second)
	return value.NewValue(ra), nil
}

/*
The constructor returns a NewArrayPut with the two operands
cast to a Function as the FunctionConstructor.
*/
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

/*
This represents the array function ARRAY_RANGE(start, end [, step ]).
It returns a new array of numbers, from start until the largest number
less than end. Successive numbers are incremented by step. If step is
omitted, it defaults to 1. If step is negative, decrements until the
smallest number greater than end. Type ArrayRange is a struct that
implements FunctionBase.
*/
type ArrayRange struct {
	FunctionBase
}

/*
The method NewArrayRange calls NewFunctionBase to
create a function named ARRAY_RANGE with input
arguments as the operands from the input expression.
*/
func NewArrayRange(operands ...Expression) Function {
	rv := &ArrayRange{
		*NewFunctionBase("array_range", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayRange) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayRange) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayRange) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method returns the input arg array from start
until the largest number less than end. Successive
numbers are incremented by step value. If either
of the input arguments are missing or not numbers
then return a missing or null value.
*/
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
Return NewArrayRange as FunctionConstructor.
*/
func (this *ArrayRange) Constructor() FunctionConstructor { return NewArrayRange }

///////////////////////////////////////////////////
//
// ArrayRemove
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_REMOVE(expr, value).
It returns a new array with all occurences of value removed.
Type ArrayRemove is a struct that implements BinaryFunctionBase.
*/
type ArrayRemove struct {
	BinaryFunctionBase
}

/*
The function NewArrayRemove calls NewBinaryFunctionBase to
create a function named ARRAY_REMOVE with the two
expressions as input.
*/
func NewArrayRemove(first, second Expression) Function {
	rv := &ArrayRemove{
		*NewBinaryFunctionBase("array_remove", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayRemove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayRemove) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method removes all the occurences of the second value from the
first array value.
*/
func (this *ArrayRemove) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return first, nil
	}

	if first.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	if second.Type() <= value.NULL {
		return first, nil
	}

	fa := first.Actual().([]interface{})
	if len(fa) == 0 {
		return first, nil
	}

	ra := make([]interface{}, 0, len(fa))
	for _, f := range fa {
		if !second.Equals(value.NewValue(f)).Truth() {
			ra = append(ra, f)
		}
	}

	return value.NewValue(ra), nil
}

/*
The constructor returns a NewArrayRemove with the two operands
cast to a Function as the FunctionConstructor.
*/
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

/*
This represents the array function ARRAY_REPEAT(value, n).
It returns a new array with value repeated n times. Type
ArrayRepeat is a struct that implements BinaryFunctionBase.
*/
type ArrayRepeat struct {
	BinaryFunctionBase
}

/*
The function NewArrayRepeat calls NewBinaryFunctionBase to
create a function named ARRAY_REPEAT with the two
expressions as input.
*/
func NewArrayRepeat(first, second Expression) Function {
	rv := &ArrayRepeat{
		*NewBinaryFunctionBase("array_repeat", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayRepeat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayRepeat) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayRepeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method creates a new slice and repeats the first value
second value number of times. If either of the input values
are missing, return a missing value. If the first value is
less than 0, or not an absolute number then return a null
value.
*/
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
	ra := make([]interface{}, n)
	for i := 0; i < n; i++ {
		ra[i] = first
	}

	return value.NewValue(ra), nil
}

/*
The constructor returns a NewArrayRepeat with the two operands
cast to a Function as the FunctionConstructor.
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
This represents the array function ARRAY_REPLACE(expr, value1, value2 [, n ]).
It returns a new array with all occurences of value1 replaced with value2.
If n is given, at most n replacements are performed. Type ArrayReplace is a
struct that implements FunctionBase.
*/
type ArrayReplace struct {
	FunctionBase
}

/*
The method NewArrayReplace calls NewFunctionBase to
create a function named ARRAY_REPLACE with input
arguments as the operands from the input expression.
*/
func NewArrayReplace(operands ...Expression) Function {
	rv := &ArrayReplace{
		*NewFunctionBase("array_replace", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayReplace) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method returns an array that contains the values
as arg 1, replaced by the 2nd argument value. If a third
input argument is given (n) then at most n replacements
are performed. Return this value.
*/
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
Return NewArrayReplace as FunctionConstructor.
*/
func (this *ArrayReplace) Constructor() FunctionConstructor { return NewArrayReplace }

///////////////////////////////////////////////////
//
// ArrayReverse
//
///////////////////////////////////////////////////

/*
This represents the array function ARRAY_REVERSE(expr).
It returns a new array with all elements in reverse order.
Type ArrayReverse is a struct that implements
UnaryFunctionBase.
*/
type ArrayReverse struct {
	UnaryFunctionBase
}

/*
The function NewArrayReverse takes as input an expression and returns
a pointer to the ArrayReverse struct that calls NewUnaryFunctionBase to
create a function named ARRAY_REVERSE with an input operand as the
expression.
*/
func NewArrayReverse(operand Expression) Function {
	rv := &ArrayReverse{
		*NewUnaryFunctionBase("array_reverse", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayReverse) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArrayReverse) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ArrayReverse) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method reverses the input array value and returns it.
If the input value is of type missing return a missing
value, and for all non array values return null. Range
through the array and add it to a new slice in reverse.
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
The constructor returns a NewArrayReverse with an operand cast to a
Function as the FunctionConstructor.
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
order. Type ArraySort is a struct that implements
UnaryFunctionBase.
*/
type ArraySort struct {
	UnaryFunctionBase
}

/*
The function NewArraySort takes as input an expression and returns
a pointer to the ArrayAvg struct that calls NewUnaryFunctionBase to
create a function named ARRAY_SORT with an input operand as the
expression.
*/
func NewArraySort(operand Expression) Function {
	rv := &ArraySort{
		*NewUnaryFunctionBase("array_sort", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArraySort) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns an Array value.
*/
func (this *ArraySort) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ArraySort) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method sorts the input array value, in N1QL collation
order. It uses the Sort method in the sort package. If the
input value is of type missing return a missing value, and
for all non array values return null.
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
The constructor returns a NewArraySort with an operand cast to a
Function as the FunctionConstructor.
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

/*
Constructor.
*/
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

/*
Result type is OBJECT.
*/
func (this *ArrayStar) Type() value.Type {
	return value.OBJECT
}

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
	dup := make([]value.Value, len(actual))
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

/*
Factory.
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
Type ArraySum is a struct that implements
UnaryFunctionBase.
*/
type ArraySum struct {
	UnaryFunctionBase
}

/*
The function NewArraySum takes as input an expression and returns
a pointer to the ArraySum struct that calls NewUnaryFunctionBase to
create a function named ARRAY_SUM with an input operand as the
expression.
*/
func NewArraySum(operand Expression) Function {
	rv := &ArraySum{
		*NewUnaryFunctionBase("array_sum", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ArraySum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *ArraySum) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ArraySum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the sum of all the fields in the array.
Range through the array and if the type of field is a number
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

/*
The constructor returns a NewArraySum with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ArraySum) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewArraySum(operands[0])
	}
}
