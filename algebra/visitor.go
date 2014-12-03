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
	// SELECT
	VisitSelect(stmt *Select) (interface{}, error)

	// DML
	VisitInsert(stmt *Insert) (interface{}, error)
	VisitUpsert(stmt *Upsert) (interface{}, error)
	VisitDelete(stmt *Delete) (interface{}, error)
	VisitUpdate(stmt *Update) (interface{}, error)
	VisitMerge(stmt *Merge) (interface{}, error)

	// DDL
	VisitCreatePrimaryIndex(stmt *CreatePrimaryIndex) (interface{}, error)
	VisitCreateIndex(stmt *CreateIndex) (interface{}, error)
	VisitDropIndex(stmt *DropIndex) (interface{}, error)
	VisitAlterIndex(stmt *AlterIndex) (interface{}, error)

	// EXPLAIN
	VisitExplain(stmt *Explain) (interface{}, error)

	// PREPARE
	VisitPrepare(stmt *Prepare) (interface{}, error)
}

type NodeVisitor interface {
	// SELECT
	VisitSubselect(node *Subselect) (interface{}, error)
	VisitKeyspaceTerm(node *KeyspaceTerm) (interface{}, error)
	VisitJoin(node *Join) (interface{}, error)
	VisitNest(node *Nest) (interface{}, error)
	VisitUnnest(node *Unnest) (interface{}, error)
	VisitUnion(node *Union) (interface{}, error)
	VisitUnionAll(node *UnionAll) (interface{}, error)
	VisitIntersect(node *Intersect) (interface{}, error)
	VisitIntersectAll(node *IntersectAll) (interface{}, error)
	VisitExcept(node *Except) (interface{}, error)
	VisitExceptAll(node *ExceptAll) (interface{}, error)
}
