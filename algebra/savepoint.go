//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the savepoint statement.
*/
type Savepoint struct {
	statementBase

	savepoint string `json:"savepoint"`
}

/*
The function NewSavepoint returns a pointer to the SAVEPOINT <name> statement
struct by assigning the input attributes to the fields of the struct
*/
func NewSavepoint(name string) *Savepoint {
	rv := &Savepoint{
		savepoint: name,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitSavepoint method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Savepoint) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSavepoint(this)
}

/*
The shape of the savepoint statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *Savepoint) Signature() value.Value {
	return _JSON_SIGNATURE
}

/*
It's set savepoint
*/
func (this *Savepoint) Type() string {
	return "SAVEPOINT"
}

/*
Applies mapper to all the expressions in the savepoint statement.
*/
func (this *Savepoint) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Returns all contained Expressions.
*/
func (this *Savepoint) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *Savepoint) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_QUERY_TRANSACTION_STMT, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the savepoint statement.
*/
func (this *Savepoint) Formalize() (err error) {
	return
}

/*
Returns the Savepoint name
*/
func (this *Savepoint) Savepoint() string {
	return this.savepoint
}
