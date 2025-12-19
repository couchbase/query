//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type Advise struct {
	statementBase

	stmt    Statement `json:"stmt"`
	query   string    `json:"query"`
	context interface{}
}

func NewAdvise(stmt Statement, text string) *Advise {
	rv := &Advise{
		stmt:  stmt,
		query: text,
	}
	rv.statementBase.stmt = rv
	return rv
}

func (this *Advise) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAdvise(this)
}

func (this *Advise) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

func (this *Advise) Formalize() error {
	return this.stmt.Formalize()
}

func (this *Advise) MapExpressions(mapper expression.Mapper) error {
	return this.stmt.MapExpressions(mapper)
}

func (this *Advise) Expressions() expression.Expressions {
	return this.stmt.Expressions()
}

func (this *Advise) Privileges() (*auth.Privileges, errors.Error) {
	return this.stmt.Privileges()
}

func (this *Advise) Statement() Statement {
	return this.stmt
}

func (this *Advise) Query() string {
	return this.query
}

func (this *Advise) Type() string {
	return "ADVISE"
}

func (this *Advise) SetContext(context interface{}) {
	this.context = context
}

func (this *Advise) Context() interface{} {
	return this.context
}

func (this *Advise) String() string {
	if this.stmt != nil {
		return "ADVISE " + this.stmt.String()
	}
	return ""
}
