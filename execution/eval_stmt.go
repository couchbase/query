//  Copyright 2019-Present Couchbase, Inc.
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/rewrite"
	"github.com/couchbase/query/sanitizer"
	"github.com/couchbase/query/semantics"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const _INITIAL_SIZE = 64

type internalOutput struct {
	mutationCount uint64
	err           errors.Error
	abort         bool
	output        Output
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
	if this.err == nil {
		this.err = err
	}
	this.abort = true
}

func (this *internalOutput) Fatal(err errors.Error) {
	if this.err == nil {
		this.err = err
	}
	this.abort = true
}

func (this *internalOutput) Error(err errors.Error) {
	if this.err == nil {
		this.err = err
	}
}

func (this *internalOutput) SetErrors(errs errors.Errors) {
	for _, err := range errs {
		this.Error(err)
	}
}

func (this *internalOutput) Warning(wrn errors.Error) {
	// empty
}

func (this *internalOutput) Errors() []errors.Error {
	if this.err == nil {
		return nil
	}
	return []errors.Error{this.err}
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
	this.output.AddPhaseCount(p, c)
}

func (this *internalOutput) AddPhaseOperator(p Phases) {
	this.output.AddPhaseOperator(p)
}

func (this *internalOutput) PhaseOperator(p Phases) uint64 {
	return uint64(0)
}

func (this *internalOutput) FmtPhaseCounts() map[string]interface{} {
	// never used
	return nil
}

func (this *internalOutput) FmtPhaseOperators() map[string]interface{} {
	// never used
	return nil
}

func (this *internalOutput) AddPhaseTime(phase Phases, duration time.Duration) {
	this.output.AddPhaseTime(phase, duration)
}

func (this *internalOutput) FmtPhaseTimes(s util.DurationStyle) map[string]interface{} {
	// never used
	return nil
}

func (this *internalOutput) RawPhaseTimes() map[string]interface{} {
	return nil
}

func (this *internalOutput) FmtOptimizerEstimates(op Operator) map[string]interface{} {
	// never used
	return nil
}

func (this *internalOutput) TrackMemory(size uint64) {
	this.output.TrackMemory(size)
}

func (this *internalOutput) SetTransactionStartTime(t time.Time) {
	// empty
}

func (this *internalOutput) AddTenantUnits(s tenant.Service, ru tenant.Unit) {
	// empty
}

func (this *internalOutput) AddCpuTime(d time.Duration) {
	this.output.AddCpuTime(d)
}

func (this *internalOutput) AddIoTime(d time.Duration) {
	this.output.AddIoTime(d)
}

func (this *internalOutput) AddWaitTime(d time.Duration) {
	this.output.AddWaitTime(d)
}

func (this *internalOutput) Loga(l logging.Level, f func() string) {
	this.output.Loga(l, f)
}

func (this *internalOutput) LogLevel() logging.Level {
	return this.output.LogLevel()
}

func (this *Context) newOutput(output *internalOutput) *internalOutput {
	output.output = this.output
	return output
}

// If the opContext is not a dummy opContext this method stops the query executed by this method
// when the calling operator stops
func (this *opContext) ParkableEvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool, profileUdfExecTrees bool, funcKey string) (value.Value, uint64, error) {

	if this.HandlesInActive() {
		return nil, 0, errors.NewExecutionStatementStoppedError(statement)
	}

	newContext := this.Copy()
	txContext := this.TxContext()
	if txContext != nil {
		newContext.SetDeltaKeyspaces(this.DeltaKeyspaces())
		atrCollection, numAtrs := this.AtrCollection()
		newContext.SetTransactionContext("", false, txContext.TxTimeout(), txContext.TxTimeout(), atrCollection, numAtrs, []byte{})
	}
	stmt, prepared, isPrepared, err := newContext.PrepareStatement(statement, namedArgs, positionalArgs, subquery, readonly, false)
	if err != nil {
		return nil, 0, err
	}

	namedArgs, positionalArgs, err = newContext.handleUsing(stmt, namedArgs, positionalArgs)
	if err != nil {
		return nil, 0, err
	}
	stmtType := stmt.Type()
	if stmtType == "EXECUTE" && isPrepared {
		stmtType = prepared.Type()
	}

	newContext.SetStmtType(stmtType)
	err = this.handleOpenStatements(stmtType)
	if err != nil {
		return nil, 0, err
	}
	rv, mutations, err := newContext.ParkableExecutePrepared(prepared, isPrepared, namedArgs, positionalArgs,
		statement, profileUdfExecTrees, funcKey)
	newErr := newContext.completeStatement(stmtType, err == nil, this.Context)
	if err == nil && newErr != nil {
		err = newErr
	}
	return rv, mutations, err
}

