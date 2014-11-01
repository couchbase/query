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

	"github.com/couchbaselabs/query/value"
)

type Function interface {
	Expression
	Name() string
	Distinct() bool
	Operands() Expressions
	MinArgs() int
	MaxArgs() int
	Constructor() FunctionConstructor
}

type FunctionConstructor func(operands ...Expression) Function

type FunctionBase struct {
	ExpressionBase
	name     string
	operands Expressions
}

func NewFunctionBase(name string, operands ...Expression) *FunctionBase {
	return &FunctionBase{
		name:     name,
		operands: operands,
	}
}

func (this *FunctionBase) Eval(applied Applied, item value.Value, context Context) (
	result value.Value, err error) {
	args := make(value.Values, len(this.operands))

	for i, op := range this.operands {
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return
		}
	}

	return applied.Apply(context, args...)
}

func (this *FunctionBase) Indexable() bool {
	for _, operand := range this.operands {
		if !operand.Indexable() {
			return false
		}
	}

	return true
}

func (this *FunctionBase) EquivalentTo(other Expression) bool {
	that, ok := other.(Function)
	if !ok {
		return false
	}

	if this.name != that.Name() ||
		len(this.operands) != len(that.Operands()) {
		return false
	}

	for i, op := range that.Operands() {
		if !this.operands[i].EquivalentTo(op) {
			return false
		}
	}

	return true
}

func (this *FunctionBase) SubsetOf(other Expression) bool {
	return this.EquivalentTo(other)
}

func (this *FunctionBase) Children() Expressions {
	return this.operands
}

func (this *FunctionBase) MapChildren(mapper Mapper) error {
	for i, op := range this.operands {
		expr, err := mapper.Map(op)
		if err != nil {
			return err
		}

		this.operands[i] = expr
	}

	return nil
}

func (this *FunctionBase) Name() string { return this.name }

func (this *FunctionBase) Distinct() bool { return false }

func (this *FunctionBase) Operands() Expressions { return this.operands }

type NullaryFunctionBase struct {
	FunctionBase
}

func NewNullaryFunctionBase(name string) *NullaryFunctionBase {
	return &NullaryFunctionBase{
		FunctionBase{
			name: name,
		},
	}
}

func (this *NullaryFunctionBase) MinArgs() int { return 0 }

func (this *NullaryFunctionBase) MaxArgs() int { return 0 }

type UnaryFunctionBase struct {
	FunctionBase
}

func NewUnaryFunctionBase(name string, operand Expression) *UnaryFunctionBase {
	return &UnaryFunctionBase{
		FunctionBase{
			name:     name,
			operands: Expressions{operand},
		},
	}
}

func (this *UnaryFunctionBase) UnaryEval(applied UnaryApplied, item value.Value, context Context) (
	value.Value, error) {
	op := this.operands[0]
	arg, err := op.Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	return applied.Apply(context, arg)
}

func (this *UnaryFunctionBase) MinArgs() int { return 1 }

func (this *UnaryFunctionBase) MaxArgs() int { return 1 }

func (this *UnaryFunctionBase) Operand() Expression {
	return this.operands[0]
}

type BinaryFunctionBase struct {
	FunctionBase
}

func NewBinaryFunctionBase(name string, first, second Expression) *BinaryFunctionBase {
	return &BinaryFunctionBase{
		FunctionBase{
			name:     name,
			operands: Expressions{first, second},
		},
	}
}

func (this *BinaryFunctionBase) BinaryEval(applied BinaryApplied, item value.Value, context Context) (
	result value.Value, err error) {
	args := make(value.Values, len(this.operands))

	for i, op := range this.operands {
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return
		}
	}

	return applied.Apply(context, args[0], args[1])
}

func (this *BinaryFunctionBase) MinArgs() int { return 2 }

func (this *BinaryFunctionBase) MaxArgs() int { return 2 }

func (this *BinaryFunctionBase) First() Expression {
	return this.operands[0]
}

func (this *BinaryFunctionBase) Second() Expression {
	return this.operands[1]
}

type TernaryFunctionBase struct {
	FunctionBase
}

func NewTernaryFunctionBase(name string, first, second, third Expression) *TernaryFunctionBase {
	return &TernaryFunctionBase{
		FunctionBase{
			name:     name,
			operands: Expressions{first, second, third},
		},
	}
}

func (this *TernaryFunctionBase) TernaryEval(applied TernaryApplied, item value.Value, context Context) (
	result value.Value, err error) {
	args := make(value.Values, len(this.operands))

	for i, op := range this.operands {
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return
		}
	}

	return applied.Apply(context, args[0], args[1], args[2])
}

func (this *TernaryFunctionBase) MinArgs() int { return 3 }

func (this *TernaryFunctionBase) MaxArgs() int { return 3 }

func (this *TernaryFunctionBase) First() Expression {
	return this.operands[0]
}

func (this *TernaryFunctionBase) Second() Expression {
	return this.operands[1]
}

func (this *TernaryFunctionBase) Third() Expression {
	return this.operands[2]
}

type CommutativeFunctionBase struct {
	FunctionBase
}

func NewCommutativeFunctionBase(name string, operands ...Expression) *CommutativeFunctionBase {
	return &CommutativeFunctionBase{
		FunctionBase{
			name:     name,
			operands: operands,
		},
	}
}

func (this *CommutativeFunctionBase) EquivalentTo(other Expression) bool {
	that, ok := other.(Function)
	if !ok {
		return false
	}

	if this.name != that.Name() ||
		len(this.operands) != len(that.Operands()) {
		return false
	}

	found := make([]bool, len(this.operands))

	for _, first := range this.operands {
		for j, second := range that.Operands() {
			if !found[j] && first.EquivalentTo(second) {
				found[j] = true
				break
			}
		}
	}

	for _, f := range found {
		if !f {
			return false
		}
	}

	return true
}

func (this *CommutativeFunctionBase) SubsetOf(other Expression) bool {
	return this.EquivalentTo(other)
}

func (this *CommutativeFunctionBase) MinArgs() int { return 2 }

func (this *CommutativeFunctionBase) MaxArgs() int { return math.MaxInt16 }

type Applied interface {
	Apply(context Context, args ...value.Value) (value.Value, error)
}

type UnaryApplied interface {
	Apply(context Context, arg value.Value) (value.Value, error)
}

type BinaryApplied interface {
	Apply(context Context, first, second value.Value) (value.Value, error)
}

type TernaryApplied interface {
	Apply(context Context, first, second, third value.Value) (value.Value, error)
}
