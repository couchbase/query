//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
//
//go:build !enterprise

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func optCalcSelectivity(filter *base.Filter, context *PrepareContext) {
	return
}

func optExprSelec(keyspaces map[string]string, pred expression.Expression, context *PrepareContext) (
	float64, float64) {
	return OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL
}

func optDefInSelec(keyspace, key string) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optDefLikeSelec(keyspace, key string) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optMarkIndexFilters(keys expression.Expressions, spans plan.Spans2,
	condition expression.Expression, filters base.Filters) {
	// no-op
}

func primaryIndexScanCost(primary datastore.PrimaryIndex, requestId string, context *PrepareContext) (
	cost, cardinality float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func indexScanCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans SargSpans, alias string, context *PrepareContext) (
	cost float64, sel float64, card float64, err error) {
	return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, errors.NewPlanInternalError("indexScanCost: unexpected in community edition")
}

func getFetchCost(keyspace datastore.Keyspace, cardinality float64) float64 {
	return OPT_COST_NOT_AVAIL
}

func getDistinctScanCost(index datastore.Index, cardinality float64) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getExpressionScanCost(expr expression.Expression, keyspaces map[string]string) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getNLJoinCost(left, right plan.Operator, filters base.Filters) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getHashJoinCost(left, right plan.Operator, buildExprs, probeExprs expression.Expressions,
	buildRight, force bool, filters base.Filters) (float64, float64, bool) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, false
}

func getSimpleFromTermCost(left, right plan.Operator, filters base.Filters) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getFilterCost(lastOp plan.Operator, expr expression.Expression,
	baseKeyspaces map[string]*base.BaseKeyspace, context *PrepareContext) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getLetCost(lastOp plan.Operator) (float64, float64) {
	return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL
}

func getUnnestPredSelec(pred expression.Expression, variable string, mapping expression.Expression,
	keyspaces map[string]string, context *PrepareContext) float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optChooseIntersectScan(keyspace datastore.Keyspace, indexes map[datastore.Index]*base.IndexCost) map[datastore.Index]*base.IndexCost {
	return indexes
}
