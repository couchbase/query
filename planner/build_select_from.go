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
	"strings"

	"github.com/couchbase/query/algebra"
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

		numUnnests := 0
		for _, keyspace := range this.baseKeyspaces {
			if keyspace.IsPrimaryUnnest() {
				numUnnests++
			}
		}
		if numUnnests > 0 {
			primKeyspace, _ := this.baseKeyspaces[primaryTerm.Alias()]

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
			optimizer.Initialize(this.baseKeyspaces, this.context.FeatureControls())
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
		if this.useCBO {
			cost, cardinality = getDummyScanCost()
		}
		this.addChildren(plan.NewDummyScan(cost, cardinality))
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

	if this.subquery && this.correlated && node.Keys() == nil {
		return nil, errors.NewSubqueryMissingKeysError(node.Keyspace())
	}

	scan, err := this.selectScan(keyspace, node)

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

	// for inner side of ANSI JOIN/NEST, scans will be considered multiple times for different
	// join methods, wait till join is finalized before marking index filters
	if this.useCBO && !node.IsAnsiJoinOp() {
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
		if this.useCBO && (cost > 0.0) {
			fetchCost := getFetchCost(keyspace, cardinality)
			if fetchCost > 0.0 {
				cost += fetchCost
			} else {
				cost = OPT_COST_NOT_AVAIL
				cardinality = OPT_CARD_NOT_AVAIL
			}
		}
		this.addChildren(plan.NewFetch(keyspace, node, names, cost, cardinality))
	}

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
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
	this.addChildren(selOp, plan.NewAlias(node.Alias(), selOp.Cost(), selOp.Cardinality()))

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
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
	if this.useCBO {
		cost, cardinality = getExpressionScanCost(node.ExpressionTerm(), this.keyspaceNames)
	}
	this.addChildren(plan.NewExpressionScan(node.ExpressionTerm(), node.Alias(), node.IsCorrelated(), cost, cardinality))

	err := this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
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
	if this.useCBO {
		cost, cardinality = getLookupJoinCost(this.lastOp, node.Outer(), right,
			this.baseKeyspaces[right.Alias()])
	}
	join := plan.NewJoin(keyspace, node, cost, cardinality)
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
	this.requirePrimaryKey = true

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
	if this.useCBO {
		cost, cardinality = getLookupNestCost(this.lastOp, node.Outer(), right,
			this.baseKeyspaces[right.Alias()])
	}
	this.addChildren(plan.NewNest(keyspace, node, cost, cardinality))

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
	this.requirePrimaryKey = true

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
		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		if this.useCBO {
			cost, cardinality = getUnnestCost(node, this.lastOp, this.keyspaceNames)
		}
		this.addSubChildren(plan.NewUnnest(node, cost, cardinality))
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
		if !ok || count.Distinct() || count.IsWindowAggregate() {
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

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	if this.useCBO {
		cost, cardinality = getCountScanCost()
	}
	this.addChildren(plan.NewCountScan(keyspace, from, cost, cardinality))
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
