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
Represents multiplication for arithmetic expressions. Type Mult is a
struct that implements CommutativeFunctionBase.
*/
type Mult struct {
	CommutativeFunctionBase
}

func NewMult(operands ...Expression) Function {
	rv := &Mult{}
	rv.Init("mult", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Mult) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMult(this)
}

func (this *Mult) Type() value.Type { return value.NUMBER }

/*
Range over input arguments, if the type is a number multiply it to
the product. If the value is missing, return a missing value. For
all other types return a null value. Return the final product.
*/
func (this *Mult) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	prod := value.ONE_NUMBER

	for _, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if !null && arg.Type() == value.NUMBER {
			prod = prod.Mult(value.AsNumberValue(arg))
		} else {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return prod, nil
}

/*
Factory method pattern.
*/
func (this *Mult) Constructor() FunctionConstructor {
	return NewMult
}
