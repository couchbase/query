//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package planner

import (
	"sort"

	"github.com/couchbase/query-ee/dictionary"
	advisor "github.com/couchbase/query-ee/indexadvisor"
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

const (
	_MAXUNNEST = 6
)

var pushdownMap = map[PushDownProperties]string{
	_PUSHDOWN_LIMIT:         "LIMIT pushdown",
	_PUSHDOWN_OFFSET:        "OFFSET pushdown",
	_PUSHDOWN_ORDER:         "ORDER pushdown",
	_PUSHDOWN_PARTIAL_ORDER: "ORDER pushdown (partial)",
	_PUSHDOWN_GROUPAGGS:     "GROUPBY & AGGREGATES pushdown",
	_PUSHDOWN_FULLGROUPAGGS: "FULL GROUPBY & AGGREGATES pushdown",
}

func (this *builder) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	this.setAdvisePhase(_RECOMMEND)
	//Temporarily turn off CBO for rule-based advisor
	considerCBO := false
	if this.useCBO {
		considerCBO = true
		this.useCBO = false
	}

	this.maxParallelism = 1
	this.queryInfos = make(map[expression.HasExpressions]*advisor.QueryInfo, 1)
	stmt.Statement().Accept(this)

	this.advise(considerCBO, this.queryInfos, stmt.Statement(), stmt.Context())
	indexAdvice := plan.NewIndexAdvice(generateIdxAdvice(this.queryInfos, this.validatedIdxes,
		this.validatedCoverIdxes, this.context.QueryContext()))

	var subqueryIndexAdvice []plan.StmtAdvice
	if len(this.subqueryInfos) > 0 {
		subqueryIndexAdvice = make([]plan.StmtAdvice, 0, len(this.subqueryInfos))
		for q, queryInfos := range this.subqueryInfos {
			this.advise(considerCBO, queryInfos, q, stmt.Context())
			indexAdvice := plan.NewIndexAdvice(generateIdxAdvice(queryInfos, this.validatedIdxes,
				this.validatedCoverIdxes, this.context.QueryContext()))
			subqueryIndexAdvice = append(subqueryIndexAdvice, plan.StmtAdvice{Op: indexAdvice, Subquery: q.String()})
		}
	}

	op := plan.NewAdvise(indexAdvice, subqueryIndexAdvice, stmt.Query())
	return plan.NewQueryPlan(op), nil
}

func (this *builder) advise(considerCBO bool, queryInfos map[expression.HasExpressions]*advisor.QueryInfo,
	stmt algebra.Statement, context interface{}) {
	if considerCBO {
		for _, queryInfo := range queryInfos {
			keyspaceInfos := queryInfo.GetKeyspaceInfos()
			for _, info := range keyspaceInfos {
				if dictionary.HasKeyspaceInfo(info.GetName()) {
					info.SetStatsOn()
				}
				//docCount, err := info.GetKeyspace().Count(datastore.NULL_QUERY_CONTEXT)
				//if err == nil && docCount > 0.0 {
				//	info.SetStatsOn()
				//}
			}
		}
	}

	coverIdxMap := advisor.AdviseIdxs(queryInfos,
		extractExistAndDeferredIdxes(queryInfos, this.context.IndexApiVersion()), doDNF(stmt.Expressions()),
		context, this.context)

	this.setAdvisePhase(_VALIDATE)
	//There are covering indexes to be validated:
	if len(coverIdxMap) > 0 {
		var vectorIdxCandidates []datastore.Index
		idxCandidates := make([]datastore.Index, 0, len(coverIdxMap))
		for _, info := range coverIdxMap {
			idx := info.VirtualIndex()
			if idx != nil {
				if idx6, ok := idx.(datastore.Index6); ok && idx6.IsVector() {
					if vectorIdxCandidates == nil {
						vectorIdxCandidates = make([]datastore.Index, 0, len(coverIdxMap))
					}
					vectorIdxCandidates = append(vectorIdxCandidates, idx)
				} else {
					idxCandidates = append(idxCandidates, idx)
				}
				if considerCBO && !info.IsCostBased() {
					considerCBO = false
				}
			}
		}

		if len(vectorIdxCandidates) > 0 {
			this.validateVectorIndexes(vectorIdxCandidates, considerCBO, stmt)
		} else if len(idxCandidates) > 0 {
			this.validateIndexes(idxCandidates, considerCBO, stmt)
		}

		if len(this.validatedIdxes) > 0 {
			this.matchIdxInfos(coverIdxMap)
		}
	}
}

