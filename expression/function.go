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

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const RANGE_LIMIT = math.MaxInt16 // Maximum range/repeat value

/*
Type Function is an interface that inherits from Expression.
It contains additional methods that help define a function
and its constraints. Most Expressions are Functions, except
for Constants, Identifiers, and a few syntactic elements.
*/
type Function interface {
	/*
	   Inherits from Expression.
	*/
	Expression

	/*
	   Unique name of the Function.
	*/
	Name() string

	/*
	   True if this is a distinct aggregate. For e.g.
	   COUNT(DISTINCT)
	*/
	Distinct() bool

	/*
	   Returns the operands of the function.
	*/
	Operands() Expressions

	/*
	   Returns the Minimum number of input arguments required
	   by the function.
	*/
	MinArgs() int

	/*
	   Returns the Maximum number of input arguments allowed
	   by the function.
	*/
	MaxArgs() int

	/*
	   Factory method pattern.
	*/
	Constructor() FunctionConstructor
}

/*
Factory method pattern.
*/
type FunctionConstructor func(operands ...Expression) Function

/*
A unary function is one that has on operand. It inherits
from Function and contains one additional method to return
the operand.
*/
type UnaryFunction interface {
	/*
	   Inherits from Function.
	*/
	Function

	/*
	   Returns the input operand.
	*/
	Operand() Expression
}

/*
A binary function is one that has two operands. It inherits
from Function and contains two additional methods to return
the first and second operand as expressions.
*/
type BinaryFunction interface {
	/*
	   Inherits from Function.
	*/
	Function

	/*
	   Returns the first input operand to a Binary Function.
	*/
	First() Expression

	/*
	   Returns the second input operand to a Binary Function.
	*/
	Second() Expression
}

/*
Base class for functions. Type FunctionBase is a struct that
implements ExpressionBase and contains additional defined
parameters such as function name, if it is volatile and the
operands of the functions which are of type expressions.
*/
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
	var buf [8]value.Value
	var args []value.Value
	if len(this.operands) <= len(buf) {
		args = buf[0:len(this.operands)]
	} else {
		args = _ARGS_POOL.GetSized(len(this.operands))
		defer _ARGS_POOL.Put(args)
	}

	for i, op := range this.operands {
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return
		}
	}

	return applied.Apply(context, args...)
}

func (this *FunctionBase) Indexable() bool {
	if this.volatile() {
		return false
	}

	for _, operand := range this.operands {
		if !operand.Indexable() {
			return false
		}
	}

	return true
}

func (this *FunctionBase) EquivalentTo(other Expression) bool {
	return !this.volatile() && this.ExpressionBase.EquivalentTo(other)
}

/*
Return the operands of the function.
*/
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

func (this *FunctionBase) Copy() Expression {
	function := this.expr.(Function)
	operands := function.Operands()
	if len(operands) == 0 {
		return function.Constructor()()
	}

	var buf [8]Expression
	var copies Expressions
	if len(operands) <= len(buf) {
		copies = buf[0:len(operands)]
	} else {
		copies = make(Expressions, len(operands))
	}

	for i, op := range operands {
		if op != nil {
			copies[i] = op.Copy()
		}
	}

	return function.Constructor()(copies...)
}

/*
Return name of the function.
*/
func (this *FunctionBase) Name() string { return this.name }

/*
Default return value is false.
*/
func (this *FunctionBase) Distinct() bool { return false }

/*
Return the operands of the function.
*/
func (this *FunctionBase) Operands() Expressions { return this.operands }

var _ARGS_POOL = value.NewValuePool(64)

/*
A Nullary function doesnt have any input operands. Type
NullaryFunctionBase is a struct that implements FunctionBase.
*/
type NullaryFunctionBase struct {
	FunctionBase
}

/*
The method NewNullaryFunctionBase returns a pointer to a
NullaryFunctionBase struct, initializing the name field to the
input name.
*/
func NewNullaryFunctionBase(name string) *NullaryFunctionBase {
	return &NullaryFunctionBase{
		FunctionBase{
			name: name,
		},
	}
}

func (this *NullaryFunctionBase) Value() value.Value {
	return nil
}

func (this *NullaryFunctionBase) Static() Expression {
	return nil
}

