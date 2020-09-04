//  Copyright (c) 2019 Couchbase, Inc.
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
	"sync/atomic"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/rewrite"
	"github.com/couchbase/query/semantics"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type internalOutput struct {
	mutationCount uint64
	err           errors.Error
}

func (this *internalOutput) SetUp() {
}

// we continue until we hit the first error
func (this *internalOutput) Result(item value.AnnotatedValue) bool {
	return (this.err == nil)
}

func (this *internalOutput) CloseResults() {
	// empty
}

func (this *internalOutput) Abort(err errors.Error) {
	// empty
}

func (this *internalOutput) Fatal(err errors.Error) {
	// empty
}

func (this *internalOutput) Error(err errors.Error) {
	if this.err == nil {
		this.err = err
	}
}

func (this *internalOutput) Warning(wrn errors.Error) {
	// empty
}

func (this *internalOutput) AddMutationCount(i uint64) {
	atomic.AddUint64(&this.mutationCount, i)
}

func (this *internalOutput) MutationCount() uint64 {
	return atomic.LoadUint64(&this.mutationCount)
}

func (this *internalOutput) SetSortCount(i uint64) {
	// empty
}

func (this *internalOutput) SortCount() uint64 {
	return uint64(0)
}

func (this *internalOutput) AddPhaseCount(p Phases, c uint64) {
	// empty
}

func (this *internalOutput) AddPhaseOperator(p Phases) {
	// empty
}

func (this *internalOutput) PhaseOperator(p Phases) uint64 {
	return uint64(0)
}

func (this *internalOutput) FmtPhaseCounts() map[string]interface{} {
	return nil
}

func (this *internalOutput) FmtPhaseOperators() map[string]interface{} {
	return nil
}

func (this *internalOutput) AddPhaseTime(phase Phases, duration time.Duration) {
	// empty
}

func (this *internalOutput) FmtPhaseTimes() map[string]interface{} {
	return nil
}

func (this *internalOutput) FmtOptimizerEstimates(op Operator) map[string]interface{} {
	return nil
}

func (this *internalOutput) TrackMemory(size uint64) {
	// empty
}

func (this *Context) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool) (value.Value, uint64, error) {
	var outputBuf internalOutput
	output := &outputBuf

	// TODO leaving profiling in place but commented out just in case
	// parse := util.Now()
	stmt, err := n1ql.ParseStatement2(statement, this.namespace, this.queryContext)
	// output.AddPhaseTime(PARSE, util.Since(parse))
	if err != nil {
		return nil, 0, err
	}

	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, 0, errors.NewRewriteError(err, "")
	}

	semChecker := semantics.NewSemChecker(true /* FIXME */, stmt.Type())
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return nil, 0, err
	}

	isprepare := false
	if _, ok := stmt.(*algebra.Prepare); ok {
		isprepare = true
	}

	if isprepare {
		namedArgs = nil
		positionalArgs = nil
	}

	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, this.requestId, this.queryContext, namedArgs,
		positionalArgs, this.indexApiVersion, this.featureControls, this.useFts, this.useCBO,
		this.optimizer)
	prepared, err := planner.BuildPrepared(stmt, this.datastore, this.systemstore, this.namespace, subquery, false,
		&prepContext)
	// output.AddPhaseTime(PLAN, util.Since(prep))
	if err != nil {
		return nil, 0, err
	}

	if prepared == nil {
		return nil, 0, fmt.Errorf("failed to build a plan")
	}

	if readonly && !prepared.Readonly() {
		return nil, 0, fmt.Errorf("not a readonly request")
	}

	// EXECUTE doesn't get a plan. Get the plan from the cache.
	isPrepared := false
	switch stmt.Type() {
	case "EXECUTE":
		var reprepTime time.Duration
		var err errors.Error

		exec, _ := stmt.(*algebra.Execute)
		prepared, err = prepareds.GetPreparedWithContext(exec.Prepared(), this.queryContext, prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY, &reprepTime)
		// if reprepTime > 0 {
		//        output.AddPhaseTime(REPREPARE, reprepTime)
		// }
		if err != nil {
			return nil, 0, err
		}
		isPrepared = true

	default:

		// even though this is not a prepared statement, add the
		// text for the benefit of context.Recover(): we can
		// output the text in case of crashes
		prepared.SetText(statement)
	}

	newContext := this.Copy()
	newContext.output = output
	newContext.SetPrepared(isPrepared)
	newContext.prepared = prepared
	newContext.namedArgs = namedArgs
	newContext.positionalArgs = positionalArgs

	pipeline, err := Build(prepared, newContext)
	if err != nil {
		return nil, 0, err
	}

	// Collect statements results
	// FIXME: this should handled by the planner
	collect := NewCollect(plan.NewCollect(), newContext)
	sequence := NewSequence(plan.NewSequence(), newContext, pipeline, collect)
	sequence.RunOnce(newContext, nil)

	// Await completion
	collect.waitComplete()

	results := collect.ValuesOnce()

	sequence.Done()

	return results, output.mutationCount, output.err
}

func (this *Context) EvaluatePrepared(prepared *plan.Prepared, isPrepared bool) (value.Value, uint64, error) {
	var outputBuf internalOutput
	output := &outputBuf

	newContext := this.Copy()
	newContext.output = output
	newContext.SetPrepared(isPrepared)
	newContext.prepared = prepared
	newContext.namedArgs = this.namedArgs
	newContext.positionalArgs = this.positionalArgs

	build := util.Now()
	pipeline, err := Build(prepared, newContext)
	this.output.AddPhaseTime(INSTANTIATE, util.Since(build))

	if err != nil {
		return nil, 0, err
	}

	// Collect statements results
	// FIXME: this should handled by the planner
	collect := NewCollect(plan.NewCollect(), newContext)
	sequence := NewSequence(plan.NewSequence(), newContext, pipeline, collect)

	exec := util.Now()
	sequence.RunOnce(newContext, nil)

	// Await completion
	collect.waitComplete()

	results := collect.ValuesOnce()

	sequence.Done()
	this.output.AddPhaseTime(RUN, util.Since(exec))

	return results, output.mutationCount, output.err
}
