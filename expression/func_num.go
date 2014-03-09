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
	"math/rand"

	"github.com/couchbaselabs/query/value"
)

type Abs struct {
	unaryBase
}

func NewAbs(arg Expression) Function {
	return &Abs{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Abs) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Abs(arg.Actual().(float64))), nil
}

func (this *Abs) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewAbs(args[0])
	}
}

type Acos struct {
	unaryBase
}

func NewAcos(arg Expression) Function {
	return &Acos{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Acos) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Acos(arg.Actual().(float64))), nil
}

func (this *Acos) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewAcos(args[0])
	}
}

type Asin struct {
	unaryBase
}

func NewAsin(arg Expression) Function {
	return &Asin{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Asin) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Asin(arg.Actual().(float64))), nil
}

func (this *Asin) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewAsin(args[0])
	}
}

type Atan struct {
	unaryBase
}

func NewAtan(arg Expression) Function {
	return &Atan{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Atan) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Atan(arg.Actual().(float64))), nil
}

func (this *Atan) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewAtan(args[0])
	}
}

type Atan2 struct {
	binaryBase
}

func NewAtan2(first, second Expression) Function {
	return &Atan2{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Atan2) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewAtan2(args[0], args[1])
	}
}

type Ceil struct {
	unaryBase
}

func NewCeil(arg Expression) Function {
	return &Ceil{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Ceil) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Ceil(arg.Actual().(float64))), nil
}

func (this *Ceil) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewCeil(args[0])
	}
}

type Cos struct {
	unaryBase
}

func NewCos(arg Expression) Function {
	return &Cos{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Cos) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Cos(arg.Actual().(float64))), nil
}

func (this *Cos) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewCos(args[0])
	}
}

type Degrees struct {
	unaryBase
}

func NewDegrees(arg Expression) Function {
	return &Degrees{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Degrees) Fold() (Expression, error) {
	return NewMultiply(this.operand, _RAD_TO_DEG).Fold()
}

func (this *Degrees) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * 180.0 / math.Pi), nil
}

func (this *Degrees) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDegrees(args[0])
	}
}

type Exp struct {
	unaryBase
}

func NewExp(arg Expression) Function {
	return &Exp{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Exp) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Exp(arg.Actual().(float64))), nil
}

func (this *Exp) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewExp(args[0])
	}
}

type Ln struct {
	unaryBase
}

func NewLn(arg Expression) Function {
	return &Ln{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Ln) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Log(arg.Actual().(float64))), nil
}

func (this *Ln) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLn(args[0])
	}
}

type Log struct {
	unaryBase
}

func NewLog(arg Expression) Function {
	return &Log{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Log) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Log10(arg.Actual().(float64))), nil
}

func (this *Log) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLog(args[0])
	}
}

type Floor struct {
	unaryBase
}

func NewFloor(arg Expression) Function {
	return &Floor{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Floor) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Floor(arg.Actual().(float64))), nil
}

func (this *Floor) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewFloor(args[0])
	}
}

type PI struct {
	ExpressionBase
}

func NewPI() Function {
	return &PI{}
}

func (this *PI) Fold() (Expression, error) {
	return _PI, nil
}

func (this *PI) evaluate() (value.Value, error) {
	return value.NewValue(math.Pi), nil
}

func (this *PI) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewPI()
	}
}

var _PI = NewConstant(value.NewValue(math.Pi))

type Power struct {
	binaryBase
}

func NewPower(first, second Expression) Function {
	return &Power{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Power) evaluate(first, second value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewPower(args[0], args[1])
	}
}

type Radians struct {
	unaryBase
}

func NewRadians(arg Expression) Function {
	return &Radians{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Radians) Fold() (Expression, error) {
	return NewMultiply(this.operand, _DEG_TO_RAD).Fold()
}

func (this *Radians) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * math.Pi / 180.0), nil
}

func (this *Radians) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewRadians(args[0])
	}
}

type Random struct {
	nAryBase
	gen *rand.Rand
}

