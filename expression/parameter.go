//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
Type NamedParameter is an interface that inherits from
Expression, and has a method Name() that returns a
string. It defines a Named Parameter, that is specified
using formal param names in a query. The main advantage
of a named parameter is that we dont have to remember
the position of the parameter.
*/
type NamedParameter interface {
	Expression
	Name() string
}

/*
Type PositionalParameter is an interface that inherits
from Expression, and has a method position that returns
an integer representing the position of the parameter
in the query.
*/
type PositionalParameter interface {
	Expression
	Position() int
}

/*
ParamReplacer is a Mapper that replaces all named and positional
parameters in an expression with the NULL constant. This can be used
to check properties of an expression (e.g. Indexable()) that do not
depend on the actual parameter values, since parameters are replaced
with values before evaluation (in places such as inline filters sent
to the indexer). Note the children of the expression are mapped in
place, callers should pass in a copy of the expression if the original
needs to be preserved, and use the returned expression (a top-level
parameter is replaced by a new expression).
*/
type ParamReplacer struct {
	MapperBase
}

func NewParamReplacer() *ParamReplacer {
	rv := &ParamReplacer{}
	rv.SetMapper(rv)
	return rv
}

func (this *ParamReplacer) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return NULL_EXPR, nil
}

func (this *ParamReplacer) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return NULL_EXPR, nil
}
