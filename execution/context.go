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
	"sync"
	"time"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

const _BUFFER_CAP = 1024

// Context.Close() must be invoked to release resources.
type Context struct {
	now       time.Time
	arguments map[string]value.Value

	warningChannel errors.ErrorChannel // Never closed, just garbage-collected
	errorChannel   errors.ErrorChannel // Never closed, just garbage-collected
	warnings       []errors.Error
	errors         []errors.Error

	subplans   *subqueryMap
	subresults *subqueryMap
}

// Context.Close() must be invoked to release resources.
func NewContext() *Context {
	rv := &Context{}
	rv.now = time.Now()
	rv.arguments = make(map[string]value.Value)
	rv.warningChannel = make(errors.ErrorChannel, _BUFFER_CAP)
	rv.errorChannel = make(errors.ErrorChannel, _BUFFER_CAP)
	rv.warnings = make([]errors.Error, 0, _BUFFER_CAP)
	rv.errors = make([]errors.Error, 0, _BUFFER_CAP)
	rv.subplans = newSubqueryMap()
	rv.subresults = newSubqueryMap()

	go rv.drain(rv.warningChannel, &rv.warnings)
	go rv.drain(rv.errorChannel, &rv.errors)

	return rv
}

// Context.Close() must be invoked to release resources.
func (this *Context) Close() {
	this.warningChannel <- nil
	this.errorChannel <- nil
}

func (this *Context) Now() time.Time {
	return this.now
}

func (this *Context) Argument(parameter string) (value.Value, error) {
	val, ok := this.arguments[parameter]
	if !ok {
		return nil, fmt.Errorf("No argument value for parameter %s.", parameter)
	}

	return val, nil
}

func (this *Context) WarningChannel() errors.ErrorChannel {
	return this.warningChannel
}

func (this *Context) ErrorChannel() errors.ErrorChannel {
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

		// Cache plan
		this.subplans.set(query, subplan)
	}

	pipeline, err := Build(subplan.(plan.Operator))
	if err != nil {
		return nil, err
	}

	// Collect subquery results
	collect := NewCollect()
	sequence := NewSequence(pipeline, collect)
	sequence.RunOnce(this, parent)

	// Await completion
	ok = true
	for ok {
		_, ok = <-collect.Output().ItemChannel()
	}

	results := value.NewValue(collect.Values())

	// Cache results
	if !planFound && !query.Subresult().IsCorrelated() {
		this.subresults.set(query, results)
	}

	return results, nil
}

func (this *Context) Stream(item value.Value) bool {
	return true
}

func (this *Context) Warnings() []errors.Error {
	return this.warnings
}

func (this *Context) Errors() []errors.Error {
	return this.errors
}

func (this *Context) drain(channel errors.ErrorChannel, buf *[]errors.Error) {
	var e errors.Error
	for {
		e = <-channel

		if e == nil {
			return
		} else {
			collect(e, buf)
		}
	}
}

func collect(e errors.Error, buf *[]errors.Error) {
	if len(*buf) == cap(*buf) {
		b := make([]errors.Error, len(*buf), len(*buf)<<1)
		copy(b, *buf)
		*buf = b
	}

	*buf = append(*buf, e)
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
