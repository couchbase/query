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

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// IfInf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFINF(expr1, expr2, ...)
for numbers. It returns the first non-MISSING, non-Inf number or
NULL. Type IfInf is a struct that implements FunctionBase.
*/
type IfInf struct {
	FunctionBase
}

/*
The function NewIfnf calls NewFunctionBase to create a function
named IFINF with input arguments as the operands from the input
expression.
*/
func NewIfInf(operands ...Expression) Function {
	rv := &IfInf{
		*NewFunctionBase("ifinf", operands...),
	}

	rv.conditional = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IfInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *IfInf) Type() value.Type { return value.NUMBER }

func (this *IfInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfInf) DependsOn(other Expression) bool {
	return len(this.operands) > 0 &&
		this.operands[0].DependsOn(other)
}

/*
This method returns the first non missing, non infinity
number in the input argument values. Range over the args
and check its type. If missing, skip that value, and if
that value is not a number, then return a NULL. In the
event a number is first encountered, check whether f is
an infinity according to the second argument (since it
is 0 IsInf reports whether f is either infinity as per
the math package in the Go Docs). If none of the above
cases are satisfied return a Null value.
*/
func (this *IfInf) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsInf(f, 0) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Minimum input arguments required is 2
*/
func (this *IfInf) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the IfInf
function is MaxInt16  = 1<<15 - 1. This is defined using the
math package.
*/
func (this *IfInf) MaxArgs() int { return math.MaxInt16 }

/*
Return NewIfInf as FunctionConstructor.
*/
func (this *IfInf) Constructor() FunctionConstructor { return NewIfInf }

///////////////////////////////////////////////////
//
// IfNaN
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFNAN(expr1, expr2, ...).
It returns the first non-MISSING, non-NaN number or NULL. Type IfNaN
is a struct that implements FunctionBase.
*/
type IfNaN struct {
	FunctionBase
}

/*
The function NewIfNaN calls NewFunctionBase to create a function
named IFNAN with input arguments as the operands from the input
expression.
*/
func NewIfNaN(operands ...Expression) Function {
	rv := &IfNaN{
		*NewFunctionBase("ifnan", operands...),
	}

	rv.conditional = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IfNaN) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *IfNaN) Type() value.Type { return value.NUMBER }

func (this *IfNaN) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfNaN) DependsOn(other Expression) bool {
	return len(this.operands) > 0 &&
		this.operands[0].DependsOn(other)
}

/*
This method returns the first non missing, non infinity
number in the input argument values. Range over the args
and check its type. If missing, skip that value, and if
that value is not a number, then return a NULL. In the
event a number is first encountered, check whether f is
an IEEE 754 "not a number" value. If false then return
that number. If none of the above cases are satisfied
return a Null value.
*/
func (this *IfNaN) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsNaN(f) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Minimum input arguments required is 2
*/
func (this *IfNaN) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the IfNaN
function is MaxInt16  = 1<<15 - 1. This is defined using the
math package.
*/
func (this *IfNaN) MaxArgs() int { return math.MaxInt16 }

/*
Return NewIfNaN as FunctionConstructor.
*/
func (this *IfNaN) Constructor() FunctionConstructor { return NewIfNaN }

///////////////////////////////////////////////////
//
// IfNaNOrInf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFNANORINF(expr1, expr2, ...).
It returns the first non-MISSING, non-Inf, non-NaN number or NULL. Type
IfNaNOrInf is a struct that implements FunctionBase.
*/
type IfNaNOrInf struct {
	FunctionBase
}

/*
The function NewIfNaNOrInf calls NewFunctionBase to create a function
named IFNANORINF with input arguments as the operands from the input
expression.
*/
func NewIfNaNOrInf(operands ...Expression) Function {
	rv := &IfNaNOrInf{
		*NewFunctionBase("ifnanorinf", operands...),
	}

	rv.conditional = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IfNaNOrInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *IfNaNOrInf) Type() value.Type { return value.NUMBER }

func (this *IfNaNOrInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfNaNOrInf) DependsOn(other Expression) bool {
	return len(this.operands) > 0 &&
		this.operands[0].DependsOn(other)
}