func (this *Context) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool, profileUdfExecTrees bool, funcKey string) (value.Value, uint64, error) {

	// create a dummy opContext
	opContext := NewOpContext(this)
	return opContext.ParkableEvaluateStatement(statement, namedArgs, positionalArgs, subquery, readonly, profileUdfExecTrees,
		funcKey)
}

func (this *Context) completeStatement(stmtType string, success bool, baseContext *Context) errors.Error {
	newErr, txDone := this.DoStatementComplete(stmtType, success)
	if newErr != nil || txDone {
		baseContext.SetTxContext(nil)
		baseContext.output.SetTransactionStartTime(time.Time{})
		baseContext.SetDeltaKeyspaces(nil)
		return newErr
	}
	if this.TxContext() != nil && baseContext.TxContext() == nil {
		baseContext.SetTxContext(this.TxContext())
		baseContext.output.SetTransactionStartTime(this.TxContext().TxStartTime())
		baseContext.SetTxTimeout(this.TxContext().TxTimeout())
		baseContext.SetAtrCollection(this.AtrCollection())
		baseContext.SetDeltaKeyspaces(this.DeltaKeyspaces())
	}
	return nil
}

// If the opContext is not a dummy opContext this method stops the handle created by this method
// when the calling operator stops
func (this *opContext) ParkableOpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool, profileUdfExecTrees bool, funcKey string) (functions.Handle, error) {

	if this.HandlesInActive() {
		return nil, errors.NewExecutionStatementStoppedError(statement)
	}

	newContext := this.Copy()
	txContext := this.TxContext()
	if txContext != nil {
		newContext.SetDeltaKeyspaces(this.DeltaKeyspaces())
		atrCollection, numAtrs := this.AtrCollection()
		newContext.SetTransactionContext("", false, txContext.TxTimeout(), txContext.TxTimeout(), atrCollection, numAtrs, []byte{})
	}
	stmt, prepared, isPrepared, err := newContext.PrepareStatement(statement, namedArgs, positionalArgs, subquery, readonly, false)
	if err != nil {
		return nil, err
	}
	stmtType := stmt.Type()
	if stmtType == "EXECUTE" && isPrepared {
		stmtType = prepared.Type()
	}

	namedArgs, positionalArgs, err = newContext.handleUsing(stmt, namedArgs, positionalArgs)
	if err != nil {
		return nil, err
	}

	newContext.SetStmtType(stmtType)
	err = this.handleOpenStatements(stmtType)
	if err != nil {
		return nil, err
	}
	return newContext.OpenPrepared(this.Context, stmtType, prepared, isPrepared, namedArgs, positionalArgs,
		statement, profileUdfExecTrees, funcKey)
}

func (this *Context) OpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool, profileUdfExecTrees bool, funcKey string) (functions.Handle, error) {

	// create a dummy opContext
	opContext := NewOpContext(this)
	return opContext.ParkableOpenStatement(statement, namedArgs, positionalArgs, subquery, readonly, profileUdfExecTrees, funcKey)
}

