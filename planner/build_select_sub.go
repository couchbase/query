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
	prevCoveringScans := this.coveringScans
	prevCoveredUnnests := this.coveredUnnests
	prevCountScan := this.countScan
	prevBasekeyspaces := this.baseKeyspaces
	prevPushableOnclause := this.pushableOnclause
	prevBuilderFlags := this.builderFlags
	prevMaxParallelism := this.maxParallelism

	indexPushDowns := this.storeIndexPushDowns()

	defer func() {
		this.cover = prevCover
		this.where = prevWhere
		this.correlated = prevCorrelated
		this.coveringScans = prevCoveringScans
		this.coveredUnnests = prevCoveredUnnests
		this.countScan = prevCountScan
		this.baseKeyspaces = prevBasekeyspaces
		this.pushableOnclause = prevPushableOnclause
		this.builderFlags = prevBuilderFlags
		this.maxParallelism = prevMaxParallelism
		this.restoreIndexPushDowns(indexPushDowns, false)
	}()

	this.coveringScans = make([]plan.CoveringOperator, 0, 4)
	this.coveredUnnests = nil
	this.countScan = nil
	this.correlated = node.IsCorrelated()
	this.baseKeyspaces = nil
	this.pushableOnclause = nil
	this.builderFlags = 0
	this.maxParallelism = 0

	this.projection = node.Projection()
	this.resetIndexGroupAggs()

	if this.cover == nil {
		this.cover = node
	}

	this.node = node

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
		this.where = this.constrainAggregate(this.where, aggs)
	}

	// Constrain projection to GROUP keys and aggregates
	if group != nil {
		proj := node.Projection().Terms()
		allowed := value.NewScopeValue(make(map[string]interface{}, len(proj)), nil)

		groupKeys := group.By()
		letting := group.Letting()
		// Only aggregates and group keys are allowed in LETTING caluse
		if letting != nil {
			for _, expr := range letting.Expressions() {
				if expr != nil {
					err = constrainGroupTerm(expr, groupKeys, allowed)
					if err != nil {
						return nil, err
					}
				}
			}
			identifiers := letting.Identifiers()
			groupKeys = append(groupKeys, identifiers...)
		}

		// Only aggregates and group keys, LETTING varaiables are allowed in HAVING caluse
		if group.Having() != nil {
			err = constrainGroupTerm(group.Having(), groupKeys, allowed)
			if err != nil {
				return nil, err
			}
			this.resetOffsetLimit()
		}

		for _, p := range proj {
			expr := p.Expression()
			if expr == nil {
				return nil, errors.NewNotGroupKeyOrAggError(p.String())
			}

			err = constrainGroupTerm(expr, groupKeys, allowed)
			if err != nil {
				return nil, err
			}
		}

		if this.order != nil {
			allow_flags := value.NewValue(uint32(expression.IDENT_IS_PROJ_ALIAS))
			for _, t := range proj {
				if t.As() != "" {
					allowed.SetField(t.As(), allow_flags)
				}
			}

			// ONLY aggregates, group kyes, LETTING varaibles, projection are allowed IN ORDER BY
			ord := this.order.Expressions()
			for _, o := range ord {
				err = constrainGroupTerm(o, groupKeys, allowed)
				if err != nil {
					return nil, err
				}
			}
		}
		this.resetProjection()
	}

	// Identify aggregates for index pushdown for old releases
	if len(aggs) == 1 && group.By() == nil {
	loop:
		for _, term := range node.Projection().Terms() {
			switch expr := term.Expression().(type) {
			case *algebra.Count, *algebra.CountDistinct, *algebra.Min, *algebra.Max:
				this.oldAggregates = true
			default:
				if expr.Value() == nil {
					this.oldAggregates = false
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

	this.setIndexGroupAggs(group, aggs, node.Let())

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

	if this.aggs != nil {
		aggs = this.aggs
	}

	if this.countScan == nil {
		// Add Let and Filter only when group/aggregates are not pushed
		if this.group == nil {
			this.addLetAndPredicate(node.Let(), node.Where())
		}

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

func (this *builder) visitGroup(group *algebra.Group, aggs algebra.Aggregates) {

	// If Index aggregates are not partial(i.e full) donot add the group operators

	partial := true
	if this.group != nil && len(this.coveringScans) == 1 && this.coveringScans[0].GroupAggs() != nil {
		partial = this.coveringScans[0].GroupAggs().Partial
	}

	if partial {
		aggv := sortAggregatesSlice(aggs)
		this.subChildren = append(this.subChildren, plan.NewInitialGroup(group.By(), aggv))
		this.children = append(this.children,
			plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism))
		this.children = append(this.children, plan.NewIntermediateGroup(group.By(), aggv))
		this.children = append(this.children, plan.NewFinalGroup(group.By(), aggv))
		this.subChildren = make([]plan.Operator, 0, 8)
	}

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

	return this.coverIndexGroupAggs()
}

func (this *builder) coverIndexGroupAggs() (err error) {
	if len(this.coveringScans) != 1 || this.coveringScans[0].GroupAggs() == nil {
		return
	}

	// Add cover to the index key expressions inside the aggregates used in group Operators
	op := this.coveringScans[0]
	var expr expression.Expression

	indexKeyCovers := op.Covers()
	idCover := indexKeyCovers[len(indexKeyCovers)-1]
	keyCoverer := expression.NewCoverer(indexKeyCovers, nil)

	err = this.coverIndexGroupAggsMap(keyCoverer)
	if err != nil {
		return
	}

	// Add cover to the index key expressions of group keys in the plan
	// generate new covers for group keys
	indexGroupAgg := op.GroupAggs()
	groupCovers := make(expression.Covers, 0, len(indexGroupAgg.Group))
	for _, indexgroupKey := range indexGroupAgg.Group {
		if indexgroupKey.Expr != nil {
			expr, err = keyCoverer.Map(indexgroupKey.Expr)
			if err != nil {
				return
			}
			indexgroupKey.Expr = expr
		}
		if indexgroupKey.KeyPos < 0 && !expression.Equivalent(idCover, indexgroupKey.Expr) {
			groupCovers = append(groupCovers, expression.NewCover(indexgroupKey.Expr.Copy()))
		}
	}

	// Add the new group covers to the existing coverers
	// replace the all the statement expressions with group covers
	if len(groupCovers) > 0 {
		op.SetCovers(append(op.Covers(), groupCovers...))
		groupCoverer := expression.NewCoverer(groupCovers, nil)
		err = this.cover.MapExpressions(groupCoverer)
		if err != nil {
			return
		}
	}

	// Add cover to the index key expressions of aggregates in the plan
	for _, indexAgg := range indexGroupAgg.Aggregates {
		if indexAgg.Expr != nil {
			expr, err = keyCoverer.Map(indexAgg.Expr)
			if err != nil {
				return
			}
			indexAgg.Expr = expr
		}
	}

	// generate new covers for aggregates
	aggCovers := make(expression.Covers, 0, len(indexGroupAgg.Aggregates))
	for _, agg := range this.aggs {
		aggCovers = append(aggCovers, expression.NewCover(agg.Copy()))
	}

	// replace the all the statement expressions with aggregates
	// it also changes group operators/aggregates (i.e countn to sum, avg sum/countn)
	// perform multi level aggregation if the results are partial aggregates
	if len(aggCovers) > 0 {
		op.SetCovers(append(op.Covers(), aggCovers...))
		if indexGroupAgg.Partial {
			aggPartialCoverer := NewPartialAggCoverer(aggCovers, this.aggs)

			err = this.coverIndexGroupAggsMap(aggPartialCoverer)
			if err != nil {
				return
			}

			aggPartialCoverer = NewPartialAggCoverer(aggCovers, this.aggs)
			err = this.cover.MapExpressions(aggPartialCoverer)
			if err != nil {
				return
			}
		} else {
			aggFullCoverer := NewFullAggCoverer(aggCovers)
			err = this.cover.MapExpressions(aggFullCoverer)
			if err != nil {
				return
			}
		}
	}

	return
}

func (this *builder) coverIndexGroupAggsMap(mapper expression.Mapper) error {
	for i, agg := range this.aggs {
		expr, err := mapper.Map(agg)
		if err != nil {
			return err
		}

		nagg, ok := expr.(algebra.Aggregate)
		if !ok {
			return fmt.Errorf("Error in Aggregates Mapping.")
		}
		this.aggs[i] = nagg
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

func allAggregates(node *algebra.Subselect, order *algebra.Order) (algebra.Aggregates, error) {
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
		collectAggregates(aggs, group.By()...)
		if len(aggs) > 0 {
			return nil, fmt.Errorf("Aggregates not allowed in GROUP BY.")
		}

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

		if !allow && group == nil && len(aggs) > 0 {
			return nil, fmt.Errorf("Aggregates not available for this ORDER BY.")
		}
	}

	if len(aggs) > 0 {
		// Disallow nested aggregates
		subAggs := make(map[string]algebra.Aggregate)
		for _, agg := range aggs {
			collectAggregates(subAggs, agg.Operand())
			if len(subAggs) > 0 {
				return nil, fmt.Errorf("Nested aggregates are not allowed.")
			}
		}
	}

	return sortAggregatesMap(aggs), nil
}

func sortAggregatesMap(aggs map[string]algebra.Aggregate) algebra.Aggregates {
	aggn := make(sort.StringSlice, 0, len(aggs))
	for n, _ := range aggs {
		aggn = append(aggn, n)
	}

	aggn.Sort()
	aggv := make(algebra.Aggregates, len(aggs))
	for i, n := range aggn {
		aggv[i] = aggs[n]
	}

	return aggv
}

func sortAggregatesSlice(aggSlice algebra.Aggregates) algebra.Aggregates {
	aggs := make(map[string]algebra.Aggregate)
	stringer := expression.NewStringer()
	for _, agg := range aggSlice {
		aggs[stringer.Visit(agg)] = agg
	}
	return sortAggregatesMap(aggs)
}

func collectAggregates(aggs map[string]algebra.Aggregate, exprs ...expression.Expression) {
	stringer := expression.NewStringer()

	for _, expr := range exprs {
		if expr == nil {
			continue
		}
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
func (this *builder) constrainAggregate(cond expression.Expression, aggs algebra.Aggregates) expression.Expression {
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

	switch first.(type) {
	case *expression.ArrayConstruct, *expression.ObjectConstruct:
		return cond
	}

	var constraint expression.Expression = expression.NewIsNotNull(first)

	if cond == nil {
		return constraint
	} else if SubsetOf(cond, constraint) {
		return cond
	} else {
		this.aggConstraint = constraint
		return expression.NewAnd(cond, constraint)
	}
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

func (this *builder) setIndexGroupAggs(group *algebra.Group, aggs algebra.Aggregates, let expression.Bindings) {

	if group != nil {
		// Group or Aggregates Depends on LET disable pushdowns
		for _, expr := range group.By() {
			if !expr.IndexAggregatable() || dependsOnLet(expr, let) {
				this.resetPushDowns()
				return
			}
		}

		for _, agg := range aggs {
			aggIndexProperties := aggToIndexAgg(agg)
			if !aggIndexProperties.supported {
				this.resetPushDowns()
				return
			}

			if agg.Operand() == nil {
				continue
			}

			if !agg.Operand().IndexAggregatable() || dependsOnLet(agg.Operand(), let) {
				this.resetPushDowns()
				return
			}

		}
		this.group = group
		this.aggs = aggs
	}
}

func dependsOnLet(expr expression.Expression, let expression.Bindings) bool {
	if let != nil && expr != nil {
		for _, id := range let.Identifiers() {
			if expr.DependsOn(id) {
				return true
			}
		}
	}

	return false
}
