//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/expression"
)

type With struct {
	alias        string
	expr         expression.Expression
	errorContext expression.ErrorContext
}

func NewWith(alias string, expr expression.Expression) expression.With {
	return &With{
		alias: alias,
		expr:  expr,
	}
}

func (this *With) Copy() expression.With {
	return &With{
		alias:        this.alias,
		expr:         this.expr,
		errorContext: this.errorContext,
	}
}

func (this *With) Alias() string {
	return this.alias
}

func (this *With) Expression() expression.Expression {
	return this.expr
}

func (this *With) SetExpression(expr expression.Expression) {
	this.expr = expr
}

func (this *With) ErrorContext() string {
	return this.errorContext.String()
}

func (this *With) SetErrorContext(line int, column int) {
	this.errorContext.Set(line, column)
}

/*
Representation as a N1QL string
*/
func (this *With) String() string {
	s := "`" + this.alias + "`" + " AS ( "
	s += this.expr.String()
	s += " ) "

	return s
}

func (this *With) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["alias"] = this.alias
	r["expr"] = this.expr.String()

	return json.Marshal(r)
}