func (this *Context) PrepareStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly, autoPrepare bool) (stmt algebra.Statement, prepared *plan.Prepared, isPrepared bool, rerr error) {

	if len(namedArgs) > 0 || len(positionalArgs) > 0 || subquery {
		autoPrepare = false
	}

	var name string
	var prepContext planner.PrepareContext
	var optimizer planner.Optimizer
	if this.optimizer != nil {
		optimizer = this.optimizer.Copy()
	}
	planner.NewPrepareContext(&prepContext, this.requestId, this.queryContext, namedArgs,
		positionalArgs, this.indexApiVersion, this.featureControls, this.useFts, this.useCBO, optimizer,
		this.deltaKeyspaces, this, false)

	if autoPrepare {
		name = prepareds.GetAutoPrepareName(statement, &prepContext)
		if name != "" {
			prepared = prepareds.GetAutoPreparePlan(name, statement, this.namespace, &prepContext)
			if prepared != nil {
				if readonly && !prepared.Readonly() {
					return nil, nil, false, fmt.Errorf("not a readonly request")
				}
				return nil, prepared, true, nil
			}
			prepContext.SetIsPrepare()
		} else {
			autoPrepare = false
		}
	}

	stmt, err := n1ql.ParseStatement2(statement, this.namespace, this.queryContext, this)
	if err != nil {
		return nil, nil, false, err
	}

	var stype string
	var allow bool
	switch estmt := stmt.(type) {
	case *algebra.Explain:
		stype = estmt.Statement().Type()
		allow = true
	case *algebra.Advise:
		stype = estmt.Statement().Type()
		allow = true
	case *algebra.Prepare:
		stype = stmt.Type()
		allow = true
	default:
		stype = stmt.Type()
		allow = false
	}

	txId := ""
	txImplicit := false
	if this.TxContext() != nil {
		txId = this.TxContext().TxId()
		txImplicit = this.TxContext().TxImplicit()
	}
	if ok, msg := transactions.IsValidStatement(txId, stype, txImplicit, allow); !ok {
		return nil, nil, false, errors.NewTranStatementNotSupportedError(stype, msg)
	}

	//  monitoring code TBD
	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, nil, false, errors.NewRewriteError(err, "")
	}

	semChecker := semantics.GetSemChecker(stmt.Type(), this.TxContext() != nil)
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return nil, nil, false, err
	}

	isPrepare := false // if the statement is a PREPARE statement
	switch st := stmt.(type) {
	case *algebra.Prepare:
		prepContext.SetNamedArgs(nil)
		prepContext.SetPositionalArgs(nil)
		prepContext.SetIsPrepare()
		isPrepare = true
	case *algebra.Advise:
		st.SetContext(NewOpContext(this))
	}

	//  monitoring code TBD
	prepared, err, _ = planner.BuildPrepared(stmt, this.datastore, this.systemstore, this.namespace, subquery, true,
		&prepContext)
	if err != nil {
		return nil, nil, false, err
	}

	if prepared == nil {
		return nil, nil, false, fmt.Errorf("failed to build a plan")
	}

	if readonly && !prepared.Readonly() {
		return nil, nil, false, fmt.Errorf("not a readonly request")
	}

	// find the time the plan was generated if the statement is a PREPARE statement or auto_prepare is true
	// So that query plans of prepared statements can have their creation time set
	var prep time.Time
	if autoPrepare || isPrepare {
		prep = time.Now()
		this.SetPlanPreparedTime(prep)
	}

	// EXECUTE doesn't get a plan. Get the plan from the cache.
	isPrepared = false
	switch stmt.Type() {
	case "EXECUTE":
		var reprepTime time.Duration

		exec, _ := stmt.(*algebra.Execute)
		prepared, err = prepareds.GetPreparedWithContext(exec.Prepared(), this.queryContext,
			this.deltaKeyspaces, prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY,
			&reprepTime, this)
		//  monitoring code TBD
		if err != nil {
			return nil, prepared, isPrepared, err
		}

		if ok, msg := transactions.IsValidStatement(txId, prepared.Type(), txImplicit, false); !ok {
			return nil, nil, false, errors.NewTranStatementNotSupportedError(stype, msg)
		}
		isPrepared = true

	default:
		// even though this is not a prepared statement, add the
		// text for the benefit of context.Recover(): we can
		// output the text in case of crashes
		prepared.SetText(statement)
		if autoPrepare {
			prepared.SetName(name)
			prepared.SetIndexApiVersion(this.indexApiVersion)
			prepared.SetFeatureControls(this.featureControls)
			prepared.SetNamespace(this.namespace)
			prepared.SetQueryContext(this.queryContext)
			prepared.SetUseFts(this.useFts)
			prepared.SetPreparedTime(prep) // set the time the plan was generated
			prepareds.AddAutoPreparePlan(stmt, prepared)
		}

	}

	return stmt, prepared, isPrepared, nil
}

