//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents a prepared statement. Type Prepare is a
struct that contains a statement (json statement).
*/
type Prepare struct {
	statementBase

	name   string    `json:"name"`
	save   bool      `json:"save"`
	force  bool      `json:"force"`
	offset int       `json:"offset"`
	text   string    `json:"text"`
	stmt   Statement `json:"stmt"`
}

/*
The function NewPrepare returns a pointer to the
Prepare struct with the input argument statement
as a field.
*/
func NewPrepare(name string, save, force bool, stmt Statement, text string, offset int) *Prepare {
	rv := &Prepare{
		name:   name,
		save:   save,
		force:  force,
		stmt:   stmt,
		text:   text,
		offset: offset,
	}
	rv.statementBase.stmt = rv
	return rv
}

/*
It calls the VisitPrepare method by passing in the receiver
and returns the interface. It is a visitor pattern.
*/
func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

/*
This method returns the shape of the result, which is
a JSON string value.
*/
func (this *Prepare) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

/*
Call Formalize for the input statement.
*/
func (this *Prepare) Formalize() error {
	return this.stmt.Formalize()
}

/*
Map statement expressions by calling MapExpressions.
*/
func (this *Prepare) MapExpressions(mapper expression.Mapper) error {
	return this.stmt.MapExpressions(mapper)
}

/*
Returns all contained Expressions.
*/
func (this *Prepare) Expressions() expression.Expressions {
	return this.stmt.Expressions()
}

/*
Returns all required privileges.
*/
func (this *Prepare) Privileges() (*auth.Privileges, errors.Error) {
	return this.stmt.Privileges()
}

/*
Return the prepared statement.
*/
func (this *Prepare) Statement() Statement {
	return this.stmt
}

/*
Return the prepared statement name.
*/
func (this *Prepare) Name() string {
	return this.name
}

/*
Return the prepared save reprepare status.
*/
func (this *Prepare) Save() bool {
	return this.save
}

/*
Return the prepared force reprepare status.
*/
func (this *Prepare) Force() bool {
	return this.force
}

/*
Return the prepared text start offset.
*/
func (this *Prepare) Offset() int {
	return this.offset
}

/*
Return the prepared statement text.
*/
func (this *Prepare) Text() string {
	return this.text
}

/*
It's whatever the statement is
*/
func (this *Prepare) Type() string {
	return this.stmt.Type()
}

func (this *Prepare) String() string {
	var s strings.Builder
	s.WriteString("PREPARE ")
	if this.force {
		s.WriteString("FORCE ")
	}
	if this.save {
		s.WriteString("SAVE ")
	}
	if this.name != "" {
		s.WriteRune('`')
		s.WriteString(this.name)
		s.WriteRune('`')
		s.WriteString(" ")
	}

	s.WriteString("AS ")
	s.WriteString(this.stmt.String())
	return s.String()
}
