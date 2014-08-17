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

func (this *Abs) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Abs) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Abs) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Abs) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Abs) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Abs) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Abs) eval(arg value.Value) (value.Value, error) {
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

func (this *Acos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Acos) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Acos) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Acos) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Acos) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Acos) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Acos) eval(arg value.Value) (value.Value, error) {
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

func (this *Asin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Asin) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Asin) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Asin) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Asin) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Asin) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Asin) eval(arg value.Value) (value.Value, error) {
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

func (this *Atan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Atan) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Atan) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Atan) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Atan) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Atan) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Atan) eval(arg value.Value) (value.Value, error) {
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

func (this *Atan2) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Atan2) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Atan2) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Atan2) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Atan2) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Atan2) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Atan2) eval(first, second value.Value) (value.Value, error) {
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

func (this *Ceil) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Ceil) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Ceil) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Ceil) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Ceil) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Ceil) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Ceil) eval(arg value.Value) (value.Value, error) {
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

func (this *Cos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Cos) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Cos) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Cos) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Cos) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Cos) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Cos) eval(arg value.Value) (value.Value, error) {
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

func (this *Degrees) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Degrees) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Degrees) Fold() (Expression, error) {
	return NewMultiply(this.operand, _RAD_TO_DEG).Fold()
}

func (this *Degrees) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Degrees) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Degrees) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Degrees) eval(arg value.Value) (value.Value, error) {
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

func (this *Exp) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Exp) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Exp) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Exp) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Exp) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Exp) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Exp) eval(arg value.Value) (value.Value, error) {
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

func (this *Ln) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Ln) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Ln) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Ln) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Ln) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Ln) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Ln) eval(arg value.Value) (value.Value, error) {
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

func (this *Log) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Log) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Log) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Log) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Log) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Log) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Log) eval(arg value.Value) (value.Value, error) {
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

func (this *Floor) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Floor) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Floor) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Floor) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Floor) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Floor) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Floor) eval(arg value.Value) (value.Value, error) {
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

func (this *PI) Evaluate(item value.Value, context Context) (value.Value, error) {
	return value.NewValue(math.Pi), nil
}

func (this *PI) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *PI) Fold() (Expression, error) {
	return _PI, nil
}

func (this *PI) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this, nil
}

func (this *PI) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *PI) VisitChildren(visitor Visitor) (Expression, error) {
	return this, nil
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

func (this *Power) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Power) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Power) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Power) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Power) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Power) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Power) eval(first, second value.Value) (value.Value, error) {
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

func (this *Radians) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Radians) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Radians) Fold() (Expression, error) {
	return NewMultiply(this.operand, _DEG_TO_RAD).Fold()
}

func (this *Radians) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Radians) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Radians) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Radians) eval(arg value.Value) (value.Value, error) {
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

func (this *Random) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Random) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Random) Fold() (Expression, error) {
	t, e := this.VisitChildren(&Folder{})
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

func (this *Random) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Random) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Random) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Random) eval(args value.Values) (value.Value, error) {
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
}

func NewRound(arguments Expressions) Function {
	return &Round{
		nAryBase: nAryBase{
			operands: arguments,
		},
	}
}

func (this *Round) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Round) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Round) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Round) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Round) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Round) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Round) eval(args value.Values) (value.Value, error) {
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

func (this *Sign) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Sign) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Sign) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Sign) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Sign) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Sign) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Sign) eval(arg value.Value) (value.Value, error) {
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

func (this *Sin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Sin) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Sin) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Sin) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Sin) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Sin) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Sin) eval(arg value.Value) (value.Value, error) {
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

func (this *Sqrt) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Sqrt) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Sqrt) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Sqrt) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Sqrt) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Sqrt) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Sqrt) eval(arg value.Value) (value.Value, error) {
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

func (this *Tan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Tan) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Tan) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Tan) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Tan) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Tan) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Tan) eval(arg value.Value) (value.Value, error) {
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
}

func NewTrunc(arguments Expressions) Function {
	return &Trunc{
		nAryBase: nAryBase{
			operands: arguments,
		},
	}
}

func (this *Trunc) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Trunc) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Trunc) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Trunc) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Trunc) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Trunc) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Trunc) eval(args value.Values) (value.Value, error) {
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

var _DEG_TO_RAD = NewConstant(value.NewValue(math.Pi / 180.0))
var _RAD_TO_DEG = NewConstant(value.NewValue(180.0 / math.Pi))
