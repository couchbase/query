//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

type Optimizer interface {
	OptimizeQueryBlock(builder Builder, node algebra.Node, limit, offset expression.Expression,
		order *algebra.Order, distinct algebra.ResultTerms) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, bool, error)
}

type Builder interface {
	GetBaseKeyspaces() map[string]*base.BaseKeyspace
	GetKeyspaceNames() map[string]string
	GetPrepareContext() *PrepareContext
	AddOuterOnclause(onclause expression.Expression, alias string,
		baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string) error
	BuildScan(node algebra.SimpleFromTerm, order, joinFilter bool) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, map[*algebra.Unnest]bool, expression.Expression, error)
	BuildNLJoin(node *algebra.AnsiJoin, outerAliases []string,
		outerPlan, outerSubPlan []plan.Operator, outerCoveringScans []plan.CoveringOperator,
		outerFilter expression.Expression) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error)
	BuildNLNest(node *algebra.AnsiNest, outerAliases []string,
		outerPlan, outerSubPlan []plan.Operator, outerCoveringScans []plan.CoveringOperator,
		outerFilter expression.Expression) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error)
	BuildHashJoin(node *algebra.AnsiJoin, outerAliases []string,
		outerPlan, outerSubPlan, innerPlan, innerSubPlan []plan.Operator,
		outerCoveringScans, innerCoveringScans []plan.CoveringOperator,
		outerFilter expression.Expression, joinFilter bool) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error)
	BuildHashNest(node *algebra.AnsiNest, outerAliases []string,
		outerPlan, outerSubPlan, innerPlan, innerSubPlan []plan.Operator,
		outerCoveringScans, innerCoveringScans []plan.CoveringOperator,
		outerFilter expression.Expression) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error)
	BuildUnnest(unnest *algebra.Unnest, outerPlan, outerSubPlan []plan.Operator,
		outerCoveringScans []plan.CoveringOperator, outerFilter expression.Expression) (
		[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error)

	MarkKeyspaceHints() error
	MarkJoinFilterHints() error
	CheckBitFilters(qPlan, subPlan []plan.Operator)
	CheckJoinFilterHints(qPlan, subPlan []plan.Operator) (hintError bool, err error)
}

func (this *builder) GetBaseKeyspaces() map[string]*base.BaseKeyspace {
	return this.baseKeyspaces
}

func (this *builder) GetKeyspaceNames() map[string]string {
	return this.keyspaceNames
}

func (this *builder) GetPrepareContext() *PrepareContext {
	return this.context
}

func (this *builder) AddOuterOnclause(onclause expression.Expression, alias string,
	baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string) error {
	baseKeyspace, _ := baseKeyspaces[alias]
	if !baseKeyspace.HasOuterFilters() {
		_, err := ClassifyExprKeyspace(onclause, baseKeyspaces, keyspaceNames, alias,
			true, true, this.advisorValidate(), this.context)
		baseKeyspace.SetOuterFilters()
		return err
	}
	return nil
}

func (this *builder) BuildScan(node algebra.SimpleFromTerm, order, joinFilter bool) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, map[*algebra.Unnest]bool, expression.Expression, error) {
	children := this.children
	subChildren := this.subChildren
	coveringScans := this.coveringScans
	coveredUnnests := this.coveredUnnests
	filter := this.filter
	countScan := this.countScan
	orderScan := this.orderScan
	lastOp := this.lastOp
	indexPushDowns := this.storeIndexPushDowns()
	joinProps := node.UnsetJoinProps()
	this.setJoinEnum()
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.coveringScans = coveringScans
		this.coveredUnnests = coveredUnnests
		this.filter = filter
		this.countScan = countScan
		this.orderScan = orderScan
		this.lastOp = lastOp
		this.restoreIndexPushDowns(indexPushDowns, true)
		node.SetJoinProps(joinProps)
		this.unsetJoinEnum()
	}()

	this.children = make([]plan.Operator, 0, 16)
	this.subChildren = make([]plan.Operator, 0, 16)
	this.coveringScans = nil
	this.coveredUnnests = nil
	this.countScan = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil
	this.lastOp = nil

	if order {
		if this.order == nil {
			return nil, nil, nil, nil, nil, nil
		}
		this.setBuilderFlag(BUILDER_CHK_INDEX_ORDER)
		defer this.unsetBuilderFlag(BUILDER_CHK_INDEX_ORDER)
	} else {
		this.order = nil
	}

	if joinFilter {
		this.setBuilderFlag(BUILDER_DO_JOIN_FILTER)
		defer this.unsetBuilderFlag(BUILDER_DO_JOIN_FILTER)
	}

	_, err := node.Accept(this)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// if costing is not available, return nil plan
	if this.lastOp != nil &&
		(this.lastOp.Cost() <= 0.0 || this.lastOp.Cardinality() <= 0.0 ||
			this.lastOp.Size() <= 0 || this.lastOp.FrCost() <= 0.0) {
		return nil, nil, nil, nil, nil, nil
	}

	return this.children, this.subChildren, this.coveringScans, this.coveredUnnests, this.filter, nil
}

