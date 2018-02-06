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
)

func (this *builder) buildAnsiJoin(node *algebra.AnsiJoin) (op plan.Operator, err error) {
	right := node.Right()

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		err := this.processOnclause(right, node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		// currently only consider hash join when USE HASH join hint is specified
		var hjoin *plan.HashJoin
		if right.PreferHash() {
			hjoin, err = this.buildHashJoin(node)
			if err != nil {
				return nil, err
			}

			if hjoin != nil {
				return hjoin, nil
			}
		}

		right.SetUnderNL()
		scans, primaryJoinKeys, newOnclause, err := this.buildAnsiJoinScan(right, node.Onclause())
		if err != nil {
			return nil, err
		}

		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		if len(scans) > 0 {
			return plan.NewNLJoin(node, plan.NewSequence(scans...)), nil
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
		newKeyspaceTerm := algebra.NewKeyspaceTerm(right.Namespace(), right.Keyspace(), right.As(),
			primaryJoinKeys, right.Indexes())
		newKeyspaceTerm.SetProperty(right.Property())
		return plan.NewJoinFromAnsi(keyspace, newKeyspaceTerm, node.Outer()), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiJoin: ANSI JOIN on %s must be a keyspace", node.Alias()))
	}
}

func (this *builder) buildAnsiNest(node *algebra.AnsiNest) (op plan.Operator, err error) {
	right := node.Right()

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		err := this.processOnclause(right, node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		// currently only consider hash nest when USE HASH join hint is specified
		var hnest *plan.HashNest
		if right.PreferHash() {
			hnest, err = this.buildHashNest(node)
			if err != nil {
				return nil, err
			}

			if hnest != nil {
				return hnest, nil
			}
		}

		right.SetUnderNL()
		scans, primaryJoinKeys, newOnclause, err := this.buildAnsiJoinScan(right, node.Onclause())
		if err != nil {
			return nil, err
		}

		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		if len(scans) > 0 {
			return plan.NewNLNest(node, plan.NewSequence(scans...)), nil
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
		newKeyspaceTerm := algebra.NewKeyspaceTerm(right.Namespace(), right.Keyspace(), right.As(),
			primaryJoinKeys, right.Indexes())
		newKeyspaceTerm.SetProperty(right.Property())
		return plan.NewNestFromAnsi(keyspace, newKeyspaceTerm, node.Outer()), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiNest: ANSI NEST on %s must be a keyspace", node.Alias()))
	}
}

func (this *builder) processOnclause(node *algebra.KeyspaceTerm, onclause expression.Expression, outer bool) (err error) {
	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return errors.NewPlanInternalError(fmt.Sprintf("processOnclause: missing baseKeyspace %s", node.Alias()))
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
		err = this.processPredicate(onclause, true)
		if err != nil {
			return err
		}
	}

	err = combineFilters(baseKeyspace, true)
	if err != nil {
		return err
	}

	return nil
}

func (this *builder) buildAnsiJoinScan(node *algebra.KeyspaceTerm, onclause expression.Expression) (
	[]plan.Operator, expression.Expression, expression.Expression, error) {

	children := this.children
	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	indexPushDowns := this.storeIndexPushDowns()
	defer func() {
		this.children = children
		this.countScan = countScan
		this.orderScan = orderScan
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

	var err error

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]

	// check whether joining on meta().id
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	var primaryJoinKeys expression.Expression

	for _, fltr := range baseKeyspace.filters {
		if fltr.isOnclause() {
			if eqFltr, ok := fltr.fltrExpr.(*expression.Eq); ok {
				if eqFltr.First().EquivalentTo(id) {
					node.SetPrimaryJoin()
					primaryJoinKeys = eqFltr.Second()
					break
				} else if eqFltr.Second().EquivalentTo(id) {
					node.SetPrimaryJoin()
					primaryJoinKeys = eqFltr.First()
					break
				}
			} else if inFltr, ok := fltr.fltrExpr.(*expression.In); ok {
				if inFltr.First().EquivalentTo(id) {
					node.SetPrimaryJoin()
					primaryJoinKeys = inFltr.Second()
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
				baseKeyspace.dnfPred != nil && baseKeyspace.onclause != nil {

				// did not find an appropriate index path using both
				// on clause and where clause filters, try using just
				// the on clause filters
				baseKeyspace.SetOnclauseOnly()
				_, err = node.Accept(this)
			}
		}

		if err != nil {
			return nil, nil, nil, err
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

	newOnclause := onclause

	// do right-hand-side covering index scan first, in case an ANY clause contains
	// a join filter, if part of the join filter gets transformed first, the ANY clause
	// will no longer match during transformation.
	// (note this assumes the ANY clause is on the right-hand-side keyspace)
	if len(this.coveringScans) > 0 {
		for _, op := range this.coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	if len(coveringScans) > 0 {
		for _, op := range coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			newOnclause, err = coverer.Map(newOnclause)
			if err != nil {
				return nil, nil, nil, err
			}

			// also need to perform cover transformation for index spans for
			// right-hand-side index scans since left-hand-side expressions
			// could be used as part of index spans for right-hand-side index scan
			for _, child := range this.children {
				if secondary, ok := child.(plan.SecondaryScan); ok {
					err := secondary.CoverJoinSpanExpressions(coverer)
					if err != nil {
						return nil, nil, nil, err
					}
				}
			}
		}
	}

	return this.children, primaryJoinKeys, newOnclause, nil
}

func (this *builder) buildHashJoin(node *algebra.AnsiJoin) (hjoin *plan.HashJoin, err error) {
	right := node.Right()

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		child, buildExprs, probeExprs, aliases, err := this.buildHashJoinScan(right, node.Outer(), "join")
		if err != nil || child == nil {
			// cannot do hash join
			return nil, err
		}
		return plan.NewHashJoin(node, child, buildExprs, probeExprs, aliases), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildHashJoin: Hash JOIN on %s must be a keyspace", node.Alias()))
	}
}

func (this *builder) buildHashNest(node *algebra.AnsiNest) (hnest *plan.HashNest, err error) {
	right := node.Right()

	switch right := right.(type) {
	case *algebra.KeyspaceTerm:
		child, buildExprs, probeExprs, aliases, err := this.buildHashJoinScan(right, node.Outer(), "nest")
		if err != nil || child == nil {
			// cannot do hash nest
			return nil, err
		}
		if len(aliases) != 1 {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("buildHashNest: multiple (%d) build aliases", len(aliases)))
		}
		return plan.NewHashNest(node, child, buildExprs, probeExprs, aliases[0]), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildHashNest: Hash Nest on %s must be a keyspace", node.Alias()))
	}
}

func (this *builder) buildHashJoinScan(right *algebra.KeyspaceTerm, outer bool, op string) (
	child plan.Operator, buildExprs expression.Expressions, probeExprs expression.Expressions, buildAliases []string, err error) {

	buildRight := false
	joinHint := right.JoinHint()
	if joinHint == algebra.USE_HASH_BUILD {
		buildRight = true
	} else if joinHint == algebra.USE_HASH_PROBE {
		// in case of outer join, cannot build on dominant side
		// also in case of nest, can only build on right-hand-side
		if outer || op == "nest" {
			return nil, nil, nil, nil, nil
		}
	}

	alias := right.Alias()

	keyspaceNames := make(map[string]bool, 1)
	keyspaceNames[alias] = true

	baseKeyspace, _ := this.baseKeyspaces[alias]

	// expressions for building and probing
	buildExprs = make(expression.Expressions, 0, 4)
	probeExprs = make(expression.Expressions, 0, 4)

	// look for equality join predicates
	for _, fltr := range baseKeyspace.filters {
		if !fltr.isJoin() {
			continue
		}

		if eqFltr, ok := fltr.fltrExpr.(*expression.Eq); ok {
			if !eqFltr.First().Indexable() || !eqFltr.Second().Indexable() {
				continue
			}

			// check keyspace references for both sides
			firstKeyspaces, err := expression.CountKeySpaces(eqFltr.First(), keyspaceNames)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			secondKeyspaces, err := expression.CountKeySpaces(eqFltr.Second(), keyspaceNames)
			if err != nil {
				return nil, nil, nil, nil, err
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

			if firstRef && !secondRef {
				if buildRight {
					buildExprs = append(buildExprs, eqFltr.First())
					probeExprs = append(probeExprs, eqFltr.Second())
				} else {
					probeExprs = append(probeExprs, eqFltr.First())
					buildExprs = append(buildExprs, eqFltr.Second())
				}
			} else if !firstRef && secondRef {
				if buildRight {
					probeExprs = append(probeExprs, eqFltr.First())
					buildExprs = append(buildExprs, eqFltr.Second())
				} else {
					buildExprs = append(buildExprs, eqFltr.First())
					probeExprs = append(probeExprs, eqFltr.Second())
				}
			}
		}
	}

	if len(buildExprs) == 0 || len(probeExprs) == 0 {
		return nil, nil, nil, nil, nil
	}

	// left hand side is already built
	if len(this.subChildren) > 0 {
		parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
		this.children = append(this.children, parallel)
		this.subChildren = make([]plan.Operator, 0, 16)
	}

	// build right hand side

	coveringScans := this.coveringScans
	countScan := this.countScan
	orderScan := this.orderScan
	indexPushDowns := this.storeIndexPushDowns()
	defer func() {
		this.countScan = countScan
		this.orderScan = orderScan
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

	children := this.children
	this.children = make([]plan.Operator, 0, 16)

	// Note that by this point join filters involving keyspaces that's already done planning
	// are already moved into filters and thus is available for index selection. This is ok
	// if we are doing nested-loop join. However, for hash join, since both sides of the
	// hash join are independet of each other, we cannot use join filters for index selection
	// when planning for the right-hand side.
	right.SetUnderHash()
	defer func() {
		if child == nil {
			right.UnsetUnderHash()
		}
	}()

	_, err = right.Accept(this)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if buildRight {
		child = plan.NewSequence(this.children...)
		this.children = children
		buildAliases = []string{alias}
	} else {
		child = plan.NewSequence(children...)
		buildAliases = make([]string, 0, len(this.baseKeyspaces))
		for _, kspace := range this.baseKeyspaces {
			if kspace.PlanDone() && kspace.name != alias {
				buildAliases = append(buildAliases, kspace.name)
			}
		}
	}

	// perform cover transformation of buildExprs and probeExprs
	var leftExprs, rightExprs expression.Expressions
	if buildRight {
		leftExprs = probeExprs
		rightExprs = buildExprs
	} else {
		leftExprs = buildExprs
		rightExprs = probeExprs
	}

	if len(this.coveringScans) > 0 {
		for _, op := range this.coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			for i, _ := range rightExprs {
				rightExprs[i], err = coverer.Map(rightExprs[i])
				if err != nil {
					return nil, nil, nil, nil, err
				}
			}
		}
	}

	if len(coveringScans) > 0 {
		for _, op := range coveringScans {
			coverer := expression.NewCoverer(op.Covers(), op.FilterCovers())

			for i, _ := range leftExprs {
				leftExprs[i], err = coverer.Map(leftExprs[i])
				if err != nil {
					return nil, nil, nil, nil, err
				}
			}
		}
	}

	return child, buildExprs, probeExprs, buildAliases, nil
}
