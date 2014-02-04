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

	warningChannel err.ErrorChannel
	errorChannel   err.ErrorChannel

	subplans   *subqueryMap
	subresults *subqueryMap
}

func NewContext() *Context {
	rv := &Context{}
	rv.now = time.Now()
	rv.arguments = make(map[string]value.Value)
	rv.warningChannel = make(err.ErrorChannel)
	rv.errorChannel = make(err.ErrorChannel)
	rv.subplans = newSubqueryMap()
	rv.subresults = newSubqueryMap()
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

func (this *Context) WarningChannel() err.ErrorChannel {
	return this.warningChannel
}

func (this *Context) ErrorChannel() err.ErrorChannel {
	return this.errorChannel
}

func (this *Context) EvaluateSubquery(query *algebra.Select, parent value.Value) (value.Value, error) {
	subresult, ok := this.subresults.get(query)
	if ok {
		return subresult.(value.Value), nil
	}

	subplan, planFound := this.subplans.get(query)

	if !planFound {
		var err error
		subplan, err = plan.Plan(query)
		if err != nil {
			return nil, err
		}

		this.subplans.set(query, subplan)
	}

	pipeline, err := Build(subplan.(plan.Operator))
	if err != nil {
		return nil, err
	}

	pipeline.RunOnce(this, parent)
	var results value.Value = nil // FIXME

	if !planFound && !query.IsCorrelated() {
		this.subresults.set(query, results)
	}

	return results, nil
}

// Mutex-synchronized map
type subqueryMap struct {
	mutex   sync.RWMutex
	entries map[*algebra.Select]interface{}
}

func newSubqueryMap() *subqueryMap {
	rv := &subqueryMap{}
	rv.entries = make(map[*algebra.Select]interface{})
	return rv
}

func (this *subqueryMap) get(key *algebra.Select) (interface{}, bool) {
	this.mutex.RLock()
	rv, ok := this.entries[key]
	this.mutex.RUnlock()
	return rv, ok
}

func (this *subqueryMap) set(key *algebra.Select, value interface{}) {
	this.mutex.Lock()
	this.entries[key] = value
	this.mutex.Unlock()
}
