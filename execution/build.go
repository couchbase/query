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
	builder := &builder{context, m, aliasMap, true}
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
	context          *Context
	scannedIndexes   map[scannedIndex]bool // Nil if scanned indexes should not be collected.
	aliasMap         map[string]string
	dynamicAuthorize bool
}

func (this *builder) setAliasMap(keyspaceTerm *algebra.KeyspaceTerm) {
	if !keyspaceTerm.IsSystem() {
		path := keyspaceTerm.Path()
		if path == nil {
			path, _ = getKeyspacePath(keyspaceTerm.FromExpression(), this.context)
			if path != nil {
				this.aliasMap[keyspaceTerm.Alias()] = path.ProtectedString()
			}
		} else {
			this.aliasMap[keyspaceTerm.Alias()] = keyspaceTerm.PathString()
		}
	}
}

// Remember the bucket of the scanned index.
func (this *builder) setScannedIndexes(keyspaceTerm *algebra.KeyspaceTerm) {
	if this.scannedIndexes != nil {
		scannedIndex := scannedIndex{keyspaceTerm.Namespace(), keyspaceTerm.Keyspace()}
		this.scannedIndexes[scannedIndex] = true
	}
}

// assert correctness of operators
func checkOp(op interface{}, context *Context) (interface{}, error) {
	if context.assert(op != nil, "operator not created!") {
		return op, nil
	}
	return nil, fmt.Errorf("lackof memory building execution tree")
}

// Scan
func (this *builder) VisitPrimaryScan(plan *plan.PrimaryScan) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewPrimaryScan(plan, this.context), this.context)
}

func (this *builder) VisitPrimaryScan3(plan *plan.PrimaryScan3) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewPrimaryScan3(plan, this.context), this.context)
}

func (this *builder) VisitParentScan(plan *plan.ParentScan) (interface{}, error) {
	return checkOp(NewParentScan(plan, this.context), this.context)
}

func (this *builder) VisitIndexScan(plan *plan.IndexScan) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexScan(plan, this.context), this.context)
}

func (this *builder) VisitIndexScan2(plan *plan.IndexScan2) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexScan2(plan, this.context), this.context)
}

func (this *builder) VisitIndexScan3(plan *plan.IndexScan3) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexScan3(plan, this.context), this.context)
}

func (this *builder) VisitIndexCountScan(plan *plan.IndexCountScan) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexCountScan(plan, this.context), this.context)
}

func (this *builder) VisitIndexCountScan2(plan *plan.IndexCountScan2) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexCountScan2(plan, this.context), this.context)
}

func (this *builder) VisitIndexCountDistinctScan2(plan *plan.IndexCountDistinctScan2) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexCountDistinctScan2(plan, this.context), this.context)
}

func (this *builder) VisitKeyScan(plan *plan.KeyScan) (interface{}, error) {
	return checkOp(NewKeyScan(plan, this.context), this.context)
}

func (this *builder) VisitExpressionScan(plan *plan.ExpressionScan) (interface{}, error) {
	return checkOp(NewExpressionScan(plan, this.context), this.context)
}

func (this *builder) VisitValueScan(plan *plan.ValueScan) (interface{}, error) {
	return checkOp(NewValueScan(plan, this.context), this.context)
}

func (this *builder) VisitDummyScan(plan *plan.DummyScan) (interface{}, error) {
	return checkOp(NewDummyScan(plan, this.context), this.context)
}

func (this *builder) VisitCountScan(plan *plan.CountScan) (interface{}, error) {
	return checkOp(NewCountScan(plan, this.context), this.context)
}

func (this *builder) VisitDistinctScan(plan *plan.DistinctScan) (interface{}, error) {
	scan, err := plan.Scan().Accept(this)
	if err != nil {
		return nil, err
	}

	return checkOp(NewDistinctScan(plan, this.context, scan.(Operator)), this.context)
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

	return checkOp(NewUnionScan(plan, this.context, scans), this.context)
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

	return checkOp(NewIntersectScan(plan, this.context, scans), this.context)
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

	return checkOp(NewOrderedIntersectScan(plan, this.context, scans), this.context)
}

// Fetch
func (this *builder) VisitFetch(plan *plan.Fetch) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return checkOp(NewFetch(plan, this.context), this.context)
}

// DummyFetch
func (this *builder) VisitDummyFetch(plan *plan.DummyFetch) (interface{}, error) {
	return checkOp(NewDummyFetch(plan, this.context), this.context)
}

// Join
func (this *builder) VisitJoin(plan *plan.Join) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return checkOp(NewJoin(plan, this.context), this.context)
}

func (this *builder) VisitIndexJoin(plan *plan.IndexJoin) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexJoin(plan, this.context), this.context)
}

func (this *builder) VisitNLJoin(plan *plan.NLJoin) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return checkOp(NewNLJoin(plan, this.context, c.(Operator), this.aliasMap), this.context)
}

