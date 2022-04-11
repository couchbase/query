//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *builder) visitFrom(node *algebra.Subselect, group *algebra.Group) error {
	count, err := this.fastCount(node)
	if err != nil {
		return err
	}

	if count {
		this.maxParallelism = 1
		this.resetPushDowns()
	} else if node.From() != nil {
		prevFrom := this.from
		this.from = node.From()
		defer func() { this.from = prevFrom }()

		// gather keyspace references
		this.baseKeyspaces = make(map[string]*base.BaseKeyspace, _MAP_KEYSPACE_CAP)
		primaryTerm := this.from.PrimaryTerm()
		keyspaceFinder := newKeyspaceFinder(this.baseKeyspaces, primaryTerm.Alias())
		_, err := node.From().Accept(keyspaceFinder)
		if err != nil {
			return err
		}
		this.pushableOnclause = keyspaceFinder.pushableOnclause
		this.collectKeyspaceNames()

		primKeyspace, _ := this.baseKeyspaces[primaryTerm.Alias()]
		primKeyspace.SetPrimaryTerm()

		numUnnests := 0
		for _, keyspace := range this.baseKeyspaces {
			if keyspace.IsPrimaryUnnest() {
				numUnnests++
			}
		}
		if numUnnests > 0 {
			// MB-38105 gather all unnest aliases for the primary keyspace
			for _, keyspace := range this.baseKeyspaces {
				if keyspace.IsPrimaryUnnest() {
					primKeyspace.AddUnnestAlias(keyspace.Name(), keyspace.Keyspace(), numUnnests)
				}
			}

		}

		// Process where clause and pushable on clause
		if this.where != nil {
			err = this.processWhere(this.where)
			if err != nil {
				return err
			}
		}

		if this.pushableOnclause != nil {
			if this.falseWhereClause() {
				this.pushableOnclause = nil
			} else {
				constant, err := this.processPredicate(this.pushableOnclause, true)
				if err != nil {
					return err
				}
				if constant != nil {
					if constant.Truth() {
						this.pushableOnclause = nil
					} else {
						// pushable on clause behaves like where clause
						this.unsetTrueWhereClause()
						this.setFalseWhereClause()
					}
				}
			}
		}

		// ANSI OUTER JOIN to INNER JOIN transformation
		if !this.falseWhereClause() {
			unnests := _UNNEST_POOL.Get()
			defer _UNNEST_POOL.Put(unnests)
			unnests = collectInnerUnnests(node.From(), unnests)

			aoj2aij := newAnsijoinOuterToInner(this.baseKeyspaces, unnests)
			_, err = node.From().Accept(aoj2aij)
			if err != nil {
				return err
			}

			if aoj2aij.pushableOnclause != nil {
				// process on clauses from transformed inner joins
				if this.pushableOnclause != nil {
					this.pushableOnclause = expression.NewAnd(this.pushableOnclause, aoj2aij.pushableOnclause)
				} else {
					this.pushableOnclause = aoj2aij.pushableOnclause
				}

				_, err = this.processPredicate(aoj2aij.pushableOnclause, true)
				if err != nil {
					return err
				}
			}
		}

		this.extractKeyspacePredicates(this.where, nil)

		var op plan.Operator

		if this.useCBO && this.context.Optimizer() != nil {
			optimizer := this.context.Optimizer()
			optimizer.Initialize(this.Copy())
			op, err = optimizer.OptimizeQueryBlock(node.From())
			if err != nil {
				return err
			}
		}

		if op != nil {
			this.addChildren(op)
		} else {
			// Use FROM clause in index selection
			_, err = node.From().Accept(this)
			if err != nil {
				return err
			}
		}
	} else {
		// No FROM clause
		this.resetPushDowns()
		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		frCost := OPT_COST_NOT_AVAIL
		if this.useCBO {
			cost, cardinality, size, frCost = getDummyScanCost()
		}
		this.addChildren(plan.NewDummyScan(cost, cardinality, size, frCost))
		this.maxParallelism = 1
	}

	return nil
}

