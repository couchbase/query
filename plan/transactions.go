//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plan

import (
	"encoding/json"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
)

type StartTransaction struct {
	ddl
	isolation datastore.IsolationLevel
}

func NewStartTransaction(stmt *algebra.StartTransaction) *StartTransaction {
	return &StartTransaction{isolation: stmt.IsolationLevel()}
}

func (this *StartTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitStartTransaction(this)
}

func (this *StartTransaction) New() Operator {
	return &StartTransaction{}
}

func (this *StartTransaction) IsolationLevel() datastore.IsolationLevel {
	return this.isolation
}

func (this *StartTransaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *StartTransaction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "StartTransaction"}
	r["isolation"] = this.IsolationLevel()

	if f != nil {
		f(r)
	}
	return r
}

func (this *StartTransaction) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string                   `json:"#operator"`
		Isolation datastore.IsolationLevel `json:"isolation"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.isolation = _unmarshalled.Isolation
	return nil
}

type CommitTransaction struct {
	ddl
}

func NewCommitTransaction(stmt *algebra.CommitTransaction) *CommitTransaction {
	return &CommitTransaction{}
}

func (this *CommitTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCommitTransaction(this)
}

func (this *CommitTransaction) New() Operator {
	return &CommitTransaction{}
}

func (this *CommitTransaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CommitTransaction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CommitTransaction"}

	if f != nil {
		f(r)
	}
	return r
}

func (this *CommitTransaction) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_ string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	return nil
}

type RollbackTransaction struct {
	ddl
	savepoint string
}

func NewRollbackTransaction(stmt *algebra.RollbackTransaction) *RollbackTransaction {
	return &RollbackTransaction{savepoint: stmt.Savepoint()}
}

func (this *RollbackTransaction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRollbackTransaction(this)
}

func (this *RollbackTransaction) New() Operator {
	return &RollbackTransaction{}
}

func (this *RollbackTransaction) Savepoint() string {
	return this.savepoint
}

func (this *RollbackTransaction) IsSavepointRollback() bool {
	return this.savepoint != ""
}

func (this *RollbackTransaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *RollbackTransaction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "RollbackTransaction"}
	if this.IsSavepointRollback() {
		r["savepoint"] = this.Savepoint()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *RollbackTransaction) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Savepoint string `json:"savepoint"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.savepoint = _unmarshalled.Savepoint
	return nil
}

type TransactionIsolation struct {
	ddl
	isolation datastore.IsolationLevel
}

func NewTransactionIsolation(stmt *algebra.TransactionIsolation) *TransactionIsolation {
	return &TransactionIsolation{isolation: stmt.IsolationLevel()}
}

func (this *TransactionIsolation) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitTransactionIsolation(this)
}

func (this *TransactionIsolation) New() Operator {
	return &TransactionIsolation{}
}

func (this *TransactionIsolation) IsolationLevel() datastore.IsolationLevel {
	return this.isolation
}

func (this *TransactionIsolation) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *TransactionIsolation) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "TransactionIsolation"}
	r["isolation"] = this.IsolationLevel()

	if f != nil {
		f(r)
	}
	return r
}

func (this *TransactionIsolation) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string                   `json:"#operator"`
		Isolation datastore.IsolationLevel `json:"isolation"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.isolation = _unmarshalled.Isolation
	return nil
}

type Savepoint struct {
	ddl
	savepoint string
}

func NewSavepoint(stmt *algebra.Savepoint) *Savepoint {
	return &Savepoint{savepoint: stmt.Savepoint()}
}

func (this *Savepoint) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSavepoint(this)
}

func (this *Savepoint) New() Operator {
	return &Savepoint{}
}

func (this *Savepoint) Savepoint() string {
	return this.savepoint
}

func (this *Savepoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Savepoint) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Savepoint"}
	r["savepoint"] = this.Savepoint()

	if f != nil {
		f(r)
	}
	return r
}

func (this *Savepoint) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Savepoint string `json:"savepoint"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.savepoint = _unmarshalled.Savepoint
	return nil
}
