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

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
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
	span *datastore.Span
}

func newSpanScan(parent *IndexScan, span *datastore.Span) *spanScan {
	rv := &spanScan{
		base: newChildBase(),
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
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		conn := datastore.NewIndexConnection(context)
		defer notifyConn(conn) // Notify index that I have stopped

		go this.plan.Index().Scan(this.span, this.plan.Distinct(), this.plan.Limit(), conn)

		var entry *datastore.IndexEntry

		ok := true
		for ok {
			select {
			case <-this.stopChannel:
				break
			default:
			}

			select {
			case entry, ok = <-conn.EntryChannel():
				if ok {
					cv := value.NewScopeValue(make(map[string]interface{}), parent)
					av := value.NewAnnotatedValue(cv)
					av.SetAttachment("meta", map[string]interface{}{"id": entry.PrimaryKey})
					ok = this.sendItem(av)
				}
			case <-this.stopChannel:
				break
			}
		}
	})
}
