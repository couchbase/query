//  Copyright 2021-Present Couchbase, Inc.
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

///////////////////////////////////////////////////
//
// InferValue
//
///////////////////////////////////////////////////

type InferValue struct {
	FunctionBase
}

func NewInferValue(operands ...Expression) Function {
	rv := &InferValue{}
	rv.Init("infer_value", operands...)

	rv.expr = rv
	return rv
}

func (this *InferValue) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *InferValue) Type() value.Type { return value.ARRAY }

func (this *InferValue) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	var with value.Value
	if len(this.operands) > 1 {
		with, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if with.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if with.Type() != value.OBJECT {
			return value.NULL_VALUE, nil
		}
	}

	v, e := context.Infer(arg, with)

	return v, e
}

func (this *InferValue) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewInferValue(operands...)
	}
}

func (this *InferValue) MinArgs() int { return 1 }

func (this *InferValue) MaxArgs() int { return 2 }

func (this *InferValue) Indexable() bool { return false }
