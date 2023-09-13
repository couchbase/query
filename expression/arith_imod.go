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
Represents IMod for arithmetic expressions. Type IMod is a struct
that implements BinaryFunctionBase.
*/
type IMod struct {
	BinaryFunctionBase
}

func NewIMod(first, second Expression) Function {
	rv := &IMod{}
	rv.Init("imod", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IMod) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IMod) Type() value.Type { return value.NUMBER }

func (this *IMod) Evaluate(item value.Value, context Context) (value.Value, error) {
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
		return value.AsNumberValue(first).IMod(value.AsNumberValue(second)), nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *IMod) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIMod(operands[0], operands[1])
	}
}
