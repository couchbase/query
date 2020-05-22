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
	"github.com/couchbase/query-ee/optutil"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func checkCostModel(featureControls uint64) {
	if util.IsFeatureEnabled(featureControls, util.N1QL_CBO_NEW) {
		optutil.SetNewCostModel()
	}
}

func optExprSelec(keyspaces map[string]string, pred expression.Expression) (
	float64, float64) {
	sel, arrSel, def := optutil.ExprSelec(keyspaces, pred)
	if def {
		return OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL
	}
	return sel, arrSel
}

func optDefInSelec(keyspace, key string) float64 {
	return optutil.DefInSelec(keyspace, key)
}

func optDefLikeSelec(keyspace, key string) float64 {
	return optutil.DefLikeSelec(keyspace, key)
}

func optMarkIndexFilters(keys expression.Expressions, spans plan.Spans2,
	condition expression.Expression, filters base.Filters) {
	optutil.MarkIndexFilters(keys, spans, condition, filters)
}

func optMinCost() float64 {
	return optutil.MinCost()
}

func primaryIndexScanCost(primary datastore.PrimaryIndex, requestId string) (cost, cardinality float64) {
	return optutil.CalcPrimaryIndexScanCost(primary, requestId)
}

func indexScanCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans SargSpans, alias string) (cost float64, sel float64, card float64, err error) {
	switch spans := spans.(type) {
	case *TermSpans:
		return optutil.CalcIndexScanCost(index, sargKeys, requestId, spans.spans, alias)
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

func getIndexProjectionCost(index datastore.Index, indexProjection *plan.IndexProjection,
	cardinality float64) (float64, float64) {
	return optutil.CalcIndexProjectionCost(index, indexProjection, cardinality, 0, 0, 0)
}

func getIndexGroupAggsCost(index datastore.Index, indexGroupAggs *plan.IndexGroupAggregates,
	indexProjection *plan.IndexProjection, keyspaces map[string]string,
	cardinality float64) (float64, float64) {
	return optutil.CalcIndexGroupAggsCost(index, indexGroupAggs, indexProjection, keyspaces, cardinality)
}

func getKeyScanCost(keys expression.Expression) (float64, float64) {
	return optutil.CalcKeyScanCost(keys)
}

func getFetchCost(keyspace datastore.Keyspace, cardinality float64) float64 {
	return optutil.CalcFetchCost(keyspace, cardinality)
}

func getDistinctScanCost(index datastore.Index, cardinality float64) (float64, float64) {
	return optutil.CalcDistinctScanCost(index, cardinality)
}

func getExpressionScanCost(expr expression.Expression, keyspaces map[string]string) (float64, float64) {
	return optutil.CalcExpressionScanCost(expr, keyspaces)
}

func getValueScanCost(pairs algebra.Pairs) (float64, float64) {
	return optutil.CalcValueScanCost(pairs)
}

func getDummyScanCost() (float64, float64) {
	return optutil.CalcDummyScanCost()
}

func getCountScanCost() (float64, float64) {
	return optutil.CalcCountScanCost()
}

func getNLJoinCost(left, right plan.Operator, filters base.Filters, outer bool, op string) (float64, float64) {
	jointype := optutil.COST_JOIN
	if op == "nest" {
		jointype = optutil.COST_NEST
	}
	return optutil.CalcNLJoinCost(left, right, filters, outer, jointype)
}

func getHashJoinCost(left, right plan.Operator, buildExprs, probeExprs expression.Expressions,
	buildRight, force bool, filters base.Filters, outer bool, op string) (float64, float64, bool) {
	jointype := optutil.COST_JOIN
	if op == "nest" {
		jointype = optutil.COST_NEST
	}
	return optutil.CalcHashJoinCost(left, right, buildExprs, probeExprs, buildRight, force,
		filters, outer, jointype)
}

func getLookupJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace) (float64, float64) {
	return optutil.CalcLookupJoinNestCost(left, outer, right, rightKeyspace, optutil.COST_JOIN)
}

func getIndexJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace, covered bool, index datastore.Index,
	requestId string) (float64, float64) {
	return optutil.CalcIndexJoinNestCost(left, outer, right, rightKeyspace, covered,
		index, requestId, optutil.COST_JOIN)
}

func getLookupNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace) (float64, float64) {
	return optutil.CalcLookupJoinNestCost(left, outer, right, rightKeyspace, optutil.COST_NEST)
}

func getIndexNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace *base.BaseKeyspace, index datastore.Index, requestId string) (float64, float64) {
	return optutil.CalcIndexJoinNestCost(left, outer, right, rightKeyspace, false,
		index, requestId, optutil.COST_NEST)
}

func getUnnestCost(node *algebra.Unnest, lastOp plan.Operator, keyspaces map[string]string) (float64, float64) {
	return optutil.CalcUnnestCost(node, lastOp, keyspaces)
}

func getSimpleFromTermCost(left, right plan.Operator, filters base.Filters) (float64, float64) {
	return optutil.CalcSimpleFromTermCost(left, right, filters)
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

	return optutil.CalcFilterCost(lastOp, dnfExpr, baseKeyspaces, keyspaceNames)
}

