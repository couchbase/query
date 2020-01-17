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
	_ "fmt"

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type DummyScan struct {
	base
	plan *plan.DummyScan
}

var _DUMMYSCAN_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_DUMMYSCAN_OP_POOL, func() interface{} {
		return &DummyScan{}
	})
}

func NewDummyScan(plan *plan.DummyScan, context *Context) *DummyScan {
	rv := _DUMMYSCAN_OP_POOL.Get().(*DummyScan)
	rv.plan = plan
	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}

func (this *DummyScan) Copy() Operator {
	rv := _DUMMYSCAN_OP_POOL.Get().(*DummyScan)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *DummyScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped
		if !active {
			return
		}

		av := value.EMPTY_ANNOTATED_OBJECT

		if parent != nil {
			cv := value.NewScopeValue(_EMPTY_OBJECT, parent)
			av = value.NewAnnotatedValue(cv)
		}

		this.sendItem(av)
	})
}

func (this *DummyScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *DummyScan) Done() {
	this.baseDone()
	if this.isComplete() {
		_DUMMYSCAN_OP_POOL.Put(this)
	}
}

var _EMPTY_OBJECT = map[string]interface{}{}
