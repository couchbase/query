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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
)

// Build a query execution pipeline from a query plan.
func Build(plan plan.Operator, context *Context) (Operator, error) {
	var m map[scannedIndex]bool
	aliasMap := make(map[string]string, 8)
	if context.ScanVectorSource().Type() == timestamp.ONE_VECTOR {
		// Collect scanned indexes.
		m = make(map[scannedIndex]bool, 8)
	}
	builder := &builder{context, m, aliasMap}
	x, err := plan.Accept(builder)

	if err != nil {
		return nil, err
	}

	if len(builder.scannedIndexes) > 1 {
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
	aliasMap       map[string]string
}

func (this *builder) setAliasMap(keyspaceTerm *algebra.KeyspaceTerm) {
	if keyspaceTerm.Namespace() != "#system" {
		this.aliasMap[keyspaceTerm.Alias()] = keyspaceTerm.Path().ProtectedString()
	}
}

// Remember the bucket of the scanned index.
func (this *builder) setScannedIndexes(keyspaceTerm *algebra.KeyspaceTerm) {
	if this.scannedIndexes != nil {
		scannedIndex := scannedIndex{keyspaceTerm.Namespace(), keyspaceTerm.Keyspace()}
		this.scannedIndexes[scannedIndex] = true
	}
}

// Scan
func (this *builder) VisitPrimaryScan(plan *plan.PrimaryScan) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewPrimaryScan(plan, this.context), nil
}

func (this *builder) VisitPrimaryScan3(plan *plan.PrimaryScan3) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewPrimaryScan3(plan, this.context), nil
}

func (this *builder) VisitParentScan(plan *plan.ParentScan) (interface{}, error) {
	return NewParentScan(plan, this.context), nil
}

func (this *builder) VisitIndexScan(plan *plan.IndexScan) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexScan(plan, this.context), nil
}

func (this *builder) VisitIndexScan2(plan *plan.IndexScan2) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexScan2(plan, this.context), nil
}

func (this *builder) VisitIndexScan3(plan *plan.IndexScan3) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexScan3(plan, this.context), nil
}

func (this *builder) VisitIndexCountScan(plan *plan.IndexCountScan) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexCountScan(plan, this.context), nil
}

func (this *builder) VisitIndexCountScan2(plan *plan.IndexCountScan2) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexCountScan2(plan, this.context), nil
}

func (this *builder) VisitIndexCountDistinctScan2(plan *plan.IndexCountDistinctScan2) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexCountDistinctScan2(plan, this.context), nil
}

func (this *builder) VisitKeyScan(plan *plan.KeyScan) (interface{}, error) {
	return NewKeyScan(plan, this.context), nil
}

func (this *builder) VisitExpressionScan(plan *plan.ExpressionScan) (interface{}, error) {
	return NewExpressionScan(plan, this.context), nil
}

func (this *builder) VisitValueScan(plan *plan.ValueScan) (interface{}, error) {
	return NewValueScan(plan, this.context), nil
}

func (this *builder) VisitDummyScan(plan *plan.DummyScan) (interface{}, error) {
	return NewDummyScan(plan, this.context), nil
}

func (this *builder) VisitCountScan(plan *plan.CountScan) (interface{}, error) {
	return NewCountScan(plan, this.context), nil
}

func (this *builder) VisitDistinctScan(plan *plan.DistinctScan) (interface{}, error) {
	scan, err := plan.Scan().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewDistinctScan(plan, this.context, scan.(Operator)), nil
}

func (this *builder) VisitUnionScan(plan *plan.UnionScan) (interface{}, error) {
	scans := _INDEX_SCAN_POOL.Get()

	for _, p := range plan.Scans() {
		s, e := p.Accept(this)
		if e != nil {
			return nil, e
		}

		scans = append(scans, s.(Operator))
	}

	return NewUnionScan(plan, this.context, scans), nil
}

func (this *builder) VisitIntersectScan(plan *plan.IntersectScan) (interface{}, error) {
	scans := _INDEX_SCAN_POOL.Get()

	for _, p := range plan.Scans() {
		s, e := p.Accept(this)
		if e != nil {
			return nil, e
		}

		scans = append(scans, s.(Operator))
	}

	return NewIntersectScan(plan, this.context, scans), nil
}

func (this *builder) VisitOrderedIntersectScan(plan *plan.OrderedIntersectScan) (interface{}, error) {
	scans := _INDEX_SCAN_POOL.Get()

	for _, p := range plan.Scans() {
		s, e := p.Accept(this)
		if e != nil {
			return nil, e
		}

		scans = append(scans, s.(Operator))
	}

	return NewOrderedIntersectScan(plan, this.context, scans), nil
}

// Fetch
func (this *builder) VisitFetch(plan *plan.Fetch) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return NewFetch(plan, this.context), nil
}

// DummyFetch
func (this *builder) VisitDummyFetch(plan *plan.DummyFetch) (interface{}, error) {
	return NewDummyFetch(plan, this.context), nil
}

// Join
func (this *builder) VisitJoin(plan *plan.Join) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return NewJoin(plan, this.context), nil
}