func getFilterCostWithInput(expr expression.Expression, baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string, cost, cardinality float64) (float64, float64) {

	// perform expression transformation, but no DNF transformation
	var err error
	dnfExpr := expr.Copy()
	dnf := NewDNF(dnfExpr, true, false)
	dnfExpr, err = dnf.Map(dnfExpr)
	if err != nil {
		return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
	}

	return optutil.CalcFilterCostWithInput(dnfExpr, baseKeyspaces, keyspaceNames, cost, cardinality)
}

func getLetCost(lastOp plan.Operator) (float64, float64) {
	return optutil.CalcLetCost(lastOp)
}

func getWithCost(lastOp plan.Operator, with expression.Bindings) (float64, float64) {
	return optutil.CalcWithCost(lastOp, with)
}

func getOffsetCost(lastOp plan.Operator, noffset int64) (float64, float64) {
	return optutil.CalcOffsetCost(lastOp, noffset)
}

func getLimitCost(lastOp plan.Operator, nlimit int64) (float64, float64) {
	return optutil.CalcLimitCost(lastOp, nlimit)
}

func getUnnestPredSelec(pred expression.Expression, variable string, mapping expression.Expression,
	keyspaces map[string]string) float64 {
	return optutil.GetUnnestPredSelec(pred, variable, mapping, keyspaces)
}

func optChooseIntersectScan(keyspace datastore.Keyspace, indexes map[datastore.Index]*base.IndexCost) map[datastore.Index]*base.IndexCost {
	return optutil.ChooseIntersectScan(keyspace, indexes)
}

func getSortCost(nterms int, cardinality float64, limit, offset int64) (float64, float64) {
	return optutil.CalcSortCost(nterms, cardinality, limit, offset)
}

func getInitialProjectCost(projection *algebra.Projection, cardinality float64) (float64, float64) {
	return optutil.CalcInitialProjectionCost(projection, cardinality)
}

func getGroupCosts(group *algebra.Group, aggregates algebra.Aggregates, cost, cardinality float64,
	keyspaces map[string]string, maxParallelism int) (
	float64, float64, float64, float64, float64, float64) {
	if maxParallelism <= 0 {
		maxParallelism = plan.GetMaxParallelism()
	}
	return optutil.CalcGroupCosts(group, aggregates, cost, cardinality, keyspaces, maxParallelism)
}

func getDistinctCost(terms algebra.ResultTerms, cardinality float64, keyspaces map[string]string) (float64, float64) {
	return optutil.CalcDistinctCost(terms, cardinality, keyspaces)
}

func getUnionDistinctCost(cost, cardinality float64, first, second plan.Operator, compatible bool) (float64, float64) {
	return optutil.CalcUnionDistinctCost(cost, cardinality, first, second, compatible)
}

func getUnionAllCost(first, second plan.Operator, compatible bool) (float64, float64) {
	return optutil.CalcSetOpCost(first, second, compatible, optutil.COST_UNION)
}

func getIntersectAllCost(first, second plan.Operator, compatible bool) (float64, float64) {
	return optutil.CalcSetOpCost(first, second, compatible, optutil.COST_INTERSECT)
}

func getExceptAllCost(first, second plan.Operator, compatible bool) (float64, float64) {
	return optutil.CalcSetOpCost(first, second, compatible, optutil.COST_EXCEPT)
}

func getInsertCost(keyspace datastore.Keyspace, key, value, options, limit expression.Expression,
	cost, cardinality float64) (float64, float64) {
	return optutil.CalcInsertCost(keyspace, key, value, options, limit, cost, cardinality)
}

func getUpsertCost(keyspace datastore.Keyspace, key, value, options expression.Expression,
	cost, cardinality float64) (float64, float64) {
	return optutil.CalcUpsertCost(keyspace, key, value, options, cost, cardinality)
}

func getDeleteCost(keyspace datastore.Keyspace, limit expression.Expression,
	cost, cardinality float64) (float64, float64) {
	return optutil.CalcDeleteCost(keyspace, limit, cost, cardinality)
}

func getCloneCost(keyspace datastore.Keyspace, cost, cardinality float64) (float64, float64) {
	return optutil.CalcCloneCost(keyspace, cost, cardinality)
}

func getUpdateSetCost(keyspace datastore.Keyspace, set *algebra.Set, cost, cardinality float64) (float64, float64) {
	return optutil.CalcUpdateSetCost(keyspace, set, cost, cardinality)
}

func getUpdateUnsetCost(keyspace datastore.Keyspace, unset *algebra.Unset, cost, cardinality float64) (float64, float64) {
	return optutil.CalcUpdateUnsetCost(keyspace, unset, cost, cardinality)
}

func getUpdateSendCost(keyspace datastore.Keyspace, limit expression.Expression,
	cost, cardinality float64) (float64, float64) {
	return optutil.CalcUpdateSendCost(keyspace, limit, cost, cardinality)
}

func getWindowAggCost(aggs algebra.Aggregates, cost, cardinality float64) (float64, float64) {
	return optutil.CalcWindowAggCost(aggs, cost, cardinality)
}
