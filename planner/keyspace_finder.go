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
)

// gather keyspace references in a FROM clause (by walking the algebra AST tree)
type keyspaceFinder struct {
	baseKeyspaces map[string]*baseKeyspace
}

func newKeyspaceFinder(baseKeyspaces map[string]*baseKeyspace) *keyspaceFinder {
	rv := &keyspaceFinder{
		baseKeyspaces: baseKeyspaces,
	}
	return rv
}

func (this *keyspaceFinder) addKeyspaceAlias(alias string) error {
	if _, ok := this.baseKeyspaces[alias]; ok {
		return errors.NewPlanInternalError(fmt.Sprintf("addKeyspaceAlias: duplicate keyspace %s", alias))
	}
	newBaseKeyspace := newBaseKeyspace(alias)
	this.baseKeyspaces[alias] = newBaseKeyspace
	return nil
}

func (this *keyspaceFinder) visitJoin(left algebra.FromTerm, right algebra.FromTerm) error {
	_, err := left.Accept(this)
	if err != nil {
		return err
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
	return nil, this.addKeyspaceAlias(node.Alias())
}

func (this *keyspaceFinder) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}

	return nil, this.addKeyspaceAlias(node.Alias())
}

func (this *keyspaceFinder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	return nil, this.addKeyspaceAlias(node.Alias())
}

func (this *keyspaceFinder) VisitJoin(node *algebra.Join) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *keyspaceFinder) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *keyspaceFinder) VisitAnsiJoin(node *algebra.AnsiJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *keyspaceFinder) VisitAnsiNest(node *algebra.AnsiNest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *keyspaceFinder) VisitNest(node *algebra.Nest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *keyspaceFinder) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *keyspaceFinder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, this.addKeyspaceAlias(node.Alias())
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
