//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build !enterprise
// +build !enterprise

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

type collectQueryInfo struct {
}

func (this *builder) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	return nil, fmt.Errorf("Not supported in CE version. Use https://index-advisor.couchbase.com")
}

func (this *builder) initialIndexAdvisor(stmt algebra.Statement) {
}

func (this *builder) extractPredicates(where, on expression.Expression) {
}

func (this *builder) extractIndexJoin(index datastore.Index, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, cover bool) {
}

func (this *builder) appendQueryInfo(scan plan.Operator, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, uncovered bool) {
}

func (this *builder) enableUnnest(alias string) {
}

func (this *builder) collectPredicates(baseKeyspace *base.BaseKeyspace, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, pred expression.Expression, ansijoin bool) error {
	return nil
}

func (this *builder) setUnnest() {
}

func (this *builder) setKeyspaceFound() {
}

func (this *builder) processadviseJF(alias string) {
}

func (this *builder) extractLetGroupProjOrder(let expression.Bindings, group *algebra.Group, projection *algebra.Projection, order *algebra.Order, aggs algebra.Aggregates) {
}

func (this *builder) storeCollectQueryInfo() *collectQueryInfo {
	return nil
}

func (this *builder) restoreCollectQueryInfo(info *collectQueryInfo) {
}

func (this *builder) addVirtualIndexes(others []datastore.Index) []datastore.Index {
	return nil
}

func (this *builder) collectPushdownProperty(index datastore.Index, alias string, property PushDownProperties) {
}

func (this *builder) getIdxCandidates() []datastore.Index {
	return nil
}
