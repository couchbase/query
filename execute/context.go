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
	"fmt"
	"sync"
	"time"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Context struct {
	now       time.Time
	arguments map[string]value.Value

	warnchan err.ErrorChannel
	errchan  err.ErrorChannel

	subplans *subplanMap
}

func NewContext() *Context {
	rv := &Context{}
	rv.now = time.Now()
	rv.arguments = make(map[string]value.Value)
	rv.warnchan = make(err.ErrorChannel)
	rv.errchan = make(err.ErrorChannel)
	rv.subplans = newSubplanMap()
	return rv
}

func (this *Context) Now() time.Time {
	return this.now
}

func (this *Context) Argument(parameter string) value.Value {
	val, ok := this.arguments[parameter]
	if !ok {
		panic(fmt.Sprintf("No argument value for parameter %s.", parameter))
	}

	return val
}

func (this *Context) Warnchan() err.ErrorChannel {
	return this.warnchan
}

func (this *Context) Errchan() err.ErrorChannel {
	return this.errchan
}

func (this *Context) EvaluateSubquery(query *algebra.Select, parent value.Value) (value.Value, error) {
	subplan, ok := this.subplans.get(query)

	if !ok {
		var err error
		subplan, err = plan.Plan(query)
		if err != nil {
			return nil, err
		}

		this.subplans.set(query, subplan)
	}

	pipeline, err := Build(subplan)
	if err != nil {
		return nil, err
	}

	pipeline.Run(this, parent)

	return nil, nil
}

// Synchronized access to subplans
type subplanMap struct {
	mutex    sync.RWMutex
	subplans map[*algebra.Select]plan.Operator
}

func newSubplanMap() *subplanMap {
	rv := &subplanMap{}
	rv.subplans = make(map[*algebra.Select]plan.Operator)
	return rv
}

func (this *subplanMap) get(key *algebra.Select) (plan.Operator, bool) {
	this.mutex.RLock()
	rv, ok := this.subplans[key]
	this.mutex.RUnlock()
	return rv, ok
}

func (this *subplanMap) set(key *algebra.Select, value plan.Operator) {
	this.mutex.Lock()
	this.subplans[key] = value
	this.mutex.Unlock()
}
