//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type IndexScan3 struct {
	base
	buildBitFilterBase
	probeBitFilterBase
	conn     *datastore.IndexConnection
	plan     *plan.IndexScan3
	children []Operator
	keys     map[string]bool
	pool     bool
	results  []*datastore.IndexEntry
}

func NewIndexScan3(plan *plan.IndexScan3, context *Context) *IndexScan3 {
	rv := &IndexScan3{plan: plan}
	newBase(&rv.base, context)
	rv.phase = INDEX_SCAN
	if p, ok := indexerPhase[plan.Index().Indexer().Name()]; ok {
		rv.phase = p.index
	}
	rv.output = rv
	return rv
}

func (this *IndexScan3) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan3(this)
}

func (this *IndexScan3) Copy() Operator {
	rv := &IndexScan3{}
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexScan3) PlanOp() plan.Operator {
	return this.plan
}

type scanDesc struct {
	scan               *IndexScan3
	context            *Context
	parent             value.Value
	indexVector        *datastore.IndexVector
	inlineFilter       string
	indexPartitionSets datastore.IndexPartitionSets
}

func scanFork(p interface{}) {
	d := p.(scanDesc)
	d.scan.scan(d.context, d.scan.conn, d.parent, d.indexVector, d.inlineFilter, d.indexPartitionSets)
}

