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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

var _EMPTY_SPAN2 *datastore.Span2 = &datastore.Span2{nil, datastore.Ranges2{&datastore.Range2{value.NULL_VALUE, value.NULL_VALUE, 0}}}

type IndexScan2 struct {
	base
	conn     *datastore.IndexConnection
	plan     *plan.IndexScan2
	children []Operator
}

func NewIndexScan2(plan *plan.IndexScan2, context *Context) *IndexScan2 {
	rv := &IndexScan2{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan2(this)
}

func (this *IndexScan2) Copy() Operator {
	rv := &IndexScan2{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexScan2) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_SCAN, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped
		if !active {
			return
		}

		this.conn = datastore.NewIndexConnection(context)
		defer this.conn.Dispose()  // Dispose of the connection
		defer this.conn.SendStop() // Notify index that I have stopped

		go this.scan(context, this.conn, parent)

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
		covers := this.plan.Covers()
		lcovers := len(covers)

		var entryKeys []int
		proj := this.plan.Projection()
		if proj != nil {
			entryKeys = proj.EntryKeys
		}

		if this.plan.Term().IsUnderNL() {
			scope_value = nil
		}

		for ok {
			entry, cont := this.getItemEntry(this.conn)
			if cont {
				if entry != nil {
					av := this.newEmptyDocumentWithKey(entry.PrimaryKey, scope_value, context)
					if lcovers > 0 {

						for c, v := range this.plan.FilterCovers() {
							av.SetCover(c.Text(), v)
						}

						// Matches planner.builder.buildCoveringScan()
						for i, ek := range entry.EntryKey {
							if proj == nil || i < len(entryKeys) {
								if i < len(entryKeys) {
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

func (this *IndexScan2) scan(context *Context, conn *datastore.IndexConnection, parent value.Value) {
	defer context.Recover(nil) // Recover from any panic

	plan := this.plan

	// for nested-loop join we need to pass in values from left-hand-side (outer) of the join
	// for span evaluation
	dspans, empty, err := evalSpan2(plan.Spans(), parent, context)
	if err != nil || empty {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		conn.Sender().Close()
		return
	}

	offset := evalLimitOffset(this.plan.Offset(), parent, int64(0), this.plan.Covering(), context)
	limit := evalLimitOffset(this.plan.Limit(), parent, math.MaxInt64, this.plan.Covering(), context)

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

func evalSpan2(pspans plan.Spans2, parent value.Value, context *Context) (datastore.Spans2, bool, error) {
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

			dsRange.Low, empty, err = evalOne(psRange.Low, context, parent)
			if err != nil {
				return nil, empty, err
			}
			if empty {
				break
			}

			dsRange.High, empty, err = evalOne(psRange.High, context, parent)
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
		dspans = append(dspans, _EMPTY_SPAN2)
	}

	return dspans, empty, nil
}

func (this *IndexScan2) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop
func (this *IndexScan2) SendStop() {
	this.connSendStop(this.conn)
}
