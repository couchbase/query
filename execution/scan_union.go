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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type UnionScan struct {
	base
	plan         *plan.UnionScan
	scans        []Operator
	keys         map[string]bool
	childChannel StopChannel
}

func NewUnionScan(plan *plan.UnionScan, context *Context, scans []Operator) *UnionScan {
	rv := &UnionScan{
		plan:         plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *UnionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionScan(this)
}

func (this *UnionScan) Copy() Operator {
	scans := _INDEX_SCAN_POOL.Get()

	for i, s := range this.scans {
		scans[i] = s.Copy()
	}

	rv := &UnionScan{
		plan:         this.plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *UnionScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		active := this.active()
		defer this.inactive() // signal that resources can be freed
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		defer func() {
			this.keys = nil
		}()

		if !active || !context.assert(len(this.scans) != 0, "Union Scan has no scans") {
			return
		}
		pipelineCap := int(context.GetPipelineCap())
		if pipelineCap <= _STRING_BOOL_POOL.Size() {
			this.keys = _STRING_BOOL_POOL.Get()
			defer func() {
				_STRING_BOOL_POOL.Put(this.keys)
			}()
		} else {
			this.keys = make(map[string]bool, pipelineCap)
		}

		channel := NewChannel(context)
		defer channel.Close()

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(channel)
			go scan.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		limit := evalLimitOffset(this.plan.Limit(), nil, int64(-1), this.plan.Covering(), context)
		offset := evalLimitOffset(this.plan.Offset(), nil, int64(0), this.plan.Covering(), context)
		n := len(this.scans)
		ok := true

	loop:
		for ok {
			this.switchPhase(_CHANTIME)
			select {
			case <-this.stopChannel:
				break loop
			default:
			}

			select {
			case item, ok = <-channel.ItemChannel():
				this.switchPhase(_EXECTIME)
				if ok {
					this.addInDocs(1)
					ok = this.processKey(item, context, limit, offset)
				}
			case <-this.childChannel:
				n--
			case <-this.stopChannel:
				break loop
			default:
				if n == 0 {
					break loop
				}
			}
		}

		// Await children
		this.switchPhase(_CHANTIME)
		notifyChildren(this.scans...)
		for ; n > 0; n-- {
			<-this.childChannel
		}
	})
}

func (this *UnionScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *UnionScan) processKey(item value.AnnotatedValue,
	context *Context, limit, offset int64) bool {

	m := item.GetAttachment("meta")
	meta, ok := m.(map[string]interface{})
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Missing or invalid meta %v of type %T.", m, m)))
		return false
	}

	k := meta["id"]
	key, ok := k.(string)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Missing or invalid primary key %v of type %T.", k, k)))
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

func (this *UnionScan) SendStop() {
	this.baseSendStop()
	for _, scan := range this.scans {
		scan.SendStop()
	}
}

func (this *UnionScan) reopen(context *Context) {
	this.baseReopen(context)
	this.childChannel = make(StopChannel, len(this.scans))
	for _, scan := range this.scans {
		scan.reopen(context)
	}
}

func (this *UnionScan) Done() {
	this.wait()
	for s, scan := range this.scans {
		scan.Done()
		this.scans[s] = nil
	}
	_INDEX_SCAN_POOL.Put(this.scans)
	this.scans = nil
}
