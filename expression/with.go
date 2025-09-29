//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"strings"

	"github.com/couchbase/query/value"
)

type With interface {
	Alias() string
	Expression() Expression
	SetExpression(expr Expression)
	SetRecursiveExpression(rexpr Expression)
	IsRecursive() bool
	SetUnion()
	IsUnion() bool
	RecursiveExpression() Expression
	Config() value.Value
	CycleFields() Expressions
	SplitRecursive() error
	ErrorContext() string
	SetErrorContext(line int, column int)
	GetErrorContext() (int, int)
	String() string
	WriteSyntaxString(sb *strings.Builder)
}

type Withs []With

func (this Withs) Expressions() Expressions {
	exprs := make(Expressions, 0, len(this))
	for _, with := range this {
		exprs = append(exprs, with.Expression())
		if with.IsRecursive() {
			exprs = append(exprs, with.RecursiveExpression())
		}
	}

	return exprs
}

func (this Withs) MapExpressions(mapper Mapper) (err error) {
	for _, b := range this {
		expr, err := mapper.Map(b.Expression())
		if err != nil {
			return err
		}

		b.SetExpression(expr)

		if b.IsRecursive() {
			rexpr, err := mapper.Map(b.RecursiveExpression())
			if err != nil {
				return err
			}
			b.SetRecursiveExpression(rexpr)
		}
	}

	return
}
