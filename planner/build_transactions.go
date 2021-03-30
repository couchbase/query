//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitStartTransaction(stmt *algebra.StartTransaction) (interface{}, error) {
	this.maxParallelism = 1
	return plan.NewSequence(plan.NewStartTransaction(stmt)), nil
}

func (this *builder) VisitCommitTransaction(stmt *algebra.CommitTransaction) (interface{}, error) {
	this.maxParallelism = 1
	return plan.NewSequence(plan.NewCommitTransaction(stmt)), nil
}

func (this *builder) VisitRollbackTransaction(stmt *algebra.RollbackTransaction) (interface{}, error) {
	this.maxParallelism = 1
	return plan.NewSequence(plan.NewRollbackTransaction(stmt)), nil
}

func (this *builder) VisitTransactionIsolation(stmt *algebra.TransactionIsolation) (interface{}, error) {
	this.maxParallelism = 1
	return plan.NewSequence(plan.NewTransactionIsolation(stmt)), nil
}

func (this *builder) VisitSavepoint(stmt *algebra.Savepoint) (interface{}, error) {
	this.maxParallelism = 1
	return plan.NewSequence(plan.NewSavepoint(stmt)), nil
}
