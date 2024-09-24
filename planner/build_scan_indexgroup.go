//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"
	"sort"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

type indexGroupAggProperties struct {
	version        int
	supported      bool
	aggtype        datastore.AggregateType
	distinct       bool
	partialAllowed bool
	filter         bool
}

var _INDEX_AGG_PROPERTIES = map[string]*indexGroupAggProperties{
	"avg":             &indexGroupAggProperties{3, true, datastore.AGG_AVG, false, false, false},
	"count":           &indexGroupAggProperties{3, true, datastore.AGG_COUNT, false, true, false},
	"countn":          &indexGroupAggProperties{3, true, datastore.AGG_COUNTN, false, true, false},
	"max":             &indexGroupAggProperties{3, true, datastore.AGG_MAX, false, true, false},
	"min":             &indexGroupAggProperties{3, true, datastore.AGG_MIN, false, true, false},
	"sum":             &indexGroupAggProperties{3, true, datastore.AGG_SUM, false, true, false},
	"avg_distinct":    &indexGroupAggProperties{3, true, datastore.AGG_AVG, true, false, false},
	"count_distinct":  &indexGroupAggProperties{3, true, datastore.AGG_COUNT, true, false, false},
	"countn_distinct": &indexGroupAggProperties{3, true, datastore.AGG_COUNTN, true, false, false},
	"sum_distinct":    &indexGroupAggProperties{3, true, datastore.AGG_SUM, true, false, false},
}

func checkAndAdd(ids []int, id int) []int {
	for _, v := range ids {
		if v == id {
			return ids
		}
	}

	return append(ids, id)
}

func aggToIndexAgg(agg algebra.Aggregate) *indexGroupAggProperties {
	name := agg.Name()
	if agg.Distinct() {
		name = name + "_distinct"
	}
	rv, _ := _INDEX_AGG_PROPERTIES[name]
	return rv
}

func indexPartialAggregateCount2SumRewrite(agg algebra.Aggregate, c *expression.Cover) algebra.Aggregate {
	switch agg.(type) {
	case *algebra.Count, *algebra.Countn:
		if agg.Operands()[0] == nil {
			return algebra.NewSum(expression.Expressions{c}, uint32(0), nil, nil).(*algebra.Sum)
		}
		return algebra.NewSum(agg.Operands(), uint32(0), nil, nil).(*algebra.Sum)
	}
	return agg
}

// rewrite Partial AVG Aggregate as  SUM/COUNTN by matching exact arguments
func indexPartialAggregateAvg2DivisionRewrite(oagg algebra.Aggregate,
	aggs algebra.Aggregates) (expr expression.Expression, err error) {

	var sagg, cagg algebra.Aggregate
	var nagg algebra.Aggregate
	var c *expression.Cover
	var ok bool

	switch oagg.(type) {
	case *algebra.Avg:
		for _, agg := range aggs {
			if sagg == nil {
				if nagg, ok = agg.(*algebra.Sum); ok {
					if c, ok = nagg.Operands()[0].(*expression.Cover); ok {
						nagg, ok = c.Covered().(*algebra.Sum)
						if ok && algebra.EqualAggregateModifiers(oagg, nagg) {
							sagg = agg
						}
					}
				}
			}

			if cagg == nil {
				if nagg, ok = agg.(*algebra.Sum); ok {
					if c, ok = nagg.Operands()[0].(*expression.Cover); ok {
						nagg, ok = c.Covered().(*algebra.Countn)
						if ok && algebra.EqualAggregateModifiers(oagg, nagg) {
							cagg = agg
						}
					}
				}
			}
			if sagg != nil && cagg != nil {
				return expression.NewDiv(sagg, cagg), nil
			}
		}
	}
	return oagg, fmt.Errorf(" indexPartailAggregateAvg2DivisionRewrite error ")
}

// rewrite Full AVG Aggregate as SUM/COUNTN by matching exact agruments

func indexFullAggregateAvg2DivisionRewrite(oagg algebra.Aggregate,
	covers []*expression.Cover) (expr expression.Expression, err error) {

	var sagg, cagg *expression.Cover
	var nagg algebra.Aggregate
	var ok bool

	switch oagg.(type) {
	case *algebra.Avg:
		for _, c := range covers {
			if sagg == nil {
				nagg, ok = c.Covered().(*algebra.Sum)
				if ok && algebra.EqualAggregateModifiers(oagg, nagg) {
					sagg = c
				}
			}

			if cagg == nil {
				nagg, ok = c.Covered().(*algebra.Countn)
				if ok && algebra.EqualAggregateModifiers(oagg, nagg) {
					cagg = c
				}
			}
			if sagg != nil && cagg != nil {
				return expression.NewDiv(sagg, cagg), nil
			}
		}
	}
	return oagg, fmt.Errorf(" indexFullAggregateAvg2DivisionRewrite error ")
}

