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
	"fmt"

	"github.com/couchbaselabs/query/plan"
)

// Build a query execution pipeline from a query plan.
func Build(plan plan.Operator) (Operator, error) {
	builder := &Builder{}
	ex, err := plan.Accept(builder)

	if err != nil {
		return nil, err
	}

	switch ex := ex.(type) {
	case Operator:
		return ex, nil
	default:
		panic(fmt.Sprintf("Expected execute.Operator instead of %T.", ex))
	}
}

type Builder struct {
}

// Scan
func (this *Builder) VisitFullScan(plan *plan.FullScan) (interface{}, error) {
	return NewFullScan(plan), nil
}

func (this *Builder) VisitParentScan(plan *plan.ParentScan) (interface{}, error) {
	return NewParentScan(), nil
}

func (this *Builder) VisitEqualScan(plan *plan.EqualScan) (interface{}, error) {
	return NewEqualScan(plan), nil
}

func (this *Builder) VisitRangeScan(plan *plan.RangeScan) (interface{}, error) {
	return NewRangeScan(plan), nil
}

func (this *Builder) VisitDualScan(plan *plan.DualScan) (interface{}, error) {
	return NewDualScan(plan), nil
}

func (this *Builder) VisitKeyScan(plan *plan.KeyScan) (interface{}, error) {
	return NewKeyScan(plan), nil
}

func (this *Builder) VisitValueScan(plan *plan.ValueScan) (interface{}, error) {
	return NewValueScan(plan), nil
}

func (this *Builder) VisitDummyScan(plan *plan.DummyScan) (interface{}, error) {
	return NewDummyScan(), nil
}

// Fetch
func (this *Builder) VisitFetch(plan *plan.Fetch) (interface{}, error) {
	return NewFetch(plan), nil
}

// Join
func (this *Builder) VisitJoin(plan *plan.Join) (interface{}, error) {
	return NewJoin(plan), nil
}

func (this *Builder) VisitNest(plan *plan.Nest) (interface{}, error) {
	return NewNest(plan), nil
}

func (this *Builder) VisitUnnest(plan *plan.Unnest) (interface{}, error) {
	return NewUnnest(plan), nil
}

// Filter
func (this *Builder) VisitFilter(plan *plan.Filter) (interface{}, error) {
	return NewFilter(plan), nil
}

// Group
func (this *Builder) VisitInitialGroup(plan *plan.InitialGroup) (interface{}, error) {
	return NewInitialGroup(plan), nil
}

func (this *Builder) VisitIntermediateGroup(plan *plan.IntermediateGroup) (interface{}, error) {
	return NewIntermediateGroup(plan), nil
}

func (this *Builder) VisitFinalGroup(plan *plan.FinalGroup) (interface{}, error) {
	return NewFinalGroup(plan), nil
}

// Precompute
func (this *Builder) VisitPrecompute(plan *plan.Precompute) (interface{}, error) {
	return NewPrecompute(plan), nil
}

// Project
func (this *Builder) VisitProject(plan *plan.Project) (interface{}, error) {
	return NewProject(plan), nil
}

// Distinct
func (this *Builder) VisitInitialDistinct(plan *plan.InitialDistinct) (interface{}, error) {
	return NewInitialDistinct(), nil
}

func (this *Builder) VisitSubsequentDistinct(plan *plan.SubsequentDistinct) (interface{}, error) {
	return NewSubsequentDistinct(), nil
}

// Order
func (this *Builder) VisitOrder(plan *plan.Order) (interface{}, error) {
	return NewOrder(plan), nil
}

// Offset
func (this *Builder) VisitOffset(plan *plan.Offset) (interface{}, error) {
	return NewOffset(plan), nil
}

func (this *Builder) VisitLimit(plan *plan.Limit) (interface{}, error) {
	return NewLimit(plan), nil
}

// Insert
func (this *Builder) VisitSendInsert(plan *plan.SendInsert) (interface{}, error) {
	return NewSendInsert(plan), nil
}

// Delete
func (this *Builder) VisitSendDelete(plan *plan.SendDelete) (interface{}, error) {
	return NewSendDelete(plan), nil
}

// Update
func (this *Builder) VisitClone(plan *plan.Clone) (interface{}, error) {
	return NewClone(), nil
}

func (this *Builder) VisitSet(plan *plan.Set) (interface{}, error) {
	return NewSet(plan), nil
}

func (this *Builder) VisitUnset(plan *plan.Unset) (interface{}, error) {
	return NewUnset(plan), nil
}

func (this *Builder) VisitSendUpdate(plan *plan.SendUpdate) (interface{}, error) {
	return NewSendUpdate(plan), nil
}

// Merge
func (this *Builder) VisitComputeMerge(plan *plan.ComputeMerge) (interface{}, error) {
	return NewComputeMerge(plan), nil
}

func (this *Builder) VisitMergeUpdate(plan *plan.MergeUpdate) (interface{}, error) {
	return NewMergeUpdate(plan), nil
}

func (this *Builder) VisitMergeDelete(plan *plan.MergeDelete) (interface{}, error) {
	return NewMergeDelete(plan), nil
}

func (this *Builder) VisitMergeInsert(plan *plan.MergeInsert) (interface{}, error) {
	return NewMergeInsert(plan), nil
}

func (this *Builder) VisitSendMerge(plan *plan.SendMerge) (interface{}, error) {
	return NewSendMerge(plan), nil
}

// Parallel
func (this *Builder) VisitParallel(plan *plan.Parallel) (interface{}, error) {
	child, err := plan.Child().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewParallel(child.(Operator)), nil
}

// Sequence
func (this *Builder) VisitSequence(plan *plan.Sequence) (interface{}, error) {
	children := make([]Operator, len(plan.Children()))

	for i, pchild := range plan.Children() {
		child, err := pchild.Accept(this)
		if err != nil {
			return nil, err
		}

		children[i] = child.(Operator)
	}

	return NewSequence(children...), nil
}

// Discard
func (this *Builder) VisitDiscard(plan *plan.Discard) (interface{}, error) {
	return NewDiscard(), nil
}

// Stream
func (this *Builder) VisitStream(plan *plan.Stream) (interface{}, error) {
	return NewStream(), nil
}

// Collect
func (this *Builder) VisitCollect(plan *plan.Collect) (interface{}, error) {
	return NewCollect(), nil
}
