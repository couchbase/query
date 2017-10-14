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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexScan struct {
	base
	plan         *plan.IndexScan
	children     []Operator
	childChannel StopChannel
}

func NewIndexScan(plan *plan.IndexScan, context *Context) *IndexScan {
	rv := &IndexScan{
		base: newBase(context),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan(this)
}

func (this *IndexScan) Copy() Operator {
	return &IndexScan{
		base: this.base.copy(),
		plan: this.plan,
	}
}

func (this *IndexScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		defer this.inactive() // signal that resources can be freed
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_SCAN, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer close(this.itemChannel)                // Broadcast that I have stopped
		defer this.notify()                          // Notify that I have stopped

		spans := this.plan.Spans()
		n := len(spans)
		if !active || !context.assert(n != 0, "Index scan has no spans") {
			return
		}
		this.childChannel = make(StopChannel, n)
		this.children = _INDEX_SCAN_POOL.Get()

		for i, span := range spans {
			scan := newSpanScan(this, span)
			scan.SetBit(this.bit)
			this.children = append(this.children, scan)
			go this.children[i].RunOnce(context, parent)
		}

		// a bit of an oversimplification, but...
		this.switchPhase(_CHANTIME)
		for n > 0 {
			select {
			case <-this.stopChannel:
				this.notifyStop()
				notifyChildren(this.children...)
			default:
			}

			select {
			case <-this.childChannel: // Never closed
				// Wait for all children
				n--
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				notifyChildren(this.children...)
			}
		}
	})
}

func (this *IndexScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *IndexScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	r["~children"] = this.children
	return json.Marshal(r)
}

func (this *IndexScan) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*IndexScan)
	childrenAccrueTimes(this.children, copy.children)
}

func (this *IndexScan) SendStop() {
	this.baseSendStop()
	for _, child := range this.children {
		child.SendStop()
	}
}

func (this *IndexScan) reopen(context *Context) {
	this.baseReopen(context)
	for _, child := range this.children {
		child.reopen(context)
	}
}

func (this *IndexScan) Done() {
	this.wait()
	for c, _ := range this.children {
		// we happen to know that there's nothing to be done for the chilren spans
		this.children[c] = nil
	}
	_INDEX_SCAN_POOL.Put(this.children)
	this.children = nil
}

type spanScan struct {
	base
	plan *plan.IndexScan
	span *plan.Span
}

func newSpanScan(parent *IndexScan, span *plan.Span) *spanScan {
	rv := &spanScan{
		base: newRedirectBase(),
		plan: parent.plan,
		span: span,
	}

	rv.parent = parent
	rv.output = parent.output
	return rv
}

func (this *spanScan) Accept(visitor Visitor) (interface{}, error) {
	panic(fmt.Sprintf("Internal operator spanScan visited by %v.", visitor))
}

func (this *spanScan) Copy() Operator {
	return &spanScan{
		base: this.base.copy(),
		plan: this.plan,
		span: this.span,
	}
}

func (this *spanScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		this.addExecPhase(INDEX_SCAN, context)       // we have already added the scan operator in the primary scan
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer close(this.itemChannel)                // Broadcast that I have stopped
		defer this.notify()                          // Notify that I have stopped

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
			this.switchPhase(_SERVTIME)
			select {
			case <-this.stopChannel:
				return
			default:
			}

			select {
			case entry, ok = <-conn.EntryChannel():
				this.switchPhase(_EXECTIME)
				if ok {

					// current policy is to only count 'in' documents
					// from operators, not kv
					// add this.addInDocs(1) if this changes
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

						// Matches planner.builder.buildCoveringScan()
						for i, ek := range entry.EntryKey {
							av.SetCover(covers[i].Text(), ek)
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

func (this *spanScan) scan(context *Context, conn *datastore.IndexConnection) {
	defer context.Recover() // Recover from any panic

	dspan, empty, err := evalSpan(this.span, context)
	if err != nil || empty {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		close(conn.EntryChannel())
		return
	}

	limit := evalLimitOffset(this.plan.Limit(), nil, math.MaxInt64, this.plan.Covering(), context)

	keyspaceTerm := this.plan.Term()
	scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Keyspace())
	this.plan.Index().Scan(context.RequestId(), dspan, this.plan.Distinct(), limit,
		context.ScanConsistency(), scanVector, conn)
}

func evalSpan(ps *plan.Span, context *Context) (*datastore.Span, bool, error) {
	var err error
	var empty bool
	ds := &datastore.Span{}

	ds.Seek, empty, err = eval(ps.Seek, context, nil)
	if err != nil || empty {
		return nil, empty, err
	}

	ds.Range.Low, empty, err = eval(ps.Range.Low, context, nil)
	if err != nil || empty {
		return nil, empty, err
	}

	ds.Range.High, empty, err = eval(ps.Range.High, context, nil)
	if err != nil || empty {
		return nil, empty, err
	}

	ds.Range.Inclusion = ps.Range.Inclusion
	return ds, empty, nil
}

func (this *spanScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