/*
This method returns the first non-MISSING, non-Inf, non-NaN
number or NULL in the input argument values. Range over the args
and check its type. If missing, skip that value, and if
that value is not a number, then return a NULL. In the
event a number is first encountered, check whether f is
an infinity according to the second argument (since it
is 0 IsInf reports whether f is either infinity as per
the math package in the Go Docs). Also check if f is
an IEEE 754 "not a number" value using the IfNaN method
defined in the math package. If false then return that
number. If none of the above cases are satisfied return a
Null value.
*/
func (this *IfNaNOrInf) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsInf(f, 0) && !math.IsNaN(f) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Minimum input arguments required is 2
*/
func (this *IfNaNOrInf) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the
function is MaxInt16  = 1<<15 - 1. This is defined using the
math package.
*/
func (this *IfNaNOrInf) MaxArgs() int { return math.MaxInt16 }

/*
Return NewIfNaNOrInf as FunctionConstructor.
*/
func (this *IfNaNOrInf) Constructor() FunctionConstructor { return NewIfNaNOrInf }

///////////////////////////////////////////////////
//
// NaNIf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function NANIF(expr1, expr2).
It returns a NaN if expr1 = expr2; else expr1. Type NaNIf
is a struct that implements BinaryFunctionBase.
*/
type NaNIf struct {
	BinaryFunctionBase
}

/*
The function NewNaNIf calls NewBinaryFunctionBase to
create a function named NANIF with the two
expressions as input.
*/
func NewNaNIf(first, second Expression) Function {
	rv := &NaNIf{
		*NewBinaryFunctionBase("nanif", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NaNIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type JSON.
*/
func (this *NaNIf) Type() value.Type { return value.JSON }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *NaNIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method checks to see if the values of the two input
expressions are equal, and if true then returns a NaN
using the math package method NaN(). If not it returns
the first input value. Use the Equals method for the
two values to determine equality.
*/
func (this *NaNIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	eq := first.Equals(second)
	switch eq.Type() {
	case value.MISSING, value.NULL:
		return eq, nil
	default:
		if eq.Truth() {
			return _NAN_VALUE, nil
		} else {
			return first, nil
		}
	}
}

/*
The constructor returns a NewNaNIf with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *NaNIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNaNIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// NegInfIf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function NEGINFIF(expr1, expr2).
It returns NegInf if expr1 = expr2; else expr1. Type NegInfIf
is a struct that implements BinaryFunctionBase.
*/
type NegInfIf struct {
	BinaryFunctionBase
}

/*
The function NewNegInfIf calls NewBinaryFunctionBase to
create a function named NEGINFIF with the two
expressions as input.
*/
func NewNegInfIf(first, second Expression) Function {
	rv := &NegInfIf{
		*NewBinaryFunctionBase("neginfif", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NegInfIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type JSON.
*/
func (this *NegInfIf) Type() value.Type { return value.JSON }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *NegInfIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method checks to see if the values of the two input
expressions are equal, and if true then returns a negative
infinity using the math package method Inf(-1). If not it
returns the first input value. Use the Equals method for the
two values to determine equality.
*/
func (this *NegInfIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	eq := first.Equals(second)
	switch eq.Type() {
	case value.MISSING, value.NULL:
		return eq, nil
	default:
		if eq.Truth() {
			return _NEG_INF_VALUE, nil
		} else {
			return first, nil
		}
	}
}

var _NEG_INF_VALUE = value.NewValue(math.Inf(-1))

/*
The constructor returns a NewNegInfIf with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *NegInfIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNegInfIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// PosInfIf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function POSINFIF(expr1, expr2).
It returns PosInf if expr1 = expr2; else expr1. Type PosInfIf
is a struct that implements BinaryFunctionBase.
*/
type PosInfIf struct {
	BinaryFunctionBase
}

/*
The function NewPOSInfIf calls NewBinaryFunctionBase to
create a function named POSINFIF with the two
expressions as input.
*/
func NewPosInfIf(first, second Expression) Function {
	rv := &PosInfIf{
		*NewBinaryFunctionBase("posinfif", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *PosInfIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type JSON.
*/
func (this *PosInfIf) Type() value.Type { return value.JSON }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *PosInfIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method checks to see if the values of the two input
expressions are equal, and if true then returns a positive
infinity using the math package method Inf(1). If not it
returns the first input value. Use the Equals method for the
two values to determine equality.
*/
func (this *PosInfIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	eq := first.Equals(second)
	switch eq.Type() {
	case value.MISSING, value.NULL:
		return eq, nil
	default:
		if eq.Truth() {
			return _POS_INF_VALUE, nil
		} else {
			return first, nil
		}
	}
}

var _POS_INF_VALUE = value.NewValue(math.Inf(1))

/*
The constructor returns a NewPosInfIf with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *PosInfIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosInfIf(operands[0], operands[1])
	}
}