func (this *IndexScan3) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhaseWithAgg(this.Phase(), context)
		defer this.cleanup(context)
		if !active {
			return
		}

		// use cached results if available
		cacheResult := this.plan.HasCacheResult()
		hasCache := cacheResult && this.results != nil

		var results []*datastore.IndexEntry
		if cacheResult && !hasCache {
			results = make([]*datastore.IndexEntry, 0, _MAX_RESULT_CACHE_SIZE)
		}

		if this.plan.HasDeltaKeyspace() {
			defer func() {
				this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
			}()
			this.keys, this.pool = this.scanDeltaKeyspace(this.plan.Keyspace(), parent,
				this.Phase(), context, this.plan.AllCovers())
		}

		if !hasCache {
			this.conn = datastore.NewIndexConnection(context)
			this.conn.SetSkipNewKeys(this.plan.SkipNewKeys())
			defer this.conn.Dispose()  // Dispose of the connection
			defer this.conn.SendStop() // Notify index that I have stopped
		}

		filter := this.plan.Filter()
		if filter != nil {
			filter.EnableInlistHash(&this.operatorCtx)
			defer filter.ResetMemory(&this.operatorCtx)
		}

		alias := this.plan.Term().Alias()

		var buildBitFltr, probeBitFltr bool
		buildBitFilters := this.plan.GetBuildBitFilters()
		if len(buildBitFilters) > 0 {
			this.createLocalBuildFilters(buildBitFilters)
			buildBitFltr = this.hasBuildBitFilter()
			defer this.setBuildBitFilters(alias, context)
		}
		probeBitFilters := this.plan.GetProbeBitFilters()
		if len(probeBitFilters) > 0 {
			err := this.getLocalProbeFilters(probeBitFilters, context)
			if err != nil {
				context.Error(err)
				return
			}
			probeBitFltr = this.hasProbeBitFilter()
			defer this.clearProbeBitFilters(context)
		}

		var squareRoot bool
		vectorPos := -1
		if !hasCache {
			var er errors.Error
			var indexVector *datastore.IndexVector
			if this.plan.IndexVector() != nil {
				index6, ok := this.plan.Index().(datastore.Index6)
				if !ok {
					context.Error(errors.NewExecutionInternalError("Vector index not Index6"))
					return
				}

				planIndexVector := this.plan.IndexVector()
				indexVector = &datastore.IndexVector{
					IndexKeyPos: planIndexVector.IndexKeyPos,
				}
				er = getIndexVector(planIndexVector, indexVector, parent,
					index6.VectorDimension(), &this.operatorCtx)
				if er != nil {
					context.Error(er)
					return
				}
				squareRoot = planIndexVector.SquareRoot
				vectorPos = planIndexVector.IndexKeyPos
			}
			var inlineFilter string
			if filter != nil {
				inlineFilter = filter.String()
			}
			var indexPartitionSets datastore.IndexPartitionSets
			planIndexPartitionSets := this.plan.IndexPartitionSets()
			if len(planIndexPartitionSets) > 0 {
				indexPartitionSets, er = getIndexPartitionSets(planIndexPartitionSets,
					parent, &this.operatorCtx)
				if er != nil {
					context.Error(er)
					return
				}
			}
			util.Fork(scanFork, scanDesc{this, context, parent, indexVector, inlineFilter, indexPartitionSets})
		}

		ok := true
		var docs uint64 = 0

		var countDocs = func() {
			if docs > 0 {
				context.AddPhaseCountWithAgg(this.Phase(), docs)
			}
		}
		defer countDocs()

		// at runtime treat Covers() and IndexKeys() the same way
		covers := this.plan.AllCovers()
		lcovers := len(covers)
		fullCover := this.plan.Covering()

		var entryKeys []int
		proj := this.plan.Projection()
		if proj != nil {
			entryKeys = proj.EntryKeys
		}

		// for right hand side of nested-loop join we don't want to include values
		// from the left hand side of the join in the returned scope value, however
		// we do want to include the "original" parent value, i.e. correlation value,
		// function arguments, WITH clause, etc.
		scope_value := parent
		if this.plan.IsUnderNL() {
			if val, ok := scope_value.(value.AnnotatedValue); ok {
				scope_value = val.GetParent()
			} else {
				scope_value = nil
			}
		}

		cacheIndex := 0
		for ok {
			var entry *datastore.IndexEntry
			var cont bool
			if hasCache {
				if cacheIndex < len(this.results) {
					entry = this.results[cacheIndex].Copy()
					cacheIndex++
				}
				cont = true
			} else {
				entry, cont = this.getItemEntry(this.conn)
			}
			if cont {
				if entry != nil {
					this.addInDocs(1)
					if _, sok := this.keys[entry.PrimaryKey]; !sok {
						av := this.newEmptyDocumentWithKey(entry.PrimaryKey, scope_value, context)
						if lcovers > 0 {

							for c, v := range this.plan.AllFilterCovers() {
								av.SetCover(c.Text(), v)
							}

							// Matches planner.builder.buildCoveringScan()
							for i, ek := range entry.EntryKey {
								if proj == nil || i < len(entryKeys) {
									if fullCover && i < len(entryKeys) {
										i = entryKeys[i]
									}

									if i < lcovers {
										if squareRoot && i == vectorPos {
											var dv value.Value
											if ek.Type() == value.NUMBER {
												ef := value.AsNumberValue(ek).Float64()
												if ef >= 0.0 {
													dv = value.NewValue(math.Sqrt(ef))
												} else {
													dv = value.NULL_VALUE
												}
											} else {
												dv = value.NULL_VALUE
											}
											av.SetCover(covers[i].Text(), dv)
										} else {
											av.SetCover(covers[i].Text(), ek)
										}
									}
								}
							}

							// Matches planner.builder.buildCoveringScan()
							if proj == nil || proj.PrimaryKey {
								av.SetCover(covers[len(covers)-1].Text(),
									value.NewValue(entry.PrimaryKey))
							}

							av.SetField(alias, av)

							if filter != nil {
								result, err := filter.Evaluate(av, &this.operatorCtx)
								if err != nil {
									context.Error(errors.NewEvaluationError(err, "filter"))
									return
								}
								if !result.Truth() {
									av.Recycle()
									continue
								}
							}
							if buildBitFltr && !this.buildBitFilters(av, &this.operatorCtx) {
								return
							}
							if probeBitFltr {
								ok1, pass := this.probeBitFilters(av, &this.operatorCtx)
								if !ok1 {
									return
								} else if !pass {
									av.Recycle()
									continue
								}
							}
						}
						if context.UseRequestQuota() {
							err := context.TrackValueSize(av.Size())
							if err != nil {
								context.Error(err)
								av.Recycle()
								ok = false
								break
							}
						}

						av.SetBit(this.bit)

						if cacheResult && !hasCache {
							if len(results) >= _MAX_RESULT_CACHE_SIZE {
								if this.plan.IsUnderNL() {
									context.Error(errors.NewNLInnerPrimaryDocsExceeded(alias, _MAX_RESULT_CACHE_SIZE))
								} else {
									context.Error(errors.NewSubqueryNumDocsExceeded(alias, _MAX_RESULT_CACHE_SIZE))
								}
								return
							}
							results = append(results, entry.Copy())
						}

						ok = this.sendItem(av)
						docs++
						if docs > _PHASE_UPDATE_COUNT {
							context.AddPhaseCountWithAgg(this.Phase(), docs)
							docs = 0
						}
					}
				} else {
					ok = false
				}
			} else {
				return
			}
		}
		if cacheResult && !hasCache {
			this.results, results = results, nil
		}
	})
}

