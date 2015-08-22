//  Copyright (c) 2014 Couchbase, Inc.
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
	"sort"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	aggs, err := allAggregates(node, this.order)
	if err != nil {
		return nil, err
	}

	this.where = node.Where()

	group := node.Group()
	if group == nil && len(aggs) > 0 {
		group = algebra.NewGroup(nil, nil, nil)
		this.where = constrainAggregate(this.where, aggs)
	}

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams

	count, err := this.fastCount(node)
	if err != nil {
		return nil, err
	}

	if count {
		this.maxParallelism = 1
	} else if node.From() != nil {
		if this.where != nil || group != nil {
			this.limit = nil
		}

		_, err := node.From().Accept(this)
		if err != nil {
			return nil, err
		}
	} else {
		// No FROM clause
		scan := plan.NewDummyScan()
		this.children = append(this.children, scan)
		this.maxParallelism = 1
	}

	if this.coveringScan != nil {
		coverer := expression.NewCoverer(this.coveringScan.Covers())
		err = this.cover.MapExpressions(coverer)
		if err != nil {
			return nil, err
		}

		if this.where != nil {
			this.where, err = coverer.Map(this.where)
			if err != nil {
				return nil, err
			}
		}
	}

	if node.Let() != nil {
		this.subChildren = append(this.subChildren, plan.NewLet(node.Let()))
	}

	if node.Where() != nil {
		this.subChildren = append(this.subChildren, plan.NewFilter(node.Where()))
	}

	if group != nil {
		this.visitGroup(group, aggs)
	}

	projection := node.Projection()
	this.subChildren = append(this.subChildren, plan.NewInitialProject(projection))

	// Initial DISTINCT (parallel)
	if projection.Distinct() || this.distinct {
		this.subChildren = append(this.subChildren, plan.NewDistinct())
	}

	if !this.delayProjection {
		// Perform the final projection if there is no subsequent ORDER BY
		this.subChildren = append(this.subChildren, plan.NewFinalProject())
	}

	// Parallelize the subChildren
	this.children = append(this.children, plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism))

	// Final DISTINCT (serial)
	if projection.Distinct() || this.distinct {
		this.children = append(this.children, plan.NewDistinct())
	}

	// Serialize the top-level children
	return plan.NewSequence(this.children...), nil
}

func (this *builder) visitGroup(group *algebra.Group, aggs map[string]algebra.Aggregate) {
	aggn := make(sort.StringSlice, 0, len(aggs))
	for n, _ := range aggs {
		aggn = append(aggn, n)
	}

	aggn.Sort()
	aggv := make(algebra.Aggregates, len(aggs))
	for i, n := range aggn {
		aggv[i] = aggs[n]
	}

	this.subChildren = append(this.subChildren, plan.NewInitialGroup(group.By(), aggv))
	this.children = append(this.children, plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism))
	this.children = append(this.children, plan.NewIntermediateGroup(group.By(), aggv))
	this.children = append(this.children, plan.NewFinalGroup(group.By(), aggv))
	this.subChildren = make([]plan.Operator, 0, 8)

	letting := group.Letting()
	if letting != nil {
		this.subChildren = append(this.subChildren, plan.NewLet(letting))
	}

	having := group.Having()
	if having != nil {
		this.subChildren = append(this.subChildren, plan.NewFilter(having))
	}
}

func (this *builder) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	node.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return nil, err
	}

	if this.subquery && node.Keys() == nil {
		return nil, errors.NewSubqueryMissingKeysError(node.Keyspace())
	}

	scan, err := this.selectScan(keyspace, node, this.limit)
	if err != nil {
		return nil, err
	}

	this.children = append(this.children, scan)

	if this.coveringScan == nil {
		fetch := plan.NewFetch(keyspace, node)
		this.subChildren = append(this.subChildren, fetch)
	}

	return nil, nil
}

