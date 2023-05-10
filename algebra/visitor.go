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
	VisitSelect(stmt *Select) (interface{}, error)

	/*
	   Visitor for DML statements. N1QL provides several data
	   modification statements such as INSERT, UPSERT, DELETE,
	   UPDATE and MERGE.
	*/
	VisitInsert(stmt *Insert) (interface{}, error)
	VisitUpsert(stmt *Upsert) (interface{}, error)
	VisitDelete(stmt *Delete) (interface{}, error)
	VisitUpdate(stmt *Update) (interface{}, error)
	VisitMerge(stmt *Merge) (interface{}, error)

	/*
	   Visitor for DDL statements. N1QL provides index
	   statements CREATE PRIMARY INDEX, CREATE INDEX, DROP
	   INDEX, ALTER INDEX, CREATE SCOPE, DROP SCOPE,
	   CREATE COLLECTION, DROP COLLECTION and FLUSH COLLECTION
	   as Data definition statements.
	*/
	VisitCreatePrimaryIndex(stmt *CreatePrimaryIndex) (interface{}, error)
	VisitCreateIndex(stmt *CreateIndex) (interface{}, error)
	VisitDropIndex(stmt *DropIndex) (interface{}, error)
	VisitAlterIndex(stmt *AlterIndex) (interface{}, error)
	VisitBuildIndexes(stmt *BuildIndexes) (interface{}, error)
	VisitCreateScope(stmt *CreateScope) (interface{}, error)
	VisitDropScope(stmt *DropScope) (interface{}, error)
	VisitCreateCollection(stmt *CreateCollection) (interface{}, error)
	VisitDropCollection(stmt *DropCollection) (interface{}, error)
	VisitFlushCollection(stmt *FlushCollection) (interface{}, error)

	/*
	   Visitor for ROLES statements.
	*/
	VisitGrantRole(stmt *GrantRole) (interface{}, error)
	VisitRevokeRole(stmt *RevokeRole) (interface{}, error)

	/*
	   Visitor for EXPLAIN statements.
	*/
	VisitExplain(stmt *Explain) (interface{}, error)

	/*
	   Visitor for EXPLAIN FUNCTION statements.
	*/
	VisitExplainFunction(stmt *ExplainFunction) (interface{}, error)

	/*
	   Visitor for ADVISE statements.
	*/
	VisitAdvise(stmt *Advise) (interface{}, error)

	/*
	   Visitor for PREPARED statements.
	*/
	VisitPrepare(stmt *Prepare) (interface{}, error)

	/*
	   Visitor for EXECUTE statements.
	*/
	VisitExecute(stmt *Execute) (interface{}, error)

	/*
	   Visitor for INFER statements.
	*/
	VisitInferKeyspace(stmt *InferKeyspace) (interface{}, error)
	VisitInferExpression(stmt *InferExpression) (interface{}, error)

	/*
	   Visitor FUNCTION statements
	*/
	VisitCreateFunction(stmt *CreateFunction) (interface{}, error)
	VisitDropFunction(stmt *DropFunction) (interface{}, error)
	VisitExecuteFunction(stmt *ExecuteFunction) (interface{}, error)

	/*
	   Visitor for UPDATE STATISTICS statements.
	*/
	VisitUpdateStatistics(stmt *UpdateStatistics) (interface{}, error)

	/*
	   Visitor for Transaction statements.
	*/
	VisitStartTransaction(stmt *StartTransaction) (interface{}, error)
	VisitCommitTransaction(stmt *CommitTransaction) (interface{}, error)
	VisitRollbackTransaction(stmt *RollbackTransaction) (interface{}, error)
	VisitTransactionIsolation(stmt *TransactionIsolation) (interface{}, error)
	VisitSavepoint(stmt *Savepoint) (interface{}, error)
}

type NodeVisitor interface {
	VisitSelectTerm(node *SelectTerm) (interface{}, error)
	VisitSubselect(node *Subselect) (interface{}, error)
	VisitKeyspaceTerm(node *KeyspaceTerm) (interface{}, error)
	VisitExpressionTerm(node *ExpressionTerm) (interface{}, error)
	VisitSubqueryTerm(node *SubqueryTerm) (interface{}, error)
	VisitJoin(node *Join) (interface{}, error)
	VisitIndexJoin(node *IndexJoin) (interface{}, error)
	VisitAnsiJoin(node *AnsiJoin) (interface{}, error)
	VisitNest(node *Nest) (interface{}, error)
	VisitIndexNest(node *IndexNest) (interface{}, error)
	VisitAnsiNest(node *AnsiNest) (interface{}, error)
	VisitUnnest(node *Unnest) (interface{}, error)
	VisitUnion(node *Union) (interface{}, error)
	VisitUnionAll(node *UnionAll) (interface{}, error)
	VisitIntersect(node *Intersect) (interface{}, error)
	VisitIntersectAll(node *IntersectAll) (interface{}, error)
	VisitExcept(node *Except) (interface{}, error)
	VisitExceptAll(node *ExceptAll) (interface{}, error)
}
