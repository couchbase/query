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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func (this *builder) buildOrScan(node *algebra.KeyspaceTerm, baseKeyspace *baseKeyspace,
	id expression.Expression, pred *expression.Or, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	scan plan.SecondaryScan, sargLength int, err error) {

	indexPushDowns := this.storeIndexPushDowns()
	if this.cover != nil || this.hasOrderOrOffsetOrLimit() {
		scan, sargLength, err = this.buildOrScanTryPushdowns(node, baseKeyspace, id, pred,
			indexes, primaryKey, formalizer)
		if err != nil || scan != nil {
			return
		}
		this.restoreIndexPushDowns(indexPushDowns, true)
	}

	return this.buildOrScanNoPushdowns(node, id, pred, indexes, primaryKey, formalizer)
}

func (this *builder) buildOrScanTryPushdowns(node *algebra.KeyspaceTerm, baseKeyspace *baseKeyspace,
	id expression.Expression, pred *expression.Or, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	plan.SecondaryScan, int, error) {

	coveringScans := this.coveringScans

	scan, sargLength, err := this.buildTermScan(node, baseKeyspace, id, indexes, primaryKey, formalizer)
	if err == nil && scan != nil {
		foundPushdown := len(this.coveringScans) > len(coveringScans) || this.countScan != nil ||
			this.hasOrderOrOffsetOrLimit()

		if foundPushdown {
			return scan, sargLength, nil
		}
	}

	return nil, 0, err
}

func (this *builder) buildOrScanNoPushdowns(node *algebra.KeyspaceTerm, id expression.Expression,
	pred *expression.Or, indexes []datastore.Index, primaryKey expression.Expressions,
	formalizer *expression.Formalizer) (plan.SecondaryScan, int, error) {

	where := this.where
	cover := this.cover

	defer func() {
		this.where = where
		this.cover = cover
	}()

	this.cover = nil
	this.resetIndexGroupAggs()

	if this.order != nil {
		this.resetOrderOffsetLimit()
	}

	limit := offsetPlusLimit(this.offset, this.limit)
	this.resetOffset()

	var buf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	if len(pred.Operands()) <= len(buf) {
		scans = buf[0:0]
	} else {
		scans = make([]plan.SecondaryScan, 0, len(pred.Operands()))
	}

	minSargLength := 0

	orTerms, truth := flattenOr(pred)
	if orTerms == nil || truth {
		return nil, minSargLength, nil
	}

	join := node.IsAnsiJoinOp()
	for _, op := range orTerms.Operands() {
		this.where = op
		this.limit = limit

		baseKeyspaces := copyBaseKeyspaces(this.baseKeyspaces)
		err := ClassifyExpr(op, baseKeyspaces, join)
		if err != nil {
			return nil, 0, err
		}

		if baseKeyspace, ok := baseKeyspaces[node.Alias()]; ok {
			baseKeyspace.dnfPred, baseKeyspace.origPred, err = combineFilters(baseKeyspace.filters, join)
			if err != nil {
				return nil, 0, err
			}

			if baseKeyspace.dnfPred == nil {
				if join {
					// for ANSI JOIN, it's possible that one subterm of the OR only contains
					// references to other keyspaces, in which case we cannot use any index
					// scans on the current keyspace. An error will be returned by caller.
					return nil, 0, nil
				} else {
					return nil, 0, errors.NewPlanInternalError("buildOrScanNoPushdown: missing OR subterm")
				}
			}

			scan, termSargLength, err := this.buildTermScan(node, baseKeyspace, id, indexes, primaryKey, formalizer)
			if scan == nil || err != nil {
				return nil, 0, err
			}

			scans = append(scans, scan)

			if minSargLength == 0 || minSargLength > termSargLength {
				minSargLength = termSargLength
			}
		} else {
			return nil, 0, errors.NewPlanInternalError(fmt.Sprintf("buildOrScanNoPushdowns: missing basekeyspace %s", node.Alias()))
		}
	}

	rv := plan.NewUnionScan(limit, nil, scans...)
	return rv.Streamline(), minSargLength, nil
}