func (this *builder) VisitHashJoin(plan *plan.HashJoin) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return checkOp(NewHashJoin(plan, this.context, c.(Operator), this.aliasMap), this.context)
}

func (this *builder) VisitNest(plan *plan.Nest) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return checkOp(NewNest(plan, this.context), this.context)
}

func (this *builder) VisitIndexNest(plan *plan.IndexNest) (interface{}, error) {
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexNest(plan, this.context), this.context)
}

func (this *builder) VisitNLNest(plan *plan.NLNest) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return checkOp(NewNLNest(plan, this.context, c.(Operator), this.aliasMap), this.context)
}

func (this *builder) VisitHashNest(plan *plan.HashNest) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}

	return checkOp(NewHashNest(plan, this.context, c.(Operator), this.aliasMap), this.context)
}

func (this *builder) VisitUnnest(plan *plan.Unnest) (interface{}, error) {
	return checkOp(NewUnnest(plan, this.context), this.context)
}

// Let + Letting
func (this *builder) VisitLet(plan *plan.Let) (interface{}, error) {
	return checkOp(NewLet(plan, this.context), this.context)
}

// With
func (this *builder) VisitWith(plan *plan.With) (interface{}, error) {
	child := plan.Child()
	c, e := child.Accept(this)
	if e != nil {
		return nil, e
	}
	return checkOp(NewWith(plan, this.context, c.(Operator)), this.context)
}

// Filter
func (this *builder) VisitFilter(plan *plan.Filter) (interface{}, error) {
	return checkOp(NewFilter(plan, this.context, this.aliasMap), this.context)
}

// Group
func (this *builder) VisitInitialGroup(plan *plan.InitialGroup) (interface{}, error) {
	return checkOp(NewInitialGroup(plan, this.context), this.context)
}

func (this *builder) VisitIntermediateGroup(plan *plan.IntermediateGroup) (interface{}, error) {
	return checkOp(NewIntermediateGroup(plan, this.context), this.context)
}

func (this *builder) VisitFinalGroup(plan *plan.FinalGroup) (interface{}, error) {
	return checkOp(NewFinalGroup(plan, this.context), this.context)
}

// Window functions
func (this *builder) VisitWindowAggregate(plan *plan.WindowAggregate) (interface{}, error) {
	return checkOp(NewWindowAggregate(plan, this.context), this.context)
}

// Project
func (this *builder) VisitInitialProject(plan *plan.InitialProject) (interface{}, error) {
	return checkOp(NewInitialProject(plan, this.context), this.context)
}

// TODO retire
func (this *builder) VisitFinalProject(plan *plan.FinalProject) (interface{}, error) {
	// skip operator
	return NewNoop(), nil
}

func (this *builder) VisitIndexCountProject(plan *plan.IndexCountProject) (interface{}, error) {
	return checkOp(NewIndexCountProject(plan, this.context), this.context)
}

// Distinct
func (this *builder) VisitDistinct(plan *plan.Distinct) (interface{}, error) {
	return checkOp(NewDistinct(plan, this.context, false), this.context)
}

// All
func (this *builder) VisitAll(plan *plan.All) (interface{}, error) {
	return checkOp(NewAll(plan, this.context, false), this.context)
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

	return checkOp(NewUnionAll(plan, this.context, children...), this.context)
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

	if plan.Distinct() {
		return checkOp(NewIntersect(plan, this.context, first.(Operator), second.(Operator)), this.context)
	}
	return checkOp(NewIntersectAll(plan, this.context, first.(Operator), second.(Operator)), this.context)
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

	if plan.Distinct() {
		return checkOp(NewExcept(plan, this.context, first.(Operator), second.(Operator)), this.context)
	}
	return checkOp(NewExceptAll(plan, this.context, first.(Operator), second.(Operator)), this.context)
}

// Order
func (this *builder) VisitOrder(plan *plan.Order) (interface{}, error) {
	if plan.LimitPushed() {
		return checkOp(NewOrderLimit(plan, this.context), this.context)
	} else {
		return checkOp(NewOrder(plan, this.context), this.context)
	}
}

// Offset
func (this *builder) VisitOffset(plan *plan.Offset) (interface{}, error) {
	return checkOp(NewOffset(plan, this.context), this.context)
}

func (this *builder) VisitLimit(plan *plan.Limit) (interface{}, error) {
	return checkOp(NewLimit(plan, this.context), this.context)
}

// Insert
func (this *builder) VisitSendInsert(plan *plan.SendInsert) (interface{}, error) {
	return checkOp(NewSendInsert(plan, this.context), this.context)
}

// Upsert
func (this *builder) VisitSendUpsert(plan *plan.SendUpsert) (interface{}, error) {
	return checkOp(NewSendUpsert(plan, this.context), this.context)
}

