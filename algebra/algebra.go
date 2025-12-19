//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package algebra provides a syntax-independent algebra. Any language
flavor or syntax that can be converted to this algebra can then be
processed by the query engine.
*/
package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
The Statement interface represents a N1QL statement, e.g. a SELECT,
UPDATE, or CREATE INDEX statement.
*/
type Statement interface {
	/*
		Visitor pattern.
	*/
	Accept(visitor Visitor) (interface{}, error)

	/*
		The shape of this statement's return values.
	*/
	Signature() value.Value

	/*
		Fully qualify all identifiers in this statement.
	*/
	Formalize() error

	/*
		Apply a Mapper to all the expressions in this statement.
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
		Returns the statement type, for accounting and other purposes
	*/
	Type() string

	/*
		Sets the parameter count, for AutoPrepare and other purposes
	*/
	SetParamsCount(params int)

	/*
		Returns the parameter count, for AutoPrepare and other purposes
	*/
	Params() int

	/*
		Returns the optimizer hints
	*/
	OptimHints() *OptimHints

	/*
		Returns the statement subqueries
	*/
	Subqueries() ([]*Subquery, errors.Error)

	/*
		Returns the string representation of the statement
	*/
	String() string
}

/*
The Node interface represents a node in the algebra tree (AST). It is
used internally within the algebra package for polymorphism and
visitor pattern.
*/
type Node interface {
	/*
	   Visitor pattern.
	*/
	Accept(visitor NodeVisitor) (interface{}, error)
}
