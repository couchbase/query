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
	"github.com/couchbase/query/value"
)

func (this *builder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	prevCover := this.cover
	prevWhere := this.where
	prevCorrelated := this.correlated
	prevCountAgg := this.countAgg
	prevCountDistinctAgg := this.countDistinctAgg
	prevMinAgg := this.minAgg
	prevMaxAgg := this.maxAgg
	prevCoveringScans := this.coveringScans
	prevCoveredUnnests := this.coveredUnnests
	prevCountScan := this.countScan
	prevProjection := this.projection
	prevBasekeyspaces := this.baseKeyspaces
	prevPushableOnclause := this.pushableOnclause
	prevMaxParallelism := this.maxParallelism

	defer func() {
		this.cover = prevCover
		this.where = prevWhere
		this.correlated = prevCorrelated
		this.countAgg = prevCountAgg
		this.countDistinctAgg = prevCountDistinctAgg
		this.minAgg = prevMinAgg
		this.maxAgg = prevMaxAgg
		this.coveringScans = prevCoveringScans
		this.coveredUnnests = prevCoveredUnnests
		this.countScan = prevCountScan
		this.projection = prevProjection
		this.baseKeyspaces = prevBasekeyspaces
		this.pushableOnclause = prevPushableOnclause
		this.maxParallelism = prevMaxParallelism
	}()

	this.coveringScans = make([]plan.CoveringOperator, 0, 4)
	this.coveredUnnests = nil
	this.countScan = nil
	this.correlated = node.IsCorrelated()
	this.projection = nil
	this.baseKeyspaces = nil
	this.pushableOnclause = nil
	this.maxParallelism = 0
	this.resetCountMinMax()

	if this.cover == nil {
		this.cover = node
	}

	// Inline LET expressions for index selection
	if node.Let() != nil && node.Where() != nil {
		var err error
		inliner := expression.NewInliner(node.Let().Mappings())
		this.where, err = inliner.Map(node.Where().Copy())
		if err != nil {
			return nil, err
		}
	} else {
		this.where = node.Where()
	}

	// Infer WHERE clause from UNNEST
	if node.From() != nil {
		this.inferUnnestPredicates(node.From())
	}

	aggs, err := allAggregates(node, this.order)
	if err != nil {
		return nil, err
	}

	// Infer WHERE clause from aggregates
	group := node.Group()
	if group == nil && len(aggs) > 0 {
		group = algebra.NewGroup(nil, nil, nil)
		this.where = constrainAggregate(this.where, aggs)
	}

	// Constrain projection to GROUP keys and aggregates
	if group != nil {
		groupKeys := group.By()
		letting := group.Letting()
		if letting != nil {
			identifiers := letting.Identifiers()
			groupKeys = append(groupKeys, identifiers...)
		}

		proj := node.Projection().Terms()
		allowed := value.NewScopeValue(make(map[string]interface{}, len(proj)), nil)
		for _, p := range proj {
			err = constrainGroupProjection(p, groupKeys, allowed)
			if err != nil {
				return nil, err
			}
		}

		if this.order != nil {
			for _, t := range proj {
				if t.As() != "" {
					allowed.SetField(t.As(), true)
				}
			}

			ord := this.order.Expressions()
			for _, o := range ord {
				err = constrainGroupTerm(o, groupKeys, allowed)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Identify aggregates for index pushdown
	if len(aggs) == 1 && group.By() == nil {
	loop:
		for _, term := range node.Projection().Terms() {
			switch expr := term.Expression().(type) {
			case *algebra.Count:
				this.countAgg = expr
			case *algebra.CountDistinct:
				this.countDistinctAgg = expr
			case *algebra.Min:
				this.minAgg = expr
			case *algebra.Max:
				this.maxAgg = expr
			default:
				if expr.Value() == nil {
					this.resetCountMinMax()
					break loop
				}
			}
		}
	}

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams

	// If SELECT DISTINCT, avoid pushing LIMIT down to index scan.
	if this.hasOffsetOrLimit() && node.Projection().Distinct() {
		this.resetOffsetLimit()
	}

	// Skip fixed values in ORDER BY
	if this.order != nil && this.where != nil {
		order := this.order
		this.order = skipFixedOrderTerms(this.order, this.where)
		if this.order == nil {
			defer func() { this.order = order }()
		}
	}

	if group == nil && node.Projection().Distinct() {
		this.projection = node.Projection()
	}

	err = this.visitFrom(node, group)
	if err != nil {
		return nil, err
	}

	if len(this.coveringScans) > 0 {
		err = this.coverExpressions()
		if err != nil {
			return nil, err
		}
	}

	if this.countScan == nil {
		this.addLetAndPredicate(node.Let(), node.Where())

		if group != nil {
			this.visitGroup(group, aggs)
		}

		projection := node.Projection()
		this.subChildren = append(this.subChildren, plan.NewInitialProject(projection))

		// Initial DISTINCT (parallel)
		if projection.Distinct() || this.setOpDistinct {
			this.subChildren = append(this.subChildren, plan.NewDistinct())
		}

		if this.order != nil {
			this.delayProjection = false
		}

		if !this.delayProjection {
			// Perform the final projection if there is no subsequent ORDER BY
			this.subChildren = append(this.subChildren, plan.NewFinalProject())
		}

		// Parallelize the subChildren
		this.children = append(this.children, plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism))

		// Final DISTINCT (serial)
		if projection.Distinct() || this.setOpDistinct {
			this.children = append(this.children, plan.NewDistinct())
		}
	} else {
		this.children = append(this.children, plan.NewIndexCountProject(node.Projection()))
	}

	// Serialize the top-level children
	return plan.NewSequence(this.children...), nil
}

func (this *builder) addLetAndPredicate(let expression.Bindings, pred expression.Expression) {
	if let != nil && pred != nil {
	outer:
		for {
			identifiers := let.Identifiers()
			for _, id := range identifiers {
				if pred.DependsOn(id) {
					break outer
				}
			}

			// Predicate does NOT depend on LET
			this.subChildren = append(this.subChildren, plan.NewFilter(pred))
			this.subChildren = append(this.subChildren, plan.NewLet(let))
			return
		}
	}

	if let != nil {
		this.subChildren = append(this.subChildren, plan.NewLet(let))
	}

	if pred != nil {
		this.subChildren = append(this.subChildren, plan.NewFilter(pred))
	}
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

	this.addLetAndPredicate(group.Letting(), group.Having())
}

func (this *builder) coverExpressions() error {
	for _, op := range this.coveringScans {
		coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

		err := this.cover.MapExpressions(coverer)
		if err != nil {
			return err
		}

		if this.where != nil {
			this.where, err = coverer.Map(this.where)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *builder) inferUnnestPredicates(from algebra.FromTerm) {
	// Enumerate INNER UNNESTs
	unnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(unnests)
	unnests = collectInnerUnnests(from, unnests)
	if len(unnests) == 0 {
		return
	}

	// Enumerate primary UNNESTs
	primaryUnnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(primaryUnnests)
	primaryUnnests = collectPrimaryUnnests(from, unnests, primaryUnnests)
	if len(primaryUnnests) == 0 {
		return
	}

	// INNER UNNESTs cannot be MISSING, so add to WHERE clause
	var andBuf [16]expression.Expression
	var andTerms []expression.Expression
	if 1+len(primaryUnnests) <= len(andBuf) {
		andTerms = andBuf[0:0]
	} else {
		andTerms = make(expression.Expressions, 0, 1+len(primaryUnnests))
	}

	if this.where != nil {
		andTerms = append(andTerms, this.where)
	}

	for _, unnest := range primaryUnnests {
		andTerms = append(andTerms, expression.NewIsArray(unnest.Expression()))
	}

	this.where = expression.NewAnd(andTerms...)
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

	if cond == nil {
		return constraint
	} else if SubsetOf(cond, constraint) {
		return cond
	} else {
		return expression.NewAnd(cond, constraint)
	}
}

func constrainGroupProjection(term *algebra.ResultTerm, groupKeys expression.Expressions,
	allowed *value.ScopeValue) errors.Error {
	expr := term.Expression()
	if expr == nil {
		return errors.NewNotGroupKeyOrAggError(term.String())
	}

	return constrainGroupTerm(expr, groupKeys, allowed)
}

func constrainGroupTerm(expr expression.Expression, groupKeys expression.Expressions,
	allowed *value.ScopeValue) errors.Error {
	ok, _ := expr.SurvivesGrouping(groupKeys, allowed)
	if !ok {
		return errors.NewNotGroupKeyOrAggError(expr.String())
	}

	return nil
}

func skipFixedOrderTerms(order *algebra.Order, pred expression.Expression) *algebra.Order {
	filterCovers := _FILTER_COVERS_POOL.Get()
	defer _FILTER_COVERS_POOL.Put(filterCovers)

	filterCovers = pred.FilterCovers(filterCovers)
	if len(filterCovers) == 0 {
		return order
	}

	sortTerms := make(algebra.SortTerms, 0, len(order.Terms()))
	for _, term := range order.Terms() {
		expr := term.Expression()
		if expr.Static() != nil {
			continue
		}

		if val, ok := filterCovers[expr.String()]; !ok || val == nil {
			sortTerms = append(sortTerms, term)
		}
	}

	if len(sortTerms) == 0 {
		return nil
	} else {
		return algebra.NewOrder(sortTerms)
	}
}