// Delete
func (this *builder) VisitSendDelete(plan *plan.SendDelete) (interface{}, error) {
	return checkOp(NewSendDelete(plan, this.context), this.context)
}

// Update
func (this *builder) VisitClone(plan *plan.Clone) (interface{}, error) {
	return checkOp(NewClone(plan, this.context), this.context)
}

func (this *builder) VisitSet(plan *plan.Set) (interface{}, error) {
	return checkOp(NewSet(plan, this.context), this.context)
}

func (this *builder) VisitUnset(plan *plan.Unset) (interface{}, error) {
	return checkOp(NewUnset(plan, this.context), this.context)
}

func (this *builder) VisitSendUpdate(plan *plan.SendUpdate) (interface{}, error) {
	return checkOp(NewSendUpdate(plan, this.context), this.context)
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

	return checkOp(NewMerge(plan, this.context, update, delete, insert), this.context)
}

// Alias
func (this *builder) VisitAlias(plan *plan.Alias) (interface{}, error) {
	return checkOp(NewAlias(plan, this.context), this.context)
}

// Authorize
func (this *builder) VisitAuthorize(plan *plan.Authorize) (interface{}, error) {
	child, err := plan.Child().Accept(this)
	if err != nil {
		return nil, err
	}

	return checkOp(NewAuthorize(plan, this.context, child.(Operator), this.dynamicAuthorize), this.context)
}

// Parallel
func (this *builder) VisitParallel(plan *plan.Parallel) (interface{}, error) {
	child, err := plan.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	if !this.context.assert(child != nil, "child operator not created") {
		return nil, fmt.Errorf("parallel operator has no child")
	}

	maxParallelism := util.MinInt(plan.MaxParallelism(), this.context.MaxParallelism())

	if maxParallelism == 1 {
		return child, nil
	} else {
		return checkOp(NewParallel(plan, this.context, child.(Operator)), this.context)
	}
}

// Sequence
func (this *builder) VisitSequence(plan *plan.Sequence) (interface{}, error) {
	children := plan.Children()

	// if there is a single child, no need for a Sequence operator
	if len(children) == 1 {
		child, err := children[0].Accept(this)
		if err != nil {
			return nil, err
		}
		if !this.context.assert(child != nil, "child operator not created") {
			return nil, fmt.Errorf("sequence operator has no child")
		}
		return child.(Operator), nil
	}

	execChildren := _SEQUENCE_POOL.Get()

	for _, pchild := range children {
		child, err := pchild.Accept(this)
		if err != nil {
			_SEQUENCE_POOL.Put(execChildren)
			return nil, err
		}
		if !this.context.assert(child != nil, "child operator not created") {
			_SEQUENCE_POOL.Put(execChildren)
			return nil, fmt.Errorf("sequence operator has missing child")
		}

		// skip noops
		switch child.(type) {
		case *Noop:
			continue
		}

		execChildren = append(execChildren, child.(Operator))
	}

	if len(execChildren) == 1 {
		child := execChildren[0]
		_SEQUENCE_POOL.Put(execChildren)
		return child.(Operator), nil
	}

	// if the first child is also a Sequence operator, then just tag the
	// new children onto the children array of the existing Sequence operator
	// this way we generate one less Sequence operator.
	if seq, ok := execChildren[0].(*Sequence); ok {
		seq.children = append(seq.children, execChildren[1:]...)
		child := execChildren[0]
		_SEQUENCE_POOL.Put(execChildren)
		return child.(Operator), nil
	}
	return checkOp(NewSequence(plan, this.context, execChildren...), this.context)
}

// Discard
func (this *builder) VisitDiscard(plan *plan.Discard) (interface{}, error) {
	return checkOp(NewDiscard(plan, this.context), this.context)
}

// Stream
func (this *builder) VisitStream(plan *plan.Stream) (interface{}, error) {
	return checkOp(NewStream(plan, this.context), this.context)
}

// Collect
func (this *builder) VisitCollect(plan *plan.Collect) (interface{}, error) {
	return checkOp(NewCollect(plan, this.context), this.context)
}

// CreateIndex
func (this *builder) VisitCreatePrimaryIndex(plan *plan.CreatePrimaryIndex) (interface{}, error) {
	return checkOp(NewCreatePrimaryIndex(plan, this.context), this.context)
}

// GrantRole
func (this *builder) VisitGrantRole(plan *plan.GrantRole) (interface{}, error) {
	return checkOp(NewGrantRole(plan, this.context), this.context)
}

// RevokeRole
func (this *builder) VisitRevokeRole(plan *plan.RevokeRole) (interface{}, error) {
	return checkOp(NewRevokeRole(plan, this.context), this.context)
}

