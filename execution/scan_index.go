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
	plan     *plan.IndexScan
	children []Operator
}

func NewIndexScan(plan *plan.IndexScan, context *Context) *IndexScan {
	rv := &IndexScan{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan(this)
}

func (this *IndexScan) Copy() Operator {
	rv := &IndexScan{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		spans := this.plan.Spans()
		n := len(spans)
		this.SetKeepAlive(n, context)
		this.setExecPhase(INDEX_SCAN, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time

		if !active || !context.assert(n != 0, "Index scan has no spans") {
			this.close(context)
			return
		}
		this.children = _INDEX_SCAN_POOL.Get()

		for i, span := range spans {
			scan := newSpanScan(this, span)
			scan.SetOutput(this.output)
			scan.SetBit(this.bit)
			this.children = append(this.children, scan)
			go this.children[i].RunOnce(context, parent)
		}
	})
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
		if child != nil {
			child.SendStop()
		}
	}
}

func (this *IndexScan) reopen(context *Context) {
	this.baseReopen(context)
	for _, child := range this.children {
		child.reopen(context)
	}
}

func (this *IndexScan) Done() {
	this.baseDone()
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
		plan: parent.plan,
		span: span,
	}

	newRedirectBase(&rv.base)
	rv.newStopChannel()
	rv.parent = parent
	rv.output = parent.output
	return rv
}

func (this *spanScan) Accept(visitor Visitor) (interface{}, error) {
	panic(fmt.Sprintf("Internal operator spanScan visited by %v.", visitor))
}

func (this *spanScan) Copy() Operator {
	rv := &spanScan{
		plan: this.plan,
		span: this.span,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *spanScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.addExecPhase(INDEX_SCAN, context)       // we have already added the scan operator in the primary scan
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped

		conn := datastore.NewIndexConnection(context)
		defer conn.Dispose()  // Dispose of the connection
		defer conn.SendStop() // Notify index that I have stopped

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
		// in the returned scope_value
		scope_value := parent
		if this.plan.Term().IsUnderNL() {
			scope_value = nil
		}

		for ok {
			entry, cont := this.getItemEntry(conn)
			if cont {
				if entry != nil {

					// current policy is to only count 'in' documents
					// from operators, not kv
					// add this.addInDocs(1) if this changes

					av := this.newEmptyDocumentWithKey(entry.PrimaryKey, scope_value, context)
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
				} else {
					ok = false
				}
			} else {
				return
			}
		}
	})
}

func (this *spanScan) scan(context *Context, conn *datastore.IndexConnection, parent value.Value) {
	defer context.Recover() // Recover from any panic

	// for nested-loop join we need to pass in values from left-hand-side (outer) of the join
	// for span evaluation
	outer_values := parent
	if !this.plan.Term().IsUnderNL() {
		outer_values = nil
	}
	dspan, empty, err := evalSpan(this.span, outer_values, context)
	if err != nil || empty {
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "span"))
		}
		conn.Sender().Close()
		return
	}

	limit := evalLimitOffset(this.plan.Limit(), nil, math.MaxInt64, this.plan.Covering(), context)

	keyspaceTerm := this.plan.Term()
	scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Keyspace())
	this.plan.Index().Scan(context.RequestId(), dspan, this.plan.Distinct(), limit,
		context.ScanConsistency(), scanVector, conn)
}

func evalSpan(ps *plan.Span, parent value.Value, context *Context) (*datastore.Span, bool, error) {
	var err error
	var empty bool

	ds := &datastore.Span{}

	ds.Seek, empty, err = eval(ps.Seek, context, nil)
	if err != nil || empty {
		return nil, empty, err
	}

	ds.Range.Low, empty, err = eval(ps.Range.Low, context, parent)
	if err != nil || empty {
		return nil, empty, err
	}

	ds.Range.High, empty, err = eval(ps.Range.High, context, parent)
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

// send a stop
func (this *spanScan) SendStop() {
	this.chanSendStop()
}
