//  Copyright (c) 2019 Couchbase, Inc.
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
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

/*
Represents the Drop function ddl statement. Type DropFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields just refer to the function name.
*/
type DropFunction struct {
	statementBase

	name functions.FunctionName `json:"name"`
}

/*
The function NewDropFunction returns a pointer to the
DropFunction struct with the input argument values as fields.
*/
func NewDropFunction(name functions.FunctionName) *DropFunction {
	rv := &DropFunction{
		name: name,
	}

	rv.stmt = rv
	return rv
}

func (this *DropFunction) Name() functions.FunctionName {
	return this.name
}

/*
It calls the VisitDropFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *DropFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropFunction(this)
}

/*
Returns nil.
*/
func (this *DropFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *DropFunction) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses, but here none have expressions
*/
func (this *DropFunction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Return expr from the create function statement.
*/
func (this *DropFunction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *DropFunction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	//	fullName := this.name.Key()
	//	privs.Add(fullName, auth.PRIV_QUERY_DROP_FUNCTION)

	return privs, nil
}

func (this *DropFunction) Type() string {
	return "DROP_FUNCTION"
}
