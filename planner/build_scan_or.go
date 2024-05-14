//  Copyright 2014-Present Couchbase, Inc.
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
)

const (
	_LARGE_OR_SUBTERMS        = 64
	_LARGE_UNIONSCAN_CHILDREN = 16
)

func (this *builder) buildOrScan(node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace,
	id expression.Expression, pred *expression.Or, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	scan plan.SecondaryScan, sargLength int, err error) {

	var cost, cardinality, orCost, orCardinality float64
	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())

	indexPushDowns := this.storeIndexPushDowns()
	indexHintErr := false
	if this.cover != nil || this.hasOrderOrOffsetOrLimit() {
		coveringScans := this.coveringScans
		scan, sargLength, err = this.buildTermScan(node, baseKeyspace, id, indexes, primaryKey, formalizer)
		if err == nil && scan != nil {
			// covering scan or pushdown happens use the scan
			// if OR expr has large number of subterms, instead of trying to use
			// a UnionScan with larger number of children, just use the current scan
			if len(this.coveringScans) > len(coveringScans) || this.countScan != nil ||
				this.hasOrderOrOffsetOrLimit() || len(pred.Operands()) > _LARGE_OR_SUBTERMS {
				return scan, sargLength, nil
			}
			if baseKeyspace.HasIndexHintError() {
				indexHintErr = true
				baseKeyspace.UnsetIndexHintError()
			}
		}
		this.restoreIndexPushDowns(indexPushDowns, true)
	}

	if this.hintIndexes && baseKeyspace.HasIndexAllHint() {
		// do not consider UNION scans if INDEX_ALL hint is in effect
		return nil, 0, nil
	}

	if useCBO {
		if scan != nil {
			cost = scan.Cost()
			cardinality = scan.Cardinality()
			if cost <= 0.0 || cardinality <= 0.0 {
				useCBO = false
			} else {
				fetchCost, _, _ := getFetchCost(baseKeyspace.Keyspace(), cardinality)
				if fetchCost > 0.0 {
					cost += fetchCost
				} else {
					useCBO = false
				}
			}
		} else {
			useCBO = false
		}
	}

	// Try individual OR terms
	orScan, orSargLength, orIndexHintErr, orErr := this.buildOrScanNoPushdowns(node, id, pred, indexes, primaryKey, formalizer)

	if scan != nil && orScan != nil {
		if indexHintErr && !orIndexHintErr {
			return orScan, orSargLength, orErr
		} else if !indexHintErr && orIndexHintErr {
			return scan, sargLength, err
		} else if indexHintErr || orIndexHintErr {
			baseKeyspace.SetIndexHintError()
		}
	} else if scan != nil {
		if indexHintErr {
			baseKeyspace.SetIndexHintError()
		}
	} else if orScan != nil {
		if orIndexHintErr {
			baseKeyspace.SetIndexHintError()
		}
	}

	if useCBO && orScan != nil {
		orCost = orScan.Cost()
		orCardinality = orScan.Cardinality()
		if orCost > 0.0 && orCardinality > 0.0 {
			fetchCost, _, _ := getFetchCost(baseKeyspace.Keyspace(), orCardinality)
			if fetchCost > 0.0 {
				orCost += fetchCost
				if cost < orCost || (cost == orCost && cardinality < orCardinality) {
					return scan, sargLength, err
				} else if orCost < cost || (orCost == cost && orCardinality < cardinality) {
					return orScan, orSargLength, orErr
				}
			}
		}
	}

	/*
	   If combined sargLength is higher than individual or use combined scan
	   ix1 ON default (c1,c2,c3)  ===> WHERE c1 = 10 AND (c2 = 20 OR (c2 = 30 AND c3 = 40))
	        Instead of 2 index scans on ix1 do 1 indexscan with 2 spans of different composite ranges
	*/
	if err == nil && scan != nil {
		if orErr != nil || orScan == nil || sargLength > orSargLength {
			return scan, sargLength, nil
		} else {
			if sargLength == orSargLength {
				if unionScan, ok := orScan.(*plan.UnionScan); ok && len(unionScan.Scans()) > _LARGE_UNIONSCAN_CHILDREN {
					return scan, sargLength, nil
				}
			}
			idx := scan.GetIndex()
			orIdx := orScan.GetIndex()
			if idx != nil && !idx.IsPrimary() && orIdx != nil {
				// if the UNION SCAN uses the same index underneath, just
				// do the regular index scan
				return scan, sargLength, nil
			}
		}
	}

	return orScan, orSargLength, orErr
}

