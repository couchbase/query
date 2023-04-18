//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/rewrite"
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

func (this *internalOutput) SetErrors(err errors.Errors) {
	// empty
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

func (this *internalOutput) FmtPhaseTimes() map[string]interface{} {
	// never used
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

func (this *Context) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool, doCaching bool, funcKey string) (value.Value, uint64, error) {
	newContext := this.Copy()

	newContext.udfPlans = this.udfPlans
	newContext.udfStmtExecTrees = this.udfStmtExecTrees

	txContext := this.TxContext()
	if txContext != nil {
		newContext.SetDeltaKeyspaces(this.DeltaKeyspaces())
		atrCollection, numAtrs := this.AtrCollection()
		newContext.SetTransactionContext("", false, txContext.TxTimeout(), txContext.TxTimeout(), atrCollection, numAtrs, []byte{})
	}

	var stmt algebra.Statement
	var prepared *plan.Prepared
	var isPrepared bool
	var err error

	doCaching = doCaching && (this.udfPlans != nil)
	var cacheKey string
	createPlan := true

	if doCaching {
		cacheKey = encodeStatement(this.queryContext, statement)
		if pe, ok := this.udfPlans.getAndValidate(cacheKey); ok {
			stmt = *pe.stmt
			prepared = pe.plan
			isPrepared = pe.isPrepared
			createPlan = false
		}
	}

	if createPlan {
		stmt, prepared, isPrepared, err = newContext.PrepareStatement(statement, namedArgs, positionalArgs, subquery, readonly, false)

		if err != nil {
			return nil, 0, err
		}

		if doCaching {
			this.udfPlans.set(cacheKey, &planMapEntry{stmt: &stmt, plan: prepared, isPrepared: isPrepared})
		}
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
	rv, mutations, err := newContext.ExecutePrepared(prepared, isPrepared, namedArgs, positionalArgs, statement, doCaching, funcKey)
	newErr := newContext.completeStatement(stmtType, err == nil, this)
	if err == nil && newErr != nil {
		err = newErr
	}
	return rv, mutations, err
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

func (this *Context) OpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool, doCaching bool, funcKey string) (interface {
	Type() string
	Mutations() uint64
	Results() (interface{}, uint64, error)
	Complete() (uint64, error)
	NextDocument() (value.Value, error)
	Cancel()
}, error) {
	newContext := this.Copy()
	newContext.udfPlans = this.udfPlans
	newContext.udfStmtExecTrees = this.udfStmtExecTrees

	txContext := this.TxContext()
	if txContext != nil {
		newContext.SetDeltaKeyspaces(this.DeltaKeyspaces())
		atrCollection, numAtrs := this.AtrCollection()
		newContext.SetTransactionContext("", false, txContext.TxTimeout(), txContext.TxTimeout(), atrCollection, numAtrs, []byte{})
	}

	var stmt algebra.Statement
	var prepared *plan.Prepared
	var isPrepared bool
	var err error

	doCaching = doCaching && (this.udfPlans != nil)
	var cacheKey string
	createPlan := true

	if doCaching {
		cacheKey = encodeStatement(this.queryContext, statement)

		if pe, ok := this.udfPlans.getAndValidate(cacheKey); ok {
			stmt = *pe.stmt
			prepared = pe.plan
			isPrepared = pe.isPrepared
			createPlan = false
		}
	}

	if createPlan {
		stmt, prepared, isPrepared, err = newContext.PrepareStatement(statement, namedArgs, positionalArgs, subquery, readonly, false)

		if err != nil {
			return nil, err
		}

		if doCaching {
			this.udfPlans.set(cacheKey, &planMapEntry{stmt: &stmt, plan: prepared, isPrepared: isPrepared})
		}
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

	return newContext.OpenPrepared(this, stmtType, prepared, isPrepared, namedArgs, positionalArgs, statement, doCaching, funcKey)
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

	semChecker := semantics.NewSemChecker(true /* FIXME */, stmt.Type(), this.TxContext() != nil)
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
		st.SetContext(this)
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
func (this *Context) handleUsing(stmt algebra.Statement, namedArgs map[string]value.Value, positionalArgs value.Values) (map[string]value.Value, value.Values, errors.Error) {

	exec, ok := stmt.(*algebra.Execute)
	if !ok {
		return namedArgs, positionalArgs, nil
	}
	using := exec.Using()
	if using != nil {
		if namedArgs != nil || positionalArgs != nil {
			return namedArgs, positionalArgs, errors.NewExecutionParameterError("cannot have both USING clause and request parameters")
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

func (this *Context) ExecutePrepared(prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values, statement string, doCaching bool, funcKey string) (value.Value, uint64, error) {
	var outputBuf internalOutput
	var results value.Value
	output := this.newOutput(&outputBuf)

	keep := this.output
	this.output = output
	this.SetIsPrepared(isPrepared)
	this.SetPrepared(prepared)
	this.namedArgs = namedArgs
	this.positionalArgs = positionalArgs

	var collect *Collect
	var root Operator
	var eTree execTreeMapEntry
	var ok bool
	var cacheKey string
	var planId int

	doCaching = doCaching && (this.udfStmtExecTrees != nil) && (this.udfPlans != nil)
	createTree := true

	if doCaching {
		cacheKey = encodeStatement(this.queryContext, statement)

		if eTree, ok = this.udfStmtExecTrees.getAndReopen(cacheKey, this); ok {
			createTree = false
			build := util.Now()

			collect = eTree.collRcv.(*Collect)
			root = eTree.root

			keep.AddPhaseTime(INSTANTIATE, util.Since(build))
			this.output = keep

			// Increment the number of times this cached tree entry was used
			if funcKey != "" {
				eTree.usageMetadata[funcKey] += 1
			}
		}
	}

	if createTree {
		build := util.Now()

		// Collect statements results
		collect = NewCollect(plan.NewCollect(), this)
		pipeline, used, err := Build2(prepared, this, collect)
		keep.AddPhaseTime(INSTANTIATE, util.Since(build))

		if err != nil {
			this.output = keep
			return nil, 0, err
		}

		if used {
			root = pipeline
		} else {
			sequence := NewSequence(plan.NewSequence(), this, pipeline, collect)
			root = sequence
		}
	}

	exec := util.Now()
	root.RunOnce(this, nil)

	// Await completion
	collect.waitComplete()
	results = collect.ValuesOnce()

	// Once execution is complete - add the exec tree to the cache
	// Cache the root and collect operator
	if doCaching {
		// Mark the execution tree for re-opening
		if collect.opState == _DONE {
			collect.opState = _COMPLETED
		}

		if createTree {
			// Get the planId of the latest cached plan
			planId, _ = this.udfPlans.getPlanId(cacheKey)
			eTree = execTreeMapEntry{root: root, collRcv: collect, planId: planId}

			if funcKey != "" {
				eTree.usageMetadata = map[string]int{funcKey: 1}
			}
		}
		this.udfStmtExecTrees.set(cacheKey, eTree)
	} else {
		root.Done()
	}

	this.output = keep
	this.output.AddPhaseTime(RUN, util.Since(exec))

	return results, output.mutationCount, output.err
}

func (this *Context) OpenPrepared(baseContext *Context, stmtType string, prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values, statement string, doCaching bool, funcKey string) (interface {
	Type() string
	Mutations() uint64
	Results() (interface{}, uint64, error)
	Complete() (uint64, error)
	NextDocument() (value.Value, error)
	Cancel()
}, error) {
	handle := &executionHandle{}
	handle.context = this
	handle.output = this.newOutput(&internalOutput{})
	handle.context.output = handle.output

	handle.context.SetIsPrepared(isPrepared)
	handle.context.SetPrepared(prepared)
	handle.context.namedArgs = namedArgs
	handle.context.positionalArgs = positionalArgs

	doCaching = doCaching && (baseContext.udfStmtExecTrees != nil) && (this.udfPlans != nil)
	var cacheKey string
	createTree := true
	var eTree execTreeMapEntry
	var ok bool

	var root Operator
	var receive *Receive
	var planId int

	if doCaching {
		cacheKey = encodeStatement(this.queryContext, statement)
		handle.statement = cacheKey

		if eTree, ok = baseContext.udfStmtExecTrees.getAndReopen(cacheKey, this); ok {
			createTree = false
			build := util.Now()
			receive = eTree.collRcv.(*Receive)
			root = eTree.root
			this.output.AddPhaseTime(INSTANTIATE, util.Since(build))

			// Increment the number of times this cached tree entry was used
			if funcKey != "" {
				eTree.usageMetadata[funcKey] += 1
			}
		}
	}

	if createTree {
		build := util.Now()

		// Collect statements results
		receive = NewReceive(plan.NewReceive(), handle.context)
		pipeline, used, err := Build2(prepared, this, receive)
		this.output.AddPhaseTime(INSTANTIATE, util.Since(build))

		if err != nil {
			return nil, err
		}

		if used {
			root = pipeline
		} else {
			root = NewSequence(plan.NewSequence(), this, pipeline, receive)
		}
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
	handle.exec = util.Now()
	root.RunOnce(handle.context, nil)

	if createTree {
		eTree = execTreeMapEntry{root: root, collRcv: receive}

		if doCaching {
			// Get the planId of the latest cached plan
			planId, _ = this.udfPlans.getPlanId(cacheKey)
			eTree.planId = planId

			if funcKey != "" {
				eTree.usageMetadata = map[string]int{funcKey: 1}
			}
		}
	}

	handle.execTree = eTree

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
	baseContext *Context
	context     *Context
	stmtType    string
	actualType  string
	output      *internalOutput
	stopped     int32
	statement   string
	execTree    execTreeMapEntry
}

func (this *executionHandle) Results() (interface{}, uint64, error) {
	if atomic.LoadInt32(&this.stopped) > 0 {
		return nil, 0, nil
	}
	values := make([]interface{}, 0, _INITIAL_SIZE)
	input := this.execTree.collRcv.(*Receive)

	for {
		item, ok := input.getItem()
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
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))

		// Send a PAUSE instead of STOP - so we can re-open cached exec trees if required
		this.execTree.root.SendAction(_ACTION_PAUSE)

		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}

		this.baseContext.mutex.Lock()

		// Once execution is complete - add the exec tree to the cache
		// Cache the root and collect operator
		if this.baseContext.udfStmtExecTrees != nil {
			this.baseContext.udfStmtExecTrees.set(this.statement, this.execTree)
		}

		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
	return values, this.output.mutationCount, this.output.err
}

func (this *executionHandle) Type() string {
	return this.actualType
}

func (this *executionHandle) Mutations() uint64 {
	return this.output.mutationCount
}

func (this *executionHandle) Complete() (uint64, error) {
	if atomic.LoadInt32(&this.stopped) > 0 {
		return 0, nil
	}

	input := this.execTree.collRcv.(*Receive)

	for {
		item, ok := input.getItem()
		if item == nil || !ok {
			break
		}
	}
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))

		// Send a PAUSE instead of STOP - so we can re-open cached exec trees if required
		this.execTree.root.SendAction(_ACTION_PAUSE)

		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}

		this.baseContext.mutex.Lock()

		// Once execution is complete - add the exec tree to the cache
		// Cache the root and collect operator
		if this.baseContext.udfStmtExecTrees != nil {
			this.baseContext.udfStmtExecTrees.set(this.statement, this.execTree)
		}

		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
	return this.output.mutationCount, this.output.err
}

func (this *executionHandle) NextDocument() (value.Value, error) {
	if !this.output.abort && this.stopped == 0 {
		input := this.execTree.collRcv.(*Receive)

		item, _ := input.getItem()
		if item != nil {
			return item, nil
		}
	}

	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))

		// Send a PAUSE instead of STOP - so we can re-open cached exec trees if required
		this.execTree.root.SendAction(_ACTION_PAUSE)

		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}

		this.baseContext.mutex.Lock()

		// Once execution is complete - add the exec tree to the cache
		// Cache the root and collect operator
		if this.baseContext.udfStmtExecTrees != nil {
			this.baseContext.udfStmtExecTrees.set(this.statement, this.execTree)
		}

		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
	return nil, this.output.err
}

func (this *executionHandle) Cancel() {
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))

		// Send a PAUSE instead of STOP - so we can re-open cached exec trees if required
		this.execTree.root.SendAction(_ACTION_PAUSE)

		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}

		this.baseContext.mutex.Lock()

		// Once execution is complete - add the exec tree to the cache
		// Cache the root and collect operator
		if this.baseContext.udfStmtExecTrees != nil {
			this.baseContext.udfStmtExecTrees.set(this.statement, this.execTree)
		}

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
			error = errors.NewError(err, "")
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
	return n1ql.ParseExpression(s)
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

func (this *internalOutput) GetErrorLimit() int {
	return this.output.GetErrorLimit()
}

func (this *internalOutput) GetErrorCount() int {
	return this.output.GetErrorCount()
}
