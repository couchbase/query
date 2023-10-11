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
	"fmt"
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type PrimaryScan3 struct {
	base
	conn *datastore.IndexConnection
	plan *plan.PrimaryScan3
	keys map[string]bool
	pool bool
}

func NewPrimaryScan3(plan *plan.PrimaryScan3, context *Context) *PrimaryScan3 {
	rv := &PrimaryScan3{plan: plan}

	newBase(&rv.base, context)
	rv.phase = PRIMARY_SCAN
	if p, ok := indexerPhase[plan.Index().Indexer().Name()]; ok {
		rv.phase = p.primary
	}
	rv.output = rv
	return rv
}

func (this *PrimaryScan3) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan3(this)
}

func (this *PrimaryScan3) Copy() Operator {
	rv := &PrimaryScan3{}
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *PrimaryScan3) PlanOp() plan.Operator {
	return this.plan
}

func (this *PrimaryScan3) RunOnce(context *Context, parent value.Value) {
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
			this.keys, this.pool = this.scanDeltaKeyspace(this.plan.Keyspace(), parent, this.Phase(), context, nil)
		}

		this.scanPrimary(context, parent)
	})
}

func (this *PrimaryScan3) scanPrimary(context *Context, parent value.Value) {
	this.conn = datastore.NewIndexConnection(context)
	this.conn.SetPrimary()
	defer this.conn.Dispose()  // Dispose of the connection
	defer this.conn.SendStop() // Notify index that I have stopped

	offset := evalLimitOffset(this.plan.Offset(), parent, int64(0), false, context)
	limit := evalLimitOffset(this.plan.Limit(), parent, math.MaxInt64, false, context)

	go func() {
		primeStack()
		this.scanEntries(context, this.conn, offset, limit)
	}()

	nitems := uint64(0)

	var docs uint64 = 0
	defer func() {
		if docs > 0 {
			context.AddPhaseCountWithAgg(this.Phase(), docs)
		}
	}()

	var lastEntry *datastore.IndexEntry
	for {
		entry, ok := this.getItemEntry(this.conn)
		if ok {
			if entry != nil {
				if _, sok := this.keys[entry.PrimaryKey]; !sok {
					// current policy is to only count 'in' documents
					// from operators, not kv
					// add this.addInDocs(1) if this changes
					av := this.newEmptyDocumentWithKey(entry.PrimaryKey, parent, context)
					ok = this.sendItem(av)
					lastEntry = entry
					nitems++
					docs++
					if docs > _PHASE_UPDATE_COUNT {
						context.AddPhaseCountWithAgg(this.Phase(), docs)
						docs = 0
					}
				}
			} else {
				break
			}
		} else {
			return
		}
	}

	emsg := "Primary index scan timeout - resorting to chunked scan"
	for this.conn.Timeout() {
		// Offset, Aggregates, Order needs to be exact.
		// On timeout return error because we cann't stitch the output
		if this.plan.Offset() != nil || len(this.plan.OrderTerms()) > 0 || this.plan.GroupAggs() != nil ||
			lastEntry == nil {
			context.Error(errors.NewCbIndexScanTimeoutError(nil))
			return
		}

		logging.Errora(func() string {
			return fmt.Sprintf("%s chunkSize=%v startingEntry=%v", emsg, nitems,
				stringifyIndexEntry(lastEntry))
		})

		// do chunked scans; lastEntry the starting point
		// old connection will be disposed by the defer above
		this.conn = datastore.NewIndexConnection(context)
		this.conn.SetPrimary()
		lastEntry, nitems = this.scanPrimaryChunk(context, parent, this.conn, lastEntry, limit)
		emsg = "Primary index chunked scan"
	}
}

func (this *PrimaryScan3) scanPrimaryChunk(context *Context, parent value.Value, conn *datastore.IndexConnection,
	indexEntry *datastore.IndexEntry, limit int64) (*datastore.IndexEntry, uint64) {
	this.switchPhase(_EXECTIME)
	defer this.switchPhase(_NOTIME)
	defer conn.Dispose()  // Dispose of the connection
	defer conn.SendStop() // Notify index that I have stopped

	go func() {
		primeStack()
		this.scanChunk(context, conn, limit, indexEntry)
	}()

	nitems := uint64(0)
	var docs uint64 = 0
	defer func() {
		if nitems > 0 {
			context.AddPhaseCountWithAgg(this.Phase(), docs)
		}
	}()

	var lastEntry *datastore.IndexEntry
	for {
		entry, ok := this.getItemEntry(conn)
		if ok {
			if entry != nil {
				if _, sok := this.keys[entry.PrimaryKey]; !sok {
					av := this.newEmptyDocumentWithKey(entry.PrimaryKey, parent, context)
					ok = this.sendItem(av)
					lastEntry = entry
					nitems++
					docs++
					if docs > _PHASE_UPDATE_COUNT {
						context.AddPhaseCountWithAgg(this.Phase(), docs)
						docs = 0
					}
				}
			} else {
				break
			}
		} else {
			return nil, nitems
		}
	}
	return lastEntry, nitems
}

func (this *PrimaryScan3) scanEntries(context *Context, conn *datastore.IndexConnection, offset, limit int64) {
	defer context.Recover(nil) // Recover from any panic

	index := this.plan.Index()
	term := this.plan.Term()
	scanVector := context.ScanVectorSource().ScanVector(term.Namespace(), term.Path().Bucket())
	indexProjection, indexOrder, indexGroupAggs := planToScanMapping(index, this.plan.Projection(),
		this.plan.OrderTerms(), this.plan.GroupAggs(), nil)

	index.ScanEntries3(context.RequestId(), indexProjection, offset, limit, indexGroupAggs, indexOrder,
		context.ScanConsistency(), scanVector, conn)
}

func (this *PrimaryScan3) scanChunk(context *Context, conn *datastore.IndexConnection, limit int64, indexEntry *datastore.IndexEntry) {
	defer context.Recover(nil) // Recover from any panic
	ds := &datastore.Span{}
	// do the scan starting from, but not including, the given index entry:
	ds.Range = datastore.Range{
		Inclusion: datastore.NEITHER,
		Low:       []value.Value{value.NewValue(indexEntry.PrimaryKey)},
	}
	term := this.plan.Term()
	scanVector := context.ScanVectorSource().ScanVector(term.Namespace(), term.Path().Bucket())
	this.plan.Index().Scan(context.RequestId(), ds, true, limit,
		context.ScanConsistency(), scanVector, conn)
}

func (this *PrimaryScan3) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop/pause
func (this *PrimaryScan3) SendAction(action opAction) {
	this.connSendAction(this.conn, action)
}

func (this *PrimaryScan3) Done() {
	this.baseDone()
	this.keys, this.pool = this.deltaKeyspaceDone(this.keys, this.pool)
}