func isValidXattrs(names []string) bool {
	if len(names) > 2 {
		return false
	}
	return len(names) <= 1 || (strings.HasPrefix(names[0], "$") && !strings.HasPrefix(names[1], "$")) ||
		(!strings.HasPrefix(names[0], "$") && strings.HasPrefix(names[1], "$"))
}

func (this *builder) GetSubPaths(keyspace string) (names []string, err error) {
	if this.node != nil {
		_, names = expression.XattrsNames(this.node.Expressions(), keyspace)
		if ok := isValidXattrs(names); !ok {
			return nil, errors.NewPlanInternalError("Can only retrieve virtual xattr and user xattr or virtual xattr and system xattr")
		}
		if len(names) == 0 {
			var exprs expression.Expressions
			switch node := this.node.(type) {
			case *algebra.Update:
				exprs = node.NonMutatedExpressions()
			case *algebra.Merge:
				exprs = node.NonMutatedExpressions()
			default:
				exprs = this.node.Expressions()
			}
			_, names = expression.MetaExpiration(exprs, keyspace)
		}
	}
	return names, nil
}

func (this *builder) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	node.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return nil, err
	}

	if this.subquery && this.correlated {
		node.SetInCorrSubq()
		if baseKeyspace, ok := this.baseKeyspaces[node.Alias()]; ok {
			baseKeyspace.SetInCorrSubq()
		}
	}

	scan, err := this.selectScan(keyspace, node, false)

	uncovered := len(this.coveringScans) == 0 && this.countScan == nil
	this.appendQueryInfo(scan, keyspace, node, uncovered)

	if err != nil {
		this.processadviseJF(node.Alias())
		return nil, err
	}

	if scan == nil {
		// if a primary join is being performed, or if hash join is being considered,
		// just return nil, and let the caller consider alternatives:
		//   primary join --> use lookup join instead of nested-loop join
		//   hash join --> use nested-loop join instead of hash join
		if node.IsPrimaryJoin() || node.IsUnderHash() {
			return nil, nil
		} else {
			return nil, errors.NewPlanInternalError("VisitKeyspaceTerm: no plan generated")
		}
	}
	this.addChildren(scan)

	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())

	// for inner side of ANSI JOIN/NEST, scans will be considered multiple times for different
	// join methods, wait till join is finalized before marking index filters
	if useCBO && !node.IsAnsiJoinOp() {
		err = this.markPlanFlags(scan, node)
		if err != nil {
			return nil, err
		}
	}

	if len(this.coveringScans) == 0 && this.countScan == nil {
		names, err := this.GetSubPaths(node.Alias())
		if err != nil {
			return nil, err
		}

		cost := scan.Cost()
		cardinality := scan.Cardinality()
		size := scan.Size()
		frCost := scan.FrCost()
		if useCBO && (cost > 0.0) && (cardinality > 0.0) && (size > 0) && (frCost > 0.0) {
			fetchCost, fsize, ffrCost := getFetchCost(keyspace, cardinality)
			if fetchCost > 0.0 && fsize > 0 && ffrCost > 0.0 {
				cost += fetchCost
				frCost += ffrCost
				size = fsize
			} else {
				cost = OPT_COST_NOT_AVAIL
				cardinality = OPT_CARD_NOT_AVAIL
				size = OPT_SIZE_NOT_AVAIL
				frCost = OPT_COST_NOT_AVAIL
			}
		}
		this.addChildren(plan.NewFetch(keyspace, node, names, cost, cardinality, size, frCost))

		filter, _, err := this.getFilter(node.Alias(), nil)
		if err != nil {
			return nil, err
		}

		if filter != nil {
			baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
			if useCBO && node.IsAnsiJoinOp() && len(baseKeyspace.Filters()) > 0 {
				// temporarily mark index filters for selectivity calculation
				// if this keyspace is not under a join, this step is already done above
				err = this.markPlanFlags(scan, node)
				if err != nil {
					return nil, err
				}
			}

			if useCBO && (cost > 0.0) && (cardinality > 0.0) && (size > 0) && (frCost > 0.0) {
				cost, cardinality, size, frCost = getFilterCost(this.lastOp, filter,
					this.baseKeyspaces, this.keyspaceNames, node.Alias(),
					this.advisorValidate(), this.context)
			}

			// Add filter as a separate Filter operator since Fetch is already
			// heavily loaded. This way the filter evaluation can happen on a
			// separate go thread and can be potentially parallelized
			this.addSubChildren(plan.NewFilter(filter, cost, cardinality, size, frCost))

			if useCBO && node.IsAnsiJoinOp() && len(baseKeyspace.Filters()) > 0 {
				// clear temporary index flags
				baseKeyspace.Filters().ClearIndexFlag()
			}
		}
	}

	if !node.IsAnsiJoinOp() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *builder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	sel, err := node.Subquery().Accept(this)
	if err != nil {
		this.processadviseJF(node.Alias())
		return nil, err
	}

	this.resetPushDowns()

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams
	selOp := sel.(plan.Operator)
	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return nil, errors.NewPlanInternalError("VisitSubqueryTerm: baseKeyspace not found for " + node.Alias())
	}
	this.addChildren(selOp, plan.NewAlias(node.Alias(), baseKeyspace.IsPrimaryTerm(),
		selOp.Cost(), selOp.Cardinality(), selOp.Size(), selOp.FrCost()))

	filter, _, err := this.getFilter(node.Alias(), nil)
	if err != nil {
		return nil, err
	}

	if filter != nil {
		// use a Filter operator if there are filters on the subquery term
		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		frCost := OPT_COST_NOT_AVAIL
		if this.useCBO {
			cost, cardinality, size, frCost = getFilterCost(this.lastOp, filter,
				this.baseKeyspaces, this.keyspaceNames, node.Alias(),
				this.advisorValidate(), this.context)
		}
		this.addSubChildren(plan.NewFilter(filter, cost, cardinality, size, frCost))
	}

	if !node.IsAnsiJoinOp() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *builder) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}

	this.resetPushDowns()

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams

	filter, selec, err := this.getFilter(node.Alias(), nil)
	if err != nil {
		return nil, err
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getExpressionScanCost(node.ExpressionTerm())
		if (filter != nil) && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
			(size > 0) && (frCost > 0.0) {
			cost, cardinality, size, frCost = getSimpleFilterCost(node.Alias(),
				cost, cardinality, selec, size, frCost)
		}
	}
	this.addChildren(plan.NewExpressionScan(node.ExpressionTerm(), node.Alias(), node.IsCorrelated(), filter, cost, cardinality, size, frCost))

	if !node.IsAnsiJoinOp() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); ok && term.IsKeyspace() &&
		this.group == nil {
		this.resetProjection()
		this.resetIndexGroupAggs()
		this.resetOffsetLimit()
	} else {
		this.resetPushDowns()
	}

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getTermKeyspace(right)
	if err != nil {
		return nil, err
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO && this.keyspaceUseCBO(node.Alias()) {
		rightKeyspace := base.GetKeyspaceName(this.baseKeyspaces, right.Alias())
		cost, cardinality, size, frCost = getLookupJoinCost(this.lastOp, node.Outer(), right,
			rightKeyspace)
	}
	join := plan.NewJoin(keyspace, node, cost, cardinality, size, frCost)
	if len(this.subChildren) > 0 {
		this.addChildren(this.addSubchildrenParallel())
	}
	this.addChildren(join)

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	this.requirePrimaryKey = true
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); ok && term.IsKeyspace() &&
		this.group == nil {
		this.resetProjection()
		this.resetIndexGroupAggs()
		this.resetOffsetLimit()
	} else {
		this.resetPushDowns()
	}

	_, err := node.Left().Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getTermKeyspace(right)
	if err != nil {
		return nil, err
	}

	join, err := this.buildIndexJoin(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.addSubChildren(join)

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitAnsiJoin(node *algebra.AnsiJoin) (interface{}, error) {
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); ok && term.IsKeyspace() &&
		this.group == nil && node.Right().JoinHint() != algebra.USE_HASH_PROBE {
		this.resetProjection()
		this.resetIndexGroupAggs()
		this.resetOffsetLimit()
	} else {
		this.resetPushDowns()
	}

	_, err := node.Left().Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}
	join, err := this.buildAnsiJoin(node)
	if err != nil {
		this.processadviseJF(node.Alias())
		return nil, err
	}

	switch join := join.(type) {
	case *plan.NLJoin:
		this.addSubChildren(join)
	case *plan.Join, *plan.HashJoin:
		if len(this.subChildren) > 0 {
			this.addChildren(this.addSubchildrenParallel())
		}
		this.addChildren(join)
	}

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	if this.group == nil && this.hasOffsetOrLimit() && !node.Outer() {
		this.resetProjection()
		this.resetIndexGroupAggs()
		this.resetOffsetLimit()
	} else {
		this.resetPushDowns()
	}

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getTermKeyspace(right)
	if err != nil {
		return nil, err
	}

	if len(this.subChildren) > 0 {
		this.addChildren(this.addSubchildrenParallel())
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO && this.keyspaceUseCBO(node.Alias()) {
		rightKeyspace := base.GetKeyspaceName(this.baseKeyspaces, right.Alias())
		cost, cardinality, size, frCost = getLookupNestCost(this.lastOp, node.Outer(), right,
			rightKeyspace)
	}
	this.addChildren(plan.NewNest(keyspace, node, cost, cardinality, size, frCost))

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	this.requirePrimaryKey = true
	if this.group == nil && this.hasOffsetOrLimit() && !node.Outer() {
		this.resetProjection()
		this.resetIndexGroupAggs()
		this.resetOffsetLimit()
	} else {
		this.resetPushDowns()
	}

	_, err := node.Left().Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getTermKeyspace(right)
	if err != nil {
		return nil, err
	}

	nest, err := this.buildIndexNest(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.addSubChildren(nest)

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitAnsiNest(node *algebra.AnsiNest) (interface{}, error) {
	if this.group == nil && node.Right().JoinHint() != algebra.USE_HASH_PROBE &&
		this.hasOffsetOrLimit() && !node.Outer() {
		this.resetProjection()
		this.resetIndexGroupAggs()
		this.resetOffsetLimit()
	} else {
		this.resetPushDowns()
	}

	_, err := node.Left().Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}

	nest, err := this.buildAnsiNest(node)
	if err != nil {
		this.processadviseJF(node.Alias())
		return nil, err
	}

	switch nest := nest.(type) {
	case *plan.NLNest:
		this.addSubChildren(nest)
	case *plan.Nest, *plan.HashNest:
		if len(this.subChildren) > 0 {
			this.addChildren(this.addSubchildrenParallel())
		}
		this.addChildren(nest)
	}

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); !ok || !term.IsKeyspace() {
		this.resetPushDowns()
	}

	this.setUnnest()

	_, err := node.Left().Accept(this)
	if err != nil {
		this.processadviseJF(node.Alias())
		return nil, err
	}

	_, found := this.coveredUnnests[node]
	if !found {
		filter, selec, err := this.getFilter(node.Alias(), nil)
		if err != nil {
			return nil, err
		}

		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		frCost := OPT_COST_NOT_AVAIL
		if this.useCBO {
			baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
			CombineFilters(baseKeyspace, true)
			cost, cardinality, size, frCost = getUnnestCost(node, this.lastOp,
				this.baseKeyspaces, this.keyspaceNames, this.advisorValidate())
			if (filter != nil) && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
				(size > 0) && (frCost > 0.0) {
				cost, cardinality, size, frCost = getSimpleFilterCost(node.Alias(),
					cost, cardinality, selec, size, frCost)
			}
		}
		this.addSubChildren(plan.NewUnnest(node, filter, cost, cardinality, size, frCost))
		this.addChildren(this.addSubchildrenParallel())
	}

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) fastCount(node *algebra.Subselect) (bool, error) {
	if node.From() == nil ||
		(node.Where() != nil && (node.Where().Value() == nil || !node.Where().Value().Truth())) ||
		node.Group() != nil {
		return false, nil
	}

	var from *algebra.KeyspaceTerm
	switch other := node.From().(type) {
	case *algebra.KeyspaceTerm:
		from = other
	case *algebra.ExpressionTerm:
		if other.IsKeyspace() {
			from = other.KeyspaceTerm()
		} else {
			return false, nil
		}
	default:
		return false, nil
	}

	if from == nil || from.Keys() != nil {
		return false, nil
	}

	from.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getTermKeyspace(from)
	if err != nil {
		return false, err
	}

	for _, term := range node.Projection().Terms() {
		count, ok := term.Expression().(*algebra.Count)
		if !ok || count.Distinct() || count.IsWindowAggregate() || count.Filter() != nil {
			return false, nil
		}

		operand := count.Operands()[0]
		if operand != nil {
			val := operand.Value()
			if val == nil || val.Type() <= value.NULL {
				return false, nil
			}
		}
	}

	baseKeyspace := base.NewBaseKeyspace(from.Alias(), from.Path())
	if this.context.HasDeltaKeyspace(baseKeyspace.Keyspace()) {
		return false, nil
	}
	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO && this.keyspaceUseCBO(from.Alias()) {
		cost, cardinality, size, frCost = getCountScanCost()
	}
	this.addChildren(plan.NewCountScan(keyspace, from, cost, cardinality, size, frCost))
	return true, nil
}

