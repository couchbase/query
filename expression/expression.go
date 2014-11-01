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
type CompositeExpressions []Expressions

type Expression interface {
	// Visitor pattern
	Accept(visitor Visitor) (interface{}, error)

	// Data type. In addition to this data type, an expression may
	// also evaluate to NULL or MISSING.
	Type() value.Type

	// Evaluate this expression for the given value and context.
	Evaluate(item value.Value, context Context) (value.Value, error)

	// Terminal identifier if this expression is a path; else "".
	Alias() string

	// Is this expression usable as a secondary index key.
	Indexable() bool

	// Is this expression equivalent to the other.
	EquivalentTo(other Expression) bool

	// Is this expression a subset of the other.
	// E.g. A < 5 is a subset of A < 10.
	SubsetOf(other Expression) bool

	// Utility
	Children() Expressions
	MapChildren(mapper Mapper) error
}

func (this Expressions) MapExpressions(mapper Mapper) (err error) {
	for i, e := range this {
		expr, err := mapper.Map(e)
		if err != nil {
			return err
		}

		this[i] = expr
	}

	return
}
