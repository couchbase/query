//  Copyright 2022-Present Couchbase, Inc.
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

type IsDistinctFrom struct {
	BinaryFunctionBase
	operator bool
}

func NewIsDistinctFrom(first, second Expression) Function {
	rv := &IsDistinctFrom{}
	rv.Init("isdistinctfrom", first, second)
	rv.expr = rv
	return rv
}

func (this *IsDistinctFrom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsDistinctFrom) Type() value.Type { return value.BOOLEAN }

func (this *IsDistinctFrom) SetOperator() {
	this.operator = true
}

func (this *IsDistinctFrom) Operator() string {
	if this.operator {
		return " IS DISTINCT FROM "
	}
	return ""
}

func (this *IsDistinctFrom) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if first.Type() == value.MISSING && second.Type() == value.MISSING {
		return value.FALSE_VALUE, nil
	} else if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.TRUE_VALUE, nil
	} else if first.Type() == value.NULL && second.Type() == value.NULL {
		return value.FALSE_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL {
		return value.TRUE_VALUE, nil
	}
	if first.Equals(second) == value.TRUE_VALUE {
		return value.FALSE_VALUE, nil
	}
	return value.TRUE_VALUE, nil
}

func (this *IsDistinctFrom) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsDistinctFrom(operands[0], operands[1])
	}
}

func (this *IsDistinctFrom) PropagatesMissing() bool {
	return false
}

func (this *IsDistinctFrom) PropagatesNull() bool {
	return false
}

type IsNotDistinctFrom struct {
	BinaryFunctionBase
	operator bool
}

func NewIsNotDistinctFrom(first, second Expression) Function {
	rv := &IsNotDistinctFrom{}
	rv.Init("isnotdistinctfrom", first, second)

	rv.expr = rv
	return rv
}

func (this *IsNotDistinctFrom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsNotDistinctFrom) Type() value.Type { return value.BOOLEAN }

func (this *IsNotDistinctFrom) SetOperator() {
	this.operator = true
}

func (this *IsNotDistinctFrom) Operator() string {
	if this.operator {
		return " IS NOT DISTINCT FROM "
	}
	return ""
}

func (this *IsNotDistinctFrom) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if first.Type() == value.MISSING && second.Type() == value.MISSING {
		return value.TRUE_VALUE, nil
	} else if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.FALSE_VALUE, nil
	} else if first.Type() == value.NULL && second.Type() == value.NULL {
		return value.TRUE_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL {
		return value.FALSE_VALUE, nil
	}
	return first.Equals(second), nil
}

func (this *IsNotDistinctFrom) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotDistinctFrom(operands[0], operands[1])
	}
}

func (this *IsNotDistinctFrom) PropagatesMissing() bool {
	return false
}

func (this *IsNotDistinctFrom) PropagatesNull() bool {
	return false
}
