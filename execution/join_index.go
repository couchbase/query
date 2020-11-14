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
	"sync"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type IndexJoin struct {
	joinBase
	sync.Mutex
	conn     *datastore.IndexConnection
	plan     *plan.IndexJoin
	joinTime time.Duration
}

func NewIndexJoin(plan *plan.IndexJoin, context *Context) *IndexJoin {
	rv := &IndexJoin{
		plan: plan,
	}

	newJoinBase(&rv.joinBase, context)
	rv.execPhase = INDEX_JOIN
	rv.output = rv
	return rv
}

func (this *IndexJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexJoin(this)
}

func (this *IndexJoin) Copy() Operator {
	rv := &IndexJoin{
		plan: this.plan,
	}
	this.joinBase.copy(&rv.joinBase)
	return rv
}

func (this *IndexJoin) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexJoin) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IndexJoin) processItem(item value.AnnotatedValue, context *Context) bool {
	defer this.switchPhase(_EXECTIME)
	idv, e := this.plan.IdExpr().Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, fmt.Sprintf("JOIN FOR %s", this.plan.For())))
		return false
	}

	entries := _INDEX_ENTRY_POOL.Get()
	defer _INDEX_ENTRY_POOL.Put(entries)

	if idv.Type() == value.STRING {
		var wg sync.WaitGroup
		defer wg.Wait()

		id := idv.Actual().(string)
		this.Lock()
		this.conn = datastore.NewIndexConnection(context)
		defer func() {
			this.Lock()
			this.conn = nil
			this.Unlock()
		}()
		defer this.conn.Dispose()  // Dispose of the connection
		defer this.conn.SendStop() // Notify index that I have stopped
		this.Unlock()

		wg.Add(1)
		go this.scan(id, context, this.conn, &wg)

		for {
			entry, cont := this.getItemEntry(this.conn)
			if cont {
				if entry != nil {
					// current policy is to only count 'in' documents
					// from operators, not kv
					// add this.addInDocs(1) if this changes
					entries = append(entries, entry)
				} else {
					break
				}
			} else {
				return false
			}
		}
	}

	if this.plan.Covering() {
		return this.joinCoveredEntries(item, entries, context)
	} else {
		var doc value.AnnotatedJoinPair

		doc.Value = item
		if len(entries) != 0 {
			doc.Keys = make([]string, 0, len(entries))
			for _, entry := range entries {
				doc.Keys = append(doc.Keys, entry.PrimaryKey)
			}
		}

		return this.joinEnbatch(doc, this, context)
	}
}

func (this *IndexJoin) scan(id string, context *Context,
	conn *datastore.IndexConnection, wg *sync.WaitGroup) {
	span := &datastore.Span{}
	span.Range.Inclusion = datastore.BOTH
	span.Range.Low = value.Values{value.NewValue(id)}
	span.Range.High = span.Range.Low

	consistency := context.ScanConsistency()
	if consistency == datastore.AT_PLUS {
		consistency = datastore.SCAN_PLUS
	}

	this.plan.Index().Scan(context.RequestId(), span, false,
		math.MaxInt64, consistency, nil, conn)
	wg.Done()
}

func (this *IndexJoin) joinCoveredEntries(item value.AnnotatedValue,
	entries []*datastore.IndexEntry, context *Context) (ok bool) {
	if len(entries) == 0 {
		return !this.plan.Outer() || this.sendItem(item)
	}

	t := util.Now()
	defer func() {
		this.joinTime += util.Since(t)
	}()

	covers := this.plan.Covers()
	filterCovers := this.plan.FilterCovers()

	useQuota := context.UseRequestQuota()
	for j, entry := range entries {
		var joined value.AnnotatedValue
		var size uint64

		if j < len(entries)-1 {
			joined = item.Copy().(value.AnnotatedValue)
			if useQuota {
				size = joined.Size()
			}
		} else {
			joined = item
		}

		// FIXME covers size
		for c, v := range filterCovers {
			joined.SetCover(c.Text(), v)
		}

		for i, ek := range entry.EntryKey {
			joined.SetCover(covers[i].Text(), ek)
		}

		joined.SetCover(covers[len(covers)-1].Text(),
			value.NewValue(entry.PrimaryKey))

		// For chained INDEX JOIN's
		jv := this.setDocumentKey(entry.PrimaryKey, value.NewAnnotatedValue(nil), 0, context)
		joined.SetField(this.plan.Term().Alias(), jv)

		if useQuota && context.TrackValueSize(size) {
			context.Error(errors.NewMemoryQuotaExceededError())
			joined.Recycle()
			return false
		}
		if !this.sendItem(joined) {
			return false
		}
	}
	// TODO Recycle

	return true
}

func (this *IndexJoin) afterItems(context *Context) {
	if len(this.plan.Covers()) == 0 {
		this.flushBatch(context)
	}
}

func (this *IndexJoin) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	if len(this.joinBatch) == 0 {
		return true
	}

	timer := util.Now()

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	pairMap := _STRING_ANNOTATED_POOL.Get()

	defer _STRING_KEYCOUNT_POOL.Put(keyCount)
	defer _STRING_ANNOTATED_POOL.Put(pairMap)
	defer func() {
		this.joinTime += util.Since(timer)
	}()

	fetchOk := this.joinFetch(this.plan.Keyspace(), keyCount, pairMap, context)

	return fetchOk && this.joinEntries(keyCount, pairMap, this.plan.Outer(), nil, this.plan.Term().Alias(), context)
}

func (this *IndexJoin) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop
func (this *IndexJoin) SendAction(action opAction) {
	rv := this.baseSendAction(action)
	this.Lock()
	if rv && this.conn != nil {
		this.conn.SendStop()
	}
	this.Unlock()
}