func (this *builder) BuildNLJoin(join *algebra.AnsiJoin, outerAliases []string,
	outerPlan, outerSubPlan []plan.Operator, outerCoveringScans []plan.CoveringOperator,
	outerFilter expression.Expression) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error) {

	return this.buildJoinOp(join, nil, outerAliases, outerPlan, outerSubPlan, nil, nil,
		outerCoveringScans, nil, outerFilter, false, false)
}

func (this *builder) BuildNLNest(nest *algebra.AnsiNest, outerAliases []string,
	outerPlan, outerSubPlan []plan.Operator, outerCoveringScans []plan.CoveringOperator,
	outerFilter expression.Expression) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error) {

	return this.buildJoinOp(nil, nest, outerAliases, outerPlan, outerSubPlan, nil, nil,
		outerCoveringScans, nil, outerFilter, false, false)
}

func (this *builder) BuildHashJoin(join *algebra.AnsiJoin, outerAliases []string,
	outerPlan, outerSubPlan, innerPlan, innerSubPlan []plan.Operator,
	outerCoveringScans, innerCoveringScans []plan.CoveringOperator,
	outerFilter expression.Expression, joinFilter bool) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error) {

	return this.buildJoinOp(join, nil, outerAliases, outerPlan, outerSubPlan,
		innerPlan, innerSubPlan, outerCoveringScans, innerCoveringScans, outerFilter, true, joinFilter)
}

func (this *builder) BuildHashNest(nest *algebra.AnsiNest, outerAliases []string,
	outerPlan, outerSubPlan, innerPlan, innerSubPlan []plan.Operator,
	outerCoveringScans, innerCoveringScans []plan.CoveringOperator,
	outerFilter expression.Expression) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error) {

	return this.buildJoinOp(nil, nest, outerAliases, outerPlan, outerSubPlan,
		innerPlan, innerSubPlan, outerCoveringScans, innerCoveringScans, outerFilter, true, false)
}

func (this *builder) buildJoinOp(join *algebra.AnsiJoin, nest *algebra.AnsiNest,
	outerAliases []string, outerPlan, outerSubPlan, innerPlan, innerSubPlan []plan.Operator,
	outerCoveringScans, innerCoveringScans []plan.CoveringOperator,
	outerFilter expression.Expression, hash, joinFilter bool) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error) {

	children := this.children
	subChildren := this.subChildren
	coveringScans := this.coveringScans
	filter := this.filter
	lastOp := this.lastOp
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.coveringScans = coveringScans
		this.filter = filter
		this.lastOp = lastOp
	}()

	this.children = plan.CopyOperators(outerPlan)
	this.subChildren = plan.CopyOperators(outerSubPlan)
	this.coveringScans = plan.CopyCoveringOperators(outerCoveringScans)
	this.filter = expression.Copy(outerFilter)
	this.offset = nil
	this.limit = nil
	if len(this.subChildren) > 0 {
		this.lastOp = this.subChildren[len(this.subChildren)-1]
	} else {
		this.lastOp = this.children[len(this.children)-1]
	}
	if hash {
		innerPlan = plan.CopyOperators(innerPlan)
		innerSubPlan = plan.CopyOperators(innerSubPlan)
		innerCoveringScans = plan.CopyCoveringOperators(innerCoveringScans)
	} else {
		// nested-loop join/nest replans the inner side, save/restore filter index flag
		var innerAlias string
		if join != nil {
			innerAlias = join.Alias()
		} else if nest != nil {
			innerAlias = nest.Alias()
		}
		baseKeyspace, ok := this.baseKeyspaces[innerAlias]
		if !ok {
			return nil, nil, nil, nil, errors.NewPlanInternalError("buildJoinOp: baseKeyspace not found for " + innerAlias)
		}
		filters := baseKeyspace.Filters()
		if len(filters) > 0 {
			filters.SaveIndexFlag()
			defer filters.RestoreIndexFlag()
		}
	}

	this.setJoinEnum()
	for _, a := range outerAliases {
		if baseKeyspace, ok := this.baseKeyspaces[a]; ok {
			baseKeyspace.SetPlanDone()
		}
	}
	defer func() {
		this.unsetJoinEnum()
		for _, a := range outerAliases {
			if baseKeyspace, ok := this.baseKeyspaces[a]; ok {
				baseKeyspace.UnsetPlanDone()
			}
		}
	}()

	if joinFilter {
		this.setBuilderFlag(BUILDER_DO_JOIN_FILTER)
		defer this.unsetBuilderFlag(BUILDER_DO_JOIN_FILTER)
	}

	if join != nil {
		if hash {
			joinFilter, selec, err := this.getFilter(join.Alias(), true, join.Onclause())
			if err != nil {
				return nil, nil, nil, nil, err
			}

			hjoin, _, err := this.buildHashJoin(join, joinFilter, selec, innerPlan, innerSubPlan, innerCoveringScans)
			if err != nil || hjoin == nil {
				return nil, nil, nil, nil, err
			}
			if len(this.subChildren) > 0 {
				this.addChildren(this.addSubchildrenParallel())
			}
			this.addChildren(hjoin)
		} else {
			nljoin, err := this.buildAnsiJoin(join)
			if err != nil || nljoin == nil {
				return nil, nil, nil, nil, err
			}
			if nlj, ok := nljoin.(*plan.NLJoin); ok {
				this.addSubChildren(nlj)
			} else {
				// could be lookup join
				if len(this.subChildren) > 0 {
					this.addChildren(this.addSubchildrenParallel())
				}
				this.addChildren(nljoin)
			}
		}
	} else if nest != nil {
		if hash {
			joinFilter, selec, err := this.getFilter(nest.Alias(), true, nest.Onclause())
			if err != nil {
				return nil, nil, nil, nil, err
			}

			hnest, _, err := this.buildHashNest(nest, joinFilter, selec, innerPlan, innerSubPlan, innerCoveringScans)
			if err != nil || hnest == nil {
				return nil, nil, nil, nil, err
			}
			if len(this.subChildren) > 0 {
				this.addChildren(this.addSubchildrenParallel())
			}
			this.addChildren(hnest)
		} else {
			nlnest, err := this.buildAnsiNest(nest)
			if err != nil || nlnest == nil {
				return nil, nil, nil, nil, err
			}
			if nln, ok := nlnest.(*plan.NLNest); ok {
				this.addSubChildren(nln)
			} else {
				// could be lookup join
				if len(this.subChildren) > 0 {
					this.addChildren(this.addSubchildrenParallel())
				}
				this.addChildren(nlnest)
			}
		}
	}

	return this.children, this.subChildren, this.coveringScans, this.filter, nil
}

