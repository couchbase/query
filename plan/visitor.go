//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

type Visitor interface {
	// Scan
	VisitPrimaryScan(op *PrimaryScan) (interface{}, error)
	VisitParentScan(op *ParentScan) (interface{}, error)
	VisitIndexScan(op *IndexScan) (interface{}, error)
	VisitKeyScan(op *KeyScan) (interface{}, error)
	VisitValueScan(op *ValueScan) (interface{}, error)
	VisitDummyScan(op *DummyScan) (interface{}, error)
	VisitCountScan(op *CountScan) (interface{}, error)
	VisitIndexCountScan(op *IndexCountScan) (interface{}, error)
	VisitDistinctScan(op *DistinctScan) (interface{}, error)
	VisitUnionScan(op *UnionScan) (interface{}, error)
	VisitIntersectScan(op *IntersectScan) (interface{}, error)
	VisitOrderedIntersectScan(op *OrderedIntersectScan) (interface{}, error)
	VisitExpressionScan(op *ExpressionScan) (interface{}, error)

	// Fetch
	VisitFetch(op *Fetch) (interface{}, error)
	VisitDummyFetch(op *DummyFetch) (interface{}, error)

	// Join
	VisitJoin(op *Join) (interface{}, error)
	VisitIndexJoin(op *IndexJoin) (interface{}, error)
	VisitNest(op *Nest) (interface{}, error)
	VisitIndexNest(op *IndexNest) (interface{}, error)
	VisitUnnest(op *Unnest) (interface{}, error)

	// Let + Letting
	VisitLet(op *Let) (interface{}, error)

	// Filter
	VisitFilter(op *Filter) (interface{}, error)

	// Group
	VisitInitialGroup(op *InitialGroup) (interface{}, error)
	VisitIntermediateGroup(op *IntermediateGroup) (interface{}, error)
	VisitFinalGroup(op *FinalGroup) (interface{}, error)

	// Project
	VisitInitialProject(op *InitialProject) (interface{}, error)
	VisitFinalProject(op *FinalProject) (interface{}, error)
	VisitIndexCountProject(op *IndexCountProject) (interface{}, error)

	// Distinct
	VisitDistinct(op *Distinct) (interface{}, error)

	// Set operators
	VisitUnionAll(op *UnionAll) (interface{}, error)
	VisitIntersectAll(op *IntersectAll) (interface{}, error)
	VisitExceptAll(op *ExceptAll) (interface{}, error)

	// Order
	VisitOrder(op *Order) (interface{}, error)

	// Paging
	VisitOffset(op *Offset) (interface{}, error)
	VisitLimit(op *Limit) (interface{}, error)

	// Insert
	VisitSendInsert(op *SendInsert) (interface{}, error)

	// Upsert
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

	// Index DDL
	VisitCreatePrimaryIndex(op *CreatePrimaryIndex) (interface{}, error)
	VisitCreateIndex(op *CreateIndex) (interface{}, error)
	VisitDropIndex(op *DropIndex) (interface{}, error)
	VisitAlterIndex(op *AlterIndex) (interface{}, error)
	VisitBuildIndexes(op *BuildIndexes) (interface{}, error)

	// Explain
	VisitExplain(op *Explain) (interface{}, error)

	// Prepare
	VisitPrepare(op *Prepare) (interface{}, error)

	// Infer
	VisitInferKeyspace(op *InferKeyspace) (interface{}, error)
}
