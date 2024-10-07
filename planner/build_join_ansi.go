//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func (this *builder) buildAnsiJoin(node *algebra.AnsiJoin) (op plan.Operator, err error) {
	op, err = this.buildAnsiJoinOp(node)
	if err != nil {
		return nil, err
	}

	if !this.joinEnum() {
		if node.Right().HasInferJoinHint() {
			if leftTerm, ok := node.Left().(algebra.SimpleFromTerm); ok {
				err = this.markOptimHints(leftTerm.Alias(), true)
				if err != nil {
					return nil, err
				}
			}
			// also mark index hints on the inner
			err = this.markOptimHints(node.Alias(), false)
			if err != nil {
				return nil, err
			}
		} else {
			err = this.markOptimHints(node.Alias(), true)
			if err != nil {
				return nil, err
			}
		}
	}

	if this.useCBO && op.Cost() > 0.0 && op.Cardinality() > 0.0 {
		// once the join is finalized, properly mark plan flags on the right-hand side
		err = this.markPlanFlags(op, node.Right())
	}

	return
}

func (this *builder) buildAnsiNest(node *algebra.AnsiNest) (op plan.Operator, err error) {
	op, err = this.buildAnsiNestOp(node)
	if err != nil {
		return nil, err
	}

	if !this.joinEnum() {
		if node.Right().HasInferJoinHint() {
			if leftTerm, ok := node.Left().(algebra.SimpleFromTerm); ok {
				err = this.markOptimHints(leftTerm.Alias(), true)
				if err != nil {
					return nil, err
				}
			}
			// also mark index hints on the inner
			err = this.markOptimHints(node.Alias(), false)
			if err != nil {
				return nil, err
			}
		} else {
			err = this.markOptimHints(node.Alias(), true)
			if err != nil {
				return nil, err
			}
		}
	}

	if this.useCBO && op.Cost() > 0.0 && op.Cardinality() > 0.0 {
		// once the join is finalized, properly mark plan flags on the right-hand side
		err = this.markPlanFlags(op, node.Right())
	}

	return
}

func (this *builder) buildAnsiJoinOp(node *algebra.AnsiJoin) (op plan.Operator, err error) {
	right := node.Right()

	if ksterm := algebra.GetKeyspaceTerm(right); ksterm != nil {
		right = ksterm
	}

	alias := right.Alias()
	useCBO := this.useCBO
	joinEnum := this.joinEnum()

	var leftBaseKeyspace *base.BaseKeyspace
	var joinHint algebra.JoinHint
	var preferHash, preferNL, inferJoinHint bool
	baseKeyspace, _ := this.baseKeyspaces[alias]
	if !joinEnum {
		joinHint = baseKeyspace.JoinHint()
		preferHash = algebra.PreferHash(joinHint)
		preferNL = algebra.PreferNL(joinHint)
		if !preferHash && !preferNL && baseKeyspace.HasJoinFilterHint() {
			preferHash = true
		}

		if node.Right().HasInferJoinHint() {
			if leftTerm, ok := node.Left().(algebra.SimpleFromTerm); ok {
				inferJoinHint = true
				leftBaseKeyspace, _ = this.baseKeyspaces[leftTerm.Alias()]
				joinHint = leftBaseKeyspace.JoinHint()
				preferHash = algebra.PreferHash(joinHint)
				preferNL = algebra.PreferNL(joinHint)
				if !preferHash && !preferNL && leftBaseKeyspace.HasJoinFilterHint() {
					preferHash = true
				}
			}
		}
	}

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		useCBO = useCBO && this.keyspaceUseCBO(alias)

		err := this.processOnclause(alias, node.Onclause(), node.Outer(), node.Pushable())
		if err != nil {
			return nil, err
		}

		this.extractKeyspacePredicates(nil, node.Onclause())

		if len(baseKeyspace.Filters()) > 0 {
			baseKeyspace.Filters().ClearPlanFlags()
		}

		filter, selec, err := this.getFilter(alias, true, node.Onclause())
		if err != nil {
			return nil, err
		}

		var hjoin *plan.HashJoin
		var buildRight bool
		var hjErr error
		var jps, hjps *joinPlannerState
		var hjOnclause expression.Expression
		jps = this.saveJoinPlannerState()
		origOnclause := node.Onclause()
		hjCost := OPT_COST_NOT_AVAIL
		nlCost := OPT_COST_NOT_AVAIL

		// When optimizer hints are specified, in case of CBO when we consider
		// both hash join and nested-loop join, if index hint error occurs we
		// remember the index hint error here and reset the flag on baseKeyspace,
		// since both hash join and nested-loop join build the scan on the inner
		// side. After we've chosen either hash join or nested-loop join, we then
		// re-set the necessary hint error flag on baseKeyspace.
		hjIndexHintError := false
		nlIndexHintError := false

		useFr := false
		if useCBO && !joinEnum && this.hasBuilderFlag(BUILDER_HAS_LIMIT) &&
			!this.hasBuilderFlag(BUILDER_HAS_GROUP|BUILDER_HAS_ORDER|BUILDER_HAS_WINDOW_AGGS) {
			useFr = true
		}

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			tryHash := true
			if useCBO && joinEnum {
				/* during join enumeration hash join is built separately */
				tryHash = false
			}
			if tryHash {
				hjoin, buildRight, hjErr = this.buildHashJoin(node, filter, selec, nil, nil, nil)
				if hjoin != nil {
					if !preferHash {
						if useCBO {
							if useFr {
								hjCost = hjoin.FrCost()
							} else {
								hjCost = hjoin.Cost()
							}
						}
						hjps = this.saveJoinPlannerState()
						hjOnclause = node.Onclause()
						if baseKeyspace.HasIndexHintError() {
							hjIndexHintError = true
							baseKeyspace.UnsetIndexHintError()
						}
					} else {
						if !joinEnum && !buildRight {
							this.resetOrder()
						}
						return hjoin, nil
					}
				}
			}
		} else {
			if preferHash {
				if inferJoinHint {
					leftBaseKeyspace.MarkHashUnavailable()
				} else {
					baseKeyspace.MarkHashUnavailable()
				}
			}
		}

		// when building hash join this.children could have been switched,
		// restore before attempting to build nested-loop join
		this.restoreJoinPlannerState(jps)
		node.SetOnclause(origOnclause)
		scans, primaryJoinKeys, newOnclause, newFilter, cost, cardinality, size, frCost, nlErr :=
			this.buildAnsiJoinScan(right, node.Onclause(), filter, node.Outer(), "join")

		if baseKeyspace.HasIndexHintError() {
			nlIndexHintError = true
			baseKeyspace.UnsetIndexHintError()
		}

		if len(scans) > 0 {
			if useCBO && !preferNL {
				if useFr {
					nlCost = frCost
				} else {
					nlCost = cost
				}
				if (hjCost > 0.0) && (nlCost > hjCost) {
					this.restoreJoinPlannerState(hjps)
					node.SetOnclause(hjOnclause)
					if hjIndexHintError {
						baseKeyspace.SetIndexHintError()
					}
					if !joinEnum && !buildRight {
						this.resetOrder()
					}
					return hjoin, nil
				}
			}

			if preferHash && !joinEnum {
				if inferJoinHint {
					leftBaseKeyspace.SetJoinHintError()
				} else {
					baseKeyspace.SetJoinHintError()
				}
			}
			if newOnclause != nil {
				node.SetOnclause(newOnclause)
			}
			if useCBO && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
				(filter != nil) && (size > 0) && (frCost > 0.0) {
				selec = this.adjustForIndexFilters(alias, origOnclause, selec)
				cost, cardinality, size, frCost = getSimpleFilterCost(alias,
					cost, cardinality, selec, size, frCost)
			}
			if nlIndexHintError {
				baseKeyspace.SetIndexHintError()
			}
			return plan.NewNLJoin(node, plan.NewSequence(scans...), newFilter, cost, cardinality, size, frCost), nil
		} else if hjoin != nil && !right.IsPrimaryJoin() && !preferNL {
			this.restoreJoinPlannerState(hjps)
			node.SetOnclause(hjOnclause)
			if hjIndexHintError {
				baseKeyspace.SetIndexHintError()
			}
			if !joinEnum && !buildRight {
				this.resetOrder()
			}
			return hjoin, nil
		}

		if !right.IsPrimaryJoin() {
			// as last resort, build primary scan as inner
			primary, newFilter, newOnclause, err := this.buildInnerPrimaryScan(right, filter, node.Onclause())
			if err != nil {
				return nil, err
			}

			if primary != nil {
				if useCBO {
					cost, cardinality, size, frCost = primary.Cost(), primary.Cardinality(), primary.Size(), primary.FrCost()
					if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
						cost, cardinality, size, frCost = getNLJoinCost(this.lastOp, primary, baseKeyspace.Filters(),
							node.Outer(), "join")
					} else {
						useCBO = false
					}

					if useCBO && (selec > 0.0) && (newFilter != nil) {
						cost, cardinality, size, frCost = getSimpleFilterCost(alias,
							cost, cardinality, selec, size, frCost)
					}
				}
				if preferHash && !joinEnum {
					if inferJoinHint {
						leftBaseKeyspace.SetJoinHintError()
					} else {
						baseKeyspace.SetJoinHintError()
					}
				}
				if newOnclause != nil {
					node.SetOnclause(newOnclause)
				}
				if nlIndexHintError {
					baseKeyspace.SetIndexHintError()
				}
				return plan.NewNLJoin(node, primary, newFilter, cost, cardinality, size, frCost), nil
			} else if hjoin != nil {
				this.restoreJoinPlannerState(hjps)
				node.SetOnclause(hjOnclause)
				if preferNL && !joinEnum {
					if inferJoinHint {
						leftBaseKeyspace.SetJoinHintError()
					} else {
						baseKeyspace.SetJoinHintError()
					}
				}
				if hjIndexHintError {
					baseKeyspace.SetIndexHintError()
				}
				if !joinEnum && !buildRight {
					this.resetOrder()
				}
				return hjoin, nil
			} else if nlErr != nil {
				return nil, nlErr
			} else if hjErr != nil {
				return nil, hjErr
			}
			return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiJoin: no plan built for %s", node.Alias()))
		}

		// put filter back to this.filter since Join cannot evaluate filter
		if filter != nil {
			if this.filter == nil {
				this.filter = filter
			} else {
				this.filter = expression.NewAnd(this.filter, filter)
			}
		}

		// if joining on primary key (meta().id) and no secondary index
		// scan is available, create a "regular" join
		keyspace, err := this.getTermKeyspace(right)
		if err != nil {
			return nil, err
		}

		// make a copy of the original KeyspaceTerm with the extra
		// primaryJoinKeys and construct a JOIN operator
		newKeyspaceTerm := algebra.NewKeyspaceTermFromPath(right.Path(), right.As(), nil, right.Indexes())
		newKeyspaceTerm.SetProperty(right.Property())
		newKeyspaceTerm.SetJoinKeys(primaryJoinKeys)

		// need to get extra filters in the ON-clause that's not the primary join filter
		onFilter, err := this.getOnclauseFilter(baseKeyspace.Filters())
		if err != nil {
			return nil, err
		}
		subPaths, err := this.GetSubPaths(right, alias)
		if err != nil {
			return nil, err
		}

		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_CARD_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
		if this.useCBO && this.keyspaceUseCBO(newKeyspaceTerm.Alias()) {
			rightKeyspace := base.GetKeyspaceName(this.baseKeyspaces, alias)
			cost, cardinality, size, frCost = getLookupJoinCost(this.lastOp, node.Outer(),
				newKeyspaceTerm, rightKeyspace)
		}
		if preferHash && !joinEnum {
			if inferJoinHint {
				leftBaseKeyspace.SetJoinHintError()
			} else {
				baseKeyspace.SetJoinHintError()
			}
		}
		if nlIndexHintError {
			baseKeyspace.SetIndexHintError()
		}
		return plan.NewJoinFromAnsi(keyspace, newKeyspaceTerm, subPaths, node.Outer(), onFilter, cost, cardinality, size,
			frCost), nil
	case *algebra.ExpressionTerm, *algebra.SubqueryTerm:
		err := this.processOnclause(alias, node.Onclause(), node.Outer(), node.Pushable())
		if err != nil {
			return nil, err
		}

		filter, selec, err := this.getFilter(alias, true, node.Onclause())
		if err != nil {
			return nil, err
		}

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			// for expression term and subquery term, consider hash join
			// even without USE HASH hint, as long as USE NL is not specified
			if !joinEnum && !preferNL {
				hjoin, _, err := this.buildHashJoin(node, filter, selec, nil, nil, nil)
				if hjoin != nil || err != nil {
					return hjoin, err
				}
			}
		} else {
			if preferHash {
				if inferJoinHint {
					leftBaseKeyspace.MarkHashUnavailable()
				} else {
					baseKeyspace.MarkHashUnavailable()
				}
			}
		}

		scans, newOnclause, cost, cardinality, size, frCost, err :=
			this.buildAnsiJoinSimpleFromTerm(right, node.Onclause(), node.Outer(), "join")
		if err != nil {
			return nil, err
		}

		if preferHash && !joinEnum {
			if inferJoinHint {
				leftBaseKeyspace.SetJoinHintError()
			} else {
				baseKeyspace.SetJoinHintError()
			}
		}
		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		if useCBO && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) && (filter != nil) &&
			(size > 0) && (frCost > 0.0) {
			cost, cardinality, size, frCost = getSimpleFilterCost(alias,
				cost, cardinality, selec, size, frCost)
		}

		return plan.NewNLJoin(node, plan.NewSequence(scans...), filter, cost, cardinality, size, frCost), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiJoin: Unexpected right-hand side node type"))
	}
}

