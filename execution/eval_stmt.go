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
	"github.com/couchbase/query/transactions"
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
	if this.err == nil {
		this.err = err
	}
}

func (this *internalOutput) Fatal(err errors.Error) {
	if this.err == nil {
		this.err = err
	}
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

func (this *Context) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly bool) (value.Value, uint64, error) {
	prepared, isPrepared, err := this.PrepareStatement(statement, namedArgs, positionalArgs, subquery, readonly, false)
	if err != nil {
		return nil, 0, err
	}

	return this.ExecutePrepared(prepared, isPrepared, namedArgs, positionalArgs)
}

func (this *Context) PrepareStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values,
	subquery, readonly, autoPrepare bool) (prepared *plan.Prepared, isPrepared bool, rerr error) {

	if len(namedArgs) > 0 || len(positionalArgs) > 0 || subquery {
		autoPrepare = false
	}

	var name string
	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, this.requestId, this.queryContext, namedArgs,
		positionalArgs, this.indexApiVersion, this.featureControls, this.useFts, this.useCBO, this.optimizer,
		this.deltaKeyspaces)

	if autoPrepare {
		name = prepareds.GetAutoPrepareName(statement, &prepContext)
		if name != "" {
			prepared = prepareds.GetAutoPreparePlan(name, statement, this.namespace, &prepContext)
			if prepared != nil {
				if readonly && !prepared.Readonly() {
					return nil, false, fmt.Errorf("not a readonly request")
				}
				return prepared, true, nil
			}
		} else {
			autoPrepare = false
		}
	}

	stmt, err := n1ql.ParseStatement2(statement, this.namespace, this.queryContext)
	if err != nil {
		return nil, false, err
	}

	//  monitoring code TBD
	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, false, errors.NewRewriteError(err, "")
	}

	semChecker := semantics.NewSemChecker(true /* FIXME */, stmt.Type(), this.TxContext() != nil)
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return nil, false, err
	}

	switch st := stmt.(type) {
	case *algebra.Prepare:
		prepContext.SetNamedArgs(nil)
		prepContext.SetPositionalArgs(nil)
	case *algebra.Advise:
		st.SetContext(this)
	}

	//  monitoring code TBD
	prepared, err = planner.BuildPrepared(stmt, this.datastore, this.systemstore, this.namespace, subquery, false,
		&prepContext)
	if err != nil {
		return nil, false, err
	}

	if prepared == nil {
		return nil, false, fmt.Errorf("failed to build a plan")
	}

	if readonly && !prepared.Readonly() {
		return nil, false, fmt.Errorf("not a readonly request")
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
			return prepared, isPrepared, err
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
			prepareds.AddAutoPreparePlan(stmt, prepared)
		}

	}

	return prepared, isPrepared, nil
}

func (this *Context) ExecutePrepared(prepared *plan.Prepared, isPrepared bool,
	namedArgs map[string]value.Value, positionalArgs value.Values) (value.Value, uint64, error) {

	var outputBuf internalOutput
	output := &outputBuf

	newContext := this.Copy()
	newContext.output = output
	newContext.SetIsPrepared(isPrepared)
	newContext.SetPrepared(prepared)
	newContext.namedArgs = namedArgs
	newContext.positionalArgs = positionalArgs

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

	prepared, isPrepared, err := this.PrepareStatement(stmt, nil, nil, false, false, true)
	if err == nil {
		res, _, err = this.ExecutePrepared(prepared, isPrepared, nil, nil)
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
	}

	return txId, nil, nil
}

func (this *Context) DoStatementComplete(stmtType string, success bool) (err errors.Error) {
	if this.txContext == nil {
		return nil
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

		if this.txContext != nil {
			if this.txContext.TxImplicit() {
				transactions.DeleteTransContext(this.txContext.TxId(), false)
			}
		}
	}

	return err
}