// handle using
func (this *Context) handleUsing(stmt algebra.Statement, namedArgs map[string]value.Value, positionalArgs value.Values) (
	map[string]value.Value, value.Values, errors.Error) {

	exec, ok := stmt.(*algebra.Execute)
	if !ok {
		return namedArgs, positionalArgs, nil
	}
	using := exec.Using()
	if using != nil {
		if namedArgs != nil || positionalArgs != nil {
			return namedArgs, positionalArgs,
				errors.NewExecutionParameterError("cannot have both USING clause and request parameters")
		}
		argsValue := using.Value()
		if argsValue == nil {
			return namedArgs, positionalArgs, errors.NewExecutionParameterError("USING clause does not evaluate to static values")
		}

		actualValue := argsValue.Actual()
		switch actualValue := actualValue.(type) {
		case map[string]interface{}:
			newArgs := make(map[string]value.Value, len(actualValue))
			for n, v := range actualValue {
				newArgs[n] = value.NewValue(v)
			}
			namedArgs = newArgs
		case []interface{}:
			newArgs := make([]value.Value, len(actualValue))
			for n, v := range actualValue {
				newArgs[n] = value.NewValue(v)
			}
			positionalArgs = newArgs
		default:

			// this never happens, but for completeness
			return namedArgs, positionalArgs, errors.NewExecutionParameterError("unexpected value type")
		}
	}
	return namedArgs, positionalArgs, nil
}

// If the opContext is not a dummy opContext this method stops the query executed by this method
// when the calling operator stops
func (this *opContext) ParkableExecutePrepared(prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values, statement string, profileUdfExecTrees bool, funcKey string) (
	value.Value, uint64, error) {

	if this.HandlesInActive() {
		return nil, 0, errors.NewExecutionStatementStoppedError(statement)
	}

	var outputBuf internalOutput
	var results value.Value
	output := this.newOutput(&outputBuf)

	keep := this.output

	this.output = output
	this.SetIsPrepared(isPrepared)
	this.SetPrepared(prepared)
	this.namedArgs = namedArgs
	this.positionalArgs = positionalArgs

	// if the statement being executed here is:
	// 1. Not inside a JS UDF i.e function key is empty
	//    change the calling operator state to prevent the double accrual of timings
	//    both the calling operator & the operators executing this query
	// 2. Inside a JS UDF - i.e function key is not empty
	//    do not change the calling operator's state here
	//    the calling operator state must only be changed once the ENTIRE UDF finishes execution
	changeCallerState := funcKey == ""

	build := util.Now()

	// Collect statements results
	collect := NewCollect(plan.NewCollect(), this.Context)
	pipeline, used, err := Build2(prepared, this.Context, collect)
	keep.AddPhaseTime(INSTANTIATE, util.Since(build))

	if err != nil {
		this.output = keep
		return nil, 0, err
	}

	exec := util.Now()
	if used {

		this.Park(func(stop bool) {
			if stop {
				output.err = errors.NewExecutionStatementStoppedError(statement)
				pipeline.SendAction(_ACTION_STOP)
			} else {
				pipeline.SendAction(_ACTION_PAUSE)
			}
		}, changeCallerState)

		pipeline.RunOnce(this.Context, nil)

		// Await completion
		// If the root op implements a fork check - make a check. To avoid infinitely waiting
		if pOp, ok := pipeline.(interface{ HasForkedChild() bool }); (ok && pOp.HasForkedChild()) || !ok {
			collect.waitComplete()
			this.Resume(changeCallerState) // TODO resume wrt double profiling time
		}

		results = collect.ValuesOnce()

		if !profileUdfExecTrees {
			pipeline.Done()
		} else {
			// if saving exec tree for profiling - delay the cleanup
			// Once execution is complete - add the exec tree to the cache
			this.udfStmtExecTrees.set(funcKey, statement, pipeline, collect)
		}

	} else {
		sequence := NewSequence(plan.NewSequence(), this.Context, pipeline, collect)

		this.Park(func(stop bool) {
			if stop {
				output.err = errors.NewExecutionStatementStoppedError(statement)
				sequence.SendAction(_ACTION_STOP)
			} else {
				sequence.SendAction(_ACTION_PAUSE)
			}
		}, changeCallerState)

		sequence.RunOnce(this.Context, nil)

		// Await completion
		collect.waitComplete()

		this.Resume(changeCallerState)

		results = collect.ValuesOnce()

		if !profileUdfExecTrees {
			sequence.Done()
		} else {
			// if saving exec tree for profiling - delay the cleanup
			// Once execution is complete - add the exec tree to the cache
			this.udfStmtExecTrees.set(funcKey, statement, sequence, collect)
		}
	}
	this.output = keep
	this.output.AddPhaseTime(RUN, util.Since(exec))

	return results, output.mutationCount, output.err
}

