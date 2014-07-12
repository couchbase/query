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
	"github.com/couchbaselabs/query/expression"
)

func Build(node algebra.Node) (Operator, error) {
	builder := newBuilder()
	op, err := node.Accept(builder)

	if err != nil {
		return nil, err
	}

	switch op := op.(type) {
	case Operator:
		return op, nil
	default:
		panic(fmt.Sprintf("Expected plan.Operator instead of %T.", op))
	}
}

type builder struct {
	projectInitial bool
	children       []Operator
	subChildren    []Operator
}

func newBuilder() *builder {
	return &builder{
		children:    make([]Operator, 0, 8),
		subChildren: make([]Operator, 0, 8),
	}
}

// SELECT

func (this *builder) VisitSelect(node *algebra.Select) (interface{}, error) {
	order := node.Order()
	offset := node.Offset()
	limit := node.Limit()

	if order != nil {
		this.projectInitial = true
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

	if this.projectInitial {
		children = append(children, NewParallel(NewFinalProject()))
	}

	return NewSequence(children...), nil
}

func (this *builder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	if node.From() != nil {
		op, err := node.From().Accept(this)
		if err != nil {
			return nil, err
		}

		this.children = append(this.children, op.(Operator))
	}

	if node.Let() != nil {
		this.subChildren = append(this.subChildren, NewLet(node.Let()))
	}

	if node.Where() != nil {
		this.subChildren = append(this.subChildren, NewFilter(node.Where()))
	}

	group := node.Group()
	projection := node.Projection()

	if group != nil {
		this.visitGroup(group, projection)
	}

	this.subChildren = append(this.subChildren, NewInitialProject(projection))

	if projection.Distinct() {
		this.subChildren = append(this.subChildren, NewFinalProject())
	}

	this.children = append(this.children, NewParallel(NewSequence(this.subChildren...)))

	if projection.Distinct() {
		this.subChildren = append(this.subChildren, NewDistinct())
	}

	return NewSequence(this.children...), nil
}

func (this *builder) visitGroup(group *algebra.Group, projection *algebra.Projection) {
	aggs := make(algebra.Aggregates, 0, 16)

	letting := group.Letting()
	for _, binding := range letting {
		aggs = collectAggregates(aggs, binding.Expression())
	}

	having := group.Having()
	if having != nil {
		aggs = collectAggregates(aggs, having)
	}

	if projection != nil {
		for _, term := range projection.Terms() {
			aggs = collectAggregates(aggs, term.Expression())
		}
	}

	this.subChildren = append(this.subChildren, NewInitialGroup(group.By(), aggs))
	this.subChildren = append(this.subChildren, NewIntermediateGroup(group.By(), aggs))
	this.children = append(this.children, NewParallel(NewSequence(this.subChildren...)))
	this.children = append(this.children, NewFinalGroup(group.By(), aggs))
	this.subChildren = make([]Operator, 0, 4)

	if letting != nil {
		this.subChildren = append(this.subChildren, NewLet(letting))
	}

	if having != nil {
		this.subChildren = append(this.subChildren, NewFilter(having))
	}
}

func (this *builder) VisitUnion(node *algebra.Union) (interface{}, error) {
	this.projectInitial = false

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	unionAll := NewUnionAll(first.(Operator), second.(Operator))
	distinct := NewDistinct()
	return NewSequence(unionAll, distinct), nil
}

func (this *builder) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	this.projectInitial = false

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

func (this *builder) VisitBucketTerm(node *algebra.BucketTerm) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitParentTerm(node *algebra.ParentTerm) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	return nil, nil
}

// DML

func (this *builder) VisitInsert(node *algebra.Insert) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitUpsert(node *algebra.Upsert) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitDelete(node *algebra.Delete) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitUpdate(node *algebra.Update) (interface{}, error) {
	return nil, nil
}

func (this *builder) VisitMerge(node *algebra.Merge) (interface{}, error) {
	return nil, nil
}

// DDL

func (this *builder) VisitCreateIndex(node *algebra.CreateIndex) (interface{}, error) {
	return NewCreateIndex(node), nil
}

func (this *builder) VisitDropIndex(node *algebra.DropIndex) (interface{}, error) {
	return NewDropIndex(node), nil
}

func (this *builder) VisitAlterIndex(node *algebra.AlterIndex) (interface{}, error) {
	return NewAlterIndex(node), nil
}

// EXPLAIN

func (this *builder) VisitExplain(node *algebra.Explain) (interface{}, error) {
	op, err := node.Statement().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewExplain(op.(Operator)), nil
}

func collectAggregates(aggs algebra.Aggregates, exprs ...expression.Expression) algebra.Aggregates {
	for _, expr := range exprs {
		agg, ok := expr.(algebra.Aggregate)

		if ok {
			if len(aggs) == cap(aggs) {
				aggs2 := make(algebra.Aggregates, len(aggs), (cap(aggs)+1)<<1)
				for i, a := range aggs {
					aggs2[i] = a
				}

				aggs = aggs2
			}

			aggs = append(aggs, agg)
		}

		aggs = collectAggregates(aggs, expr.Children()...)
	}

	return aggs
}
