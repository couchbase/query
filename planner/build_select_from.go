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
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *builder) visitFrom(node *algebra.Subselect, group *algebra.Group,
	projection *algebra.Projection, indexPushDowns *indexPushDowns) error {

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
		this.recordSubTime("keyspace.metadata", keyspaceFinder.metadataDuration)
		if err != nil {
			return err
		}
		this.pushableOnclause = keyspaceFinder.pushableOnclause
		this.collectKeyspaceNames()
		this.collectAliases(node)

		node.SetOptimHints(deriveOptimHints(this.baseKeyspaces, node.OptimHints()))
		if node.OptimHints() != nil {
			this.processOptimHints(node.OptimHints())
		}

		primKeyspace, _ := this.baseKeyspaces[primaryTerm.Alias()]
		primKeyspace.SetPrimaryTerm()

		if len(keyspaceFinder.unnestDepends) > 1 {
			// MB-38105 gather all unnest aliases for the primary keyspace
			for a, _ := range keyspaceFinder.unnestDepends {
				if a == primaryTerm.Alias() {
					continue
				}
				keyspace, _ := this.baseKeyspaces[a]
				primKeyspace.AddUnnestAlias(keyspace.Name(), keyspace.Keyspace(),
					len(keyspaceFinder.unnestDepends)-1)
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
				extraExpr, err := this.processPredicate(this.pushableOnclause, true)
				if err != nil {
					return err
				}
				if extraExpr != nil {
					constant := extraExpr.Value()
					if constant != nil {
						if constant.Truth() {
							this.pushableOnclause = nil
						} else {
							// pushable on clause behaves like where clause
							this.unsetTrueWhereClause()
							this.setFalseWhereClause()

						}
					} else {
						this.setBuilderFlag(BUILDER_HAS_EXTRA_FLTR)
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

				extraExpr, err := this.processPredicate(aoj2aij.pushableOnclause, true)
				if err != nil {
					return err
				}
				if extraExpr != nil && extraExpr.Value() == nil {
					this.setBuilderFlag(BUILDER_HAS_EXTRA_FLTR)
				}
			}

			if this.useCBO && this.keyspaceUseCBO(primKeyspace.Name()) && len(unnests) > 0 {
				keyspaceNames := make(map[string]string, 1)
				keyspaceNames[primKeyspace.Name()] = primKeyspace.Keyspace()
				for alias, _ := range primKeyspace.GetUnnests() {
					unnestKeyspace, _ := this.baseKeyspaces[alias]
					for _, un := range unnests {
						if un.Alias() == alias {
							for _, fl := range unnestKeyspace.Filters() {
								if !fl.IsSelecDone() {
									sel := getUnnestPredSelec(fl.FltrExpr(), alias, un.Expression(), keyspaceNames, this.advisorValidate(), this.context)
									fl.SetSelec(sel)
									fl.SetSelecDone()
								}
							}
						}
					}
				}
			}
		}

		this.extractKeyspacePredicates(this.where, nil)

		var ops, subOps []plan.Operator
		var coveringOps []plan.CoveringOperator
		var filter expression.Expression
		var hasOrder bool

		orderedHint := hasOrderedHint(node.OptimHints())
		if this.useCBO && !this.indexAdvisor && this.context.Optimizer() != nil && !orderedHint &&
			util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_JOIN_ENUMERATION) {
			var limit, offset expression.Expression
			var order *algebra.Order
			var distinct algebra.ResultTerms
			if group == nil {
				if indexPushDowns != nil {
					// use limit/offset/order from saved info since they
					// could have been reset in the builder itself
					limit = indexPushDowns.limit
					offset = indexPushDowns.offset
					order = indexPushDowns.order
				}
				if projection != nil && (projection.Distinct() || this.setOpDistinct) {
					distinct = projection.Terms()
				}
			}

			optimizer := this.context.Optimizer()
			ops, subOps, coveringOps, filter, hasOrder, err = optimizer.OptimizeQueryBlock(this.Copy(), node.From(), limit, offset, order, distinct)
			if err != nil {
				return err
			}
		}

		if len(ops) > 0 || len(subOps) > 0 {
			this.addChildren(ops...)
			this.addSubChildren(subOps...)
			this.coveringScans = append(this.coveringScans, coveringOps...)
			this.filter = filter
			if hasOrder {
				this.setBuilderFlag(BUILDER_PLAN_HAS_ORDER)
				this.resetProjection()
				this.resetIndexGroupAggs()
				this.resetOffsetLimit()
			} else {
				this.resetPushDowns()
			}
		} else {
			// Use FROM clause in index selection
			_, err = node.From().Accept(this)
			if err != nil {
				return err
			}
			// join filter hints are checked/marked after all join/nest/unnest are done
			err = this.MarkJoinFilterHints(this.children, this.subChildren)
			if err != nil {
				return err
			}
			this.CheckBitFilters(this.children, this.subChildren)
		}
		node.AddSubqueryTermHints(this.gatherSubqueryTermHints())
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

func (this *builder) collectAliases(node *algebra.Subselect) {
	this.aliases = make(map[string]bool, len(this.baseKeyspaces))
	for a, _ := range this.baseKeyspaces {
		this.aliases[a] = true
	}
	for _, b := range node.Let() {
		this.aliases[b.Variable()] = true
	}
}

func (this *builder) GetSubPaths(ksTerm *algebra.KeyspaceTerm, keyspace string) (names []string, err error) {
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
		} else {
			isSystemXattr := false
			for _, v := range names {
				if v[0] == '_' {
					isSystemXattr = true
					break
				}
			}

			// MB-51136 system xattrs access require system xattrs read privilege
			if isSystemXattr {
				ksTerm.SetExtraPrivilege(auth.PRIV_XATTRS)
			}
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

	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return nil, errors.NewPlanInternalError(fmt.Sprintf("VisitKeyspaceTerm: baseKeyspace for %s not found", node.Alias()))
	}

	if this.subquery && this.correlated {
		node.SetInCorrSubq()
		baseKeyspace.SetInCorrSubq()
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
		// or during join enumeration
		// just return nil, and let the caller consider alternatives:
		//   primary join --> use lookup join instead of nested-loop join
		//   hash join --> use nested-loop join instead of hash join
		//   join enumeration --> if no scan path, wait till join
		if node.IsPrimaryJoin() || node.IsUnderHash() || this.joinEnum() {
			return nil, nil
		} else {
			return nil, errors.NewPlanInternalError("VisitKeyspaceTerm: no plan generated")
		}
	}
	this.addChildren(scan)

	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())

	if useCBO {
		err = this.markPlanFlags(scan, node)
		if err != nil {
			return nil, err
		}
	}

	if len(this.coveringScans) == 0 && this.countScan == nil {
		names, err := this.GetSubPaths(node, node.Alias())
		if err != nil {
			return nil, err
		}

		cost := scan.Cost()
		cardinality := scan.Cardinality()
		size := scan.Size()
		frCost := scan.FrCost()
		if useCBO && (cost <= 0.0 || cardinality <= 0.0 || size <= 0 || frCost <= 0.0) {
			useCBO = false
		}

		if iscan3, ok := scan.(*plan.IndexScan3); ok {
			if iscan3.HasEarlyOrder() {
				op, err := this.buildEarlyOrder(iscan3, useCBO)
				if err != nil {
					return nil, err
				}
				if useCBO {
					cost, cardinality, size, frCost = op.Cost(), op.Cardinality(), op.Size(), op.FrCost()
					if cost <= 0.0 || cardinality <= 0.0 || size <= 0 || frCost <= 0.0 {
						useCBO = false
					}
				}
			} else if this.partialSortTermCount == 0 {
				if iscan3.HasEarlyOffset() {
					op, err := this.buildEarlyOffset(iscan3, useCBO)
					if err != nil {
						return nil, err
					}
					if useCBO {
						cost, cardinality, size, frCost = op.Cost(), op.Cardinality(), op.Size(), op.FrCost()
						if cost <= 0.0 || cardinality <= 0.0 || size <= 0 || frCost <= 0.0 {
							useCBO = false
						}
					}
				}
				if iscan3.HasEarlyLimit() {
					op, err := this.buildEarlyLimit(iscan3, useCBO)
					if err != nil {
						return nil, err
					}
					if useCBO {
						cost, cardinality, size, frCost = op.Cost(), op.Cardinality(), op.Size(), op.FrCost()
						if cost <= 0.0 || cardinality <= 0.0 || size <= 0 || frCost <= 0.0 {
							useCBO = false
						}
					}
				}
			}
		}

		if useCBO {
			fetchCost, fsize, ffrCost := OPT_COST_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
			if keyspace != nil {
				fetchCost, fsize, ffrCost = getFetchCost(keyspace.QualifiedName(), cardinality)
			}
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
		this.addChildren(plan.NewFetch(keyspace, node, names, cost, cardinality, size, frCost, node.ValidateKeys()))

		// no need to separate out the filter if the query has a single keyspace
		if len(this.baseKeyspaces) > 1 &&
			(!this.hasBuilderFlag(BUILDER_JOIN_ON_PRIMARY) || !node.IsInCorrSubq()) {

			filter, _, err := this.getFilter(node.Alias(), false, nil)
			if err != nil {
				return nil, err
			}

			if filter != nil {
				if useCBO && (cost > 0.0) && (cardinality > 0.0) && (size > 0) && (frCost > 0.0) {
					cost, cardinality, size, frCost = getFilterCost(this.lastOp,
						filter, this.baseKeyspaces, this.keyspaceNames,
						node.Alias(), this.advisorValidate(), this.context)
				}

				// Add filter as a separate Filter operator since Fetch is already
				// heavily loaded. This way the filter evaluation can happen on a
				// separate go thread and can be potentially parallelized
				this.addSubChildren(plan.NewFilter(filter, node.Alias(), cost, cardinality, size, frCost))
			}
		}
	}

	if !this.joinEnum() && !node.IsAnsiJoinOp() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
		if useCBO && this.lastOp != nil {
			baseKeyspace.SetCardinality(this.lastOp.Cardinality())
			baseKeyspace.SetSize(this.lastOp.Size())
		}
	}

	return nil, nil
}