func (this *builder) validateVectorIndexes(idxCandidates []datastore.Index, considerCBO bool, stmt algebra.Statement) {
	if len(idxCandidates) > 1 {
		// for advise output stability
		sort.Slice(idxCandidates, func(i, j int) bool {
			i6, oki := idxCandidates[i].(datastore.Index6)
			j6, okj := idxCandidates[j].(datastore.Index6)
			if oki && okj && !i6.IsBhive() && j6.IsBhive() {
				return true
			}
			return false
		})
	}
	// consider vector index one at a time
	for i := range idxCandidates {
		this.validateIndexes(idxCandidates[i:i+1], considerCBO, stmt)
	}
}

func (this *builder) validateIndexes(idxCandidates []datastore.Index, considerCBO bool, stmt algebra.Statement) {
	prevIdxCandidates := this.idxCandidates
	prevValidatedIdxes := this.validatedIdxes
	prevUseCBO := this.useCBO
	defer func() {
		this.idxCandidates = prevIdxCandidates
		this.useCBO = prevUseCBO
		if len(this.validatedIdxes) == 0 {
			this.validatedIdxes = prevValidatedIdxes
		} else {
			this.validatedIdxes = append(prevValidatedIdxes, this.validatedIdxes...)
		}
	}()

	this.idxCandidates = idxCandidates
	this.validatedIdxes = nil
	if considerCBO {
		this.useCBO = true
	}
	stmt.Accept(this)
}

func generateIdxAdvice(queryInfos map[expression.HasExpressions]*advisor.QueryInfo, nonCoverIdxes, coverIdxes iaplan.IndexInfos,
	queryContext string) (iaplan.IndexInfos, iaplan.IndexInfos, iaplan.IndexInfos) {
	cntKeyspaceNotFound := 0
	curIndexes := make(iaplan.IndexInfos, 0, 1) //initialize to distinguish between nil and empty for error message
	recIndexes := make(iaplan.IndexInfos, 0, 1)
	if len(nonCoverIdxes) > 0 {
		nonCoverIdxes.SetQueryContext(queryContext)
	}
	recIndexes = append(recIndexes, nonCoverIdxes...)
	nvector := 0
	for _, info := range nonCoverIdxes {
		if info.HasVectorInfo() {
			nvector++
		}
	}

	for _, v := range queryInfos {
		if !v.IsKeyspaceFound() {
			cntKeyspaceNotFound += 1
			continue
		}
		checkVector := v.HasVectorIdx()
		hasVector := false
		cIndexes := v.GetCurIndexes()
		if len(cIndexes) > 0 {
			curIdxMap := make(map[string]iaplan.IndexInfos, len(cIndexes))
			for _, cIdx := range cIndexes {
				if checkVector {
					if cIdx.HasVectorInfo() {
						hasVector = true
					} else {
						// do not show "OPTIMAL COVERING INDEX"
						cIdx.UnsetIdxStatusCovering()
					}
				}
				idxName := cIdx.GetIndexName()
				if infos, ok := curIdxMap[idxName]; ok {
					found := false
					for _, info := range infos {
						if info.EquivalentTo(cIdx, false) {
							info.AddAlias(cIdx.GetAlias())
							found = true
							break
						}
					}
					if !found {
						infos = append(infos, cIdx)
						curIdxMap[idxName] = infos
						curIndexes = append(curIndexes, cIdx)
					}
				} else {
					curIdxMap[idxName] = iaplan.IndexInfos{cIdx}
					curIndexes = append(curIndexes, cIdx)
				}
			}
		}

		// if a query references a vector term, and vector index is already suggested above, or
		// vector index already exists, do not advise additional indexes without vector index key
		if len(v.GetUncoverIndexes()) > 0 && nvector == 0 && !hasVector {
			v.GetUncoverIndexes().SetQueryContext(queryContext)
			for _, uci := range v.GetUncoverIndexes() {
				if uci.IsFound(nonCoverIdxes, false) {
					continue
				}
				if checkVector && !uci.HasVectorInfo() {
					continue
				}
				duplicate := false
				for _, info := range coverIdxes {
					if uci.String() == info.String() {
						duplicate = true
						break
					}
				}
				if !duplicate {
					recIndexes = append(recIndexes, uci)
				}
			}
		}
	}

	if cntKeyspaceNotFound == len(queryInfos) && len(curIndexes) == 0 {
		curIndexes = nil
	}

	if len(coverIdxes) > 0 {
		coverIdxes.SetQueryContext(queryContext)
	}
	return curIndexes, recIndexes, coverIdxes
}