func (this *Context) ExecutePrepared(prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values, statement string, profileUdfExecTrees bool, funcKey string) (
	value.Value, uint64, error) {

	opContext := NewOpContext(this)
	return opContext.ParkableExecutePrepared(prepared, isPrepared, namedArgs, positionalArgs, statement, profileUdfExecTrees,
		funcKey)
}

// If the opContext is not a dummy opContext this method stops the handle created by this method
// when the calling operator stops
func (this *opContext) OpenPrepared(baseContext *Context, stmtType string, prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values, statement string, profileUdfExecTrees bool, funcKey string) (
	functions.Handle, error) {

	if this.HandlesInActive() {
		return nil, errors.NewExecutionStatementStoppedError(statement)
	}

	handle := &executionHandle{}
	handle.statement = statement
	handle.udfKey = funcKey
	handle.profileHandleTree = profileUdfExecTrees
	handle.opContext = this
	handle.output = this.newOutput(&internalOutput{})
	handle.opContext.output = handle.output

	handle.opContext.SetIsPrepared(isPrepared)
	handle.opContext.SetPrepared(prepared)
	handle.opContext.namedArgs = namedArgs
	handle.opContext.positionalArgs = positionalArgs

	build := util.Now()

	// Collect statements results
	handle.input = NewReceive(plan.NewReceive(), handle.context())
	pipeline, used, err := Build2(prepared, this.Context, handle.input)
	this.output.AddPhaseTime(INSTANTIATE, util.Since(build))
	if err != nil {
		return nil, err
	}

	if used {
		handle.root = pipeline
	} else {
		handle.root = NewSequence(plan.NewSequence(), this.Context, pipeline, handle.input)
	}

	handle.stmtType = stmtType
	handle.actualType = prepared.Type()
	if handle.actualType == "" {
		handle.actualType = stmtType
	}
	handle.baseContext = baseContext
	baseContext.mutex.Lock()
	if baseContext.udfHandleMap == nil {
		baseContext.udfHandleMap = make(map[*executionHandle]bool)
	}
	baseContext.udfHandleMap[handle] = handle.actualType != "SELECT"
	baseContext.mutex.Unlock()

	// add this active handle to the opContext
	this.AddUdfHandle(handle)

	handle.exec = util.Now()
	handle.root.RunOnce(handle.context(), nil)
	return handle, nil
}

func (this *Context) handleOpenStatements(stmtType string) error {
	var newHandleMap map[*executionHandle]bool
	var err error

	// technically the lock is not needed as we will only ditch DMLs if the UDF is executed using EXECUTE FUNCTION
	// which means that there will ever only be one thread executing this loop, but still
	this.mutex.Lock()
	if len(this.udfHandleMap) == 0 {
		this.mutex.Unlock()
		return err
	}

	// for transaction statements, everything will be closed
	// for non transaction statements, only non SELECTS
	if stmtType == "START_TRANSACTION" || stmtType == "COMMIT" || stmtType == "ROLLBACK" {
		newHandleMap = this.udfHandleMap
		this.udfHandleMap = nil
	} else {
		newHandleMap = make(map[*executionHandle]bool, len(this.udfHandleMap))
		for k, v := range this.udfHandleMap {
			if v {
				newHandleMap[k] = v
				delete(this.udfHandleMap, k)
			}
		}
	}
	this.mutex.Unlock()

	// if we are asked to silently complete DML statements
	// we ignore errors and mutations
	for k, v := range newHandleMap {
		if v {
			_, newErr := k.Complete()
			if newErr != nil && err == nil {
				err = newErr
			}
		} else {
			k.Cancel()
		}
	}
	return err
}

type executionHandle struct {
	exec        util.Time
	root        Operator
	input       *Receive
	baseContext *Context
	stmtType    string
	actualType  string
	output      *internalOutput
	stopped     int32
	statement   string
	udfKey      string

	// Whether to save the tree for request profiling. currently only implemented for UDFs
	profileHandleTree bool

	// the calling operator's opContext.
	// opContext.Context is the the execution context used in the handle's query's execution
	opContext *opContext
}

// returns the execution context used in the handle's query execution
func (this *executionHandle) context() *Context {
	return this.opContext.Context
}