func (this *builder) VisitIndexJoin(plan *plan.IndexJoin) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return NewIndexJoin(plan, this.context), nil
}

func (this *builder) VisitNLJoin(plan *plan.NLJoin) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return NewNLJoin(plan, this.context, c.(Operator), this.aliasMap), nil
}

func (this *builder) VisitHashJoin(plan *plan.HashJoin) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return NewHashJoin(plan, this.context, c.(Operator), this.aliasMap), nil
}

func (this *builder) VisitNest(plan *plan.Nest) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return NewNest(plan, this.context), nil
}

func (this *builder) VisitIndexNest(plan *plan.IndexNest) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return NewIndexNest(plan, this.context), nil
}

func (this *builder) VisitNLNest(plan *plan.NLNest) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return NewNLNest(plan, this.context, c.(Operator), this.aliasMap), nil
}

func (this *builder) VisitHashNest(plan *plan.HashNest) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return NewHashNest(plan, this.context, c.(Operator), this.aliasMap), nil
}

func (this *builder) VisitUnnest(plan *plan.Unnest) (interface{}, error) {
	return NewUnnest(plan, this.context), nil
}

// Let + Letting
func (this *builder) VisitLet(plan *plan.Let) (interface{}, error) {
	return NewLet(plan, this.context), nil
}

// With
func (this *builder) VisitWith(plan *plan.With) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}
	return NewWith(plan, this.context, c.(Operator)), nil
}

// Filter
func (this *builder) VisitFilter(plan *plan.Filter) (interface{}, error) {
	return NewFilter(plan, this.context, this.aliasMap), nil
}

// Group
func (this *builder) VisitInitialGroup(plan *plan.InitialGroup) (interface{}, error) {
	return NewInitialGroup(plan, this.context), nil
}

func (this *builder) VisitIntermediateGroup(plan *plan.IntermediateGroup) (interface{}, error) {
	return NewIntermediateGroup(plan, this.context), nil
}

func (this *builder) VisitFinalGroup(plan *plan.FinalGroup) (interface{}, error) {
	return NewFinalGroup(plan, this.context), nil
}

// Window functions
func (this *builder) VisitWindowAggregate(plan *plan.WindowAggregate) (interface{}, error) {
	return NewWindowAggregate(plan, this.context), nil
}

// Project
func (this *builder) VisitInitialProject(plan *plan.InitialProject) (interface{}, error) {
	return NewInitialProject(plan, this.context), nil
}

func (this *builder) VisitFinalProject(plan *plan.FinalProject) (interface{}, error) {
	return NewFinalProject(plan, this.context), nil
}

func (this *builder) VisitIndexCountProject(plan *plan.IndexCountProject) (interface{}, error) {
	return NewIndexCountProject(plan, this.context), nil
}

// Distinct
func (this *builder) VisitDistinct(plan *plan.Distinct) (interface{}, error) {
	return NewDistinct(plan, this.context, false), nil
}

// Set operators
func (this *builder) VisitUnionAll(plan *plan.UnionAll) (interface{}, error) {
	children := _UNION_POOL.Get()

	saliasMap := this.aliasMap
	defer func() { this.aliasMap = saliasMap }()

	for _, child := range plan.Children() {
		this.aliasMap = make(map[string]string, 8)
		c, e := child.Accept(this)
		if e != nil {
			return nil, e
		}

		children = append(children, c.(Operator))
	}

	return NewUnionAll(plan, this.context, children...), nil
}

func (this *builder) VisitIntersectAll(plan *plan.IntersectAll) (interface{}, error) {
	saliasMap := this.aliasMap
	defer func() { this.aliasMap = saliasMap }()
	this.aliasMap = make(map[string]string, 8)

	first, e := plan.First().Accept(this)
	if e != nil {
		return nil, e
	}

	this.aliasMap = make(map[string]string, 8)
	second, e := plan.Second().Accept(this)
	if e != nil {
		return nil, e
	}

	return NewIntersectAll(plan, this.context, first.(Operator), second.(Operator)), nil
}

func (this *builder) VisitExceptAll(plan *plan.ExceptAll) (interface{}, error) {
	saliasMap := this.aliasMap
	defer func() { this.aliasMap = saliasMap }()
	this.aliasMap = make(map[string]string, 8)

	first, e := plan.First().Accept(this)
	if e != nil {
		return nil, e
	}

	this.aliasMap = make(map[string]string, 8)
	second, e := plan.Second().Accept(this)
	if e != nil {
		return nil, e
	}

	return NewExceptAll(plan, this.context, first.(Operator), second.(Operator)), nil
}

// Order
func (this *builder) VisitOrder(plan *plan.Order) (interface{}, error) {
	if plan.LimitPushed() {
		return NewOrderLimit(plan, this.context), nil
	} else {
		return NewOrder(plan, this.context), nil
	}
}

// Offset
func (this *builder) VisitOffset(plan *plan.Offset) (interface{}, error) {
	return NewOffset(plan, this.context), nil
}

func (this *builder) VisitLimit(plan *plan.Limit) (interface{}, error) {
	return NewLimit(plan, this.context), nil
}

