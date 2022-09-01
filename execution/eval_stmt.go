//  Copyright 2019-Present Couchbase, Inc.
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

func (this *Context) newOutput(output *internalOutput) *internalOutput {
	output.output = this.output
	return output
}

func (this *Context) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool) (value.Value, uint64, error) {
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
	err = this.handleOpenStatements(stmtType)
	if err != nil {
		return nil, 0, err
	}
	rv, mutations, err := newContext.ExecutePrepared(prepared, isPrepared, namedArgs, positionalArgs)
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
	subquery, readonly bool) (interface {
	Type() string
	Mutations() uint64
	Results() (interface{}, uint64, error)
	Complete() (uint64, error)
	NextDocument() (value.Value, error)
	Cancel()
}, error) {
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
	err = this.handleOpenStatements(stmtType)
	if err != nil {
		return nil, err
	}
	return newContext.OpenPrepared(this, stmtType, prepared, isPrepared, namedArgs, positionalArgs)
}

func (this *Context) PrepareStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly, autoPrepare bool) (stmt algebra.Statement, prepared *plan.Prepared, isPrepared bool, rerr error) {

	if len(namedArgs) > 0 || len(positionalArgs) > 0 || subquery {
		autoPrepare = false
	}

	var name string
	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, this.requestId, this.queryContext, namedArgs,
		positionalArgs, this.indexApiVersion, this.featureControls, this.useFts, this.useCBO, this.optimizer,
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

	stmt, err := n1ql.ParseStatement2(statement, this.namespace, this.queryContext)
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
	prepared, err = planner.BuildPrepared(stmt, this.datastore, this.systemstore, this.namespace, subquery, true,
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
			&reprepTime)
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
	namedArgs map[string]value.Value, positionalArgs value.Values) (value.Value, uint64, error) {

	var outputBuf internalOutput
	var results value.Value
	output := this.newOutput(&outputBuf)

	keep := this.output

	this.output = output
	this.SetIsPrepared(isPrepared)
	this.SetPrepared(prepared)
	this.namedArgs = namedArgs
	this.positionalArgs = positionalArgs

	build := util.Now()

	// Collect statements results
	collect := NewCollect(plan.NewCollect(), this)
	pipeline, used, err := Build2(prepared, this, collect)
	keep.AddPhaseTime(INSTANTIATE, util.Since(build))

	if err != nil {
		this.output = keep
		return nil, 0, err
	}

	exec := util.Now()
	if used {
		pipeline.RunOnce(this, nil)

		// Await completion
		collect.waitComplete()

		results = collect.ValuesOnce()
		pipeline.Done()

	} else {
		sequence := NewSequence(plan.NewSequence(), this, pipeline, collect)
		sequence.RunOnce(this, nil)

		// Await completion
		collect.waitComplete()

		results = collect.ValuesOnce()
		sequence.Done()
	}
	this.output = keep
	this.output.AddPhaseTime(RUN, util.Since(exec))

	return results, output.mutationCount, output.err
}

func (this *Context) OpenPrepared(baseContext *Context, stmtType string, prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values) (interface {
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

	build := util.Now()

	// Collect statements results
	handle.input = NewReceive(plan.NewReceive(), handle.context)
	pipeline, used, err := Build2(prepared, this, handle.input)
	this.output.AddPhaseTime(INSTANTIATE, util.Since(build))
	if err != nil {
		return nil, err
	}

	if used {
		handle.root = pipeline
	} else {
		handle.root = NewSequence(plan.NewSequence(), this, pipeline, handle.input)
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
	handle.root.RunOnce(handle.context, nil)
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
	context     *Context
	stmtType    string
	actualType  string
	output      *internalOutput
	stopped     int32
}

func (this *executionHandle) Results() (interface{}, uint64, error) {
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
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))
		this.root.SendAction(_ACTION_STOP)
		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}
		this.root.Done()
		this.baseContext.mutex.Lock()
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
	for {
		item, ok := this.input.getItem()
		if item == nil || !ok {
			break
		}
	}
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))
		this.root.SendAction(_ACTION_STOP)
		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}
		this.root.Done()
		this.baseContext.mutex.Lock()
		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
	return this.output.mutationCount, this.output.err
}

func (this *executionHandle) NextDocument() (value.Value, error) {
	if !this.output.abort && this.stopped == 0 {
		item, _ := this.input.getItem()
		if item != nil {
			return item, nil
		}
	}

	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))
		this.root.SendAction(_ACTION_STOP)
		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}
		this.root.Done()
		this.baseContext.mutex.Lock()
		delete(this.baseContext.udfHandleMap, this)
		this.baseContext.mutex.Unlock()
	}
	return nil, this.output.err
}

func (this *executionHandle) Cancel() {
	if atomic.AddInt32(&this.stopped, 1) == 1 {
		this.context.output.AddPhaseTime(RUN, util.Since(this.exec))
		this.root.SendAction(_ACTION_STOP)
		newErr := this.context.completeStatement(this.stmtType, this.output.err == nil, this.baseContext)
		if this.output.err == nil && newErr != nil {
			this.output.err = newErr
		}
		this.root.Done()
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
		res, _, err = newContext.ExecutePrepared(prepared, isPrepared, nil, nil)
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