func (this *builder) buildAnsiNestOp(node *algebra.AnsiNest) (op plan.Operator, err error) {
	right := node.Right()

	if ksterm := algebra.GetKeyspaceTerm(right); ksterm != nil {
		right = ksterm
	}

	alias := right.Alias()
	useCBO := this.useCBO
	joinEnum := this.joinEnum()

	var leftBaseKeyspace *base.BaseKeyspace
	var joinHint algebra.JoinHint
	var preferHash, preferNL, inferJoinHint bool
	baseKeyspace, _ := this.baseKeyspaces[alias]
	if !joinEnum {
		joinHint = baseKeyspace.JoinHint()
		preferHash = algebra.PreferHash(joinHint)
		preferNL = algebra.PreferNL(joinHint)
		if !preferHash && !preferNL && baseKeyspace.HasJoinFilterHint() {
			preferHash = true
		}

		if node.Right().HasInferJoinHint() {
			if leftTerm, ok := node.Left().(algebra.SimpleFromTerm); ok {
				inferJoinHint = true
				leftBaseKeyspace, _ = this.baseKeyspaces[leftTerm.Alias()]
				joinHint = leftBaseKeyspace.JoinHint()
				preferHash = algebra.PreferHash(joinHint)
				preferNL = algebra.PreferNL(joinHint)
				if !preferHash && !preferNL && leftBaseKeyspace.HasJoinFilterHint() {
					preferHash = true
				}
			}
		}
	}

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		useCBO = useCBO && this.keyspaceUseCBO(alias)

		err := this.processOnclause(alias, node.Onclause(), node.Outer(), node.Pushable())
		if err != nil {
			return nil, err
		}

		this.extractKeyspacePredicates(nil, node.Onclause())

		if len(baseKeyspace.Filters()) > 0 {
			baseKeyspace.Filters().ClearPlanFlags()
		}

		filter, selec, err := this.getFilter(alias, true, node.Onclause())
		if err != nil {
			return nil, err
		}

		var hnest *plan.HashNest
		var buildRight bool
		var hnErr error
		var jps, hjps *joinPlannerState
		var hnOnclause expression.Expression
		jps = this.saveJoinPlannerState()
		origOnclause := node.Onclause()
		hnCost := float64(OPT_COST_NOT_AVAIL)

		// When optimizer hints are specified, in case of CBO when we consider
		// both hash nest and nested-loop nest, if index hint error occurs we
		// remember the index hint error here and reset the flag on baseKeyspace,
		// since both hash nest and nested-loop nest build the scan on the inner
		// side. After we've chosen either hash nest or nested-loop nest, we then
		// re-set the necessary hint error flag on baseKeyspace.
		hjIndexHintError := false
		nlIndexHintError := false

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			tryHash := true
			if useCBO && joinEnum {
				/* during join enumeration hash join is built separately */
				tryHash = false
			}
			if tryHash {
				hnest, buildRight, hnErr = this.buildHashNest(node, filter, selec, nil, nil, nil)
				if hnest != nil {
					if !preferHash {
						if useCBO {
							hnCost = hnest.Cost()
						}
						hjps = this.saveJoinPlannerState()
						hnOnclause = node.Onclause()
						if baseKeyspace.HasIndexHintError() {
							hjIndexHintError = true
							baseKeyspace.UnsetIndexHintError()
						}
					} else {
						if !joinEnum && !buildRight {
							this.resetOrder()
						}
						return hnest, nil
					}
				}
			}
		} else {
			if preferHash {
				if inferJoinHint {
					leftBaseKeyspace.MarkHashUnavailable()
				} else {
					baseKeyspace.MarkHashUnavailable()
				}
			}
		}

		// when building hash nest this.children could have been switched,
		// restore before attempting to build nested-loop nest
		this.restoreJoinPlannerState(jps)
		node.SetOnclause(origOnclause)
		scans, primaryJoinKeys, newOnclause, newFilter, cost, cardinality, size, frCost, nlErr :=
			this.buildAnsiJoinScan(right, node.Onclause(), nil, node.Outer(), "nest")

		if baseKeyspace.HasIndexHintError() {
			nlIndexHintError = true
			baseKeyspace.UnsetIndexHintError()
		}

		if len(scans) > 0 {
			if useCBO && !preferNL && (hnCost > 0.0) && (cost > hnCost) {
				this.restoreJoinPlannerState(hjps)
				node.SetOnclause(hnOnclause)
				if hjIndexHintError {
					baseKeyspace.SetIndexHintError()
				}
				if !joinEnum && !buildRight {
					this.resetOrder()
				}
				return hnest, nil
			}

			if preferHash && !joinEnum {
				if inferJoinHint {
					leftBaseKeyspace.SetJoinHintError()
				} else {
					baseKeyspace.SetJoinHintError()
				}
			}
			if newOnclause != nil {
				node.SetOnclause(newOnclause)
			}
			if useCBO && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
				(filter != nil) && (size > 0) && (frCost > 0.0) {
				selec = this.adjustForIndexFilters(alias, origOnclause, selec)
				cost, cardinality, size, frCost = getSimpleFilterCost(alias,
					cost, cardinality, selec, size, frCost)
			}
			if nlIndexHintError {
				baseKeyspace.SetIndexHintError()
			}
			return plan.NewNLNest(node, plan.NewSequence(scans...), newFilter, cost, cardinality, size, frCost), nil
		} else if hnest != nil && !right.IsPrimaryJoin() && !preferNL {
			this.restoreJoinPlannerState(hjps)
			node.SetOnclause(hnOnclause)
			if hjIndexHintError {
				baseKeyspace.SetIndexHintError()
			}
			if !joinEnum && !buildRight {
				this.resetOrder()
			}
			return hnest, nil
		}

		if !right.IsPrimaryJoin() {
			// as last resort, build primary scan as inner
			primary, newFilter, newOnclause, err := this.buildInnerPrimaryScan(right, filter, node.Onclause())
			if err != nil {
				return nil, err
			}

			if primary != nil {
				if useCBO {
					cost, cardinality, size, frCost = primary.Cost(), primary.Cardinality(), primary.Size(), primary.FrCost()
					if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
						cost, cardinality, size, frCost = getNLJoinCost(this.lastOp, primary, baseKeyspace.Filters(),
							node.Outer(), "nest")
					} else {
						useCBO = false
					}

					if useCBO && (selec > 0.0) && (newFilter != nil) {
						cost, cardinality, size, frCost = getSimpleFilterCost(alias,
							cost, cardinality, selec, size, frCost)
					}
				}
				if preferHash && !joinEnum {
					if inferJoinHint {
						leftBaseKeyspace.SetJoinHintError()
					} else {
						baseKeyspace.SetJoinHintError()
					}
				}
				if newOnclause != nil {
					node.SetOnclause(newOnclause)
				}
				if nlIndexHintError {
					baseKeyspace.SetIndexHintError()
				}
				return plan.NewNLNest(node, primary, newFilter, cost, cardinality, size, frCost), nil
			} else if hnest != nil {
				this.restoreJoinPlannerState(hjps)
				node.SetOnclause(hnOnclause)
				if preferNL && !joinEnum {
					if inferJoinHint {
						leftBaseKeyspace.SetJoinHintError()
					} else {
						baseKeyspace.SetJoinHintError()
					}
				}
				if hjIndexHintError {
					baseKeyspace.SetIndexHintError()
				}
				if !joinEnum && !buildRight {
					this.resetOrder()
				}
				return hnest, nil
			} else if nlErr != nil {
				return nil, nlErr
			} else if hnErr != nil {
				return nil, hnErr
			}
			return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiNest: no plan built for %s", node.Alias()))
		}

		// put filter back to this.filter since Join cannot evaluate filter
		if filter != nil {
			if this.filter == nil {
				this.filter = filter
			} else {
				this.filter = expression.NewAnd(this.filter, filter)
			}
		}

		// if joining on primary key (meta().id) and no secondary index
		// scan is available, create a "regular" nest
		keyspace, err := this.getTermKeyspace(right)
		if err != nil {
			return nil, err
		}
		subPaths, err := this.GetSubPaths(right, alias)
		if err != nil {
			return nil, err
		}

		// make a copy of the original KeyspaceTerm with the extra
		// primaryJoinKeys and construct a NEST operator
		newKeyspaceTerm := algebra.NewKeyspaceTermFromPath(right.Path(), right.As(), nil, right.Indexes())
		newKeyspaceTerm.SetProperty(right.Property())
		newKeyspaceTerm.SetJoinKeys(primaryJoinKeys)

		// need to get extra filters in the ON-clause that's not the primary join filter
		onFilter, err := this.getOnclauseFilter(baseKeyspace.Filters())
		if err != nil {
			return nil, err
		}

		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_CARD_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
		if this.useCBO && this.keyspaceUseCBO(newKeyspaceTerm.Alias()) {
			rightKeyspace := base.GetKeyspaceName(this.baseKeyspaces, alias)
			cost, cardinality, size, frCost = getLookupNestCost(this.lastOp, node.Outer(),
				newKeyspaceTerm, rightKeyspace)
		}
		if preferHash && !joinEnum {
			if inferJoinHint {
				leftBaseKeyspace.SetJoinHintError()
			} else {
				baseKeyspace.SetJoinHintError()
			}
		}
		if nlIndexHintError {
			baseKeyspace.SetIndexHintError()
		}
		return plan.NewNestFromAnsi(keyspace, newKeyspaceTerm, subPaths, node.Outer(),
			onFilter, cost, cardinality, size, frCost), nil
	case *algebra.ExpressionTerm, *algebra.SubqueryTerm:
		filter, selec, err := this.getFilter(alias, true, node.Onclause())
		if err != nil {
			return nil, err
		}

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			// for expression term and subquery term, consider hash join
			// even without USE HASH hint, as long as USE NL is not specified
			if !joinEnum && !preferNL {
				hnest, _, err := this.buildHashNest(node, filter, selec, nil, nil, nil)
				if hnest != nil || err != nil {
					return hnest, err
				}
			}
		} else {
			if preferHash {
				if inferJoinHint {
					leftBaseKeyspace.MarkHashUnavailable()
				} else {
					baseKeyspace.MarkHashUnavailable()
				}
			}
		}

		scans, newOnclause, cost, cardinality, size, frCost, err := this.buildAnsiJoinSimpleFromTerm(right, node.Onclause(),
			node.Outer(), "nest")
		if err != nil {
			return nil, err
		}

		if preferHash && !joinEnum {
			if inferJoinHint {
				leftBaseKeyspace.SetJoinHintError()
			} else {
				baseKeyspace.SetJoinHintError()
			}
		}
		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		if useCBO && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) &&
			(size > 0) && (frCost > 0.0) {
			cost, cardinality, size, frCost = getSimpleFilterCost(alias,
				cost, cardinality, selec, size, frCost)
		}

		return plan.NewNLNest(node, plan.NewSequence(scans...), filter, cost, cardinality, size, frCost), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiNest: Unexpected right-hand side node type"))
	}
}

