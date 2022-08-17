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

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// IfMissing
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFMISSING(expr1, expr2, ...).
It returns the first non-MISSING value.
*/
type IfMissing struct {
	FunctionBase
}

func NewIfMissing(operands ...Expression) Function {
	rv := &IfMissing{
		*NewFunctionBase("ifmissing", operands...),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IfMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfMissing) Type() value.Type { return value.JSON }

/*
This method returns the first non missing value. Range over
the input arguments and check for its type. For all values
other than a missing, return the value itself. Otherwise
return a Null.
*/
func (this *IfMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if rv == nil && a.Type() != value.MISSING {
			rv = a
		}
	}

	if rv == nil {
		return value.NULL_VALUE, nil
	}
	return rv, nil
}

func (this *IfMissing) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Minimum input arguments required is 2
*/
func (this *IfMissing) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined is MaxInt16 = 1<<15 - 1.
*/
func (this *IfMissing) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *IfMissing) Constructor() FunctionConstructor {
	return NewIfMissing
}

///////////////////////////////////////////////////
//
// IfMissingOrNull
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFMISSINGORNULL(expr1, expr2,
...). It returns the first non-NULL, non-MISSING value.
*/
type IfMissingOrNull struct {
	FunctionBase
}

func NewIfMissingOrNull(operands ...Expression) Function {
	rv := &IfMissingOrNull{
		*NewFunctionBase("ifmissingornull", operands...),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IfMissingOrNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfMissingOrNull) Type() value.Type { return value.JSON }

/*
This method returns the first non-NULL, non-MISSING value, or null.
*/
func (this *IfMissingOrNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if rv == nil && a.Type() > value.NULL {
			rv = a
		}
	}

	if rv == nil {
		return value.NULL_VALUE, nil
	}
	return rv, nil
}

func (this *IfMissingOrNull) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Minimum input arguments required is 2.
*/
func (this *IfMissingOrNull) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined is MaxInt16 = 1<<15 - 1.
*/
func (this *IfMissingOrNull) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *IfMissingOrNull) Constructor() FunctionConstructor {
	return NewIfMissingOrNull
}

///////////////////////////////////////////////////
//
// IfNull
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFNULL(expr1, expr2, ...).
It returns the first non-NULL value. Note that this function may
return MISSING.
*/
type IfNull struct {
	FunctionBase
}

func NewIfNull(operands ...Expression) Function {
	rv := &IfNull{
		*NewFunctionBase("ifnull", operands...),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IfNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNull) Type() value.Type { return value.JSON }

/*
This method returns the first non null value, or null.
*/
func (this *IfNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if rv == nil && a.Type() != value.NULL {
			rv = a
		}
	}

	if rv == nil {
		return value.NULL_VALUE, nil
	}
	return rv, nil
}

func (this *IfNull) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Minimum input arguments required is 2.
*/
func (this *IfNull) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined is MaxInt16 = 1<<15 - 1.
*/
func (this *IfNull) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *IfNull) Constructor() FunctionConstructor {
	return NewIfNull
}

///////////////////////////////////////////////////
//
// MissingIf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function MISSINGIF(expr1, expr2).  It
returns MISSING if expr1 = expr2, else expr1.
*/
type MissingIf struct {
	BinaryFunctionBase
}

