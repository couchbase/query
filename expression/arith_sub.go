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
Represents subtraction for arithmetic expressions. Type Sub is a
struct that implements BinaryFunctionBase.
*/
type Sub struct {
	BinaryFunctionBase
}

func NewSub(first, second Expression) Function {
	rv := &Sub{}
	rv.Init("sub", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Sub) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSub(this)
}

func (this *Sub) Type() value.Type { return value.NUMBER }

/*
Evaluate the difference for the first and second input
values to return a value. If both values are numbers, calculate
the difference and return it. If either of the expressions is
missing then return a missing value. For all other cases return
a null value.
*/
func (this *Sub) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.NUMBER && second.Type() == value.NUMBER {
		return value.AsNumberValue(first).Sub(value.AsNumberValue(second)), nil
	} else if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *Sub) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSub(operands[0], operands[1])
	}
}
