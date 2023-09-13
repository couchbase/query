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
	rv.Init("add", operands...)

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