func (this *builder) processKeyspaceDone(keyspace string) error {
	var err error
	for _, baseKeyspace := range this.baseKeyspaces {
		if baseKeyspace.PlanDone() {
			continue
		} else if keyspace == baseKeyspace.Name() {
			baseKeyspace.SetPlanDone()
			continue
		}

		err = base.MoveJoinFilters(keyspace, baseKeyspace)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *builder) resetOrderOffsetLimit() {
	this.resetOrder()
	this.resetLimit()
	this.resetOffset()
}

func (this *builder) resetOffsetLimit() {
	this.resetLimit()
	this.resetOffset()
}

func (this *builder) resetLimit() {
	this.limit = nil
}

func (this *builder) resetOffset() {
	this.offset = nil
}

func (this *builder) resetOrder() {
	this.order = nil
}

func (this *builder) hasOrderOrOffsetOrLimit() bool {
	return this.order != nil || this.offset != nil || this.limit != nil
}

func (this *builder) hasOffsetOrLimit() bool {
	return this.offset != nil || this.limit != nil
}

func (this *builder) resetProjection() {
	this.projection = nil
}

func (this *builder) resetIndexGroupAggs() {
	this.oldAggregates = false
	this.group = nil
	this.aggs = nil
	this.aggConstraint = nil
}

func (this *builder) resetPushDowns() {
	this.resetOrderOffsetLimit()
	this.resetProjection()
	this.resetIndexGroupAggs()
}

func offsetPlusLimit(offset, limit expression.Expression) expression.Expression {
	if offset != nil && limit != nil {
		if offset.Value() == nil {
			offset = expression.NewGreatest(expression.ZERO_EXPR, offset)
		}

		if limit.Value() == nil {
			limit = expression.NewGreatest(expression.ZERO_EXPR, limit)
		}

		return expression.NewAdd(limit, offset)
	} else {
		return limit
	}
}

func (this *builder) getIndexFilter(index datastore.Index, alias string, sargSpans SargSpans,
	covers expression.Covers, filterCovers map[*expression.Cover]value.Value,
	cost, cardinality float64, size int64, frCost float64) (
	expression.Expression, float64, float64, int64, float64, error) {

	var err error
	baseKeyspace, _ := this.baseKeyspaces[alias]

	// cannot do early filtering on subservient side of outer join
	if baseKeyspace.Outerlevel() > 0 {
		return nil, cost, cardinality, size, frCost, nil
	}

	var filter expression.Expression
	var selec float64
	var spans plan.Spans2
	if sargSpans != nil {
		// since we call this function only from covering index scans,
		// we expect only TermSpans
		if termSpans, ok := sargSpans.(*TermSpans); ok {
			spans = termSpans.Spans()
		}
	}

	if len(spans) > 0 {
		// mark index filters for seletivity calculation
		markIndexFlags(index, spans, baseKeyspace)
	}

	filter, selec, err = this.getFilter(alias, nil)
	if err != nil {
		return nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
	}

	if filter != nil && (len(covers) > 0 || len(filterCovers) > 0) {
		coverer := expression.NewCoverer(covers, filterCovers)
		filter, err = coverer.Map(filter)
		if err != nil {
			return nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
		}
	}

	// clear the index flags marked above (temporary marking)
	baseKeyspace.Filters().ClearIndexFlag()

	if this.useCBO && (filter != nil) && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
		(size > 0) && (frCost > 0.0) {
		cost, cardinality, size, frCost = getSimpleFilterCost(alias,
			cost, cardinality, selec, size, frCost)
	}
	return filter, cost, cardinality, size, frCost, nil
}

func (this *builder) getFilter(alias string, onclause expression.Expression) (
	expression.Expression, float64, error) {

	var err error
	baseKeyspace, _ := this.baseKeyspaces[alias]

	join := onclause != nil

	// cannot do early filtering on subservient side of outer join
	outer := baseKeyspace.Outerlevel() > 0
	if outer && !join {
		return nil, OPT_SELEC_NOT_AVAIL, nil
	}

	filters := baseKeyspace.Filters()
	terms := make(expression.Expressions, 0, len(filters))
	selec := OPT_SELEC_NOT_AVAIL
	doSelec := false
	if this.useCBO && this.keyspaceUseCBO(alias) {
		doSelec = true
		selec = float64(1.0)
	}

	for _, fl := range filters {
		// unnest filters can only be evaluated after the UNNEST operation
		// subquery filters are not pushed down
		if fl.IsUnnest() || fl.HasSubq() {
			continue
		}

		if join {
			if !fl.IsPostjoinFilter(onclause, outer) {
				continue
			}
		} else {
			if fl.IsJoin() {
				continue
			}
		}

		fltr := fl.FltrExpr()
		origFltr := fl.OrigExpr()
		if origFltr != nil {
			terms = append(terms, origFltr.Copy())
			if this.filter != nil {
				this.filter, err = expression.RemoveExpr(this.filter, fl.OrigExpr())
				if err != nil {
					return nil, OPT_SELEC_NOT_AVAIL, err
				}
			}
		} else if !base.IsDerivedExpr(fltr) {
			terms = append(terms, fltr.Copy())
		}

		if doSelec && !fl.HasPlanFlags() {
			if fl.Selec() > 0.0 {
				selec *= fl.Selec()
			} else {
				doSelec = false
				selec = OPT_SELEC_NOT_AVAIL
			}
		}
	}

	var filter expression.Expression
	if len(terms) == 1 {
		filter = terms[0]
	} else if len(terms) > 1 {
		filter = expression.NewAnd(terms...)
	}

	return filter, selec, nil
}

func (this *builder) adjustForHashFilters(alias string, onclause expression.Expression,
	selec float64) float64 {

	baseKeyspace, _ := this.baseKeyspaces[alias]
	outer := baseKeyspace.Outerlevel() > 0

	for _, fl := range baseKeyspace.Filters() {
		if fl.HasHJFlag() && (fl.Selec() > 0.0) && fl.IsPostjoinFilter(onclause, outer) {
			selec /= fl.Selec()
		}
	}

	return selec
}

func (this *builder) adjustForIndexFilters(alias string, onclause expression.Expression,
	selec float64) float64 {

	baseKeyspace, _ := this.baseKeyspaces[alias]
	outer := baseKeyspace.Outerlevel() > 0

	for _, fl := range baseKeyspace.Filters() {
		if fl.HasIndexFlag() && (fl.Selec() > 0.0) && fl.IsPostjoinFilter(onclause, outer) {
			selec /= fl.Selec()
		}
	}

	return selec
}
