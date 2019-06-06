//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package planner

import (
	"strings"

	"github.com/couchbase/query-ee/indexadvisor"
	"github.com/couchbase/query-ee/indexadvisor/iaplan"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *builder) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	this.indexAdvisor = true
	this.maxParallelism = 1
	this.queryInfos = make(map[expression.HasExpressions]*iaplan.QueryInfo, 1)
	stmt.Statement().Accept(this)
	indexadvisor.AdviseIdxs(this.queryInfos, extractDeferredIdxes(this.queryInfos, this.indexApiVersion))
	return plan.NewAdvise(plan.NewIndexAdvice(this.queryInfos), stmt.Query()), nil
}

type collectQueryInfo struct {
	keyspaceInfos  iaplan.KeyspaceInfos
	queryInfo      *iaplan.QueryInfo
	queryInfos     map[expression.HasExpressions]*iaplan.QueryInfo
	indexCollector *scanIdxCol
}

func (this *builder) initialIndexAdvisor(stmt algebra.Statement) {
	if this.indexAdvisor {
		if stmt != nil {
			this.queryInfo = iaplan.NewQueryInfo(stmt.Type())
			this.keyspaceInfos = iaplan.NewKeyspaceInfos()
			if s, ok := stmt.(*algebra.Select); ok {
				if s.Order() != nil {
					this.queryInfos[s] = this.queryInfo
				} else {
					this.queryInfos[s.Subresult()] = this.queryInfo
				}
			} else {
				this.queryInfos[stmt] = this.queryInfo
			}
		}
	}
}

func (this *builder) extractPredicates(where, on expression.Expression) {
	if this.indexAdvisor {
		if where != nil {
			this.queryInfo.SetWhere(where)
		}
		if on != nil {
			this.queryInfo.SetOn(on)
		}
	}
}

func (this *builder) extractIndexJoin(index datastore.Index, node *algebra.KeyspaceTerm, cover bool) {
	if this.indexAdvisor {
		if index != nil {
			info := extractInfo(index, node.Keyspace(), node.Alias(), false)
			if cover { //covering index
				info.SetIdxStatusCovering()
			}
			this.queryInfo.SetCurIndex(info)
		}
		if !cover {
			this.keyspaceInfos.SetUncovered()
		}
		this.queryInfo.AppendKeyspaceInfos(this.keyspaceInfos)
		this.keyspaceInfos = iaplan.NewKeyspaceInfos()
	}
}

func (this *builder) appendQueryInfo(scan plan.Operator, node *algebra.KeyspaceTerm, uncovered bool) {
	if this.indexAdvisor {
		if scan != nil {
			this.indexCollector = NewScanIdxCol()
			this.indexCollector.setNode(node)
			scan.Accept(this.indexCollector)
		}
		if uncovered {
			this.keyspaceInfos.SetUncovered()
			if this.indexCollector != nil {
				this.indexCollector.setUnCovering()
			}
		}
		this.queryInfo.AppendKeyspaceInfos(this.keyspaceInfos)
		if this.indexCollector != nil {
			this.queryInfo.AppendCurIndexes(this.indexCollector.indexInfos, this.indexCollector.covering)
		}
		this.keyspaceInfos = iaplan.NewKeyspaceInfos()
		this.indexCollector = nil
	}
}

func (this *builder) storeCollectQueryInfo() *collectQueryInfo {
	info := &collectQueryInfo{}
	info.queryInfo = this.queryInfo
	return info
}

func (this *builder) restoreCollectQueryInfo(info *collectQueryInfo) {
	this.queryInfo = info.queryInfo
}

func (this *builder) extractLetGroupProjOrder(let expression.Bindings, group *algebra.Group, projection *algebra.Projection, order *algebra.Order) {
	if this.indexAdvisor {
		if let != nil {
			this.queryInfo.SetLet(let)
		}
		if group != nil {
			this.queryInfo.SetGroup(group.By().Copy(), group.Letting())
		}
		if projection != nil {
			this.queryInfo.SetProjection(projection.Terms())
		}
		if order != nil {
			this.queryInfo.SetOrder(order.Terms())
		}
	}
}

func (this *builder) enableUnnest(alias string) {
	if this.indexAdvisor {
		if this.queryInfo.ContainsUnnest() {
			this.queryInfo.InitializeUnnestMap()
			collectInnerUnnestMap(this.from, this.queryInfo, expression.NewIdentifier(alias))
			this.queryInfo.SetUnnest(false)
		}
	}
}

