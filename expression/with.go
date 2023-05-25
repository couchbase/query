//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

type With interface {
	Alias() string
	Expression() Expression
	SetExpression(expr Expression)
	ErrorContext() string
	SetErrorContext(line int, column int)
	String() string
}

type Withs []With

func (this Withs) Expressions() Expressions {
	exprs := make(Expressions, 0, len(this))
	for _, with := range this {
		exprs = append(exprs, with.Expression())
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
	}

	return
}
