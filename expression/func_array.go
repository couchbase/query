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
