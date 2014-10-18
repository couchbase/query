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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
)

// SELECT

func (this *builder) VisitSelect(node *algebra.Select) (interface{}, error) {
	order := node.Order()
	offset := node.Offset()
	limit := node.Limit()

	// If there is an ORDER BY, delay the final projection
	if order != nil {
		this.projectFinal = false
	}

	sub, err := node.Subresult().Accept(this)
	if err != nil {
		return nil, err
	}

	if order == nil && offset == nil && limit == nil {
		return sub, nil
	}

	children := make([]Operator, 0, 5)
	children = append(children, sub.(Operator))

	if order != nil {
		children = append(children, NewOrder(order))
	}

	if offset != nil {
		children = append(children, NewOffset(offset))
	}

	if limit != nil {
		children = append(children, NewLimit(limit))
	}

	// Perform the delayed final projection now, after the ORDER BY
	if !this.projectFinal {
		children = append(children, NewParallel(NewFinalProject()))
	}

	return NewSequence(children...), nil
}

func (this *builder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	this.children = make([]Operator, 0, 8)     // top-level children, executed sequentially
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

	if node.Let() != nil {
		this.subChildren = append(this.subChildren, NewLet(node.Let()))
	}

	if node.Where() != nil {
		this.subChildren = append(this.subChildren, NewFilter(node.Where()))
	}

	// Check for aggregates
	aggs := make(algebra.Aggregates, 0, 16)
	projection := node.Projection()
	if projection != nil {
		for _, term := range projection.Terms() {
			if term.Expression() != nil {
				aggs = collectAggregates(aggs, term.Expression())
			}
		}
	}

	group := node.Group()
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

	// Perform the final projection if there is no subsequent ORDER BY
	if this.projectFinal {
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

func (this *builder) visitGroup(group *algebra.Group, aggs algebra.Aggregates) {
	letting := group.Letting()
	for _, binding := range letting {
		aggs = collectAggregates(aggs, binding.Expression())
	}

	having := group.Having()
	if having != nil {
		aggs = collectAggregates(aggs, having)
	}

	this.subChildren = append(this.subChildren, NewInitialGroup(group.By(), aggs))
	this.subChildren = append(this.subChildren, NewIntermediateGroup(group.By(), aggs))
	this.children = append(this.children, NewParallel(NewSequence(this.subChildren...)))
	this.children = append(this.children, NewIntermediateGroup(group.By(), aggs))
	this.children = append(this.children, NewFinalGroup(group.By(), aggs))
	this.subChildren = make([]Operator, 0, 4)

	if letting != nil {
		this.subChildren = append(this.subChildren, NewLet(letting))
	}

	if having != nil {
		this.subChildren = append(this.subChildren, NewFilter(having))
	}
}

func (this *builder) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
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

		index, err := keyspace.IndexByPrimary()
		if err != nil {
			return nil, err
		}

		scan := NewPrimaryScan(index)
		this.children = append(this.children, scan)
	}

	fetch := NewFetch(keyspace, node.Project(), node.Alias())
	this.subChildren = append(this.subChildren, fetch)
	return fetch, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	ns := node.Right().Namespace()
	if ns == "" {
		ns = this.namespace
	}

	namespace, err := this.datastore.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(node.Right().Keyspace())
	if err != nil {
		return nil, err
	}

	join := NewJoin(keyspace, node)
	this.subChildren = append(this.subChildren, join)

	return join, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	ns := node.Right().Namespace()
	if ns == "" {
		ns = this.namespace
	}

	namespace, err := this.datastore.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(node.Right().Keyspace())
	if err != nil {
		return nil, err
	}

	nest := NewNest(keyspace, node)
	this.subChildren = append(this.subChildren, nest)

	return nest, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	unnest := NewUnnest(node)
	this.subChildren = append(this.subChildren, unnest)

	return unnest, nil
}

func (this *builder) VisitUnion(node *algebra.Union) (interface{}, error) {
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	this.projectFinal = true

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
	this.projectFinal = true

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
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	this.projectFinal = true

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	intersectAll := NewIntersectAll(first.(Operator), second.(Operator))
	return NewSequence(intersectAll, NewDistinct()), nil
}

func (this *builder) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	this.projectFinal = true

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
	distinct := this.distinct
	this.distinct = true
	defer func() { this.distinct = distinct }()

	this.projectFinal = true

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	exceptAll := NewExceptAll(first.(Operator), second.(Operator))
	return NewSequence(exceptAll, NewDistinct()), nil
}

func (this *builder) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	this.projectFinal = true

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

func collectAggregates(aggs algebra.Aggregates, exprs ...expression.Expression) algebra.Aggregates {
	for _, expr := range exprs {
		agg, ok := expr.(algebra.Aggregate)

		if ok {
			if len(aggs) == cap(aggs) {
				aggs2 := make(algebra.Aggregates, len(aggs), (len(aggs)+1)<<1)
				copy(aggs2, aggs)
				aggs = aggs2
			}

			aggs = append(aggs, agg)
		}

		children := expr.Children()
		if len(children) > 0 {
			aggs = collectAggregates(aggs, children...)
		}
	}

	return aggs
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

	scan := NewCountScan(keyspace)
	this.children = append(this.children, scan)
	return true, nil
}
