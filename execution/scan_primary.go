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
	"fmt"
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type PrimaryScan struct {
	base
	plan *plan.PrimaryScan
}

func NewPrimaryScan(plan *plan.PrimaryScan, context *Context) *PrimaryScan {
	rv := &PrimaryScan{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.newStopChannel()
	rv.output = rv
	return rv
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) Copy() Operator {
	rv := &PrimaryScan{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *PrimaryScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.active()
		defer this.close(context)
		this.setExecPhase(PRIMARY_SCAN, context)
		defer this.notify() // Notify that I have stopped

		this.scanPrimary(context, parent)
	})
}

func stringifyIndexEntry(lastEntry *datastore.IndexEntry) string {
	str := fmt.Sprintf("EntryKey : <ud>%v</ud>\n", lastEntry.EntryKey)
	str += fmt.Sprintf("Primary Key : <ud>%v</ud>\n", lastEntry.PrimaryKey)
	return str
}

func (this *PrimaryScan) scanPrimary(context *Context, parent value.Value) {
	this.switchPhase(_EXECTIME)
	defer this.switchPhase(_NOTIME)
	conn := datastore.NewIndexConnection(context)
	conn.SetPrimary()
	defer notifyConn(conn.StopChannel()) // Notify index that I have stopped

	limit := evalLimitOffset(this.plan.Limit(), nil, math.MaxInt64, false, context)

	go this.scanEntries(context, conn, limit)

	nitems := uint64(0)

	var docs uint64 = 0
	defer func() {
		if docs > 0 {
			context.AddPhaseCount(PRIMARY_SCAN, docs)
		}
	}()

	var lastEntry *datastore.IndexEntry
	for {
		entry, ok := this.getItemEntry(conn.EntryChannel())
		if ok {
			if entry != nil {
				// current policy is to only count 'in' documents
				// from operators, not kv
				// add this.addInDocs(1) if this changes
				cv := value.NewScopeValue(make(map[string]interface{}), parent)
				av := value.NewAnnotatedValue(cv)
				av.SetAttachment("meta", map[string]interface{}{"id": entry.PrimaryKey})
				ok = this.sendItem(av)
				lastEntry = entry
				nitems++
				docs++
				if docs > _PHASE_UPDATE_COUNT {
					context.AddPhaseCount(PRIMARY_SCAN, docs)
					docs = 0
				}
			} else {
				break
			}
		} else {
			return
		}
	}

	emsg := "Primary index scan timeout - resorting to chunked scan"
	for conn.Timeout() {
		if lastEntry == nil {
			// no key for chunked scans (primary scan returned 0 items)
			context.Error(errors.NewCbIndexScanTimeoutError(nil))
			return
		}

		logging.Errorp(emsg, logging.Pair{"chunkSize", nitems},
			logging.Pair{"startingEntry", stringifyIndexEntry(lastEntry)})

		// do chunked scans; lastEntry the starting point
		conn = datastore.NewIndexConnection(context)
		conn.SetPrimary()
		lastEntry, nitems = this.scanPrimaryChunk(context, parent, conn, lastEntry, limit)
		emsg = "Primary index chunked scan"
	}
}

func (this *PrimaryScan) scanPrimaryChunk(context *Context, parent value.Value, conn *datastore.IndexConnection,
	indexEntry *datastore.IndexEntry, limit int64) (*datastore.IndexEntry, uint64) {

	this.switchPhase(_EXECTIME)
	defer this.switchPhase(_NOTIME)
	defer notifyConn(conn.StopChannel()) // Notify index that I have stopped

	go this.scanChunk(context, conn, limit, indexEntry)

	nitems := uint64(0)
	var docs uint64 = 0
	defer func() {
		if nitems > 0 {
			context.AddPhaseCount(PRIMARY_SCAN, docs)
		}
	}()

	var lastEntry *datastore.IndexEntry
	for {
		entry, ok := this.getItemEntry(conn.EntryChannel())
		if ok {
			if entry != nil {
				cv := value.NewScopeValue(make(map[string]interface{}), parent)
				av := value.NewAnnotatedValue(cv)
				av.SetAttachment("meta", map[string]interface{}{"id": entry.PrimaryKey})
				ok = this.sendItem(av)
				lastEntry = entry
				nitems++
				docs++
				if docs > _PHASE_UPDATE_COUNT {
					context.AddPhaseCount(PRIMARY_SCAN, docs)
					docs = 0
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

func (this *PrimaryScan) scanEntries(context *Context, conn *datastore.IndexConnection, limit int64) {
	defer context.Recover() // Recover from any panic

	keyspace := this.plan.Keyspace()
	scanVector := context.ScanVectorSource().ScanVector(keyspace.NamespaceId(), keyspace.Name())

	index := this.plan.Index()
	index.ScanEntries(context.RequestId(), limit, context.ScanConsistency(), scanVector, conn)
}

func (this *PrimaryScan) scanChunk(context *Context, conn *datastore.IndexConnection, limit int64, indexEntry *datastore.IndexEntry) {
	defer context.Recover() // Recover from any panic
	ds := &datastore.Span{}
	// do the scan starting from, but not including, the given index entry:
	ds.Range = datastore.Range{
		Inclusion: datastore.NEITHER,
		Low:       []value.Value{value.NewValue(indexEntry.PrimaryKey)},
	}
	keyspace := this.plan.Keyspace()
	scanVector := context.ScanVectorSource().ScanVector(keyspace.NamespaceId(), keyspace.Name())
	this.plan.Index().Scan(context.RequestId(), ds, true, limit,
		context.ScanConsistency(), scanVector, conn)
}

func (this *PrimaryScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop
func (this *PrimaryScan) SendStop() {
	this.chanSendStop()
}
