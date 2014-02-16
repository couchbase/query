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

type PrimaryScan struct {
	base
	plan *plan.PrimaryScan
}

func NewPrimaryScan(plan *plan.PrimaryScan) *PrimaryScan {
	rv := &PrimaryScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) Copy() Operator {
	return &PrimaryScan{this.base.copy(), this.plan}
}

func (this *PrimaryScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.scanPrimary(context, parent)
	})
}

func (this *PrimaryScan) scanPrimary(context *Context, parent value.Value) {
	conn := catalog.NewIndexConnection(
		context.WarningChannel(),
		context.ErrorChannel(),
	)

	defer notifyConn(conn) // Notify index that I have stopped

	go this.plan.Index().PrimaryScan(conn)

	var entry *catalog.IndexEntry

	ok := true
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
			return
		}
	}
}
