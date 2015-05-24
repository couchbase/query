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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexScan struct {
	base
	plan         *plan.IndexScan
	childChannel StopChannel
}

func NewIndexScan(plan *plan.IndexScan) *IndexScan {
	rv := &IndexScan{
		base: newBase(),
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
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		spans := this.plan.Spans()
		n := len(spans)
		this.childChannel = make(StopChannel, n)
		children := make([]Operator, n)
		for i, span := range spans {
			children[i] = newSpanScan(this, span)
			go children[i].RunOnce(context, parent)
		}

		for n > 0 {
			select {
			case <-this.stopChannel:
				this.notifyStop()
				notifyChildren(children...)
			default:
			}

			select {
			case <-this.childChannel: // Never closed
				// Wait for all children
				n--
			case <-this.stopChannel: // Never closed
				this.notifyStop()
				notifyChildren(children...)
			}
		}
	})
}

func (this *IndexScan) ChildChannel() StopChannel {
	return this.childChannel
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
	return &spanScan{this.base.copy(), this.plan, this.span}
}

func (this *spanScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		conn := datastore.NewIndexConnection(context)
		defer notifyConn(conn) // Notify index that I have stopped

		var duration time.Duration
		timer := time.Now()
		defer context.AddPhaseTime("scan", time.Since(timer)-duration)

		go this.scan(context, conn)

		var entry *datastore.IndexEntry
		ok := true
		for ok {
			select {
			case <-this.stopChannel:
				return
			default:
			}

			select {
			case entry, ok = <-conn.EntryChannel():
				t := time.Now()

				if ok {
					cv := value.NewScopeValue(make(map[string]interface{}), parent)
					av := value.NewAnnotatedValue(cv)
					av.SetAttachment("meta", map[string]interface{}{"id": entry.PrimaryKey})
					ok = this.sendItem(av)
				}

				duration += time.Since(t)
			case <-this.stopChannel:
				return
			}
		}
	})
}

func (this *spanScan) scan(context *Context, conn *datastore.IndexConnection) {
	defer context.Recover() // Recover from any panic

	dspan, err := evalSpan(this.span, context)
	if err != nil {
		context.Error(errors.NewEvaluationError(err, "span"))
		close(conn.EntryChannel())
		return
	}

	this.plan.Index().Scan(dspan, this.plan.Distinct(), this.plan.Limit(),
		context.ScanConsistency(), context.ScanVector(), conn)
}

func evalSpan(ps *plan.Span, context *Context) (*datastore.Span, error) {
	var err error
	ds := &datastore.Span{}

	ds.Seek, err = evalExprs(ps.Seek, context)
	if err != nil {
		return nil, err
	}

	ds.Range.Low, err = evalExprs(ps.Range.Low, context)
	if err != nil {
		return nil, err
	}

	ds.Range.High, err = evalExprs(ps.Range.High, context)
	if err != nil {
		return nil, err
	}

	ds.Range.Inclusion = ps.Range.Inclusion
	return ds, nil
}

func evalExprs(exprs expression.Expressions, context *Context) (value.Values, error) {
	if exprs == nil {
		return nil, nil
	}

	values := make(value.Values, len(exprs))

	var err error
	for i, expr := range exprs {
		if expr == nil {
			continue
		}

		values[i], err = expr.Evaluate(nil, context)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
