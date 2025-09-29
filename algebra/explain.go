//  Copyright 2014-Present Couchbase, Inc.
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

/*
Represents the explain text for a query. Type Explain is
a struct that represents the explain json statement.
*/
type Explain struct {
	statementBase

	stmt Statement `json:"stmt"`
	text string    `json:"text"`
}

/*
The function NewExplain returns a pointer to the Explain
struct that has its field stmt set to the input Statement.
*/
func NewExplain(stmt Statement, text string) *Explain {
	rv := &Explain{
		stmt: stmt,
		text: text,
	}

	rv.statementBase.stmt = rv
	return rv
}

/*
It calls the VisitExplain method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Explain) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplain(this)
}

/*
This method returns the shape of the result, which is
a JSON string value.
*/
func (this *Explain) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

/*
Call Formalize for the input statement.
*/
func (this *Explain) Formalize() error {
	return this.stmt.Formalize()
}

/*
Map statement expressions by calling MapExpressions.
*/
func (this *Explain) MapExpressions(mapper expression.Mapper) error {
	return this.stmt.MapExpressions(mapper)
}

/*
Return all contained Expressions.
*/
func (this *Explain) Expressions() expression.Expressions {
	return this.stmt.Expressions()
}

/*
Returns all required privileges.
*/
func (this *Explain) Privileges() (*auth.Privileges, errors.Error) {
	return this.stmt.Privileges()
}

/*
Return the explain statement.
*/
func (this *Explain) Statement() Statement {
	return this.stmt
}

/*
Return the text of the statement being explained
*/
func (this *Explain) Text() string {
	return this.text
}

func (this *Explain) Type() string {
	return "EXPLAIN"
}

func (this *Explain) String() string {
	s := "EXPLAIN " + this.stmt.String()
	return s
}