func NewRandom(arguments Expressions) Function {
	return &Random{
		nAryBase: nAryBase{
			operands: arguments,
		},
	}
}

func (this *Random) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	if len(this.operands) == 0 {
		return this, nil
	}

	switch o := this.operands[0].(type) {
	case *Constant:
		v := o.Value().Actual()
		switch v := v.(type) {
		case float64:
			if v != math.Trunc(v) {
				return nil, fmt.Errorf("Non-integer RANDOM seed %v.", v)
			}
			this.gen = &rand.Rand{}
			this.gen.Seed(int64(v))
		default:
			return nil, fmt.Errorf("Invalid RANDOM seed %v of type %T.", v, v)
		}
	}

	return this, nil
}

func (this *Random) evaluate(args value.Values) (value.Value, error) {
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

type Round struct {
	nAryBase
	precision int
}

func NewRound(arguments Expressions) Function {
	return &Round{
		nAryBase: nAryBase{
			operands: arguments,
		},
	}
}

func (this *Round) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	if len(this.operands) < 2 {
		return this, nil
	}

	switch o := this.operands[1].(type) {
	case *Constant:
		v := o.Value().Actual()
		switch v := v.(type) {
		case float64:
			if v != math.Trunc(v) {
				return nil, fmt.Errorf("Non-integer ROUND precision %v.", v)
			}
			this.precision = int(v)
			this.operands = nil
		default:
			return nil, fmt.Errorf("Invalid ROUND precision %v of type %T.", v, v)
		}
	}

	return this, nil
}

func (this *Round) evaluate(args value.Values) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)

	if len(this.operands) == 0 {
		return value.NewValue(roundFloat(v, this.precision)), nil
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

type Sign struct {
	unaryBase
}

func NewSign(arg Expression) Function {
	return &Sign{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Sign) evaluate(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewSign(args[0])
	}
}

type Sin struct {
	unaryBase
}

func NewSin(arg Expression) Function {
	return &Sin{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Sin) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sin(arg.Actual().(float64))), nil
}

func (this *Sin) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewSin(args[0])
	}
}

type Sqrt struct {
	unaryBase
}

func NewSqrt(arg Expression) Function {
	return &Sqrt{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Sqrt) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sqrt(arg.Actual().(float64))), nil
}

func (this *Sqrt) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewSqrt(args[0])
	}
}

type Tan struct {
	unaryBase
}

func NewTan(arg Expression) Function {
	return &Tan{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Tan) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Tan(arg.Actual().(float64))), nil
}

func (this *Tan) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewTan(args[0])
	}
}

type Trunc struct {
	nAryBase
	precision int
}

func NewTrunc(arguments Expressions) Function {
	return &Trunc{
		nAryBase: nAryBase{
			operands: arguments,
		},
	}
}

func (this *Trunc) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	if len(this.operands) < 2 {
		return this, nil
	}

	switch o := this.operands[1].(type) {
	case *Constant:
		v := o.Value().Actual()
		switch v := v.(type) {
		case float64:
			if v != math.Trunc(v) {
				return nil, fmt.Errorf("Non-integer TRUNC precision %v.", v)
			}
			this.precision = int(v)
			this.operands = nil
		default:
			return nil, fmt.Errorf("Invalid TRUNC precision %v of type %T.", v, v)
		}
	}

	return this, nil
}

func (this *Trunc) evaluate(args value.Values) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)

	if len(this.operands) == 0 {
		return value.NewValue(truncateFloat(v, this.precision)), nil
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
	rounder := math.Floor(intermed)
	return rounder / pow
}

func roundFloat(x float64, prec int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}

	sign := 1.0
	if x < 0 {
		sign = -1
		x = -x
	}

	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	rounder := math.Floor(intermed + 0.5)
	if rounder == math.Trunc(rounder) && math.Mod(rounder, 2) != 0 {
		// For frac 0.5, round towards even
		rounder--
	}
	return sign * rounder / pow
}

var _DEG_TO_RAD = NewConstant(value.NewValue(math.Pi / 180.0))
var _RAD_TO_DEG = NewConstant(value.NewValue(180.0 / math.Pi))
