//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type RangeScan struct {
	base
	plan *plan.RangeScan
}

func NewRangeScan(plan *plan.RangeScan) *RangeScan {
	rv := &RangeScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *RangeScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRangeScan(this)
}

func (this *RangeScan) Copy() Operator {
	return &RangeScan{this.base.copy(), this.plan}
}

func (this *RangeScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		for _, ranje := range this.plan.Ranges() {
			if !this.scanRange(context, parent, ranje) {
				return
			}
		}
	})
}

func (this *RangeScan) scanRange(context *Context, parent value.Value, ranje *plan.Range) bool {
	conn := catalog.NewIndexConnection(
		context.WarningChannel(),
		context.ErrorChannel(),
	)

	defer notifyConn(conn) // Notify index that I have stopped

	rv := &catalog.Range{}
	var ok bool

	rv.Low, ok = eval(ranje.Low, context, parent)
	if !ok {
		return false
	}

	rv.High, ok = eval(ranje.High, context, parent)
	if !ok {
		return false
	}

	rv.Inclusion = ranje.Inclusion
	go this.plan.Index().RangeScan(rv, conn)

	var entry *catalog.IndexEntry

	for ok {
		select {
		case entry, ok = <-conn.EntryChannel():
			if ok {
				cv := value.NewCorrelatedValue(make(map[string]interface{}), parent)
				av := value.NewAnnotatedValue(cv)
				av.SetAttachment("meta", map[string]interface{}{"id": entry.PrimaryKey})
				ok = this.sendItem(av)
			}
		case <-this.stopChannel:
			return false
		}
	}

	return true
}