func (this *builder) processOnclause(alias string, onclause expression.Expression, outer, pushable bool) (err error) {
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return errors.NewPlanInternalError(fmt.Sprintf("processOnclause: missing baseKeyspace %s", alias))
	}

	// add ON-clause if it's not already part of this.pushableOnclause
	if !this.joinEnum() && (outer || !pushable) && onclause != nil {
		_, err = ClassifyExprKeyspace(onclause, this.baseKeyspaces, this.keyspaceNames,
			alias, true, this.useCBO, this.advisorValidate(), this.context)
		if err != nil {
			return err
		}
	}

	err = CombineFilters(baseKeyspace, true)
	if err != nil {
		return err
	}

	return nil
}

func (this *builder) buildAnsiJoinScan(node *algebra.KeyspaceTerm, onclause, filter expression.Expression,
	outer bool, op string) ([]plan.Operator, expression.Expression, expression.Expression, expression.Expression, float64,
	float64, int64, float64, error) {

	children := this.children
	subChildren := this.subChildren
	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	lastOp := this.lastOp
	indexPushDowns := this.storeIndexPushDowns()
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.countScan = countScan
		this.orderScan = orderScan
		this.lastOp = lastOp
		this.restoreIndexPushDowns(indexPushDowns, true)

		if len(this.coveringScans) > 0 {
			this.coveringScans = append(coveringScans, this.coveringScans...)
		} else {
			this.coveringScans = coveringScans
		}
	}()

	this.children = make([]plan.Operator, 0, 16)
	this.subChildren = make([]plan.Operator, 0, 16)
	this.coveringScans = nil
	this.countScan = nil
	this.order = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil
	this.lastOp = nil

	var err error

	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())
	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	filters := baseKeyspace.Filters()

	// check whether joining on meta().id
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	var primaryJoinKeys expression.Expression

	if !node.IsCommaJoin() {
		for _, fltr := range filters {
			if fltr.IsOnclause() {
				if eqFltr, ok := fltr.FltrExpr().(*expression.Eq); ok {
					if eqFltr.First().EquivalentTo(id) {
						node.SetPrimaryJoin()
						fltr.SetPrimaryJoin()
						primaryJoinKeys = eqFltr.Second().Copy()
						break
					} else if eqFltr.Second().EquivalentTo(id) {
						node.SetPrimaryJoin()
						fltr.SetPrimaryJoin()
						primaryJoinKeys = eqFltr.First().Copy()
						break
					}
				} else if inFltr, ok := fltr.FltrExpr().(*expression.In); ok {
					if inFltr.First().EquivalentTo(id) {
						node.SetPrimaryJoin()
						fltr.SetPrimaryJoin()
						primaryJoinKeys = inFltr.Second().Copy()
						break
					}
				}
			}
		}
	}

	nlInner := this.setNLInner()
	_, err = node.Accept(this)
	this.restoreNLInner(nlInner)
	if err != nil {
		switch e := err.(type) {
		case errors.Error:
			if e.Code() == errors.E_NO_ANSI_JOIN &&
				baseKeyspace.DnfPred() != nil && baseKeyspace.Onclause() != nil {

				// did not find an appropriate index path using both
				// on clause and where clause filters, try using just
				// the on clause filters
				baseKeyspace.SetOnclauseOnly()
				nlInner = this.setNLInner()
				_, err = node.Accept(this)
				this.restoreNLInner(nlInner)
			}
		}

		if err != nil {
			return nil, primaryJoinKeys, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, err
		}
	}

	if len(this.subChildren) > 0 {
		this.addChildren(this.addSubchildrenParallel())
	}

	// The Accept() call above for the inner side would have marked the index flag
	// on the filters, which is necessary for cost calculations later in the function.
	// Make sure the index flag is cleared since this is temporary.
	// The index flag will be permenantly marked after we've chosen a join method.
	if useCBO && len(filters) > 0 {
		defer filters.ClearIndexFlag()
	}

	// perform cover transformation for ON-clause
	// this needs to be done here since we build plan.AnsiJoin or plan.AnsiNest
	// by the caller right after returning from this function, and the plan
	// operators gets onclause expression from algebra.AnsiJoin or algebra.AnsiNest,
	// in case the entire ON-clause is transformed into a cover() expression
	// (e.g., an ANY clause as the entire ON-clause), this transformation needs to
	// be done before we build plan.AnsiJoin or plan.AnsiNest (since the root of
	// the expression changes), otherwise the transformed onclause will not be in
	// the plan operators.

	var newFilter, newOnclause expression.Expression

	if this.joinEnum() {
		// no need to do cover transformation (will be done at the end when the final
		// plan is chosen); just set newFilter, no need to set newOnclause (will keep
		// the original onclause if newOnclause is not set).
		newFilter = filter
	} else {
		newFilter, newOnclause, primaryJoinKeys, err = this.joinCoverTransformation(coveringScans,
			this.coveringScans, filter, onclause, primaryJoinKeys, nil, nil, true)
		if err != nil {
			return nil, primaryJoinKeys, nil, nil,
				OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
		}
	}

	cost, cardinality, size, frCost := OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
	if useCBO && len(this.children) > 0 {
		cost, cardinality, size, frCost = getNLJoinCost(lastOp, this.lastOp, filters, outer, op)
	}

	return this.children, primaryJoinKeys, newOnclause, newFilter, cost, cardinality, size, frCost, nil
}

