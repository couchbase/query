//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
SelectTerm wraps a Select and implements the Node interface. It
enables Select to be a term in a UNION, INTERSECT, or EXCEPT.
*/
type SelectTerm struct {
	query *Select `json:"query"`
}

/*
Constructor.
*/
func NewSelectTerm(term *Select) *SelectTerm {
	return &SelectTerm{term}
}

/*
Visitor pattern.
*/
func (this *SelectTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitSelectTerm(this)
}

/*
This method returns the shape of the select clause. It returns a value
that represents the signature of the projection.
*/
func (this *SelectTerm) Signature() value.Value {
	return this.query.Signature()
}

/*
 */
func (this *SelectTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	err = this.query.FormalizeSubquery(parent, this.query.setop)
	if err != nil {
		return nil, err
	}

	return expression.NewFormalizer("", nil), nil
}

/*
This method maps all the constituent clauses, namely the from,
let, where, group by and projection(select) within a SelectTerm
statement.
*/
func (this *SelectTerm) MapExpressions(mapper expression.Mapper) (err error) {
	return this.query.MapExpressions(mapper)
}

/*
   Returns all contained Expressions.
*/
func (this *SelectTerm) Expressions() expression.Expressions {
	return this.query.Expressions()
}

/*
   Result terms.
*/
func (this *SelectTerm) ResultTerms() ResultTerms {
	return this.query.Subresult().ResultTerms()
}

/*
   Raw projection.
*/
func (this *SelectTerm) Raw() bool {
	return this.query.Subresult().Raw()
}

/*
Returns all required privileges.
*/
func (this *SelectTerm) Privileges() (*auth.Privileges, errors.Error) {
	return this.query.Privileges()
}

/*
   Representation as a N1QL string.
*/
func (this *SelectTerm) String() string {
	return "(" + this.query.String() + ")"
}

/*
Returns bool value that depicts if query is correlated
or not.
*/
func (this *SelectTerm) IsCorrelated() bool {
	return this.query.IsCorrelated()
}

func (this *SelectTerm) GetCorrelation() map[string]uint32 {
	return this.query.GetCorrelation()
}

/*
Accessor.
*/
func (this *SelectTerm) Select() *Select {
	return this.query
}

func (this *SelectTerm) OptimHints() *OptimHints {
	return this.query.OptimHints()
}
