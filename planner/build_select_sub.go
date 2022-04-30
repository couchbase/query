//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"
	"math"
	"sort"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *builder) VisitSubselect(node *algebra.Subselect) (interface{}, error) {
	prevCover := this.cover
	prevWhere := this.where
	prevFilter := this.filter
	prevCorrelated := this.correlated
	prevCoveringScans := this.coveringScans
	prevCoveredUnnests := this.coveredUnnests
	prevCountScan := this.countScan
	prevBasekeyspaces := this.baseKeyspaces
	prevKeyspaceNames := this.keyspaceNames
	prevPushableOnclause := this.pushableOnclause
	prevBuilderFlags := this.builderFlags
	prevMaxParallelism := this.maxParallelism
	prevAliases := this.aliases
	prevLastOp := this.lastOp

	indexPushDowns := this.storeIndexPushDowns()

	defer func() {
		this.cover = prevCover
		this.where = prevWhere
		this.filter = prevFilter
		this.correlated = prevCorrelated
		this.coveringScans = prevCoveringScans
		this.coveredUnnests = prevCoveredUnnests
		this.countScan = prevCountScan
		this.baseKeyspaces = prevBasekeyspaces
		this.keyspaceNames = prevKeyspaceNames
		this.pushableOnclause = prevPushableOnclause
		this.resetBuilderFlags(prevBuilderFlags)
		this.maxParallelism = prevMaxParallelism
		this.lastOp = prevLastOp
		this.aliases = prevAliases
		this.restoreIndexPushDowns(indexPushDowns, false)
	}()

	this.coveringScans = make([]plan.CoveringOperator, 0, 4)
	this.coveredUnnests = nil
	this.countScan = nil
	this.correlated = node.IsCorrelated()
	this.baseKeyspaces = nil
	this.keyspaceNames = nil
	this.pushableOnclause = nil
	this.passthruBuilderFlags(prevBuilderFlags)
	this.maxParallelism = 0
	this.lastOp = nil
	this.aliases = nil

	this.projection = node.Projection()
	this.resetIndexGroupAggs()

	if this.cover == nil {
		this.cover = node
	}

	this.node = node

	if this.limit != nil {
		this.setBuilderFlag(BUILDER_HAS_LIMIT)
	}
	if this.offset != nil {
		this.setBuilderFlag(BUILDER_HAS_OFFSET)
	}
	if this.order != nil {
		this.setBuilderFlag(BUILDER_HAS_ORDER)

		if node.Let() != nil {
			identifiers := node.Let().Identifiers()
			// check whether ORDER expressions depends on LET bindings
			depends := false
		outer:
			for _, term := range this.order.Expressions() {
				for _, id := range identifiers {
					if term.DependsOn(id) {
						depends = true
						break outer
					}
				}
			}
			if depends {
				this.setBuilderFlag(BUILDER_ORDER_DEPENDS_ON_LET)
			}
		}
	}

	var err error

	// Inline LET expressions for index selection
	if node.Let() != nil && node.Where() != nil {
		inliner := expression.NewInliner(node.Let().Mappings())
		level := getMaxLevelOfLetBindings(node.Let())
		this.where, err = dereferenceLet(node.Where().Copy(), inliner, level)
		if err != nil {
			return nil, err
		}
	} else {
		this.where = node.Where()
	}

	this.where, err = this.getWhere(this.where)
	if err != nil {
		return nil, err
	}

	this.filter = nil
	if this.where != nil {
		this.filter = this.where.Copy()
	}

	this.extractLetGroupProjOrder(node.Let(), nil, node.Projection(), this.order, nil)

	// Infer WHERE clause from UNNEST
	if node.From() != nil {
		this.inferUnnestPredicates(node.From())
	}

	aggs, windowAggs, err := allAggregates(node, this.order)
	if err != nil {
		return nil, err
	}

	if len(windowAggs) > 0 {
		this.resetOrderOffsetLimit()
		this.resetProjection()
		this.setBuilderFlag(BUILDER_HAS_WINDOW_AGGS)
	}

	// Infer WHERE clause from aggregates
	group := node.Group()
	if group == nil && len(aggs) > 0 {
		group = algebra.NewGroup(nil, nil, nil)
		this.where = this.constrainAggregate(this.where, aggs)
	}

	// Constrain projection to GROUP keys and aggregates
	if group != nil {
		this.setBuilderFlag(BUILDER_HAS_GROUP)
		proj := node.Projection().Terms()
		allowed := value.NewScopeValue(make(map[string]interface{}, len(proj)), nil)

		groupKeys := group.By()
		letting := group.Letting()
		// Only aggregates and group keys are allowed in LETTING caluse
		if letting != nil {
			for _, bind := range letting {
				expr := bind.Expression()
				if expr != nil {
					err = constrainGroupTerm(expr, groupKeys, allowed)
					if err != nil {
						return nil, err
					}
				}
				vid := expression.NewIdentifier(bind.Variable())
				vid.SetBindingVariable(true)
				groupKeys = append(groupKeys, vid)
			}
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
				if !t.Star() && t.Alias() != "" {
					allowed.SetField(t.Alias(), allow_flags)
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
	}

	// Identify aggregates for index pushdown for old releases
	if len(aggs) == 1 && len(group.By()) == 0 {
	loop:
		for _, term := range node.Projection().Terms() {
			switch expr := term.Expression().(type) {
			case *algebra.Count, *algebra.Min, *algebra.Max:
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
	projection := node.Projection()
	if this.hasOffsetOrLimit() && projection.Distinct() {
		this.resetOffsetLimit()
	}

	this.setIndexGroupAggs(group, aggs, node.Let())
	this.extractLetGroupProjOrder(nil, group, nil, nil, aggs)

	err = this.visitFrom(node, group, projection, indexPushDowns)
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

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL

	if this.countScan == nil {
		// Add Let and Filter only when group/aggregates are not pushed
		if this.group == nil {
			this.addLetAndPredicate(node.Let(), this.filter)
		}

		if group != nil {
			this.visitGroup(group, aggs)
		}

		if len(windowAggs) > 0 {
			this.visitWindowAggregates(windowAggs)
		}

		if this.useCBO && this.lastOp != nil {
			cost = this.lastOp.Cost()
			cardinality = this.lastOp.Cardinality()
			size = this.lastOp.Size()
			frCost = this.lastOp.FrCost()
			if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
				cost, cardinality, size, frCost = getInitialProjectCost(projection, cost, cardinality, size, frCost)
			}
		}
		this.addSubChildren(plan.NewInitialProject(projection, cost, cardinality, size, frCost))

		// Initial DISTINCT (parallel)
		if projection.Distinct() || this.setOpDistinct {
			if this.useCBO && this.lastOp != nil {
				cost = this.lastOp.Cost()
				cardinality = this.lastOp.Cardinality()
				size = this.lastOp.Size()
				frCost = this.lastOp.FrCost()
				if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
					cost, cardinality, size, frCost = getDistinctCost(projection.Terms(), cost, cardinality, size, frCost, this.keyspaceNames)
				}
			}
			this.addSubChildren(plan.NewDistinct(cost, cardinality, size, frCost))
		}

		if this.order != nil {
			this.delayProjection = false
		}

		if !this.delayProjection {

			// Perform the final projection if there is no subsequent ORDER BY
			// TODO retire
			this.subChildren = maybeFinalProject(this.subChildren)
		}

		// Parallelize the subChildren
		this.addChildren(this.addSubchildrenParallel())

		// Final DISTINCT (serial)
		if projection.Distinct() || this.setOpDistinct {
			// use the same cost/cardinality calculated above for DISTINCT
			this.addChildren(plan.NewDistinct(cost, cardinality, size, frCost))
		}
	} else {
		this.addChildren(plan.NewIndexCountProject(projection))
	}

	// Serialize the top-level children
	return plan.NewSequence(this.children...), nil
}

func (this *builder) addLetAndPredicate(let expression.Bindings, pred expression.Expression) {
	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	advisorValidate := this.advisorValidate()

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
			if this.useCBO {
				cost, cardinality, size, frCost = getFilterCost(this.lastOp, pred,
					this.baseKeyspaces, this.keyspaceNames, "", advisorValidate, this.context)
			}
			filter := plan.NewFilter(pred, "", cost, cardinality, size, frCost)
			if this.useCBO {
				cost, cardinality, size, frCost = getLetCost(this.lastOp)
			}
			letop := plan.NewLet(let, cost, cardinality, size, frCost)
			this.addSubChildren(filter, letop)
			return
		}
	}

	if let != nil {
		if this.useCBO {
			cost, cardinality, size, frCost = getLetCost(this.lastOp)
		}
		this.addSubChildren(plan.NewLet(let, cost, cardinality, size, frCost))
	}

	if pred != nil {
		if this.useCBO {
			cost, cardinality, size, frCost = getFilterCost(this.lastOp, pred, this.baseKeyspaces,
				this.keyspaceNames, "", advisorValidate, this.context)
		}
		this.addSubChildren(plan.NewFilter(pred, "", cost, cardinality, size, frCost))
	}
}

func (this *builder) visitGroup(group *algebra.Group, aggs algebra.Aggregates) {

	// If Index aggregates are not partial(i.e full) donot add the group operators

	partial := true
	if this.group != nil && len(this.coveringScans) == 1 && this.coveringScans[0].GroupAggs() != nil {
		partial = this.coveringScans[0].GroupAggs().Partial
	}

	if partial {
		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		costInitial := OPT_COST_NOT_AVAIL
		cardinalityInitial := OPT_CARD_NOT_AVAIL
		costIntermediate := OPT_COST_NOT_AVAIL
		cardinalityIntermediate := OPT_CARD_NOT_AVAIL
		costFinal := OPT_COST_NOT_AVAIL
		cardinalityFinal := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		last := this.lastOp
		if this.useCBO && last != nil {
			cost = last.Cost()
			cardinality = last.Cardinality()
			size = last.Size()
			if cost > 0.0 && cardinality > 0.0 && size > 0 {
				costInitial, cardinalityInitial, costIntermediate, cardinalityIntermediate, costFinal, cardinalityFinal =
					getGroupCosts(group, aggs, cost, cardinality, size, this.keyspaceNames, this.maxParallelism)
			}
		}
		aggv := sortAggregatesSlice(aggs)
		canSpill := util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_SPILL_TO_DISK)
		this.addSubChildren(plan.NewInitialGroup(group.By(), aggv,
			costInitial, cardinalityInitial, size, costInitial, canSpill))
		this.addChildren(this.addSubchildrenParallel())
		this.addChildren(plan.NewIntermediateGroup(group.By(), aggv,
			costIntermediate, cardinalityIntermediate, size, costIntermediate, canSpill))
		this.addChildren(plan.NewFinalGroup(group.By(), aggv,
			costFinal, cardinalityFinal, size, costFinal, canSpill))
	}

	this.addLetAndPredicate(group.Letting(), group.Having())
}

