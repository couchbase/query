//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.
//
// +build !enterprise

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

func optCalcSelectivity(filter *base.Filter) {
	return
}

func optExprSelec(keyspaces map[string]string, pred expression.Expression) (
	float64, float64, bool) {
	return OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, false
}

func optDefInSelec() float64 {
	return OPT_SELEC_NOT_AVAIL
}

func optDefLikeSelec() float64 {
	return OPT_SELEC_NOT_AVAIL
}

func indexScanCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans SargSpans) (cost float64, sel float64, card float64, err error) {
	return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, errors.NewPlanInternalError("indexScanCost: unexpected in community edition")
}