// CreateIndex
func (this *builder) VisitCreateIndex(plan *plan.CreateIndex) (interface{}, error) {
	return checkOp(NewCreateIndex(plan, this.context), this.context)
}

// DropIndex
func (this *builder) VisitDropIndex(plan *plan.DropIndex) (interface{}, error) {
	return checkOp(NewDropIndex(plan, this.context), this.context)
}

// AlterIndex
func (this *builder) VisitAlterIndex(plan *plan.AlterIndex) (interface{}, error) {
	return checkOp(NewAlterIndex(plan, this.context), this.context)
}

// BuildIndexes
func (this *builder) VisitBuildIndexes(plan *plan.BuildIndexes) (interface{}, error) {
	return checkOp(NewBuildIndexes(plan, this.context), this.context)
}

// CreateScope
func (this *builder) VisitCreateScope(plan *plan.CreateScope) (interface{}, error) {
	return checkOp(NewCreateScope(plan, this.context), this.context)
}

// DropScope
func (this *builder) VisitDropScope(plan *plan.DropScope) (interface{}, error) {
	return checkOp(NewDropScope(plan, this.context), this.context)
}

// CreateCollection
func (this *builder) VisitCreateCollection(plan *plan.CreateCollection) (interface{}, error) {
	return checkOp(NewCreateCollection(plan, this.context), this.context)
}

// DropCollection
func (this *builder) VisitDropCollection(plan *plan.DropCollection) (interface{}, error) {
	return checkOp(NewDropCollection(plan, this.context), this.context)
}

// FlushCollection
func (this *builder) VisitFlushCollection(plan *plan.FlushCollection) (interface{}, error) {
	return checkOp(NewFlushCollection(plan, this.context), this.context)
}

// Prepare
func (this *builder) VisitPrepare(plan *plan.Prepare) (interface{}, error) {
	this.dynamicAuthorize = false
	return checkOp(NewPrepare(plan, this.context, plan.Prepared()), this.context)
}

// Explain
func (this *builder) VisitExplain(plan *plan.Explain) (interface{}, error) {
	this.dynamicAuthorize = false
	return checkOp(NewExplain(plan, this.context), this.context)
}

// Infer
func (this *builder) VisitInferKeyspace(plan *plan.InferKeyspace) (interface{}, error) {
	return checkOp(NewInferKeyspace(plan, this.context), this.context)
}

// CreateFunction
func (this *builder) VisitCreateFunction(plan *plan.CreateFunction) (interface{}, error) {
	return checkOp(NewCreateFunction(plan, this.context), this.context)
}

// DropFunction
func (this *builder) VisitDropFunction(plan *plan.DropFunction) (interface{}, error) {
	return checkOp(NewDropFunction(plan, this.context), this.context)
}

// ExecuteFunction
func (this *builder) VisitExecuteFunction(plan *plan.ExecuteFunction) (interface{}, error) {
	return checkOp(NewExecuteFunction(plan, this.context), this.context)
}

// IndexFtsSearch
func (this *builder) VisitIndexFtsSearch(plan *plan.IndexFtsSearch) (interface{}, error) {
	this.setScannedIndexes(plan.Term())
	this.setAliasMap(plan.Term())
	return checkOp(NewIndexFtsSearch(plan, this.context), this.context)
}

// Index Advisor
func (this *builder) VisitIndexAdvice(plan *plan.IndexAdvice) (interface{}, error) {
	return checkOp(NewIndexAdvisor(plan, this.context), this.context)
}

func (this *builder) VisitAdvise(plan *plan.Advise) (interface{}, error) {
	this.dynamicAuthorize = false
	return checkOp(NewAdviseIndex(plan, this.context), this.context)
}

// Update Statistics
func (this *builder) VisitUpdateStatistics(plan *plan.UpdateStatistics) (interface{}, error) {
	return checkOp(NewUpdateStatistics(plan, this.context), this.context)
}

// Start Transaction
func (this *builder) VisitStartTransaction(plan *plan.StartTransaction) (interface{}, error) {
	return checkOp(NewStartTransaction(plan, this.context), this.context)
}

// Commit Transaction
func (this *builder) VisitCommitTransaction(plan *plan.CommitTransaction) (interface{}, error) {
	return checkOp(NewCommitTransaction(plan, this.context), this.context)
}

// Rollback Transaction
func (this *builder) VisitRollbackTransaction(plan *plan.RollbackTransaction) (interface{}, error) {
	return checkOp(NewRollbackTransaction(plan, this.context), this.context)
}

// Transaction Isolation
func (this *builder) VisitTransactionIsolation(plan *plan.TransactionIsolation) (interface{}, error) {
	return checkOp(NewTransactionIsolation(plan, this.context), this.context)
}

// Savepoint
func (this *builder) VisitSavepoint(plan *plan.Savepoint) (interface{}, error) {
	return checkOp(NewSavepoint(plan, this.context), this.context)
}
