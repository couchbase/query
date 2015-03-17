//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"fmt"
	"sort"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

// SELECT

func (this *builder) VisitSelect(stmt *algebra.Select) (interface{}, error) {
	order := stmt.Order()
	offset := stmt.Offset()
	limit := stmt.Limit()
	delayProjection := this.delayProjection

	// If there is an ORDER BY, delay the final projection
	if order != nil {
		this.order = order
		this.delayProjection = true
	}

	sub, err := stmt.Subresult().Accept(this)
	if err != nil {
		return nil, err
	}

	if order == nil && offset == nil && limit == nil {
		return sub, nil
	}

	children := make([]Operator, 0, 5)
	children = append(children, sub.(Operator))

	if order != nil {
		if this.order == nil {
			// Disallow aggregates in ORDER BY
			aggs := make(map[string]algebra.Aggregate)
			for _, term := range order.Terms() {
				collectAggregates(aggs, term.Expression())
				if len(aggs) > 0 {
					return nil, fmt.Errorf("Aggregates not available for this ORDER BY.")
				}
			}
		}

		children = append(children, NewOrder(order))
	}

	if offset != nil {
		children = append(children, NewOffset(offset))
	}

	if limit != nil {
		children = append(children, NewLimit(limit))
	}

	// Perform the delayed final projection now, after the ORDER BY
	if this.delayProjection {
		children = append(children, NewParallel(NewFinalProject()))
		this.delayProjection = delayProjection
	}

	return NewSequence(children...), nil
}

func (this *builder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	this.where = node.Where()
	this.children = make([]Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]Operator, 0, 16) // sub-children, executed across data-parallel streams

	count, err := this.fastCount(node)
	if err != nil {
		return nil, err
	}

	if count {
		// do nothing
	} else if node.From() != nil {
		_, err := node.From().Accept(this)
		if err != nil {
			return nil, err
		}
	} else {
		// No FROM clause
		scan := NewDummyScan()
		this.children = append(this.children, scan)
	}

	aggs := make(map[string]algebra.Aggregate)

	if node.Let() != nil {
		for _, binding := range node.Let() {
			collectAggregates(aggs, binding.Expression())
			if len(aggs) > 0 {
				return nil, fmt.Errorf("Aggregates not allowed in LET.")
			}
		}

		this.subChildren = append(this.subChildren, NewLet(node.Let()))
	}

	if node.Where() != nil {
		collectAggregates(aggs, node.Where())
		if len(aggs) > 0 {
			return nil, fmt.Errorf("Aggregates not allowed in WHERE.")
		}

		this.subChildren = append(this.subChildren, NewFilter(node.Where()))
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

	this.subChildren = append(this.subChildren, NewInitialProject(projection))

	// Initial DISTINCT (parallel)
	if projection.Distinct() || this.distinct {
		this.subChildren = append(this.subChildren, NewDistinct())
	}

	if !this.delayProjection {
		// Perform the final projection if there is no subsequent ORDER BY
		this.subChildren = append(this.subChildren, NewFinalProject())
	}

	// Parallelize the subChildren
	this.children = append(this.children, NewParallel(NewSequence(this.subChildren...)))

	// Final DISTINCT (serial)
	if projection.Distinct() || this.distinct {
		this.children = append(this.children, NewDistinct())
	}

	// Serialize the top-level children
	return NewSequence(this.children...), nil
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

	this.subChildren = append(this.subChildren, NewInitialGroup(group.By(), aggv))
	this.children = append(this.children, NewParallel(NewSequence(this.subChildren...)))
	this.children = append(this.children, NewIntermediateGroup(group.By(), aggv))
	this.children = append(this.children, NewFinalGroup(group.By(), aggv))
	this.subChildren = make([]Operator, 0, 4)

	if letting != nil {
		this.subChildren = append(this.subChildren, NewLet(letting))
	}

	if having != nil {
		this.subChildren = append(this.subChildren, NewFilter(having))
	}
}

func (this *builder) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	node.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return nil, err
	}

	if node.Keys() != nil {
		scan := NewKeyScan(node.Keys())
		this.children = append(this.children, scan)
	} else {
		if this.subquery {
			return nil, errors.NewError(nil, fmt.Sprintf(
				"FROM in subquery must use KEYS clause: FROM %s.",
				node.Keyspace()))
		}

		scan, err := this.selectScan(keyspace, node)
		if err != nil {
			return nil, err
		}

		this.children = append(this.children, scan)
	}

	fetch := NewFetch(keyspace, node)
	this.subChildren = append(this.subChildren, fetch)
	return nil, nil
}

func (this *builder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	sel, err := node.Subquery().Accept(this)
	if err != nil {
		return nil, err
	}

	this.children = make([]Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]Operator, 0, 16) // sub-children, executed across data-parallel streams
	this.children = append(this.children, sel.(Operator), NewAlias(node.Alias()))
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

	join := NewJoin(keyspace, node)
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

	nest := NewNest(keyspace, node)
	this.subChildren = append(this.subChildren, nest)
	return nil, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	unnest := NewUnnest(node)
	this.subChildren = append(this.subChildren, unnest)
	return nil, nil
}

func (this *builder) VisitUnion(node *algebra.Union) (interface{}, error) {
	// Inject DISTINCT into both terms
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	this.order = nil             // Disable aggregates from ORDER BY
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	unionAll := NewUnionAll(first.(Operator), second.(Operator))
	return NewSequence(unionAll, NewDistinct()), nil
}

func (this *builder) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	this.order = nil             // Disable aggregates from ORDER BY
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewUnionAll(first.(Operator), second.(Operator)), nil
}

func (this *builder) VisitIntersect(node *algebra.Intersect) (interface{}, error) {
	// Inject DISTINCT into both terms
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	this.order = nil             // Disable aggregates from ORDER BY
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewIntersectAll(first.(Operator), second.(Operator)), nil
}

func (this *builder) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	this.order = nil             // Disable aggregates from ORDER BY
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	// Inject DISTINCT into second term
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewIntersectAll(first.(Operator), second.(Operator)), nil
}

func (this *builder) VisitExcept(node *algebra.Except) (interface{}, error) {
	// Inject DISTINCT into both terms
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	this.order = nil             // Disable aggregates from ORDER BY
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewExceptAll(first.(Operator), second.(Operator)), nil
}

func (this *builder) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	this.order = nil             // Disable aggregates from ORDER BY
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	// Inject DISTINCT into second term
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewExceptAll(first.(Operator), second.(Operator)), nil
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

	scan := NewCountScan(keyspace, from)
	this.children = append(this.children, scan)
	return true, nil
}