func (this *builder) collectPredicates(baseKeyspace *base.BaseKeyspace, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, pred expression.Expression, ansijoin bool) error {
	if !this.indexAdvisor {
		return nil
	}
	//not advise index to system keyspace
	if strings.ToLower(keyspace.Namespace().Name()) == "#system" {
		return nil
	}
	if baseKeyspace == nil {
		baseKeyspace = this.baseKeyspaces[node.Alias()]
	}

	if pred != nil {
		if or, ok := pred.(*expression.Or); ok {
			orTerms, _ := flattenOr(or)
		outer:
			for _, op := range orTerms.Operands() {
				baseKeyspacesCopy := base.CopyBaseKeyspaces(this.baseKeyspaces)

				_, err := ClassifyExpr(op, baseKeyspacesCopy, ansijoin, this.useCBO)
				if err != nil {
					continue outer
				}

				bk, _ := baseKeyspacesCopy[node.Alias()]
				if !ansijoin {
					addUnnestPreds(baseKeyspacesCopy, bk)
				}

				p := iaplan.NewKeyspaceInfo(keyspace, node, getFilterInfos(bk.Filters()), getFilterInfos(bk.JoinFilters()), baseKeyspace.Onclause(), pred, true)
				this.keyspaceInfos = append(this.keyspaceInfos, p)
			}
		} else {
			baseKeyspacesCopy := base.CopyBaseKeyspaces(this.baseKeyspaces)
			_, err := ClassifyExpr(pred, baseKeyspacesCopy, false, this.useCBO)
			if err != nil {
				return err
			}
			baseKeyspaceCopy, _ := baseKeyspacesCopy[node.Alias()]
			p := iaplan.NewKeyspaceInfo(keyspace, node, getFilterInfos(baseKeyspaceCopy.Filters()), getFilterInfos(baseKeyspaceCopy.JoinFilters()), baseKeyspace.Onclause(), pred, false)
			this.keyspaceInfos = append(this.keyspaceInfos, p)
		}
	} else if _, ok := baseKeyspace.DnfPred().(*expression.Or); !ok {
		p := iaplan.NewKeyspaceInfo(keyspace, node, getFilterInfos(baseKeyspace.Filters()), getFilterInfos(baseKeyspace.JoinFilters()), baseKeyspace.Onclause(), baseKeyspace.DnfPred(), false)
		this.keyspaceInfos = append(this.keyspaceInfos, p)
	}
	return nil
}

func getFilterInfos(filters base.Filters) iaplan.FilterInfos {
	exprs := make(iaplan.FilterInfos, 0, len(filters))
	for _, f := range filters {
		exprs = append(exprs, iaplan.NewFilterInfo(f.FltrExpr().Copy(), f.IsUnnest()))
	}
	return exprs
}

func (this *builder) setUnnest() {
	if this.indexAdvisor {
		this.queryInfo.SetUnnest(true)
	}
}

func (this *builder) processadviseJF(alias string) {
	if this.indexAdvisor {
		this.processKeyspaceDone(alias)
	}
}

func collectInnerUnnestMap(from algebra.FromTerm, q *iaplan.QueryInfo, primaryIdentifier *expression.Identifier) {
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return
	}

	collectInnerUnnestMap(joinTerm.Left(), q, primaryIdentifier)
	unnest, ok := joinTerm.(*algebra.Unnest)
	if ok && !unnest.Outer() {
		q.AddToUnnestMap(unnest.Alias(), unnest.Expression())
		// to add the top level expression which should belong to the unnest filters
		if unnest.Expression().DependsOn(primaryIdentifier) {
			q.AddToUnnestMap(expression.NewStringer().Visit(unnest.Expression()), unnest.Expression())
		}
	}
}

func extractDeferredIdxes(queryInfos map[expression.HasExpressions]*iaplan.QueryInfo, indexApiVersion int) map[string]iaplan.IndexInfos {
	if len(queryInfos) == 0 {
		return nil
	}

	infoMap := make(map[string]iaplan.IndexInfos, 1)
	for _, queryInfo := range queryInfos {
		for _, keyspaceInfo := range queryInfo.GetKeyspaceInfos() {
			if _, ok := infoMap[keyspaceInfo.GetName()]; !ok {
				//use nil value to mark one keyspace has been processed and no deferred indexes are found or errors occur.
				infoMap[keyspaceInfo.GetName()] = getDeferredIndexes(keyspaceInfo.GetKeyspace(), keyspaceInfo.GetAlias(), indexApiVersion)
			}
		}
	}
	return infoMap
}

func getDeferredIndexes(keyspace datastore.Keyspace, alias string, indexApiVersion int) iaplan.IndexInfos {
	var infos iaplan.IndexInfos
	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil
	}

	for _, indexer := range indexers {
		idxes, err := indexer.Indexes()
		if err != nil {
			return nil
		}

		for _, idx := range idxes {
			state, _, er := idx.State()
			if er != nil || state != datastore.DEFERRED || idx.IsPrimary() {
				continue
			}

			if !useIndex2API(idx, indexApiVersion) && indexHasDesc(idx) && idx.IsPrimary() {
				continue
			}

			if infos == nil {
				infos = make(iaplan.IndexInfos, 0, 1)
			}

			infos = append(infos, extractInfo(idx, keyspace.Name(), alias, true))
		}
	}
	return infos
}