func (this *builder) matchIdxInfos(m map[string]*iaplan.IndexInfo) {
	i := 0
	for _, info := range this.validatedIdxes {
		key := info.GetKeyspaceName() + "_" + info.GetIndexName() + "_virtual"
		if origInfo, ok := m[key]; ok {
			if !info.IsCostBased() {
				origInfo.SetCostBased(false)
			}
			origInfo.SetCovering(info.Covering())
			ix := this.matchPushdownProperty(key, origInfo)
			if info.HasEarlyOrder() {
				ix.SetPushdown("EARLY ORDER")
			}
			if origInfo.Covering() {
				this.validatedCoverIdxes = append(this.validatedCoverIdxes, ix)
			} else {
				this.validatedIdxes[i] = ix
				i++
			}
			delete(m, key)
		}
	}
	this.validatedIdxes = this.validatedIdxes[:i]
	return
}

type collectQueryInfo struct {
	keyspaceInfos       advisor.KeyspaceInfos
	queryInfo           *advisor.QueryInfo
	queryInfos          map[expression.HasExpressions]*advisor.QueryInfo
	subqueryInfos       map[*algebra.Select]map[expression.HasExpressions]*advisor.QueryInfo
	indexCollector      *scanIdxCol
	idxCandidates       []datastore.Index
	validatedIdxes      iaplan.IndexInfos
	validatedCoverIdxes iaplan.IndexInfos
	pushDownPropMap     map[string]PushDownProperties // key->"keyspace_alias_idxName_typeofIdx" e.g. "default:b.s.k_d_idx1_virtual"
	advisePhase         int
}

type saveQueryInfo struct {
	keyspaceInfos advisor.KeyspaceInfos
	queryInfo     *advisor.QueryInfo
	queryInfos    map[expression.HasExpressions]*advisor.QueryInfo
}

func (this *builder) setAdvisePhase(op int) {
	this.indexAdvisor = true
	this.advisePhase = op
}

func (this *builder) saveQueryInfo() *saveQueryInfo {
	return &saveQueryInfo{
		keyspaceInfos: this.keyspaceInfos,
		queryInfo:     this.queryInfo,
		queryInfos:    this.queryInfos,
	}
}

func (this *builder) restoreQueryInfo(saveQInfo *saveQueryInfo) {
	if saveQInfo != nil {
		this.queryInfos = saveQInfo.queryInfos
		this.keyspaceInfos = saveQInfo.keyspaceInfos
		this.queryInfo = saveQInfo.queryInfo
	} else {
		this.queryInfos = nil
		this.keyspaceInfos = nil
		this.queryInfo = nil
	}
}

// Initializes a new map for subqueryInfos only if not already created
func (this *builder) makeSubqueryInfos(l int) {
	if this.subqueryInfos == nil {
		this.subqueryInfos = make(map[*algebra.Select]map[expression.HasExpressions]*advisor.QueryInfo, l)
	}
}

func (this *builder) startSubqIndexAdvisor() {
	this.queryInfos = make(map[expression.HasExpressions]*advisor.QueryInfo, 1)
	this.keyspaceInfos = nil
	this.queryInfo = nil
}

func (this *builder) endSubqIndexAdvisor(s *algebra.Select) {
	this.subqueryInfos[s] = this.queryInfos
	this.queryInfos = nil
}

func (this *builder) initialIndexAdvisor(stmt algebra.Statement) {
	if this.indexAdvisor {
		if this.advisePhase == _RECOMMEND {
			if stmt != nil {
				this.queryInfo = advisor.NewQueryInfo(stmt.Type())
				this.keyspaceInfos = advisor.NewKeyspaceInfos()
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
			this.validatedIdxes = make(iaplan.IndexInfos, 0, 1)
			this.validatedCoverIdxes = make(iaplan.IndexInfos, 0, 1)
		}
	}
}

func (this *builder) extractKeyspacePredicates(where, on expression.Expression) {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		if where != nil {
			this.queryInfo.SetWhere(where)
		}
		if on != nil {
			this.queryInfo.SetOn(on)
		}
		this.queryInfo.SetKeyspaceNames(this.keyspaceNames)
	}
}