// Insert
func (this *builder) VisitSendInsert(plan *plan.SendInsert) (interface{}, error) {
	return NewSendInsert(plan, this.context), nil
}

// Upsert
func (this *builder) VisitSendUpsert(plan *plan.SendUpsert) (interface{}, error) {
	return NewSendUpsert(plan, this.context), nil
}

// Delete
func (this *builder) VisitSendDelete(plan *plan.SendDelete) (interface{}, error) {
	return NewSendDelete(plan, this.context), nil
}

// Update
func (this *builder) VisitClone(plan *plan.Clone) (interface{}, error) {
	return NewClone(plan, this.context), nil
}

func (this *builder) VisitSet(plan *plan.Set) (interface{}, error) {
	return NewSet(plan, this.context), nil
}

func (this *builder) VisitUnset(plan *plan.Unset) (interface{}, error) {
	return NewUnset(plan, this.context), nil
}

func (this *builder) VisitSendUpdate(plan *plan.SendUpdate) (interface{}, error) {
	return NewSendUpdate(plan, this.context), nil
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

	return NewMerge(plan, this.context, update, delete, insert), nil
}

// Alias
func (this *builder) VisitAlias(plan *plan.Alias) (interface{}, error) {
	return NewAlias(plan, this.context), nil
}

// Authorize
func (this *builder) VisitAuthorize(plan *plan.Authorize) (interface{}, error) {
	child, err := plan.Child().Accept(this)
	if err != nil {
		return nil, err
	}

	return NewAuthorize(plan, this.context, child.(Operator)), nil
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
		return NewParallel(plan, this.context, child.(Operator)), nil
	}
}

// Sequence
func (this *builder) VisitSequence(plan *plan.Sequence) (interface{}, error) {
	children := plan.Children()

	if len(children) == 1 {
		child, err := children[0].Accept(this)
		if err != nil {
			return nil, err
		}
		return child.(Operator), nil
	}

	execChildren := _SEQUENCE_POOL.Get()

	for _, pchild := range children {
		child, err := pchild.Accept(this)
		if err != nil {
			return nil, err
		}

		execChildren = append(execChildren, child.(Operator))
	}
	return NewSequence(plan, this.context, execChildren...), nil
}

// Discard
func (this *builder) VisitDiscard(plan *plan.Discard) (interface{}, error) {
	return NewDiscard(plan, this.context), nil
}

// Stream
func (this *builder) VisitStream(plan *plan.Stream) (interface{}, error) {
	return NewStream(plan, this.context), nil
}

// Collect
func (this *builder) VisitCollect(plan *plan.Collect) (interface{}, error) {
	return NewCollect(plan, this.context), nil
}

// CreateIndex
func (this *builder) VisitCreatePrimaryIndex(plan *plan.CreatePrimaryIndex) (interface{}, error) {
	return NewCreatePrimaryIndex(plan, this.context), nil
}

// GrantRole
func (this *builder) VisitGrantRole(plan *plan.GrantRole) (interface{}, error) {
	return NewGrantRole(plan, this.context), nil
}

// RevokeRole
func (this *builder) VisitRevokeRole(plan *plan.RevokeRole) (interface{}, error) {
	return NewRevokeRole(plan, this.context), nil
}

// CreateIndex
func (this *builder) VisitCreateIndex(plan *plan.CreateIndex) (interface{}, error) {
	return NewCreateIndex(plan, this.context), nil
}

// DropIndex
func (this *builder) VisitDropIndex(plan *plan.DropIndex) (interface{}, error) {
	return NewDropIndex(plan, this.context), nil
}

// AlterIndex
func (this *builder) VisitAlterIndex(plan *plan.AlterIndex) (interface{}, error) {
	return NewAlterIndex(plan, this.context), nil
}

// BuildIndexes
func (this *builder) VisitBuildIndexes(plan *plan.BuildIndexes) (interface{}, error) {
	return NewBuildIndexes(plan, this.context), nil
}

// Prepare
func (this *builder) VisitPrepare(plan *plan.Prepare) (interface{}, error) {
	return NewPrepare(plan, this.context, plan.Prepared()), nil
}

// Explain
func (this *builder) VisitExplain(plan *plan.Explain) (interface{}, error) {
	return NewExplain(plan, this.context), nil
}

// Infer
func (this *builder) VisitInferKeyspace(plan *plan.InferKeyspace) (interface{}, error) {
	return NewInferKeyspace(plan, this.context), nil
}

// CreateFunction
func (this *builder) VisitCreateFunction(plan *plan.CreateFunction) (interface{}, error) {
	return NewCreateFunction(plan, this.context), nil
}

// DropFunction
func (this *builder) VisitDropFunction(plan *plan.DropFunction) (interface{}, error) {
	return NewDropFunction(plan, this.context), nil
}

// ExecuteFunction
func (this *builder) VisitExecuteFunction(plan *plan.ExecuteFunction) (interface{}, error) {
	return NewExecuteFunction(plan, this.context), nil
}
func (this *builder) VisitIndexFtsSearch(plan *plan.IndexFtsSearch) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return NewIndexFtsSearch(plan, this.context), nil
}
