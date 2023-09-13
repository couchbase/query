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
// IfInf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFINF(expr1, expr2, ...)
for numbers. It returns the first non-MISSING, non-Inf number or
NULL.
*/
type IfInf struct {
	FunctionBase
}

func NewIfInf(operands ...Expression) Function {
	rv := &IfInf{}
	rv.Init("ifinf", operands...)

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IfInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfInf) Type() value.Type { return value.NUMBER }

/*
First non missing, non infinity number in the input argument values,
or null.
*/
func (this *IfInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	var rv value.Value
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if rv == nil {
			if a.Type() == value.MISSING {
				missing = true
			} else if a.Type() != value.NUMBER && rv == nil {
				rv = value.NULL_VALUE
			} else {
				f := a.Actual().(float64)
				if !math.IsInf(f, 0) {
					rv = value.NewValue(f)
				}
			}
		}
	}
	if rv == nil {
		if missing {
			rv = value.MISSING_VALUE
		} else {
			rv = value.NULL_VALUE
		}
	}
	return rv, nil
}

func (this *IfInf) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Minimum input arguments required is 2
*/
func (this *IfInf) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the IfInf function is
MaxInt16 = 1<<15 - 1.
*/
func (this *IfInf) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *IfInf) Constructor() FunctionConstructor {
	return NewIfInf
}

///////////////////////////////////////////////////
//
// IfNaN
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFNAN(expr1, expr2, ...).  It
returns the first non-MISSING, non-NaN number or NULL.
*/
type IfNaN struct {
	FunctionBase
}

func NewIfNaN(operands ...Expression) Function {
	rv := &IfNaN{}
	rv.Init("ifnan", operands...)

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IfNaN) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNaN) Type() value.Type { return value.NUMBER }

func (this *IfNaN) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	var rv value.Value
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if rv == nil {
			if a.Type() == value.MISSING {
				missing = true
			} else if a.Type() != value.NUMBER {
				rv = value.NULL_VALUE
			} else {
				f := a.Actual().(float64)
				if !math.IsNaN(f) {
					rv = value.NewValue(f)
				}
			}
		}
	}
	if rv == nil {
		if missing {
			rv = value.MISSING_VALUE
		} else {
			rv = value.NULL_VALUE
		}
	}
	return rv, nil
}

func (this *IfNaN) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Minimum input arguments required is 2.
*/
func (this *IfNaN) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the IfNaN
function is MaxInt16  = 1<<15 - 1.
*/
func (this *IfNaN) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *IfNaN) Constructor() FunctionConstructor {
	return NewIfNaN
}

///////////////////////////////////////////////////
//
// IfNaNOrInf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function IFNANORINF(expr1, expr2, ...).
It returns the first non-MISSING, non-Inf, non-NaN number or NULL.
*/
type IfNaNOrInf struct {
	FunctionBase
}

func NewIfNaNOrInf(operands ...Expression) Function {
	rv := &IfNaNOrInf{}
	rv.Init("ifnanorinf", operands...)

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IfNaNOrInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNaNOrInf) Type() value.Type { return value.NUMBER }

func (this *IfNaNOrInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	var rv value.Value
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if rv == nil {
			if a.Type() == value.MISSING {
				missing = true
			} else if a.Type() != value.NUMBER {
				rv = value.NULL_VALUE
			} else {
				f := a.Actual().(float64)
				if !math.IsInf(f, 0) && !math.IsNaN(f) {
					rv = value.NewValue(f)
				}
			}
		}
	}
	if rv == nil {
		if missing {
			rv = value.MISSING_VALUE
		} else {
			rv = value.NULL_VALUE
		}
	}
	return rv, nil
}

func (this *IfNaNOrInf) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

/*
Minimum input arguments required is 2
*/
func (this *IfNaNOrInf) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the
function is MaxInt16  = 1<<15 - 1.
*/
func (this *IfNaNOrInf) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *IfNaNOrInf) Constructor() FunctionConstructor {
	return NewIfNaNOrInf
}

///////////////////////////////////////////////////
//
// NaNIf
//
///////////////////////////////////////////////////

/*
This represents the Conditional function NANIF(expr1, expr2).
It returns a NaN if expr1 = expr2; else expr1.
*/
type NaNIf struct {
	BinaryFunctionBase
}

func NewNaNIf(first, second Expression) Function {
	rv := &NaNIf{}
	rv.Init("nanif", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NaNIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NaNIf) Type() value.Type { return value.JSON }

/*
This method checks to see if the values of the two input expressions
are equal, and if true then returns a NaN. If not it returns the first
input value. Use the Equals method for the two values to determine
equality.
*/
func (this *NaNIf) Evaluate(item value.Value, context Context) (value.Value, error) {
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
			return _NAN_VALUE, nil
		} else {
			return first, nil
		}
	}
}

/*
Factory method pattern.
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
It returns NegInf if expr1 = expr2; else expr1.
*/
type NegInfIf struct {
	BinaryFunctionBase
}

func NewNegInfIf(first, second Expression) Function {
	rv := &NegInfIf{}
	rv.Init("neginfif", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *NegInfIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NegInfIf) Type() value.Type { return value.JSON }

/*
This method checks to see if the values of the two input expressions
are equal, and if true then returns a negative infinity.. If not it
returns the first input value. Use the Equals method for the two
values to determine equality.
*/
func (this *NegInfIf) Evaluate(item value.Value, context Context) (value.Value, error) {
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
			return _NEG_INF_VALUE, nil
		} else {
			return first, nil
		}
	}
}

var _NEG_INF_VALUE = value.NewValue(math.Inf(-1))

/*
Factory method pattern.
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
It returns PosInf if expr1 = expr2; else expr1.
*/
type PosInfIf struct {
	BinaryFunctionBase
}

func NewPosInfIf(first, second Expression) Function {
	rv := &PosInfIf{}
	rv.Init("posinfif", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *PosInfIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *PosInfIf) Type() value.Type { return value.JSON }

/*
This method checks to see if the values of the two input expressions
are equal, and if true then returns a positive infinity. If not it
returns the first input value. Use the Equals method for the two
values to determine equality.
*/
func (this *PosInfIf) Evaluate(item value.Value, context Context) (value.Value, error) {
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
			return _POS_INF_VALUE, nil
		} else {
			return first, nil
		}
	}
}

var _POS_INF_VALUE = value.NewValue(math.Inf(1))

/*
Factory method pattern.
*/
func (this *PosInfIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosInfIf(operands[0], operands[1])
	}
}
