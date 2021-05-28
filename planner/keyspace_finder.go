//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

// gather keyspace references in a FROM clause (by walking the algebra AST tree)
type keyspaceFinder struct {
	baseKeyspaces    map[string]*base.BaseKeyspace
	outerlevel       int32
	pushableOnclause expression.Expression
	unnestDepends    map[string]*expression.Identifier
}

func newKeyspaceFinder(baseKeyspaces map[string]*base.BaseKeyspace, primary string) *keyspaceFinder {
	rv := &keyspaceFinder{
		baseKeyspaces: baseKeyspaces,
	}
	rv.unnestDepends = make(map[string]*expression.Identifier, len(baseKeyspaces))
	rv.unnestDepends[primary] = expression.NewIdentifier(primary)
	return rv
}

func (this *keyspaceFinder) addKeyspaceAlias(alias, keyspace string) error {
	if _, ok := this.baseKeyspaces[alias]; ok {
		return errors.NewPlanInternalError(fmt.Sprintf("addKeyspaceAlias: duplicate keyspace %s", alias))
	}
	newBaseKeyspace := base.NewBaseKeyspace(alias, keyspace)
	newBaseKeyspace.SetOuterlevel(this.outerlevel)
	this.baseKeyspaces[alias] = newBaseKeyspace
	return nil
}

func (this *keyspaceFinder) addOnclause(onclause expression.Expression) {
	if onclause != nil {
		if this.pushableOnclause != nil {
			this.pushableOnclause = expression.NewAnd(this.pushableOnclause, onclause)
		} else {
			this.pushableOnclause = onclause
		}
	}
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
	return nil, this.addKeyspaceAlias(node.Alias(), node.Keyspace())
}

func (this *keyspaceFinder) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}

	return nil, this.addKeyspaceAlias(node.Alias(), "")
}

func (this *keyspaceFinder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	return nil, this.addKeyspaceAlias(node.Alias(), "")
}

func (this *keyspaceFinder) VisitJoin(node *algebra.Join) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right(), false)
}

func (this *keyspaceFinder) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right(), false)
}

func (this *keyspaceFinder) VisitAnsiJoin(node *algebra.AnsiJoin) (interface{}, error) {
	err := this.visitJoin(node.Left(), node.Right(), node.Outer())

	// if this is inner join, gather ON-clause
	if !node.Outer() {
		this.addOnclause(node.Onclause())
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

	// if this is inner nest, gather ON-clause
	if !node.Outer() {
		this.addOnclause(node.Onclause())
	}

	return nil, err
}

func (this *keyspaceFinder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	err = this.addKeyspaceAlias(node.Alias(), "")
	if err != nil {
		return nil, err
	}

	if !node.Outer() {
		for _, unnest := range this.unnestDepends {
			if node.Expression().DependsOn(unnest) {
				ks, _ := this.baseKeyspaces[node.Alias()]
				ks.SetPrimaryUnnest()
				this.unnestDepends[node.Alias()] = expression.NewIdentifier(node.Alias())
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
