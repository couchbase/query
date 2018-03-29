//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"encoding/json"
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexScan3 struct {
	base
	plan     *plan.IndexScan3
	children []Operator
}

func NewIndexScan3(plan *plan.IndexScan3, context *Context) *IndexScan3 {
	rv := &IndexScan3{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.newStopChannel()
	rv.output = rv
	return rv
}

func (this *IndexScan3) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan3(this)
}

func (this *IndexScan3) Copy() Operator {
	rv := &IndexScan3{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexScan3) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_SCAN, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped

		conn := datastore.NewIndexConnection(context)
		defer notifyConn(conn.StopChannel()) // Notify index that I have stopped

		go this.scan(context, conn, parent)

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
		if this.plan.Term().IsUnderNL() {
			scope_value = nil
		}

		for ok {
			entry, cont := this.getItemEntry(conn.EntryChannel())
			if cont {
				if entry != nil {
					cv := value.NewScopeValue(make(map[string]interface{}), scope_value)
					av := value.NewAnnotatedValue(cv)

					// For downstream Fetch
					meta := map[string]interface{}{"id": entry.PrimaryKey}
					av.SetAttachment("meta", meta)

					covers := this.plan.Covers()
					if len(covers) > 0 {

						for c, v := range this.plan.FilterCovers() {
							av.SetCover(c.Text(), v)
						}

						var entryKeys []int
						proj := this.plan.Projection()
						if proj != nil {
							entryKeys = proj.EntryKeys
						}

						// Matches planner.builder.buildCoveringScan()
						for i, ek := range entry.EntryKey {
							j := i
							if i < len(entryKeys) {
								j = entryKeys[i]
							}
							av.SetCover(covers[j].Text(), ek)
						}

						// Matches planner.builder.buildCoveringScan()
						if proj == nil || proj.PrimaryKey {
							av.SetCover(covers[len(covers)-1].Text(),
								value.NewValue(entry.PrimaryKey))
						}

						av.SetField(this.plan.Term().Alias(), av)
					}

					av.SetBit(this.bit)
					ok = this.sendItem(av)
					docs++
					if docs > _PHASE_UPDATE_COUNT {
						context.AddPhaseCount(INDEX_SCAN, docs)
						docs = 0
					}
				} else {
					ok = false
				}
			} else {
				return
			}
		}
	})
}

func (this *IndexScan3) scan(context *Context, conn *datastore.IndexConnection, parent value.Value) {
	defer context.Recover() // Recover from any panic

	plan := this.plan

	// for nested-loop join we need to pass in values from left-hand-side (outer) of the join
	// for span evaluation
	outer_values := parent
	if !this.plan.Term().IsUnderNL() {
		outer_values = nil
	}

	groupAggs := plan.GroupAggs()
	dspans, empty, err := evalSpan2(plan.Spans(), outer_values, context)

	// empty span with Index aggregation is present and no group by requies produce default row.
	// Therefore, do IndexScan

	if err != nil || (empty && (groupAggs == nil || len(groupAggs.Group) > 0)) {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		close(conn.EntryChannel())
		return
	}

	offset := evalLimitOffset(this.plan.Offset(), nil, int64(0), this.plan.Covering(), context)
	limit := evalLimitOffset(this.plan.Limit(), nil, math.MaxInt64, this.plan.Covering(), context)
	scanVector := context.ScanVectorSource().ScanVector(plan.Term().Namespace(), plan.Term().Keyspace())

	indexProjection, indexOrder, indexGroupAggs := planToScanMapping(plan.Index(), plan.Projection(),
		plan.OrderTerms(), plan.GroupAggs(), plan.Covers())

	plan.Index().Scan3(context.RequestId(), dspans, plan.Reverse(), plan.Distinct(),
		indexProjection, offset, limit, indexGroupAggs, indexOrder,
		context.ScanConsistency(), scanVector, conn)
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

		// include META().id which is at nKeys+1
		nKeys := len(index.RangeKey())
		IndexKeyNames := make([]string, 0, nKeys+1)
		for i := 0; i <= nKeys; i++ {
			IndexKeyNames = append(IndexKeyNames, covers[i].Text())
		}

		indexGroupAggs = &datastore.IndexGroupAggregates{Name: groupAggs.Name, Group: group,
			Aggregates: aggs, DependsOnIndexKeys: groupAggs.DependsOnIndexKeys,
			IndexKeyNames: IndexKeyNames, OneForPrimaryKey: groupAggs.DistinctDocid,
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

// send a stop
func (this *IndexScan3) SendStop() {
	this.chanSendStop()
}
