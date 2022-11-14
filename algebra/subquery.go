//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
func NewSubquery(query *Select) *Subquery {
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
	var s string
	if this.IsCorrelated() || this.query.subresult.IsCorrelated() {
		s += "correlated "
	}

	return s + "(" + this.query.String() + ")"
}

/*
Visitor pattern.
*/
func (this *Subquery) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitSubquery(this)
}

/*
Subqueries return a value of type ARRAY.
*/
func (this *Subquery) Type() value.Type { return value.ARRAY }

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
func (this *Subquery) IndexAggregatable() bool {
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
	return this.query.MapExpressions(mapper)
}

/*
Return this subquery expression.
*/
func (this *Subquery) Copy() expression.Expression {
	return this
}

/*
TODO: This is overly broad. Ideally, we would allow:

SELECT g, (SELECT d2.* FROM d2 USE KEYS d.g) AS d2
FROM d
GROUP BY g;

but not allow:

SELECT g, (SELECT d2.* FROM d2 USE KEYS d.foo) AS d2
FROM d
GROUP BY g;
*/
func (this *Subquery) SurvivesGrouping(groupKeys expression.Expressions, allowed *value.ScopeValue) (
	bool, expression.Expression) {
	return !this.query.IsCorrelated(), nil
}

/*
This method calls FormalizeSubquery to qualify all the children
of the query, and returns an error if any.
*/
func (this *Subquery) Formalize(parent *expression.Formalizer) error {
	var keyspace string
	if parent != nil {
		keyspace = parent.Keyspace()
	}
	f := expression.NewFormalizer(keyspace, parent)
	return this.query.FormalizeSubquery(f)
}

/*
Returns the subquery select statement, namely the input
query.
*/
func (this *Subquery) Select() *Select {
	return this.query
}

func (this *Subquery) IsCorrelated() bool {
	return this.query.IsCorrelated()
}

func (this *Subquery) GetCorrelation() map[string]bool {
	return this.query.GetCorrelation()
}
