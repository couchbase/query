//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

var _EMPTY_SPAN2 *datastore.Span2 = &datastore.Span2{nil, datastore.Ranges2{&datastore.Range2{value.NULL_VALUE, value.NULL_VALUE, 0}}}

type IndexScan2 struct {
	base
	conn     *datastore.IndexConnection
	plan     *plan.IndexScan2
	children []Operator
	keys     map[string]bool
	pool     bool
}

func NewIndexScan2(plan *plan.IndexScan2, context *Context) *IndexScan2 {
	rv := &IndexScan2{plan: plan}
	newBase(&rv.base, context)
	rv.phase = INDEX_SCAN
	if p, ok := indexerPhase[plan.Index().Indexer().Name()]; ok {
		rv.phase = p.index
	}
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

func (this *IndexScan2) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexScan2) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhaseWithAgg(this.Phase(), context)
		defer this.cleanup(context)
		if !active {
			return
		}

		if this.plan.HasDeltaKeyspace() {
			defer func() {
				this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
			}()
			this.keys, this.pool = this.scanDeltaKeyspace(this.plan.Keyspace(), parent, this.Phase(), context, this.plan.Covers())
		}

		this.conn = datastore.NewIndexConnection(context)
		defer this.conn.Dispose()  // Dispose of the connection
		defer this.conn.SendStop() // Notify index that I have stopped

		go this.scan(context, this.conn, parent)

		ok := true
		var docs uint64 = 0

		var countDocs = func() {
			if docs > 0 {
				context.AddPhaseCountWithAgg(this.Phase(), docs)
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
	})
}

func (this *IndexScan2) scan(context *Context, conn *datastore.IndexConnection, parent value.Value) {
	defer context.Recover(nil) // Recover from any panic

	plan := this.plan

	// for nested-loop join we need to pass in values from left-hand-side (outer) of the join
	// for span evaluation
	dspans, empty, err := evalSpan2(plan.Spans(), parent, &this.operatorCtx)
	if err != nil || empty {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		conn.Sender().Close()
		return
	}

	offset := evalLimitOffset(this.plan.Offset(), parent, int64(0), this.plan.Covering(), &this.operatorCtx)
	limit := evalLimitOffset(this.plan.Limit(), parent, math.MaxInt64, this.plan.Covering(), &this.operatorCtx)

	var indexProjection *datastore.IndexProjection

	proj := plan.Projection()
	scanVector := context.ScanVectorSource().ScanVector(plan.Term().Namespace(), plan.Term().Path().Bucket())

	if proj != nil {
		indexProjection = &datastore.IndexProjection{EntryKeys: proj.EntryKeys, PrimaryKey: proj.PrimaryKey}
	}

	plan.Index().Scan2(context.RequestId(), dspans, plan.Reverse(), plan.Distinct(), plan.Ordered(),
		indexProjection, offset, limit,
		context.ScanConsistency(), scanVector, conn)
}

func evalSpan2(pspans plan.Spans2, parent value.Value, context *opContext) (datastore.Spans2, bool, error) {
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

		if !empty && len(ds.Ranges) > 0 {
			dspans = append(dspans, ds)
		}
	}

	empty = false
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

// send a stop/pause
func (this *IndexScan2) SendAction(action opAction) {
	this.connSendAction(this.conn, action)
}

func (this *IndexScan2) Done() {
	this.baseDone()
	this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
}
