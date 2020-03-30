//  Copyright (c) 2017 Couchbase, Inc.
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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

func (this *builder) buildAnsiJoin(node *algebra.AnsiJoin) (op plan.Operator, err error) {
	right := node.Right()

	if ksterm := algebra.GetKeyspaceTerm(right); ksterm != nil {
		right = ksterm
	}

	useCBO := this.useCBO

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		err := this.processOnclause(right.Alias(), node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		this.extractPredicates(nil, node.Onclause())

		var hjoin *plan.HashJoin
		var jps, hjps *joinPlannerState
		var hjOnclause expression.Expression
		jps = this.saveJoinPlannerState()
		origOnclause := node.Onclause()
		hjCost := float64(OPT_COST_NOT_AVAIL)

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			tryHash := false
			if useCBO {
				tryHash = true
			} else if right.PreferHash() {
				// only consider hash join when USE HASH hint is specified
				tryHash = true
			}
			if tryHash {
				hjoin, err = this.buildHashJoin(node)
				if err != nil && !useCBO {
					// in case of CBO, ignore error (e.g. no index found)
					// try nested-loop below
					return nil, err
				}
				if hjoin != nil {
					if useCBO && !right.PreferHash() {
						hjCost = hjoin.Cost()
						hjps = this.saveJoinPlannerState()
						hjOnclause = node.Onclause()
					} else {
						return hjoin, nil
					}
				}
			}
		}

		// when building hash join this.children could have been switched,
		// restore before attempting to build nested-loop join
		this.restoreJoinPlannerState(jps)
		node.SetOnclause(origOnclause)
		right.SetUnderNL()
		scans, primaryJoinKeys, newOnclause, cost, cardinality, err := this.buildAnsiJoinScan(right, node.Onclause())
		if err != nil {
			return nil, err
		}

		if len(scans) > 0 {
			if useCBO && (hjCost > 0.0) && (cost > hjCost) {
				this.restoreJoinPlannerState(hjps)
				node.SetOnclause(hjOnclause)
				return hjoin, nil
			}

			if newOnclause != nil {
				node.SetOnclause(newOnclause)
			}
			return plan.NewNLJoin(node, plan.NewSequence(scans...), cost, cardinality), nil
		} else if hjCost > 0.0 {
			this.restoreJoinPlannerState(hjps)
			node.SetOnclause(hjOnclause)
			return hjoin, nil
		}

		right.UnsetUnderNL()

		if !right.IsPrimaryJoin() {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiJoin: no plan built for %s", node.Alias()))
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
		return plan.NewJoinFromAnsi(keyspace, newKeyspaceTerm, node.Outer()), nil
	case *algebra.ExpressionTerm, *algebra.SubqueryTerm:
		err := this.processOnclause(right.Alias(), node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			// for expression term and subquery term, consider hash join
			// even without USE HASH hint, as long as USE NL is not specified
			if !right.PreferNL() {
				hjoin, err := this.buildHashJoin(node)
				if hjoin != nil || err != nil {
					return hjoin, err
				}
			}
		}

		scans, newOnclause, cost, cardinality, err := this.buildAnsiJoinSimpleFromTerm(right, node.Onclause())
		if err != nil {
			return nil, err
		}

		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		return plan.NewNLJoin(node, plan.NewSequence(scans...), cost, cardinality), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiJoin: Unexpected right-hand side node type"))
	}
}

