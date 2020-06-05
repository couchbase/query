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
Represents the Create function ddl statement. Type CreateFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields refer to the function name and
and function body
*/
type CreateFunction struct {
	statementBase

	name functions.FunctionName `json:"name"`
	body functions.FunctionBody `json:"keys"`
}

/*
The function NewCreateFunction returns a pointer to the
CreateFunction struct with the input argument values as fields.
*/
func NewCreateFunction(name functions.FunctionName, body functions.FunctionBody) *CreateFunction {
	rv := &CreateFunction{
		name: name,
		body: body,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateFunction) Name() functions.FunctionName {
	return this.name
}

func (this *CreateFunction) Body() functions.FunctionBody {
	return this.body
}

/*
It calls the VisitCreateFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateFunction(this)
}

/*
Returns nil.
*/
func (this *CreateFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateFunction) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses, but here non have expressions
*/
func (this *CreateFunction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Return expr from the create function statement.
*/
func (this *CreateFunction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreateFunction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	priv := functions.GetPrivilege(this.name, this.body)
	privs.Add(this.name.Key(), priv)

	return privs, nil
}

func (this *CreateFunction) Type() string {
	return "CREATE_FUNCTION"
}
