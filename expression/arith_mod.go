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

/*
Represents Mod for arithmetic expressions. Type Mod is a struct
that implements BinaryFunctionBase.
*/
type Mod struct {
	BinaryFunctionBase
}

func NewMod(first, second Expression) Function {
	rv := &Mod{}
	rv.Init("mod", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Mod) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMod(this)
}

func (this *Mod) Type() value.Type { return value.NUMBER }

func (this *Mod) Evaluate(item value.Value, context Context) (value.Value, error) {
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
			return value.NULL_VALUE, nil
		}

		if first.Type() == value.NUMBER {
			m := math.Mod(first.Actual().(float64), s)
			return value.NewValue(m), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *Mod) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMod(operands[0], operands[1])
	}
}