func (this *builder) buildAnsiNest(node *algebra.AnsiNest) (op plan.Operator, err error) {
	right := node.Right()

	if ksterm := algebra.GetKeyspaceTerm(right); ksterm != nil {
		right = ksterm
	}

	useCBO := this.useCBO

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		err := this.processOnclause(right.Alias(), node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		this.extractPredicates(nil, node.Onclause())

		var hnest *plan.HashNest
		var jps, hjps *joinPlannerState
		var hnOnclause expression.Expression
		jps = this.saveJoinPlannerState()
		origOnclause := node.Onclause()
		hnCost := float64(OPT_COST_NOT_AVAIL)

		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			tryHash := false
			if useCBO {
				tryHash = true
			} else if right.PreferHash() {
				// only consider hash nest when USE HASH hint is specified
				tryHash = true
			}
			if tryHash {
				hnest, err = this.buildHashNest(node)
				if err != nil && !useCBO {
					// in case of CBO, ignore error (e.g. no index found)
					// try nested-loop below
					return nil, err
				}
				if hnest != nil {
					if useCBO && !right.PreferHash() {
						hnCost = hnest.Cost()
						hjps = this.saveJoinPlannerState()
						hnOnclause = node.Onclause()
					} else {
						return hnest, nil
					}
				}
			}
		}

		// when building hash nest this.children could have been switched,
		// restore before attempting to build nested-loop nest
		this.restoreJoinPlannerState(jps)
		node.SetOnclause(origOnclause)
		right.SetUnderNL()
		scans, primaryJoinKeys, newOnclause, cost, cardinality, err := this.buildAnsiJoinScan(right, node.Onclause())
		if err != nil {
			return nil, err
		}

		if len(scans) > 0 {
			if useCBO && (hnCost > 0.0) && (cost > hnCost) {
				this.restoreJoinPlannerState(hjps)
				node.SetOnclause(hnOnclause)
				return hnest, nil
			}

			if newOnclause != nil {
				node.SetOnclause(newOnclause)
			}
			return plan.NewNLNest(node, plan.NewSequence(scans...), cost, cardinality), nil
		} else if hnCost > 0.0 {
			this.restoreJoinPlannerState(hjps)
			node.SetOnclause(hnOnclause)
			return hnest, nil
		}

		right.UnsetUnderNL()

		if !right.IsPrimaryJoin() {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiNest: no plan built for %s", node.Alias()))
		}

		// if joining on primary key (meta().id) and no secondary index
		// scan is available, create a "regular" nest
		keyspace, err := this.getTermKeyspace(right)
		if err != nil {
			return nil, err
		}

		// make a copy of the original KeyspaceTerm with the extra
		// primaryJoinKeys and construct a NEST operator
		newKeyspaceTerm := algebra.NewKeyspaceTermFromPath(right.Path(), right.As(), nil, right.Indexes())
		newKeyspaceTerm.SetProperty(right.Property())
		newKeyspaceTerm.SetJoinKeys(primaryJoinKeys)
		return plan.NewNestFromAnsi(keyspace, newKeyspaceTerm, node.Outer()), nil
	case *algebra.ExpressionTerm, *algebra.SubqueryTerm:
		if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_HASH_JOIN) {
			// for expression term and subquery term, consider hash join
			// even without USE HASH hint, as long as USE NL is not specified
			if !right.PreferNL() {
				hnest, err := this.buildHashNest(node)
				if hnest != nil || err != nil {
					return hnest, err
				}
			}
		}

		scans, newOnclause, cost, cardinality, err := this.buildAnsiJoinSimpleFromTerm(right, node.Onclause())
		if err != nil {
			return nil, err
		}

		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		return plan.NewNLNest(node, plan.NewSequence(scans...), cost, cardinality), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiNest: Unexpected right-hand side node type"))
	}
}

