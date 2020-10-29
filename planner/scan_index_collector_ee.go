//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package planner

import (
	"github.com/couchbase/query-ee/indexadvisor/iaplan"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

type scanIdxCol struct {
	keyspace      datastore.Keyspace
	alias         string
	indexInfos    iaplan.IndexInfos
	covering      bool
	validatePhase bool
}

func NewScanIdxCol() *scanIdxCol {
	return &scanIdxCol{
		indexInfos: make(iaplan.IndexInfos, 0, 1),
		covering:   true,
	}
}

func (this *scanIdxCol) setKeyspace(keyspace datastore.Keyspace) {
	this.keyspace = keyspace
}

func (this *scanIdxCol) setAlias(alias string) {
	this.alias = alias
}

func (this *scanIdxCol) setUnCovering() {
	this.covering = false
}

func (this *scanIdxCol) setValidatePhase(b bool) {
	this.validatePhase = b
}

func (this *scanIdxCol) isCovering() bool {
	return this.covering
}

func (this *scanIdxCol) addIndexInfo(indexInfo *iaplan.IndexInfo) {
	if indexInfo == nil {
		return
	}

	for _, info := range this.indexInfos {
		if info.EquivalentTo(indexInfo, true) {
			return
		}
	}
	this.indexInfos = append(this.indexInfos, indexInfo)
}

func (this *scanIdxCol) VisitPrimaryScan(op *plan.PrimaryScan) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitPrimaryScan3(op *plan.PrimaryScan3) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitParentScan(op *plan.ParentScan) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexScan(op *plan.IndexScan) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexScan2(op *plan.IndexScan2) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexScan3(op *plan.IndexScan3) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitKeyScan(op *plan.KeyScan) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitValueScan(op *plan.ValueScan) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDummyScan(op *plan.DummyScan) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCountScan(op *plan.CountScan) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountScan(op *plan.IndexCountScan) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountScan2(op *plan.IndexCountScan2) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountDistinctScan2(op *plan.IndexCountDistinctScan2) (interface{}, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitDistinctScan(op *plan.DistinctScan) (interface{}, error) {
	op.Scan().Accept(this)
	return nil, nil
}

func (this *scanIdxCol) VisitUnionScan(op *plan.UnionScan) (interface{}, error) {
	for _, scan := range op.Scans() {
		scan.Accept(this)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIntersectScan(op *plan.IntersectScan) (interface{}, error) {
	for _, scan := range op.Scans() {
		scan.Accept(this)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitOrderedIntersectScan(op *plan.OrderedIntersectScan) (interface{}, error) {
	for _, scan := range op.Scans() {
		scan.Accept(this)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitExpressionScan(op *plan.ExpressionScan) (interface{}, error) {
	return nil, nil
}

// Fetch
func (this *scanIdxCol) VisitFetch(op *plan.Fetch) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDummyFetch(op *plan.DummyFetch) (interface{}, error) {
	return nil, nil
}

// Join
func (this *scanIdxCol) VisitJoin(op *plan.Join) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexJoin(op *plan.IndexJoin) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitNest(op *plan.Nest) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexNest(op *plan.IndexNest) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitUnnest(op *plan.Unnest) (interface{}, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitNLJoin(op *plan.NLJoin) (interface{}, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitNLNest(op *plan.NLNest) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitHashJoin(op *plan.HashJoin) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitHashNest(op *plan.HashNest) (interface{}, error) {
	return nil, nil
}

// Let + Letting, With
func (this *scanIdxCol) VisitLet(op *plan.Let) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitWith(op *plan.With) (interface{}, error) {
	return nil, nil
}

// Filter
func (this *scanIdxCol) VisitFilter(op *plan.Filter) (interface{}, error) {
	return nil, nil
}

// Group
func (this *scanIdxCol) VisitInitialGroup(op *plan.InitialGroup) (interface{}, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitIntermediateGroup(op *plan.IntermediateGroup) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitFinalGroup(op *plan.FinalGroup) (interface{}, error) {
	return nil, nil
}

// Window functions
func (this *scanIdxCol) VisitWindowAggregate(op *plan.WindowAggregate) (interface{}, error) {
	return nil, nil
}

// Project
func (this *scanIdxCol) VisitInitialProject(op *plan.InitialProject) (interface{}, error) {
	return nil, nil
}

// TODO retire
func (this *scanIdxCol) VisitFinalProject(op *plan.FinalProject) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountProject(op *plan.IndexCountProject) (interface{}, error) {
	return nil, nil
}

// Distinct
func (this *scanIdxCol) VisitDistinct(op *plan.Distinct) (interface{}, error) {
	return nil, nil
}

// All
func (this *scanIdxCol) VisitAll(op *plan.All) (interface{}, error) {
	return nil, nil
}

// Set operators
func (this *scanIdxCol) VisitUnionAll(op *plan.UnionAll) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIntersectAll(op *plan.IntersectAll) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitExceptAll(op *plan.ExceptAll) (interface{}, error) {
	return nil, nil
}

// Order
func (this *scanIdxCol) VisitOrder(op *plan.Order) (interface{}, error) {
	return nil, nil
}

// Paging
func (this *scanIdxCol) VisitOffset(op *plan.Offset) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitLimit(op *plan.Limit) (interface{}, error) {
	return nil, nil
}

// Insert
func (this *scanIdxCol) VisitSendInsert(op *plan.SendInsert) (interface{}, error) {
	return nil, nil
}

// Upsert
func (this *scanIdxCol) VisitSendUpsert(op *plan.SendUpsert) (interface{}, error) {
	return nil, nil
}

// Delete
func (this *scanIdxCol) VisitSendDelete(op *plan.SendDelete) (interface{}, error) {
	return nil, nil
}

// Update
func (this *scanIdxCol) VisitClone(op *plan.Clone) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSet(op *plan.Set) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitUnset(op *plan.Unset) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSendUpdate(op *plan.SendUpdate) (interface{}, error) {
	return nil, nil
}

// Merge
func (this *scanIdxCol) VisitMerge(op *plan.Merge) (interface{}, error) {
	return nil, nil
}

// Framework
func (this *scanIdxCol) VisitAlias(op *plan.Alias) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAuthorize(op *plan.Authorize) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitParallel(op *plan.Parallel) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSequence(op *plan.Sequence) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDiscard(op *plan.Discard) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitStream(op *plan.Stream) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCollect(op *plan.Collect) (interface{}, error) {
	return nil, nil
}

// Index DDL
func (this *scanIdxCol) VisitCreatePrimaryIndex(op *plan.CreatePrimaryIndex) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCreateIndex(op *plan.CreateIndex) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropIndex(op *plan.DropIndex) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterIndex(op *plan.AlterIndex) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitBuildIndexes(op *plan.BuildIndexes) (interface{}, error) {
	return nil, nil
}

// Scope and Collection DDL
func (this *scanIdxCol) VisitCreateScope(op *plan.CreateScope) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropScope(op *plan.DropScope) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCreateCollection(op *plan.CreateCollection) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropCollection(op *plan.DropCollection) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitFlushCollection(op *plan.FlushCollection) (interface{}, error) {
	return nil, nil
}

// Roles
func (this *scanIdxCol) VisitGrantRole(op *plan.GrantRole) (interface{}, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitRevokeRole(op *plan.RevokeRole) (interface{}, error) {
	return nil, nil
}

// Explain
func (this *scanIdxCol) VisitExplain(op *plan.Explain) (interface{}, error) {
	return nil, nil
}

// Prepare
func (this *scanIdxCol) VisitPrepare(op *plan.Prepare) (interface{}, error) {
	return nil, nil
}

// Infer
func (this *scanIdxCol) VisitInferKeyspace(op *plan.InferKeyspace) (interface{}, error) {
	return nil, nil
}

// Function statements
func (this *scanIdxCol) VisitCreateFunction(op *plan.CreateFunction) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropFunction(op *plan.DropFunction) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitExecuteFunction(op *plan.ExecuteFunction) (interface{}, error) {
	return nil, nil
}

// IndexFtsSearch
func (this *scanIdxCol) VisitIndexFtsSearch(op *plan.IndexFtsSearch) (interface{}, error) {
	this.addIndexInfo(extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase))
	return nil, nil
}

// Index Advisor
func (this *scanIdxCol) VisitAdvise(op *plan.Advise) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexAdvice(op *plan.IndexAdvice) (interface{}, error) {
	return nil, nil
}

// Update Statistics
func (this *scanIdxCol) VisitUpdateStatistics(op *plan.UpdateStatistics) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitStartTransaction(op *plan.StartTransaction) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCommitTransaction(op *plan.CommitTransaction) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitRollbackTransaction(op *plan.RollbackTransaction) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitTransactionIsolation(op *plan.TransactionIsolation) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSavepoint(op *plan.Savepoint) (interface{}, error) {
	return nil, nil
}

func formalizeIndexKeys(alias string, keys expression.Expressions) expression.Expressions {
	formalizer := expression.NewSelfFormalizer(alias, nil)
	keys = keys.Copy()

	for i, key := range keys {
		expr, err := formalizeExpr(formalizer, key)
		if err != nil {
			return nil
		}
		keys[i] = expr
	}
	return keys
}

func formalizeExpr(formalizer *expression.Formalizer, key expression.Expression) (expression.Expression, error) {
	key = key.Copy()

	formalizer.SetIndexScope()
	key, err := formalizer.Map(key)
	formalizer.ClearIndexScope()
	if err != nil {
		return nil, err
	}

	dnf := base.NewDNF(key, true, true)
	key, err = dnf.Map(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func extractInfo(index datastore.Index, keyspaceAlias string, keyspace datastore.Keyspace, deferred, validatePhase bool) *iaplan.IndexInfo {
	if index == nil || (validatePhase && index.Type() != datastore.VIRTUAL) {
		return nil
	}

	info := iaplan.NewIndexInfo(index.Name(), keyspaceAlias, keyspace, index.IsPrimary(), "", nil, "", deferred, index.Type())
	if validatePhase {
		info.SetCovering()
	} else if index.Type() == datastore.GSI {
		if index2, ok := index.(datastore.Index2); ok {
			info.SetFormalizedKeyExprs(formalizeIndexKeys(keyspaceAlias, getIndexKeyExpressions(index2.RangeKey2())))
		} else {
			info.SetFormalizedKeyExprs(formalizeIndexKeys(keyspaceAlias, index.RangeKey()))
		}
		info.SetKeyStrings(getIndexKeyStringArray(index))
		info.SetCondition(index.Condition())
		info.SetPartition(getIndexPartitionToString(index))
	}
	return info
}

func getIndexKeyExpressions(keys datastore.IndexKeys) expression.Expressions {
	indexKeyExprs := make(expression.Expressions, 0, len(keys))
	for _, k := range keys {
		indexKeyExprs = append(indexKeyExprs, k.Expr)

	}
	return indexKeyExprs
}

func getIndexKeyStringArray(index datastore.Index) (rv []string, desc []bool) {
	stringer := expression.NewStringer()
	if index2, ok2 := index.(datastore.Index2); ok2 {
		keys := index2.RangeKey2()
		rv = make([]string, len(keys))
		desc = make([]bool, len(keys))
		for i, kp := range keys {
			rv[i] = stringer.Visit(kp.Expr)
			desc[i] = kp.HasAttribute(datastore.IK_DESC)
		}
	} else {
		rv = make([]string, len(index.RangeKey()))
		desc = make([]bool, len(index.RangeKey()))
		for i, kp := range index.RangeKey() {
			rv[i] = stringer.Visit(kp)
		}
	}
	return
}

func getIndexPartitionToString(index datastore.Index) (rv string) {
	index3, ok3 := index.(datastore.Index3)
	if !ok3 {
		return
	}
	partition, _ := index3.PartitionKeys()
	if partition == nil || partition.Strategy == datastore.NO_PARTITION {
		return
	}

	stringer := expression.NewStringer()
	rv = string(partition.Strategy) + "("
	for i, expr := range partition.Exprs {
		if i > 0 {
			rv += ","
		}
		rv += stringer.Visit(expr)
	}
	rv += ")"
	return
}
