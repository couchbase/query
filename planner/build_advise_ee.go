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

const (
	_RECOMMEND = iota
	_VALIDATE
)

var pushdownMap = map[PushDownProperties]string{
	_PUSHDOWN_LIMIT:         "LIMIT pushdown",
	_PUSHDOWN_OFFSET:        "OFFSET pushdown",
	_PUSHDOWN_ORDER:         "ORDER pushdown",
	_PUSHDOWN_GROUPAGGS:     "GROUPBY & AGGREGATES pushdown",
	_PUSHDOWN_FULLGROUPAGGS: "FULL GROUPBY & AGGREGATES pushdown",
}

func (this *builder) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	this.setAdvisePhase(_RECOMMEND)
	//Temporarily turn off CBO for rule-based advisor
	this.useCBO = false
	this.maxParallelism = 1
	this.queryInfos = make(map[expression.HasExpressions]*iaplan.QueryInfo, 1)
	stmt.Statement().Accept(this)
	coverIdxMap := indexadvisor.AdviseIdxs(this.queryInfos, extractDeferredIdxes(this.queryInfos, this.context.IndexApiVersion()), doDNF(stmt.Statement().Expressions()))

	this.setAdvisePhase(_VALIDATE)
	//There are covering indexes to be validated:
	if len(coverIdxMap) > 0 {
		this.idxCandidates = make([]datastore.Index, 0, len(coverIdxMap))
		for _, info := range coverIdxMap {
			idx := info.VirtualIndex()
			if idx != nil {
				this.idxCandidates = append(this.idxCandidates, idx)
			}
		}

		if len(this.idxCandidates) > 0 {
			stmt.Statement().Accept(this)
			if len(this.validatedCoverIdxes) > 0 {
				this.matchIdxInfos(coverIdxMap)
			}
		}
	}
	return plan.NewAdvise(plan.NewIndexAdvice(this.queryInfos, this.validatedCoverIdxes), stmt.Query()), nil
}

func (this *builder) matchIdxInfos(m map[string]*iaplan.IndexInfo) {
	for i, info := range this.validatedCoverIdxes {
		key := info.GetKeyspaceName() + "_" + info.GetAlias() + "_" + info.GetIndexName() + "_virtual"
		if origInfo, ok := m[key]; ok {
			this.validatedCoverIdxes[i] = this.matchPushdownProperty(key, origInfo)
		}
	}
	return
}

type collectQueryInfo struct {
	keyspaceInfos       iaplan.KeyspaceInfos
	queryInfo           *iaplan.QueryInfo
	queryInfos          map[expression.HasExpressions]*iaplan.QueryInfo
	indexCollector      *scanIdxCol
	idxCandidates       []datastore.Index
	validatedCoverIdxes iaplan.IndexInfos
	pushDownPropMap     map[string]PushDownProperties // key->"keyspace_alias_indexname_typeofIdx" e.g. "default_d_idx1_virtual"
	advisePhase         int
}

func (this *builder) setAdvisePhase(op int) {
	this.indexAdvisor = true
	this.advisePhase = op
}

func (this *builder) initialIndexAdvisor(stmt algebra.Statement) {
	if this.indexAdvisor {
		if this.advisePhase == _RECOMMEND {
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
		} else {
			this.validatedCoverIdxes = make(iaplan.IndexInfos, 0, 1)
		}
	}
}

func (this *builder) extractPredicates(where, on expression.Expression) {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		if where != nil {
			this.queryInfo.SetWhere(where)
		}
		if on != nil {
			this.queryInfo.SetOn(on)
		}
	}
}

func (this *builder) extractIndexJoin(index datastore.Index, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, cover bool) {
	if this.indexAdvisor {
		if index != nil {
			info := extractInfo(index, node.Alias(), keyspace, false, this.advisePhase == _VALIDATE)
			if cover { //covering index
				info.SetIdxStatusCovering()
			}
			if this.advisePhase == _VALIDATE {
				this.validatedCoverIdxes = append(this.validatedCoverIdxes, info)
				return
			}
			this.queryInfo.SetCurIndex(info)
		}
		if this.advisePhase == _VALIDATE {
			if !cover {
				this.keyspaceInfos.SetUncovered()
			}
			this.queryInfo.AppendKeyspaceInfos(this.keyspaceInfos)
			this.keyspaceInfos = iaplan.NewKeyspaceInfos()
		}
	}
}

