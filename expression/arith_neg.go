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
Represents negation for arithmetic expressions. Type Neg is a struct
that implements UnaryFunctionBase.
*/
type Neg struct {
	UnaryFunctionBase
}

func NewNeg(operand Expression) Function {
	rv := &Neg{}
	rv.Init("neg", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Neg) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNeg(this)
}

func (this *Neg) Type() value.Type { return value.NUMBER }

/*
Return the neagation of the input value, if the type of input is a number.
For missing return a missing value, and for all other input types return a
null.
*/
func (this *Neg) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.NUMBER {
		return value.AsNumberValue(arg).Neg(), nil
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *Neg) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNeg(operands[0])
	}
}
