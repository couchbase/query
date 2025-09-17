//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package planner

import (
	"fmt"

	"github.com/couchbase/query-ee/indexadvisor/iaplan"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/virtual"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

type scanIdxCol struct {
	keyspace      datastore.Keyspace
	alias         string
	indexInfos    iaplan.IndexInfos
	covering      bool
	vector        bool
	validatePhase bool
	property      uint32
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
		info.SetCovering(len(op.Covers()) > 0)
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		info.SetProperty(len(op.Covers()) > 0, op.Limit() != nil, op.Offset() != nil,
			len(op.OrderTerms()) > 0, op.GroupAggs() != nil, op.HasEarlyOrder())
		this.property = info.Property()
		this.vector = info.HasVectorInfo()
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

func (this *scanIdxCol) VisitReceive(op *plan.Receive) (interface{}, error) {
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

// Bucket DDL
func (this *scanIdxCol) VisitCreateBucket(op *plan.CreateBucket) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterBucket(op *plan.AlterBucket) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropBucket(op *plan.DropBucket) (interface{}, error) {
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

// Users
func (this *scanIdxCol) VisitCreateUser(op *plan.CreateUser) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterUser(op *plan.AlterUser) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropUser(op *plan.DropUser) (interface{}, error) {
	return nil, nil
}

// Groups
func (this *scanIdxCol) VisitCreateGroup(op *plan.CreateGroup) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterGroup(op *plan.AlterGroup) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropGroup(op *plan.DropGroup) (interface{}, error) {
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

// Explain Function
func (this *scanIdxCol) VisitExplainFunction(op *plan.ExplainFunction) (interface{}, error) {
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

func (this *scanIdxCol) VisitInferExpression(op *plan.InferExpression) (interface{}, error) {
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
		expr, _, err := formalizeExpr(formalizer, key, false)
		if err != nil {
			return nil
		}
		keys[i] = expr
	}
	return keys
}

func extractInfo(index datastore.Index, keyspaceAlias string, keyspace datastore.Keyspace, deferred,
	validatePhase bool) *iaplan.IndexInfo {

	if index == nil || (validatePhase && index.Type() != datastore.VIRTUAL) || index.Type() == datastore.SEQ_SCAN {
		return nil
	}

	lkmissing := indexHasLeadingKeyMissingValues(index, uint64(0))
	info := iaplan.NewIndexInfo(index.Name(), keyspaceAlias, keyspace, index.IsPrimary(), "", nil, nil,
		"", deferred, lkmissing, index.Type())
	if validatePhase {
		info.SetCovering(true)
		if virtualIdx, ok := index.(*virtual.VirtualIndex); ok {
			vectorPos := virtualIdx.VectorPos()
			dimension := fmt.Sprintf("%d", virtualIdx.VectorDimension())
			similarity := string(virtualIdx.VectorDistanceType())
			description := virtualIdx.VectorDescription()
			vectorInfo := iaplan.NewVectorInfo(dimension, similarity, description)
			info.SetVectorInfo(vectorInfo, vectorPos)
			if virtualIdx.IsBhive() {
				info.SetBhive()
			}

		}
	} else if index.Type() == datastore.GSI {
		info.SetFormalizedKeyExprs(formalizeIndexKeys(keyspaceAlias, index.RangeKey()))
		info.SetKeyStrings(getIndexKeyStringArray(index))
		info.SetCondition(index.Condition())
		info.SetPartition(getIndexPartitionToString(index))
	}
	return info
}

func getIndexKeyStringArray(index datastore.Index) (keys, includes []string, desc []bool,
	lkmissing, isBhive bool, vectorPos int, vectorInfo *iaplan.VectorInfo) {
	vectorPos = -1
	stringer := expression.NewStringer()
	if index2, ok2 := index.(datastore.Index2); ok2 {
		idxkeys := index2.RangeKey2()
		keys = make([]string, len(idxkeys))
		desc = make([]bool, len(idxkeys))
		index6, ok6 := index.(datastore.Index6)
		for i, kp := range idxkeys {
			keys[i] = stringer.Visit(kp.Expr)
			desc[i] = kp.HasAttribute(datastore.IK_DESC)
			if i == 0 {
				lkmissing = kp.HasAttribute(datastore.IK_MISSING)
			}
			if vectorPos < 0 && kp.HasAttribute(datastore.IK_VECTOR) {
				vectorPos = i
				if ok6 {
					isBhive = index6.IsBhive()
					dimension := fmt.Sprintf("%d", index6.VectorDimension())
					similarity := string(index6.VectorDistanceType())
					description := index6.VectorDescription()
					vectorInfo = iaplan.NewVectorInfo(dimension, similarity, description)
				}
			}
		}
		if ok6 {
			idxIncludes := index6.Include()
			includes = make([]string, len(idxIncludes))
			for i, incl := range idxIncludes {
				includes[i] = stringer.Visit(incl)
			}
		}
	} else {
		idxkeys := index.RangeKey()
		keys = make([]string, len(idxkeys))
		desc = make([]bool, len(idxkeys))
		for i, kp := range idxkeys {
			keys[i] = stringer.Visit(kp)
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

func (this *scanIdxCol) VisitCreateSequence(op *plan.CreateSequence) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropSequence(op *plan.DropSequence) (interface{}, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterSequence(op *plan.AlterSequence) (interface{}, error) {
	return nil, nil
}
