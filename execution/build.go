//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

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
		panic(fmt.Sprintf("Expected execution.Operator instead of %T.", ex))
	}
}

type Builder struct {
}

// Scan
func (this *Builder) VisitPrimaryScan(plan *plan.PrimaryScan) (interface{}, error) {
	return NewPrimaryScan(plan), nil
}

func (this *Builder) VisitParentScan(plan *plan.ParentScan) (interface{}, error) {
	return NewParentScan(), nil
}

func (this *Builder) VisitIndexScan(plan *plan.IndexScan) (interface{}, error) {
	return NewIndexScan(plan), nil
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

func (this *Builder) VisitCountScan(plan *plan.CountScan) (interface{}, error) {
	return NewCountScan(plan), nil
}

func (this *Builder) VisitIntersectScan(plan *plan.IntersectScan) (interface{}, error) {
	scans := make([]Operator, len(plan.Scans()))

	for i, p := range plan.Scans() {
		s, e := p.Accept(this)
		if e != nil {
			return nil, e
		}

		scans[i] = s.(Operator)
	}

	return NewIntersectScan(scans), nil
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

// Let + Letting
func (this *Builder) VisitLet(plan *plan.Let) (interface{}, error) {
	return NewLet(plan), nil
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

// Project
func (this *Builder) VisitInitialProject(plan *plan.InitialProject) (interface{}, error) {
	return NewInitialProject(plan), nil
}

func (this *Builder) VisitFinalProject(plan *plan.FinalProject) (interface{}, error) {
	return NewFinalProject(), nil
}

// Distinct
func (this *Builder) VisitDistinct(plan *plan.Distinct) (interface{}, error) {
	return NewDistinct(false), nil
}

// Set operators
func (this *Builder) VisitUnionAll(plan *plan.UnionAll) (interface{}, error) {
	children := make([]Operator, len(plan.Children()))
	for i, child := range plan.Children() {
		c, e := child.Accept(this)
		if e != nil {
			return nil, e
		}

		children[i] = c.(Operator)
	}

	return NewUnionAll(children...), nil
}

func (this *Builder) VisitIntersectAll(plan *plan.IntersectAll) (interface{}, error) {
	first, e := plan.First().Accept(this)
	if e != nil {
		return nil, e
	}

	second, e := plan.Second().Accept(this)
	if e != nil {
		return nil, e
	}

	return NewIntersectAll(first.(Operator), second.(Operator)), nil
}

func (this *Builder) VisitExceptAll(plan *plan.ExceptAll) (interface{}, error) {
	first, e := plan.First().Accept(this)
	if e != nil {
		return nil, e
	}

	second, e := plan.Second().Accept(this)
	if e != nil {
		return nil, e
	}

	return NewExceptAll(first.(Operator), second.(Operator)), nil
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

// Upsert
func (this *Builder) VisitSendUpsert(plan *plan.SendUpsert) (interface{}, error) {
	return NewSendUpsert(plan), nil
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
func (this *Builder) VisitMerge(plan *plan.Merge) (interface{}, error) {
	var update, delete, insert Operator

	if plan.Update() != nil {
		op, e := plan.Update().Accept(this)
		if e != nil {
			return nil, e
		}
		update = op.(Operator)
	}

	if plan.Delete() != nil {
		op, e := plan.Delete().Accept(this)
		if e != nil {
			return nil, e
		}
		delete = op.(Operator)
	}

	if plan.Insert() != nil {
		op, e := plan.Insert().Accept(this)
		if e != nil {
			return nil, e
		}
		insert = op.(Operator)
	}

	return NewMerge(plan, update, delete, insert), nil
}

func (this *Builder) VisitAlias(plan *plan.Alias) (interface{}, error) {
	return NewAlias(plan), nil
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

// Channel
func (this *Builder) VisitChannel(plan *plan.Channel) (interface{}, error) {
	return NewChannel(), nil
}

// CreateIndex
func (this *Builder) VisitCreateIndex(plan *plan.CreateIndex) (interface{}, error) {
	return NewCreateIndex(plan), nil
}

// DropIndex
func (this *Builder) VisitDropIndex(plan *plan.DropIndex) (interface{}, error) {
	return NewDropIndex(plan), nil
}

// AlterIndex
func (this *Builder) VisitAlterIndex(plan *plan.AlterIndex) (interface{}, error) {
	return NewAlterIndex(plan), nil
}

// Explain
func (this *Builder) VisitExplain(plan *plan.Explain) (interface{}, error) {
	return NewExplain(plan.Operator()), nil
}