func (this *builder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	sel, err := node.Subquery().Accept(this)
	if err != nil {
		return nil, err
	}

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams
	this.children = append(this.children, sel.(plan.Operator), plan.NewAlias(node.Alias()))
	return nil, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	this.limit = nil

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	join := plan.NewJoin(keyspace, node)
	this.subChildren = append(this.subChildren, join)
	return nil, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	nest := plan.NewNest(keyspace, node)
	this.subChildren = append(this.subChildren, nest)
	return nil, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	this.limit = nil

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	unnest := plan.NewUnnest(node)
	this.subChildren = append(this.subChildren, unnest)
	return nil, nil
}

func allAggregates(node *algebra.Subselect, order *algebra.Order) (map[string]algebra.Aggregate, error) {
	aggs := make(map[string]algebra.Aggregate)

	if node.Let() != nil {
		for _, binding := range node.Let() {
			collectAggregates(aggs, binding.Expression())
			if len(aggs) > 0 {
				return nil, fmt.Errorf("Aggregates not allowed in LET.")
			}
		}
	}

	if node.Where() != nil {
		collectAggregates(aggs, node.Where())
		if len(aggs) > 0 {
			return nil, fmt.Errorf("Aggregates not allowed in WHERE.")
		}
	}

	group := node.Group()
	if group != nil {
		letting := group.Letting()
		for _, binding := range letting {
			collectAggregates(aggs, binding.Expression())
		}

		having := group.Having()
		if having != nil {
			collectAggregates(aggs, having)
		}
	}

	projection := node.Projection()
	if projection != nil {
		for _, term := range projection.Terms() {
			if term.Expression() != nil {
				collectAggregates(aggs, term.Expression())
			}
		}
	}

	if order != nil {
		allow := len(aggs) > 0

		for _, term := range order.Terms() {
			if term.Expression() != nil {
				collectAggregates(aggs, term.Expression())
			}
		}

		if !allow && len(aggs) > 0 {
			return nil, fmt.Errorf("Aggregates not available for this ORDER BY.")
		}
	}

	return aggs, nil
}

func collectAggregates(aggs map[string]algebra.Aggregate, exprs ...expression.Expression) {
	stringer := expression.NewStringer()

	for _, expr := range exprs {
		agg, ok := expr.(algebra.Aggregate)
		if ok {
			str := stringer.Visit(agg)
			aggs[str] = agg
		}

		_, ok = expr.(*algebra.Subquery)
		if !ok {
			children := expr.Children()
			if len(children) > 0 {
				collectAggregates(aggs, children...)
			}
		}
	}
}

func (this *builder) fastCount(node *algebra.Subselect) (bool, error) {
	if node.From() == nil ||
		node.Where() != nil ||
		node.Group() != nil {
		return false, nil
	}

	from, ok := node.From().(*algebra.KeyspaceTerm)
	if !ok || from.Projection() != nil {
		return false, nil
	}

	from.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(from)
	if err != nil {
		return false, err
	}

	for _, term := range node.Projection().Terms() {
		count, ok := term.Expression().(*algebra.Count)
		if !ok || count.Operand() != nil {
			return false, nil
		}
	}

	scan := plan.NewCountScan(keyspace, from)
	this.children = append(this.children, scan)
	return true, nil
}

/*

Constrain the WHERE condition to reflect the aggregate query. For
example:

SELECT AVG(v) FROM widget w;

is rewritten as:

SELECT AVG(v) FROM widget w WHERE v IS NOT NULL;

This enables the query to use an index on v.

*/
func constrainAggregate(cond expression.Expression, aggs map[string]algebra.Aggregate) expression.Expression {
	var first expression.Expression
	for _, agg := range aggs {
		if first == nil {
			first = agg.Operand()
			if first == nil {
				return cond
			}

			continue
		}

		op := agg.Operand()
		if op == nil || !first.EquivalentTo(op) {
			return cond
		}
	}

	if first == nil {
		return cond
	}

	var constraint expression.Expression = expression.NewIsNotNull(first)
	if cond != nil {
		constraint = expression.NewAnd(cond, constraint)
	}

	return constraint
}
