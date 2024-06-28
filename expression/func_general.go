//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"encoding/json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Len
//
///////////////////////////////////////////////////

type Len struct {
	UnaryFunctionBase
}

func NewLen(operand Expression) Function {
	rv := &Len{
		*NewUnaryFunctionBase("len", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Len) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Len) Type() value.Type { return value.NUMBER }

func (this *Len) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING:
		return value.MISSING_VALUE, nil
	case value.STRING:
		return value.NewValue(arg.Size()), nil
	case value.OBJECT:
		oa := arg.Actual().(map[string]interface{})
		return value.NewValue(len(oa)), nil
	case value.ARRAY:
		aa := arg.Actual().([]interface{})
		return value.NewValue(len(aa)), nil
	case value.BINARY:
		return value.NewValue(arg.Size()), nil
	case value.BOOLEAN:
		return value.ONE_VALUE, nil
	case value.NUMBER:
		return value.NewValue(len(arg.ToString())), nil
	}
	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *Len) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLen(operands[0])
	}
}

// Finderr
type Finderr struct {
	UnaryFunctionBase
}

func NewFinderr(operand Expression) Function {
	rv := &Finderr{}
	// rv.Init("finderr", operand)
	rv.name = "finderr"
	rv.operands = append(rv.operands, operand)
	rv.expr = rv
	return rv
}
func (this *Finderr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}
func (this *Finderr) Type() value.Type { return value.OBJECT }
func (this *Finderr) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING && arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}
	if arg.Type() == value.NUMBER {
		ed := errors.DescribeError(errors.ErrorCode(value.AsNumberValue(arg).Int64()))
		if ed == nil {
			return value.NULL_VALUE, nil
		}
		b, err := json.Marshal(ed)
		if err != nil {
			return value.NULL_VALUE, nil
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(b, &m)
		if err != nil {
			return value.NULL_VALUE, nil
		}
		res := make([]interface{}, 1)
		res[0] = m
		return value.NewValue(res), nil
	} else {
		errs := errors.SearchErrors(arg.ToString())
		if len(errs) == 0 {
			return value.NULL_VALUE, nil
		}
		res := make([]interface{}, 0, len(errs))
		for _, ed := range errs {
			b, err := json.Marshal(ed)
			if err != nil {
				return value.NULL_VALUE, nil
			}
			m := make(map[string]interface{})
			err = json.Unmarshal(b, &m)
			if err != nil {
				return value.NULL_VALUE, nil
			}
			res = append(res, value.NewValue(m))
		}
		return value.NewValue(res), nil
	}
}
func (this *Finderr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewFinderr(operands[0])
	}
}
func (this *Finderr) Indexable() bool {
	return false
}