func (this *IndexScan3) scan(context *Context, conn *datastore.IndexConnection, parent value.Value,
	indexVector *datastore.IndexVector, inlineFilter string, indexPartitionSets datastore.IndexPartitionSets) {

	defer context.Recover(nil) // Recover from any panic

	plan := this.plan
	index3 := plan.Index()

	// for nested-loop join we need to pass in values from left-hand-side (outer) of the join
	// for span evaluation

	groupAggs := plan.GroupAggs()
	dspans, empty, err := evalSpan3(plan.Spans(), parent, plan.HasDynamicInSpan(), &this.operatorCtx)

	// empty span with Index aggregation is present and no group by requies produce default row.
	// Therefore, do IndexScan

	if err != nil || (empty && (groupAggs == nil || len(groupAggs.Group) > 0)) {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		conn.Sender().Close()
		return
	}

	offset := evalLimitOffset(this.plan.Offset(), parent, int64(0), this.plan.Covering(), &this.operatorCtx)
	limit := evalLimitOffset(this.plan.Limit(), parent, math.MaxInt64, this.plan.Covering(), &this.operatorCtx)
	scanVector := context.ScanVectorSource().ScanVector(plan.Term().Namespace(), plan.Term().Path().Bucket())

	indexProjection, indexOrder, indexGroupAggs := planToScanMapping(index3, plan.Projection(),
		plan.OrderTerms(), plan.GroupAggs(), plan.Covers())

	if index6, ok := index3.(datastore.Index6); ok && indexVector != nil {
		index6.Scan6(context.RequestId(), dspans, plan.Reverse(), plan.Distinct(),
			indexProjection, offset, limit, indexGroupAggs, indexOrder,
			this.plan.IndexKeyNames(), inlineFilter, indexVector, indexPartitionSets,
			context.ScanConsistency(), scanVector, conn)
	} else {
		index3.Scan3(context.RequestId(), dspans, plan.Reverse(), plan.Distinct(),
			indexProjection, offset, limit, indexGroupAggs, indexOrder,
			context.ScanConsistency(), scanVector, conn)
	}
}

