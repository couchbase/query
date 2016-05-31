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
	prevCorrelated := this.correlated
	prevCountAgg := this.countAgg
	prevMinAgg := this.minAgg
	defer func() {
		this.cover = prevCover
		this.correlated = prevCorrelated
		this.countAgg = prevCountAgg
		this.minAgg = prevMinAgg
	}()

	this.correlated = node.IsCorrelated()
	this.resetCountMin()

	if this.cover == nil {
		this.cover = node
	}

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

	// Constrain projection to GROUP keys and aggregates
	if group != nil {
		groupKeys := group.By()
		letting := group.Letting()
		if letting != nil {
			groupKeys = append(groupKeys, letting.Identifiers()...)
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

	if len(aggs) == 1 && group.By() == nil {
		for _, term := range node.Projection().Terms() {
			switch expr := term.Expression().(type) {
			case *algebra.Count:
				this.countAgg = expr
			case *algebra.Min:
				this.minAgg = expr
			default:
				if expr.Value() == nil {
					this.resetCountMin()
					break
				}
			}
		}
	}

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams

	// If SELECT DISTINCT, avoid pushing LIMIT down to index scan.
	if this.limit != nil && node.Projection().Distinct() {
		this.resetOrderLimit()
	}

	err = this.visitFrom(node, group)
	if err != nil {
		return nil, err
	}

	if this.coveringScan != nil {
		var covers expression.Covers
		var filterCovers map[*expression.Cover]value.Value

		covers = this.coveringScan.Covers()
		filterCovers = this.coveringScan.FilterCovers()

		coverer := expression.NewCoverer(covers, filterCovers)
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

	if this.countScan == nil {
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
	} else {
		this.children = append(this.children, plan.NewIndexCountProject(node.Projection()))
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