func (this *executionHandle) Results() (interface{}, uint64, error) {

	// if the calling operator has stopped - reject any interaction with any handle
	// note: this action is possible only when the opContext is not a dummy opContext
	if this.opContext.HandlesInActive() {
		this.externalStop()
		return nil, 0, this.output.err
	}

	if atomic.LoadInt32(&this.stopped) > 0 {
		return nil, 0, nil
	}
	values := make([]interface{}, 0, _INITIAL_SIZE)

	for {
		item, ok := this.input.getItem()
		if item != nil {
			if len(values) == cap(values) {
				newValues := make([]interface{}, len(values), len(values)<<1)
				copy(newValues, values)
				values = newValues
			}
			values = append(values, item)
		}
		if !ok {
			break
		}
	}

	// Close the handle
	this.Cancel()

	return values, this.output.mutationCount, this.output.err
}

func (this *executionHandle) Type() string {
	return this.actualType
}

func (this *executionHandle) Mutations() uint64 {
	return this.output.mutationCount
}

func (this *executionHandle) Complete() (uint64, error) {

	// if the calling operator has stopped - reject any interaction with any handle
	// note: this action is possible only when the opContext is not a dummy opContext
	if this.opContext.HandlesInActive() {
		this.externalStop()
		return 0, this.output.err
	}

	if atomic.LoadInt32(&this.stopped) > 0 {
		return 0, nil
	}
	for {
		item, ok := this.input.getItem()
		if item == nil || !ok {
			break
		}
	}

	// Close the handle
	this.Cancel()

	return this.output.mutationCount, this.output.err
}

func (this *executionHandle) NextDocument() (value.Value, error) {

	// if the calling operator has stopped - reject any interaction with any handle
	// note: this action is possible only when the opContext is not a dummy opContext
	if this.opContext.HandlesInActive() {
		this.externalStop()
		return nil, this.output.err
	}

	if !this.output.abort && this.stopped == 0 {
		item, _ := this.input.getItem()
		if item != nil {
			return item, nil
		}
	}

	// Close the handle
	this.Cancel()

	return nil, this.output.err
}

func (this *executionHandle) Cancel() {
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context().output.AddPhaseTime(RUN, util.Since(this.exec))
		this.root.SendAction(_ACTION_STOP)
		newErr := this.context().completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}

		// Once execution is complete - add the exec tree to the cache
		if this.profileHandleTree {
			this.baseContext.udfStmtExecTrees.set(this.udfKey, this.statement, this.root, this.input)
		}

		// Delete the now stopped handle from the opContext
		this.opContext.DeleteUdfHandle(this)

		this.baseContext.mutex.Lock()
		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
}

// Takes care of closing/ stopping the UDF handle
// Stops the execution of the handle's query
// adds the execution tree to the cache for profiling purposes
func (this *executionHandle) externalStop() {
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context().output.AddPhaseTime(RUN, util.Since(this.exec))

		// stop the execution of the handle's query
		this.root.SendAction(_ACTION_STOP)
		newErr := this.context().completeStatement(this.stmtType, this.output.err == nil, this.baseContext)

		//  add the execution tree to the cache for profiling purposes
		if this.profileHandleTree {
			this.baseContext.udfStmtExecTrees.set(this.udfKey, this.statement, this.root, this.input)
		}

		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		} else {
			this.output.err = errors.NewExecutionStatementStoppedError(this.statement)
		}

		this.baseContext.mutex.Lock()
		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
}

func (this *Context) executeTranStatementAtomicity(stmtType string) (map[string]bool, errors.Error) {
	if this.txContext == nil {
		return nil, nil
	}

	switch stmtType {
	case "START":
		return this.datastore.StartTransaction(true, this)
	case "COMMIT":
		return nil, this.datastore.CommitTransaction(true, this)
	case "ROLLBACK":
		return nil, this.datastore.RollbackTransaction(true, this, "")
	}

	return nil, errors.NewTransactionError(fmt.Errorf("Atomic Transaction: %s unknown statement", stmtType), "")

}

var implicitTranStmts = map[string]string{
	"START":    "START TRANSACTION",
	"COMMIT":   "COMMIT TRANSACTION",
	"ROLLBACK": "ROLLBACK TRANSACTION"}

