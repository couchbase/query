//  Copyright 2021-Present Couchbase, Inc.
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

// Len

type Len struct {
	UnaryFunctionBase
}

func NewLen(operand Expression) Function {
	rv := &Len{}
	rv.Init("len", operand)

	rv.expr = rv
	return rv
}

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

func (this *Len) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLen(operands[0])
	}
}

// Evaluate

type Evaluate struct {
	FunctionBase
}

func NewEvaluate(operands ...Expression) Function {
	rv := &Evaluate{}
	rv.Init("evaluate", operands...)

	rv.expr = rv
	return rv
}

func (this *Evaluate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Evaluate) Type() value.Type { return value.OBJECT }

func (this *Evaluate) Evaluate(item value.Value, context Context) (value.Value, error) {
	var stmt string
	var named map[string]value.Value
	var positional value.Values

	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			if arg.Type() == value.MISSING {
				missing = true
			} else if arg.Type() != value.STRING {
				null = true
			}
			stmt = arg.ToString()
		} else {
			if arg.Type() == value.OBJECT {
				act := arg.Actual().(map[string]interface{})
				named = make(map[string]value.Value, len(act))
				for k, v := range act {
					named[k] = value.NewValue(v)
				}
			} else if arg.Type() == value.ARRAY {
				act := arg.Actual().([]interface{})
				positional = make(value.Values, 0, len(act))
				for i := range act {
					positional = append(positional, value.NewValue(act[i]))
				}
			} else {
				null = true
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// only read-only statements are permitted
	pcontext, ok := context.(ParkableContext)
	if !ok {
		return value.NULL_VALUE, nil
	}
	rv, _, err := pcontext.ParkableEvaluateStatement(stmt, named, positional, false, true, false, "")
	if err != nil {
		// to help with diagnosing problems in the provided statement, we return the error encountered and not just the NULL_VALUE
		return value.NULL_VALUE, errors.NewEvaluationError(err, "statement")
	}
	return rv, nil
}

func (this *Evaluate) MinArgs() int { return 1 }

func (this *Evaluate) MaxArgs() int { return 2 }

func (this *Evaluate) Constructor() FunctionConstructor {
	return NewEvaluate
}

func (this *Evaluate) Indexable() bool {
	return false
}