func (this *builder) buildHashJoin(node *algebra.AnsiJoin, filter expression.Expression, selec float64,
	qPlan, subPlan []plan.Operator, coveringOps []plan.CoveringOperator) (*plan.HashJoin, bool, error) {
	child, buildExprs, probeExprs, aliases, newOnclause, newFilter, buildRight, cost, cardinality, size, frCost, err :=
		this.buildHashJoinOp(node.Right(), node.Left(), node.Outer(), node.Onclause(), filter, "join", qPlan, subPlan, coveringOps)
	if err != nil || child == nil {
		// cannot do hash join
		return nil, false, err
	}
	if this.useCBO && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) && (filter != nil) &&
		(size > 0) && (frCost > 0.0) {
		selec = this.adjustForHashFilters(node.Alias(), node.Onclause(), selec)
		cost, cardinality, size, frCost = getSimpleFilterCost(node.Alias(),
			cost, cardinality, selec, size, frCost)
	}
	if newOnclause != nil {
		node.SetOnclause(newOnclause)
	}
	return plan.NewHashJoin(node, child, buildExprs, probeExprs, aliases, newFilter, cost, cardinality, size, frCost),
		buildRight, nil
}

func (this *builder) buildHashNest(node *algebra.AnsiNest, filter expression.Expression, selec float64,
	qPlan, subPlan []plan.Operator, coveringOps []plan.CoveringOperator) (*plan.HashNest, bool, error) {
	child, buildExprs, probeExprs, aliases, newOnclause, newFilter, buildRight, cost, cardinality, size, frCost, err :=
		this.buildHashJoinOp(node.Right(), node.Left(), node.Outer(), node.Onclause(), nil, "nest", qPlan, subPlan, coveringOps)
	if err != nil || child == nil {
		// cannot do hash nest
		return nil, false, err
	}
	if len(aliases) != 1 {
		return nil, false, errors.NewPlanInternalError(fmt.Sprintf("buildHashNest: multiple (%d) build aliases", len(aliases)))
	}
	if this.useCBO && (cost > 0.0) && (cardinality > 0.0) && (selec > 0.0) && (filter != nil) &&
		(size > 0) && (frCost > 0.0) {
		selec = this.adjustForHashFilters(node.Alias(), node.Onclause(), selec)
		cost, cardinality, size, frCost = getSimpleFilterCost(node.Alias(),
			cost, cardinality, selec, size, frCost)
	}
	if newOnclause != nil {
		node.SetOnclause(newOnclause)
	}
	return plan.NewHashNest(node, child, buildExprs, probeExprs, aliases[0], newFilter, cost, cardinality, size, frCost),
		buildRight, nil
}

func (this *builder) buildHashJoinOp(right algebra.SimpleFromTerm, left algebra.FromTerm, outer bool,
	onclause, filter expression.Expression, op string, qPlan, subPlan []plan.Operator,
	coveringOps []plan.CoveringOperator) (child plan.Operator, buildExprs expression.Expressions,
	probeExprs expression.Expressions, buildAliases []string,
	newOnclause, newFilter expression.Expression, buildRight bool,
	cost, cardinality float64, size int64, frCost float64, err error) {

	var ksterm *algebra.KeyspaceTerm
	var keyspace string
	var defaultBuildRight bool
	isNest := op == "nest"

	if ksterm = algebra.GetKeyspaceTerm(right); ksterm != nil {
		right = ksterm
	}

	alias := right.Alias()
	useCBO := this.useCBO

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		// if USE HASH and USE KEYS are specified together, make sure the document key
		// expressions does not reference any keyspaces, otherwise hash join cannot be
		// used.
		if right.IsLateralJoin() {
			return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, nil
		}
		useCBO = useCBO && this.keyspaceUseCBO(alias)
		keyspace = ksterm.Keyspace()
	case *algebra.ExpressionTerm:
		// hash join cannot handle expression term with any LATERAL correlated references
		if right.IsLateralJoin() {
			return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, nil
		}

		defaultBuildRight = true
	case *algebra.SubqueryTerm:
		// hash join cannot handle subquery with any LATERAL correlated references
		if right.IsLateralJoin() {
			return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, nil
		}

		defaultBuildRight = true
	default:
		return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
			OPT_COST_NOT_AVAIL, errors.NewPlanInternalError(fmt.Sprintf("buildHashJoinOp: unexpected right-hand side node type"))
	}

	joinEnum := this.joinEnum()
	autoJoinFilter := joinEnum && this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER)
	baseKeyspace, _ := this.baseKeyspaces[alias]
	force := true
	inferJoinFilterHint := false
	joinHint := baseKeyspace.JoinHint()
	if right.HasInferJoinHint() ||
		(joinEnum && joinHint != algebra.USE_HASH_BUILD && joinHint != algebra.USE_HASH_PROBE) {
		if leftTerm, ok := left.(algebra.SimpleFromTerm); ok {
			leftBaseKeyspace, _ := this.baseKeyspaces[leftTerm.Alias()]
			leftJoinHint := leftBaseKeyspace.JoinHint()
			switch leftJoinHint {
			case algebra.USE_HASH_BUILD:
				joinHint = algebra.USE_HASH_PROBE
			case algebra.USE_HASH_PROBE:
				joinHint = algebra.USE_HASH_BUILD
			default:
				// replace if joinHint from right-hand side is not set
				// can just assign directly since no build/probe side is specified
				//
				// left-hand side      right-hand side
				// USE_HASH_EITHER --> USE_HASH_EITHER
				// USE_NL          --> USE_NL
				// NO_USE_HASH     --> NO_USE_HASH
				// NO_USE_NL       --> NO_USE_NL
				// JOIN_HINT_NONE  --> JOIN_HINT_NONE
				if joinHint == algebra.JOIN_HINT_NONE {
					joinHint = leftJoinHint
				}
			}
			inferJoinFilterHint = leftBaseKeyspace.HasJoinFilterHint()
		}
	}

	if joinHint == algebra.USE_HASH_BUILD {
		buildRight = true
	} else if joinHint == algebra.USE_HASH_PROBE {
		// in case of outer join, cannot build on dominant side
		// also in case of nest, can only build on right-hand-side
		if outer || isNest {
			return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, nil
		}
	} else if outer || isNest {
		// for outer join or nest, must build on right-hand side
		buildRight = true
	} else if autoJoinFilter {
		// when build side is not specified via join hint
		buildRight = true
	} else if inferJoinFilterHint {
		buildRight = true
	} else if !baseKeyspace.HasJoinFilterHint() {
		if defaultBuildRight {
			// for expression term and subquery term, if no USE HASH hint is
			// specified, then consider hash join/nest with the right-hand side
			// as build side
			buildRight = true
			force = false
		} else {
			force = false
		}
	}

	keyspaceNames := make(map[string]string, 1)
	keyspaceNames[alias] = keyspace

	filters := baseKeyspace.Filters()
	if useCBO && len(filters) > 0 {
		filters.ClearHashFlag()
	}

	// expressions for building and probing
	leftExprs := make(expression.Expressions, 0, 4)
	rightExprs := make(expression.Expressions, 0, 4)

	// look for equality join predicates
	for _, fltr := range filters {
		if !fltr.IsJoin() {
			continue
		}

		if eqFltr, ok := fltr.FltrExpr().(*expression.Eq); ok {
			first := eqFltr.First()
			second := eqFltr.Second()
			if !first.Indexable() || !second.Indexable() {
				continue
			}

			// make sure only one side of the equality predicate references
			// alias (which is right-hand-side of the join)
			firstRef := expression.HasSingleKeyspaceReference(first, alias, this.keyspaceNames)
			secondRef := expression.HasSingleKeyspaceReference(second, alias, this.keyspaceNames)

			found := false
			if firstRef && !secondRef {
				rightExprs = append(rightExprs, first.Copy())
				leftExprs = append(leftExprs, second.Copy())
				found = true
			} else if !firstRef && secondRef {
				leftExprs = append(leftExprs, first.Copy())
				rightExprs = append(rightExprs, second.Copy())
				found = true
			}

			if useCBO && found {
				if fltr.Selec() > 0.0 {
					fltr.SetHJFlag()
				} else {
					useCBO = false
				}
			}
		}
	}

	if len(leftExprs) == 0 || len(rightExprs) == 0 {
		return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
			OPT_COST_NOT_AVAIL, nil
	}

	// left hand side is already built
	if len(this.subChildren) > 0 {
		this.addChildren(this.addSubchildrenParallel())
	}

	// build right hand side

	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	lastOp := this.lastOp
	indexPushDowns := this.storeIndexPushDowns()
	defer func() {
		this.countScan = countScan
		this.orderScan = orderScan
		this.lastOp = lastOp
		this.restoreIndexPushDowns(indexPushDowns, true)

		if len(this.coveringScans) > 0 {
			this.coveringScans = append(coveringScans, this.coveringScans...)
		} else {
			this.coveringScans = coveringScans
		}
	}()

	children := this.children
	subChildren := this.subChildren

	if joinEnum {
		this.children = qPlan
		this.subChildren = subPlan
		this.coveringScans = coveringOps
		if len(subPlan) > 0 {
			this.lastOp = subPlan[len(subPlan)-1]
		} else if len(qPlan) > 0 {
			this.lastOp = qPlan[len(qPlan)-1]
		} else {
			/* should not come here */
			return nil, nil, nil, nil, nil, nil, false,
				OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL,
				errors.NewPlanInternalError("buildHashjoinOp: no plan for inner side")
		}
		_, _, err := this.getFilter(alias, true, nil)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, false,
				OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
		}
		// no need to do cover transformation (will be done at the end when the final
		// plan is chosen); just set newFilter, no need to set newOnclause (will keep
		// the original onclause if newOnclause is not set).
		newFilter = filter
	} else {
		this.coveringScans = nil
		this.countScan = nil
		this.order = nil
		this.orderScan = nil
		this.limit = nil
		this.offset = nil
		this.lastOp = nil

		this.children = make([]plan.Operator, 0, 16)
		this.subChildren = make([]plan.Operator, 0, 16)

		// Note that by this point join filters involving keyspaces that's already done planning
		// are already moved into filters and thus is available for index selection. This is ok
		// if we are doing nested-loop join. However, for hash join, since both sides of the
		// hash join are independent of each other, we cannot use join filters for index selection
		// when planning for the right-hand side.

		this.setBuilderFlag(BUILDER_UNDER_HASH)
		_, err = right.Accept(this)
		this.unsetBuilderFlag(BUILDER_UNDER_HASH)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, false,
				OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
		}

		if useCBO {
			// The Accept() call above for the inner side would have marked the index flag
			// on the filters, which is necessary for cost calculations later in the function.
			// Make sure the index flag is cleared since this is temporary.
			// The index flag will be permenantly marked after we've chosen a join method.
			if len(filters) > 0 {
				defer filters.ClearIndexFlag()
			}

			if this.lastOp != nil {
				baseKeyspace.SetCardinality(this.lastOp.Cardinality())
				baseKeyspace.SetSize(this.lastOp.Size())
			}
		}

		// if no plan generated, bail out
		if len(this.children) == 0 {
			return nil, nil, nil, nil, nil, nil, false,
				OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, nil
		}

		// perform cover transformation of leftExprs and rightExprs and onclause
		newFilter, newOnclause, _, err = this.joinCoverTransformation(coveringScans,
			this.coveringScans, filter, onclause, nil, leftExprs, rightExprs, false)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, false,
				OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, nil
		}
	}

	if !useCBO {
		cost, cardinality, size, frCost = OPT_COST_NOT_AVAIL, OPT_COST_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
	} else if !force || outer || isNest {
		// if the build/probe side is not already forced, then let the costing figure out
		// the build/probe side; if it is already forced, calculate the cost later
		// after potential bit filter determination
		var bldRight bool
		cost, cardinality, size, frCost, bldRight =
			getHashJoinCost(lastOp, this.lastOp, leftExprs, rightExprs, buildRight, force, filters, outer, op)
		if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			buildRight = bldRight
		} else {
			useCBO = false
		}
	}

	var probeAliases []string
	leftAliases := make([]string, 0, len(this.baseKeyspaces))
	for _, kspace := range this.baseKeyspaces {
		if kspace.PlanDone() && kspace.Name() != alias {
			leftAliases = append(leftAliases, kspace.Name())
		}
	}
	if buildRight {
		if len(this.subChildren) > 0 {
			this.addChildren(this.addSubchildrenParallel())
		}
		child = plan.NewSequence(this.children...)
		this.children = children
		this.subChildren = subChildren
		probeExprs = leftExprs
		buildExprs = rightExprs
		buildAliases = []string{alias}
		probeAliases = leftAliases
	} else {
		if len(subChildren) > 0 {
			children = append(children, this.addParallel(subChildren...))
		}
		child = plan.NewSequence(children...)
		buildExprs = leftExprs
		probeExprs = rightExprs
		buildAliases = leftAliases
		probeAliases = []string{alias}
		this.lastOp = this.children[len(this.children)-1]
	}

	if !outer && !isNest {
		var auto, found bool
		if joinEnum {
			auto = autoJoinFilter
		} else {
			auto = useCBO
		}

		buildInfosMap := make(map[string]*base.BuildInfo, len(buildAliases))
		for _, a := range buildAliases {
			buildInfosMap[a] = base.NewBuildInfo()
		}
		err = getBuildBFInfo(buildInfosMap, buildAliases, child)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, err
		}
		for _, a := range buildAliases {
			buildInfo := buildInfosMap[a]
			if buildInfo != nil && !buildInfo.Skip() {
				buildInfo.NewBFInfos()
			}
		}
		found, err = setProbeBitFilters(this.baseKeyspaces, probeAliases, buildAliases,
			buildInfosMap, auto, joinEnum, filters, this.children...)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL, err
		}
		if found {
			err = setBuildBitFilters(this.baseKeyspaces, buildAliases, probeAliases,
				buildInfosMap, joinEnum, child)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, false, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
					OPT_COST_NOT_AVAIL, err
			}
		}

		if force || found {
			// calculate cost
			cost, cardinality, size, frCost, _ =
				getHashJoinCost(lastOp, this.lastOp, leftExprs, rightExprs, buildRight, force, filters, outer, op)
		}

		if found {
			unmarkJoinFilters(this.baseKeyspaces, probeAliases, buildAliases, filters)
		}
	}

	return child, buildExprs, probeExprs, buildAliases, newOnclause, newFilter, buildRight, cost, cardinality, size, frCost, nil
}

