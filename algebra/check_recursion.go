// Copyright 2023-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

// mark from expression as recursive
func checkRecursive(alias string, node Node) (bool, error) {
	checkRecur := newCheckRecursion(alias)
	_, err := node.Accept(checkRecur)

	return checkRecur.found, err
}

// Algebra Node visitor
type checkRecursion struct {
	alias string
	found bool
}

func newCheckRecursion(alias string) *checkRecursion {
	return &checkRecursion{
		alias: alias,
	}
}

func (this *checkRecursion) visitJoin(left FromTerm, right FromTerm) error {
	_, err := left.Accept(this)
	if err == nil {
		_, err = right.Accept(this)
	}

	return err
}

func (this *checkRecursion) visitSetop(first Subresult, second Subresult) error {
	_, err := first.Accept(this)
	if err == nil {
		_, err = second.Accept(this)
	}

	return err
}

func (this *checkRecursion) VisitSelectTerm(node *SelectTerm) (interface{}, error) {
	return node.query.subresult.Accept(this)
}

func (this *checkRecursion) VisitSubselect(node *Subselect) (interface{}, error) {
	if node.From() != nil {
		return node.From().Accept(this)
	}
	return nil, nil
}

func (this *checkRecursion) VisitKeyspaceTerm(node *KeyspaceTerm) (interface{}, error) {
	return nil, nil
}

/*
check for recursive cte identifier
*/
func (this *checkRecursion) VisitExpressionTerm(node *ExpressionTerm) (interface{}, error) {
	if expr := node.ExpressionTerm(); expr != nil {
		identExpr, isIdent := expr.(*expression.Identifier)
		if isIdent {
			ident := identExpr.Identifier()
			if this.alias == ident {
				if this.found {
					/* Recursion in N1QL
					is implemented according to Linear Recursion in SQL std
					i.e a RECURSIVE definition can only make 1 one reference to
					itself in FROM CLAUSE
					*/
					return nil, errors.NewMoreThanOneRecursiveRefError(this.alias)
				}
				this.found = true
				return nil, nil
			}
		}
	}
	return nil, nil
}

func (this *checkRecursion) VisitSubqueryTerm(node *SubqueryTerm) (interface{}, error) {
	return nil, nil
}

func (this *checkRecursion) VisitJoin(node *Join) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *checkRecursion) VisitIndexJoin(node *IndexJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *checkRecursion) VisitAnsiJoin(node *AnsiJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *checkRecursion) VisitNest(node *Nest) (interface{}, error) {
	return nil, errors.NewRecurionUnsupportedError("NEST", node.String())
}

func (this *checkRecursion) VisitIndexNest(node *IndexNest) (interface{}, error) {
	return nil, errors.NewRecurionUnsupportedError("NEST", node.String())
}

func (this *checkRecursion) VisitAnsiNest(node *AnsiNest) (interface{}, error) {
	return nil, errors.NewRecurionUnsupportedError("NEST", node.String())
}

func (this *checkRecursion) VisitUnnest(node *Unnest) (interface{}, error) {
	return nil, errors.NewRecurionUnsupportedError("UNNEST", node.String())
}

func (this *checkRecursion) VisitUnion(node *Union) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *checkRecursion) VisitUnionAll(node *UnionAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *checkRecursion) VisitIntersect(node *Intersect) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *checkRecursion) VisitIntersectAll(node *IntersectAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *checkRecursion) VisitExcept(node *Except) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *checkRecursion) VisitExceptAll(node *ExceptAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}
