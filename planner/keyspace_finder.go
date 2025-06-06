//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

// gather keyspace references in a FROM clause (by walking the algebra AST tree)
type keyspaceFinder struct {
	baseKeyspaces    map[string]*base.BaseKeyspace
	keyspaceMap      map[string]string
	outerlevel       int32
	pushableOnclause expression.Expression
	unnestDepends    map[string]*expression.Identifier
	metadataDuration time.Duration
	arrayId          int
	useCBO           bool
}

func newKeyspaceFinder(baseKeyspaces map[string]*base.BaseKeyspace, primary string, arrayId int, useCBO bool) *keyspaceFinder {
	rv := &keyspaceFinder{
		baseKeyspaces: baseKeyspaces,
		arrayId:       arrayId,
		useCBO:        useCBO,
	}
	rv.keyspaceMap = make(map[string]string, len(baseKeyspaces))
	rv.unnestDepends = make(map[string]*expression.Identifier, len(baseKeyspaces))
	rv.unnestDepends[primary] = expression.NewIdentifier(primary)
	return rv
}

func (this *keyspaceFinder) addKeyspaceAlias(alias string, path *algebra.Path,
	node algebra.SimpleFromTerm) error {
	if _, ok := this.baseKeyspaces[alias]; ok {
		return errors.NewPlanInternalError(fmt.Sprintf("addKeyspaceAlias: duplicate keyspace %s", alias))
	}
	newBaseKeyspace, duration := base.NewBaseKeyspace(alias, path, node, (1 << len(this.baseKeyspaces)))
	this.metadataDuration += duration
	newBaseKeyspace.SetOuterlevel(this.outerlevel)
	this.baseKeyspaces[alias] = newBaseKeyspace
	keyspace := newBaseKeyspace.Keyspace()
	this.keyspaceMap[alias] = keyspace
	if keyspace != "" {
		newBaseKeyspace.SetDocCount(optDocCount(keyspace, this.useCBO))
		newBaseKeyspace.SetHasDocCount()
	}
	return nil
}

func (this *keyspaceFinder) addOnclause(onclause expression.Expression) bool {
	// add onclause if it does not reference any previous outer tables
	if onclause != nil && pushableOnclause(onclause, this.baseKeyspaces, this.keyspaceMap) {
		if this.pushableOnclause != nil {
			this.pushableOnclause = expression.NewAnd(this.pushableOnclause, onclause)
		} else {
			this.pushableOnclause = onclause
		}
		return true
	}

	return false
}

func (this *keyspaceFinder) visitJoin(left algebra.FromTerm, right algebra.SimpleFromTerm, outer bool) error {
	outerlevel := this.outerlevel
	defer func() { this.outerlevel = outerlevel }()

	_, err := left.Accept(this)
	if err != nil {
		return err
	}

	if outer {
		this.outerlevel++
	}

	_, err = right.Accept(this)
	if err != nil {
		return err
	}

	return nil
}

func (this *keyspaceFinder) visitSetop(first algebra.Subresult, second algebra.Subresult) error {
	// keyspaceFinder is initialized at FROM clause processing, i.e., for each statement,
	// and thus we don't expect it'll reach any of the set operations node
	return errors.NewPlanInternalError("keyspaceFinder.visitSetop: unexpected code path")
}

func (this *keyspaceFinder) VisitSelectTerm(node *algebra.SelectTerm) (interface{}, error) {
	return nil, errors.NewPlanInternalError("keyspaceFinder.visitSelectTerm: unexpected code path")
}

func (this *keyspaceFinder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	if node.From() != nil {
		return node.From().Accept(this)
	}
	return nil, nil
}

func (this *keyspaceFinder) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	return nil, this.addKeyspaceAlias(node.Alias(), node.Path(), node)
}

func (this *keyspaceFinder) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}

	return nil, this.addKeyspaceAlias(node.Alias(), nil, node)
}

func (this *keyspaceFinder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	return nil, this.addKeyspaceAlias(node.Alias(), nil, node)
}

func (this *keyspaceFinder) VisitJoin(node *algebra.Join) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right(), false)
}

func (this *keyspaceFinder) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right(), false)
}

func (this *keyspaceFinder) VisitAnsiJoin(node *algebra.AnsiJoin) (interface{}, error) {
	err := this.visitJoin(node.Left(), node.Right(), node.Outer())

	onclause := node.Onclause()

	this.arrayId, err = expression.AssignArrayId(onclause, this.arrayId)
	if err != nil {
		return nil, err
	}

	// if this is inner join, gather ON-clause
	if !node.Outer() {
		node.SetPushable(this.addOnclause(onclause))
	}

	return nil, err
}

func (this *keyspaceFinder) VisitNest(node *algebra.Nest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right(), false)
}

func (this *keyspaceFinder) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right(), false)
}

func (this *keyspaceFinder) VisitAnsiNest(node *algebra.AnsiNest) (interface{}, error) {
	err := this.visitJoin(node.Left(), node.Right(), node.Outer())

	onclause := node.Onclause()

	this.arrayId, err = expression.AssignArrayId(onclause, this.arrayId)
	if err != nil {
		return nil, err
	}

	// if this is inner nest, gather ON-clause
	if !node.Outer() {
		node.SetPushable(this.addOnclause(onclause))
	}

	return nil, err
}

func (this *keyspaceFinder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	alias := node.Alias()

	err = this.addKeyspaceAlias(alias, nil, nil)
	if err != nil {
		return nil, err
	}

	ks, _ := this.baseKeyspaces[alias]
	ks.SetUnnest()
	if node.Outer() {
		ks.SetOuterlevel(this.outerlevel + 1)
	} else {
		for _, unnest := range this.unnestDepends {
			if node.Expression().DependsOn(unnest) {
				this.unnestDepends[alias] = expression.NewIdentifier(alias)
				break
			}
		}
	}

	return nil, nil
}

func (this *keyspaceFinder) VisitUnion(node *algebra.Union) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *keyspaceFinder) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *keyspaceFinder) VisitIntersect(node *algebra.Intersect) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *keyspaceFinder) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *keyspaceFinder) VisitExcept(node *algebra.Except) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *keyspaceFinder) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func pushableOnclause(onclause expression.Expression, baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string) bool {

	keyspaces, err := expression.CountKeySpaces(onclause, keyspaceNames)
	if err != nil {
		return false
	}

	chkNullRej := newChkNullRej()
	for ks, _ := range keyspaces {
		baseKspace, ok := baseKeyspaces[ks]
		if !ok {
			return false
		} else if baseKspace.Outerlevel() > 0 {
			if !nullRejExpr(chkNullRej, ks, onclause) {
				return false
			}
		}
	}

	return true
}