func (this *builder) extractIndexJoin(index datastore.Index, keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, cover bool,
	cost, cardinality float64) {

	if this.indexAdvisor {
		if index != nil {
			info := extractInfo(index, node.Alias(), keyspace, false, this.advisePhase == _VALIDATE)
			if info != nil {
				if cover { //covering index
					info.SetIdxStatusCovering()
				}
				if this.advisePhase == _VALIDATE {
					info.SetCostBased(cost > 0 && cardinality > 0)
					this.validatedIdxes = append(this.validatedIdxes, info)
					return
				}
				this.queryInfo.SetCurIndex(info)
			}
		}
		if this.advisePhase == _RECOMMEND {
			if !cover {
				this.keyspaceInfos.SetUncovered()
			}
			this.queryInfo.AppendKeyspaceInfos(this.keyspaceInfos)
			this.keyspaceInfos = advisor.NewKeyspaceInfos()
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
				if this.indexCollector.property != 0 {
					this.validatedIdxes = append(this.validatedIdxes, this.indexCollector.indexInfos...)
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
			this.keyspaceInfos = advisor.NewKeyspaceInfos()
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

func (this *builder) extractPagination(order *algebra.Order, offset, limit expression.Expression) {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		if order != nil {
			this.queryInfo.SetOrder(order.Terms())
		}
		if limit != nil {
			this.queryInfo.SetLimit(limit)
		}
		if offset != nil {
			this.queryInfo.SetOffset(offset)
		}
	}
}

func (this *builder) extractLetGroupProjOrder(let expression.Bindings, group *algebra.Group,
	projection *algebra.Projection, order *algebra.Order, aggs algebra.Aggregates) {
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

func (this *builder) collectUnnests(node *algebra.KeyspaceTerm) {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND && this.queryInfo.ContainsUnnest() {
		unnests, unnestsIdentifiers := this.queryInfo.Unnests()
		if len(unnests) == 0 {
			unnests = collectInnerUnnests(this.from, unnests)
			unnestsIdentifiers = collectUnnestsIdentifiers(unnests)
			this.queryInfo.SetUnnests(unnests, unnestsIdentifiers)
		}
		unnestMap := make(map[string]*unnestAdvisorEntry, len(unnests))
		for i, u := range unnests {
			if len(unnestsIdentifiers[i]) > 0 && u.Expression().Indexable() {
				unnestMap[u.Alias()] = &unnestAdvisorEntry{unnest: u, dependent: unnestsIdentifiers[i]}
			}
		}
		aliases := make(map[string]bool, 1)
		aliases[node.Alias()] = true
		collectAdvisorUnnests(aliases, unnestMap)

		for s, eu := range collectAdvisorUnnestsLevel(unnestMap, _MAXUNNEST) {
			this.queryInfo.AddToUnnestMap(node.Alias(), s, eu.unnest.Expression(), eu.level)
		}
	}
}

func (this *builder) collectPredicates(baseKeyspace *base.BaseKeyspace, keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm, pred expression.Expression, ansijoin, unnest bool) error {
	if !(this.indexAdvisor && this.advisePhase == _RECOMMEND) {
		return nil
	}
	//not advise index to system keyspace
	if algebra.IsSystemId(keyspace.NamespaceId()) {
		return nil
	}
	if baseKeyspace == nil {
		baseKeyspace = this.baseKeyspaces[node.Alias()]
	}

	if unnest {
		this.collectUnnests(node)
	}

	hasVector := len(baseKeyspace.VectorFilters()) > 0
	if pred == nil {
		//This is for collecting predicates from build_scan when predicate is not disjunction.
		if _, ok := baseKeyspace.DnfPred().(*expression.Or); !ok {
			p := advisor.NewKeyspaceInfo(keyspace, node,
				getFilterInfos(baseKeyspace.Filters(), this.context),
				getFilterInfos(baseKeyspace.JoinFilters(), this.context),
				baseKeyspace.Onclause(), baseKeyspace.DnfPred(), false, hasVector, nil)
			this.keyspaceInfos = append(this.keyspaceInfos, p)
		} else {
			pred = baseKeyspace.DnfPred()
		}
	}

	if pred != nil {
		//This is for collecting predicates from build_scan when predicates is disjunction.
		if or, ok := pred.(*expression.Or); ok {
			orTerms, _ := expression.FlattenOr(or)
			var predConjunc expression.Expressions
			if andTerm, ok := baseKeyspace.OrigPred().(*expression.And); ok {
				predConjunc = getAndTerms(andTerm)
			}
		outer:
			for _, op := range orTerms.Operands() {
				baseKeyspacesCopy := base.CopyBaseKeyspaces(this.baseKeyspaces)
				_, err := this.processPredicateBase(op, baseKeyspacesCopy, ansijoin)
				if err != nil {
					continue outer
				}

				bk, _ := baseKeyspacesCopy[node.Alias()]
				if !ansijoin {
					err = addUnnestPreds(baseKeyspacesCopy, bk)
					if err != nil {
						continue outer
					}
				}
				p := advisor.NewKeyspaceInfo(keyspace, node,
					getFilterInfos(bk.Filters(), this.context),
					getFilterInfos(bk.JoinFilters(), this.context),
					baseKeyspace.Onclause(), op, true, hasVector, predConjunc)
				this.keyspaceInfos = append(this.keyspaceInfos, p)
			}
		} else {
			//This is for collecting predicates for build_join_index.
			baseKeyspacesCopy := base.CopyBaseKeyspaces(this.baseKeyspaces)
			_, err := this.processPredicateBase(pred, baseKeyspacesCopy, false)
			if err != nil {
				return err
			}
			baseKeyspaceCopy, _ := baseKeyspacesCopy[node.Alias()]
			p := advisor.NewKeyspaceInfo(keyspace, node,
				getFilterInfos(baseKeyspaceCopy.Filters(), this.context),
				getFilterInfos(baseKeyspaceCopy.JoinFilters(), this.context),
				baseKeyspace.Onclause(), pred, false, hasVector, nil)
			this.keyspaceInfos = append(this.keyspaceInfos, p)
		}
	}
	return nil
}

func (this *builder) collectPushdownProperty(index datastore.Index, alias string, property PushDownProperties) {
	if this.advisePhase == _VALIDATE && index.Type() != datastore.VIRTUAL {
		return
	}
	if this.pushDownPropMap == nil {
		this.pushDownPropMap = make(map[string]PushDownProperties, 1)
	}

	key := datastore.IndexQualifiedKeyspacePath(index) + "_" + index.Name() + "_" + string(index.Type())
	if _, ok := this.pushDownPropMap[key]; !ok {
		this.pushDownPropMap[key] = property
	} else {
		this.pushDownPropMap[key] = this.pushDownPropMap[key] | property
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

func getFilterInfos(filters base.Filters, context *PrepareContext) base.Filters {
	exprs := make(base.Filters, 0, len(filters))
	for _, f := range filters {
		var fl *base.Filter
		if context != nil && (len(context.NamedArgs()) > 0 || len(context.PositionalArgs()) > 0) {
			namedArgs := context.NamedArgs()
			positionalArgs := context.PositionalArgs()
			fltrExpr, err := base.ReplaceParameters(f.FltrExpr(), namedArgs, positionalArgs)
			if err != nil {
				continue
			}
			origExpr, err := base.ReplaceParameters(f.OrigExpr(), namedArgs, positionalArgs)
			if err != nil {
				continue
			}
			fl = base.NewFilter(fltrExpr, origExpr, f.Keyspaces(), f.OrigKeyspaces(),
				f.IsOnclause(), f.IsJoin())
			fl.SetOptBits(f.OptBits())
		} else {
			fl = f.Copy()
		}
		exprs = append(exprs, fl)
	}
	return exprs
}

func (this *builder) setUnnest() {
	if this.indexAdvisor && this.advisePhase == _RECOMMEND {
		this.queryInfo.SetContainsUnnest(true)
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
			addgpa := true
			for set > _PUSHDOWN_EXACTSPANS {
				if isPushDownProperty(property, set) {
					if set == _PUSHDOWN_GROUPAGGS && !addgpa {
						set >>= 1
						continue
					} else if set == _PUSHDOWN_FULLGROUPAGGS {
						addgpa = false
					}
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

func (this *builder) getIdxCandidates() []datastore.Index {
	return this.idxCandidates
}

func (this *builder) advisorValidate() bool {
	return this.indexAdvisor && this.advisePhase == _VALIDATE
}

func (this *builder) advisorRecommend() bool {
	return this.indexAdvisor && this.advisePhase == _RECOMMEND
}

// collect identfiers for the unnests used

func collectUnnestsIdentifiers(unnests []*algebra.Unnest) (rv []map[string]expression.Expression) {
	rv = make([]map[string]expression.Expression, len(unnests))
	for pos, u := range unnests {
		rv[pos] = expression.GetIdentifiers(u.Expression())
	}
	return
}

// indexable unnests with dependency check and assign level
func collectAdvisorInnerUnnests(aliases map[string]bool, unnestMap map[string]*unnestAdvisorEntry) (rv []string) {
outer:
	for _, u := range unnestMap {
		if u.done {
			continue
		}
		level := 0
		for s, _ := range u.dependent {
			if _, ok := aliases[s]; !ok {
				continue outer
			}
			if u1, ok := unnestMap[s]; ok && level < u1.level {
				level = u1.level
			}
		}
		rv = append(rv, u.unnest.Alias())
		u.level = level + 1
		u.done = true
	}

	return rv
}

// collect unnests that can be indexed
func collectAdvisorUnnests(aliases map[string]bool, unnestMap map[string]*unnestAdvisorEntry) {
	rv := collectAdvisorInnerUnnests(aliases, unnestMap)
	for _, a := range rv {
		newAliases := make(map[string]bool, len(aliases)+1)
		for k, v := range aliases {
			newAliases[k] = v
		}
		newAliases[a] = true

		collectAdvisorUnnests(newAliases, unnestMap)
	}
	return
}

// Get highest nested unnests
func collectAdvisorUnnestsLevel(unnestMap map[string]*unnestAdvisorEntry, level int) (rv map[string]*unnestAdvisorEntry) {
	var fu *unnestAdvisorEntry
	for _, eu := range unnestMap {
		if eu.done && (fu == nil || (eu.level > fu.level && eu.level <= level)) {
			fu = eu
		}
	}

	if fu != nil {
		rv = make(map[string]*unnestAdvisorEntry, len(unnestMap))
		rv[fu.unnest.Alias()] = fu
		addUnnestDependent(unnestMap, rv, fu)
	}
	return
}

func addUnnestDependent(unnestMap, rv map[string]*unnestAdvisorEntry, ue *unnestAdvisorEntry) {
	if ue != nil {
		for s, _ := range ue.dependent {
			if u, ok := unnestMap[s]; ok {
				rv[s] = u
				addUnnestDependent(unnestMap, rv, u)
			}
		}
	}
	return
}

type unnestAdvisorEntry struct {
	unnest    *algebra.Unnest
	level     int
	done      bool
	dependent map[string]expression.Expression
}

func extractExistAndDeferredIdxes(queryInfos map[expression.HasExpressions]*advisor.QueryInfo,
	indexApiVersion int) map[string]iaplan.IndexInfos {
	if len(queryInfos) == 0 {
		return nil
	}

	infoMap := make(map[string]iaplan.IndexInfos, 1)
	for _, queryInfo := range queryInfos {
		for _, keyspaceInfo := range queryInfo.GetKeyspaceInfos() {
			if _, ok := infoMap[keyspaceInfo.GetName()]; !ok {
				//use nil value to mark one keyspace has been processed and no deferred indexes are found or errors occur.
				infoMap[keyspaceInfo.GetName()] = getExistAndDeferredIndexes(keyspaceInfo.GetKeyspace(), keyspaceInfo.GetAlias(),
					indexApiVersion)
			}
		}
	}
	return infoMap
}

func getExistAndDeferredIndexes(keyspace datastore.Keyspace, alias string, indexApiVersion int) iaplan.IndexInfos {
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
			if isValidIndex(idx, indexApiVersion) {
				if infos == nil {
					infos = make(iaplan.IndexInfos, 0, 1)
				}
				info := extractInfo(idx, alias, keyspace, false, false)
				if info != nil {
					infos = append(infos, info)
				}

			} else {
				state, _, er := idx.State()
				if er != nil || state != datastore.DEFERRED || idx.IsPrimary() {
					continue
				}

				//Not (useIndex2API(idx, indexApiVersion) || !indexHasDesc(idx))
				if !useIndex2API(idx, indexApiVersion) && indexHasDesc(idx) {
					continue
				}

				if infos == nil {
					infos = make(iaplan.IndexInfos, 0, 1)
				}
				info := extractInfo(idx, alias, keyspace, true, false)
				if info != nil {
					infos = append(infos, info)
				}
			}

		}
	}
	return infos
}

func doDNF(stmtExprs expression.Expressions) expression.Expressions {
	exprs := make(expression.Expressions, 0, len(stmtExprs))
	for _, e := range stmtExprs {
		dnf := base.NewDNF(e, true, true)
		e, err := dnf.Map(e)
		if err != nil {
			return nil
		}
		exprs = append(exprs, e)
	}
	return exprs
}
