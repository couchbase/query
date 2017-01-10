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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type DistinctScan struct {
	base
	scan         Operator
	keys         map[string]bool
	childChannel StopChannel
}

func NewDistinctScan(scan Operator) *DistinctScan {
	rv := &DistinctScan{
		base:         newBase(),
		scan:         scan,
		childChannel: make(StopChannel, 1),
	}

	rv.output = rv
	return rv
}

func (this *DistinctScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDistinctScan(this)
}

func (this *DistinctScan) Copy() Operator {
	return &DistinctScan{
		base:         this.base.copy(),
		scan:         this.scan.Copy(),
		childChannel: make(StopChannel, 1),
	}
}

func (this *DistinctScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.keys = _STRING_BOOL_POOL.Get()
		defer func() {
			_STRING_BOOL_POOL.Put(this.keys)
			this.keys = nil
		}()

		this.scan.SetParent(this)
		go this.scan.RunOnce(context, parent)

		var item value.AnnotatedValue
		n := 1
		ok := true

	loop:
		for ok {
			select {
			case <-this.stopChannel:
				break loop
			default:
			}

			select {
			case item, ok = <-this.scan.ItemChannel():
				if ok {
					ok = this.processKey(item, context)
				}
			case <-this.childChannel:
				n--
			case <-this.stopChannel:
				break loop
			}
		}

		// Await child scan
		if n > 0 {
			notifyChildren(this.scan)
			<-this.childChannel
		}
	})
}

func (this *DistinctScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *DistinctScan) processKey(item value.AnnotatedValue, context *Context) bool {
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
	item.SetBit(this.bit)
	return this.sendItem(item)
}

var _STRING_BOOL_POOL = util.NewStringBoolPool(1024)