func (this *builder) appendQueryInfo(scan plan.Operator, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, uncovered bool) {
	if this.indexAdvisor {
		// Index collector collects index information in both recommend and validate phases.
		if scan != nil {
			this.indexCollector = NewScanIdxCol()
			this.indexCollector.setKeyspace(keyspace)
			this.indexCollector.setAlias(node.Alias())
			this.indexCollector.setValidatePhase(this.advisePhase == _VALIDATE)
			scan.Accept(this.indexCollector)
			if uncovered {
				this.indexCollector.setUnCovering()
			}
			if this.advisePhase == _VALIDATE {
				if this.indexCollector.isCovering() {
					this.validatedCoverIdxes = append(this.validatedCoverIdxes, this.indexCollector.indexInfos...)
				}
				this.indexCollector = nil
				return
			}
			this.queryInfo.AppendCurIndexes(this.indexCollector.indexInfos, this.indexCollector.covering)
		}

		if this.advisePhase == _RECOMMEND {
			if uncovered {
				this.keyspaceInfos.SetUncovered()
			}
			this.queryInfo.AppendKeyspaceInfos(this.keyspaceInfos)
			this.keyspaceInfos = iaplan.NewKeyspaceInfos()
			this.indexCollector = nil
		}
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

func (this *builder) extractLetGroupProjOrder(let expression.Bindings, group *algebra.Group, projection *algebra.Projection, order *algebra.Order, aggs algebra.Aggregates) {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
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
		if len(aggs) > 0 {
			this.queryInfo.SetAggs(aggs)
		}
	}
}

func (this *builder) enableUnnest(alias string) {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		if this.queryInfo.ContainsUnnest() {
			this.queryInfo.InitializeUnnestMap()
			collectInnerUnnestMap(this.from, this.queryInfo, expression.NewIdentifier(alias), 1)
			this.queryInfo.SetUnnest(false)
		}
	}
}

func (this *builder) collectPredicates(baseKeyspace *base.BaseKeyspace, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, pred expression.Expression, ansijoin bool) error {
	if !(this.indexAdvisor && this.advisePhase == _RECOMMEND) {
		return nil
	}
	//not advise index to system keyspace
	if strings.ToLower(keyspace.Namespace().Name()) == "#system" {
		return nil
	}
	if baseKeyspace == nil {
		baseKeyspace = this.baseKeyspaces[node.Alias()]
	}

	if pred == nil {
		if _, ok := baseKeyspace.DnfPred().(*expression.Or); !ok {
			p := iaplan.NewKeyspaceInfo(keyspace, node, getFilterInfos(baseKeyspace.Filters()), getFilterInfos(baseKeyspace.JoinFilters()), baseKeyspace.Onclause(), baseKeyspace.DnfPred(), false, nil)
			this.keyspaceInfos = append(this.keyspaceInfos, p)
		} else {
			pred = baseKeyspace.DnfPred()
		}
	}

	if pred != nil {
		if or, ok := pred.(*expression.Or); ok {
			orTerms, _ := flattenOr(or)
			var predConjunc expression.Expressions
			if andTerm, ok := baseKeyspace.OrigPred().(*expression.And); ok {
				predConjunc = getAndTerms(andTerm)
			}
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
				p := iaplan.NewKeyspaceInfo(keyspace, node, getFilterInfos(bk.Filters()), getFilterInfos(bk.JoinFilters()), baseKeyspace.Onclause(), op, true, predConjunc)
				this.keyspaceInfos = append(this.keyspaceInfos, p)
			}
		} else {
			baseKeyspacesCopy := base.CopyBaseKeyspaces(this.baseKeyspaces)
			_, err := ClassifyExpr(pred, baseKeyspacesCopy, false, this.useCBO)
			if err != nil {
				return err
			}
			baseKeyspaceCopy, _ := baseKeyspacesCopy[node.Alias()]
			p := iaplan.NewKeyspaceInfo(keyspace, node, getFilterInfos(baseKeyspaceCopy.Filters()), getFilterInfos(baseKeyspaceCopy.JoinFilters()), baseKeyspace.Onclause(), pred, false, nil)
			this.keyspaceInfos = append(this.keyspaceInfos, p)
		}
	}
	return nil
}

