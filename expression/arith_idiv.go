//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

/*
Represents integer div for arithmetic expressions. Type IDiv is a
struct that implements BinaryFunctionBase.
*/
type IDiv struct {
	BinaryFunctionBase
}

func NewIDiv(first, second Expression) Function {
	rv := &IDiv{
		*NewBinaryFunctionBase("idiv", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IDiv) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IDiv) Type() value.Type { return value.NUMBER }

func (this *IDiv) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.NUMBER && second.Type() == value.NUMBER {
		s := value.AsNumberValue(second)
		if s.Int64() == 0 {
			if ectx, ok := context.(interface{ Warning(errors.Error) }); ok {
				ectx.Warning(errors.NewDivideByZeroWarning())
			}
			return value.NULL_VALUE, nil
		}
		return value.AsNumberValue(first).IDiv(s), nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *IDiv) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIDiv(operands[0], operands[1])
	}
}
