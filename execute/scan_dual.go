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
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type DualScan struct {
	base
	plan *plan.DualScan
}

func NewDualScan(plan *plan.DualScan) *DualScan {
	rv := &DualScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *DualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDualScan(this)
}

func (this *DualScan) Copy() Operator {
	return &DualScan{this.base.copy(), this.plan}
}

func (this *DualScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped

		for _, dual := range this.plan.Duals() {
			if !this.scanDual(context, parent, dual) {
				return
			}
		}
	})
}

func (this *DualScan) scanDual(context *Context, parent value.Value, dual *plan.Dual) bool {
	conn := catalog.NewIndexConnection(
		context.WarningChannel(),
		context.ErrorChannel(),
	)

	defer func() { conn.StopChannel() <- false }() // Notify that I have stopped

	dv := &catalog.Dual{}
	var ok bool

	if dual.Equal == nil {
		context.ErrorChannel() <- err.NewError(nil, "No equality term for dual filter.")
		return false
	}

	dv.Equal, ok = eval(dual.Equal, context, parent)
	if !ok {
		return false
	}

	dv.Low, ok = eval(dual.Low, context, parent)
	if !ok {
		return false
	}

	dv.High, ok = eval(dual.High, context, parent)
	if !ok {
		return false
	}

	dv.Inclusion = dual.Inclusion
	go this.plan.Index().DualScan(dv, conn)

	var entry *catalog.IndexEntry

	for ok {
		select {
		case entry, ok = <-conn.EntryChannel():
			if ok {
				cv := value.NewCorrelatedValue(parent)
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
