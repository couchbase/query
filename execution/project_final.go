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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type FinalProject struct {
	base
	plan *plan.FinalProject
}

var _FINALPROJ_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_FINALPROJ_OP_POOL, func() interface{} {
		return &FinalProject{}
	})
}

func NewFinalProject(plan *plan.FinalProject, context *Context) *FinalProject {
	rv := _FINALPROJ_OP_POOL.Get().(*FinalProject)
	rv.plan = plan
	newSerializedBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *FinalProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalProject(this)
}

func (this *FinalProject) Copy() Operator {
	rv := _FINALPROJ_OP_POOL.Get().(*FinalProject)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *FinalProject) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *FinalProject) processItem(item value.AnnotatedValue, context *Context) bool {
	pv := item.GetAttachment("projection")
	if pv != nil {
		v := pv.(value.Value)
		return this.sendItem(value.NewAnnotatedValue(v))
	}

	return this.sendItem(item)
}

func (this *FinalProject) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *FinalProject) Done() {
	this.baseDone()
	_FINALPROJ_OP_POOL.Put(this)
}
