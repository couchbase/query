//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"math"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

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

func (this *builder) buildSearchCoveringScan(searchSargables []*indexEntry, node *algebra.KeyspaceTerm,
	baseKeyspace *baseKeyspace, id expression.Expression) (plan.SecondaryScan, int, error) {
	if this.cover == nil || len(searchSargables) != 1 || !searchSargables[0].exactSpans {
		return nil, 0, nil
	}

	pred := baseKeyspace.dnfPred
	entry := searchSargables[0]
	alias := node.Alias()
	exprs := this.cover.Expressions()
	sfn := entry.sargKeys[0].(*search.Search)
	coveringExprs := make(expression.Expressions, 0, len(entry.keys)+3)
	coveringExprs = append(coveringExprs, entry.keys...)
	coveringExprs = append(coveringExprs, id)
	coveringExprs = append(coveringExprs, search.NewSearchScore(sfn.IndexMetaField()))
	coveringExprs = append(coveringExprs, search.NewSearchMeta(sfn.IndexMetaField()))

	for _, expr := range exprs {
		if !expression.IsCovered(expr, alias, coveringExprs) {
			return nil, 0, nil
		}
	}

	covers := make(expression.Covers, 0, len(coveringExprs))
	for _, expr := range coveringExprs {
		covers = append(covers, expression.NewCover(expr))
	}

	searchOrderEntry, searchOrders, _ := this.searchPagination(searchSargables, pred)

	this.resetProjection()
	if this.group != nil {
		this.resetPushDowns()
	}

	if this.order != nil && searchOrderEntry == nil {
		this.resetOrderOffsetLimit()
	}
	scan := this.CreateFTSSearch(entry.index, node, sfn, searchOrders, covers)
	if scan != nil {
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan, len(entry.sargKeys), nil
}

func (this *builder) CreateFTSSearch(index datastore.Index, node *algebra.KeyspaceTerm,
	sfn *search.Search, order []string, covers expression.Covers) plan.SecondaryScan {

	return plan.NewIndexFtsSearch(index, node,
		plan.NewFTSSearchInfo(expression.NewConstant(sfn.FieldName()), sfn.Query(), sfn.Options(),
			this.offset, this.limit, order, sfn.OutName()), covers)
}

func (this *builder) searchPagination(searchSargables []*indexEntry, pred expression.Expression) (
	orderEntry *indexEntry, orders []string, err error) {

	if len(searchSargables) == 0 {
		return nil, nil, nil
	} else if this.hasOffsetOrLimit() && (len(searchSargables) > 1 ||
		!expression.Equivalent(pred, searchSargables[0].sargKeys[0]) || !searchSargables[0].exactSpans) {
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
					if orderTerm.Descending() {
						collation = " DESC"
					}
					orders = append(orders, "score"+collation)
					if index.Pageable(orders, offset, limit) {
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
	} else {
		index := searchSargables[0].index.(datastore.FTSIndex)
		if !index.Pageable(nil, offset, limit) {
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

func (this *builder) sargableSearchIndexes(indexes []datastore.Index, pred expression.Expression,
	searchFns map[string]*search.Search) (searchSargables []*indexEntry, err error) {

	searchSargables = make([]*indexEntry, 0, len(searchFns))

	for _, s := range searchFns {
		siname := s.IndexName()
		keys := expression.Expressions{s.Copy()}
		if !SubsetOf(pred, keys[0]) {
			continue
		}

		var mappings interface{}
		var n int
		var size, esize int64
		var exact bool
		var entry *indexEntry

		qprams := s.Query().Value() == nil || (s.Options() != nil && s.Options().Value() == nil)

		for _, idx := range indexes {
			index, ok := idx.(datastore.FTSIndex)
			if !ok || (siname != "" && siname != index.Name()) {
				continue
			}

			n, size, exact, mappings, err = index.Sargable(s.FieldName(), s.Query(), s.Options(), mappings)
			if err != nil {
				return nil, err
			}

			if n > 0 {
				exact = exact && !qprams
				if entry == nil || size < esize {
					entry = &indexEntry{
						index, keys, keys, nil, 1, 1, nil, nil, nil, exact, _PUSHDOWN_NONE}
					esize = size
				}
			}
		}

		if entry != nil {
			searchSargables = append(searchSargables, entry)
		}
	}

	return
}
