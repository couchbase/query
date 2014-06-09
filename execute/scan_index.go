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
	"math"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type IndexScan struct {
	base
	plan *plan.IndexScan
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
	return &IndexScan{this.base.copy(), this.plan}
}

func (this *IndexScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		for _, span := range this.plan.Spans() {
			if !this.scanIndex(context, parent, span) {
				return
			}
		}
	})
}

func (this *IndexScan) scanIndex(context *Context, parent value.Value, span *catalog.Span) bool {
	conn := catalog.NewIndexConnection(
		context.WarningChannel(),
		context.ErrorChannel(),
	)

	defer notifyConn(conn) // Notify index that I have stopped

	go this.plan.Index().Scan(span, math.MaxInt64, conn)

	var entry *catalog.IndexEntry

	ok := true
	for ok {
		select {
		case entry, ok = <-conn.EntryChannel():
			if ok {
				cv := value.NewScopeValue(make(map[string]interface{}), parent)
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
