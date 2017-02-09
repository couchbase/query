//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/datastore"
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

	name string    `json:"name"`
	text string    `json:"text"`
	stmt Statement `json:"stmt"`
}

/*
The function NewPrepare returns a pointer to the
Prepare struct with the input argument statement
as a field.
*/
func NewPrepare(name string, stmt Statement, text string) *Prepare {
	rv := &Prepare{
		name: name,
		stmt: stmt,
		text: text,
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
func (this *Prepare) Privileges() (*datastore.Privileges, errors.Error) {
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
Return the prepared statement text.
*/
func (this *Prepare) Text() string {
	return this.text
}
