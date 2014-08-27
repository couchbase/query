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

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// Abs
//
///////////////////////////////////////////////////

type Abs struct {
	UnaryFunctionBase
}

func NewAbs(operand Expression) Function {
	return &Abs{
		*NewUnaryFunctionBase("abs", operand),
	}
}

func (this *Abs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Abs) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Abs) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Abs(arg.Actual().(float64))), nil
}

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

type Acos struct {
	UnaryFunctionBase
}

func NewAcos(operand Expression) Function {
	return &Acos{
		*NewUnaryFunctionBase("acos", operand),
	}
}

func (this *Acos) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Acos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Acos) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Acos(arg.Actual().(float64))), nil
}

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

type Asin struct {
	UnaryFunctionBase
}

func NewAsin(operand Expression) Function {
	return &Asin{
		*NewUnaryFunctionBase("asin", operand),
	}
}

func (this *Asin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Asin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Asin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Asin(arg.Actual().(float64))), nil
}

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

type Atan struct {
	UnaryFunctionBase
}

func NewAtan(operand Expression) Function {
	return &Atan{
		*NewUnaryFunctionBase("atan", operand),
	}
}

func (this *Atan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Atan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Atan) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Atan(arg.Actual().(float64))), nil
}

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

type Atan2 struct {
	BinaryFunctionBase
}

func NewAtan2(first, second Expression) Function {
	return &Atan2{
		*NewBinaryFunctionBase("atan2", first, second),
	}
}

func (this *Atan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Atan2) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

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

type Ceil struct {
	UnaryFunctionBase
}

func NewCeil(operand Expression) Function {
	return &Ceil{
		*NewUnaryFunctionBase("ceil", operand),
	}
}

func (this *Ceil) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Ceil) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Ceil) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Ceil(arg.Actual().(float64))), nil
}

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

type Cos struct {
	UnaryFunctionBase
}

func NewCos(operand Expression) Function {
	return &Cos{
		*NewUnaryFunctionBase("cos", operand),
	}
}

func (this *Cos) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Cos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Cos) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Cos(arg.Actual().(float64))), nil
}

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

type Degrees struct {
	UnaryFunctionBase
}

func NewDegrees(operand Expression) Function {
	return &Degrees{
		*NewUnaryFunctionBase("degrees", operand),
	}
}

func (this *Degrees) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Degrees) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Degrees) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * 180.0 / math.Pi), nil
}

func (this *Degrees) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDegrees(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Exp
//
///////////////////////////////////////////////////

type Exp struct {
	UnaryFunctionBase
}

func NewExp(operand Expression) Function {
	return &Exp{
		*NewUnaryFunctionBase("exp", operand),
	}
}

func (this *Exp) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

type Ln struct {
	UnaryFunctionBase
}

func NewLn(operand Expression) Function {
	return &Ln{
		*NewUnaryFunctionBase("ln", operand),
	}
}

func (this *Ln) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

type Log struct {
	UnaryFunctionBase
}

func NewLog(operand Expression) Function {
	return &Log{
		*NewUnaryFunctionBase("log", operand),
	}
}

func (this *Log) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

type Floor struct {
	UnaryFunctionBase
}

func NewFloor(operand Expression) Function {
	return &Floor{
		*NewUnaryFunctionBase("floor", operand),
	}
}

func (this *Floor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

func (this *Floor) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewFloor(operands[0])
	}
}

///////////////////////////////////////////////////
//
// PI
//
///////////////////////////////////////////////////

type PI struct {
	NullaryFunctionBase
}

func NewPI() Function {
	return &PI{
		*NewNullaryFunctionBase("pi"),
	}
}

func (this *PI) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *PI) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _PI_VALUE, nil
}

func (this *PI) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return this }
}

var _PI_VALUE = value.NewValue(math.Pi)

///////////////////////////////////////////////////
//
// Power
//
///////////////////////////////////////////////////

type Power struct {
	BinaryFunctionBase
}

func NewPower(first, second Expression) Function {
	return &Power{
		*NewBinaryFunctionBase("power", first, second),
	}
}

func (this *Power) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

type Radians struct {
	UnaryFunctionBase
}

func NewRadians(operand Expression) Function {
	return &Radians{
		*NewUnaryFunctionBase("radians", operand),
	}
}

func (this *Radians) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

type Random struct {
	FunctionBase
	gen *rand.Rand
}

func NewRandom(operands ...Expression) Function {
	return &Random{
		*NewFunctionBase("random", operands...),
		nil,
	}
}

func (this *Random) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Random) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
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

	var gen rand.Rand
	gen.Seed(int64(v))
	return value.NewValue(gen.Float64()), nil
}

func (this *Random) MinArgs() int { return 0 }

func (this *Random) MaxArgs() int { return 1 }

func (this *Random) Constructor() FunctionConstructor { return NewRandom }

///////////////////////////////////////////////////
//
// Round
//
///////////////////////////////////////////////////

type Round struct {
	FunctionBase
}

func NewRound(operands ...Expression) Function {
	return &Round{
		*NewFunctionBase("round", operands...),
	}
}

func (this *Round) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

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

func (this *Round) MinArgs() int { return 1 }

func (this *Round) MaxArgs() int { return 2 }

func (this *Round) Constructor() FunctionConstructor { return NewRound }

///////////////////////////////////////////////////
//
// Sign
//
///////////////////////////////////////////////////

type Sign struct {
	UnaryFunctionBase
}

func NewSign(operand Expression) Function {
	return &Sign{
		*NewUnaryFunctionBase("sign", operand),
	}
}

func (this *Sign) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sign) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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

type Sin struct {
	UnaryFunctionBase
}

func NewSin(operand Expression) Function {
	return &Sin{
		*NewUnaryFunctionBase("sin", operand),
	}
}

func (this *Sin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Sin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sin(arg.Actual().(float64))), nil
}

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

type Sqrt struct {
	UnaryFunctionBase
}

func NewSqrt(operand Expression) Function {
	return &Sqrt{
		*NewUnaryFunctionBase("sqrt", operand),
	}
}

func (this *Sqrt) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sqrt) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Sqrt) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sqrt(arg.Actual().(float64))), nil
}

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

type Tan struct {
	UnaryFunctionBase
}

func NewTan(operand Expression) Function {
	return &Tan{
		*NewUnaryFunctionBase("tan", operand),
	}
}

func (this *Tan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Tan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Tan) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Tan(arg.Actual().(float64))), nil
}

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

type Trunc struct {
	FunctionBase
}

func NewTrunc(operands ...Expression) Function {
	return &Trunc{
		*NewFunctionBase("trunc", operands...),
	}
}

func (this *Trunc) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Trunc) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

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

func (this *Trunc) MinArgs() int { return 1 }

func (this *Trunc) MaxArgs() int { return 2 }

func (this *Trunc) Constructor() FunctionConstructor { return NewTrunc }

func truncateFloat(x float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	rounder := math.Trunc(intermed)
	return rounder / pow
}

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