func (this *builder) buildAnsiJoinSimpleFromTerm(node algebra.SimpleFromTerm, onclause expression.Expression,
	outer bool, op string) ([]plan.Operator, expression.Expression, float64, float64, int64, float64, error) {

	var newOnclause expression.Expression
	var err error

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	filters := baseKeyspace.Filters()
	if this.useCBO && len(filters) > 0 {
		filters.ClearIndexFlag()
	}

	if !this.joinEnum() {
		// perform covering transformation
		if len(this.coveringScans) > 0 {
			var exprTerm *algebra.ExpressionTerm
			var fromExpr expression.Expression

			if term, ok := node.(*algebra.ExpressionTerm); ok {
				exprTerm = term
				if exprTerm.IsCorrelated() {
					fromExpr = exprTerm.ExpressionTerm().Copy()
				}
			}

			if onclause != nil {
				newOnclause = onclause.Copy()
			}

			if newOnclause != nil || fromExpr != nil {
				for _, op := range this.coveringScans {
					coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())
					if arrayKey := op.ImplicitArrayKey(); arrayKey != nil {
						_, newOnclause, fromExpr, err =
							this.renameAnyExpression(arrayKey, nil, newOnclause, fromExpr)
					}
					if err == nil {
						_, newOnclause, fromExpr, err =
							this.coverExpression(coverer, nil, newOnclause, fromExpr)
					}
					if err != nil {
						return nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL,
							OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
					}
				}

				if exprTerm != nil && fromExpr != nil {
					exprTerm.SetExpressionTerm(fromExpr)
				}
			}
		}
	}

	children := this.children
	subChildren := this.subChildren
	lastOp := this.lastOp
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.lastOp = lastOp
	}()

	// new slices of this.children and this.subChildren are made in function
	// VisitSubqueryTerm() or VisitExpressionTerm()
	this.children = nil
	this.subChildren = nil
	this.lastOp = nil

	nlInner := this.setNLInner()
	_, err = node.Accept(this)
	this.restoreNLInner(nlInner)
	if err != nil {
		return nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, err
	}

	if len(this.subChildren) > 0 {
		this.addChildren(this.addSubchildrenParallel())
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL

	if this.useCBO {
		cost, cardinality, size, frCost = getSimpleFromTermCost(lastOp, this.lastOp, filters, outer, op)
	}

	return this.children, newOnclause, cost, cardinality, size, frCost, nil
}

const _MAX_PRIMARY_INDEX_CACHE_SIZE = 1000

func (this *builder) buildInnerPrimaryScan(right *algebra.KeyspaceTerm,
	filter, onclause expression.Expression) (
	plan.Operator, expression.Expression, expression.Expression, error) {

	docCount := this.getDocCount(right.Alias())
	if docCount > _MAX_PRIMARY_INDEX_CACHE_SIZE {
		return nil, nil, nil, nil
	}

	children := this.children
	subChildren := this.subChildren
	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	lastOp := this.lastOp
	indexPushDowns := this.storeIndexPushDowns()
	this.setBuilderFlag(BUILDER_JOIN_ON_PRIMARY)
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.countScan = countScan
		this.orderScan = orderScan
		this.lastOp = lastOp
		this.restoreIndexPushDowns(indexPushDowns, true)

		if len(this.coveringScans) > 0 {
			this.coveringScans = append(coveringScans, this.coveringScans...)
		} else {
			this.coveringScans = coveringScans
		}

		this.unsetBuilderFlag(BUILDER_JOIN_ON_PRIMARY)
	}()

	this.children = make([]plan.Operator, 0, 8)
	this.subChildren = make([]plan.Operator, 0, 8)
	this.coveringScans = nil
	this.countScan = nil
	this.order = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil
	this.lastOp = nil

	alias := right.Alias()
	useCBO := this.useCBO && this.keyspaceUseCBO(alias)
	baseKeyspace, _ := this.baseKeyspaces[alias]
	filters := baseKeyspace.Filters()

	nlinner := this.setNLInner()
	_, err := right.Accept(this)
	this.restoreNLInner(nlinner)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(this.subChildren) > 0 {
		this.addChildren(this.addSubchildrenParallel())
	}

	// The Accept() call above for the inner side would have marked the index flag
	// on the filters, which is necessary for cost calculations later in the function.
	// Make sure the index flag is cleared since this is temporary.
	// The index flag will be permenantly marked after we've chosen a join method.
	if useCBO && len(filters) > 0 {
		defer filters.ClearIndexFlag()
	}

	var newFilter, newOnclause expression.Expression
	if !this.joinEnum() {
		newFilter, newOnclause, _, err = this.joinCoverTransformation(coveringScans,
			this.coveringScans, filter, onclause, nil, nil, nil, true)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if len(this.children) > 0 {
		return plan.NewSequence(this.children...), newFilter, newOnclause, nil
	}
	return nil, nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildInnerPrimaryScan: no scan built for inner keyspace %s",
		alias))
}

