//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

// Perform ANSI OUTER JOIN to ANSI INNER JOIN transforation on all nodes in a FROM clause (by walking the algebra AST tree)
type ansijoinOuterToInner struct {
	baseKeyspaces    map[string]*base.BaseKeyspace
	keyspaceNames    map[string]string
	unnests          []*algebra.Unnest
	pushableOnclause expression.Expression
}

func newAnsijoinOuterToInner(baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string, unnests []*algebra.Unnest) *ansijoinOuterToInner {
	return &ansijoinOuterToInner{
		baseKeyspaces: baseKeyspaces,
		keyspaceNames: keyspaceNames,
		unnests:       unnests,
	}
}

func (this *ansijoinOuterToInner) addOnclause(onclause expression.Expression) bool {
	if onclause != nil && pushableOnclause(onclause, this.baseKeyspaces, this.keyspaceNames) {
		if this.pushableOnclause != nil {
			this.pushableOnclause = expression.NewAnd(this.pushableOnclause, onclause)
		} else {
			this.pushableOnclause = onclause
		}
		return true
	}

	return false
}

func (this *ansijoinOuterToInner) decrementOuterlevel(alias string) {
	if baseKeyspace, ok := this.baseKeyspaces[alias]; ok {
		baseKeyspace.SetOuterlevel(baseKeyspace.Outerlevel() - 1)
	}
}

func (this *ansijoinOuterToInner) visitAnsiJoin(left algebra.FromTerm, outer bool,
	alias string) (bool, error) {
	_, err := left.Accept(this)
	if err != nil {
		return false, err
	}
	// right-hand side is currently simple term only so no need to traverse

	if !outer {
		return false, nil
	}

	aliasIdent := expression.NewIdentifier(alias)
	for _, unnest := range this.unnests {
		if unnest.Expression().DependsOn(aliasIdent) {
			// if an inner unnest expression includes reference to alias,
			// then alias cannot be outer
			return true, nil
		}
	}

	chkNullRej := newChkNullRej()

	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return false, errors.NewPlanInternalError(fmt.Sprintf("ansijoinOuterToInner: missing baseKeyspace for %s", alias))
	}

	// the filters and joinfilters attached to each keyspace at this point
	// are from either WHERE clause or pushable ON clauses
	for _, fl := range baseKeyspace.Filters() {
		if nullRejExpr(chkNullRej, alias, fl.FltrExpr()) {
			return true, nil
		}
	}

	for _, jfl := range baseKeyspace.JoinFilters() {
		if nullRejExpr(chkNullRej, alias, jfl.FltrExpr()) {
			return true, nil
		}
	}

	return false, nil
}

func (this *ansijoinOuterToInner) visitSetop(first algebra.Subresult, second algebra.Subresult) error {
	// ansijoinOuterToInner is initialized at FROM clause processing, i.e., for each statement,
	// and thus we don't expect it'll reach any of the set operations node
	return errors.NewPlanInternalError("ansijoinOuterToInner.visitSetop: unexpected code path")
}

func (this *ansijoinOuterToInner) VisitSelectTerm(node *algebra.SelectTerm) (interface{}, error) {
	return nil, errors.NewPlanInternalError("ansijoinOuterToInner.VisitSelectTerm: unexpected code path")
}

func (this *ansijoinOuterToInner) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	if node.From() != nil {
		return node.From().Accept(this)
	}
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitJoin(node *algebra.Join) (interface{}, error) {
	// no mixing of lookup join and ANSI JOIN/NEST
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	// no mixing of index join and ANSI JOIN/NEST
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitAnsiJoin(node *algebra.AnsiJoin) (interface{}, error) {
	aoj2aij, err := this.visitAnsiJoin(node.Left(), node.Outer(), node.Alias())
	if err != nil {
		return nil, err
	}
	if aoj2aij {
		node.SetPushable(this.addOnclause(node.Onclause()))
		node.SetOuter(false)
		this.decrementOuterlevel(node.Alias())
	}
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitNest(node *algebra.Nest) (interface{}, error) {
	// no mixing of lookup nest and ANSI JOIN/NEST
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	// no mixing of index nest and ANSI JOIN/NEST
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitAnsiNest(node *algebra.AnsiNest) (interface{}, error) {
	aoj2aij, err := this.visitAnsiJoin(node.Left(), node.Outer(), node.Alias())
	if err != nil {
		return nil, err
	}
	if aoj2aij {
		node.SetPushable(this.addOnclause(node.Onclause()))
		node.SetOuter(false)
		this.decrementOuterlevel(node.Alias())
	}
	return nil, nil
}

func (this *ansijoinOuterToInner) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	return node.Left().Accept(this)
}

func (this *ansijoinOuterToInner) VisitUnion(node *algebra.Union) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *ansijoinOuterToInner) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *ansijoinOuterToInner) VisitIntersect(node *algebra.Intersect) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *ansijoinOuterToInner) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *ansijoinOuterToInner) VisitExcept(node *algebra.Except) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *ansijoinOuterToInner) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}
