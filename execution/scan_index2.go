//  Copyright (c) 2014 Couchbase, Inc.
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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexScan2 struct {
	base
	plan         *plan.IndexScan2
	children     []Operator
	childChannel StopChannel
}

func NewIndexScan2(plan *plan.IndexScan2, context *Context) *IndexScan2 {
	rv := &IndexScan2{
		base: newBase(context),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan2(this)
}

func (this *IndexScan2) Copy() Operator {
	return &IndexScan2{
		base: this.base.copy(),
		plan: this.plan,
	}
}

func (this *IndexScan2) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		this.phaseTimes = func(d time.Duration) { context.AddPhaseTime(INDEX_SCAN, d) }
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer close(this.itemChannel)                // Broadcast that I have stopped
		defer this.notify()                          // Notify that I have stopped
		context.AddPhaseOperator(INDEX_SCAN)

		conn := datastore.NewIndexConnection(context)
		defer notifyConn(conn.StopChannel()) // Notify index that I have stopped

		go this.scan(context, conn)

		var entry *datastore.IndexEntry
		ok := true
		var docs uint64 = 0

		var countDocs = func() {
			if docs > 0 {
				context.AddPhaseCount(INDEX_SCAN, docs)
			}
		}
		defer countDocs()

		for ok {
			this.switchPhase(_CHANTIME) // could be _SERVTIME
			select {
			case <-this.stopChannel:
				return
			default:
			}

			select {
			case entry, ok = <-conn.EntryChannel():
				this.switchPhase(_EXECTIME)
				if ok {
					cv := value.NewScopeValue(make(map[string]interface{}), parent)
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
						av.SetCover(covers[len(covers)-1].Text(),
							value.NewValue(entry.PrimaryKey))

						av.SetField(this.plan.Term().Alias(), av)
					}

					av.SetBit(this.bit)
					ok = this.sendItem(av)
					docs++
					if docs > _PHASE_UPDATE_COUNT {
						context.AddPhaseCount(INDEX_SCAN, docs)
						docs = 0
					}
				}

			case <-this.stopChannel:
				return
			}
		}
	})
}

func (this *IndexScan2) scan(context *Context, conn *datastore.IndexConnection) {
	defer context.Recover() // Recover from any panic

	plan := this.plan

	dspans, empty, err := evalSpan2(plan.Spans(), context)
	if err != nil || empty {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		close(conn.EntryChannel())
		return
	}

	offset := evalLimitOffset(this.plan.Offset(), nil, int64(0), this.plan.Covering(), context)
	limit := evalLimitOffset(this.plan.Limit(), nil, math.MaxInt64, this.plan.Covering(), context)

	var indexProjection *datastore.IndexProjection

	proj := plan.Projection()
	scanVector := context.ScanVectorSource().ScanVector(plan.Term().Namespace(), plan.Term().Keyspace())

	if proj != nil {
		indexProjection = &datastore.IndexProjection{EntryKeys: proj.EntryKeys, PrimaryKey: proj.PrimaryKey}
	}

	plan.Index().Scan2(context.RequestId(), dspans, plan.Reverse(), plan.Distinct(), plan.Ordered(),
		indexProjection, offset, limit,
		context.ScanConsistency(), scanVector, conn)
}

func evalSpan2(pspans plan.Spans2, context *Context) (datastore.Spans2, bool, error) {
	var err error
	var empty bool

	dspans := make(datastore.Spans2, 0, len(pspans))

	for _, ps := range pspans {
		ds := &datastore.Span2{}

		ds.Seek, empty, err = eval(ps.Seek, context, nil)
		if err != nil {
			return nil, empty, err
		}

		ds.Ranges = make(datastore.Ranges2, 0, len(ps.Ranges))
		for _, psRange := range ps.Ranges {
			dsRange := &datastore.Range2{}
			dsRange.Inclusion = psRange.Inclusion

			dsRange.Low, empty, err = evalOne(psRange.Low, context, nil)
			if err != nil {
				return nil, empty, err
			}
			if empty {
				break
			}

			dsRange.High, empty, err = evalOne(psRange.High, context, nil)
			if err != nil {
				return nil, empty, err
			}
			if empty {
				break
			}

			ds.Ranges = append(ds.Ranges, dsRange)
		}

		if len(ds.Ranges) > 0 {
			dspans = append(dspans, ds)
		}
	}

	if len(dspans) == 0 {
		empty = true
	}

	return dspans, empty, nil
}

func (this *IndexScan2) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
