//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build !enterprise

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func checkCostModel(featureControls uint64) {
	// no-op
}

func optDocCount(keyspace string) int64 {
	return -1
}

func optFilterSelectivity(filter *base.Filter, advisorValidate bool, context *PrepareContext) {
	return
}
func optExprSelec(keyspaces map[string]string, pred expression.Expression, advisorValidate bool,
	context *PrepareContext) (float64, float64) {
	return OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL
}

func optDefInSelec(keyspace, key string, advisorValidate bool) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optDefLikeSelec(keyspace, key string, advisorValidate bool) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optMarkIndexFilters(keys expression.Expressions, spans plan.Spans2,
	condition, filter expression.Expression, unnestAliases []string, baseKeyspace *base.BaseKeyspace) {
	// no-op
}

func optMinCost() float64 {
	return OPT_COST_NOT_AVAIL
}

func optCheckRangeExprs(baseKeyspaces map[string]*base.BaseKeyspace, advisorValidate bool,
	context *PrepareContext) {
	// no-op
}

func primaryIndexScanCost(primary datastore.PrimaryIndex, requestId string, context *PrepareContext) (
	float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func indexScanCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans SargSpans, alias, keyspace string, advisorValidate bool, context *PrepareContext) (
	float64, float64, float64, int64, float64, error) {
	return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL,
		errors.NewPlanInternalError("indexScanCost: unexpected in community edition")
}

func (this *builder) getIndexLimitCost(cost, cardinality, frCost, selec float64) (float64, float64, float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL
}

func getIndexProjectionCost(index datastore.Index, indexProjection *plan.IndexProjection,
	cardinality float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getIndexGroupAggsCost(index datastore.Index, indexGroupAggs *plan.IndexGroupAggregates,
	indexProjection *plan.IndexProjection, keyspaces map[string]string,
	cardinality float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getKeyScanCost(keys expression.Expression) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getFetchCost(keyspaceName string, cardinality float64) (float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getDistinctScanCost(index datastore.Index, cardinality float64, spans plan.Spans2,
	baseKeyspace *base.BaseKeyspace) (float64, float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getExpressionScanCost(expr expression.Expression) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getValueScanCost(pairs algebra.Pairs) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getDummyScanCost() (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getCountScanCost() (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getNLJoinCost(left, right plan.Operator, filters base.Filters, outer bool, op string) (
	float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getHashJoinCost(left, right plan.Operator, buildExprs, probeExprs expression.Expressions,
	buildRight, force bool, filters base.Filters, outer bool, op string) (
	float64, float64, int64, float64, bool) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, false
}

func getLookupJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getIndexJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string, covered bool, index datastore.Index, requestId string,
	advisorValidate bool, context *PrepareContext) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getLookupNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getIndexNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string, index datastore.Index, requestId string, advisorValidate bool,
	context *PrepareContext) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUnnestCost(node *algebra.Unnest, lastOp plan.Operator,
	baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string,
	advisorValidate bool) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getSimpleFromTermCost(left, right plan.Operator, filters base.Filters, outer bool, op string) (
	float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getSimpleFilterCost(alias string, cost, cardinality, selec float64, size int64, frCost float64) (
	float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getFilterCost(lastOp plan.Operator, expr expression.Expression,
	baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string,
	alias string, advisorValidate bool, context *PrepareContext) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getFilterCostWithInput(expr expression.Expression, baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string, alias string, cost, cardinality float64, size int64, frCost float64,
	advisorValidate bool, context *PrepareContext) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getLetCost(lastOp plan.Operator) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getWithCost(lastOp plan.Operator, with expression.Withs) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getOffsetCost(lastOp plan.Operator, noffset int64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getLimitCost(lastOp plan.Operator, nlimit, noffset int64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUnnestPredSelec(pred expression.Expression, variable string, mapping expression.Expression,
	keyspaces map[string]string, advisorValidate bool, context *PrepareContext) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optChooseIntersectScan(keyspace datastore.Keyspace, sargables map[datastore.Index]*indexEntry,
	nTerms int, baseKeyspace *base.BaseKeyspace, limit, offset expression.Expression,
	advisorValidate, singleKeyspace, vector bool, context *PrepareContext) map[datastore.Index]*indexEntry {
	return sargables
}

func getSortCost(totalSize int64, nterms int, cardinality float64, limit, offset int64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getInitialProjectCost(projection *algebra.Projection, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getGroupCosts(group *algebra.Group, aggregates algebra.Aggregates, cost, cardinality float64,
	size int64, keyspaces map[string]string, maxParallelism int) (
	float64, float64, float64, float64, float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getDistinctCost(terms algebra.ResultTerms, cost, cardinality float64, size int64, frCost float64,
	keyspaces map[string]string) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUnionDistinctCost(cost, cardinality float64, first, second plan.Operator, compatible bool) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getUnionAllCost(first, second plan.Operator, compatible bool) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getIntersectAllCost(first, second plan.Operator, compatible bool) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getExceptAllCost(first, second plan.Operator, compatible bool) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getInsertCost(key, value, options, limit expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUpsertCost(key, value, options expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getDeleteCost(limit expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getCloneCost(cost, cardinality float64, size int64, frCost float64) (
	float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUpdateSetCost(set *algebra.Set, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUpdateUnsetCost(unset *algebra.Unset, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getUpdateSendCost(limit expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getWindowAggCost(aggs algebra.Aggregates, cost, cardinality float64, size int64, frCost float64) (
	float64, float64, int64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
}

func getKeyspaceSize(keyspace string) int64 {
	return OPT_SIZE_NOT_AVAIL
}

func optGetJoinFilterSelec(selec, cardinality float64) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optChooseJoinFilters(baseKeyspace *base.BaseKeyspace, index datastore.Index) (
	float64, float64, float64, expression.Expressions) {
	return OPT_SELEC_NOT_AVAIL, OPT_COST_NOT_AVAIL, OPT_COST_NOT_AVAIL, baseKeyspace.GetAllJoinFilterExprs(index)
}

func optBuildBitFilterSize(baseKeyspace *base.BaseKeyspace, exprs expression.Expressions) (size int) {

	return int(OPT_SIZE_NOT_AVAIL)
}