func (this *builder) processOnclause(alias string, onclause expression.Expression, outer bool) (err error) {
	baseKeyspace, ok := this.baseKeyspaces[alias]
	if !ok {
		return errors.NewPlanInternalError(fmt.Sprintf("processOnclause: missing baseKeyspace %s", alias))
	}

	// for inner join, the following processing is already done as part of
	// this.pushableOnclause
	if outer {
		// For the keyspace as the inner of an ANSI JOIN, the processPredicate() call
		// will effectively put ON clause filters on top of WHERE clause filters
		// for each keyspace, as a result, both ON clause filters and WHERE clause
		// filters will be used for index selection for the inner keyspace, which
		// is ok for outer joins.
		// Note this will also put ON clause filters of an outer join on the outer
		// keyspace as well however since index selection for the outer keyspace
		// is already done, ON clause filters from an outer join is NOT used for
		// index selection consideration of the outer keyspace (ON-clause of an
		// inner join is used for index selection for outer keyspace, as part of
		// this.pushableOnclause).
		_, err = this.processPredicate(onclause, true)
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

func (this *builder) buildAnsiJoinScan(node *algebra.KeyspaceTerm, onclause expression.Expression) (
	[]plan.Operator, expression.Expression, expression.Expression, float64, float64, error) {

	children := this.children
	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	lastOp := this.lastOp
	indexPushDowns := this.storeIndexPushDowns()
	defer func() {
		this.children = children
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
	this.coveringScans = nil
	this.countScan = nil
	this.order = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil
	this.lastOp = nil

	var err error

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]

	// check whether joining on meta().id
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	var primaryJoinKeys expression.Expression

	for _, fltr := range baseKeyspace.Filters() {
		if fltr.IsOnclause() {
			if eqFltr, ok := fltr.FltrExpr().(*expression.Eq); ok {
				if eqFltr.First().EquivalentTo(id) {
					node.SetPrimaryJoin()
					primaryJoinKeys = eqFltr.Second().Copy()
					break
				} else if eqFltr.Second().EquivalentTo(id) {
					node.SetPrimaryJoin()
					primaryJoinKeys = eqFltr.First().Copy()
					break
				}
			} else if inFltr, ok := fltr.FltrExpr().(*expression.In); ok {
				if inFltr.First().EquivalentTo(id) {
					node.SetPrimaryJoin()
					primaryJoinKeys = inFltr.Second().Copy()
					break
				}
			}
		}
	}

	_, err = node.Accept(this)
	if err != nil {
		switch e := err.(type) {
		case errors.Error:
			if e.Code() == errors.NO_ANSI_JOIN &&
				baseKeyspace.DnfPred() != nil && baseKeyspace.Onclause() != nil {

				// did not find an appropriate index path using both
				// on clause and where clause filters, try using just
				// the on clause filters
				baseKeyspace.SetOnclauseOnly()
				_, err = node.Accept(this)
			}
		}

		if err != nil {
			return nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
		}
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

	newOnclause := onclause.Copy()

	// do right-hand-side covering index scan first, in case an ANY clause contains
	// a join filter, if part of the join filter gets transformed first, the ANY clause
	// will no longer match during transformation.
	// (note this assumes the ANY clause is on the right-hand-side keyspace)
	if len(this.coveringScans) > 0 {
		for _, op := range this.coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			if primaryJoinKeys != nil {
				primaryJoinKeys, err = coverer.Map(primaryJoinKeys)
				if err != nil {
					return nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
				}
			}
			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}
		}
	}

	if len(coveringScans) > 0 {
		for _, op := range coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			if primaryJoinKeys != nil {
				primaryJoinKeys, err = coverer.Map(primaryJoinKeys)
				if err != nil {
					return nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
				}
			}
			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}

			// also need to perform cover transformation for index spans for
			// right-hand-side index scans since left-hand-side expressions
			// could be used as part of index spans for right-hand-side index scan
			for _, child := range this.children {
				if secondary, ok := child.(plan.SecondaryScan); ok {
					err := secondary.CoverJoinSpanExpressions(coverer)
					if err != nil {
						return nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
					}
				}
			}
		}
	}

	cost := float64(OPT_COST_NOT_AVAIL)
	cardinality := float64(OPT_CARD_NOT_AVAIL)
	useCBO := this.useCBO
	if useCBO {
		if len(this.children) > 0 {
			cost, cardinality = getNLJoinCost(lastOp, this.lastOp, baseKeyspace.Filters())
		} // TODO: else calculate cost for lookup join
	}

	return this.children, primaryJoinKeys, newOnclause, cost, cardinality, nil
}

func (this *builder) buildHashJoin(node *algebra.AnsiJoin) (hjoin *plan.HashJoin, err error) {
	child, buildExprs, probeExprs, aliases, newOnclause, cost, cardinality, err := this.buildHashJoinScan(node.Right(), node.Outer(), node.Onclause(), "join")
	if err != nil || child == nil {
		// cannot do hash join
		return nil, err
	}
	if newOnclause != nil {
		node.SetOnclause(newOnclause)
	}
	return plan.NewHashJoin(node, child, buildExprs, probeExprs, aliases, cost, cardinality), nil
}