// Check if aggregate is supported and generate new index aggregates for AVG i.e SUM(), COUNTN()
func (this *builder) indexAggregateRewrite() algebra.Aggregates {
	naggs := make(map[string]algebra.Aggregate)
	stringer := expression.NewStringer()

	for _, agg := range this.aggs {
		aggOp := agg.Operands()[0]
		aggProprties := aggToIndexAgg(agg)
		if aggProprties == nil || !aggProprties.supported ||
			(aggOp == nil && aggProprties.aggtype != datastore.AGG_COUNT) {
			this.resetIndexGroupAggs()
			return nil
		}

		switch agg.(type) {
		case *algebra.Avg:
			naggSum := algebra.NewSum(agg.Operands().Copy(), agg.Flags(), expression.Copy(agg.Filter()), nil)
			naggs[stringer.Visit(naggSum)] = naggSum

			naggCountn := algebra.NewCountn(agg.Operands().Copy(), agg.Flags(), expression.Copy(agg.Filter()), nil)
			naggs[stringer.Visit(naggCountn)] = naggCountn

		default:
			naggs[stringer.Visit(agg)] = agg
		}
	}

	return sortAggregatesMap(naggs)
}

func (this *builder) buildIndexGroupAggs(entry *indexEntry, indexKeys expression.Expressions,
	unnest bool, indexProjection *plan.IndexProjection) (
	*plan.IndexGroupAggregates, *plan.IndexProjection) {

	_, ok := entry.spans.(*TermSpans)
	if this.group == nil || !ok || entry.index.Type() == datastore.SEQ_SCAN ||
		!useIndex3API(entry.index, this.context.IndexApiVersion()) {

		this.resetIndexGroupAggs()
		return nil, indexProjection
	}

	size := len(this.group.By()) + len(this.aggs)
	idxProj := plan.NewIndexProjection(size, false)
	dependsOnIndexKeys := make([]int, 0, len(indexKeys))
	idNum := len(indexKeys)
	nKeys := len(entry.keys)
	if entry.index.IsPrimary() {
		nKeys = 1
	}

	var indexGroup plan.IndexGroupKeys
	var indexAggs plan.IndexAggregates

	if len(this.group.By()) > 0 {
		indexGroup, dependsOnIndexKeys, idNum = this.buildIndexGroup(indexKeys, idxProj,
			dependsOnIndexKeys, idNum, nKeys)
		if len(indexGroup) == 0 {
			return nil, indexProjection
		}
	}

	if len(this.aggs) > 0 {
		indexAggs, dependsOnIndexKeys, idNum = this.buildIndexAggregates(indexKeys, idxProj,
			dependsOnIndexKeys, idNum, nKeys)
		if len(indexAggs) == 0 {
			return nil, indexProjection
		}
	}

	// Indexer gives partial aggregates
	partial := !entry.IsPushDownProperty(_PUSHDOWN_FULLGROUPAGGS)

	// First index key is ALL array key and Not Unnest Scan we need one per META().id
	distinctDocid := false
	if entry.arrayKey != nil && entry.arrayKeyPos == 0 && !unnest {
		distinctDocid = !entry.arrayKey.NoAll()
	}

	sort.Ints(dependsOnIndexKeys)
	sort.Ints(idxProj.EntryKeys)
	return plan.NewIndexGroupAggregates("", indexGroup, indexAggs, dependsOnIndexKeys, partial, distinctDocid), idxProj
}

