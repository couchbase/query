//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"math"

	"github.com/couchbase/query/value"
)

type Pow struct {
	BinaryFunctionBase
}

func NewPow(first, second Expression) Function {
	rv := &Pow{
		*NewBinaryFunctionBase("pow", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *Pow) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPow(this)
}

func (this *Pow) Type() value.Type { return value.NUMBER }

func (this *Pow) Evaluate(item value.Value, context Context) (value.Value, error) {
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

	if first.Type() != value.NUMBER || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	m := math.Pow(first.(value.NumberValue).Float64(), second.(value.NumberValue).Float64())
	return value.NewValue(m), nil
}

func (this *Pow) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPow(operands[0], operands[1])
	}
}