func (this *builder) buildHashNest(node *algebra.AnsiNest) (hnest *plan.HashNest, err error) {
	child, buildExprs, probeExprs, aliases, newOnclause, cost, cardinality, err := this.buildHashJoinScan(node.Right(), node.Outer(), node.Onclause(), "nest")
	if err != nil || child == nil {
		// cannot do hash nest
		return nil, err
	}
	if len(aliases) != 1 {
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildHashNest: multiple (%d) build aliases", len(aliases)))
	}
	if newOnclause != nil {
		node.SetOnclause(newOnclause)
	}
	return plan.NewHashNest(node, child, buildExprs, probeExprs, aliases[0], cost, cardinality), nil
}

func (this *builder) buildHashJoinScan(right algebra.SimpleFromTerm, outer bool,
	onclause expression.Expression, op string) (
	child plan.Operator, buildExprs expression.Expressions, probeExprs expression.Expressions,
	buildAliases []string, newOnclause expression.Expression, cost, cardinality float64, err error) {

	var ksterm *algebra.KeyspaceTerm
	var keyspace string
	var defaultBuildRight bool

	if ksterm = algebra.GetKeyspaceTerm(right); ksterm != nil {
		right = ksterm
	}

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		// if USE HASH and USE KEYS are specified together, make sure the document key
		// expressions does not reference any keyspaces, otherwise hash join cannot be
		// used.
		if ksterm.Keys() != nil && ksterm.Keys().Static() == nil {
			return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, nil
		}
		keyspace = ksterm.Keyspace()
	case *algebra.ExpressionTerm:
		// hash join cannot handle expression term with any correlated references
		if right.IsCorrelated() {
			return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, nil
		}

		defaultBuildRight = true
	case *algebra.SubqueryTerm:
		// hash join cannot handle correlated subquery
		if right.Subquery().IsCorrelated() {
			return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, nil
		}

		defaultBuildRight = true
	default:
		return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, errors.NewPlanInternalError(fmt.Sprintf("buildHashJoinScan: unexpected right-hand side node type"))
	}

	useCBO := this.useCBO
	buildRight := false
	force := true
	joinHint := right.JoinHint()
	if joinHint == algebra.USE_HASH_BUILD {
		buildRight = true
	} else if joinHint == algebra.USE_HASH_PROBE {
		// in case of outer join, cannot build on dominant side
		// also in case of nest, can only build on right-hand-side
		if outer || op == "nest" {
			return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, nil
		}
	} else if outer || op == "nest" {
		// for outer join or nest, must build on right-hand side
		buildRight = true
	} else if defaultBuildRight {
		// for expression term and subquery term, if no USE HASH hint is
		// specified, then consider hash join/nest with the right-hand side
		// as build side
		buildRight = true
		force = false
	} else {
		force = false
	}

	alias := right.Alias()

	keyspaceNames := make(map[string]string, 1)
	keyspaceNames[alias] = keyspace

	baseKeyspace, _ := this.baseKeyspaces[alias]
	filters := baseKeyspace.Filters()
	if filters != nil {
		filters.ClearPlanFlags()
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
			if !eqFltr.First().Indexable() || !eqFltr.Second().Indexable() {
				continue
			}

			// check keyspace references for both sides
			firstKeyspaces, err := expression.CountKeySpaces(eqFltr.First(), keyspaceNames)
			if err != nil {
				return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}
			secondKeyspaces, err := expression.CountKeySpaces(eqFltr.Second(), keyspaceNames)
			if err != nil {
				return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}

			// make sure only one side of the equality predicate references
			// alias (which is right-hand-side of the join)
			firstRef := false
			secondRef := false
			if _, ok := firstKeyspaces[alias]; ok {
				firstRef = true
			}
			if _, ok := secondKeyspaces[alias]; ok {
				secondRef = true
			}

			found := false
			if firstRef && !secondRef {
				rightExprs = append(rightExprs, eqFltr.First().Copy())
				leftExprs = append(leftExprs, eqFltr.Second().Copy())
				found = true
			} else if !firstRef && secondRef {
				leftExprs = append(leftExprs, eqFltr.First().Copy())
				rightExprs = append(rightExprs, eqFltr.Second().Copy())
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
		return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, nil
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

	this.coveringScans = nil
	this.countScan = nil
	this.order = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil
	this.lastOp = nil

	children := this.children
	this.children = make([]plan.Operator, 0, 16)

	// Note that by this point join filters involving keyspaces that's already done planning
	// are already moved into filters and thus is available for index selection. This is ok
	// if we are doing nested-loop join. However, for hash join, since both sides of the
	// hash join are independent of each other, we cannot use join filters for index selection
	// when planning for the right-hand side.
	if ksterm != nil {
		ksterm.SetUnderHash()
		defer func() {
			ksterm.UnsetUnderHash()
		}()
	}

	_, err = right.Accept(this)
	if err != nil {
		return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
	}

	// if no plan generated, bail out
	if len(this.children) == 0 {
		return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, nil
	}

	if buildRight {
		child = plan.NewSequence(this.children...)
		this.children = children
		buildAliases = []string{alias}
	} else {
		child = plan.NewSequence(children...)
		buildAliases = make([]string, 0, len(this.baseKeyspaces))
		for _, kspace := range this.baseKeyspaces {
			if kspace.PlanDone() && kspace.Name() != alias {
				buildAliases = append(buildAliases, kspace.Name())
			}
		}
	}

	// perform cover transformation of leftExprs and rightExprs and onclause
	newOnclause = onclause.Copy()

	if len(this.coveringScans) > 0 {
		for _, op := range this.coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}

			for i, _ := range rightExprs {
				rightExprs[i], err = coverer.Map(rightExprs[i])
				if err != nil {
					return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
				}
			}
		}
	}

	if len(coveringScans) > 0 {
		for _, op := range coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}

			for i, _ := range leftExprs {
				leftExprs[i], err = coverer.Map(leftExprs[i])
				if err != nil {
					return nil, nil, nil, nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
				}
			}
		}
	}

	if useCBO {
		var bldRight bool
		cost, cardinality, bldRight = getHashJoinCost(lastOp, this.lastOp, leftExprs, rightExprs, buildRight, force, filters)
		if cost > 0.0 && cardinality > 0.0 {
			buildRight = bldRight
		}
	} else {
		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_COST_NOT_AVAIL
	}

	if buildRight {
		probeExprs = leftExprs
		buildExprs = rightExprs
	} else {
		buildExprs = leftExprs
		probeExprs = rightExprs
	}

	return child, buildExprs, probeExprs, buildAliases, newOnclause, cost, cardinality, nil
}

