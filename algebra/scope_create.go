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
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the Create scope ddl statement. Type CreateScope is
a struct that contains fields mapping to each clause in the
create scope statement.
*/
type CreateScope struct {
	statementBase

	scope *ScopeRef `json:"scope"`
}

/*
The function NewCreateScope returns a pointer to the
CreateScope struct with the input argument values as fields.
*/
func NewCreateScope(scope *ScopeRef) *CreateScope {
	rv := &CreateScope{
		scope: scope,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitCreateScope method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateScope(this)
}

/*
Returns nil.
*/
func (this *CreateScope) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateScope) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses.
*/
func (this *CreateScope) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Return expr from the create scope statement.
*/
func (this *CreateScope) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreateScope) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	//	fullName := this.scope.FullName()
	//	privs.Add(fullName, auth.PRIV_QUERY_CREATE_SCOPE)

	return privs, nil
}

/*
Returns the scope reference of the scope to be created
*/
func (this *CreateScope) Scope() *ScopeRef {
	return this.scope
}

/*
Returns the name of the scope to be created
*/
func (this *CreateScope) Name() string {
	return this.scope.Path().Scope()
}

/*
Marshals input receiver into byte array.
*/
func (this *CreateScope) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createScope"}
	r["scopeRef"] = this.scope
	return json.Marshal(r)
}

func (this *CreateScope) Type() string {
	return "CREATE_SCOPE"
}
