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
	VisitSelect(node *Select) (interface{}, error)
	VisitSubselect(node *Subselect) (interface{}, error)
	VisitUnion(node *Union) (interface{}, error)
	VisitUnionAll(node *UnionAll) (interface{}, error)
	VisitBucketTerm(node *BucketTerm) (interface{}, error)
	VisitParentTerm(node *ParentTerm) (interface{}, error)
	VisitJoin(node *Join) (interface{}, error)
	VisitNest(node *Nest) (interface{}, error)
	VisitUnnest(node *Unnest) (interface{}, error)

	// DML
	VisitInsert(node *Insert) (interface{}, error)
	VisitUpsert(node *Upsert) (interface{}, error)
	VisitDelete(node *Delete) (interface{}, error)
	VisitUpdate(node *Update) (interface{}, error)
	VisitMerge(node *Merge) (interface{}, error)

	// DDL
	VisitCreateIndex(node *CreateIndex) (interface{}, error)
	VisitDropIndex(node *DropIndex) (interface{}, error)
	VisitAlterIndex(node *AlterIndex) (interface{}, error)

	// EXPLAIN
	VisitExplain(node *Explain) (interface{}, error)
}
