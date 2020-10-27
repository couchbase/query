//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the Start transaction statement.
*/
type StartTransaction struct {
	statementBase

	isolation datastore.IsolationLevel `json:"isolation"`
}

/*
The function NewStartTransaction returns a pointer to the StartTransaction
struct by assigning the input attributes to the fields of the struct
*/
func NewStartTransaction(isolation datastore.IsolationLevel) *StartTransaction {
	rv := &StartTransaction{
		isolation: isolation,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitStartTransaction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *StartTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitStartTransaction(this)
}

/*
The shape of the start transaction statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *StartTransaction) Signature() value.Value {
	return _JSON_SIGNATURE
}

/*
It's a start transaction
*/
func (this *StartTransaction) Type() string {
	return "START_TRANSACTION"
}

/*
Applies mapper to all the expressions in the start transactions statement.
*/
func (this *StartTransaction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Returns all contained Expressions.
*/
func (this *StartTransaction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *StartTransaction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_QUERY_TRANSACTION_STMT, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the start transaction statement.
*/
func (this *StartTransaction) Formalize() (err error) {
	return
}

/*
Returns the Isolation Level of the transaction
*/
func (this *StartTransaction) IsolationLevel() datastore.IsolationLevel {
	return this.isolation
}

func (this *StartTransaction) HasIsolationLevel(isolation datastore.IsolationLevel) bool {
	return (this.isolation & isolation) != 0
}

/*
Represents the Commit transaction statement.
*/
type CommitTransaction struct {
	statementBase
}

/*
The function NewCommitTransaction returns a pointer to the CommitTransaction
struct by assigning the input attributes to the fields of the struct
*/
func NewCommitTransaction() *CommitTransaction {
	rv := &CommitTransaction{}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitCommitTransaction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *CommitTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCommitTransaction(this)
}

/*
The shape of the commit transaction statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *CommitTransaction) Signature() value.Value {
	return _JSON_SIGNATURE
}

/*
It's a Commit transaction
*/
func (this *CommitTransaction) Type() string {
	return "COMMIT"
}

/*
Applies mapper to all the expressions in the commit transactions statement.
*/
func (this *CommitTransaction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Returns all contained Expressions.
*/
func (this *CommitTransaction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CommitTransaction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_QUERY_TRANSACTION_STMT, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the commit transaction statement.
*/
func (this *CommitTransaction) Formalize() (err error) {
	return
}

/*
Represents the rollback transaction statement.
*/
type RollbackTransaction struct {
	statementBase

	savepoint string `json:"savepoint"`
}

/*
The function NewRollbackTransaction returns a pointer to the RollbackTransaction
struct by assigning the input attributes to the fields of the struct
*/
func NewRollbackTransaction(savepoint string) *RollbackTransaction {
	rv := &RollbackTransaction{savepoint: savepoint}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitRollbackTransaction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RollbackTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRollbackTransaction(this)
}

/*
The shape of the rollback transaction statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *RollbackTransaction) Signature() value.Value {
	return _JSON_SIGNATURE
}

/*
It's a rollback transaction
*/
func (this *RollbackTransaction) Type() string {
	if this.savepoint == "" {
		return "ROLLBACK"
	}
	return "ROLLBACK_SAVEPOINT"
}

/*
Applies mapper to all the expressions in the rollback transactions statement.
*/
func (this *RollbackTransaction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Returns all contained Expressions.
*/
func (this *RollbackTransaction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *RollbackTransaction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_QUERY_TRANSACTION_STMT, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the rollback transaction statement.
*/
func (this *RollbackTransaction) Formalize() (err error) {
	return
}

func (this *RollbackTransaction) Savepoint() string {
	return this.savepoint
}

func (this *RollbackTransaction) IsSavepointRollback() bool {
	return this.savepoint != ""
}

/*
Represents the Set transaction isolation statement.
*/
type TransactionIsolation struct {
	statementBase

	isolation datastore.IsolationLevel `json:"isolation"`
}

/*
The function NewTransactionIsolation returns a pointer to the SET TRANSACTION ISOLATION
struct by assigning the input attributes to the fields of the struct
*/
func NewTransactionIsolation(isolation datastore.IsolationLevel) *TransactionIsolation {
	rv := &TransactionIsolation{
		isolation: isolation,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitTransactionIsolation method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *TransactionIsolation) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitTransactionIsolation(this)
}

/*
The shape of the Set Transaction Isolation statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *TransactionIsolation) Signature() value.Value {
	return _JSON_SIGNATURE
}

/*
It's set transaction isolation
*/
func (this *TransactionIsolation) Type() string {
	return "SET_TRANSACTION_ISOLATION"
}

/*
Applies mapper to all the expressions in the set transaction isolation statement.
*/
func (this *TransactionIsolation) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Returns all contained Expressions.
*/
func (this *TransactionIsolation) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *TransactionIsolation) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_QUERY_TRANSACTION_STMT, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the set transaction isolation statement.
*/
func (this *TransactionIsolation) Formalize() (err error) {
	return
}

/*
Returns the Isolation Level of the transaction
*/
func (this *TransactionIsolation) IsolationLevel() datastore.IsolationLevel {
	return this.isolation
}

func (this *TransactionIsolation) HasIsolationLevel(isolation datastore.IsolationLevel) bool {
	return (this.isolation & isolation) != 0
}
