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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents a subquery statement. It inherits from
ExpressionBase since the result representation of
the subquery is an expression and contains a field
that refers to the select statement to represent
the subquery.
*/
type Subquery struct {
	expression.ExpressionBase
	query *Select
}

/*
The function NewSubquery returns a pointer to the
Subquery struct by assigning the input attributes
to the fields of the struct.
*/
func NewSubquery(query *Select) expression.Expression {
	rv := &Subquery{
		query: query,
	}

	rv.SetExpr(rv)
	return rv
}

/*
   Representation as a N1QL string.
*/
func (this *Subquery) String() string {
	return "(" + this.query.String() + ")"
}

/*
It calls the VisitSubquery method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Subquery) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitSubquery(this)
}

/*
Return a value of type ARRAY. The result of the subquery
is returned as an array.
*/
func (this *Subquery) Type() value.Type { return value.ARRAY }

/*
Call the evaluate method for subqueries and pass in the query and
current item. Call the method using the current context.
*/
func (this *Subquery) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return context.(Context).EvaluateSubquery(this.query, item)
}

/*
Return false. Subquery cannot be used as a secondary
index key.
*/
func (this *Subquery) Indexable() bool {
	return false
}

/*
Return false.
*/
func (this *Subquery) EquivalentTo(other expression.Expression) bool {
	return false
}

/*
Return false.
*/
func (this *Subquery) SubsetOf(other expression.Expression) bool {
	return false
}

/*
Return inner query's Expressions.
*/
func (this *Subquery) Children() expression.Expressions {
	return this.query.Expressions()
}

/*
Map inner query's Expressions.
*/
func (this *Subquery) MapChildren(mapper expression.Mapper) error {
	return this.query.Expressions().MapExpressions(mapper)
}

/*
Return the subquery (receiver) expression.
*/
func (this *Subquery) Copy() expression.Expression {
	return this
}

/*
This method calls FormalizeSubquery to qualify all the children
of the query, and returns an error if any.
*/
func (this *Subquery) Formalize(parent *expression.Formalizer) error {
	return this.query.FormalizeSubquery(parent)
}

/*
Returns the subquery select statement, namely the input
query.
*/
func (this *Subquery) Select() *Select {
	return this.query
}