func (this *builder) BuildUnnest(unnest *algebra.Unnest, outerPlan, outerSubPlan []plan.Operator,
	outerCoveringScans []plan.CoveringOperator, outerFilter expression.Expression) (
	[]plan.Operator, []plan.Operator, []plan.CoveringOperator, expression.Expression, error) {

	children := this.children
	subChildren := this.subChildren
	coveringScans := this.coveringScans
	filter := this.filter
	lastOp := this.lastOp
	defer func() {
		this.children = children
		this.subChildren = subChildren
		this.coveringScans = coveringScans
		this.filter = filter
		this.lastOp = lastOp
	}()

	this.children = plan.CopyOperators(outerPlan)
	this.subChildren = plan.CopyOperators(outerSubPlan)
	this.coveringScans = plan.CopyCoveringOperators(outerCoveringScans)
	this.filter = expression.Copy(outerFilter)
	if len(this.subChildren) > 0 {
		this.lastOp = this.subChildren[len(this.subChildren)-1]
	} else {
		this.lastOp = this.children[len(this.children)-1]
	}

	this.setJoinEnum()
	defer this.unsetJoinEnum()

	err := this.buildUnnest(unnest)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return this.children, this.subChildren, this.coveringScans, this.filter, nil
}

func (this *builder) MarkKeyspaceHints() (err error) {
	for alias, _ := range this.baseKeyspaces {
		err = this.markOptimHints(alias, true)
		if err != nil {
			return err
		}
	}
	return
}

func (this *builder) CheckJoinFilterHints(qPlan, subPlan []plan.Operator) (hintError bool, err error) {
	err = checkJoinFilterHint(this.baseKeyspaces, subPlan...)
	for _, baseKeyspace := range this.baseKeyspaces {
		if baseKeyspace.HasJoinFltrHintError() {
			hintError = true
			baseKeyspace.UnsetJoinFltrHintError()
		}
	}
	if err != nil || hintError {
		return
	}

	err = checkJoinFilterHint(this.baseKeyspaces, qPlan...)
	for _, baseKeyspace := range this.baseKeyspaces {
		if baseKeyspace.HasJoinFltrHintError() {
			hintError = true
			baseKeyspace.UnsetJoinFltrHintError()
		}
	}
	return
}

func (this *builder) CheckBitFilters(qPlan, subPlan []plan.Operator) {
	probeAliases := make(map[string]map[string]bool, len(this.baseKeyspaces))
	checkProbeBFAliases(probeAliases, true, subPlan...)
	checkProbeBFAliases(probeAliases, true, qPlan...)

	checkBuildBFAliases(probeAliases, subPlan...)
	checkBuildBFAliases(probeAliases, qPlan...)

	if len(probeAliases) > 0 {
		checkProbeBFAliases(probeAliases, false, subPlan...)
		checkProbeBFAliases(probeAliases, false, qPlan...)
	}
}
