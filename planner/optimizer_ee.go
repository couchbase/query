//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
//
// +build enterprise

package planner

import (
	"github.com/couchbase/query-ee/optimizer"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func optCalcSelectivity(filter *base.Filter) {
	optimizer.CalcSelectivity(filter)
	return
}

func optExprSelec(keyspaces map[string]string, pred expression.Expression) (
	float64, float64) {
	sel, arrSel, def := optimizer.ExprSelec(keyspaces, pred)
	if def {
		return OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL
	}
	return sel, arrSel
}

func optDefInSelec(keyspace, key string) float64 {
	return optimizer.DefInSelec(keyspace, key)
}

func optDefLikeSelec(keyspace, key string) float64 {
	return optimizer.DefLikeSelec(keyspace, key)
}

func optMarkIndexFilters(keys expression.Expressions, spans plan.Spans2,
	condition expression.Expression, filters base.Filters) {
	optimizer.MarkIndexFilters(keys, spans, condition, filters)
}

func primaryIndexScanCost(primary datastore.PrimaryIndex, requestId string) (cost, cardinality float64) {
	return optimizer.CalcPrimaryIndexScanCost(primary, requestId)
}

func indexScanCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans SargSpans, alias string) (cost float64, sel float64, card float64, err error) {
	switch spans := spans.(type) {
	case *TermSpans:
		return optimizer.CalcIndexScanCost(index, sargKeys, requestId, spans.spans, alias)
	case *IntersectSpans:
		return multiIndexCost(index, sargKeys, requestId, spans.spans, alias, false)
	case *UnionSpans:
		return multiIndexCost(index, sargKeys, requestId, spans.spans, alias, true)
	}

	return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, errors.NewPlanInternalError("indexScanCost: unexpected span type")
}

func multiIndexCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans []SargSpans, alias string, union bool) (cost float64, sel float64, card float64, err error) {
	var nrows float64
	for i, span := range spans {
		tcost, tsel, tcard, e := indexScanCost(index, sargKeys, requestId, span, alias)
		if e != nil {
			return tcost, tsel, tcard, e
		}
		cost += tcost
		tnrows := tcard / tsel
		if i == 0 {
			sel = tsel
			nrows = tnrows
		} else {
			tsel = tsel * (tnrows / nrows)
			if union {
				sel = sel + tsel - (sel * tsel)
			} else {
				sel = sel * tsel
			}
		}
	}

	return cost, sel, (sel * nrows), nil
}

func getKeyScanCost(keys expression.Expression) (float64, float64) {
	return optimizer.CalcKeyScanCost(keys)
}

func getFetchCost(keyspace datastore.Keyspace, cardinality float64) float64 {
	return optimizer.CalcFetchCost(keyspace, cardinality)
}

func getDistinctScanCost(index datastore.Index, cardinality float64) (float64, float64) {
	return optimizer.CalcDistinctScanCost(index, cardinality)
}

func getExpressionScanCost(expr expression.Expression, keyspaces map[string]string) (float64, float64) {
	return optimizer.CalcExpressionScanCost(expr, keyspaces)
}

func getValueScanCost(pairs algebra.Pairs) (float64, float64) {
	return optimizer.CalcValueScanCost(pairs)
}

func getDummyScanCost() (float64, float64) {
	return optimizer.CalcDummyScanCost()
}

func getCountScanCost() (float64, float64) {
	return optimizer.CalcCountScanCost()
}

func getNLJoinCost(left, right plan.Operator, filters base.Filters, outer bool, op string) (float64, float64) {
	jointype := optimizer.COST_JOIN
	if op == "nest" {
		jointype = optimizer.COST_NEST
	}
	return optimizer.CalcNLJoinCost(left, right, filters, outer, jointype)
}

func getHashJoinCost(left, right plan.Operator, buildExprs, probeExprs expression.Expressions,
	buildRight, force bool, filters base.Filters, outer bool, op string) (float64, float64, bool) {
	jointype := optimizer.COST_JOIN
	if op == "nest" {
		jointype = optimizer.COST_NEST
	}
	return optimizer.CalcHashJoinCost(left, right, buildExprs, probeExprs, buildRight, force,
		filters, outer, jointype)
}

func getLookupJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace) (float64, float64) {
	return optimizer.CalcLookupJoinNestCost(left, outer, right, rightKeyspace, optimizer.COST_JOIN)
}

func getIndexJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace, covered bool, index datastore.Index,
	requestId string) (float64, float64) {
	return optimizer.CalcIndexJoinNestCost(left, outer, right, rightKeyspace, covered,
		index, requestId, optimizer.COST_JOIN)
}

func getLookupNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace) (float64, float64) {
	return optimizer.CalcLookupJoinNestCost(left, outer, right, rightKeyspace, optimizer.COST_NEST)
}

func getIndexNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace, index datastore.Index, requestId string) (float64, float64) {
	return optimizer.CalcIndexJoinNestCost(left, outer, right, rightKeyspace, false,
		index, requestId, optimizer.COST_NEST)
}

func getUnnestCost(node *algebra.Unnest, lastOp plan.Operator, keyspaces map[string]string) (float64, float64) {
	return optimizer.CalcUnnestCost(node, lastOp, keyspaces)
}

func getSimpleFromTermCost(left, right plan.Operator, filters base.Filters) (float64, float64) {
	return optimizer.CalcSimpleFromTermCost(left, right, filters)
}

func getFilterCost(lastOp plan.Operator, expr expression.Expression,
	baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string) (float64, float64) {

	// perform expression transformation, but no DNF transformation
	var err error
	dnfExpr := expr.Copy()
	dnf := NewDNF(dnfExpr, true, false)
	dnfExpr, err = dnf.Map(dnfExpr)
	if err != nil {
		return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
	}

	return optimizer.CalcFilterCost(lastOp, dnfExpr, baseKeyspaces, keyspaceNames)
}

func getLetCost(lastOp plan.Operator) (float64, float64) {
	return optimizer.CalcLetCost(lastOp)
}

func getOffsetCost(lastOp plan.Operator, noffset int64) (float64, float64) {
	return optimizer.CalcOffsetCost(lastOp, noffset)
}

func getLimitCost(lastOp plan.Operator, nlimit int64) (float64, float64) {
	return optimizer.CalcLimitCost(lastOp, nlimit)
}

func getUnnestPredSelec(pred expression.Expression, variable string, mapping expression.Expression,
	keyspaces map[string]string) float64 {
	return optimizer.GetUnnestPredSelec(pred, variable, mapping, keyspaces)
}

func optChooseIntersectScan(keyspace datastore.Keyspace, indexes map[datastore.Index]*base.IndexCost) map[datastore.Index]*base.IndexCost {
	return optimizer.ChooseIntersectScan(keyspace, indexes)
}

func getSortCost(nterms int, cardinality float64, limit, offset int64) (float64, float64) {
	return optimizer.CalcSortCost(nterms, cardinality, limit, offset)
}

func getInitialProjectCost(projection *algebra.Projection, cardinality float64) (float64, float64) {
	return optimizer.CalcInitialProjectionCost(projection, cardinality)
}

func getGroupCosts(group *algebra.Group, aggregates algebra.Aggregates, cost, cardinality float64,
	keyspaces map[string]string, maxParallelism int) (
	float64, float64, float64, float64, float64, float64) {
	if maxParallelism <= 0 {
		maxParallelism = plan.GetMaxParallelism()
	}
	return optimizer.CalcGroupCosts(group, aggregates, cost, cardinality, keyspaces, maxParallelism)
}

func getDistinctCost(terms algebra.ResultTerms, cardinality float64, keyspaces map[string]string) (float64, float64) {
	return optimizer.CalcDistinctCost(terms, cardinality, keyspaces)
}

func getUnionDistinctCost(cost, cardinality float64, first, second plan.Operator, compatible bool) (float64, float64) {
	return optimizer.CalcUnionDistinctCost(cost, cardinality, first, second, compatible)
}

func getUnionAllCost(first, second plan.Operator, compatible bool) (float64, float64) {
	return optimizer.CalcSetOpCost(first, second, compatible, optimizer.COST_UNION)
}

func getIntersectAllCost(first, second plan.Operator, compatible bool) (float64, float64) {
	return optimizer.CalcSetOpCost(first, second, compatible, optimizer.COST_INTERSECT)
}

func getExceptAllCost(first, second plan.Operator, compatible bool) (float64, float64) {
	return optimizer.CalcSetOpCost(first, second, compatible, optimizer.COST_EXCEPT)
}
