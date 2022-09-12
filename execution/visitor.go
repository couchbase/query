//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

type Visitor interface {
	// Scan
	VisitPrimaryScan(op *PrimaryScan) (interface{}, error)
	VisitPrimaryScan3(op *PrimaryScan3) (interface{}, error)
	VisitIndexScan(op *IndexScan) (interface{}, error)
	VisitIndexScan2(op *IndexScan2) (interface{}, error)
	VisitIndexScan3(op *IndexScan3) (interface{}, error)
	VisitKeyScan(op *KeyScan) (interface{}, error)
	VisitValueScan(op *ValueScan) (interface{}, error)
	VisitDummyScan(op *DummyScan) (interface{}, error)
	VisitCountScan(op *CountScan) (interface{}, error)
	VisitIndexCountScan(op *IndexCountScan) (interface{}, error)
	VisitIndexCountScan2(op *IndexCountScan2) (interface{}, error)
	VisitIndexCountDistinctScan2(op *IndexCountDistinctScan2) (interface{}, error)
	VisitDistinctScan(op *DistinctScan) (interface{}, error)
	VisitUnionScan(op *UnionScan) (interface{}, error)
	VisitIntersectScan(op *IntersectScan) (interface{}, error)
	VisitOrderedIntersectScan(op *OrderedIntersectScan) (interface{}, error)
	VisitExpressionScan(op *ExpressionScan) (interface{}, error)

	// FTS Search
	VisitIndexFtsSearch(op *IndexFtsSearch) (interface{}, error)

	// Fetch
	VisitFetch(op *Fetch) (interface{}, error)
	VisitDummyFetch(op *DummyFetch) (interface{}, error)

	// Join
	VisitJoin(op *Join) (interface{}, error)
	VisitIndexJoin(op *IndexJoin) (interface{}, error)
	VisitNest(op *Nest) (interface{}, error)
	VisitIndexNest(op *IndexNest) (interface{}, error)
	VisitUnnest(op *Unnest) (interface{}, error)
	VisitNLJoin(op *NLJoin) (interface{}, error)
	VisitNLNest(op *NLNest) (interface{}, error)
	VisitHashJoin(op *HashJoin) (interface{}, error)
	VisitHashNest(op *HashNest) (interface{}, error)

	// Let + Letting, With
	VisitLet(op *Let) (interface{}, error)
	VisitWith(op *With) (interface{}, error)

	// Filter
	VisitFilter(op *Filter) (interface{}, error)

	// Group
	VisitInitialGroup(op *InitialGroup) (interface{}, error)
	VisitIntermediateGroup(op *IntermediateGroup) (interface{}, error)
	VisitFinalGroup(op *FinalGroup) (interface{}, error)

	// Window functions
	VisitWindowAggregate(op *WindowAggregate) (interface{}, error)

	// Project
	VisitInitialProject(op *InitialProject) (interface{}, error)

	// TODO retire
	VisitFinalProject(op interface{}) (interface{}, error)
	VisitIndexCountProject(op *IndexCountProject) (interface{}, error)

	// Distinct
	VisitDistinct(op *Distinct) (interface{}, error)

	//All
	VisitAll(op *All) (interface{}, error)

	// Set operators
	VisitUnionAll(op *UnionAll) (interface{}, error)
	VisitIntersect(op *Intersect) (interface{}, error)
	VisitIntersectAll(op *IntersectAll) (interface{}, error)
	VisitExcept(op *Except) (interface{}, error)
	VisitExceptAll(op *ExceptAll) (interface{}, error)

	// Order
	VisitOrder(op *Order) (interface{}, error)

	// Offset
	VisitOffset(op *Offset) (interface{}, error)
	VisitLimit(op *Limit) (interface{}, error)

	// Insert
	VisitSendInsert(op *SendInsert) (interface{}, error)

	// Insert
	VisitSendUpsert(op *SendUpsert) (interface{}, error)

	// Delete
	VisitSendDelete(op *SendDelete) (interface{}, error)

	// Update
	VisitClone(op *Clone) (interface{}, error)
	VisitSet(op *Set) (interface{}, error)
	VisitUnset(op *Unset) (interface{}, error)
	VisitSendUpdate(op *SendUpdate) (interface{}, error)

	// Merge
	VisitMerge(op *Merge) (interface{}, error)

	// Framework
	VisitAlias(op *Alias) (interface{}, error)
	VisitAuthorize(op *Authorize) (interface{}, error)
	VisitParallel(op *Parallel) (interface{}, error)
	VisitSequence(op *Sequence) (interface{}, error)
	VisitDiscard(op *Discard) (interface{}, error)
	VisitStream(op *Stream) (interface{}, error)
	VisitCollect(op *Collect) (interface{}, error)
	VisitReceive(op *Receive) (interface{}, error)
	VisitChannel(op *Channel) (interface{}, error)

	// Index DDL
	VisitCreatePrimaryIndex(op *CreatePrimaryIndex) (interface{}, error)
	VisitCreateIndex(op *CreateIndex) (interface{}, error)
	VisitDropIndex(op *DropIndex) (interface{}, error)
	VisitAlterIndex(op *AlterIndex) (interface{}, error)
	VisitBuildIndexes(op *BuildIndexes) (interface{}, error)

	// Collections, Roles DDL
	VisitCreateScope(op *CreateScope) (interface{}, error)
	VisitDropScope(op *DropScope) (interface{}, error)
	VisitCreateCollection(op *CreateCollection) (interface{}, error)
	VisitDropCollection(op *DropCollection) (interface{}, error)
	VisitFlushCollection(op *FlushCollection) (interface{}, error)

	// Roles
	VisitGrantRole(op *GrantRole) (interface{}, error)
	VisitRevokeRole(op *RevokeRole) (interface{}, error)

	// Explain
	VisitExplain(op *Explain) (interface{}, error)

	// Explain Function
	VisitExplainFunction(op *ExplainFunction) (interface{}, error)

	// Prepare
	VisitPrepare(op *Prepare) (interface{}, error)

	// Infer
	VisitInferKeyspace(op *InferKeyspace) (interface{}, error)
	VisitInferExpression(op *InferExpression) (interface{}, error)

	// Functions
	VisitCreateFunction(op *CreateFunction) (interface{}, error)
	VisitDropFunction(op *DropFunction) (interface{}, error)
	VisitExecuteFunction(op *ExecuteFunction) (interface{}, error)

	// Index Advisor
	VisitIndexAdvice(op *IndexAdvice) (interface{}, error)
	VisitAdvise(op *Advise) (interface{}, error)

	// Update Statistics
	VisitUpdateStatistics(op *UpdateStatistics) (interface{}, error)

	// Transactions
	VisitStartTransaction(op *StartTransaction) (interface{}, error)
	VisitCommitTransaction(op *CommitTransaction) (interface{}, error)
	VisitRollbackTransaction(op *RollbackTransaction) (interface{}, error)
	VisitTransactionIsolation(op *TransactionIsolation) (interface{}, error)
	VisitSavepoint(op *Savepoint) (interface{}, error)

	// Sequences
	VisitCreateSequence(op *CreateSequence) (interface{}, error)
	VisitDropSequence(op *DropSequence) (interface{}, error)
	VisitAlterSequence(op *AlterSequence) (interface{}, error)
}
