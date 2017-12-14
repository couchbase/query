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
		scans, primaryJoinKeys, newOnclause, err := this.buildAnsiJoinScan(right, node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		if len(scans) > 0 {
			return plan.NewAnsiJoin(node, plan.NewSequence(scans...)), nil
		}

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
		newKeyspaceTerm := algebra.NewKeyspaceTerm(right.Namespace(), right.Keyspace(), right.As(), primaryJoinKeys, right.Indexes())
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
		scans, primaryJoinKeys, newOnclause, err := this.buildAnsiJoinScan(right, node.Onclause(), node.Outer())
		if err != nil {
			return nil, err
		}

		if newOnclause != nil {
			node.SetOnclause(newOnclause)
		}

		if len(scans) > 0 {
			return plan.NewAnsiNest(node, plan.NewSequence(scans...)), nil
		}

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
		newKeyspaceTerm := algebra.NewKeyspaceTerm(right.Namespace(), right.Keyspace(), right.As(), primaryJoinKeys, right.Indexes())
		newKeyspaceTerm.SetProperty(right.Property())
		return plan.NewNestFromAnsi(keyspace, newKeyspaceTerm, node.Outer()), nil
	default:
		return nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiNest: ANSI NEST on %s must be a keyspace", node.Alias()))
	}
}

func (this *builder) buildAnsiJoinScan(node *algebra.KeyspaceTerm, onclause expression.Expression, outer bool) (
	[]plan.Operator, expression.Expression, expression.Expression, error) {

	children := this.children
	coveringScans := this.coveringScans
	countScan := this.countScan
	order := this.order
	orderScan := this.orderScan
	limit := this.limit
	offset := this.offset
	defer func() {
		this.children = children
		this.countScan = countScan
		this.order = order
		this.orderScan = orderScan
		this.limit = limit
		this.offset = offset

		if len(this.coveringScans) > 0 {
			this.coveringScans = append(coveringScans, this.coveringScans...)
		} else {
			this.coveringScans = coveringScans
		}
	}()

	this.children = nil
	this.coveringScans = nil
	this.countScan = nil
	this.order = nil
	this.orderScan = nil
	this.limit = nil
	this.offset = nil

	var err error

	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return nil, nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildAnsiJoinScan: missing baseKeyspace %s", node.Alias()))
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
			return nil, nil, nil, err
		}
	}

	baseKeyspace.dnfPred, baseKeyspace.origPred, err = combineFilters(baseKeyspace.filters, true)
	if err != nil {
		return nil, nil, nil, err
	}

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
		return nil, nil, nil, err
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