func (this *builder) joinCoverTransformation(leftCoveringScans, rightCoveringScans []plan.CoveringOperator,
	filter, onclause, primaryJoinKeys expression.Expression,
	leftExprs, rightExprs expression.Expressions, nl bool) (
	newFilter, newOnclause, newPrimaryJoinKeys expression.Expression, err error) {

	if len(leftCoveringScans) == 0 && len(rightCoveringScans) == 0 {
		newFilter, newOnclause, newPrimaryJoinKeys = filter, onclause, primaryJoinKeys
		return
	}

	if filter != nil {
		newFilter = filter.Copy()
	}

	if onclause != nil {
		newOnclause = onclause.Copy()
	}

	if primaryJoinKeys != nil {
		newPrimaryJoinKeys = primaryJoinKeys.Copy()
	}

	// do right-hand-side covering index scan first, in case an ANY clause contains
	// a join filter, if part of the join filter gets transformed first, the ANY clause
	// will no longer match during transformation.
	// (note this assumes the ANY clause is on the right-hand-side keyspace)
	for _, op := range rightCoveringScans {
		coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())
		if arrayKey := op.ImplicitArrayKey(); arrayKey != nil {
			newFilter, newOnclause, newPrimaryJoinKeys, err =
				this.renameAnyExpression(arrayKey, newFilter, newOnclause, newPrimaryJoinKeys)
			if err != nil {
				return
			}
			if len(rightExprs) > 0 {
				anyRenamer := expression.NewAnyRenamer(arrayKey)
				for i, _ := range rightExprs {
					rightExprs[i], err = anyRenamer.Map(rightExprs[i])
					if err != nil {
						return
					}
				}
			}
		}

		newFilter, newOnclause, newPrimaryJoinKeys, err = this.coverExpression(coverer, newFilter, newOnclause, newPrimaryJoinKeys)
		if err != nil {
			return
		}

		for i, _ := range rightExprs {
			rightExprs[i], err = coverer.Map(rightExprs[i])
			if err != nil {
				return
			}
		}
	}

	for _, op := range leftCoveringScans {
		coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())
		if arrayKey := op.ImplicitArrayKey(); arrayKey != nil {
			newFilter, newOnclause, newPrimaryJoinKeys, err =
				this.renameAnyExpression(arrayKey, newFilter, newOnclause, newPrimaryJoinKeys)
			if err != nil {
				return
			}
			if len(leftExprs) > 0 {
				anyRenamer := expression.NewAnyRenamer(arrayKey)
				for i, _ := range leftExprs {
					leftExprs[i], err = anyRenamer.Map(leftExprs[i])
					if err != nil {
						return
					}
				}
			}
		}

		newFilter, newOnclause, newPrimaryJoinKeys, err = this.coverExpression(coverer, newFilter, newOnclause, newPrimaryJoinKeys)
		if err != nil {
			return
		}

		for i, _ := range leftExprs {
			leftExprs[i], err = coverer.Map(leftExprs[i])
			if err != nil {
				return
			}
		}

		if nl {
			// also need to perform cover transformation for index spans for
			// right-hand-side index scans since left-hand-side expressions
			// could be used as part of index spans for right-hand-side index scan
			// (note this.children at this point contains the plan for the inner)
			for _, child := range this.children {
				if secondary, ok := child.(plan.SecondaryScan); ok {
					err = secondary.CoverJoinSpanExpressions(coverer, op.ImplicitArrayKey())
					if err != nil {
						return
					}
				}
			}
		}
	}

	return
}

func (this *builder) markPlanFlags(op plan.Operator, term algebra.SimpleFromTerm) error {
	if op == nil || term == nil {
		s := ""
		if op == nil {
			s += "op == nil"
		}
		if term == nil {
			if len(s) > 0 {
				s += " "
			}
			s += "term == nil"
		}
		return errors.NewPlanInternalError(fmt.Sprintf("markPlanFlags: invalid arguments %s", s))
	}

	if op.Cost() <= 0.0 || op.Cardinality() <= 0.0 {
		return nil
	}

	ksterm := algebra.GetKeyspaceTerm(term)
	if ksterm == nil {
		// nothing to do
		return nil
	}

	alias := ksterm.Alias()
	baseKeyspace, _ := this.baseKeyspaces[alias]
	filters := baseKeyspace.Filters()
	if len(filters) > 0 {
		filters.ClearIndexFlag()
	}
	var children []plan.Operator

	switch op := op.(type) {
	case *plan.Join, *plan.Nest:
		// nothing to do
		return nil
	case *plan.NLJoin:
		// expect the child to be a sequence operator
		if seq, ok := op.Child().(*plan.Sequence); ok {
			children = seq.Children()
		}
		if len(filters) > 0 {
			filters.ClearHashFlag()
		}
	case *plan.NLNest:
		// expect the child to be a sequence operator
		if seq, ok := op.Child().(*plan.Sequence); ok {
			children = seq.Children()
		}
		if len(filters) > 0 {
			filters.ClearHashFlag()
		}
	case *plan.HashJoin:
		buildRight := false
		for _, ba := range op.BuildAliases() {
			if ba == alias {
				buildRight = true
				break
			}
		}
		if buildRight {
			// expect the child to be a sequence operator
			if seq, ok := op.Child().(*plan.Sequence); ok {
				children = seq.Children()
			}
		} else {
			children = this.children
		}
	case *plan.HashNest:
		if op.BuildAlias() == alias {
			// expect the child to be a sequence operator
			if seq, ok := op.Child().(*plan.Sequence); ok {
				children = seq.Children()
			}
		} else {
			children = this.children
		}
	case *plan.DistinctScan, *plan.IntersectScan, *plan.OrderedIntersectScan, *plan.UnionScan, *plan.IndexScan3:
		return markPlanFlagsScanOperator(baseKeyspace, op.(plan.SecondaryScan))
	case *plan.PrimaryScan3:
		// nothing to do
		return nil
	}

	if len(children) == 0 {
		return nil
	}

	return markPlanFlagsChildren(baseKeyspace, children)
}

func markPlanFlagsChildren(baseKeyspace *base.BaseKeyspace, children []plan.Operator) error {
	for _, child := range children {
		// only linear join is supported currently
		// if more complex plan shape is supported in the future, needs
		// update logic below to handle more operator types
		// (e.g. Sequence, Parallel, NLJoin, HashJoin, NLNest, HashNest, etc)
		if scan, ok := child.(plan.SecondaryScan); ok {
			// recurse to handle SecondaryScans under join/nest
			err := markPlanFlagsScanOperator(baseKeyspace, scan)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func markPlanFlagsScanOperator(baseKeyspace *base.BaseKeyspace, scan plan.SecondaryScan) error {
	switch op := scan.(type) {
	case *plan.DistinctScan:
		return markPlanFlagsSecondaryScans(baseKeyspace, op.Scan())
	case *plan.IntersectScan:
		return markPlanFlagsSecondaryScans(baseKeyspace, op.Scans()...)
	case *plan.OrderedIntersectScan:
		return markPlanFlagsSecondaryScans(baseKeyspace, op.Scans()...)
	case *plan.UnionScan:
		return markPlanFlagsSecondaryScans(baseKeyspace, op.Scans()...)
	case *plan.IndexScan3:
		return markPlanFlagsSecondaryScans(baseKeyspace, op)
	}

	return nil
}

func markPlanFlagsSecondaryScans(baseKeyspace *base.BaseKeyspace, scans ...plan.SecondaryScan) error {
	// look for index scan
	var err error
	for _, scan := range scans {
		if iscan, ok := scan.(*plan.IndexScan3); ok {
			sterm := iscan.Term()
			if sterm != nil && sterm.Alias() == baseKeyspace.Name() {
				err = markIndexFlags(iscan.Index(), iscan.Spans(), iscan.Filter(),
					baseKeyspace)
				if err != nil {
					return err
				}
			}
		} else if sscan, ok := scan.(plan.SecondaryScan); ok {
			err = markPlanFlagsScanOperator(baseKeyspace, sscan)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func markIndexFlags(index datastore.Index, spans plan.Spans2, filter expression.Expression,
	baseKeyspace *base.BaseKeyspace) error {
	var err error
	var keys expression.Expressions
	var condition expression.Expression

	alias := baseKeyspace.Name()
	if index.IsPrimary() {
		keys = expression.Expressions{expression.NewMeta(expression.NewIdentifier(alias))}
	} else {
		formalizer := expression.NewSelfFormalizer(alias, nil)
		if index.Condition() != nil {
			formalizer.SetIndexScope()
			condition, err = formalizer.Map(index.Condition().Copy())
			formalizer.ClearIndexScope()
			if err != nil {
				return err
			}
		}
		keys = expression.GetFlattenKeys(index.RangeKey()).Copy()
		for i, key := range keys {
			formalizer.SetIndexScope()
			key, err = formalizer.Map(key.Copy())
			formalizer.ClearIndexScope()
			if err != nil {
				return err
			}
			keys[i] = key
		}
	}

	unnestAliases := baseKeyspace.GetUnnestIndexAliases(index)

	optMarkIndexFilters(keys, spans, condition, filter, unnestAliases, baseKeyspace)

	return nil
}

func (this *builder) getOnclauseFilter(filters base.Filters) (expression.Expression, error) {
	terms := make(expression.Expressions, 0, len(filters))
	for _, fltr := range filters {
		if fltr.IsOnclause() && !fltr.IsPrimaryJoin() {
			terms = append(terms, fltr.FltrExpr())
		}
	}
	var filter expression.Expression
	var err error
	if len(terms) == 0 {
		return nil, nil
	} else if len(terms) == 1 {
		filter = terms[0]
	} else {
		filter = expression.NewAnd(terms...)
	}
	if this.joinEnum() {
		return filter, nil
	}
	if len(this.coveringScans) > 0 {
		filter = filter.Copy()
	}
	for _, op := range this.coveringScans {
		coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())
		if arrayKey := op.ImplicitArrayKey(); arrayKey != nil {
			filter, _, _, err = this.renameAnyExpression(arrayKey, filter, nil, nil)
		}
		if err == nil {
			filter, _, _, err = this.coverExpression(coverer, filter, nil, nil)
		}
		if err != nil {
			return nil, err
		}
	}
	return filter, nil
}

// if both nested-loop join and hash join are to be attempted (in case of CBO),
// need to save/restore certain planner states in between consideration of
// the two join methods
type joinPlannerState struct {
	children      []plan.Operator
	subChildren   []plan.Operator
	coveringScans []plan.CoveringOperator
	lastOp        plan.Operator
	filter        expression.Expression
}

func (this *builder) saveJoinPlannerState() *joinPlannerState {
	return &joinPlannerState{
		children:      this.children,
		subChildren:   this.subChildren,
		coveringScans: this.coveringScans,
		lastOp:        this.lastOp,
		filter:        this.filter,
	}
}

func (this *builder) restoreJoinPlannerState(jps *joinPlannerState) {
	this.children = jps.children
	this.subChildren = jps.subChildren
	this.coveringScans = jps.coveringScans
	this.lastOp = jps.lastOp
	this.filter = jps.filter
}

func hasAlias(alias string, aliases []string) bool {
	for _, a := range aliases {
		if alias == a {
			return true
		}
	}
	return false
}

func setProbeBitFilters(baseKeyspaces map[string]*base.BaseKeyspace,
	probeAliases, buildAliases []string, buildInfosMap map[string]*base.BuildInfo,
	auto, joinEnum bool, filters base.Filters, ops ...plan.Operator) (found bool, err error) {

	for i := len(ops) - 1; i >= 0; i-- {
		switch op := ops[i].(type) {
		case *plan.IndexScan3:
			alias := op.Term().Alias()
			baseKeyspace, ok := baseKeyspaces[alias]
			if !ok {
				return false, errors.NewPlanInternalError(fmt.Sprintf("setProbeBitFilters: baseKeyspace for %s not found", alias))
			}
			if !baseKeyspace.HasNoJoinFilterHint() &&
				(auto || baseKeyspace.HasJoinFilterHint()) && !op.Covering() &&
				hasAlias(alias, probeAliases) && len(op.IndexKeys()) > 0 {
				coverer := expression.NewCoverer(op.IndexKeys(), op.IndexConditions())
				for a, binfo := range buildInfosMap {
					if binfo.Skip() {
						continue
					}
					buildBFInfos := binfo.BFInfos()
					bfSource := baseKeyspace.GetBFSource(a)
					if bfSource == nil {
						continue
					}
					index := op.Index()
					probeExprs := bfSource.GetIndexExprs(index, alias, buildBFInfos)
					if len(probeExprs) > 0 {
						found = true
						markJoinFilters(bfSource, index, filters)
						coverExprs := make(expression.Expressions, 0, len(probeExprs))
						for _, exp := range probeExprs {
							coverExpr, err := coverer.Map(exp.Copy())
							if err != nil {
								return false, err
							}
							coverExprs = append(coverExprs, coverExpr)
						}
						probeIdxExprs := []*plan.BitFilterIndex{plan.NewBitFilterIndex(index, coverExprs)}
						dups, err := op.SetProbeBitFilters(a, probeIdxExprs)
						if err != nil {
							return false, err
						}
						// it's ok to have duplicated probe bit filters,
						// since during join enumeration, at different stages
						// we may end up generating the same information
						// multiple times (the plans may be reused)
						if !joinEnum && len(dups) > 0 && dups[0] {
							delete(buildBFInfos, index)
							if len(buildBFInfos) == 0 {
								binfo.SetSkip()
							}
						}
					}
				}
			}
		case *plan.DistinctScan:
			found, err = setScanProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
				op.Scan())
		case *plan.IntersectScan:
			found, err = setScanProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
				op.Scans()...)
		case *plan.OrderedIntersectScan:
			found, err = setScanProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
				op.Scans()...)
		case *plan.UnionScan:
			found, err = setScanProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
				op.Scans()...)
		case *plan.Parallel:
			found, err = setProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
				op.Child())
		case *plan.Sequence:
			found, err = setProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
				op.Children()...)
		case *plan.HashJoin:
			if !op.Outer() {
				found, err = setProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters,
					op.Child())
			}
		case *plan.Alias:
			// skip next term (SubqueryTerm)
			i--
		}
		// do not traverse down inner side of NL join/nest (accessed repeatedly)
		// do not traverse down inner side of Hash Nest either (becoming arrays)
		if err != nil {
			return
		}
	}
	return
}