// Used for implicit, explicit transactions
func (this *Context) ExecuteTranStatement(stmtType string, stmtAtomicity bool) (string, map[string]bool, errors.Error) {
	if stmtAtomicity {
		dks, err := this.executeTranStatementAtomicity(stmtType)
		return "", dks, err
	}

	var res value.Value
	var txId string
	stmt, ok := implicitTranStmts[stmtType]
	if !ok {
		return txId, nil, errors.NewTransactionError(fmt.Errorf("Implicit Transaction: %s unknown statement", stmtType), "")
	}

	newContext := this.Copy()
	newContext.queryContext = ""
	newContext.indexApiVersion = 0
	newContext.featureControls = 0
	newContext.useFts = false
	newContext.useCBO = false
	newContext.deltaKeyspaces = nil
	newContext.namedArgs = nil
	newContext.positionalArgs = nil

	_, prepared, isPrepared, err := newContext.PrepareStatement(stmt, nil, nil, false, false, true)
	if err == nil {
		res, _, err = newContext.ExecutePrepared(prepared, isPrepared, nil, nil, stmt, false, "")
	}
	if err != nil {
		error, ok := err.(errors.Error)
		if !ok {
			error = errors.NewTransactionError(err, "")
		}
		return "", nil, error
	}

	if stmtType == "START" {
		if actual, ok := res.Actual().([]interface{}); ok {
			if fields, ok := actual[0].(map[string]interface{}); ok {
				txId, _ = fields["txid"].(string)
			}
		}
		if txId == "" {
			return "", nil, errors.NewStartTransactionError(fmt.Errorf("Implicit Transaction"), nil)
		}
		this.consistency = newContext.TxContext().TxScanConsistency()
	}

	return txId, nil, nil
}

func (this *Context) DoStatementComplete(stmtType string, success bool) (err errors.Error, done bool) {
	done = false
	if this.txContext == nil {
		return
	}

	switch stmtType {
	case "SET_TRANSACTION_ISOLATION", "SAVEPOINT", "ROLLBACK_SAVEPOINT":
	case "START_TRANSACTION", "COMMIT", "ROLLBACK":
		if !success {
			_, _, err = this.ExecuteTranStatement("ROLLBACK", false)
		}
		if this.txContext != nil {
			if stmtType != "START_TRANSACTION" || !success {
				transactions.DeleteTransContext(this.txContext.TxId(), false)
				done = true
			}
		}

	default:
		tranStmt := "ROLLBACK"
		if success {
			tranStmt = "COMMIT"
		}

		_, _, err = this.ExecuteTranStatement(tranStmt, !this.txImplicit)
		if err != nil && tranStmt == "COMMIT" && this.txContext != nil {
			this.ExecuteTranStatement("ROLLBACK", !this.txImplicit)
		}

		if tranStmt == "ROLLBACK" || (err != nil &&
			err.Code() != errors.E_AMBIGUOUS_COMMIT_TRANSACTION &&
			err.Code() != errors.E_POST_COMMIT_TRANSACTION) {
			this.AddMutationCount(-this.MutationCount())
		}

		if this.txContext != nil {
			if this.txContext.TxImplicit() {
				transactions.DeleteTransContext(this.txContext.TxId(), false)
			}
		}
	}

	return
}

func (this *Context) Parse(s string) (interface{}, error) {
	rv, err := n1ql.ParseExpression(s)
	if err == nil {
		return rv, nil
	}
	rvs, err2 := n1ql.ParseStatement2(s, this.Namespace(), this.QueryContext())
	if err2 != nil {
		if !strings.Contains(err.Error(), "Input was not an expression") {
			return nil, err
		}
		return nil, err2
	}
	return rvs, nil
}

func (this *Context) Infer(v value.Value, with value.Value) (value.Value, error) {
	infer, err := this.Datastore().Inferencer(datastore.INF_DEFAULT)
	if err != nil {
		return nil, errors.NewInferencerNotFoundError(err, string(datastore.INF_DEFAULT))
	}

	expr := expression.NewConstant(v)
	conn := datastore.NewValueConnection(this)
	infer.InferExpression(this, expr, with, conn)

	item, ok := <-conn.ValueChannel()
	if item != nil && ok {
		val := item.(value.Value)
		return val, nil
	}
	return value.NULL_VALUE, nil
}

func (this *Context) InferKeyspace(ks interface{}, with value.Value) (value.Value, error) {

	var path *algebra.Path
	if ksref, ok := ks.(*algebra.KeyspaceRef); ok {
		path = ksref.Path()
	} else {
		return nil, errors.NewExecutionInternalError(fmt.Sprintf("Incorrect type assertion for variable ks:"+
			" Expected *algebra.KeyspaceRef got %T", ks))
	}

	keyspace, err := datastore.GetKeyspace(path.Parts()...)
	if err != nil {
		return nil, err
	}
	conn := datastore.NewValueConnection(this)
	infer, err := this.Datastore().Inferencer(datastore.INF_DEFAULT)

	infer.InferKeyspace(this, keyspace, with, conn)

	item, ok := <-conn.ValueChannel()
	if item != nil && ok {
		val := item.(value.Value)
		return val, nil
	}

	return value.NULL_VALUE, nil
}

