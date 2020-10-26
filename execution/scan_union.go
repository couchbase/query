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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type UnionScan struct {
	base
	plan    *plan.UnionScan
	scans   []Operator
	keys    map[string]bool
	channel *Channel
}

func NewUnionScan(plan *plan.UnionScan, context *Context, scans []Operator) *UnionScan {
	rv := &UnionScan{
		plan:  plan,
		scans: scans,
	}

	newBase(&rv.base, context)
	rv.trackChildren(len(scans))
	rv.output = rv
	return rv
}

func (this *UnionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionScan(this)
}

func (this *UnionScan) Copy() Operator {
	scans := _INDEX_SCAN_POOL.Get()

	for _, s := range this.scans {
		scans = append(scans, s.Copy())
	}

	rv := &UnionScan{
		plan:  this.plan,
		scans: scans,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *UnionScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *UnionScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || !context.assert(len(this.scans) != 0, "Union Scan has no scans") {
			return
		}

		defer func() {
			this.keys = nil
		}()

		pipelineCap := int(context.GetPipelineCap())
		if pipelineCap <= _STRING_BOOL_POOL.Size() {
			this.keys = _STRING_BOOL_POOL.Get()
			defer func() {
				_STRING_BOOL_POOL.Put(this.keys)
			}()
		} else {
			this.keys = make(map[string]bool, pipelineCap)
		}

		this.channel = NewChannel(context)
		this.SetInput(this.channel)

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(this.channel)
			this.fork(scan, context, parent)
		}

		limit := evalLimitOffset(this.plan.Limit(), parent, int64(-1), this.plan.Covering(), context)
		offset := evalLimitOffset(this.plan.Offset(), parent, int64(0), this.plan.Covering(), context)
		n := len(this.scans)
		ok := true

	loop:
		for ok {
			item, child, cont := this.getItemChildren()
			if cont {
				if item != nil {
					this.addInDocs(1)
					ok = this.processKey(item, context, limit, offset)
				} else if child >= 0 {
					n--

					// now that no child is left behind, signal that there
					// is no further input coming in past what already queued
					if n == 0 {
						this.channel.close(context)
					}
				} else {
					break loop
				}
			} else {

				// stop children, wait and clean up
				if n > 0 {
					sendChildren(this.plan, this.scans...)
					this.childrenWaitNoStop(n)
					this.channel.close(context)
				}
				break loop
			}
		}

	})
}

func (this *UnionScan) processKey(item value.AnnotatedValue,
	context *Context, limit, offset int64) bool {

	key, ok := this.getDocumentKey(item, context)
	if !ok {
		return false
	}

	if _, ok = this.keys[key]; ok {
		return true
	}

	this.keys[key] = true

	length := int64(len(this.keys))
	if offset > 0 && length <= offset {
		return true
	}

	if limit > 0 && length > (limit+offset) {
		return false
	}

	item.SetBit(this.bit)
	return this.sendItem(item)
}

func (this *UnionScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["scans"] = this.scans
	})
	return json.Marshal(r)
}

func (this *UnionScan) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*UnionScan)
	childrenAccrueTimes(this.scans, copy.scans)
}

func (this *UnionScan) SendAction(action opAction) {
	if this.baseSendAction(action) {
		scans := this.scans
		for _, scan := range scans {
			if scan != nil {
				scan.SendAction(action)
			}
			if this.scans == nil {
				break
			}
		}
	}
}

func (this *UnionScan) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv {
		for _, scan := range this.scans {
			if !scan.reopen(context) {
				return false
			}
		}
	}
	return rv
}

func (this *UnionScan) Done() {
	this.baseDone()
	for s, scan := range this.scans {
		this.scans[s] = nil
		scan.Done()
	}
	_INDEX_SCAN_POOL.Put(this.scans)
	this.scans = nil
	channel := this.channel
	this.channel = nil
	if channel != nil {
		channel.Done()
	}
}