func NewMissingIf(first, second Expression) Function {
	rv := &MissingIf{
		*NewBinaryFunctionBase("missingif", first, second),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *MissingIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MissingIf) Type() value.Type { return value.JSON }

/*
This method checks to see if the values of the two input expressions
are equal, and if true then returns a missing value. If not it returns
the first input value. Use the Equals method for the two values to
determine equality.
*/
func (this *MissingIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	eq := first.Equals(second)
	switch eq.Type() {
	case value.MISSING, value.NULL:
		return eq, nil
	default:
		if eq.Truth() {
			return value.MISSING_VALUE, nil
		} else {
			return first, nil
		}
	}
}

func (this *MissingIf) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Factory method pattern.
*/
func (this *MissingIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMissingIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// NullIf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function NULLIF(expr1, expr2).  It
returns a NULL if expr1 = expr2; else expr1.
*/
type NullIf struct {
	BinaryFunctionBase
}

func NewNullIf(first, second Expression) Function {
	rv := &NullIf{
		*NewBinaryFunctionBase("nullif", first, second),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NullIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NullIf) Type() value.Type { return value.JSON }

/*
This method checks to see if the values of the two input expressions
are equal, and if true then returns a null value. If not it returns
the first input value. Use the Equals method for the two values to
determine equality.
*/
func (this *NullIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	eq := first.Equals(second)
	switch eq.Type() {
	case value.MISSING, value.NULL:
		return eq, nil
	default:
		if eq.Truth() {
			return value.NULL_VALUE, nil
		} else {
			return first, nil
		}
	}
}

/*
Factory method pattern.
*/
func (this *NullIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNullIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// NVL
//
///////////////////////////////////////////////////

/*
This represents the Conditional function NVL (expr1, expr2).
Cases:
Expr1 is Null: return expr2;
Expr1 is Missing: return expr2;
For all other values of Expr1: return Expr1.
*/

type NVL struct {
	BinaryFunctionBase
}

func NewNVL(first, second Expression) Function {
	rv := &NVL{
		*NewBinaryFunctionBase("nvl", first, second),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NVL) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NVL) Type() value.Type { return value.JSON }

/*
Cases:
Expr1 is Null: return expr2;
Expr1 is Missing: return expr2;
For all other values of Expr1: return Expr1.
*/
func (this *NVL) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() > value.NULL {
		return first, nil
	}
	return second, nil
}

func (this *NVL) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Factory method pattern.
*/
func (this *NVL) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNVL(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// NVL2
//
///////////////////////////////////////////////////

/*
This represents the Conditional function NVL2 (expr1, expr2, expr3).
Case expr1 is neither missing nor NULL: return expr2
Case expr1 is missing or NULL: return expr3
*/

type NVL2 struct {
	TernaryFunctionBase
}

func NewNVL2(first, second, third Expression) Function {
	rv := &NVL2{
		*NewTernaryFunctionBase("nvl2", first, second, third),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

func (this *NVL2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NVL2) Type() value.Type { return value.JSON }

func (this *NVL2) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	third, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if first.Type() > value.NULL {
		return second, nil
	}
	return third, nil
}

/*
Factory method pattern.
*/
func (this *NVL2) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNVL2(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// Decode
//
///////////////////////////////////////////////////
/*
This represents the Conditional function Decode(expr, search1, result1, ..., searchN, resultN, default(optional))
It compares expr to each search value one by one.
If expr is equal to a search, it returns the corresponding result.
If no match is found, it returns the default value.
If default is omitted, it returns value.NULL_VALUE.
*/

type Decode struct {
	FunctionBase
}

func NewDecode(operands ...Expression) Function {
	rv := &Decode{
		*NewFunctionBase("decode", operands...),
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

func (this *Decode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Decode) Type() value.Type { return value.JSON }

func (this *Decode) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	length := len(this.operands) - 1

	def := value.NULL_VALUE
	if length%2 == 1 {
		def, err = this.operands[length].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}

	var rv value.Value
	for i := 1; i+1 < len(this.operands); i += 2 {
		arg, err := this.operands[i].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		res, err := this.operands[i+1].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		if rv == nil && first.EquivalentTo(arg) {
			rv = res
		}
	}

	if rv == nil {
		return def, nil
	}

	return rv, nil
}

func (this *Decode) MinArgs() int { return 3 }

func (this *Decode) MaxArgs() int { return math.MaxInt32 }

func (this *Decode) Constructor() FunctionConstructor {
	return NewDecode
}
