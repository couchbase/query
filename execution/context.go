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
	"sync"
	"time"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Output interface {
	Result(item value.Value) bool
	CloseResults()
	Fatal(err errors.Error)
	Error(err errors.Error)
	Warning(wrn errors.Error)
}

type Context struct {
	datastore   datastore.Datastore
	systemstore datastore.Datastore
	namespace   string
	readonly    bool
	now         time.Time
	arguments   map[string]value.Value
	output      Output
	subplans    *subqueryMap
	subresults  *subqueryMap
}

func NewContext(datastore, systemstore datastore.Datastore, namespace string,
	readonly bool, arguments map[string]value.Value, output Output) *Context {
	return &Context{
		datastore:   datastore,
		systemstore: systemstore,
		namespace:   namespace,
		readonly:    readonly,
		now:         time.Now(),
		arguments:   arguments,
		output:      output,
		subplans:    newSubqueryMap(),
		subresults:  newSubqueryMap(),
	}
}

func (this *Context) Datastore() datastore.Datastore {
	return this.datastore
}

func (this *Context) Systemstore() datastore.Datastore {
	return this.systemstore
}

func (this *Context) Namespace() string {
	return this.namespace
}

func (this *Context) Readonly() bool {
	return this.readonly
}

func (this *Context) Now() time.Time {
	return this.now
}

func (this *Context) Argument(parameter string) (value.Value, bool) {
	val, ok := this.arguments[parameter]
	return val, ok
}

func (this *Context) Result(item value.Value) bool {
	return this.output.Result(item)
}

func (this *Context) CloseResults() {
	this.output.CloseResults()
}

func (this *Context) Error(err errors.Error) {
	this.output.Error(err)
}

func (this *Context) Fatal(err errors.Error) {
	this.output.Fatal(err)
}

func (this *Context) Warning(wrn errors.Error) {
	this.output.Warning(wrn)
}

func (this *Context) EvaluateSubquery(query *algebra.Select, parent value.Value) (value.Value, error) {
	subresult, ok := this.subresults.get(query)
	if ok {
		return subresult.(value.Value), nil
	}

	subplan, planFound := this.subplans.get(query)

	if !planFound {
		var err error
		subplan, err = plan.Build(query, this.datastore, this.systemstore, this.namespace, true)
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

	results := collect.Values()

	// Cache results
	if !planFound && !query.Subresult().IsCorrelated() {
		this.subresults.set(query, results)
	}

	return results, nil
}

// Synchronized map
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
