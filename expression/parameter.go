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
