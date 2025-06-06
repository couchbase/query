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
	Copy() Optimizer
	OptimizeQueryBlock(builder Builder, node algebra.Node, limit, offset expression.Expression,
		order *algebra.Order, distinct algebra.ResultTerms, advisorValidate bool) (
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
	MarkJoinFilterHints(children, subChildren []plan.Operator) error
	CheckBitFilters(qPlan, subPlan []plan.Operator)
	CheckJoinFilterHints(qPlan, subPlan []plan.Operator) (hintError bool, err error)
	DoCoveringTransformation(ops []plan.Operator, covers []plan.CoveringOperator, aggs algebra.Aggregates) error
	RemoveFromSubqueries(ops ...plan.Operator)
	NoExecute() bool
	SubqCoveringInfo() map[*algebra.Subselect]CoveringSubqInfo
	SkipCoverTransform() bool
	CoverSubSelect(sub *algebra.Subselect, subqUnderJoin bool) error
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

	var ksTerm *algebra.KeyspaceTerm
	var filters base.Filters
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
		filters = baseKeyspace.Filters()
		if len(filters) > 0 {
			filters.SaveIndexFlag()
			defer filters.RestoreIndexFlag()
		}
		if term, ok := baseKeyspace.Node().(*algebra.KeyspaceTerm); ok {
			ksTerm = term
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
			if ksTerm != nil && ksTerm.IsPrimaryJoin() {
				ksTerm.UnsetPrimaryJoin()
				for _, fl := range filters {
					fl.UnsetPrimaryJoin()
				}
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
			if ksTerm != nil && ksTerm.IsPrimaryJoin() {
				ksTerm.UnsetPrimaryJoin()
				for _, fl := range filters {
					fl.UnsetPrimaryJoin()
				}
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

// perform covering transformation for the final plan
func (this *builder) DoCoveringTransformation(ops []plan.Operator, covers []plan.CoveringOperator, aggs algebra.Aggregates) error {

	var err error
	var groupCoverer *expression.Coverer
	var aggPartialCoverer *PartialAggCoverer
	var aggFullCoverer *FullAggCoverer

	// check for group and aggregate pushdown
	if len(covers) == 1 && covers[0].GroupAggs() != nil {
		groupCoverer, aggPartialCoverer, aggFullCoverer, err = prepIndexGroupAggs(covers[0], aggs)
		if err != nil {
			return err
		}
	}

	return this.opsCoveringTransformation(ops, covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
}

func (this *builder) opsCoveringTransformation(ops []plan.Operator, covers []plan.CoveringOperator,
	groupCoverer *expression.Coverer, aggPartialCoverer *PartialAggCoverer, aggFullCoverer *FullAggCoverer) error {
	for _, op := range ops {
		err := this.opCoveringTransformation(op, covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *builder) opCoveringTransformation(op plan.Operator, covers []plan.CoveringOperator,
	groupCoverer *expression.Coverer, aggPartialCoverer *PartialAggCoverer, aggFullCoverer *FullAggCoverer) error {
	switch op := op.(type) {
	case *plan.NLJoin:
		newExprs, err := doCoverExprs(expression.Expressions{op.Onclause(), op.Filter()}, covers,
			groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetOnclause(newExprs[0])
		op.SetFilter(newExprs[1])
		// expect child to be a sequence
		if seq, ok := op.Child().(*plan.Sequence); ok {
			err = coverIndexSpans(seq.Children(), covers)
			if err != nil {
				return err
			}
		}
	case *plan.NLNest:
		newExprs, err := doCoverExprs(expression.Expressions{op.Onclause(), op.Filter()}, covers,
			groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetOnclause(newExprs[0])
		op.SetFilter(newExprs[1])
		// expect child to be a sequence
		if seq, ok := op.Child().(*plan.Sequence); ok {
			err = coverIndexSpans(seq.Children(), covers)
			if err != nil {
				return err
			}
		}
	case *plan.HashJoin:
		buildExprs := op.BuildExprs()
		probeExprs := op.ProbeExprs()
		buildLen := len(buildExprs)
		probeLen := len(probeExprs)
		exprs := make(expression.Expressions, 0, (2 + buildLen + probeLen))
		exprs = append(exprs, op.Onclause(), op.Filter())
		exprs = append(exprs, buildExprs...)
		exprs = append(exprs, probeExprs...)
		newExprs, err := doCoverExprs(exprs, covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetOnclause(newExprs[0])
		op.SetFilter(newExprs[1])
		op.SetBuildExprs(newExprs[2:(2 + buildLen)])
		op.SetProbeExprs(newExprs[(2 + buildLen):(2 + buildLen + probeLen)])
		err = this.opCoveringTransformation(op.Child(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
	case *plan.HashNest:
		buildExprs := op.BuildExprs()
		probeExprs := op.ProbeExprs()
		buildLen := len(buildExprs)
		probeLen := len(probeExprs)
		exprs := make(expression.Expressions, 0, (2 + buildLen + probeLen))
		exprs = append(exprs, op.Onclause(), op.Filter())
		exprs = append(exprs, buildExprs...)
		exprs = append(exprs, probeExprs...)
		newExprs, err := doCoverExprs(exprs, covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetOnclause(newExprs[0])
		op.SetFilter(newExprs[1])
		op.SetBuildExprs(newExprs[2:(2 + buildLen)])
		op.SetProbeExprs(newExprs[(2 + buildLen):(2 + buildLen + probeLen)])
		err = this.opCoveringTransformation(op.Child(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
	case *plan.Join:
		term := op.Term()
		newExprs, err := doCoverExprs(expression.Expressions{term.JoinKeys(), op.OnFilter()}, covers,
			groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		term.SetJoinKeys(newExprs[0])
		op.SetOnFilter(newExprs[1])
	case *plan.Nest:
		term := op.Term()
		newExprs, err := doCoverExprs(expression.Expressions{term.JoinKeys(), op.OnFilter()}, covers,
			groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		term.SetJoinKeys(newExprs[0])
		op.SetOnFilter(newExprs[1])
	case *plan.Unnest:
		term := op.Term()
		for _, cop := range covers {
			coverer := expression.NewCoverer(cop.Covers(), cop.FilterCovers())
			var anyRenamer *expression.AnyRenamer
			if arrayKey := cop.ImplicitArrayKey(); arrayKey != nil {
				anyRenamer = expression.NewAnyRenamer(arrayKey)
			}
			if anyRenamer != nil {
				err := term.MapExpression(anyRenamer)
				if err != nil {
					return err
				}
			}
			err := term.MapExpression(coverer)
			if err != nil {
				return err
			}
		}
		if groupCoverer != nil {
			err := term.MapExpression(groupCoverer)
			if err != nil {
				return err
			}
		}
		if aggPartialCoverer != nil {
			err := term.MapExpression(aggPartialCoverer)
			if err != nil {
				return err
			}
		} else if aggFullCoverer != nil {
			err := term.MapExpression(aggFullCoverer)
			if err != nil {
				return err
			}
		}
		if op.Filter() != nil {
			newExprs, err := doCoverExprs(expression.Expressions{op.Filter()}, covers,
				groupCoverer, aggPartialCoverer, aggFullCoverer)
			if err != nil {
				return err
			}
			op.SetFilter(newExprs[0])
		}
	case *plan.ExpressionScan:
		newExprs, err := doCoverExprs(expression.Expressions{op.FromExpr(), op.Filter()}, covers,
			groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetFromExpr(newExprs[0])
		op.SetFilter(newExprs[1])
	case *plan.Sequence:
		return this.opsCoveringTransformation(op.Children(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
	case *plan.Parallel:
		return this.opCoveringTransformation(op.Child(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
	case *plan.Filter:
		newExprs, err := doCoverExprs(expression.Expressions{op.Condition()}, covers,
			groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetCondition(newExprs[0])
	case *plan.Let:
		bindings := op.Bindings()
		for _, cop := range covers {
			coverer := expression.NewCoverer(cop.Covers(), cop.FilterCovers())
			var anyRenamer *expression.AnyRenamer
			if arrayKey := cop.ImplicitArrayKey(); arrayKey != nil {
				anyRenamer = expression.NewAnyRenamer(arrayKey)
			}
			if anyRenamer != nil {
				err := bindings.MapExpressions(anyRenamer)
				if err != nil {
					return err
				}
			}
			err := bindings.MapExpressions(coverer)
			if err != nil {
				return err
			}
		}
		if groupCoverer != nil {
			err := bindings.MapExpressions(groupCoverer)
			if err != nil {
				return err
			}
		}
		if aggPartialCoverer != nil {
			err := bindings.MapExpressions(aggPartialCoverer)
			if err != nil {
				return err
			}
		} else if aggFullCoverer != nil {
			err := bindings.MapExpressions(aggFullCoverer)
			if err != nil {
				return err
			}
		}
	case *plan.InitialProject:
		terms := op.Terms()
		for _, cop := range covers {
			coverer := expression.NewCoverer(cop.Covers(), cop.FilterCovers())
			var anyRenamer *expression.AnyRenamer
			if arrayKey := cop.ImplicitArrayKey(); arrayKey != nil {
				anyRenamer = expression.NewAnyRenamer(arrayKey)
			}
			for i := range terms {
				resTerm := terms[i].Result()
				if anyRenamer != nil {
					err := resTerm.MapExpression(anyRenamer)
					if err != nil {
						return err
					}
				}
				err := resTerm.MapExpression(coverer)
				if err != nil {
					return err
				}
			}
		}
		if groupCoverer != nil || aggPartialCoverer != nil || aggFullCoverer != nil {
			for i := range terms {
				resTerm := terms[i].Result()
				if groupCoverer != nil {
					err := resTerm.MapExpression(groupCoverer)
					if err != nil {
						return err
					}
				}
				if aggPartialCoverer != nil {
					err := resTerm.MapExpression(aggPartialCoverer)
					if err != nil {
						return err
					}
				} else if aggFullCoverer != nil {
					err := resTerm.MapExpression(aggFullCoverer)
					if err != nil {
						return err
					}
				}
			}
		}
	case *plan.Order:
		terms := op.Terms()
		for _, cop := range covers {
			coverer := expression.NewCoverer(cop.Covers(), cop.FilterCovers())
			var anyRenamer *expression.AnyRenamer
			if arrayKey := cop.ImplicitArrayKey(); arrayKey != nil {
				anyRenamer = expression.NewAnyRenamer(arrayKey)
			}
			if anyRenamer != nil {
				err := terms.MapExpressions(anyRenamer)
				if err != nil {
					return err
				}
			}
			err := terms.MapExpressions(coverer)
			if err != nil {
				return err
			}
		}
		if groupCoverer != nil {
			err := terms.MapExpressions(groupCoverer)
			if err != nil {
				return err
			}
		}
		if aggPartialCoverer != nil {
			err := terms.MapExpressions(aggPartialCoverer)
			if err != nil {
				return err
			}
		} else if aggFullCoverer != nil {
			err := terms.MapExpressions(aggFullCoverer)
			if err != nil {
				return err
			}
		}
	case *plan.InitialGroup:
		newKeys, err := doCoverExprs(op.Keys(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetKeys(newKeys)
	case *plan.IntermediateGroup:
		newKeys, err := doCoverExprs(op.Keys(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetKeys(newKeys)
	case *plan.FinalGroup:
		newKeys, err := doCoverExprs(op.Keys(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)
		if err != nil {
			return err
		}
		op.SetKeys(newKeys)
	case *plan.With:
		return this.opCoveringTransformation(op.Child(), covers, groupCoverer, aggPartialCoverer, aggFullCoverer)

		// case *plan.Limit, *plan.Offset:
	}
	return nil
}

func doCoverExprs(exprs expression.Expressions, covers []plan.CoveringOperator,
	groupCoverer *expression.Coverer, aggPartialCoverer *PartialAggCoverer,
	aggFullCoverer *FullAggCoverer) (expression.Expressions, error) {

	if len(exprs) == 0 {
		return exprs, nil
	}

	var err error
	newExprs := exprs.Copy()
	for _, cop := range covers {
		coverer := expression.NewCoverer(cop.Covers(), cop.FilterCovers())
		var anyRenamer *expression.AnyRenamer
		if arrayKey := cop.ImplicitArrayKey(); arrayKey != nil {
			anyRenamer = expression.NewAnyRenamer(arrayKey)
		}

		for i := 0; i < len(newExprs); i++ {
			if newExprs[i] != nil {
				if anyRenamer != nil {
					newExprs[i], err = anyRenamer.Map(newExprs[i])
					if err != nil {
						return nil, err
					}
				}
				newExprs[i], err = coverer.Map(newExprs[i])
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if groupCoverer != nil || aggPartialCoverer != nil || aggFullCoverer != nil {
		for i := 0; i < len(newExprs); i++ {
			if newExprs[i] != nil {
				if groupCoverer != nil {
					newExprs[i], err = groupCoverer.Map(newExprs[i])
					if err != nil {
						return nil, err
					}
				}
				if aggPartialCoverer != nil {
					newExprs[i], err = aggPartialCoverer.Map(newExprs[i])
					if err != nil {
						return nil, err
					}
				} else if aggFullCoverer != nil {
					newExprs[i], err = aggFullCoverer.Map(newExprs[i])
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return newExprs, nil
}

func coverIndexSpans(ops []plan.Operator, covers []plan.CoveringOperator) error {
	var err error
	for _, cop := range covers {
		coverer := expression.NewCoverer(cop.Covers(), cop.FilterCovers())

		for _, op := range ops {
			if secondary, ok := op.(plan.SecondaryScan); ok {
				err = secondary.CoverJoinSpanExpressions(coverer, cop.ImplicitArrayKey())
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// check for any SubqueryTerm that falls under inner of nested-loop join, in which case we build an
// ExpressionScan on top of the subquery; need to remove from subqCoveringInfo
func (this *builder) RemoveFromSubqueries(ops ...plan.Operator) {
	for _, op := range ops {
		switch op := op.(type) {
		case *plan.ExpressionScan:
			if subq, ok := op.FromExpr().(*algebra.Subquery); ok && op.SubqueryPlan() != nil {
				for _, sub := range subq.Select().Subselects() {
					delete(this.subqCoveringInfo, sub)
				}
			}
		case *plan.Parallel:
			this.RemoveFromSubqueries(op.Child())
		case *plan.Sequence:
			this.RemoveFromSubqueries(op.Children()...)
		case *plan.NLJoin:
			this.RemoveFromSubqueries(op.Child())
		case *plan.NLNest:
			this.RemoveFromSubqueries(op.Child())
		case *plan.HashJoin:
			this.RemoveFromSubqueries(op.Child())
		case *plan.HashNest:
			this.RemoveFromSubqueries(op.Child())
		case *plan.With:
			this.RemoveFromSubqueries(op.Child())
		}
	}
}

func (this *builder) NoExecute() bool {
	return this.hasBuilderFlag(BUILDER_NO_EXECUTE)
}

func (this *builder) SubqCoveringInfo() map[*algebra.Subselect]CoveringSubqInfo {
	return this.subqCoveringInfo
}

func (this *builder) SkipCoverTransform() bool {
	return this.joinEnum() || this.subqInJoinEnum() || this.subqUnderJoin() || this.indexAdvisor
}

type CoveringSubqInfo interface {
	SubqTermPlans() []*subqTermPlan
	AddSubqTermPlan(subPlan *subqTermPlan)
	CoveringScans() []plan.CoveringOperator
	SubqueryTerms() []*algebra.SubqueryTerm
	Aggregates() algebra.Aggregates
}

func (this *builder) CoverSubSelect(sub *algebra.Subselect, subqUnderJoin bool) (err error) {

	if info, ok := this.subqCoveringInfo[sub]; ok && info != nil {
		for _, subqTermPlan := range info.SubqTermPlans() {
			if (subqUnderJoin && !subqTermPlan.IsUnderJoin()) ||
				(!subqUnderJoin && !subqTermPlan.IsJoinEnum()) {
				continue
			}
			err = this.DoCoveringTransformation([]plan.Operator{subqTermPlan.Operator()},
				info.CoveringScans(), info.Aggregates())
			if err != nil {
				return err
			}
		}
		subqTerms := info.SubqueryTerms()
		if len(subqTerms) > 0 {
			err = this.coverSubqTerms(subqTerms, subqUnderJoin)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *builder) coverSubqTerms(subqTerms []*algebra.SubqueryTerm, subqUnderJoin bool) (err error) {

	for _, subqTerm := range subqTerms {
		for _, sub := range subqTerm.Subquery().Subselects() {
			err = this.CoverSubSelect(sub, subqUnderJoin)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