func setScanProbeBitFilters(baseKeyspaces map[string]*base.BaseKeyspace,
	probeAliases, buildAliases []string, buildInfosMap map[string]*base.BuildInfo,
	auto, joinEnum bool, filters base.Filters, scans ...plan.SecondaryScan) (found bool, err error) {

	var f bool
	for _, scan := range scans {
		f, err = setProbeBitFilters(baseKeyspaces, probeAliases, buildAliases, buildInfosMap, auto, joinEnum, filters, scan)
		found = found || f
		if err != nil {
			return
		}
	}
	return
}

func markJoinFilters(bfSource *base.BFSource, index datastore.Index, filters base.Filters) {
	joinKeyspace := bfSource.JoinKeyspace()
	if joinKeyspace == nil {
		return
	}
	cardinality := joinKeyspace.Cardinality()
	if cardinality <= 0.0 {
		return
	}
	bfInfos := bfSource.BFInfos()
	for _, bfInfo := range bfInfos {
		if bfInfo.HasIndex(index) {
			fltrExpr := bfInfo.Filter().FltrExpr()
			// find the "equivalent" filter in filters
			for _, fltr := range filters {
				if fltr.FltrExpr().EquivalentTo(fltrExpr) {
					selec := fltr.Selec()
					if selec > 0.0 {
						fltr.SetAdjustedBitSelec()
						fltr.SetAdjSelec(selec / optGetJoinFilterSelec(selec, cardinality))
					}
				}
			}
		}
	}
}

func unmarkJoinFilters(baseKeyspaces map[string]*base.BaseKeyspace,
	probeAliases, buildAliases []string, filters base.Filters) {
	for _, alias := range probeAliases {
		baseKeyspace, _ := baseKeyspaces[alias]
		for _, a := range buildAliases {
			bfSource := baseKeyspace.GetBFSource(a)
			if bfSource == nil {
				continue
			}
			bfInfos := bfSource.BFInfos()
			for _, bfInfo := range bfInfos {
				fltrExpr := bfInfo.Filter().FltrExpr()
				// find the "equivalent" filter in filters
				for _, fltr := range filters {
					if fltr.FltrExpr().EquivalentTo(fltrExpr) {
						fltr.UnsetAdjustedBitSelec()
					}
				}
			}
		}
	}
}

func setBuildBitFilters(baseKeyspaces map[string]*base.BaseKeyspace,
	buildAliases, probeAliases []string, buildInfosMap map[string]*base.BuildInfo,
	joinEnum bool, ops ...plan.Operator) (err error) {

	for i := len(ops) - 1; i >= 0; i-- {
		switch op := ops[i].(type) {
		case *plan.IndexScan3:
			alias := op.Term().Alias()
			if hasAlias(alias, buildAliases) && op.Covering() {
				coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())
				buildInfo, _ := buildInfosMap[alias]
				if buildInfo == nil || buildInfo.Skip() {
					break
				}
				buildBFInfos := buildInfo.BFInfos()
				for _, a := range probeAliases {
					buildBFIndexes := base.GetBFInfoExprs(a, buildBFInfos)
					if len(buildBFIndexes) > 0 {
						for _, bfIndex := range buildBFIndexes {
							exps := bfIndex.Expressions()
							coverExprs := make(expression.Expressions, 0, len(exps))
							for _, exp := range exps {
								coverExpr, err := coverer.Map(exp.Copy())
								if err != nil {
									return err
								}
								coverExprs = append(coverExprs, coverExpr)
							}
							bfIndex.SetExpressions(coverExprs)
							size := optBuildBitFilterSize(baseKeyspaces[alias], exps)
							bfIndex.SetSize(size)
						}
						dups, err := op.SetBuildBitFilters(a, buildBFIndexes)
						if err != nil {
							return err
						}
						// it's ok to have duplicated build bit filters,
						// since during join enumeration, at different stages
						// we may end up generating the same information
						// multiple times (the plans may be reused)
						if !joinEnum && len(dups) > 0 && dups[0] {
							return errors.NewPlanInternalError(fmt.Sprintf("setBuildBitFilters: duplicated bit filter detected "+
								"for alias %s", a))
						}
					}
				}
			}
		case *plan.DistinctScan:
			err = setScanBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Scan())
		case *plan.IntersectScan:
			err = setScanBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Scans()...)
		case *plan.OrderedIntersectScan:
			err = setScanBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Scans()...)
		case *plan.UnionScan:
			err = setScanBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Scans()...)
		case *plan.ExpressionScan:
			alias := op.Alias()
			if hasAlias(alias, buildAliases) {
				return setBuildFilterBaseFilters(probeAliases, op.GetBuildFilterBase(), buildInfosMap[alias],
					baseKeyspaces[alias], joinEnum)
			}
		case *plan.Unnest:
			alias := op.Alias()
			if hasAlias(alias, buildAliases) {
				return setBuildFilterBaseFilters(probeAliases, op.GetBuildFilterBase(), buildInfosMap[alias],
					baseKeyspaces[alias], joinEnum)
			}
		case *plan.Filter:
			alias := op.Alias()
			if hasAlias(alias, buildAliases) {
				return setBuildFilterBaseFilters(probeAliases, op.GetBuildFilterBase(), buildInfosMap[alias],
					baseKeyspaces[alias], joinEnum)
			}
		case *plan.Parallel:
			err = setBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Child())
		case *plan.Sequence:
			err = setBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Children()...)
		case *plan.HashJoin:
			if !op.Outer() {
				err = setBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, op.Child())
			}
		case *plan.Alias:
			alias := op.Alias()
			if hasAlias(alias, buildAliases) {
				return setBuildFilterBaseFilters(probeAliases, op.GetBuildFilterBase(), buildInfosMap[alias],
					baseKeyspaces[alias], joinEnum)
			}
			// skip next term (SubqueryTerm)
			i--
		}
		// do not traverse down inner side of NL join/nest (accessed repeatedly)
		// do not traverse down inner side of Hash Nest either (becoming arrays)
		if err != nil {
			return
		}
	}
	return
}

func setScanBuildBitFilters(baseKeyspaces map[string]*base.BaseKeyspace,
	buildAliases, probeAliases []string, buildInfosMap map[string]*base.BuildInfo,
	joinEnum bool, scans ...plan.SecondaryScan) (err error) {

	for _, scan := range scans {
		err = setBuildBitFilters(baseKeyspaces, buildAliases, probeAliases, buildInfosMap, joinEnum, scan)
		if err != nil {
			return
		}
	}
	return
}