func (this *builder) buildOrScanNoPushdowns(node *algebra.KeyspaceTerm, id expression.Expression,
	pred *expression.Or, indexes []datastore.Index, primaryKey expression.Expressions,
	formalizer *expression.Formalizer) (plan.SecondaryScan, int, bool, error) {

	where := this.where
	cover := this.cover
	this.setBuilderFlag(BUILDER_OR_SUBTERM)

	defer func() {
		this.where = where
		this.cover = cover
		this.unsetBuilderFlag(BUILDER_OR_SUBTERM)
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

	maxSargLength := 0

	orTerms, truth := expression.FlattenOr(pred)
	if orTerms == nil || truth {
		return nil, 0, false, nil
	}

	cost := float64(0.0)
	cardinality := float64(0.0)
	selec := float64(1.0)
	docCount := float64(0.0)
	size := int64(0)
	frCost := float64(0.0)
	var err error
	useCBO := this.useCBO
	if useCBO {
		cnt := this.getDocCount(node.Alias())
		if cnt > 0 {
			docCount = float64(cnt)
		} else if cnt == 0 {
			// empty keyspace, use 1 instead to avoid divide by 0
			docCount = float64(1.0)
		} else {
			useCBO = false
		}
	}

	join := node.IsAnsiJoinOp()
	hintErr := false
	for i, op := range orTerms.Operands() {
		if op != nil {
			if val := op.Value(); val != nil && !val.Truth() {
				continue
			}
		}

		this.where = op
		this.limit = limit

		var extraExpr expression.Expression
		baseKeyspaces := base.CopyBaseKeyspaces(this.baseKeyspaces)
		extraExpr, err = this.processPredicateBase(op, baseKeyspaces, join)
		if err != nil {
			return nil, 0, false, err
		}

		if baseKeyspace, ok := baseKeyspaces[node.Alias()]; ok {
			if !join {
				err = addUnnestPreds(baseKeyspaces, baseKeyspace)
				if err != nil {
					return nil, 0, false, err
				}
			}

			err = CombineFilters(baseKeyspace, join)
			if err != nil {
				return nil, 0, false, err
			}

			if baseKeyspace.DnfPred() == nil {
				if join || (extraExpr != nil && extraExpr.Value() == nil) {
					// if an arm of OR does not reference the keyspace,
					// which could happen if:
					//   - in case of ANSI JOIN, an arm references other keyspaces
					//   - an arm only references named/positional parameters
					// then OR index path is not feasible.
					return nil, 0, false, nil
				} else {
					return nil, 0, false, errors.NewPlanInternalError("buildOrScanNoPushdown: missing OR subterm")
				}
			}

			scan, termSargLength, err := this.buildTermScan(node, baseKeyspace,
				id, indexes, primaryKey, formalizer)
			if scan == nil || err != nil {
				return nil, 0, false, err
			}

			scans = append(scans, scan)

			if maxSargLength == 0 || maxSargLength < termSargLength {
				maxSargLength = termSargLength
			}

			if baseKeyspace.HasIndexHintError() {
				hintErr = true
				baseKeyspace.UnsetIndexHintError()
			}

			scost := scan.Cost()
			scardinality := scan.Cardinality()
			ssize := scan.Size()
			sfrCost := scan.FrCost()
			if useCBO && ((scost <= 0.0) || (scardinality <= 0.0) || (ssize <= 0) || (sfrCost <= 0.0)) {
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
					size = ssize
					frCost = sfrCost
				} else {
					selec = selec + selec1 - (selec * selec1)
					if ssize > size {
						size = ssize
					}
				}
			}
		} else {
			return nil, 0, false, errors.NewPlanInternalError(fmt.Sprintf("buildOrScanNoPushdowns: missing basekeyspace %s",
				node.Alias()))
		}
	}

	if useCBO {
		// cost calculated in for loop above
		cardinality = selec * docCount
	} else {
		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_CARD_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
	}

	rv := plan.NewUnionScan(limit, nil, cost, cardinality, size, frCost, scans...)
	return rv.Streamline(), maxSargLength, hintErr, nil
}