func (this *builder) renameAnyExpression(arrayKey *expression.All, filter, where, joinKeys expression.Expression) (
	expression.Expression, expression.Expression, expression.Expression, error) {
	var err error
	if arrayKey != nil {
		anyRenamer := expression.NewAnyRenamer(arrayKey)
		if filter != nil {
			filter, err = anyRenamer.Map(filter)
		}
		if err == nil && where != nil {
			where, err = anyRenamer.Map(where)
		}
		if err == nil && joinKeys != nil {
			joinKeys, err = anyRenamer.Map(joinKeys)
		}
	}
	return filter, where, joinKeys, err
}

func (this *builder) coverExpression(coverer *expression.Coverer, filter, where, joinKeys expression.Expression) (
	expression.Expression, expression.Expression, expression.Expression, error) {

	var err error
	filter, err = coverer.CoverExpr(filter)
	if err == nil {
		where, err = coverer.CoverExpr(where)
	}
	if err == nil {
		joinKeys, err = coverer.CoverExpr(joinKeys)
	}
	return filter, where, joinKeys, err
}

func (this *builder) coverExpressions() (err error) {
	for _, op := range this.coveringScans {
		if arrayKey := op.ImplicitArrayKey(); arrayKey != nil {
			anyRenamer := expression.NewAnyRenamer(arrayKey)
			err = this.cover.MapExpressions(anyRenamer)
			if err == nil {
				this.filter, this.where, _, err = this.renameAnyExpression(arrayKey, this.filter, this.where, nil)
			}
			if err != nil {
				return err
			}
		}
	}

	for _, op := range this.coveringScans {
		coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())
		err = this.cover.MapExpressions(coverer)
		if err == nil {
			this.filter, this.where, _, err = this.coverExpression(coverer, this.filter, this.where, nil)
		}
		if err != nil {
			return err
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
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return
	}
	unnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(unnests)
	unnests = collectInnerUnnestsFromJoinTerm(joinTerm, unnests)

	// Enumerate primary UNNESTs
	primaryUnnests := collectPrimaryUnnests(from.PrimaryTerm(), unnests)
	if nil != primaryUnnests {
		defer _UNNEST_POOL.Put(primaryUnnests)
	}
	if len(primaryUnnests) == 0 {
		return
	}

	// INNER UNNESTs cannot be MISSING, so add to WHERE clause
	var andBuf [32]expression.Expression
	var andTerms []expression.Expression
	if 1+(2*len(primaryUnnests)) <= len(andBuf) {
		andTerms = andBuf[0:0]
	} else {
		andTerms = make(expression.Expressions, 0, 1+(2*len(primaryUnnests)))
	}

	if this.where != nil {
		andTerms = append(andTerms, this.where)
	}

	for _, unnest := range primaryUnnests {
		ident := expression.NewIdentifier(unnest.Alias())
		ident.SetUnnestAlias(true)
		notMissing := expression.NewIsNotMissing(ident)
		notMissing.SetExprFlag(expression.EXPR_UNNEST_NOT_MISSING)
		isArray := expression.NewIsArray(unnest.Expression())
		isArray.SetExprFlag(expression.EXPR_UNNEST_ISARRAY)
		andTerms = append(andTerms, isArray, notMissing)
	}

	this.where = expression.NewAnd(andTerms...)
}