func (this *builder) buildIndexGroup(indexKeys expression.Expressions, indexProjection *plan.IndexProjection,
	dependsOnIndexKeys []int, idNum, nKeys int) (plan.IndexGroupKeys, []int, int) {

	groupKeys := this.group.By()
	indexGroup := make(plan.IndexGroupKeys, 0, len(groupKeys))
	indexPosGroup := make(plan.IndexGroupKeys, len(indexKeys), len(indexKeys))
	indexExprGroup := make(plan.IndexGroupKeys, 0, len(groupKeys))

nextgroup:
	for _, groupKey := range groupKeys {

		for _, idxGroup := range indexExprGroup {
			if groupKey.EquivalentTo(idxGroup.Expr) {
				continue nextgroup
			}
		}

		for indexKeyPos, indexKey := range indexKeys {
			if groupKey.EquivalentTo(indexKey) {
				if indexPosGroup[indexKeyPos] == nil {
					dependsOnIndexKeys = checkAndAdd(dependsOnIndexKeys, indexKeyPos)
					indexProjection.EntryKeys = checkAndAdd(indexProjection.EntryKeys, indexKeyPos)
					exprId := indexKeyPos
					if indexKeyPos == nKeys {
						exprId = -1
					}
					indexPosGroup[indexKeyPos] = plan.NewIndexGroupKey(indexKeyPos, exprId,
						groupKey.Copy(), []int{indexKeyPos})
				}
				continue nextgroup
			}
		}

		var idxExprGroup *plan.IndexGroupKey
		for indexKeyPos, indexKey := range indexKeys {
			if groupKey.DependsOn(indexKey) {
				if idxExprGroup == nil {
					indexProjection.EntryKeys = checkAndAdd(indexProjection.EntryKeys, idNum)
					idxExprGroup = plan.NewIndexGroupKey(idNum, -1, groupKey.Copy(), []int{indexKeyPos})
					idNum++
				} else {
					idxExprGroup.Depends = append(idxExprGroup.Depends, indexKeyPos)
				}

				dependsOnIndexKeys = checkAndAdd(dependsOnIndexKeys, indexKeyPos)
			}
		}

		if idxExprGroup != nil {
			indexExprGroup = append(indexExprGroup, idxExprGroup)
		} else {
			this.resetIndexGroupAggs()
			return nil, dependsOnIndexKeys, idNum
		}
	}

	for _, idxGroup := range indexPosGroup {
		if idxGroup != nil {
			indexGroup = append(indexGroup, idxGroup)
		}
	}

	indexGroup = append(indexGroup, indexExprGroup...)
	return indexGroup, dependsOnIndexKeys, idNum
}

func (this *builder) buildIndexAggregates(indexKeys expression.Expressions, indexProjection *plan.IndexProjection,
	dependsOnIndexKeys []int, idNum, nKeys int) (plan.IndexAggregates, []int, int) {

	naggs := this.indexAggregateRewrite()
	indexAggs := make(plan.IndexAggregates, 0, len(naggs))

nextagg:
	for _, agg := range naggs {
		aggExpr := agg.Operands()[0]
		aggProprties := aggToIndexAgg(agg)
		if aggProprties == nil || (aggExpr == nil && aggProprties.aggtype != datastore.AGG_COUNT) {
			this.resetIndexGroupAggs()
			return nil, dependsOnIndexKeys, idNum
		} else if aggExpr == nil {
			indexAggs = append(indexAggs, plan.NewIndexAggregate(aggProprties.aggtype,
				idNum, -1, expression.ONE_EXPR, aggProprties.distinct, nil))
			indexProjection.EntryKeys = checkAndAdd(indexProjection.EntryKeys, idNum)
			idNum++
			continue
		} else if aggExpr.Value() != nil {
			indexAggs = append(indexAggs, plan.NewIndexAggregate(aggProprties.aggtype,
				idNum, -1, aggExpr, aggProprties.distinct, nil))
			indexProjection.EntryKeys = checkAndAdd(indexProjection.EntryKeys, idNum)
			idNum++
			continue
		}

		for indexKeyPos, indexKey := range indexKeys {
			if aggExpr.EquivalentTo(indexKey) {
				dependsOnIndexKeys = checkAndAdd(dependsOnIndexKeys, indexKeyPos)
				indexProjection.EntryKeys = checkAndAdd(indexProjection.EntryKeys, idNum)
				exprId := indexKeyPos
				if indexKeyPos == nKeys {
					exprId = -1
				}
				indexAggs = append(indexAggs, plan.NewIndexAggregate(aggProprties.aggtype, idNum,
					exprId, aggExpr.Copy(), aggProprties.distinct, []int{indexKeyPos}))
				idNum++
				continue nextagg
			}
		}

		var idxAgg *plan.IndexAggregate
		for indexKeyPos, indexKey := range indexKeys {
			if aggExpr.DependsOn(indexKey) {
				if idxAgg == nil {
					indexProjection.EntryKeys = checkAndAdd(indexProjection.EntryKeys, idNum)
					idxAgg = plan.NewIndexAggregate(aggProprties.aggtype, idNum,
						-1, aggExpr.Copy(), aggProprties.distinct, []int{indexKeyPos})
					idNum++
				} else {
					idxAgg.Depends = append(idxAgg.Depends, indexKeyPos)
				}
				dependsOnIndexKeys = checkAndAdd(dependsOnIndexKeys, indexKeyPos)
			}
		}

		if idxAgg != nil {
			indexAggs = append(indexAggs, idxAgg)
		} else {
			this.resetIndexGroupAggs()
			return nil, dependsOnIndexKeys, idNum
		}
	}

	this.aggs = naggs
	return indexAggs, dependsOnIndexKeys, idNum
}
