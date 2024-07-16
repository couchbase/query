//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type ParenInfer struct {
	expression.ExpressionBase
	infer Statement
}

func NewParenInfer(infer Statement) *ParenInfer {
	rv := &ParenInfer{
		infer: infer,
	}

	rv.SetExpr(rv)
	return rv
}

func (this *ParenInfer) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitParenInfer(this)
}

func (this *ParenInfer) Children() expression.Expressions {

	if v, ok := this.infer.(*InferExpression); ok {
		return v.Expressions()
	}
	return nil
}

func (this *ParenInfer) Copy() expression.Expression {
	return this
}

func (this *ParenInfer) Evaluate(item value.Value, context expression.Context) (value.Value, error) {

	var inferRes value.Value
	var inferErr error

	switch v := this.infer.(type) {
	case *InferExpression:
		exprValue, evalErr := v.Expression().Evaluate(item, context)
		if evalErr != nil {
			return nil, evalErr
		}

		if exprValue.Type() == value.MISSING || exprValue.Type() == value.NULL {
			return exprValue, nil
		}

		inferRes, inferErr = context.Infer(exprValue, v.With())
		if inferErr != nil {
			return nil, inferErr
		}
	case *InferKeyspace:
		inferRes, inferErr = context.InferKeyspace(v.Keyspace(), v.With())
		if inferErr != nil {
			return nil, inferErr
		}
	default:
		inferRes = value.NULL_VALUE
	}

	return inferRes, nil
}

func (this *ParenInfer) MapChildren(m expression.Mapper) error {
	return this.infer.MapExpressions(m)
}

func (this *ParenInfer) Type() value.Type {
	return value.ARRAY
}

func (this *ParenInfer) EquivalentTo(other expression.Expression) bool {
	return false
}

/*
Returns all required privileges.
*/
func (this *ParenInfer) Privileges() *auth.Privileges {
	priv, _ := this.infer.Privileges()
	return priv
}

func (this *ParenInfer) String() string {

	var s string

	switch v := this.infer.(type) {
	case *InferKeyspace:
		s = "INFER KEYSPACE " + v.keyspace.FullName()
		if v.with != nil {
			s = s + " WITH " + v.with.String()
		}
	case *InferExpression:
		s = "INFER " + v.expr.String()
		if v.with != nil {
			s = s + " WITH " + v.with.String()
		}
	}

	return s
}

// Unused-> only to make the interface more specific
func (this *ParenInfer) IsInferKeyspace() bool {
	_, ok := this.infer.(*InferKeyspace)
	return ok
}

// Unused-> only to make the interface more specific
func (this *ParenInfer) IsInferExpression() bool {
	_, ok := this.infer.(*InferExpression)
	return ok
}
