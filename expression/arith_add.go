//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

/*
Represents Add for arithmetic expressions. Type Add is a struct
that implements CommutativeFunctionBase.
*/
type Add struct {
	CommutativeFunctionBase
}

func NewAdd(operands ...Expression) Function {
	rv := &Add{}

	/*
		If the first input operand is an Add expression, "flatten" the structure by extracting its operands.
		And use these extracted operands directly in the new Add expression. This reduces nesting due to left-associativity in the
		constructed Add expression. Flattening is applied conservatively to preserve the intentional grouping (eg. via parantheses)
		and evaluation order, established by the parser.  As changing it can affect precision and other semantics.
		This is why flattening is applied only to the first input operand and the operands extracted from it are not recursively
		flattened further. And why later input operands are not flattened, even if they are Add expressions.
		For example a + (b + c) must not be flattened to a + b + c.
	*/
	var flatten bool
	if len(operands) > 0 {
		if add, ok := operands[0].(*Add); ok {
			flattenedOps := make(Expressions, 0, len(add.Operands())+len(operands)-1)
			flattenedOps = append(flattenedOps, add.Operands()...)

			if len(operands) > 1 {
				flattenedOps = append(flattenedOps, operands[1:]...)
			}

			flatten = true
			rv.Init("add", flattenedOps...)
		}
	}

	if !flatten {
		rv.Init("add", operands...)
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Add) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAdd(this)
}

func (this *Add) Type() value.Type { return value.NUMBER }

/*
Range over input arguments, if the type is a number add it to the sum.
If the value is missing, return a missing value. For all other types
return a null value. Return the final sum.
*/
func (this *Add) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	sum := value.ZERO_NUMBER

	for _, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if nil != err {
			return nil, err
		}
		if !null && arg.Type() == value.NUMBER {
			sum = sum.Add(value.AsNumberValue(arg))
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			null = true
		}
	}
	if null {
		return value.NULL_VALUE, nil
	}
	return sum, nil
}

/*
Factory method pattern.
*/
func (this *Add) Constructor() FunctionConstructor {
	return NewAdd
}
