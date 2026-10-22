//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

type Visitor interface {
	// Scan
	VisitPrimaryScan(op *PrimaryScan) (any, error)
	VisitPrimaryScan3(op *PrimaryScan3) (any, error)
	VisitIndexScan(op *IndexScan) (any, error)
	VisitIndexScan2(op *IndexScan2) (any, error)
	VisitIndexScan3(op *IndexScan3) (any, error)
	VisitKeyScan(op *KeyScan) (any, error)
	VisitValueScan(op *ValueScan) (any, error)
	VisitDummyScan(op *DummyScan) (any, error)
	VisitExternalScan(op *ExternalScan) (any, error)
	VisitCountScan(op *CountScan) (any, error)
	VisitIndexCountScan(op *IndexCountScan) (any, error)
	VisitIndexCountScan2(op *IndexCountScan2) (any, error)
	VisitIndexCountDistinctScan2(op *IndexCountDistinctScan2) (any, error)
	VisitDistinctScan(op *DistinctScan) (any, error)
	VisitUnionScan(op *UnionScan) (any, error)
	VisitIntersectScan(op *IntersectScan) (any, error)
	VisitOrderedIntersectScan(op *OrderedIntersectScan) (any, error)
	VisitExpressionScan(op *ExpressionScan) (any, error)

	// FTS Search
	VisitIndexFtsSearch(op *IndexFtsSearch) (any, error)

	// Fetch
	VisitFetch(op *Fetch) (any, error)
	VisitDummyFetch(op *DummyFetch) (any, error)

	// Join
	VisitJoin(op *Join) (any, error)
	VisitIndexJoin(op *IndexJoin) (any, error)
	VisitNest(op *Nest) (any, error)
	VisitIndexNest(op *IndexNest) (any, error)
	VisitUnnest(op *Unnest) (any, error)
	VisitNLJoin(op *NLJoin) (any, error)
	VisitNLNest(op *NLNest) (any, error)
	VisitHashJoin(op *HashJoin) (any, error)
	VisitHashNest(op *HashNest) (any, error)

	// Let + Letting, With
	VisitLet(op *Let) (any, error)
	VisitWith(op *With) (any, error)

	// Filter
	VisitFilter(op *Filter) (any, error)

	// Group
	VisitInitialGroup(op *InitialGroup) (any, error)
	VisitIntermediateGroup(op *IntermediateGroup) (any, error)
	VisitFinalGroup(op *FinalGroup) (any, error)

	// Window functions
	VisitWindowAggregate(op *WindowAggregate) (any, error)

	// Project
	VisitInitialProject(op *InitialProject) (any, error)

	VisitIndexCountProject(op *IndexCountProject) (any, error)

	// Distinct
	VisitDistinct(op *Distinct) (any, error)

	// All
	VisitAll(op *All) (any, error)

	// Set operators
	VisitUnionAll(op *UnionAll) (any, error)
	VisitIntersectAll(op *IntersectAll) (any, error)
	VisitExceptAll(op *ExceptAll) (any, error)

	// Order
	VisitOrder(op *Order) (any, error)

	// Paging
	VisitOffset(op *Offset) (any, error)
	VisitLimit(op *Limit) (any, error)

	// Insert
	VisitSendInsert(op *SendInsert) (any, error)

	// Upsert
	VisitSendUpsert(op *SendUpsert) (any, error)

	// Delete
	VisitSendDelete(op *SendDelete) (any, error)

	// Update
	VisitClone(op *Clone) (any, error)
	VisitSet(op *Set) (any, error)
	VisitUnset(op *Unset) (any, error)
	VisitSendUpdate(op *SendUpdate) (any, error)

	// Merge
	VisitMerge(op *Merge) (any, error)

	// Framework
	VisitAlias(op *Alias) (any, error)
	VisitAuthorize(op *Authorize) (any, error)
	VisitParallel(op *Parallel) (any, error)
	VisitSequence(op *Sequence) (any, error)
	VisitDiscard(op *Discard) (any, error)
	VisitStream(op *Stream) (any, error)
	VisitCollect(op *Collect) (any, error)
	VisitReceive(op *Receive) (any, error)

	// Index DDL
	VisitCreatePrimaryIndex(op *CreatePrimaryIndex) (any, error)
	VisitCreateIndex(op *CreateIndex) (any, error)
	VisitDropIndex(op *DropIndex) (any, error)
	VisitAlterIndex(op *AlterIndex) (any, error)
	VisitBuildIndexes(op *BuildIndexes) (any, error)

	// Buckets DDL
	VisitCreateBucket(op *CreateBucket) (any, error)
	VisitAlterBucket(op *AlterBucket) (any, error)
	VisitDropBucket(op *DropBucket) (any, error)

	// Catalog DDL
	VisitCreateCatalog(op *CreateCatalog) (any, error)
	VisitAlterCatalog(op *AlterCatalog) (any, error)
	VisitDropCatalog(op *DropCatalog) (any, error)

	// Scope and Collection  DDL
	VisitCreateScope(op *CreateScope) (any, error)
	VisitDropScope(op *DropScope) (any, error)
	VisitCreateCollection(op *CreateCollection) (any, error)
	VisitDropCollection(op *DropCollection) (any, error)
	VisitFlushCollection(op *FlushCollection) (any, error)
	VisitAlterCollection(op *AlterCollection) (any, error)

	// Users
	VisitCreateUser(op *CreateUser) (any, error)
	VisitAlterUser(op *AlterUser) (any, error)
	VisitDropUser(op *DropUser) (any, error)

	// Groups
	VisitCreateGroup(op *CreateGroup) (any, error)
	VisitAlterGroup(op *AlterGroup) (any, error)
	VisitDropGroup(op *DropGroup) (any, error)

	// Roles
	VisitGrantRole(op *GrantRole) (any, error)
	VisitRevokeRole(op *RevokeRole) (any, error)

	// Explain
	VisitExplain(op *Explain) (any, error)

	// Explain Function
	VisitExplainFunction(op *ExplainFunction) (any, error)

	// Prepare
	VisitPrepare(op *Prepare) (any, error)

	// Infer
	VisitInferKeyspace(op *InferKeyspace) (any, error)
	VisitInferExpression(op *InferExpression) (any, error)

	// Function statements
	VisitCreateFunction(op *CreateFunction) (any, error)
	VisitDropFunction(op *DropFunction) (any, error)
	VisitExecuteFunction(op *ExecuteFunction) (any, error)

	// Index Advisor
	VisitIndexAdvice(op *IndexAdvice) (any, error)
	VisitAdvise(op *Advise) (any, error)

	// Update Statistics
	VisitUpdateStatistics(op *UpdateStatistics) (any, error)

	// Transactions
	VisitStartTransaction(op *StartTransaction) (any, error)
	VisitCommitTransaction(op *CommitTransaction) (any, error)
	VisitRollbackTransaction(op *RollbackTransaction) (any, error)
	VisitTransactionIsolation(op *TransactionIsolation) (any, error)
	VisitSavepoint(op *Savepoint) (any, error)

	// Sequences
	VisitCreateSequence(op *CreateSequence) (any, error)
	VisitDropSequence(op *DropSequence) (any, error)
	VisitAlterSequence(op *AlterSequence) (any, error)

	// Knowledge
	VisitCreateKnowledge(op *CreateKnowledge) (any, error)
	VisitDropKnowledge(op *DropKnowledge) (any, error)

	// CredentialStore
	VisitCreateCredentialStore(op *CreateCredentialStore) (any, error)
	VisitAlterCredentialStore(op *AlterCredentialStore) (any, error)
	VisitDropCredentialStore(op *DropCredentialStore) (any, error)
}
