//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	_ "fmt"
)

type Visitor interface {
	// Scan
	VisitEqualScan(op *EqualScan) (interface{}, error)
	VisitRangeScan(op *RangeScan) (interface{}, error)
	VisitDualScan(op *DualScan) (interface{}, error)
	VisitKeyScan(op *KeyScan) (interface{}, error)
	VisitValueScan(op *ValueScan) (interface{}, error)

	// Scatter
	VisitScatter(op *Scatter) (interface{}, error)

	// Sequence
	VisitSequence(op *Sequence) (interface{}, error)

	// Fetch
	VisitFetch(op *Fetch) (interface{}, error)

	// Join
	VisitJoin(op *Join) (interface{}, error)
	VisitNest(op *Nest) (interface{}, error)
	VisitUnnest(op *Unnest) (interface{}, error)

	// Filter
	VisitFilter(op *Filter) (interface{}, error)

	// Group
	VisitInitialGroup(op *InitialGroup) (interface{}, error)
	VisitIntermediateGroup(op *IntermediateGroup) (interface{}, error)
	VisitFinalGroup(op *FinalGroup) (interface{}, error)

	// Precompute
	VisitPrecompute(op *Precompute) (interface{}, error)

	// Project
	VisitProject(op *Project) (interface{}, error)

	// Distinct
	VisitInitialDistinct(op *InitialDistinct) (interface{}, error)
	VisitSubsequentDistinct(op *SubsequentDistinct) (interface{}, error)

	// Order
	VisitOrder(op *Order) (interface{}, error)

	// Offset
	VisitOffset(op *Offset) (interface{}, error)
	VisitLimit(op *Limit) (interface{}, error)

	// Insert
	VisitInsert(op *Insert) (interface{}, error)

	// Delete
	VisitDelete(op *Delete) (interface{}, error)

	// Update
	VisitCopy(op *Copy) (interface{}, error)
	VisitSet(op *Set) (interface{}, error)
	VisitUnset(op *Unset) (interface{}, error)
	VisitUpdate(op *Update) (interface{}, error)

	// Merge
	/*
		VisitJoin(op *Join) (interface{}, error)
		VisitJoin(op *Join) (interface{}, error)
		VisitJoin(op *Join) (interface{}, error)
		VisitJoin(op *Join) (interface{}, error)
	*/
	VisitMerge(op *Merge) (interface{}, error)
}
