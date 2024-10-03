//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"math"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// collect SEARCH() functions depends on the keyspace alias
func collectFTSSearch(alias string, ftsSearch map[string]*search.Search,
	exprs ...expression.Expression) (err error) {
	stringer := expression.NewStringer()
	for _, expr := range exprs {
		if expr == nil {
			continue
		} else if s, ok := expr.(*search.Search); ok && s.KeyspaceAlias() == alias {
			ftsSearch[stringer.Visit(s)] = s
		} else if _, ok := expr.(*algebra.Subquery); !ok {
			if err = collectFTSSearch(alias, ftsSearch, expr.Children()...); err != nil {
				return err
			}
		}
	}
	return
}

// Covering Search Accesspath for SEARCH() function

func (this *builder) buildSearchCovering(searchSargables []*indexEntry, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression) (plan.SecondaryScan, int, error) {

	// must be single index and Search Accesspath must be exact and no false positives.
	if this.cover == nil || len(searchSargables) != 1 || !searchSargables[0].exactSpans {
		return nil, 0, nil
	}

	pred := baseKeyspace.DnfPred()
	entry := searchSargables[0]
	alias := node.Alias()
	exprs := this.getExprsToCover()
	sfn := entry.sargKeys[0].(*search.Search)
	keys := make(expression.Expressions, 0, len(entry.keys)+3)
	keys = append(keys, entry.keys...)
	keys = append(keys, id, search.NewSearchScore(sfn.IndexMetaField()),
		search.NewSearchMeta(sfn.IndexMetaField()))

	coveringExprs, filterCovers, err := indexCoverExpressions(entry, keys, pred, pred, alias, this.context)
	if err != nil {
		return nil, 0, err
	}

	for _, expr := range exprs {
		if !expression.IsCovered(expr, alias, coveringExprs, false) {
			return nil, 0, nil
		}
	}

	covers := make(expression.Covers, 0, len(keys))
	for _, expr := range keys {
		covers = append(covers, expression.NewCover(expr))
	}

	if this.group != nil {
		this.resetPushDowns()
	}

	searchOrderEntry, searchOrders, _ := this.searchPagination(searchSargables, pred, alias)
	this.resetProjection()

	if this.order != nil && searchOrderEntry == nil {
		this.resetOrderOffsetLimit()
	}
	hasDeltaKeySpace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	scan := this.CreateFTSSearch(entry.index, node, sfn, searchOrders, covers, filterCovers, hasDeltaKeySpace)
	if scan != nil {
		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan, len(entry.sargKeys), nil
}

// Covering Search Accesspath for Flex Index

func (this *builder) buildFlexSearchCovering(flex map[datastore.Index]*indexEntry, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression) (plan.SecondaryScan, int, error) {

	// no JOINS or no UNNEST
	if this.cover == nil || len(flex) == 0 || len(this.baseKeyspaces) > 1 {
		return nil, 0, nil
	}

	pred := baseKeyspace.DnfPred()
	alias := node.Alias()
	coveringExprs := expression.Expressions{pred, id}

	for _, expr := range this.getExprsToCover() {
		if !expression.IsCovered(expr, alias, coveringExprs, false) {
			return nil, 0, nil
		}
	}

	// check sargable, exact and no false positive index
	var bentry *indexEntry
	for _, entry := range flex {
		if entry.exactSpans {
			if bentry == nil || entry.PushDownProperty() > bentry.PushDownProperty() {
				bentry = entry
			}
		}
	}

	if bentry == nil {
		return nil, 0, nil
	}

	// reset group pushdowns
	if this.group != nil {
		this.resetPushDowns()
	}

	// build Covering Search Access path
	covers := make(expression.Covers, 0, len(coveringExprs))
	for _, expr := range coveringExprs {
		covers = append(covers, expression.NewCover(expr))
	}

	sfn := bentry.sargKeys[0].(*search.Search)
	searchOrders := this.checkFlexSearchResetPaginations(bentry)
	this.resetProjection()

	hasDeltaKeySpace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	scan := this.CreateFTSSearch(bentry.index, node, sfn, searchOrders, covers, nil, hasDeltaKeySpace)
	if scan != nil {
		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan, len(bentry.sargKeys), nil
}

// Create FTS search Access Path

func (this *builder) CreateFTSSearch(index datastore.Index, node *algebra.KeyspaceTerm,
	sfn *search.Search, order []string, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value, hasDeltaKeySpace bool) plan.SecondaryScan {

	return plan.NewIndexFtsSearch(index, node,
		plan.NewFTSSearchInfo(expression.NewConstant(sfn.FieldName()), sfn.Query(), sfn.Options(),
			this.offset, this.limit, order, sfn.OutName()), covers, filterCovers,
		hasDeltaKeySpace, this.hasBuilderFlag(BUILDER_NL_INNER))
}

// SEARCH() function pagination
func (this *builder) searchPagination(searchSargables []*indexEntry, pred expression.Expression,
	alias string) (orderEntry *indexEntry, orders []string, err error) {

	if len(searchSargables) == 0 {
		return nil, nil, nil
	} else if this.hasOffsetOrLimit() && (len(searchSargables) > 1 ||
		!searchSargables[0].exactSpans || len(this.baseKeyspaces) != 1 ||
		!this.checkExactSpans(searchSargables[0], pred, nil, alias, nil, nil, false)) {
		this.resetOffsetLimit()
	}

	offset := getLimitOffset(this.offset, int64(0))
	limit := getLimitOffset(this.limit, math.MaxInt64)

	if this.order != nil && len(this.order.Terms()) == 1 {
		var hashProj map[string]expression.Expression

		if this.projection != nil {
			hashProj = make(map[string]expression.Expression, len(this.projection.Terms()))
			for _, term := range this.projection.Terms() {
				hashProj[term.Alias()] = term.Expression()
			}
		}

		orderTerm := this.order.Terms()[0]
		oExpr := orderTerm.Expression()
		if projexpr, projalias := hashProj[oExpr.Alias()]; projalias {
			oExpr = projexpr
		}

		if scorefn, ok := oExpr.(*search.SearchScore); ok {
			for _, entry := range searchSargables {
				sfn, ok := entry.sargKeys[0].(*search.Search)
				if ok && expression.Equivalent(sfn.IndexMetaField(),
					scorefn.IndexMetaField()) {
					index := entry.index.(datastore.FTSIndex)
					collation := " ASC"
					if orderTerm.Descending(nil, nil) {
						collation = " DESC"
						if orderTerm.NullsLast(nil, nil) {
							collation += " NULLS FIRST"
						}
					} else if orderTerm.NullsLast(nil, nil) {
						collation += " NULLS LAST"
					}

					orders = append(orders, "score"+collation)
					if index.Pageable(orders, offset, limit, sfn.Query(), sfn.Options()) {
						return entry, orders, nil
					}
					this.resetOrderOffsetLimit()
					return nil, nil, nil
				}
			}
		}
	}

	if this.order != nil {
		this.resetOffsetLimit()
	} else if this.hasOffsetOrLimit() {
		index := searchSargables[0].index.(datastore.FTSIndex)
		sfn, _ := searchSargables[0].sargKeys[0].(*search.Search)
		if !index.Pageable(nil, offset, limit, sfn.Query(), sfn.Options()) {
			this.resetOffsetLimit()
		}
	}

	return nil, nil, nil
}

func getLimitOffset(expr expression.Expression, defval int64) int64 {
	if expr != nil {
		val := expr.Value()
		if val != nil && val.Type() == value.NUMBER {
			return val.(value.NumberValue).Int64()
		}
	}

	return defval
}

//  FTSindex sargablity for SEARCH() function

func (this *builder) sargableSearchIndexes(indexes []datastore.Index, pred expression.Expression,
	searchFns map[string]*search.Search, formalizer *expression.Formalizer) (
	searchSargables []*indexEntry, err error) {

	if len(searchFns) == 0 {
		return
	}

	searchSargables = make([]*indexEntry, 0, len(searchFns))
	for _, s := range searchFns {
		siname := s.IndexName()
		keys := datastore.IndexKeys{&datastore.IndexKey{s.Copy(), datastore.IK_NONE}}
		if !base.SubsetOf(pred, keys[0].Expr) {
			continue
		}

		var mappings interface{}
		var n, en int
		var size, esize int64
		var exact bool
		var entry *indexEntry

		//qprams := s.Query().Value() == nil || (s.Options() != nil && s.Options().Value() == nil)

		for _, idx := range indexes {
			index, ok := idx.(datastore.FTSIndex)
			if !ok || (siname != "" && siname != index.Name()) {
				continue
			}

			var cond, origCond expression.Expression
			ok, cond, origCond, err = this.sargableSearchCondition(index, pred, formalizer)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}

			n, size, exact, mappings, err = index.Sargable(s.FieldName(), s.Query(), s.Options(), mappings)
			if err != nil {
				return nil, err
			}

			if n > 0 {
				//		exact = exact && !qprams
				if entry == nil || n > en || size < esize {
					entry = newIndexEntry(index, keys, len(keys), nil, 1, 1, 1,
						cond, origCond, nil, exact, []bool{true})
					esize = size
					en = n
				}
			}
		}

		if entry != nil {
			searchSargables = append(searchSargables, entry)
		}
	}

	return
}

func (this *builder) sargableSearchCondition(index datastore.Index, subset expression.Expression,
	formalizer *expression.Formalizer) (bool, expression.Expression, expression.Expression, error) {

	var err error
	var origCond expression.Expression

	cond := index.Condition()
	if cond == nil {
		return true, nil, nil, nil
	}
	cond = cond.Copy()

	formalizer.SetIndexScope()
	cond, err = formalizer.Map(cond)
	formalizer.ClearIndexScope()
	if err != nil {
		return false, nil, nil, err
	}

	origCond = cond.Copy()
	dnf := base.NewDNF(cond, true, true)
	cond, err = dnf.Map(cond)
	if err != nil {
		return false, nil, nil, err
	}

	return base.SubsetOf(subset, cond), cond, origCond, nil
}

// helper function

func (this *builder) flexIndexPushDownProperty(resp *datastore.FTSFlexResponse) (pushDownProperty PushDownProperties) {
	if (resp.RespFlags & datastore.FTS_FLEXINDEX_EXACT) != 0 {
		pushDownProperty |= _PUSHDOWN_EXACTSPANS
	}
	if (resp.RespFlags & datastore.FTS_FLEXINDEX_LIMIT) != 0 {
		pushDownProperty |= _PUSHDOWN_LIMIT
	}
	if (resp.RespFlags & datastore.FTS_FLEXINDEX_OFFSET) != 0 {
		pushDownProperty |= _PUSHDOWN_OFFSET
	}
	if (resp.RespFlags & datastore.FTS_FLEXINDEX_ORDER) != 0 {
		pushDownProperty |= _PUSHDOWN_ORDER
	}
	return
}

// build FTS Flex Request
func (this *builder) buildFTSFlexRequest(alias string, pred expression.Expression,
	ubs expression.Bindings) (flexRequest *datastore.FTSFlexRequest) {

	pageable := this.hasOrderOrOffsetOrLimit()
	var flexOrder []*datastore.SortTerm

	// order request
	if this.order != nil {
		flexOrder = make([]*datastore.SortTerm, 0, len(this.order.Terms()))
		var hashProj map[string]expression.Expression

		// handle order using projection alias
		if this.projection != nil {
			hashProj = make(map[string]expression.Expression, len(this.projection.Terms()))
			for _, term := range this.projection.Terms() {
				hashProj[term.Alias()] = term.Expression()
			}
		}

		for _, term := range this.order.Terms() {
			expr := term.Expression()
			if _, ok := expr.(*expression.Identifier); ok {
				expr, _ = hashProj[expr.Alias()]
			}

			var nullsPos uint32
			d := term.Descending(nil, nil)
			nl := term.NullsLast(nil, nil)
			if !d && nl {
				nullsPos = datastore.ORDER_NULLS_LAST
			} else if d && nl {
				nullsPos = datastore.ORDER_NULLS_FIRST
			}

			flexOrder = append(flexOrder, &datastore.SortTerm{
				Expr:       expr,
				Descending: d,
				NullsPos:   nullsPos,
			})
		}
	}

	flexRequest = &datastore.FTSFlexRequest{
		Keyspace:      alias,
		Pred:          pred.Copy(),
		Bindings:      ubs,
		Opaque:        make(map[string]interface{}, 4),
		CheckPageable: pageable,
	}

	if flexRequest.CheckPageable {
		flexRequest.Order = flexOrder
		if len(this.baseKeyspaces) == 1 {
			flexRequest.Offset = getLimitOffset(this.offset, int64(0))
			flexRequest.Limit = getLimitOffset(this.limit, math.MaxInt64)
		}
	}

	return
}

// FTS Flex index Saragbility
func (this *builder) sargableFlexSearchIndex(idx datastore.Index, flexRequest *datastore.FTSFlexRequest, join bool) (
	entry *indexEntry, err error) {

	index, ok := idx.(datastore.FTSIndex)
	if !ok || !util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_FLEXINDEX) {
		return
	}

	// call n1fty to get Search Access Path for given index and predicate
	var resp *datastore.FTSFlexResponse
	resp, err = index.SargableFlex("", flexRequest)
	if err != nil || resp == nil || resp.SearchQuery == "" || (join && len(resp.StaticSargKeys) == 0) {
		return
	}

	// handle FTS Flex response
	var sqe, soe expression.Expression

	keyspaceIdent := expression.NewIdentifier(flexRequest.Keyspace)
	keyspaceIdent.SetKeyspaceAlias(true)

	sqe, err = parser.Parse(resp.SearchQuery)
	if err == nil && resp.SearchOptions != "" {
		soe, err = parser.Parse(resp.SearchOptions)

	}
	if err != nil {
		return
	}

	keys := make(datastore.IndexKeys, 0, len(resp.StaticSargKeys)+len(resp.DynamicSargKeys)+1)
	keys = append(keys, &datastore.IndexKey{search.NewSearch(keyspaceIdent, sqe, soe), datastore.IK_NONE})
	for _, expr := range resp.StaticSargKeys {
		keys = append(keys, &datastore.IndexKey{expr, datastore.IK_NONE})
	}
	for _, expr := range resp.DynamicSargKeys {
		keys = append(keys, &datastore.IndexKey{expr, datastore.IK_NONE})
	}

	pushDownProperty := this.flexIndexPushDownProperty(resp)

	entry = newIndexEntry(index, keys, 1, nil,
		len(resp.StaticSargKeys),
		len(resp.StaticSargKeys)+len(resp.DynamicSargKeys),
		len(resp.StaticSargKeys)+len(resp.DynamicSargKeys),
		flexRequest.Cond, flexRequest.OrigCond, nil, isPushDownProperty(pushDownProperty, _PUSHDOWN_EXACTSPANS), nil)
	entry.setSearchOrders(resp.SearchOrders)
	entry.pushDownProperty = pushDownProperty
	entry.numIndexedKeys = resp.NumIndexedKeys
	return entry, nil
}