/*
Return false (not indexable).
*/
func (this *NullaryFunctionBase) Indexable() bool {
	return false
}

func (this *NullaryFunctionBase) CoveredBy(keyspace string, exprs Expressions, options coveredOptions) Covered {
	return CoveredTrue
}

func (this *NullaryFunctionBase) Copy() Expression {
	function := this.expr.(Function)
	return function.Constructor()()
}

func (this *NullaryFunctionBase) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	return true, nil
}

/*
Minimum input arguments required is 0.
*/
func (this *NullaryFunctionBase) MinArgs() int { return 0 }

/*
Maximum number of input arguments allowed is 0.
*/
func (this *NullaryFunctionBase) MaxArgs() int { return 0 }

/*
A Unary function has one input operand. Type UnaryFunctionBase
is a struct that implements FunctionBase.
*/
type UnaryFunctionBase struct {
	FunctionBase
}

/*
The method NewUnaryFunctionBase returns a pointer to a
UnaryFunctionBase struct, initializing the name and operand
field to the input name and input operand expression.
*/
func NewUnaryFunctionBase(name string, operand Expression) *UnaryFunctionBase {
	return &UnaryFunctionBase{
		FunctionBase{
			name:     name,
			operands: Expressions{operand},
		},
	}
}

/*
This method Evaluates the unary function. It evaluates the
operand using the input item and context, and Evaluates
this using the Apply method defined for UnaryApplied interfaces
for each defined Unary function. Return Apply's return value.
*/
func (this *UnaryFunctionBase) UnaryEval(applied UnaryApplied, item value.Value, context Context) (
	value.Value, error) {
	op := this.operands[0]
	arg, err := op.Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	return applied.Apply(context, arg)
}

/*
Minimum input arguments required is 1.
*/
func (this *UnaryFunctionBase) MinArgs() int { return 1 }

/*
Maximum number of input arguments allowed is 1.
*/
func (this *UnaryFunctionBase) MaxArgs() int { return 1 }

/*
Return the operand of the Unary Function.
*/
func (this *UnaryFunctionBase) Operand() Expression {
	return this.operands[0]
}

/*
A Binary function has two input operands. Type BinaryFunctionBase
is a struct that implements FunctionBase.
*/
type BinaryFunctionBase struct {
	FunctionBase
}

/*
The method NewBinaryFunctionBase returns a pointer to a
BinaryFunctionBase struct, initializing the name and operand
field to the input name and both input operand expressions.
*/
func NewBinaryFunctionBase(name string, first, second Expression) *BinaryFunctionBase {
	return &BinaryFunctionBase{
		FunctionBase{
			name:     name,
			operands: Expressions{first, second},
		},
	}
}

/*
This method Evaluates the binary function. It evaluates both
the operands using the input item and context, and Evaluates
this using the Apply method (by passing both arguments )
defined for BinaryApplied interfaces for each defined Binary
function. Return Apply's return value.
*/
func (this *BinaryFunctionBase) BinaryEval(applied BinaryApplied, item value.Value, context Context) (
	result value.Value, err error) {
	var args [2]value.Value
	for i, op := range this.operands {
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return
		}
	}

	return applied.Apply(context, args[0], args[1])
}

/*
Minimum input arguments required is 2.
*/
func (this *BinaryFunctionBase) MinArgs() int { return 2 }

/*
Maximum number of input arguments allowed is 2.
*/
func (this *BinaryFunctionBase) MaxArgs() int { return 2 }

/*
Return the first operand of the Binary Function.
*/
func (this *BinaryFunctionBase) First() Expression {
	return this.operands[0]
}

/*
Return the second operand of the Binary Function.
*/
func (this *BinaryFunctionBase) Second() Expression {
	return this.operands[1]
}

/*
Represents binary functions that are commutative in
nature. Type CommutativeBinaryFunctionBase is a struct
that implements BinaryFunctionBase.
*/
type CommutativeBinaryFunctionBase struct {
	BinaryFunctionBase
}

