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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
)

// Build a query execution pipeline from a query plan.
func Build(plan plan.Operator, context *Context) (Operator, error) {
	var m map[scannedIndex]bool
	if context.ScanVectorSource().Type() == timestamp.ONE_VECTOR {
		// Collect scanned indexes.
		m = make(map[scannedIndex]bool)
	}
	builder := &builder{context, m}
	x, err := plan.Accept(builder)

	if err != nil {
		return nil, err
	}

	if builder.scannedIndexes != nil && len(builder.scannedIndexes) > 1 {
		scannedIndexArr := make([]string, len(builder.scannedIndexes))
		for si := range builder.scannedIndexes {
			scannedIndexArr = append(scannedIndexArr, fmt.Sprintf("%s:%s", si.namespace, si.keyspace))
		}
		return nil, errors.NewScanVectorTooManyScannedBuckets(scannedIndexArr)
	}

	ex := x.(Operator)
	return ex, nil
}

type scannedIndex struct {
	namespace string
	keyspace  string
}

type builder struct {
	context        *Context
	scannedIndexes map[scannedIndex]bool // Nil if scanned indexes should not be collected.
}

// Scan
func (this *builder) VisitPrimaryScan(plan *plan.PrimaryScan) (interface{}, error) {
	// Remember the bucket of the scanned index.
	if this.scannedIndexes != nil {
		keyspace := plan.Keyspace()
		scannedIndex := scannedIndex{keyspace.NamespaceId(), keyspace.Name()}
		this.scannedIndexes[scannedIndex] = true
	}

	return NewPrimaryScan(plan), nil
}

func (this *builder) VisitParentScan(plan *plan.ParentScan) (interface{}, error) {
	return NewParentScan(), nil
}

func (this *builder) VisitIndexScan(plan *plan.IndexScan) (interface{}, error) {
	// Remember the bucket of the scanned index.
	if this.scannedIndexes != nil {
		keyspaceTerm := plan.Term()
		scannedIndex := scannedIndex{keyspaceTerm.Namespace(), keyspaceTerm.Keyspace()}
		this.scannedIndexes[scannedIndex] = true
	}

	return NewIndexScan(plan), nil
}

func (this *builder) VisitIndexCountScan(plan *plan.IndexCountScan) (interface{}, error) {
	// Remember the bucket of the scanned index.
	if this.scannedIndexes != nil {
		keyspaceTerm := plan.Term()
		scannedIndex := scannedIndex{keyspaceTerm.Namespace(), keyspaceTerm.Keyspace()}
		this.scannedIndexes[scannedIndex] = true
	}

	return NewIndexCountScan(plan), nil
}

func (this *builder) VisitKeyScan(plan *plan.KeyScan) (interface{}, error) {
	return NewKeyScan(plan), nil
}

func (this *builder) VisitValueScan(plan *plan.ValueScan) (interface{}, error) {
	return NewValueScan(plan), nil
}

func (this *builder) VisitDummyScan(plan *plan.DummyScan) (interface{}, error) {
	return NewDummyScan(), nil
}

func (this *builder) VisitCountScan(plan *plan.CountScan) (interface{}, error) {
	return NewCountScan(plan), nil
}

func (this *builder) VisitIntersectScan(plan *plan.IntersectScan) (interface{}, error) {
	scans := _SCAN_POOL.Get()

	for _, p := range plan.Scans() {
		s, e := p.Accept(this)
		if e != nil {
			return nil, e
		}

		scans = append(scans, s.(Operator))
	}

	return NewIntersectScan(scans), nil
}

func (this *builder) VisitUnionScan(plan *plan.UnionScan) (interface{}, error) {
	scans := _SCAN_POOL.Get()

	for _, p := range plan.Scans() {
		s, e := p.Accept(this)
		if e != nil {
			return nil, e
		}

		scans = append(scans, s.(Operator))
	}

	return NewUnionScan(scans), nil
}

// Fetch
func (this *builder) VisitFetch(plan *plan.Fetch) (interface{}, error) {
	return NewFetch(plan), nil
}

// DummyFetch
func (this *builder) VisitDummyFetch(plan *plan.DummyFetch) (interface{}, error) {
	return NewDummyFetch(plan), nil
}

// Join
func (this *builder) VisitJoin(plan *plan.Join) (interface{}, error) {
	return NewJoin(plan), nil
}

func (this *builder) VisitIndexJoin(plan *plan.IndexJoin) (interface{}, error) {
	return NewIndexJoin(plan), nil
}

func (this *builder) VisitNest(plan *plan.Nest) (interface{}, error) {
	return NewNest(plan), nil
}

func (this *builder) VisitIndexNest(plan *plan.IndexNest) (interface{}, error) {
	return NewIndexNest(plan), nil
}

func (this *builder) VisitUnnest(plan *plan.Unnest) (interface{}, error) {
	return NewUnnest(plan), nil
}

// Let + Letting
func (this *builder) VisitLet(plan *plan.Let) (interface{}, error) {
	return NewLet(plan), nil
}

// Filter
func (this *builder) VisitFilter(plan *plan.Filter) (interface{}, error) {
	return NewFilter(plan), nil
}

// Group
func (this *builder) VisitInitialGroup(plan *plan.InitialGroup) (interface{}, error) {
	return NewInitialGroup(plan), nil
}

func (this *builder) VisitIntermediateGroup(plan *plan.IntermediateGroup) (interface{}, error) {
	return NewIntermediateGroup(plan), nil
}

func (this *builder) VisitFinalGroup(plan *plan.FinalGroup) (interface{}, error) {
	return NewFinalGroup(plan), nil
}