func (this *builder) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	alias := node.Alias()
	sa := this.subquery
	this.subquery = true
	subquery := node.Subquery()
	qp, err := subquery.Accept(this)
	this.subquery = sa
	if err != nil {
		this.processadviseJF(alias)
		return nil, err
	}

	this.resetPushDowns()

	this.children = make([]plan.Operator, 0, 16)    // top-level children, executed sequentially
	this.subChildren = make([]plan.Operator, 0, 16) // sub-children, executed across data-parallel streams
	selQP := qp.(*plan.QueryPlan)
	selOp := selQP.PlanOp()
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return nil, errors.NewPlanInternalError(fmt.Sprintf("VisitSubqueryTerm: baseKeyspace for %s not found", alias))
	}

	if this.hasBuilderFlag(BUILDER_NL_INNER) {
		// make an ExpressionScan with the subquery as expression
		// also save the subquery plan such that it can be added later to "~subqueries"
		exprScan := plan.NewExpressionScan(algebra.NewSubquery(subquery), alias, subquery.IsCorrelated(), nil, selOp.Cost(), selOp.Cardinality(), selOp.Size(), selOp.FrCost())
		exprScan.SetSubqueryPlan(selOp)
		this.addChildren(exprScan)
	} else {
		this.addChildren(selOp, plan.NewAlias(alias, baseKeyspace.IsPrimaryTerm(),
			selOp.Cost(), selOp.Cardinality(), selOp.Size(), selOp.FrCost()))
	}

	if len(this.baseKeyspaces) > 1 {
		filter, _, err := this.getFilter(alias, false, nil)
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
					this.baseKeyspaces, this.keyspaceNames, alias,
					this.advisorValidate(), this.context)
			}
			this.addSubChildren(plan.NewFilter(filter, alias, cost, cardinality, size, frCost))
		}
	}

	if !this.joinEnum() && !node.IsAnsiJoinOp() {
		if !node.HasTransferJoinHint() {
			baseKeyspace.MarkJoinHintError(algebra.JOIN_HINT_FIRST_TERM + alias)
		}
		err = this.processKeyspaceDone(alias)
		if err != nil {
			return nil, err
		}
		if this.useCBO && this.lastOp != nil {
			baseKeyspace.SetCardinality(this.lastOp.Cardinality())
			baseKeyspace.SetSize(this.lastOp.Size())
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

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getExpressionScanCost(node.ExpressionTerm())
	}

	var filter expression.Expression
	var selec float64
	var err error
	alias := node.Alias()
	if len(this.baseKeyspaces) > 1 {
		filter, selec, err = this.getFilter(alias, false, nil)
		if err != nil {
			return nil, err
		}

		if this.useCBO && (filter != nil) && (cost > 0.0) && (cardinality > 0.0) &&
			(selec > 0.0) && (size > 0) && (frCost > 0.0) {
			cost, cardinality, size, frCost = getSimpleFilterCost(alias,
				cost, cardinality, selec, size, frCost)
		}
	}
	this.addChildren(plan.NewExpressionScan(node.ExpressionTerm(), alias, node.IsCorrelated(), filter, cost, cardinality, size, frCost))

	if !this.joinEnum() && !node.IsAnsiJoinOp() {
		baseKeyspace, ok := this.baseKeyspaces[alias]
		if !ok {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("VisitExpressionTerm: baseKeyspace for %s not found", alias))
		}
		if !node.HasTransferJoinHint() {
			baseKeyspace.MarkJoinHintError(algebra.JOIN_HINT_FIRST_TERM + alias)
		}
		err = this.processKeyspaceDone(alias)
		if err != nil {
			return nil, err
		}
		if this.useCBO && this.lastOp != nil {
			baseKeyspace.SetCardinality(this.lastOp.Cardinality())
			baseKeyspace.SetSize(this.lastOp.Size())
		}
	}

	return nil, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) != nil &&
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

	subPaths, err := this.GetSubPaths(right, right.Alias())
	if err != nil {
		return nil, err
	}

	err = this.markJoinIndexAllHint(right.Alias())
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
	join := plan.NewJoin(keyspace, node, subPaths, cost, cardinality, size, frCost)
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
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) != nil &&
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

	err = this.markJoinIndexAllHint(right.Alias())
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
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) != nil &&
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

	if !this.joinEnum() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) != nil && this.group == nil {
		this.resetProjection()
		this.resetIndexGroupAggs()
		if !node.Outer() {
			this.resetOffsetLimit()
		}
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

	subPaths, err := this.GetSubPaths(right, right.Alias())
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
	this.addChildren(plan.NewNest(keyspace, node, subPaths, cost, cardinality, size, frCost))

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	this.requirePrimaryKey = true
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) != nil && this.group == nil {
		this.resetProjection()
		this.resetIndexGroupAggs()
		if !node.Outer() {
			this.resetOffsetLimit()
		}
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
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) != nil && this.group == nil {
		this.resetProjection()
		this.resetIndexGroupAggs()
		if !node.Outer() {
			this.resetOffsetLimit()
		}
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

	if !this.joinEnum() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *builder) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	if algebra.GetKeyspaceTerm(node.PrimaryTerm()) == nil {
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
		err = this.buildUnnest(node)
		if err != nil {
			return nil, err
		}
	}

	if !this.joinEnum() {
		err = this.processKeyspaceDone(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (this *builder) buildUnnest(node *algebra.Unnest) error {
	filter, selec, err := this.getFilter(node.Alias(), false, nil)
	if err != nil {
		return err
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

			var unnestIndexInfo *base.UnnestIndexInfo
			if this.joinEnum() {
				for _, ks := range this.baseKeyspaces {
					if !ks.IsUnnest() {
						unnestIndexInfo = getUnnestIndexInfo(ks, node.Alias())
						if unnestIndexInfo != nil {
							break
						}
					}
				}
			} else {
				primaryTerm := algebra.GetKeyspaceTerm(node.PrimaryTerm())
				if primaryTerm != nil {
					primKeyspace, _ := this.baseKeyspaces[primaryTerm.Alias()]
					unnestIndexInfo = getUnnestIndexInfo(primKeyspace, node.Alias())
				}
			}
			if unnestIndexInfo != nil {
				idxSel := unnestIndexInfo.GetSelec()
				if idxSel > 0.0 {
					selec /= idxSel
				}
			}

			cost, cardinality, size, frCost = getSimpleFilterCost(node.Alias(),
				cost, cardinality, selec, size, frCost)
		}
	}
	this.addSubChildren(plan.NewUnnest(node, filter, cost, cardinality, size, frCost))
	this.addChildren(this.addSubchildrenParallel())

	return nil
}

func getUnnestIndexInfo(baseKeyspace *base.BaseKeyspace, alias string) *base.UnnestIndexInfo {
	for _, idxInfo := range baseKeyspace.GetUnnestIndexes() {
		if idxInfo != nil {
			for _, a := range idxInfo.GetAliases() {
				if a == alias {
					return idxInfo
				}
			}
		}
	}
	return nil
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

	baseKeyspace, duration := base.NewBaseKeyspace(from.Alias(), from.Path(), from, 1)
	this.recordSubTime("keyspace.metadata", duration)
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
	arrayKey *expression.All, covers expression.Covers, filterCovers map[*expression.Cover]value.Value,
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

	if this.useCBO && len(spans) > 0 {
		// mark index filters for seletivity calculation
		markIndexFlags(index, spans, nil, baseKeyspace)
	}

	filter, selec, err = this.getFilter(alias, false, nil)
	if err != nil {
		return nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
	}

	if filter != nil && (len(covers) > 0 || len(filterCovers) > 0) {
		coverer := expression.NewCoverer(covers, filterCovers)
		filter, err = expression.RenameAnyExpr(filter, arrayKey)
		if err == nil {
			filter, err = coverer.Map(filter)
		}
		if err != nil {
			return nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
		}
	}

	if this.useCBO {
		// clear the index flags marked above (temporary marking)
		baseKeyspace.Filters().ClearIndexFlag()
	}

	if this.useCBO && (filter != nil) && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
		(size > 0) && (frCost > 0.0) {
		cost, cardinality, size, frCost = getSimpleFilterCost(alias,
			cost, cardinality, selec, size, frCost)
	}
	return filter, cost, cardinality, size, frCost, nil
}

func (this *builder) getFilter(alias string, join bool, onclause expression.Expression) (
	expression.Expression, float64, error) {

	var err error
	baseKeyspace, _ := this.baseKeyspaces[alias]

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
				this.filter, err = expression.RemoveExpr(this.filter, origFltr)
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

func (this *builder) buildEarlyOrder(iscan3 *plan.IndexScan3, useCBO bool) (plan.Operator, error) {
	if this.order == nil || this.limit == nil {
		return nil, errors.NewPlanInternalError("buildEarlyOrder: early order without expected order and/or limit information")
	}
	earlyOrderExprs := iscan3.EarlyOrderExprs()
	if len(this.order.Terms()) != len(earlyOrderExprs) {
		return nil, errors.NewPlanInternalError("buildEarlyOrder: early order expressions mismatch")
	}

	cost := iscan3.Cost()
	cardinality := iscan3.Cardinality()
	size := iscan3.Size()
	frCost := iscan3.FrCost()
	if useCBO && (cost <= 0.0 || cardinality <= 0.0 || size <= 0 || frCost <= 0.0) {
		useCBO = false
	}

	// make a copy of this.order and change expressions to _index_key exprs
	coverer := expression.NewCoverer(iscan3.IndexKeys(), iscan3.IndexConditions())
	newTerms := make(algebra.SortTerms, len(this.order.Terms()))
	for i, term := range this.order.Terms() {
		newExpr, err := coverer.Map(earlyOrderExprs[i].Copy())
		if err != nil {
			return nil, err
		}
		newTerm := algebra.NewSortTerm(newExpr, term.DescendingExpr(), term.NullsPosExpr())
		newTerms[i] = newTerm
	}
	order := algebra.NewOrder(newTerms)
	// no need for any cost information for Limit/Offset inside Order
	var limit *plan.Limit
	var offset *plan.Offset
	if iscan3.HasEarlyLimit() {
		limit = plan.NewLimit(this.limit, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL)
		if this.offset != nil && iscan3.HasEarlyOffset() {
			offset = plan.NewOffset(this.offset, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL)
		}
		if useCBO {
			var nlimit, noffset int64
			if this.limit != nil {
				nlimit, _ = base.GetStaticInt(this.limit)
			}
			if this.offset != nil && iscan3.HasEarlyOffset() {
				noffset, _ = base.GetStaticInt(this.offset)
			}
			scost, scard, ssize, sfrCost := getSortCost(size, len(this.order.Terms()), cardinality, nlimit, noffset)
			if scost > 0.0 && scard > 0.0 && ssize > 0.0 && sfrCost > 0.0 {
				cost += scost
				cardinality = scard
				size = ssize
				frCost += sfrCost
			} else {
				useCBO = false
				cost, cardinality, size, frCost = OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
			}
		}
	}

	canSpill := util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_SPILL_TO_DISK)
	orderOp := plan.NewOrder(order, this.partialSortTermCount, offset, limit, cost, cardinality, size, frCost, true, canSpill)
	orderOp.SetEarlyOrder()
	this.addChildren(orderOp)
	this.setBuilderFlag(BUILDER_HAS_EARLY_ORDER)

	this.maxParallelism = 1

	if iscan3.HasEarlyOffset() && this.partialSortTermCount == 0 {
		offsetOp, err := this.buildEarlyOffset(orderOp, useCBO)
		this.resetLimit()
		return offsetOp, err
	}

	this.resetOffsetLimit()
	return orderOp, nil
}

func (this *builder) buildEarlyLimit(lastOp plan.Operator, useCBO bool) (
	plan.Operator, error) {

	if this.limit == nil {
		return nil, errors.NewPlanInternalError("buildEarlyLimit: early limit without expected limit information")
	}
	cost, cardinality, size, frCost := lastOp.Cost(), lastOp.Cardinality(), lastOp.Size(), lastOp.FrCost()
	if useCBO {
		nlimit, _ := base.GetStaticInt(this.limit)
		noffset := int64(-1)
		if this.offset != nil {
			noffset, _ = base.GetStaticInt(this.offset)
		}
		cost, cardinality, size, frCost = getLimitCost(lastOp, nlimit, noffset)
	}
	limitOp := plan.NewLimit(this.limit, cost, cardinality, size, frCost)
	this.addChildren(limitOp)
	return limitOp, nil
}

func (this *builder) buildEarlyOffset(lastOp plan.Operator, useCBO bool) (
	plan.Operator, error) {

	if this.offset == nil {
		return nil, errors.NewPlanInternalError("buildEarlyOffset: early offset without expected offset information")
	}
	cost, cardinality, size, frCost := lastOp.Cost(), lastOp.Cardinality(), lastOp.Size(), lastOp.FrCost()
	if useCBO {
		noffset, _ := base.GetStaticInt(this.offset)
		cost, cardinality, size, frCost = getOffsetCost(lastOp, noffset)
	}
	offsetOp := plan.NewOffset(this.offset, cost, cardinality, size, frCost)
	this.addChildren(offsetOp)
	return offsetOp, nil
}
