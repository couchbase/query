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

type IndexJoin struct {
	base
	plan     *plan.IndexJoin
	joinTime time.Duration
}

func NewIndexJoin(plan *plan.IndexJoin) *IndexJoin {
	rv := &IndexJoin{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexJoin(this)
}

func (this *IndexJoin) Copy() Operator {
	return &IndexJoin{
		base: this.base.copy(),
		plan: this.plan,
	}
}

func (this *IndexJoin) RunOnce(context *Context, parent value.Value) {
	start := time.Now()
	this.runConsumer(this, context, parent)
	t := time.Since(start) - this.joinTime - this.chanTime
	context.AddPhaseTime("index_join", t)
	this.plan.AddTime(t)
}

func (this *IndexJoin) processItem(item value.AnnotatedValue, context *Context) bool {
	idv, e := this.plan.IdExpr().Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, fmt.Sprintf("JOIN FOR %s", this.plan.For())))
		return false
	}

	found, foundOne := false, false

	if idv.Type() == value.STRING {
		var wg sync.WaitGroup
		defer wg.Wait()

		id := idv.Actual().(string)
		conn := datastore.NewIndexConnection(context)
		defer notifyConn(conn.StopChannel()) // Notify index that I have stopped

		wg.Add(1)
		go this.scan(id, context, conn, &wg)

		var entry *datastore.IndexEntry
		ok := true
		for ok {
			select {
			case <-this.stopChannel:
				return false
			default:
			}

			select {
			case entry, ok = <-conn.EntryChannel():
				t := time.Now()

				if ok {
					foundOne, ok = this.joinEntry(item, entry, context)
					found = found || foundOne
				}

				this.joinTime += time.Since(t)
			case <-this.stopChannel:
				return false
			}
		}
	}

	return found || !this.plan.Outer() || this.sendItem(item)
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

func (this *IndexJoin) joinEntry(item value.AnnotatedValue,
	entry *datastore.IndexEntry, context *Context) (found, ok bool) {
	var jv value.AnnotatedValue
	covers := this.plan.Covers()

	if len(covers) == 0 {
		jv, ok = this.fetch(entry, context)
		if jv == nil || !ok {
			return jv != nil, ok
		}
	} else {
		jv = value.NewAnnotatedValue(nil)
		meta := map[string]interface{}{"id": entry.PrimaryKey}
		jv.SetAttachment("meta", meta)

		for i, c := range covers {
			jv.SetCover(c.Text(), entry.EntryKey[i])
		}
	}

	joined := item.Copy().(value.AnnotatedValue)
	joined.SetField(this.plan.Term().Alias(), jv)
	return true, this.sendItem(joined)
}

func (this *IndexJoin) fetch(entry *datastore.IndexEntry, context *Context) (
	value.AnnotatedValue, bool) {
	// Build list of keys
	keys := []string{entry.PrimaryKey}

	// Fetch
	pairs, errs := this.plan.Keyspace().Fetch(keys)

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	if len(pairs) == 0 {
		return nil, fetchOk
	}

	pair := pairs[0]
	av := pair.Value

	// Apply projection, if any
	projection := this.plan.Term().Projection()
	if projection != nil {
		projectedItem, e := projection.Evaluate(av, context)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "join path"))
			return nil, false
		}

		pv := value.NewAnnotatedValue(projectedItem)
		pv.SetAnnotations(av)
		av = pv
	}

	return av, fetchOk
}