// Project
func (this *builder) VisitInitialProject(plan *plan.InitialProject) (interface{}, error) {
	return NewInitialProject(plan), nil
}

func (this *builder) VisitFinalProject(plan *plan.FinalProject) (interface{}, error) {
	return NewFinalProject(), nil
}

func (this *builder) VisitIndexCountProject(plan *plan.IndexCountProject) (interface{}, error) {
	return NewIndexCountProject(plan), nil
}

// Distinct
func (this *builder) VisitDistinct(plan *plan.Distinct) (interface{}, error) {
	return NewDistinct(plan, false), nil
}

// Set operators
func (this *builder) VisitUnionAll(plan *plan.UnionAll) (interface{}, error) {
	children := _UNION_POOL.Get()

	for _, child := range plan.Children() {
		c, e := child.Accept(this)
		if e != nil {
			return nil, e
		}

		children = append(children, c.(Operator))
	}

	return NewUnionAll(children...), nil
}

func (this *builder) VisitIntersectAll(plan *plan.IntersectAll) (interface{}, error) {
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

func (this *builder) VisitExceptAll(plan *plan.ExceptAll) (interface{}, error) {
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
func (this *builder) VisitOrder(plan *plan.Order) (interface{}, error) {
	if plan.LimitPushed() {
		return NewOrderLimit(plan), nil
	} else {
		return NewOrder(plan), nil
	}
}

// Offset
func (this *builder) VisitOffset(plan *plan.Offset) (interface{}, error) {
	return NewOffset(plan), nil
}

func (this *builder) VisitLimit(plan *plan.Limit) (interface{}, error) {
	return NewLimit(plan), nil
}

// Insert
func (this *builder) VisitSendInsert(plan *plan.SendInsert) (interface{}, error) {
	return NewSendInsert(plan), nil
}

// Upsert
func (this *builder) VisitSendUpsert(plan *plan.SendUpsert) (interface{}, error) {
	return NewSendUpsert(plan), nil
}

// Delete
func (this *builder) VisitSendDelete(plan *plan.SendDelete) (interface{}, error) {
	return NewSendDelete(plan), nil
}

// Update
func (this *builder) VisitClone(plan *plan.Clone) (interface{}, error) {
	return NewClone(plan), nil
}

func (this *builder) VisitSet(plan *plan.Set) (interface{}, error) {
	return NewSet(plan), nil
}

func (this *builder) VisitUnset(plan *plan.Unset) (interface{}, error) {
	return NewUnset(plan), nil
}

func (this *builder) VisitSendUpdate(plan *plan.SendUpdate) (interface{}, error) {
	return NewSendUpdate(plan), nil
}

// Merge
func (this *builder) VisitMerge(plan *plan.Merge) (interface{}, error) {
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

// Alias
func (this *builder) VisitAlias(plan *plan.Alias) (interface{}, error) {
	return NewAlias(plan), nil
}

// Authorize
func (this *builder) VisitAuthorize(plan *plan.Authorize) (interface{}, error) {
	child, err := plan.Child().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewAuthorize(plan, child.(Operator)), nil
}

// Parallel
func (this *builder) VisitParallel(plan *plan.Parallel) (interface{}, error) {
	child, err := plan.Child().Accept(this)
	if err != nil {
		return nil, err
	}

	maxParallelism := util.MinInt(plan.MaxParallelism(), this.context.MaxParallelism())

	if maxParallelism == 1 {
		return child, nil
	} else {
		return NewParallel(plan, child.(Operator)), nil
	}
}

// Sequence
func (this *builder) VisitSequence(plan *plan.Sequence) (interface{}, error) {
	children := _SEQUENCE_POOL.Get()

	for _, pchild := range plan.Children() {
		child, err := pchild.Accept(this)
		if err != nil {
			return nil, err
		}

		children = append(children, child.(Operator))
	}

	return NewSequence(children...), nil
}

// Discard
func (this *builder) VisitDiscard(plan *plan.Discard) (interface{}, error) {
	return NewDiscard(), nil
}

// Stream
func (this *builder) VisitStream(plan *plan.Stream) (interface{}, error) {
	return NewStream(), nil
}

// Collect
func (this *builder) VisitCollect(plan *plan.Collect) (interface{}, error) {
	return NewCollect(), nil
}

// Channel
func (this *builder) VisitChannel(plan *plan.Channel) (interface{}, error) {
	return NewChannel(), nil
}

// CreateIndex
func (this *builder) VisitCreatePrimaryIndex(plan *plan.CreatePrimaryIndex) (interface{}, error) {
	return NewCreatePrimaryIndex(plan), nil
}

// CreateIndex
func (this *builder) VisitCreateIndex(plan *plan.CreateIndex) (interface{}, error) {
	return NewCreateIndex(plan), nil
}

// DropIndex
func (this *builder) VisitDropIndex(plan *plan.DropIndex) (interface{}, error) {
	return NewDropIndex(plan), nil
}

// AlterIndex
func (this *builder) VisitAlterIndex(plan *plan.AlterIndex) (interface{}, error) {
	return NewAlterIndex(plan), nil
}

// BuildIndexes
func (this *builder) VisitBuildIndexes(plan *plan.BuildIndexes) (interface{}, error) {
	return NewBuildIndexes(plan), nil
}

// Prepare
func (this *builder) VisitPrepare(plan *plan.Prepare) (interface{}, error) {
	return NewPrepare(plan.Prepared()), nil
}

// Explain
func (this *builder) VisitExplain(plan *plan.Explain) (interface{}, error) {
	return NewExplain(plan), nil
}

// Infer
func (this *builder) VisitInferKeyspace(plan *plan.InferKeyspace) (interface{}, error) {
	return NewInferKeyspace(plan), nil
}
