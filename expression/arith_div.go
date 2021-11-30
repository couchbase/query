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
Represents Div for arithmetic expressions. Type Div is a struct
that implements BinaryFunctionBase.
*/
type Div struct {
	BinaryFunctionBase
}

func NewDiv(first, second Expression) Function {
	rv := &Div{
		*NewBinaryFunctionBase("div", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Div) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDiv(this)
}

func (this *Div) Type() value.Type { return value.NUMBER }

func (this *Div) Evaluate(item value.Value, context Context) (value.Value, error) {
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

	if second.Type() == value.NUMBER {
		s := second.Actual().(float64)
		if s == 0.0 {
			if ectx, ok := context.(interface{ Warning(errors.Error) }); ok {
				ectx.Warning(errors.NewDivideByZeroWarning())
			}
			return value.NULL_VALUE, nil
		}

		if first.Type() == value.NUMBER {
			d := first.Actual().(float64) / s
			return value.NewValue(d), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *Div) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDiv(operands[0], operands[1])
	}
}