// minimize FTS Flex Indexes. best=true produces single index
func (this *builder) minimalFTSFlexIndexes(sargables map[datastore.Index]*indexEntry,
	best bool) map[datastore.Index]*indexEntry {

	for s, se := range sargables {
		for t, te := range sargables {
			if t == s {
				continue
			}

			if p := this.purgeFTSFlexIndex(se, te, best); p != nil {
				delete(sargables, p.index)
				if p == se {
					break
				}
			}
		}
	}

	return sargables
}

/*
 purge the FTS Flex indexes
     best=true
        keep highest sargable keys (static + dynamic)
        both equal highest static sargable keys
        both equal highest pushdwon property
        both equal lowest numIndexedKeys
        both equal random
    best=false
        both equal follow best=true
        one is subset of other keep superset
        if not keep both
*/

func (this *builder) purgeFTSFlexIndex(se, te *indexEntry, best bool) (ri *indexEntry) {
	var s, t *indexEntry
	if se.sumKeys > te.sumKeys || (se.sumKeys == te.sumKeys && se.minKeys >= te.minKeys) {
		if best {
			if te.sumKeys == se.sumKeys &&
				se.minKeys == te.minKeys &&
				(te.PushDownProperty() > se.PushDownProperty() ||
					(te.PushDownProperty() == se.PushDownProperty() && te.numIndexedKeys <= se.numIndexedKeys)) {
				return se
			}
			return te
		}
		s = se
		t = te
	} else {
		if best {
			return se
		}
		s = te
		t = se
	}

	stringer := expression.NewStringer()
	hashMap := make(map[string]expression.Expression, s.sumKeys)
	for _, k := range s.keys[1:] {
		hashMap[stringer.Visit(k)] = k
	}

	for _, k := range t.keys[1:] {
		if _, ok := hashMap[stringer.Visit(k)]; !ok {
			return nil
		}
	}

	if t.sumKeys != s.sumKeys ||
		t.minKeys < s.minKeys ||
		t.PushDownProperty() < s.PushDownProperty() ||
		t.numIndexedKeys > s.numIndexedKeys {
		return t
	}

	return s
}

// If SEARCH() function is already hadndled as part of FTS Flex index puerge it
func (this *builder) minimalSearchIndexes(flex map[datastore.Index]*indexEntry,
	searchSargables []*indexEntry) (rv []*indexEntry) {

	if len(flex) == 0 || len(searchSargables) == 0 {
		return searchSargables
	}

	rv = make([]*indexEntry, 0, len(searchSargables))

outer:
	for _, entry := range searchSargables {
		for _, fi := range flex {
			for _, exp := range fi.keys {
				if expression.Equivalent(exp, entry.sargKeys[0]) {
					continue outer
				}
			}
		}
		rv = append(rv, entry)
	}
	return rv
}

func (this *builder) checkFlexSearchResetPaginations(entry *indexEntry) (seacrhOrders []string) {
	// check order pushdown and reset
	if this.order != nil {
		if entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			seacrhOrders = entry.searchOrders
			this.maxParallelism = 1
		} else {
			this.resetOrderOffsetLimit()
			return
		}
	}

	// check offset push down and convert limit = limit + offset
	if this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
		this.limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffset()
	}

	// check limit and reset
	if this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT) {
		this.resetLimit()
	}
	return
}