func evalSpan3(pspans plan.Spans2, parent value.Value, hasDynamicInSpan bool, context *opContext) (
	datastore.Spans2, bool, error) {
	spans := pspans
	if hasDynamicInSpan {
		numspans := len(pspans)
		minPos := 0
		maxPos := 0
		for _, ps := range pspans {
			for i, rg := range ps.Ranges {
				if !rg.IsDynamicIn() {
					continue
				}

				av, empty, err := evalOne(rg.GetDynamicInExpr(), context, parent)
				if err != nil {
					return nil, false, err
				}
				if !empty && av.Type() == value.ARRAY {
					arr := av.ActualForIndex().([]interface{})
					set := value.NewSet(len(arr), true, false)
					set.AddAll(arr)
					arr = set.Actuals()
					sort.Sort(value.NewSorter(value.NewValue(arr)))
					newlength := numspans + (maxPos-minPos+1)*(len(arr)-1)
					if newlength <= plan.FULL_SPAN_FANOUT {
						ospans := spans
						spans = make(plan.Spans2, 0, newlength)
						add := 0
						for j, sp := range ospans {
							if j >= minPos && j <= maxPos {
								for _, v := range arr {
									spn := sp.Copy()
									nrg := spn.Ranges[i]
									nrg.Low = expression.NewConstant(v)
									nrg.High = nrg.Low
									nrg.Inclusion = datastore.BOTH
									spans = append(spans, spn)
								}
								add = add + len(arr) - 1
							} else {
								spans = append(spans, sp)
							}
						}
						numspans = len(spans)
						maxPos = maxPos + add
					}
				}
			}
			minPos = maxPos + 1
			maxPos = minPos
		}
	}

	return evalSpan2(spans, parent, context)
}

func planToScanMapping(index datastore.Index, proj *plan.IndexProjection, indexOrderTerms plan.IndexKeyOrders,
	groupAggs *plan.IndexGroupAggregates, covers expression.Covers) (indexProjection *datastore.IndexProjection,
	indexOrder datastore.IndexKeyOrders, indexGroupAggs *datastore.IndexGroupAggregates) {

	if proj != nil {
		indexProjection = &datastore.IndexProjection{EntryKeys: proj.EntryKeys, PrimaryKey: proj.PrimaryKey}
	}

	if len(indexOrderTerms) > 0 {
		indexOrder = make(datastore.IndexKeyOrders, 0, len(indexOrderTerms))
		for _, o := range indexOrderTerms {
			indexOrder = append(indexOrder, &datastore.IndexKeyOrder{KeyPos: o.KeyPos, Desc: o.Desc})
		}
	}

	if groupAggs != nil {
		var group datastore.IndexGroupKeys
		var aggs datastore.IndexAggregates

		if len(groupAggs.Group) > 0 {
			group = make(datastore.IndexGroupKeys, 0, len(groupAggs.Group))
			for _, g := range groupAggs.Group {
				group = append(group, &datastore.IndexGroupKey{EntryKeyId: g.EntryKeyId,
					KeyPos: g.KeyPos, Expr: g.Expr})
			}
		}

		if len(groupAggs.Aggregates) > 0 {
			aggs = make(datastore.IndexAggregates, 0, len(groupAggs.Aggregates))
			for _, a := range groupAggs.Aggregates {
				aggs = append(aggs, &datastore.IndexAggregate{Operation: a.Operation,
					EntryKeyId: a.EntryKeyId, KeyPos: a.KeyPos, Expr: a.Expr,
					Distinct: a.Distinct})
			}
		}

		indexGroupAggs = &datastore.IndexGroupAggregates{Name: groupAggs.Name, Group: group,
			Aggregates: aggs, DependsOnIndexKeys: groupAggs.DependsOnIndexKeys,
			IndexKeyNames: getIndexKeyNames(index, covers), OneForPrimaryKey: groupAggs.DistinctDocid,
			AllowPartialAggr: groupAggs.Partial}
	}

	return
}

func (this *IndexScan3) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop/pause
func (this *IndexScan3) SendAction(action opAction) {
	this.connSendAction(this.conn, action)
}

func (this *IndexScan3) Done() {
	this.baseDone()
	this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
	if this.plan.HasCacheResult() && this.results != nil {
		this.results = nil
	}
}

func getIndexKeyNames(index datastore.Index, covers expression.Covers) []string {
	pos := 0
	names := make([]string, 0, len(covers))
	for _, key := range index.RangeKey() {
		if all, ok := key.(*expression.All); ok && all.Flatten() {
			for _, _ = range all.FlattenKeys().Operands() {
				names = append(names, covers[pos].Text())
				pos++
			}
		} else {
			names = append(names, covers[pos].Text())
			pos++
		}
	}
	names = append(names, covers[pos].Text()) // include META().id
	return names
}
