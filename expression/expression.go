//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package expression provides expression evaluation for query and
indexing.

*/
package expression

import (
	"github.com/couchbaselabs/query/value"
)

type Expressions []Expression
type CompositeExpression []Expression
type CompositeExpressions []CompositeExpression

type Expression interface {
	Evaluate(item value.Value, context Context) (value.Value, error)

	// Is this expression equivalent to the other
	EquivalentTo(other Expression) bool

	// A list of other Expressions on which this depends
	Dependencies() Expressions

	// Terminal identifier if this is a path; else nil
	Alias() string

	// Constant and other folding
	Fold() Expression

	// Formal notation
	Formalize()

	// Is this expression a subset of the other
	// E.g. A < 5 is a subset of A < 10
	SubsetOf(other Expression) bool

	// Index spans to satisfy this expression, or nil
	Spans(index Index) Spans
}