/*
Returns a pointer to a CommutativeBinaryFunctionBase
that calls the NewBinaryFunctionBase method to set
the name and the operands.
*/
func NewCommutativeBinaryFunctionBase(name string, first, second Expression) *CommutativeBinaryFunctionBase {
	return &CommutativeBinaryFunctionBase{
		BinaryFunctionBase: *NewBinaryFunctionBase(name, first, second),
	}
}

func (this *CommutativeBinaryFunctionBase) EquivalentTo(other Expression) bool {
	if this.valueEquivalentTo(other) {
		return true
	}

	that, ok := other.(Function)
	if !ok {
		return false
	}

	if this.name != that.Name() ||
		len(this.operands) != len(that.Operands()) {
		return false
	}

	return (this.operands[0].EquivalentTo(that.Operands()[0]) && this.operands[1].EquivalentTo(that.Operands()[1])) ||
		(this.operands[0].EquivalentTo(that.Operands()[1]) && this.operands[1].EquivalentTo(that.Operands()[0]))
}

/*
A Ternary function has three input operands. Type TernaryFunctionBase
is a struct that implements FunctionBase.
*/
type TernaryFunctionBase struct {
	FunctionBase
}

/*
The method NewTernaryFunctionBase returns a pointer to a
TernaryFunctionBase struct, initializing the name and operand
field to the input name and three input operand expressions.
*/
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
	var args [3]value.Value
	for i, op := range this.operands {
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return
		}
	}

	return applied.Apply(context, args[0], args[1], args[2])
}

/*
Minimum input arguments required is 3.
*/
func (this *TernaryFunctionBase) MinArgs() int { return 3 }

/*
Maximum number of input arguments allowed is 3.
*/
func (this *TernaryFunctionBase) MaxArgs() int { return 3 }

/*
Return the first operand of the Ternary Function.
*/
func (this *TernaryFunctionBase) First() Expression {
	return this.operands[0]
}

/*
Return the second operand of the Ternary Function.
*/
func (this *TernaryFunctionBase) Second() Expression {
	return this.operands[1]
}

/*
Return the third operand of the Ternary Function.
*/
func (this *TernaryFunctionBase) Third() Expression {
	return this.operands[2]
}

/*
Represents functions that are commutative in
nature. Type CommutativeFunctionBase is a struct
that implements FunctionBase.
*/
type CommutativeFunctionBase struct {
	FunctionBase
}

/*
Returns a pointer to a CommutativeFunctionBase
that uses the FunctionBase struct to set
the name and the operands.
*/
func NewCommutativeFunctionBase(name string, operands ...Expression) *CommutativeFunctionBase {
	return &CommutativeFunctionBase{
		FunctionBase{
			name:     name,
			operands: operands,
		},
	}
}

func (this *CommutativeFunctionBase) EquivalentTo(other Expression) bool {
	if this.valueEquivalentTo(other) {
		return true
	}

	that, ok := other.(Function)
	if !ok {
		return false
	}

	if this.name != that.Name() ||
		len(this.operands) != len(that.Operands()) {
		return false
	}

	var buf [8]bool
	var found []bool
	if len(this.operands) <= len(buf) {
		found = buf[0:len(this.operands)]
	} else {
		found = _FOUND_POOL.GetSized(len(this.operands))
		defer _FOUND_POOL.Put(found)
	}

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

/*
Minimum input arguments required is 2.
*/
func (this *CommutativeFunctionBase) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined is
MaxInt16  = 1<<15 - 1. This is defined using the
math package.
*/
func (this *CommutativeFunctionBase) MaxArgs() int { return math.MaxInt16 }

var _FOUND_POOL = util.NewBoolPool(64)

/*
Used to define Apply methods for general functions. The
Apply method is used to evaluate the functions, based on
its type and rules.
*/
type Applied interface {
	Apply(context Context, args ...value.Value) (value.Value, error)
}

/*
Define Apply methods to evaluate Unary functions.
*/
type UnaryApplied interface {
	Apply(context Context, arg value.Value) (value.Value, error)
}

/*
Define Apply methods to evaluate Binary functions.
*/
type BinaryApplied interface {
	Apply(context Context, first, second value.Value) (value.Value, error)
}

/*
Define Apply methods to evaluate Ternary functions.
*/
type TernaryApplied interface {
	Apply(context Context, first, second, third value.Value) (value.Value, error)
}
