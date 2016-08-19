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
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexNest struct {
	joinBase
	plan *plan.IndexNest
}

func NewIndexNest(plan *plan.IndexNest) *IndexNest {
	rv := &IndexNest{
		joinBase: newJoinBase(),
		plan:     plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexNest(this)
}

func (this *IndexNest) Copy() Operator {
	return &IndexNest{
		joinBase: this.joinBase.copy(),
		plan:     this.plan,
	}
}

func (this *IndexNest) RunOnce(context *Context, parent value.Value) {
	start := time.Now()
	this.runConsumer(this, context, parent)
	t := time.Since(start) - this.chanTime
	context.AddPhaseTime("index_nest", t)
	this.plan.AddTime(t)
}

func (this *IndexNest) processItem(item value.AnnotatedValue, context *Context) bool {
	idv, e := this.plan.IdExpr().Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, fmt.Sprintf("NEST FOR %s", this.plan.For())))
		return false
	}

	var entry *datastore.IndexEntry
	entries := _INDEX_ENTRY_POOL.Get()
	defer _INDEX_ENTRY_POOL.Put(entries)

	if idv.Type() == value.STRING {
		var wg sync.WaitGroup
		defer wg.Wait()

		id := idv.Actual().(string)
		conn := datastore.NewIndexConnection(context)
		defer notifyConn(conn.StopChannel()) // Notify index that I have stopped

		wg.Add(1)
		go this.scan(id, context, conn, &wg)

		ok := true
		for ok {
			select {
			case <-this.stopChannel:
				return false
			default:
			}

			select {
			case entry, ok = <-conn.EntryChannel():
				if ok {
					entries = append(entries, entry)
				}
			case <-this.stopChannel:
				return false
			}
		}
	}

	if len(this.plan.Covers()) != 0 {
		return this.nestCoveredEntries(item, entries, context)
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

func (this *IndexNest) scan(id string, context *Context,
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

func (this *IndexNest) nestCoveredEntries(item value.AnnotatedValue,
	entries []*datastore.IndexEntry, context *Context) (ok bool) {
	var nvs []interface{}

	if len(entries) > 0 {
		covers := this.plan.Covers()
		nvs = make([]interface{}, len(entries))
		for i, entry := range entries {
			nv := value.NewAnnotatedValue(nil)
			meta := map[string]interface{}{"id": entry.PrimaryKey}
			nv.SetAttachment("meta", meta)

			for i, c := range covers {
				nv.SetCover(c.Text(), entry.EntryKey[i])
			}

			nvs[i] = nv
		}
	}

	if len(nvs) > 0 {
		item.SetField(this.plan.Term().Alias(), nvs)
	} else {
		if !this.plan.Outer() {
			return true
		}

		item.SetField(this.plan.Term().Alias(), value.EMPTY_ARRAY_VALUE)
	}

	return this.sendItem(item)
}

func (this *IndexNest) afterItems(context *Context) {
	if len(this.plan.Covers()) == 0 {
		this.flushBatch(context)
	}
}

func (this *IndexNest) flushBatch(context *Context) bool {
	defer this.releaseBatch()

	if len(this.joinBatch) == 0 {
		return true
	}

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	pairMap := _STRING_ANNOTATED_POOL.Get()

	defer _STRING_KEYCOUNT_POOL.Put(keyCount)
	defer _STRING_ANNOTATED_POOL.Put(pairMap)

	fetchOk := this.joinFetch(this.plan.Keyspace(), keyCount, pairMap, context)

	return fetchOk && this.nestEntries(keyCount, pairMap, this.plan.Outer(), this.plan.Term().Alias())
}

var _INDEX_ENTRY_POOL = datastore.NewIndexEntryPool(16)
