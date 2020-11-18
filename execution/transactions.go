//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/value"
)

func init() {
	rv := &Context{datastore: datastore.GetDatastore()}
	transactions.SetQueryContext(rv)
}

type StartTransaction struct {
	base
	plan *plan.StartTransaction
}

func NewStartTransaction(plan *plan.StartTransaction, context *Context) *StartTransaction {
	rv := &StartTransaction{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *StartTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitStartTransaction(this)
}

func (this *StartTransaction) Copy() Operator {
	rv := &StartTransaction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *StartTransaction) PlanOp() plan.Operator {
	return this.plan
}

func (this *StartTransaction) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		var err errors.Error
		defer func() {
			if err != nil {
				// on error rollback any backend transaction and remove from cache
				if context.txContext != nil {
					context.datastore.RollbackTransaction(false, context, "")
					transactions.DeleteTransContext(context.txContext.TxId(), false)
					context.txContext = nil
				}
				context.Error(err)
			}
		}()

		if !active || context.Readonly() {
			return
		}

		this.switchPhase(_SERVTIME)

		// Allocate transaction structure
		durabilityLevel := context.durabilityLevel
		if durabilityLevel == datastore.DL_UNSET {
			durabilityLevel = datastore.DEF_DURABILITY_LEVEL
		}

		consistency := context.consistency
		if context.originalConsistency == datastore.NOT_SET || context.originalConsistency == datastore.AT_PLUS {
			consistency = datastore.SCAN_PLUS
		}

		txData := context.txData
		if context.txImplicit {
			txData = nil
		}

		context.txContext = transactions.NewTxContext(context.txImplicit, txData, context.txTimeout,
			context.durabilityTimeout, context.kvTimeout, durabilityLevel, this.plan.IsolationLevel(),
			consistency, context.atrCollection, context.numAtrs)

		if context.txContext == nil {
			err = errors.NewStartTransactionError(fmt.Errorf("txcontext allocation"), nil)
			return
		}

		// Start transaction
		if _, err = context.datastore.StartTransaction(false, context); err != nil {
			context.txContext = nil
			return
		}

		if err = transactions.AddTransContext(context.txContext); err != nil {
			return
		}

		// return transaction id
		sv := value.NewScopeValue(make(map[string]interface{}, 2), parent)
		sv.SetField("txid", context.txContext.TxId())
		if !this.sendItem(value.NewAnnotatedValue(sv)) {
			err = errors.NewStartTransactionError(fmt.Errorf("sendItem"), nil)
			return
		}
	})
}

func (this *StartTransaction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *StartTransaction) Done() {
	this.baseDone()
	if this.isComplete() {
	}
}

type CommitTransaction struct {
	base
	plan *plan.CommitTransaction
}

func NewCommitTransaction(plan *plan.CommitTransaction, context *Context) *CommitTransaction {
	rv := &CommitTransaction{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *CommitTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCommitTransaction(this)
}

func (this *CommitTransaction) Copy() Operator {
	rv := &CommitTransaction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CommitTransaction) PlanOp() plan.Operator {
	return this.plan
}

func (this *CommitTransaction) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		this.switchPhase(_SERVTIME)
		if context.txContext == nil {
			context.Error(errors.NewTransactionContextError(fmt.Errorf("txcontext is nil in commit")))
			return
		}

		// Commit transaction
		if err := context.datastore.CommitTransaction(false, context); err != nil {
			context.Error(err)
			return
		}
	})
}

func (this *CommitTransaction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *CommitTransaction) Done() {
	this.baseDone()
	if this.isComplete() {
	}
}

type RollbackTransaction struct {
	base
	plan *plan.RollbackTransaction
}

func NewRollbackTransaction(plan *plan.RollbackTransaction, context *Context) *RollbackTransaction {
	rv := &RollbackTransaction{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *RollbackTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRollbackTransaction(this)
}

func (this *RollbackTransaction) Copy() Operator {
	rv := &RollbackTransaction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *RollbackTransaction) PlanOp() plan.Operator {
	return this.plan
}

func (this *RollbackTransaction) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		this.switchPhase(_SERVTIME)
		if context.txContext == nil {
			context.Error(errors.NewTransactionContextError(fmt.Errorf("txcontext is nil in rollback")))
			return
		}

		// Rollback transaction
		if err := context.datastore.RollbackTransaction(false, context, this.plan.Savepoint()); err != nil {
			context.Error(err)
			return
		}

	})
}

func (this *RollbackTransaction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *RollbackTransaction) Done() {
	this.baseDone()
	if this.isComplete() {
	}
}

type TransactionIsolation struct {
	base
	plan *plan.TransactionIsolation
}

func NewTransactionIsolation(plan *plan.TransactionIsolation, context *Context) *TransactionIsolation {
	rv := &TransactionIsolation{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *TransactionIsolation) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitTransactionIsolation(this)
}

func (this *TransactionIsolation) Copy() Operator {
	rv := &TransactionIsolation{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *TransactionIsolation) PlanOp() plan.Operator {
	return this.plan
}

func (this *TransactionIsolation) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		this.switchPhase(_SERVTIME)
		if context.txContext == nil {
			context.Error(errors.NewTransactionContextError(fmt.Errorf("txcontext is nil in transaction Isolation")))
			return
		}

		// Set Isolation transaction
		context.txContext.SetTxIsolationLevel(this.plan.IsolationLevel())
	})
}

func (this *TransactionIsolation) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *TransactionIsolation) Done() {
	this.baseDone()
	if this.isComplete() {
	}
}

type Savepoint struct {
	base
	plan *plan.Savepoint
}

func NewSavepoint(plan *plan.Savepoint, context *Context) *Savepoint {
	rv := &Savepoint{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *Savepoint) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSavepoint(this)
}

func (this *Savepoint) Copy() Operator {
	rv := &Savepoint{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Savepoint) PlanOp() plan.Operator {
	return this.plan
}

func (this *Savepoint) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		this.switchPhase(_SERVTIME)
		if context.txContext == nil {
			context.Error(errors.NewTransactionContextError(fmt.Errorf("txcontext is nil in savepoint")))
			return
		}

		// Set Savepoint
		if err := context.datastore.SetSavepoint(false, context, this.plan.Savepoint()); err != nil {
			context.Error(err)
			return
		}
	})
}

func (this *Savepoint) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Savepoint) Done() {
	this.baseDone()
	if this.isComplete() {
	}
}
