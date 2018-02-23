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
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/*
Represents the from clause in a select statement.
*/
type FromTerm interface {
	/*
	   Represents the Node interface.
	*/
	Node

	/*
	   Apply a Mapper to all the expressions in this statement
	*/
	MapExpressions(mapper expression.Mapper) error

	/*
	   Returns all contained Expressions.
	*/
	Expressions() expression.Expressions

	/*
	   Returns all required privileges.
	*/
	Privileges() (*auth.Privileges, errors.Error)

	/*
	   Representation as a N1QL string.
	*/
	String() string

	/*
	   Qualify all identifiers for the parent expression.
	*/
	Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error)

	/*
	   Represents the primary (first) term of this FROM term.
	*/
	PrimaryTerm() FromTerm

	/*
	   Represents alias string.
	*/
	Alias() string
}

type JoinTerm interface {
	FromTerm
	Left() FromTerm
	Right() *KeyspaceTerm
	Outer() bool
}

func GetKeyspaceTerm(term FromTerm) *KeyspaceTerm {
	if term == nil {
		return nil
	}

	switch term := term.(type) {
	case *KeyspaceTerm:
		return term
	case *ExpressionTerm:
		if term.IsKeyspace() {
			return term.KeyspaceTerm()
		}
		return nil
	default:
		return nil
	}
}
