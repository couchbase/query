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
	this.where = node.Where()
	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams

	count, err := this.fastCount(node)
	if err != nil {
		return nil, err
	}

	if count {
		this.maxParallelism = 1
	} else if node.From() != nil {
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

	aggs := make(map[string]algebra.Aggregate)

	if node.Let() != nil {
		for _, binding := range node.Let() {
			collectAggregates(aggs, binding.Expression())
			if len(aggs) > 0 {
				return nil, fmt.Errorf("Aggregates not allowed in LET.")
			}
		}

		this.subChildren = append(this.subChildren, plan.NewLet(node.Let()))
	}

	if node.Where() != nil {
		collectAggregates(aggs, node.Where())
		if len(aggs) > 0 {
			return nil, fmt.Errorf("Aggregates not allowed in WHERE.")
		}

		this.subChildren = append(this.subChildren, plan.NewFilter(node.Where()))
	}

	// Check for aggregates
	projection := node.Projection()
	if projection != nil {
		for _, term := range projection.Terms() {
			if term.Expression() != nil {
				collectAggregates(aggs, term.Expression())
			}
		}
	}

	group := node.Group()
	if this.order != nil && (group != nil || len(aggs) > 0) {
		// Grouping -- include aggregates from ORDER BY
		for _, term := range this.order.Terms() {
			if term.Expression() != nil {
				collectAggregates(aggs, term.Expression())
			}
		}
	} else {
		// Not grouping -- disallow aggregates in ORDER BY
		this.order = nil
	}

	if group == nil && len(aggs) > 0 {
		group = algebra.NewGroup(nil, nil, nil)
	}

	if group != nil {
		this.visitGroup(group, aggs)
	}

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
	letting := group.Letting()
	for _, binding := range letting {
		collectAggregates(aggs, binding.Expression())
	}

	having := group.Having()
	if having != nil {
		collectAggregates(aggs, having)
	}

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
	this.subChildren = make([]plan.Operator, 0, 4)

	if letting != nil {
		this.subChildren = append(this.subChildren, plan.NewLet(letting))
	}

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

	scan, err := this.selectScan(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.children = append(this.children, scan)

	fetch := plan.NewFetch(keyspace, node)
	this.subChildren = append(this.subChildren, fetch)
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
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	unnest := plan.NewUnnest(node)
	this.subChildren = append(this.subChildren, unnest)
	return nil, nil
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
	if !ok {
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
