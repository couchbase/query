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
	base "github.com/couchbase/query/plannerbase"
)

func (this *builder) buildOrScan(node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace,
	id expression.Expression, pred *expression.Or, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	scan plan.SecondaryScan, sargLength int, err error) {

	indexPushDowns := this.storeIndexPushDowns()
	if this.cover != nil || this.hasOrderOrOffsetOrLimit() {
		coveringScans := this.coveringScans
		scan, sargLength, err = this.buildTermScan(node, baseKeyspace, id, indexes, primaryKey, formalizer)
		if err == nil && scan != nil {
			// covering scan or pushdown happens use the scan
			if len(this.coveringScans) > len(coveringScans) || this.countScan != nil ||
				this.hasOrderOrOffsetOrLimit() {
				return scan, sargLength, nil
			}
		}
		this.restoreIndexPushDowns(indexPushDowns, true)
	}

	// Try individual OR terms
	orScan, orSargLength, orErr := this.buildOrScanNoPushdowns(node, id, pred, indexes, primaryKey, formalizer)
	/*
	   If combined sargLength is higher than individual or use combined scan
	   ix1 ON default (c1,c2,c3)  ===> WHERE c1 = 10 AND (c2 = 20 OR (c2 = 30 AND c3 = 40))
	        Instead of 2 index scans on ix1 do 1 indexscan with 2 spans of different composite ranges
	*/
	if err == nil && scan != nil && sargLength >= orSargLength {
		return scan, sargLength, nil
	}

	return orScan, orSargLength, orErr
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

	orTerms, truth := expression.FlattenOr(pred)
	if orTerms == nil || truth {
		return nil, minSargLength, nil
	}

	cost := float64(0.0)
	cardinality := float64(0.0)
	selec := float64(1.0)
	docCount := float64(0.0)
	var err error
	useCBO := this.useCBO
	if useCBO {
		docCount, err = this.getDocCount(node)
		if err != nil || (docCount <= 0.0) {
			useCBO = false
		}
	}

	join := node.IsAnsiJoinOp()
	for i, op := range orTerms.Operands() {
		if op != nil {
			if val := op.Value(); val != nil && !val.Truth() {
				continue
			}
		}

		this.where = op
		this.limit = limit

		baseKeyspaces := base.CopyBaseKeyspaces(this.baseKeyspaces)
		_, err = ClassifyExpr(op, baseKeyspaces, this.keyspaceNames, join, this.useCBO,
			this.advisorValidate(), this.context)
		if err != nil {
			return nil, 0, err
		}

		if baseKeyspace, ok := baseKeyspaces[node.Alias()]; ok {
			if !join {
				addUnnestPreds(baseKeyspaces, baseKeyspace)
			}

			// for ANSI JOIN, all predicates in a sub-arm of OR clause are classified as ON-clause
			// filters above (since it's not easy to actually determine whether it's ON-clause or not),
			// the 3rd argument can be passed in as true in this case.
			// It's ok to classify all predicates as ON-clause since we know the entire OR clause
			// is applicable and thus can be used for index selection. This is also done on a temporary
			// bases (a copy of BaseKeyspace) and does not affect original filters.
			err = CombineFilters(baseKeyspace, join, join)
			if err != nil {
				return nil, 0, err
			}

			if baseKeyspace.DnfPred() == nil {
				if join {
					// for ANSI JOIN, it's possible that one subterm of the OR only contains
					// references to other keyspaces, in which case we cannot use any index
					// scans on the current keyspace. An error will be returned by caller.
					return nil, 0, nil
				} else {
					return nil, 0, errors.NewPlanInternalError("buildOrScanNoPushdown: missing OR subterm")
				}
			}

			scan, termSargLength, err := this.buildTermScan(node, baseKeyspace,
				id, indexes, primaryKey, formalizer)
			if scan == nil || err != nil {
				return nil, 0, err
			}

			scans = append(scans, scan)

			if minSargLength == 0 || minSargLength > termSargLength {
				minSargLength = termSargLength
			}

			scost := scan.Cost()
			scardinality := scan.Cardinality()
			if useCBO && ((scost <= 0.0) || (scardinality <= 0.0)) {
				useCBO = false
			}
			if useCBO {
				cost += scost
				selec1 := scardinality / docCount
				if selec1 > 1.0 {
					selec1 = 1.0
				}
				if i == 0 {
					selec = selec1
				} else {
					selec = selec + selec1 - (selec * selec1)
				}
			}
		} else {
			return nil, 0, errors.NewPlanInternalError(fmt.Sprintf("buildOrScanNoPushdowns: missing basekeyspace %s", node.Alias()))
		}
	}

	if useCBO {
		// cost calculated in for loop above
		cardinality = selec * docCount
	} else {
		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_CARD_NOT_AVAIL
	}

	rv := plan.NewUnionScan(limit, nil, cost, cardinality, scans...)
	return rv.Streamline(), minSargLength, nil
}
