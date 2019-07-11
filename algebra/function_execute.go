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
Represents the Execute function ddl statement. Type ExecuteFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields refer to the function name
and argument expression list
*/
type ExecuteFunction struct {
	statementBase

	name  functions.FunctionName `json:"name"`
	exprs expression.Expressions `json:"expressions"`
}

/*
The function NewExecuteFunction returns a pointer to the
ExecuteFunction struct with the input argument values as fields.
*/
func NewExecuteFunction(name functions.FunctionName, exprs expression.Expressions) *ExecuteFunction {
	rv := &ExecuteFunction{
		name:  name,
		exprs: exprs,
	}

	rv.stmt = rv
	return rv
}

func (this *ExecuteFunction) Name() functions.FunctionName {
	return this.name
}

/*
It calls the VisitExecuteFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *ExecuteFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecuteFunction(this)
}

/*
Returns nil.
*/
func (this *ExecuteFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *ExecuteFunction) Formalize() error {
	return this.MapExpressions(expression.NewFormalizer("", nil))
}

/*
This method maps all the constituent clauses, namely the argument expression list
*/
func (this *ExecuteFunction) MapExpressions(mapper expression.Mapper) (err error) {
	if len(this.exprs) > 0 {
		err = this.exprs.MapExpressions(mapper)
		if err != nil {
			return
		}
	}
	return
}

/*
Return expr from the create function statement.
*/
func (this *ExecuteFunction) Expressions() expression.Expressions {
	return this.exprs
}

/*
Returns all required privileges.
*/
func (this *ExecuteFunction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	//	fullName := this.name.Key()
	//	privs.Add(fullName, auth.PRIV_QUERY_EXECUTE_FUNCTION)

	return privs, nil
}

func (this *ExecuteFunction) Type() string {
	return "EXECUTE_FUNCTION"
}
