//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	   INDEX and ALTER INDEX as Data definition statements.
	*/
	VisitCreatePrimaryIndex(stmt *CreatePrimaryIndex) (interface{}, error)
	VisitCreateIndex(stmt *CreateIndex) (interface{}, error)
	VisitDropIndex(stmt *DropIndex) (interface{}, error)
	VisitAlterIndex(stmt *AlterIndex) (interface{}, error)
	VisitBuildIndexes(stmt *BuildIndexes) (interface{}, error)

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
}

type NodeVisitor interface {
	VisitSelectTerm(node *SelectTerm) (interface{}, error)
	VisitSubselect(node *Subselect) (interface{}, error)
	VisitKeyspaceTerm(node *KeyspaceTerm) (interface{}, error)
	VisitExpressionTerm(node *ExpressionTerm) (interface{}, error)
	VisitSubqueryTerm(node *SubqueryTerm) (interface{}, error)
	VisitJoin(node *Join) (interface{}, error)
	VisitIndexJoin(node *IndexJoin) (interface{}, error)
	VisitNest(node *Nest) (interface{}, error)
	VisitIndexNest(node *IndexNest) (interface{}, error)
	VisitUnnest(node *Unnest) (interface{}, error)
	VisitUnion(node *Union) (interface{}, error)
	VisitUnionAll(node *UnionAll) (interface{}, error)
	VisitIntersect(node *Intersect) (interface{}, error)
	VisitIntersectAll(node *IntersectAll) (interface{}, error)
	VisitExcept(node *Except) (interface{}, error)
	VisitExceptAll(node *ExceptAll) (interface{}, error)
}
