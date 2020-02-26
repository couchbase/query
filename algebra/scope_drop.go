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
Represents the Drop scope ddl statement. Type DropScope is
a struct that contains fields mapping to each clause in the
drop scope statement.
*/
type DropScope struct {
	statementBase

	scope *ScopeRef `json:"scope"`
}

/*
The function NewDropScope returns a pointer to the
DropScope struct with the input argument values as fields.
*/
func NewDropScope(scope *ScopeRef) *DropScope {
	rv := &DropScope{
		scope: scope,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitDropScope method by passing in the
receiver and returns the interface. It is a visitor
pattern.
*/
func (this *DropScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropScope(this)
}

/*
Returns nil.
*/
func (this *DropScope) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *DropScope) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *DropScope) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *DropScope) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *DropScope) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	//	fullName := this.scope.FullName()
	//	privs.Add(fullName, auth.PRIV_QUERY_DROP_SCOPE)
	return privs, nil
}

/*
Returns the scope reference of the scope to be dropped
*/
func (this *DropScope) Scope() *ScopeRef {
	return this.scope
}

/*
Returns the name of the scope to be dropped
*/
func (this *DropScope) Name() string {
	return this.scope.Path().Scope()
}

/*
Marshals input receiver into byte array.
*/
func (this *DropScope) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropScope"}
	r["scopeRef"] = this.scope
	return json.Marshal(r)
}

func (this *DropScope) Type() string {
	return "DROP_SCOPE"
}
