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
	"math/rand"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Abs
//
///////////////////////////////////////////////////

/*
This represents the number function ABS(expr). It returns
the absolute value of the number.
*/
type Abs struct {
	UnaryFunctionBase
}

func NewAbs(operand Expression) Function {
	rv := &Abs{
		*NewUnaryFunctionBase("abs", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Abs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Abs) Type() value.Type { return value.NUMBER }

func (this *Abs) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns an absolute
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Abs) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Abs(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Abs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAbs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Acos
//
///////////////////////////////////////////////////

/*
This represents the number function ACOS(expr). It returns
the arccosine in radians of the input value.
*/
type Acos struct {
	UnaryFunctionBase
}

func NewAcos(operand Expression) Function {
	rv := &Acos{
		*NewUnaryFunctionBase("acos", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Acos) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Acos) Type() value.Type { return value.NUMBER }

func (this *Acos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its acos
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Acos) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Acos(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Acos) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAcos(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Asin
//
///////////////////////////////////////////////////

/*
This represents the number function ASIN(expr). It returns
the arcsine in radians of the input value.
*/
type Asin struct {
	UnaryFunctionBase
}

func NewAsin(operand Expression) Function {
	rv := &Asin{
		*NewUnaryFunctionBase("asin", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Asin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Asin) Type() value.Type { return value.NUMBER }

func (this *Asin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the Asin
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Asin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Asin(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Asin) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAsin(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Atan
//
///////////////////////////////////////////////////

/*
This represents the number function ATAN(expr). It returns
the arctangent in radians of the input value.
*/
type Atan struct {
	UnaryFunctionBase
}

func NewAtan(operand Expression) Function {
	rv := &Atan{
		*NewUnaryFunctionBase("atan", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Atan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Atan) Type() value.Type { return value.NUMBER }

func (this *Atan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its arctangent
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Atan) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Atan(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Atan) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAtan(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Atan2
//
///////////////////////////////////////////////////

/*
This represents the number function ATAN2(expr1, expr2).
It returns the arctangent of expr2/expr1.
*/
type Atan2 struct {
	BinaryFunctionBase
}

func NewAtan2(first, second Expression) Function {
	rv := &Atan2{
		*NewBinaryFunctionBase("atan2", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Atan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Atan2) Type() value.Type { return value.NUMBER }

func (this *Atan2) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the atan2 value of the input expressions. If either
of the input argument types are missing, or not a number return a missing
and null value respectively.
*/
func (this *Atan2) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Atan2(
		first.Actual().(float64),
		second.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Atan2) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAtan2(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Ceil
//
///////////////////////////////////////////////////

/*
This represents the number function CEIL(expr). It
represents the smallest integer not less than the
number.
*/
type Ceil struct {
	UnaryFunctionBase
}

func NewCeil(operand Expression) Function {
	rv := &Ceil{
		*NewUnaryFunctionBase("ceil", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Ceil) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Ceil) Type() value.Type { return value.NUMBER }

func (this *Ceil) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the int
value not less than the input. If the type of operand is missing then
return it. For values that are not of type Number, return a null
value.
*/
func (this *Ceil) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Ceil(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Ceil) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewCeil(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Cos
//
///////////////////////////////////////////////////

/*
This represents the number function COS(expr). It returns
the cosine of the input value.
*/
type Cos struct {
	UnaryFunctionBase
}

func NewCos(operand Expression) Function {
	rv := &Cos{
		*NewUnaryFunctionBase("cos", operand),
	}

	rv.expr = rv
	return rv
}

/*
visitor pattern.
*/
func (this *Cos) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Cos) Type() value.Type { return value.NUMBER }

func (this *Cos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its cos
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Cos) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Cos(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Cos) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewCos(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Degrees
//
///////////////////////////////////////////////////

/*
This represents the number function DEGREES(expr). It
converts input radians to degrees.
*/
type Degrees struct {
	UnaryFunctionBase
}

func NewDegrees(operand Expression) Function {
	rv := &Degrees{
		*NewUnaryFunctionBase("degrees", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Degrees) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Degrees) Type() value.Type { return value.NUMBER }

func (this *Degrees) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and converts it from
radians to degrees. If the type of operand is missing then return it.
For values that are not of type Number, return a null value.
*/
func (this *Degrees) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * 180.0 / math.Pi), nil
}

/*
Factory method pattern.
*/
func (this *Degrees) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDegrees(operands[0])
	}
}

///////////////////////////////////////////////////
//
// E
//
///////////////////////////////////////////////////

/*
This represents the number function E(). It
returns eulers number which is used as a base
of natural logarithms.
*/
type E struct {
	NullaryFunctionBase
}

var _E = NewE()

func NewE() Function {
	rv := &E{
		*NewNullaryFunctionBase("e"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *E) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *E) Type() value.Type { return value.NUMBER }

/*
Returns _E_VALUE.
*/
func (this *E) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _E_VALUE, nil
}

/*
Returns _E_VALUE.
*/
func (this *E) Value() value.Value {
	return _E_VALUE
}

func (this *E) Static() Expression {
	return this
}

/*
Factory method pattern.
*/
func (this *E) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _E }
}

/*
Variable _E_VALUE uses the math package to define
the Euler number used as a base in logs.
*/
var _E_VALUE = value.NewValue(math.E)

///////////////////////////////////////////////////
//
// Exp
//
///////////////////////////////////////////////////

/*
This represents the number function EXP(expr). It
represents e to the power expr.
*/
type Exp struct {
	UnaryFunctionBase
}

func NewExp(operand Expression) Function {
	rv := &Exp{
		*NewUnaryFunctionBase("exp", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Exp) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Exp) Type() value.Type { return value.NUMBER }

func (this *Exp) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Exp) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Exp(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Exp) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewExp(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Ln
//
///////////////////////////////////////////////////

/*
This represents the number function LN(expr). It
computes log base e.
*/
type Ln struct {
	UnaryFunctionBase
}

func NewLn(operand Expression) Function {
	rv := &Ln{
		*NewUnaryFunctionBase("ln", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Ln) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Ln) Type() value.Type { return value.NUMBER }

func (this *Ln) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Ln) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Log(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Ln) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLn(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Log
//
///////////////////////////////////////////////////

/*
This represents the number function LOG(expr). It
computes log base 10.
*/
type Log struct {
	UnaryFunctionBase
}

func NewLog(operand Expression) Function {
	rv := &Log{
		*NewUnaryFunctionBase("log", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Log) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Log) Type() value.Type { return value.NUMBER }

func (this *Log) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Log) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Log10(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Log) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLog(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Floor
//
///////////////////////////////////////////////////

/*
This represents the number function FLOOR(expr). It
returns the largest integer not greater than the
number.
*/
type Floor struct {
	UnaryFunctionBase
}

func NewFloor(operand Expression) Function {
	rv := &Floor{
		*NewUnaryFunctionBase("floor", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Floor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Floor) Type() value.Type { return value.NUMBER }

func (this *Floor) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Floor) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Floor(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Floor) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewFloor(operands[0])
	}
}

///////////////////////////////////////////////////
//
// NaN
//
///////////////////////////////////////////////////

/*
This represents the number function NaN(). It returns
an IEEE 754 “not-a-number” value.
*/
type NaN struct {
	NullaryFunctionBase
}

var _NAN = NewNaN()

func NewNaN() Function {
	rv := &NaN{
		*NewNullaryFunctionBase("nan"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NaN) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NaN) Type() value.Type { return value.NUMBER }

/*
Return _NAN_VALUE.
*/
func (this *NaN) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _NAN_VALUE, nil
}

/*
Return _NAN_VALUE.
*/
func (this *NaN) Value() value.Value {
	return _NAN_VALUE
}

func (this *NaN) Static() Expression {
	return this
}

/*
Factory method pattern.
*/
func (this *NaN) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _NAN }
}

/*
Var _NAN_VALUE uses the math package NaN method to
define an IEEE 754 “not-a-number” value.
*/
var _NAN_VALUE = value.NewValue(math.NaN())

///////////////////////////////////////////////////
//
// NegInf
//
///////////////////////////////////////////////////

/*
This represents the number function NEGINF(). It
returns the negative infinity number value.
*/
type NegInf struct {
	NullaryFunctionBase
}

var _NEG_INF = NewNegInf()

func NewNegInf() Function {
	rv := &NegInf{
		*NewNullaryFunctionBase("neginf"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NegInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NegInf) Type() value.Type { return value.NUMBER }

/*
Returns _NEGINF_VALUE.
*/
func (this *NegInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _NEGINF_VALUE, nil
}

/*
Returns _NEGINF_VALUE.
*/
func (this *NegInf) Value() value.Value {
	return _NEGINF_VALUE
}

func (this *NegInf) Static() Expression {
	return this
}

/*
Factory method pattern.
*/
func (this *NegInf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _NEG_INF }
}

/*
Variable _NEGINF_VALUE the negative infinity value
defined using the method Inf(-1) in the  math package.
*/
var _NEGINF_VALUE = value.NewValue(math.Inf(-1))

///////////////////////////////////////////////////
//
// PI
//
///////////////////////////////////////////////////

/*
This represents the number function PI(). It returns
PI.
*/
type PI struct {
	NullaryFunctionBase
}

var _PI = NewPI()

func NewPI() Function {
	rv := &PI{
		*NewNullaryFunctionBase("pi"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *PI) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *PI) Type() value.Type { return value.NUMBER }

/*
Return _PI_VALUE.
*/
func (this *PI) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _PI_VALUE, nil
}

/*
Return _PI_VALUE.
*/
func (this *PI) Value() value.Value {
	return _PI_VALUE
}

func (this *PI) Static() Expression {
	return this
}

/*
Factory method pattern.
*/
func (this *PI) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _PI }
}

/*
Variable _PI_VALUE uses the math package to define PI.
*/
var _PI_VALUE = value.NewValue(math.Pi)

///////////////////////////////////////////////////
//
// PosInf
//
///////////////////////////////////////////////////

/*
This represents the number function POSINF(). It
returns the positive infinity number value.
*/
type PosInf struct {
	NullaryFunctionBase
}

var _POS_INF = NewPosInf()

func NewPosInf() Function {
	rv := &PosInf{
		*NewNullaryFunctionBase("posinf"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *PosInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *PosInf) Type() value.Type { return value.NUMBER }

/*
Returns _POSINF_VALUE.
*/
func (this *PosInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _POSINF_VALUE, nil
}

/*
Returns _POSINF_VALUE.
*/
func (this *PosInf) Value() value.Value {
	return _POSINF_VALUE
}

func (this *PosInf) Static() Expression {
	return this
}

/*
Factory method pattern.
*/
func (this *PosInf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _POS_INF }
}

/*
Variable _POSINF_VALUE the negative infinity value
defined using the method Inf(1) in the math package.
*/
var _POSINF_VALUE = value.NewValue(math.Inf(1))

///////////////////////////////////////////////////
//
// Power
//
///////////////////////////////////////////////////

/*
This represents the number function POWER(expr1, expr2).
It returns expr1 to the power of expr2.
*/
type Power struct {
	BinaryFunctionBase
}

func NewPower(first, second Expression) Function {
	rv := &Power{
		*NewBinaryFunctionBase("power", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Power) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Power) Type() value.Type { return value.NUMBER }

func (this *Power) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Power) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Pow(
		first.Actual().(float64),
		second.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Power) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPower(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Radians
//
///////////////////////////////////////////////////

/*
This represents the number function RADIANS(expr).
It converts degrees to radians.
*/
type Radians struct {
	UnaryFunctionBase
}

func NewRadians(operand Expression) Function {
	rv := &Radians{
		*NewUnaryFunctionBase("radians", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Radians) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Radians) Type() value.Type { return value.NUMBER }

func (this *Radians) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Radians) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * math.Pi / 180.0), nil
}

/*
Factory method pattern.
*/
func (this *Radians) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRadians(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Random
//
///////////////////////////////////////////////////

/*
This represents the function RANDOM(), with an optional seed.
*/
type Random struct {
	FunctionBase
	gen *rand.Rand
}

func NewRandom(operands ...Expression) Function {
	rv := &Random{
		*NewFunctionBase("random", operands...),
		nil,
	}

	rv.setVolatile()
	rv.expr = rv

	if len(operands) < 1 {
		return rv
	}

	switch op := operands[0].(type) {
	case *Constant:
		switch val := op.Value().Actual().(type) {
		case float64:
			rv.gen = rand.New(rand.NewSource(int64(val)))
		}
	}

	return rv
}

/*
Visitor pattern.
*/
func (this *Random) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Random) Type() value.Type { return value.NUMBER }

func (this *Random) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Random) Value() value.Value {
	return nil
}

func (this *Random) Static() Expression {
	return nil
}

func (this *Random) Apply(context Context, args ...value.Value) (value.Value, error) {
	if this.gen != nil {
		return value.NewValue(this.gen.Float64()), nil
	}

	if len(args) == 0 {
		return value.NewValue(rand.Float64()), nil
	}

	arg := args[0]

	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)
	if v != math.Trunc(v) {
		return value.NULL_VALUE, nil
	}

	gen := rand.New(rand.NewSource(int64(v)))
	return value.NewValue(gen.Float64()), nil
}

/*
Minimum input arguments required is 0.
*/
func (this *Random) MinArgs() int { return 0 }

/*
Maximum input arguments allowed is 1.
*/
func (this *Random) MaxArgs() int { return 1 }

/*
Factory method pattern.
*/
func (this *Random) Constructor() FunctionConstructor {
	return NewRandom
}

///////////////////////////////////////////////////
//
// Round
//
///////////////////////////////////////////////////

/*
This represents the number function ROUND(expr [, digits ]).
It rounds the value to the given number of integer digits
to the right of the decimal point (left if digits is
negative). digits is 0 if not given.
*/
type Round struct {
	FunctionBase
}

func NewRound(operands ...Expression) Function {
	rv := &Round{
		*NewFunctionBase("round", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Round) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Round) Type() value.Type { return value.NUMBER }

func (this *Round) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Round) Apply(context Context, args ...value.Value) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)

	if len(this.operands) == 1 {
		return value.NewValue(roundFloat(v, 0)), nil
	}

	p := 0
	prec := args[1]
	if prec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if prec.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	} else {
		pf := prec.Actual().(float64)
		if pf != math.Trunc(pf) {
			return value.NULL_VALUE, nil
		}
		p = int(pf)
	}

	return value.NewValue(roundFloat(v, p)), nil
}

/*
Minimum input arguments required is 1.
*/
func (this *Round) MinArgs() int { return 1 }

/*
Maximum input arguments allowed is 2.
*/
func (this *Round) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Round) Constructor() FunctionConstructor {
	return NewRound
}

///////////////////////////////////////////////////
//
// Sign
//
///////////////////////////////////////////////////

/*
This represents the number function SIGN(expr). It returns
-1, 0, or 1 for negative, zero, or positive numbers
respectively.
*/
type Sign struct {
	UnaryFunctionBase
}

func NewSign(operand Expression) Function {
	rv := &Sign{
		*NewUnaryFunctionBase("sign", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Sign) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sign) Type() value.Type { return value.NUMBER }

func (this *Sign) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the
sign of the value. If the type of operand is missing then return
missing. For values that are not of type Number, return a null
value. For numbers, compare to 0.0; if smaller return -1, equal
return 0 and greater return 1.
*/
func (this *Sign) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	f := arg.Actual().(float64)
	s := 0.0
	if f < 0.0 {
		s = -1.0
	} else if f > 0.0 {
		s = 1.0
	}

	return value.NewValue(s), nil
}

/*
Factory method pattern.
*/
func (this *Sign) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSign(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Sin
//
///////////////////////////////////////////////////

/*
This represents the number function SIN(expr). It
returns the sine of the input number value.
*/
type Sin struct {
	UnaryFunctionBase
}

func NewSin(operand Expression) Function {
	rv := &Sin{
		*NewUnaryFunctionBase("sin", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Sin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sin) Type() value.Type { return value.NUMBER }

func (this *Sin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its sin
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Sin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sin(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Sin) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSin(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Sqrt
//
///////////////////////////////////////////////////

/*
This represents the number function SQRT(expr). It returns
the square root of the input.
*/
type Sqrt struct {
	UnaryFunctionBase
}

func NewSqrt(operand Expression) Function {
	rv := &Sqrt{
		*NewUnaryFunctionBase("sqrt", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Sqrt) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sqrt) Type() value.Type { return value.NUMBER }

func (this *Sqrt) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its sqrt
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Sqrt) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sqrt(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Sqrt) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSqrt(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Tan
//
///////////////////////////////////////////////////

/*
This represents the number function TAN(expr). It returns
the tangent of the input.
*/
type Tan struct {
	UnaryFunctionBase
}

func NewTan(operand Expression) Function {
	rv := &Tan{
		*NewUnaryFunctionBase("tan", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Tan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Tan) Type() value.Type { return value.NUMBER }

func (this *Tan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its Tan
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value.
*/
func (this *Tan) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Tan(arg.Actual().(float64))), nil
}

/*
Factory method pattern.
*/
func (this *Tan) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTan(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Trunc
//
///////////////////////////////////////////////////

/*
This represents the number function TRUNC(expr [, digits ]).
It truncates the number to the given number of integer
digits to the right of the decimal point (left if digits is
negative). digits is 0 if not given.
*/
type Trunc struct {
	FunctionBase
}

func NewTrunc(operands ...Expression) Function {
	rv := &Trunc{
		*NewFunctionBase("trunc", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Trunc) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Trunc) Type() value.Type { return value.NUMBER }

func (this *Trunc) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method evaluates the trunc function to truncate the given
number of integer digits to the right or left of the decimal.
If the input args (precision if given) is missing or not a
number, return a missing value or a null value respectively.
*/
func (this *Trunc) Apply(context Context, args ...value.Value) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)

	if len(this.operands) == 1 {
		return value.NewValue(truncateFloat(v, 0)), nil
	}

	p := 0
	prec := args[1]
	if prec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if prec.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	} else {
		pf := prec.Actual().(float64)
		if pf != math.Trunc(pf) {
			return value.NULL_VALUE, nil
		}
		p = int(pf)
	}

	return value.NewValue(truncateFloat(v, p)), nil
}

/*
Minimum input arguments required is 1.
*/
func (this *Trunc) MinArgs() int { return 1 }

/*
Maximum input arguments allowed is 2.
*/
func (this *Trunc) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Trunc) Constructor() FunctionConstructor {
	return NewTrunc
}

/*
This method is used to truncate input number to either
side of the decimal point using the integer precision.
A negative precision allows truncate to the left of
the decimal point.
*/
func truncateFloat(x float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	rounder := math.Trunc(intermed)
	return rounder / pow
}

/*
This method is used to round the input number to the
given number of precision digits to the right or left
of the decimal point depending on whether it is
positive or negative. For the fraction 0.5
round towards the even value.
*/
func roundFloat(x float64, prec int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}

	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}

	pow := math.Pow(10, float64(prec))
	intermed := (x * pow) + 0.5
	rounder := math.Floor(intermed)

	// For frac 0.5, round towards even
	if rounder == intermed && math.Mod(rounder, 2) != 0 {
		rounder--
	}

	return sign * rounder / pow
}
