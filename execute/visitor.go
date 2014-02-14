//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	_ "fmt"
)

type Visitor interface {
	// Scan
	VisitPrimaryScan(op *PrimaryScan) (interface{}, error)
	VisitParentScan(op *ParentScan) (interface{}, error)
	VisitEqualScan(op *EqualScan) (interface{}, error)
	VisitRangeScan(op *RangeScan) (interface{}, error)
	VisitDualScan(op *DualScan) (interface{}, error)
	VisitKeyScan(op *KeyScan) (interface{}, error)
	VisitValueScan(op *ValueScan) (interface{}, error)
	VisitDummyScan(op *DummyScan) (interface{}, error)
	VisitCountScan(op *CountScan) (interface{}, error)

	// Fetch
	VisitFetch(op *Fetch) (interface{}, error)

	// Join
	VisitJoin(op *Join) (interface{}, error)
	VisitNest(op *Nest) (interface{}, error)
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

	// Distinct
	VisitDistinct(op *Distinct) (interface{}, error)

	// Order
	VisitOrder(op *Order) (interface{}, error)

	// Offset
	VisitOffset(op *Offset) (interface{}, error)
	VisitLimit(op *Limit) (interface{}, error)

	// Insert
	VisitSendInsert(op *SendInsert) (interface{}, error)

	// Delete
	VisitSendDelete(op *SendDelete) (interface{}, error)

	// Update
	VisitClone(op *Clone) (interface{}, error)
	VisitSet(op *Set) (interface{}, error)
	VisitUnset(op *Unset) (interface{}, error)
	VisitSendUpdate(op *SendUpdate) (interface{}, error)

	// Merge
	VisitMerge(op *Merge) (interface{}, error)
	VisitAlias(op *Alias) (interface{}, error)

	// Framework
	VisitParallel(op *Parallel) (interface{}, error)
	VisitSequence(op *Sequence) (interface{}, error)
	VisitDiscard(op *Discard) (interface{}, error)
	VisitStream(op *Stream) (interface{}, error)
	VisitCollect(op *Collect) (interface{}, error)
}
