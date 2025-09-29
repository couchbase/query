//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type InferExpression struct {
	statementBase

	expr  expression.Expression
	using datastore.InferenceType `json:"using"`
	with  value.Value             `json:"with"`
}

func NewInferExpression(expr expression.Expression, using datastore.InferenceType, with value.Value) *InferExpression {
	rv := &InferExpression{
		expr:  expr,
		using: using,
		with:  with,
	}

	rv.stmt = rv
	return rv
}

func (this *InferExpression) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferExpression(this)
}

func (this *InferExpression) Signature() value.Value {
	return nil
}

func (this *InferExpression) Formalize() error {
	f := expression.NewFormalizer("", nil)
	return this.MapExpressions(f)
}

func (this *InferExpression) MapExpressions(mapper expression.Mapper) error {
	expr, err := mapper.Map(this.expr)
	if err == nil {
		this.expr = expr
	}
	return err
}

func (this *InferExpression) Expressions() expression.Expressions {
	return append(expression.Expressions(nil), this.expr)
}

/*
Returns all required privileges.
*/
func (this *InferExpression) Privileges() (*auth.Privileges, errors.Error) {
	return nil, nil
}

func (this *InferExpression) Expression() expression.Expression {
	return this.expr
}

func (this *InferExpression) Using() datastore.InferenceType {
	return this.using
}

func (this *InferExpression) With() value.Value {
	return this.with
}

func (this *InferExpression) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "InferExpression"}
	r["expr"] = this.expr
	r["using"] = this.using
	r["with"] = this.with
	return json.Marshal(r)
}

func (this *InferExpression) Type() string {
	return "INFER"
}

func (this *InferExpression) String() string {
	var s strings.Builder
	s.WriteString("INFER ")
	s.WriteString(this.expr.String())

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
