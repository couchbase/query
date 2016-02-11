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
Represents the explain text for a query. Type Explain is
a struct that represents the explain json statement.
*/
type Explain struct {
	statementBase

	stmt Statement `json:"stmt"`
}

/*
The function NewExplain returns a pointer to the Explain
struct that has its field stmt set to the input Statement.
*/
func NewExplain(stmt Statement) *Explain {
	rv := &Explain{
		stmt: stmt,
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
func (this *Explain) Privileges() (datastore.Privileges, errors.Error) {
	return nil, nil
}

/*
Return the explain statement.
*/
func (this *Explain) Statement() Statement {
	return this.stmt
}
