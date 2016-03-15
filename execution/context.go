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
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type Phases int

const (
	FETCH = Phases(iota)
	INDEX_SCAN
	PRIMARY_SCAN
	SORT
	PHASES // Sizer
)

func (phase Phases) String() string {
	return _PHASE_NAMES[phase]
}

var _PHASE_NAMES = []string{
	FETCH:        "Fetch",
	INDEX_SCAN:   "IndexScan",
	PRIMARY_SCAN: "PrimaryScan",
	SORT:         "Sort",
}

const _PHASE_UPDATE_COUNT uint64 = 100

type Output interface {
	Result(item value.Value) bool
	CloseResults()
	Fatal(err errors.Error)
	Error(err errors.Error)
	Warning(wrn errors.Error)
	AddMutationCount(uint64)
	MutationCount() uint64
	SortCount() uint64
	SetSortCount(i uint64)
	AddPhaseOperator(p Phases)
	AddPhaseCount(p Phases, c uint64)
	FmtPhaseCounts() map[string]interface{}
	FmtPhaseOperators() map[string]interface{}
	AddPhaseTime(phase string, duration time.Duration)
	PhaseTimes() map[string]time.Duration
	FmtPhaseTimes() map[string]interface{}
}

type Context struct {
	requestId        string
	datastore        datastore.Datastore
	systemstore      datastore.Datastore
	namespace        string
	readonly         bool
	maxParallelism   int
	now              time.Time
	namedArgs        map[string]value.Value
	positionalArgs   value.Values
	credentials      datastore.Credentials
	consistency      datastore.ScanConsistency
	scanVectorSource timestamp.ScanVectorSource
	output           Output
	subplans         *subqueryMap
	subresults       *subqueryMap
	mutex            sync.RWMutex
}

func NewContext(requestId string, datastore, systemstore datastore.Datastore,
	namespace string, readonly bool, maxParallelism int, namedArgs map[string]value.Value,
	positionalArgs value.Values, credentials datastore.Credentials,
	consistency datastore.ScanConsistency, scanVectorSource timestamp.ScanVectorSource, output Output) *Context {
	rv := &Context{
		requestId:        requestId,
		datastore:        datastore,
		systemstore:      systemstore,
		namespace:        namespace,
		readonly:         readonly,
		maxParallelism:   maxParallelism,
		now:              time.Now(),
		namedArgs:        namedArgs,
		positionalArgs:   positionalArgs,
		credentials:      credentials,
		consistency:      consistency,
		scanVectorSource: scanVectorSource,
		output:           output,
		subplans:         nil,
		subresults:       nil,
	}

	if rv.maxParallelism <= 0 || rv.maxParallelism > runtime.NumCPU() {
		rv.maxParallelism = runtime.NumCPU()
	}

	return rv
}

func (this *Context) RequestId() string {
	return this.requestId
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

func (this *Context) MaxParallelism() int {
	return this.maxParallelism
}

func (this *Context) Now() time.Time {
	return this.now
}

func (this *Context) NamedArg(name string) (value.Value, bool) {
	val, ok := this.namedArgs[name]
	return val, ok
}

// The position is 1-based (i.e. 1 is the first position)
func (this *Context) PositionalArg(position int) (value.Value, bool) {
	position--

	if position >= 0 && position < len(this.positionalArgs) {
		return this.positionalArgs[position], true
	} else {
		return nil, false
	}
}

func (this *Context) Credentials() datastore.Credentials {
	return this.credentials
}

func (this *Context) ScanConsistency() datastore.ScanConsistency {
	return this.consistency
}

func (this *Context) ScanVectorSource() timestamp.ScanVectorSource {
	return this.scanVectorSource
}

func (this *Context) AddMutationCount(i uint64) {
	this.output.AddMutationCount(i)
}

func (this *Context) MutationCount() uint64 {
	return this.output.MutationCount()
}

func (this *Context) SetSortCount(i uint64) {
	this.output.SetSortCount(i)
}

func (this *Context) SortCount() uint64 {
	return this.output.SortCount()
}

func (this *Context) AddPhaseOperator(p Phases) {
	this.output.AddPhaseOperator(p)
}

func (this *Context) AddPhaseCount(p Phases, c uint64) {
	this.output.AddPhaseCount(p, c)
}

func (this *Context) AddPhaseTime(phase string, duration time.Duration) {
	this.output.AddPhaseTime(phase, duration)
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
	subresults := this.getSubresults()
	subresult, ok := subresults.get(query)
	if ok {
		return subresult.(value.Value), nil
	}

	subplans := this.getSubplans()
	subplan, planFound := subplans.get(query)

	if !planFound {
		var err error
		subplan, err = planner.Build(query, this.datastore, this.systemstore, this.namespace, true)
		if err != nil {
			return nil, err
		}

		// Cache plan
		subplans.set(query, subplan)
	}

	pipeline, err := Build(subplan.(plan.Operator), this)
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

	results := collect.ValuesOnce()

	// Cache results
	if !planFound && !query.IsCorrelated() {
		subresults.set(query, results)
	}

	return results, nil
}

func (this *Context) getSubplans() *subqueryMap {
	if this.contextSubplans() == nil {
		this.initSubplans()
	}
	return this.contextSubplans()
}

func (this *Context) initSubplans() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.subplans == nil {
		this.subplans = newSubqueryMap()
	}
}

func (this *Context) contextSubplans() *subqueryMap {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.subplans
}

func (this *Context) getSubresults() *subqueryMap {
	if this.contextSubresults() == nil {
		this.initSubresults()
	}
	return this.contextSubresults()
}

func (this *Context) contextSubresults() *subqueryMap {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.subresults
}

func (this *Context) initSubresults() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.subresults == nil {
		this.subresults = newSubqueryMap()
	}
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

func (this *Context) Recover() {
	err := recover()
	if err != nil {
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, false)
		s := string(buf[0:n])
		logging.Severep("", logging.Pair{"panic", err},
			logging.Pair{"stack", s})
		os.Stderr.WriteString(s)
		os.Stderr.Sync()

		switch err := err.(type) {
		case error:
			this.Fatal(errors.NewError(err, fmt.Sprintf("Panic: %v", err)))
		default:
			this.Fatal(errors.NewError(nil, fmt.Sprintf("Panic: %v", err)))
		}
	}
}
