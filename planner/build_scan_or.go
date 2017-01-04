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
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func (this *builder) buildOrScan(node *algebra.KeyspaceTerm, id expression.Expression,
	pred *expression.Or, limit expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	plan.Operator, int, error) {

	if this.countAgg != nil {
		return this.buildOrScanTryCountPushdown(node, id, pred, limit, indexes, primaryKey, formalizer)
	} else {
		return this.buildOrScanTryPushdowns(node, id, pred, limit, indexes, primaryKey, formalizer)
	}
}

func (this *builder) buildOrScanTryCountPushdown(node *algebra.KeyspaceTerm, id expression.Expression,
	pred *expression.Or, limit expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	plan.Operator, int, error) {

	scan, sargLength, err := this.buildTermScan(node, id, pred, limit, indexes, primaryKey, formalizer)
	if err != nil {
		return nil, 0, err
	}

	switch scan.(type) {
	case *plan.IndexCountScan:
		return scan, sargLength, nil
	default:
		return this.buildOrScanTryPushdowns(node, id, pred, limit, indexes, primaryKey, formalizer)
	}
}

func (this *builder) buildOrScanTryPushdowns(node *algebra.KeyspaceTerm, id expression.Expression,
	pred *expression.Or, limit expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	plan.Operator, int, error) {

	where := this.where
	defer func() {
		this.where = where
	}()

	coveringScans := this.coveringScans
	this.countAgg = nil

	var buf [16]plan.Operator
	var scans []plan.Operator
	if len(pred.Operands()) <= len(buf) {
		scans = buf[0:0]
	} else {
		scans = make([]plan.Operator, 0, len(pred.Operands()))
	}

	var index datastore.Index
	minSargLength := 0

	for _, op := range pred.Operands() {
		this.where = op
		scan, termSargLength, err := this.buildTermScan(node, id, op, limit, indexes, primaryKey, formalizer)
		if scan == nil || err != nil {
			this.coveringScans = coveringScans
			return nil, 0, err
		}

		if distinctScan, ok := scan.(*plan.DistinctScan); ok {
			scan = distinctScan.Scan()
		}

		if indexScan, ok := scan.(*plan.IndexScan); ok {
			if index == nil {
				index = indexScan.Index()
			}

			if index == indexScan.Index() {
				scans = append(scans, scan)

				if minSargLength == 0 || minSargLength > termSargLength {
					minSargLength = termSargLength
				}

				continue
			}
		}

		// TODO: Some work is duplicated here if no scan is performing pushdowns
		this.coveringScans = coveringScans
		return this.buildOrScanNoPushdowns(node, id, pred, limit, indexes, primaryKey, formalizer)
	}

	spans := make(plan.Spans, 0, 2*len(scans))
	for _, scan := range scans {
		indexScan := scan.(*plan.IndexScan)
		spans = append(spans, indexScan.Spans()...)
	}

	spans = deDupDiscardEmptySpans(spans)
	indexScan0 := scans[0].(*plan.IndexScan)
	indexScan0.SetSpans(spans)

	if len(this.coveringScans) > len(coveringScans) {
		this.coveringScans = append(coveringScans, indexScan0)
	}

	if len(spans) > 1 {
		this.resetOrderLimit()
		indexScan0.SetLimit(nil)
		return plan.NewDistinctScan(indexScan0), minSargLength, nil
	} else {
		return indexScan0, minSargLength, nil
	}
}

func (this *builder) buildOrScanNoPushdowns(node *algebra.KeyspaceTerm, id expression.Expression,
	pred *expression.Or, limit expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	plan.Operator, int, error) {

	where := this.where
	cover := this.cover
	defer func() {
		this.where = where
		this.cover = cover
	}()

	this.cover = nil
	this.resetCountMin()

	if this.order != nil {
		this.resetOrderLimit()
		limit = nil
	}

	var buf [16]plan.Operator
	var scans []plan.Operator
	if len(pred.Operands()) <= len(buf) {
		scans = buf[0:0]
	} else {
		scans = make([]plan.Operator, 0, len(pred.Operands()))
	}

	minSargLength := 0

	for _, op := range pred.Operands() {
		this.where = op
		scan, termSargLength, err := this.buildTermScan(node, id, op, limit, indexes, primaryKey, formalizer)
		if scan == nil || err != nil {
			return nil, 0, err
		}

		scans = append(scans, scan)

		if minSargLength == 0 || minSargLength > termSargLength {
			minSargLength = termSargLength
		}
	}

	return plan.NewUnionScan(scans...), minSargLength, nil
}