func setBuildFilterBaseFilters(probeAliases []string, op *plan.BuildBitFilterBase,
	buildInfo *base.BuildInfo, baseKeyspace *base.BaseKeyspace, joinEnum bool) (err error) {

	if buildInfo == nil || buildInfo.Skip() {
		return
	}
	buildBFInfos := buildInfo.BFInfos()
	for _, a := range probeAliases {
		buildBFIndexes := base.GetBFInfoExprs(a, buildBFInfos)
		if len(buildBFIndexes) > 0 {
			for _, bf := range buildBFIndexes {
				size := optBuildBitFilterSize(baseKeyspace, bf.Expressions())
				bf.SetSize(size)
			}
			dups, err := op.SetBuildBitFilters(a, buildBFIndexes)
			if err != nil {
				return err
			}
			// it's ok to have duplicated build bit filters,
			// since during join enumeration, at different stages
			// we may end up generating the same information
			// multiple times (the plans may be reused)
			if !joinEnum && len(dups) > 0 && dups[0] {
				return errors.NewPlanInternalError(fmt.Sprintf("setBuildFilterBaseFilters: duplicated bit filter detected for "+
					"alias %s", a))
			}
		}
	}
	return
}

// gather information from build side
func getBuildBFInfo(buildInfosMap map[string]*base.BuildInfo,
	buildAliases []string, ops ...plan.Operator) (err error) {

	for i := len(ops) - 1; i >= 0; i-- {
		switch op := ops[i].(type) {
		case *plan.NLJoin:
			err = skipBuildAlias(buildInfosMap, op.Alias())
		case *plan.NLNest:
			err = skipBuildAlias(buildInfosMap, op.Alias())
		case *plan.HashNest:
			err = skipBuildAlias(buildInfosMap, op.BuildAlias())
		case *plan.HashJoin:
			if !op.Outer() {
				err = getBuildBFInfo(buildInfosMap, buildAliases, op.Child())
			} else {
				for _, a := range op.BuildAliases() {
					err = skipBuildAlias(buildInfosMap, a)
					if err != nil {
						return
					}
				}
			}
		case *plan.Sequence:
			err = getBuildBFInfo(buildInfosMap, buildAliases, op.Children()...)
		case *plan.Parallel:
			err = getBuildBFInfo(buildInfosMap, buildAliases, op.Child())
		case *plan.Alias:
			// skip next term (SubqueryTerm)
			i--
		}
		if err != nil {
			return
		}
	}
	return
}

func skipBuildAlias(buildInfosMap map[string]*base.BuildInfo, alias string) (err error) {
	buildInfo := buildInfosMap[alias]
	if buildInfo == nil {
		return errors.NewPlanInternalError(fmt.Sprintf("skipBuildAlias: map entry for %s not found", alias))
	}
	buildInfo.SetSkip()
	return nil
}

func checkJoinFilterHint(baseKeyspaces map[string]*base.BaseKeyspace, ops ...plan.Operator) (err error) {
	for i := len(ops) - 1; i >= 0; i-- {
		switch op := ops[i].(type) {
		case *plan.IndexScan3:
			alias := op.Term().Alias()
			baseKeyspace, ok := baseKeyspaces[alias]
			if !ok {
				return errors.NewPlanInternalError(fmt.Sprintf("checkJoinFilterHint: baseKeyspace for %s not found", alias))
			}
			if len(op.GetProbeBitFilters()) == 0 {
				if baseKeyspace.HasJoinFilterHint() {
					baseKeyspace.SetJoinFltrHintError()
				}
				if len(op.IndexKeys()) > 0 && op.Filter() == nil && !op.HasEarlyOrder() {
					op.ResetIndexKeys()
				}
			}
		case *plan.DistinctScan:
			err = checkScanJoinFilterHint(baseKeyspaces, op.Scan())
		case *plan.IntersectScan:
			err = checkScanJoinFilterHint(baseKeyspaces, op.Scans()...)
		case *plan.OrderedIntersectScan:
			err = checkScanJoinFilterHint(baseKeyspaces, op.Scans()...)
		case *plan.UnionScan:
			err = checkScanJoinFilterHint(baseKeyspaces, op.Scans()...)
		case *plan.Parallel:
			err = checkJoinFilterHint(baseKeyspaces, op.Child())
		case *plan.Sequence:
			err = checkJoinFilterHint(baseKeyspaces, op.Children()...)
		case *plan.HashJoin:
			err = checkJoinFilterHint(baseKeyspaces, op.Child())
		case *plan.HashNest:
			err = checkJoinFilterHint(baseKeyspaces, op.Child())
		case *plan.NLJoin:
			err = checkJoinFilterHint(baseKeyspaces, op.Child())
		case *plan.NLNest:
			err = checkJoinFilterHint(baseKeyspaces, op.Child())
		case *plan.Alias:
			// skip next term (SubqueryTerm)
			i--
		}
		if err != nil {
			return
		}
	}
	return
}

func checkScanJoinFilterHint(baseKeyspaces map[string]*base.BaseKeyspace,
	scans ...plan.SecondaryScan) (err error) {

	for _, scan := range scans {
		err = checkJoinFilterHint(baseKeyspaces, scan)
		if err != nil {
			return
		}
	}
	return
}

func checkProbeBFAliases(probeAliases map[string]map[string]bool, gather bool, ops ...plan.Operator) {
	for i := len(ops) - 1; i >= 0; i-- {
		switch op := ops[i].(type) {
		case *plan.IndexScan3:
			if !op.Covering() {
				alias := op.Term().Alias()
				buildAliases := probeAliases[alias]
				if gather && buildAliases == nil {
					buildAliases = make(map[string]bool, len(probeAliases)-1)
					probeAliases[alias] = buildAliases
				}
				for _, probeBF := range op.GetProbeBitFilters() {
					if gather {
						buildAliases[probeBF.Alias()] = true
					} else {
						checkBaseProbeBFAliases(buildAliases, op.GetProbeFilterBase())
					}
				}
			}
		case *plan.DistinctScan:
			checkScanProbeBFAliases(probeAliases, gather, op.Scan())
		case *plan.IntersectScan:
			checkScanProbeBFAliases(probeAliases, gather, op.Scans()...)
		case *plan.OrderedIntersectScan:
			checkScanProbeBFAliases(probeAliases, gather, op.Scans()...)
		case *plan.UnionScan:
			checkScanProbeBFAliases(probeAliases, gather, op.Scans()...)
		case *plan.Parallel:
			checkProbeBFAliases(probeAliases, gather, op.Child())
		case *plan.Sequence:
			checkProbeBFAliases(probeAliases, gather, op.Children()...)
		case *plan.HashJoin:
			if !op.Outer() {
				checkProbeBFAliases(probeAliases, gather, op.Child())
			}
		case *plan.Alias:
			// skip next term (SubqueryTerm)
			i--
		}
	}
}

func checkScanProbeBFAliases(probeAliases map[string]map[string]bool, gather bool, scans ...plan.SecondaryScan) {
	for _, scan := range scans {
		checkProbeBFAliases(probeAliases, gather, scan)
	}
}

func checkBaseProbeBFAliases(buildAliases map[string]bool, op *plan.ProbeBitFilterBase) {
	compact := false
	for _, probeBF := range op.GetProbeBitFilters() {
		buildAlias := probeBF.Alias()
		if _, ok := buildAliases[buildAlias]; ok {
			op.RemoveProbeBitFilter(buildAlias)
			compact = true
		}
	}
	if compact {
		op.CompactProbeBitFilters()
	}
}

func checkBuildBFAliases(probeAliases map[string]map[string]bool, ops ...plan.Operator) {
	for i := len(ops) - 1; i >= 0; i-- {
		switch op := ops[i].(type) {
		case *plan.IndexScan3:
			if op.Covering() {
				checkBaseBuildBFAliases(probeAliases, op.Term().Alias(), op.GetBuildFilterBase())
			}
		case *plan.DistinctScan:
			checkScanBuildBFAliases(probeAliases, op.Scan())
		case *plan.IntersectScan:
			checkScanBuildBFAliases(probeAliases, op.Scans()...)
		case *plan.OrderedIntersectScan:
			checkScanBuildBFAliases(probeAliases, op.Scans()...)
		case *plan.UnionScan:
			checkScanBuildBFAliases(probeAliases, op.Scans()...)
		case *plan.ExpressionScan:
			checkBaseBuildBFAliases(probeAliases, op.Alias(), op.GetBuildFilterBase())
		case *plan.Unnest:
			checkBaseBuildBFAliases(probeAliases, op.Alias(), op.GetBuildFilterBase())
		case *plan.Filter:
			checkBaseBuildBFAliases(probeAliases, op.Alias(), op.GetBuildFilterBase())
		case *plan.Parallel:
			checkBuildBFAliases(probeAliases, op.Child())
		case *plan.Sequence:
			checkBuildBFAliases(probeAliases, op.Children()...)
		case *plan.HashJoin:
			if !op.Outer() {
				checkBuildBFAliases(probeAliases, op.Child())
			}
		case *plan.Alias:
			checkBaseBuildBFAliases(probeAliases, op.Alias(), op.GetBuildFilterBase())
			// skip next term (SubqueryTerm)
			i--
		}
	}
}

func checkScanBuildBFAliases(probeAliases map[string]map[string]bool, scans ...plan.SecondaryScan) {
	for _, scan := range scans {
		checkBuildBFAliases(probeAliases, scan)
	}
}

func checkBaseBuildBFAliases(probeAliases map[string]map[string]bool, buildAlias string, op *plan.BuildBitFilterBase) {
	compact := false
	for _, buildBF := range op.GetBuildBitFilters() {
		probeAlias := buildBF.Alias()
		buildAliases := probeAliases[probeAlias]
		if _, ok := buildAliases[buildAlias]; ok {
			delete(buildAliases, buildAlias)
			if len(buildAliases) == 0 {
				delete(probeAliases, probeAlias)
			}
		} else {
			op.RemoveBuildBitFilter(probeAlias)
			compact = true
		}
	}
	if compact {
		op.CompactBuildBitFilters()
	}
}