func (this *builder) addVirtualIndexes(others []datastore.Index) []datastore.Index {
	if len(this.idxCandidates) > 0 {
		others = append(others, this.idxCandidates...)
	}
	return others
}

func (this *builder) collectPushdownProperty(index datastore.Index, alias string, property PushDownProperties) {
	if this.advisePhase == _VALIDATE && index.Type() != datastore.VIRTUAL {
		return
	}
	if this.pushDownPropMap == nil {
		this.pushDownPropMap = make(map[string]PushDownProperties, 1)
	}
	key := index.KeyspaceId() + "_" + alias + "_" + index.Name() + "_" + string(index.Type())
	if _, ok := this.pushDownPropMap[key]; !ok {
		this.pushDownPropMap[key] = property
	}
}

func getAndTerms(pred *expression.And) expression.Expressions {
	res := make(expression.Expressions, 0, 2)
	for _, e := range pred.Operands() {
		if _, ok := e.(*expression.Or); ok {
			continue
		} else if _, ok := e.(*expression.Not); ok {
			continue
		} else if and, ok := e.(*expression.And); ok {
			res = append(res, getAndTerms(and)...)
		} else {
			res = append(res, e)
		}
	}
	return res
}

func getFilterInfos(filters base.Filters) iaplan.FilterInfos {
	exprs := make(iaplan.FilterInfos, 0, len(filters))
	for _, f := range filters {
		exprs = append(exprs, iaplan.NewFilterInfo(f.FltrExpr().Copy(), f.IsUnnest(), f.IsDerived(), f.IsJoin()))
	}
	return exprs
}

func (this *builder) setUnnest() {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		this.queryInfo.SetUnnest(true)
	}
}

func (this *builder) processadviseJF(alias string) {
	if this.indexAdvisor {
		this.processKeyspaceDone(alias)
	}
}

func (this *builder) setKeyspaceFound() {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		this.queryInfo.SetKeyspaceFound()
	}
}

func (this *builder) matchPushdownProperty(key string, idxInfo *iaplan.IndexInfo) *iaplan.IndexInfo {
	if property, ok := this.pushDownPropMap[key]; ok {
		if property > _PUSHDOWN_EXACTSPANS {
			var propertyString string
			set := _PUSHDOWN_FULLGROUPAGGS
			for set > _PUSHDOWN_EXACTSPANS {
				if isPushDownProperty(property, set) {
					if len(propertyString) > 0 {
						propertyString += ", "
					}
					propertyString += pushdownMap[set]
				}
				set >>= 1
			}
			if len(propertyString) > 0 {
				idxInfo.SetPushdown(propertyString)
			}
		}
	}
	return idxInfo
}

func collectInnerUnnestMap(from algebra.FromTerm, q *iaplan.QueryInfo, primaryIdentifier *expression.Identifier, level int) int {
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return 0
	}

	level = collectInnerUnnestMap(joinTerm.Left(), q, primaryIdentifier, level)
	unnest, ok := joinTerm.(*algebra.Unnest)
	if ok && !unnest.Outer() {
		// to add the top level expression which should belong to the unnest filters
		if unnest.Expression().DependsOn(primaryIdentifier) {
			q.AddToUnnestMap(expression.NewStringer().Visit(unnest.Expression()), unnest.Expression(), level)
			level += 1
		}
		q.AddToUnnestMap(unnest.Alias(), unnest.Expression(), level)
		level += 1
	}
	return level
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

			infos = append(infos, extractInfo(idx, alias, keyspace, true, false))
		}
	}
	return infos
}

func doDNF(stmtExprs expression.Expressions) expression.Expressions {
	exprs := make(expression.Expressions, 0, len(stmtExprs))
	for _, e := range stmtExprs {
		dnf := NewDNF(e, true, true)
		e, err := dnf.Map(e)
		if err != nil {
			return nil
		}
		exprs = append(exprs, e)
	}
	return exprs
}
