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
		keyspaceFinder := newKeyspaceFinder(this.baseKeyspaces, this.from.PrimaryTerm().Alias())
		_, err := node.From().Accept(keyspaceFinder)
		if err != nil {
			return err
		}
		this.pushableOnclause = keyspaceFinder.pushableOnclause

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

		this.extractPredicates(this.where, nil)

		// Use FROM clause in index selection
		_, err = node.From().Accept(this)
		if err != nil {
			return err
		}
	} else {
		// No FROM clause
		this.resetPushDowns()
		scan := plan.NewDummyScan()
		this.children = append(this.children, scan)
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
	this.appendQueryInfo(scan, node, uncovered)

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
	this.children = append(this.children, scan)
	this.lastOp = scan

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
		fetch := plan.NewFetch(keyspace, node, names, cost, cardinality)
		this.children = append(this.children, fetch)
		this.lastOp = fetch
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
	this.children = append(this.children, sel.(plan.Operator), plan.NewAlias(node.Alias()))
	this.lastOp = sel.(plan.Operator)

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
		keyspaces := make(map[string]string, len(this.baseKeyspaces))
		for _, bks := range this.baseKeyspaces {
			keyspaces[bks.Name()] = keyspaces[bks.Keyspace()]
		}
		cost, cardinality = getExpressionScanCost(node.ExpressionTerm(), keyspaces)
	}
	scan := plan.NewExpressionScan(node.ExpressionTerm(), node.Alias(), node.IsCorrelated(), cost, cardinality)
	this.children = append(this.children, scan)
	this.lastOp = scan

	err := this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitJoin(node *algebra.Join) (interface{}, error) {
	this.resetProjection()
	this.resetIndexGroupAggs()
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); ok && term.IsKeyspace() {
		this.resetOffsetLimit()
	} else {
		this.resetOrderOffsetLimit()
	}

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	join := plan.NewJoin(keyspace, node)
	if len(this.subChildren) > 0 {
		parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
		this.children = append(this.children, parallel)
		this.subChildren = make([]plan.Operator, 0, 16)
	}
	this.children = append(this.children, join)
	this.lastOp = join

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	this.requirePrimaryKey = true
	this.resetIndexGroupAggs()
	this.resetProjection()
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); ok && term.IsKeyspace() {
		this.resetOffsetLimit()
	} else {
		this.resetOrderOffsetLimit()
	}

	_, err := node.Left().Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	join, err := this.buildIndexJoin(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.subChildren = append(this.subChildren, join)
	this.lastOp = join

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitAnsiJoin(node *algebra.AnsiJoin) (interface{}, error) {
	this.requirePrimaryKey = true
	this.resetIndexGroupAggs()
	this.resetProjection()
	if term, ok := node.PrimaryTerm().(*algebra.ExpressionTerm); ok && term.IsKeyspace() {
		this.resetOffsetLimit()
	} else {
		this.resetOrderOffsetLimit()
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
		this.subChildren = append(this.subChildren, join)
	case *plan.Join, *plan.HashJoin:
		if len(this.subChildren) > 0 {
			parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
			this.children = append(this.children, parallel)
			this.subChildren = make([]plan.Operator, 0, 16)
		}
		this.children = append(this.children, join)
	}
	this.lastOp = join

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitNest(node *algebra.Nest) (interface{}, error) {
	this.resetIndexGroupAggs()
	this.resetProjection()

	if this.hasOffsetOrLimit() && !node.Outer() {
		this.resetOffsetLimit()
	}

	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	if len(this.subChildren) > 0 {
		parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
		this.children = append(this.children, parallel)
		this.subChildren = make([]plan.Operator, 0, 16)
	}

	nest := plan.NewNest(keyspace, node)
	this.children = append(this.children, nest)
	this.lastOp = nest

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	this.requirePrimaryKey = true
	this.resetIndexGroupAggs()
	this.resetProjection()

	if this.hasOffsetOrLimit() && !node.Outer() {
		this.resetOffsetLimit()
	}

	_, err := node.Left().Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}

	right := node.Right()
	right.SetDefaultNamespace(this.namespace)
	namespace, err := this.datastore.NamespaceByName(right.Namespace())
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(right.Keyspace())
	if err != nil {
		return nil, err
	}

	nest, err := this.buildIndexNest(keyspace, node)
	if err != nil {
		return nil, err
	}

	this.subChildren = append(this.subChildren, nest)
	this.lastOp = nest

	err = this.processKeyspaceDone(node.Alias())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (this *builder) VisitAnsiNest(node *algebra.AnsiNest) (interface{}, error) {
	this.requirePrimaryKey = true
	this.resetIndexGroupAggs()
	this.resetProjection()

	if this.hasOffsetOrLimit() && !node.Outer() {
		this.resetOffsetLimit()
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
		this.subChildren = append(this.subChildren, nest)
	case *plan.Nest, *plan.HashNest:
		if len(this.subChildren) > 0 {
			parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
			this.children = append(this.children, parallel)
			this.subChildren = make([]plan.Operator, 0, 16)
		}
		this.children = append(this.children, nest)
	}
	this.lastOp = nest

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
		unnest := plan.NewUnnest(node)
		this.subChildren = append(this.subChildren, unnest)
		parallel := plan.NewParallel(plan.NewSequence(this.subChildren...), this.maxParallelism)
		this.children = append(this.children, parallel)
		this.subChildren = make([]plan.Operator, 0, 16)
		this.lastOp = unnest
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

	scan := plan.NewCountScan(keyspace, from)
	this.children = append(this.children, scan)
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
