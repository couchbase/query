//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

type saveQueryInfo struct {
}

func (this *builder) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	return nil, fmt.Errorf("Not supported in CE version. Use https://index-advisor.couchbase.com")
}

func (this *builder) setAdvisePhase(op int) {
}

func (this *builder) saveQueryInfo() *saveQueryInfo {
	return nil
}

func (this *builder) restoreQueryInfo(saveQInfo *saveQueryInfo) {
}

func (this *builder) makeSubqueryInfos(l int) {
}

func (this *builder) startSubqIndexAdvisor() {
}

func (this *builder) endSubqIndexAdvisor(s *algebra.Select) {
}

func (this *builder) initialIndexAdvisor(stmt algebra.Statement) {
}

func (this *builder) extractKeyspacePredicates(where, on expression.Expression) {
}

func (this *builder) extractIndexJoin(index datastore.Index, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, cover bool, cost, cardinality float64) {
}

func (this *builder) appendQueryInfo(scan plan.Operator, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, uncovered bool) {
}

func (this *builder) enableUnnest(alias string) {
}

func (this *builder) collectPredicates(baseKeyspace *base.BaseKeyspace, keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm, pred expression.Expression, ansijoin, unnest bool) error {
	return nil
}

func (this *builder) setUnnest() {
}

func (this *builder) setKeyspaceFound() {
}

func (this *builder) processadviseJF(alias string) {
}

func (this *builder) extractPagination(order *algebra.Order, offset, limit expression.Expression) {
}

func (this *builder) extractLetGroupProjOrder(let expression.Bindings, group *algebra.Group, projection *algebra.Projection, order *algebra.Order, aggs algebra.Aggregates) {
}

func (this *builder) storeCollectQueryInfo() *collectQueryInfo {
	return nil
}

func (this *builder) restoreCollectQueryInfo(info *collectQueryInfo) {
}

func (this *builder) collectPushdownProperty(index datastore.Index, alias string, property PushDownProperties) {
}

func (this *builder) getIdxCandidates() []datastore.Index {
	return nil
}

func (this *builder) advisorValidate() bool {
	return false
}
