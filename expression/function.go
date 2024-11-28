//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"math"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const RANGE_LIMIT = math.MaxInt32 // Maximum range/repeat value

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
	   True if this is a aggregate function
	*/
	Aggregate() bool

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

	SetOperator()

	Operator() string
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

func (this *FunctionBase) Init(name string, operands ...Expression) {
	this.name = name
	this.operands = operands
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
	var rv Function
	function := this.expr.(Function)
	operands := function.Operands()
	if len(operands) == 0 {
		rv = function.Constructor()()
	} else {
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

		rv = function.Constructor()(copies...)
	}

	rv.BaseCopy(this.expr)
	return rv
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
Default return value is false.
*/
func (this *FunctionBase) Aggregate() bool { return false }

/*
Return the operands of the function.
*/
func (this *FunctionBase) Operands() Expressions { return this.operands }

func (this *FunctionBase) SetOperator() {}

func (this *FunctionBase) Operator() string { return "" }

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

func (this *NullaryFunctionBase) StaticNoVariable() Expression {
	return nil
}

/*
Return false (not indexable).
*/
func (this *NullaryFunctionBase) Indexable() bool {
	return false
}

func (this *NullaryFunctionBase) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {
	return CoveredTrue
}

func (this *NullaryFunctionBase) Copy() Expression {
	function := this.expr.(Function)
	rv := function.Constructor()()
	rv.BaseCopy(this.expr)
	return rv
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
	return this.doEquivalentTo(other)
}

func (this *CommutativeFunctionBase) doEquivalentTo(other Expression) bool {
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

type UserDefinedFunctionBase struct {
	FunctionBase
}

/*
The method NewUserDefinedFunctionBase returns a pointer to a
UserDefinedFunctionBase struct, initializing the name
*/
func NewUserDefinedFunctionBase(name string, operands ...Expression) *UserDefinedFunctionBase {
	return &UserDefinedFunctionBase{
		FunctionBase{
			name:     name,
			operands: operands,
		},
	}
}

var _FOUND_POOL = util.NewBoolPool(64)