func allAggregates(node *algebra.Subselect, order *algebra.Order) (algebra.Aggregates, algebra.Aggregates, error) {

	var err error
	aggs := make(map[string]algebra.Aggregate)
	windowAggs := make(map[string]algebra.Aggregate)

	if node.Let() != nil {
		if err = collectAggregates(aggs, windowAggs, node.Let().Expressions()...); err != nil {
			return nil, nil, err
		}

		if len(aggs) > 0 {
			ec := getFirstErrorContext(aggs)
			if len(ec) == 0 {
				ec = node.Let()[0].Expression().ErrorContext()
			}
			return nil, nil, fmt.Errorf("Aggregates not allowed in LET%v.", ec)
		}

		if len(windowAggs) > 0 {
			ec := getFirstErrorContext(windowAggs)
			if len(ec) == 0 {
				ec = node.Let()[0].Expression().ErrorContext()
			}
			return nil, nil, fmt.Errorf("Window Aggregates not allowed in LET%v.", ec)
		}
	}

	if node.Where() != nil {
		if err = collectAggregates(aggs, windowAggs, node.Where()); err != nil {
			return nil, nil, err
		}

		if len(aggs) > 0 {
			ec := getFirstErrorContext(aggs)
			if len(ec) == 0 {
				ec = node.Where().ErrorContext()
			}
			return nil, nil, fmt.Errorf("Aggregates not allowed in WHERE%v.", ec)
		}

		if len(windowAggs) > 0 {
			ec := getFirstErrorContext(windowAggs)
			if len(ec) == 0 {
				ec = node.Where().ErrorContext()
			}
			return nil, nil, fmt.Errorf("Window Aggregates not allowed in WHERE%v.", ec)
		}
	}

	group := node.Group()
	if group != nil {
		if err = collectAggregates(aggs, windowAggs, group.By()...); err != nil {
			return nil, nil, err
		}

		if len(aggs) > 0 {
			ec := getFirstErrorContext(aggs)
			if len(ec) == 0 {
				ec = group.By()[0].ErrorContext()
			}
			return nil, nil, fmt.Errorf("Aggregates not allowed in GROUP BY%v.", ec)
		}

		if len(windowAggs) > 0 {
			ec := getFirstErrorContext(windowAggs)
			if len(ec) == 0 {
				ec = group.By()[0].ErrorContext()
			}
			return nil, nil, fmt.Errorf("Window Aggregates not allowed in GROUP BY%v.", ec)
		}

		if group.Letting() != nil {
			if err = collectAggregates(aggs, windowAggs, group.Letting().Expressions()...); err != nil {
				return nil, nil, err
			}

			if len(windowAggs) > 0 {
				ec := getFirstErrorContext(windowAggs)
				if len(ec) == 0 {
					ec = group.Letting()[0].Expression().ErrorContext()
				}
				return nil, nil, fmt.Errorf("Window Aggregates not allowed in LETTING%v.", ec)
			}
		}

		if group.Having() != nil {
			if err = collectAggregates(aggs, windowAggs, group.Having()); err != nil {
				return nil, nil, err
			}

			if len(windowAggs) > 0 {
				ec := getFirstErrorContext(windowAggs)
				if len(ec) == 0 {
					ec = group.Having().ErrorContext()
				}
				return nil, nil, fmt.Errorf("Window Aggregates not allowed in HAVING%v.", ec)
			}
		}
	}

	if node.Projection() != nil {
		if err = collectAggregates(aggs, windowAggs, node.Projection().Expressions()...); err != nil {
			return nil, nil, err
		}
	}

	if order != nil {
		allow := len(aggs) > 0

		if err = collectAggregates(aggs, windowAggs, order.Expressions()...); err != nil {
			return nil, nil, err
		}

		if !allow && group == nil && len(aggs) > 0 {
			ec := getFirstErrorContext(aggs)
			return nil, nil, fmt.Errorf("Aggregates not available for this ORDER BY%v.", ec)
		}
	}

	if group != nil && group.Letting() != nil {
		for _, agg := range aggs {
			for _, op := range agg.Operands() {
				if op != nil && dependsOnLet(op, group.Letting()) {
					return nil, nil, fmt.Errorf("Aggregate can't depend on GROUP alias or LETTING variable%v.", agg.ErrorContext())
				}
			}
		}
	}

	return sortAggregatesMap(aggs), sortWindowAggregates(windowAggs), nil
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

func collectAggregates(aggs, windowAggs map[string]algebra.Aggregate, exprs ...expression.Expression) (err error) {

	stringer := expression.NewStringer()
	for _, expr := range exprs {
		if expr == nil {
			continue
		} else if agg, ok := expr.(algebra.Aggregate); ok {
			str := stringer.Visit(agg)
			wTerm := agg.WindowTerm()
			nAggs := len(aggs)
			nWindowAggs := len(windowAggs)

			if err = collectAggregates(aggs, windowAggs, agg.Children()...); err != nil {
				return
			}

			if wTerm == nil {
				if nAggs != len(aggs) {
					return fmt.Errorf("Nested aggregates are not allowed%v.", expr.ErrorContext())
				}

				if nWindowAggs != len(windowAggs) {
					return fmt.Errorf("Window aggregates are not allowed inside Aggregates%v.", expr.ErrorContext())
				}
				aggs[str] = agg
			} else {
				if nWindowAggs != len(windowAggs) {
					return fmt.Errorf("Window aggregates are not allowed inside Window Aggregates%v.", expr.ErrorContext())
				}
				windowAggs[str] = agg
			}

			if agg.Filter() != nil {
				subqueries, err1 := expression.ListSubqueries(expression.Expressions{agg.Filter()}, false)
				if err1 != nil {
					return err1
				}
				if len(subqueries) > 0 {
					return fmt.Errorf("Subqueries are not allowed in aggregate filter%v.", agg.Filter().ErrorContext())
				}
			}
		} else if _, ok := expr.(*algebra.Subquery); !ok {
			if err = collectAggregates(aggs, windowAggs, expr.Children()...); err != nil {
				return err
			}
		}
	}

	return
}

func getFirstErrorContext(m map[string]algebra.Aggregate) string {
	ec := ""
	ml, mc := math.MaxInt32, math.MaxInt32
	for _, v := range m {
		l, c := v.GetErrorContext()
		if l < ml || (l == ml && c < mc) {
			ml, mc = l, c
			ec = v.ErrorContext()
		}
	}
	return ec
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
			first = agg.Operands()[0]
			if first == nil {
				return cond
			}

			continue
		}

		op := agg.Operands()[0]
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
	} else if base.SubsetOf(cond, constraint) {
		return cond
	} else {
		this.aggConstraint = constraint
		return expression.NewAnd(cond, constraint)
	}
}

