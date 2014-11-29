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
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Subquery struct {
	expression.ExpressionBase
	query *Select
}

func NewSubquery(query *Select) expression.Expression {
	rv := &Subquery{
		query: query,
	}

	rv.SetExpr(rv)
	return rv
}

func (this *Subquery) String() string {
	// FIXME
	return "(select 1)"
}

func (this *Subquery) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitSubquery(this)
}

func (this *Subquery) Type() value.Type { return value.ARRAY }

func (this *Subquery) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return context.(Context).EvaluateSubquery(this.query, item)
}

func (this *Subquery) Indexable() bool {
	return false
}

func (this *Subquery) EquivalentTo(other expression.Expression) bool {
	return false
}

func (this *Subquery) SubsetOf(other expression.Expression) bool {
	return false
}

func (this *Subquery) Children() expression.Expressions {
	return nil
}

func (this *Subquery) MapChildren(mapper expression.Mapper) error {
	return nil
}

func (this *Subquery) Formalize(parent *expression.Formalizer) error {
	return this.query.FormalizeSubquery(parent)
}

func (this *Subquery) Select() *Select {
	return this.query
}