func (this *internalOutput) GetErrorLimit() int {
	return this.output.GetErrorLimit()
}

func (this *internalOutput) GetErrorCount() int {
	return this.output.GetErrorCount()
}

func (this *opContext) PrepareStatementExt(statement string) (interface{}, error) {
	_, prepared, _, err := this.PrepareStatement(statement, nil, nil, false, true, false)
	return prepared, err
}

func (this *opContext) ExecutePreparedExt(prepared interface{}, namedArgs map[string]value.Value, positionalArgs value.Values) (
	value.Value, uint64, error) {

	orgPrepared, ok := prepared.(*plan.Prepared)

	if !ok {
		return nil, 0, errors.NewExecutionPanicError(nil, "casting prepared interface to plan.prepared failed")
	}

	return this.ParkableExecutePrepared(orgPrepared, false, namedArgs, positionalArgs, orgPrepared.Text(), false, "")
}

// Returns the Query Plan for a given query statement
// Akin to EXPLAIN :
//
// 		EXPLAIN, EXECUTE, ADVISE statements are not explained
// 		Statements being explained are not cached in prepareds when auto_prepare = true
// 	 	If inside a Tx - check if the statement being explained is valid inside a Tx
//
// For PREPARE stmts - the query plan of the actual stmt being prepared is returned
// Returns:
// 	if statement can be explained
// 	algebra root node
// 	query plan
// 	error

func (this *Context) ExplainStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery bool) (bool, algebra.Statement, *plan.QueryPlan, error) {

	var prepContext planner.PrepareContext

	planner.NewPrepareContext(&prepContext, this.requestId, this.queryContext, namedArgs, positionalArgs, this.indexApiVersion,
		this.featureControls, this.useFts, this.useCBO, this.optimizer, this.deltaKeyspaces, this, false)

	stmt, err := n1ql.ParseStatement2(statement, this.namespace, this.queryContext, this)

	if err != nil {
		return false, nil, nil, err
	}

	var stype string
	var stmtBuild algebra.Statement

	stmtBuild = nil     // The statement to create the plan on
	canExplain := false // If the statement can be Explained. ( ADVISE, EXECUTE, EXPLAIN statements cannot be explained )

	switch estmt := stmt.(type) {
	case *algebra.Explain:
		stype = estmt.Statement().Type()
	case *algebra.Advise:
		stype = estmt.Statement().Type()
	case *algebra.Prepare:
		stype = stmt.Type()
		stmtBuild = estmt.Statement()
		canExplain = true // explain the statement being prepared instead
	case *algebra.Execute:
		stype = stmt.Type()
	default:
		stype = stmt.Type()
		stmtBuild = stmt
		canExplain = true
	}

	if !canExplain {
		return false, stmt, nil, nil
	}

	// Check if the statement is valid inside a Tx
	txId := ""
	txImplicit := false
	if this.TxContext() != nil {
		txId = this.TxContext().TxId()
		txImplicit = this.TxContext().TxImplicit()
	}
	if ok, msg := transactions.IsValidStatement(txId, stype, txImplicit, true); !ok {
		return false, nil, nil, errors.NewTranStatementNotSupportedError(stype, msg)
	}

	//  monitoring code TBD
	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return false, nil, nil, errors.NewRewriteError(err, "")
	}

	// Semantic Checks
	semChecker := semantics.GetSemChecker(stmt.Type(), this.TxContext() != nil)
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return false, nil, nil, err
	}

	// Set ForceSQBuild = true - so that subquery plans are built as well
	qp, _, err, _ := planner.Build(stmtBuild, this.datastore, this.systemstore, this.namespace, subquery, true, true, &prepContext)

	if err != nil {
		return false, nil, nil, err
	}

	if qp == nil {
		return false, nil, nil, fmt.Errorf("Failed to build query plan")
	}

	return canExplain, stmt, qp, nil
}

func (this *Context) SanitizeStatement(statement string) (string, value.Value, error) {
	return sanitizer.SanitizeStatement(statement, this.namespace, this.queryContext, this.TxContext() != nil, this)
}