func (this *builder) buildAnsiJoinSimpleFromTerm(node algebra.SimpleFromTerm, onclause expression.Expression) (
	[]plan.Operator, expression.Expression, float64, float64, error) {

	var newOnclause expression.Expression
	var err error

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	filters := baseKeyspace.Filters()
	if filters != nil {
		filters.ClearPlanFlags()
	}

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

		newOnclause = onclause.Copy()

		for _, op := range this.coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
			}

			if fromExpr != nil {
				fromExpr, err = coverer.Map(fromExpr)
				if err != nil {
					return nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
				}
			}
		}

		if exprTerm != nil && fromExpr != nil {
			exprTerm.SetExpressionTerm(fromExpr)
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

	_, err = node.Accept(this)
	if err != nil {
		return nil, nil, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, err
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL

	if this.useCBO {
		cost, cardinality = getSimpleFromTermCost(lastOp, this.lastOp, filters)
	}

	return this.children, newOnclause, cost, cardinality, nil
}

// if both nested-loop join and hash join are to be attempted (in case of CBO),
// need to save/restore certain planner states in between consideration of
// the two join methods
type joinPlannerState struct {
	children      []plan.Operator
	subChildren   []plan.Operator
	coveringScans []plan.CoveringOperator
}

func (this *builder) saveJoinPlannerState() *joinPlannerState {
	return &joinPlannerState{
		children:      this.children,
		subChildren:   this.subChildren,
		coveringScans: this.coveringScans,
	}
}

func (this *builder) restoreJoinPlannerState(jps *joinPlannerState) {
	this.children = jps.children
	this.subChildren = jps.subChildren
	this.coveringScans = jps.coveringScans
}
