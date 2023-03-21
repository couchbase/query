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
	results  value.AnnotatedValues
	context  *Context
}

func NewIndexScan3(plan *plan.IndexScan3, context *Context) *IndexScan3 {
	rv := &IndexScan3{}
	rv.plan = plan

	newBase(&rv.base, context)
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
	scan    *IndexScan3
	context *Context
	parent  value.Value
}

func scanFork(p interface{}) {
	d := p.(scanDesc)
	d.scan.scan(d.context, d.scan.conn, d.parent)
}

func (this *IndexScan3) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_SCAN, context)
		defer this.cleanup(context)
		if !active {
			return
		}

		// use cached results if available
		cacheResult := this.plan.HasCacheResult()
		if cacheResult && this.results != nil {
			for _, av := range this.results {
				av.Track()
				if context.UseRequestQuota() {
					err := context.TrackValueSize(av.Size())
					if err != nil {
						context.Error(err)
						av.Recycle()
						return
					}
				}
				if !this.sendItem(av) {
					av.Recycle()
					break
				}
			}
			return
		}

		var results value.AnnotatedValues
		if cacheResult {
			var size int = _MAX_RESULT_CACHE_SIZE
			cardinality := int(math.Ceil(this.plan.Cardinality()))
			if cardinality > 0 && cardinality < size {
				size = cardinality
			}
			results = make(value.AnnotatedValues, 0, size)
			this.context = context
			defer func() {
				for _, av := range results {
					av.Recycle()
				}
			}()
		}

		if this.plan.HasDeltaKeyspace() {
			defer func() {
				this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
			}()
			this.keys, this.pool = this.scanDeltaKeyspace(this.plan.Keyspace(), parent,
				INDEX_SCAN, context, this.plan.AllCovers())
		}

		this.conn = datastore.NewIndexConnection(context)
		this.conn.SetSkipNewKeys(this.plan.SkipNewKeys())
		defer this.conn.Dispose()  // Dispose of the connection
		defer this.conn.SendStop() // Notify index that I have stopped

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

		util.Fork(scanFork, scanDesc{this, context, parent})

		ok := true
		var docs uint64 = 0

		var countDocs = func() {
			if docs > 0 {
				context.AddPhaseCount(INDEX_SCAN, docs)
			}
		}
		defer countDocs()

		// for right hand side of nested-loop join we don't want to include parent values
		// in the returned scope value
		scope_value := parent

		// at runtime treat Covers() and IndexKeys() the same way
		covers := this.plan.AllCovers()
		lcovers := len(covers)
		fullCover := this.plan.Covering()

		var entryKeys []int
		proj := this.plan.Projection()
		if proj != nil {
			entryKeys = proj.EntryKeys
		}

		if this.plan.IsUnderNL() {
			scope_value = nil
		}

		for ok {
			entry, cont := this.getItemEntry(this.conn)
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
										av.SetCover(covers[i].Text(), ek)
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
							if context.UseRequestQuota() {
								err := context.TrackValueSize(av.Size())
								if err != nil {
									context.Error(err)
									av.Recycle()
									ok = false
									break
								}
							}
						}

						av.SetBit(this.bit)

						if cacheResult {
							av.Track()
							if context.UseRequestQuota() {
								err := context.TrackValueSize(av.Size())
								if err != nil {
									context.Error(errors.NewMemoryQuotaExceededError())
									av.Recycle()
									return
								}
							}
							if len(results) >= _MAX_RESULT_CACHE_SIZE {
								context.Error(errors.NewNLInnerPrimaryDocsExceeded(this.plan.Term().Alias(), _MAX_RESULT_CACHE_SIZE))
								av.Recycle()
								return
							}
							results = append(results, av)
						}

						ok = this.sendItem(av)
						docs++
						if docs > _PHASE_UPDATE_COUNT {
							context.AddPhaseCount(INDEX_SCAN, docs)
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
		this.results, results = results, nil
	})
}

func (this *IndexScan3) scan(context *Context, conn *datastore.IndexConnection, parent value.Value) {
	defer context.Recover(nil) // Recover from any panic

	plan := this.plan

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

	indexProjection, indexOrder, indexGroupAggs := planToScanMapping(plan.Index(), plan.Projection(),
		plan.OrderTerms(), plan.GroupAggs(), plan.Covers())

	plan.Index().Scan3(context.RequestId(), dspans, plan.Reverse(), plan.Distinct(),
		indexProjection, offset, limit, indexGroupAggs, indexOrder,
		context.ScanConsistency(), scanVector, conn)
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
		context := this.context
		for _, av := range this.results {
			if context != nil && context.UseRequestQuota() {
				context.ReleaseValueSize(av.Size())
			}
			av.Recycle()
		}
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