func constrainGroupTerm(expr expression.Expression, groupKeys expression.Expressions,
	allowed *value.ScopeValue) errors.Error {
	if ok, rexpr := expr.SurvivesGrouping(groupKeys, allowed); !ok {
		if rexpr == nil {
			rexpr = expr
		}
		return errors.NewNotGroupKeyOrAggError(rexpr.String() + rexpr.ErrorContext())
	}
	return nil
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
			if aggIndexProperties == nil || !aggIndexProperties.supported ||
				(agg.Filter() != nil && !aggIndexProperties.filter) {
				this.resetPushDowns()
				return
			}

			if agg.Operands()[0] == nil {
				continue
			}

			if !agg.Operands()[0].IndexAggregatable() || dependsOnLet(agg.Operands()[0], let) {
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

func getMaxLevelOfLetBindings(bindings expression.Bindings) int {
	exprList := bindings.Expressions()
	if len(exprList) < 2 {
		return 1
	}

	level := 1
	varList := bindings.Identifiers()
	levels := make([]int, len(exprList))
	for i := range levels {
		levels[i] = 1
	}

	for i := 0; i < len(exprList); i++ {
		for j := i - 1; j >= 0; j-- {
			if exprList[i].DependsOn(varList[j]) && levels[i] <= levels[j] {
				levels[i] = levels[j] + 1
				if levels[i] > level {
					level = levels[i]
					break
				}
			}
		}
	}
	return level
}

func dereferenceLet(expr expression.Expression, inliner *expression.Inliner, level int) (expression.Expression, error) {
	for level > 0 {
		expr_new, e := inliner.Map(expr)
		if e != nil {
			return nil, e
		}
		level--
		expr = expr_new
	}
	return expr, nil
}
