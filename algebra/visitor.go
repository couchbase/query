//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

type Visitor interface {
	/*
	   Visitor for SELECT statement.
	*/
	VisitSelect(stmt *Select) (any, error)

	/*
	   Visitor for DML statements. N1QL provides several data
	   modification statements such as INSERT, UPSERT, DELETE,
	   UPDATE and MERGE.
	*/
	VisitInsert(stmt *Insert) (any, error)
	VisitUpsert(stmt *Upsert) (any, error)
	VisitDelete(stmt *Delete) (any, error)
	VisitUpdate(stmt *Update) (any, error)
	VisitMerge(stmt *Merge) (any, error)

	/*
	   Visitor for DDL statements. N1QL provides index
	   statements CREATE PRIMARY INDEX, CREATE INDEX, DROP
	   INDEX, ALTER INDEX, CREATE SCOPE, DROP SCOPE,
	   CREATE COLLECTION, DROP COLLECTION and FLUSH COLLECTION
	   as Data definition statements.
	*/
	VisitCreatePrimaryIndex(stmt *CreatePrimaryIndex) (any, error)
	VisitCreateIndex(stmt *CreateIndex) (any, error)
	VisitDropIndex(stmt *DropIndex) (any, error)
	VisitAlterIndex(stmt *AlterIndex) (any, error)
	VisitBuildIndexes(stmt *BuildIndexes) (any, error)
	VisitCreateScope(stmt *CreateScope) (any, error)
	VisitDropScope(stmt *DropScope) (any, error)
	VisitCreateCollection(stmt *CreateCollection) (any, error)
	VisitDropCollection(stmt *DropCollection) (any, error)
	VisitFlushCollection(stmt *FlushCollection) (any, error)
	VisitCreateBucket(stmt *CreateBucket) (any, error)
	VisitAlterBucket(stmt *AlterBucket) (any, error)
	VisitDropBucket(stmt *DropBucket) (any, error)
	VisitCreateCatalog(stmt *CreateCatalog) (any, error)
	VisitAlterCatalog(stmt *AlterCatalog) (any, error)
	VisitDropCatalog(stmt *DropCatalog) (any, error)
	VisitAlterCollection(stmt *AlterCollection) (any, error)

	/*
		USER statements.
	*/
	VisitCreateUser(stmt *CreateUser) (any, error)
	VisitAlterUser(stmt *AlterUser) (any, error)
	VisitDropUser(stmt *DropUser) (any, error)

	/*
		GROUP statements.
	*/
	VisitCreateGroup(stmt *CreateGroup) (any, error)
	VisitAlterGroup(stmt *AlterGroup) (any, error)
	VisitDropGroup(stmt *DropGroup) (any, error)

	/*
	   Visitor for ROLES statements.
	*/
	VisitGrantRole(stmt *GrantRole) (any, error)
	VisitRevokeRole(stmt *RevokeRole) (any, error)

	/*
	   Visitor for EXPLAIN statements.
	*/
	VisitExplain(stmt *Explain) (any, error)

	/*
	   Visitor for EXPLAIN FUNCTION statements.
	*/
	VisitExplainFunction(stmt *ExplainFunction) (any, error)

	/*
	   Visitor for ADVISE statements.
	*/
	VisitAdvise(stmt *Advise) (any, error)

	/*
	   Visitor for PREPARED statements.
	*/
	VisitPrepare(stmt *Prepare) (any, error)

	/*
	   Visitor for EXECUTE statements.
	*/
	VisitExecute(stmt *Execute) (any, error)

	/*
	   Visitor for INFER statements.
	*/
	VisitInferKeyspace(stmt *InferKeyspace) (any, error)
	VisitInferExpression(stmt *InferExpression) (any, error)

	/*
	   Visitor FUNCTION statements
	*/
	VisitCreateFunction(stmt *CreateFunction) (any, error)
	VisitDropFunction(stmt *DropFunction) (any, error)
	VisitExecuteFunction(stmt *ExecuteFunction) (any, error)

	/*
	   Visitor for UPDATE STATISTICS statements.
	*/
	VisitUpdateStatistics(stmt *UpdateStatistics) (any, error)

	/*
	   Visitor for Transaction statements.
	*/
	VisitStartTransaction(stmt *StartTransaction) (any, error)
	VisitCommitTransaction(stmt *CommitTransaction) (any, error)
	VisitRollbackTransaction(stmt *RollbackTransaction) (any, error)
	VisitTransactionIsolation(stmt *TransactionIsolation) (any, error)
	VisitSavepoint(stmt *Savepoint) (any, error)

	VisitCreateSequence(stmt *CreateSequence) (any, error)
	VisitDropSequence(stmt *DropSequence) (any, error)
	VisitAlterSequence(stmt *AlterSequence) (any, error)

	VisitCreateKnowledge(stmt *CreateKnowledge) (any, error)
	VisitDropKnowledge(stmt *DropKnowledge) (any, error)

	VisitCreateCredentialStore(stmt *CreateCredentialStore) (any, error)
	VisitAlterCredentialStore(stmt *AlterCredentialStore) (any, error)
	VisitDropCredentialStore(stmt *DropCredentialStore) (any, error)
}

type NodeVisitor interface {
	VisitSelectTerm(node *SelectTerm) (any, error)
	VisitSubselect(node *Subselect) (any, error)
	VisitKeyspaceTerm(node *KeyspaceTerm) (any, error)
	VisitExpressionTerm(node *ExpressionTerm) (any, error)
	VisitSubqueryTerm(node *SubqueryTerm) (any, error)
	VisitJoin(node *Join) (any, error)
	VisitIndexJoin(node *IndexJoin) (any, error)
	VisitAnsiJoin(node *AnsiJoin) (any, error)
	VisitNest(node *Nest) (any, error)
	VisitIndexNest(node *IndexNest) (any, error)
	VisitAnsiNest(node *AnsiNest) (any, error)
	VisitUnnest(node *Unnest) (any, error)
	VisitUnion(node *Union) (any, error)
	VisitUnionAll(node *UnionAll) (any, error)
	VisitIntersect(node *Intersect) (any, error)
	VisitIntersectAll(node *IntersectAll) (any, error)
	VisitExcept(node *Except) (any, error)
	VisitExceptAll(node *ExceptAll) (any, error)
}
