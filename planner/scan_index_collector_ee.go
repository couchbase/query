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

func (this *scanIdxCol) VisitPrimaryScan(op *plan.PrimaryScan) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitPrimaryScan3(op *plan.PrimaryScan3) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexScan(op *plan.IndexScan) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexScan2(op *plan.IndexScan2) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexScan3(op *plan.IndexScan3) (any, error) {
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

func (this *scanIdxCol) VisitKeyScan(op *plan.KeyScan) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitValueScan(op *plan.ValueScan) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDummyScan(op *plan.DummyScan) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitExternalScan(op *plan.ExternalScan) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCountScan(op *plan.CountScan) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountScan(op *plan.IndexCountScan) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountScan2(op *plan.IndexCountScan2) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountDistinctScan2(op *plan.IndexCountDistinctScan2) (any, error) {
	info := extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase)
	if info != nil {
		info.SetCostBased(op.Cost() > 0 && op.Cardinality() > 0)
		this.addIndexInfo(info)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitDistinctScan(op *plan.DistinctScan) (any, error) {
	op.Scan().Accept(this)
	return nil, nil
}

func (this *scanIdxCol) VisitUnionScan(op *plan.UnionScan) (any, error) {
	for _, scan := range op.Scans() {
		scan.Accept(this)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitIntersectScan(op *plan.IntersectScan) (any, error) {
	for _, scan := range op.Scans() {
		scan.Accept(this)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitOrderedIntersectScan(op *plan.OrderedIntersectScan) (any, error) {
	for _, scan := range op.Scans() {
		scan.Accept(this)
	}
	return nil, nil
}

func (this *scanIdxCol) VisitExpressionScan(op *plan.ExpressionScan) (any, error) {
	return nil, nil
}

// Fetch
func (this *scanIdxCol) VisitFetch(op *plan.Fetch) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDummyFetch(op *plan.DummyFetch) (any, error) {
	return nil, nil
}

// Join
func (this *scanIdxCol) VisitJoin(op *plan.Join) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexJoin(op *plan.IndexJoin) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitNest(op *plan.Nest) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexNest(op *plan.IndexNest) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitUnnest(op *plan.Unnest) (any, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitNLJoin(op *plan.NLJoin) (any, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitNLNest(op *plan.NLNest) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitHashJoin(op *plan.HashJoin) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitHashNest(op *plan.HashNest) (any, error) {
	return nil, nil
}

// Let + Letting, With
func (this *scanIdxCol) VisitLet(op *plan.Let) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitWith(op *plan.With) (any, error) {
	return nil, nil
}

// Filter
func (this *scanIdxCol) VisitFilter(op *plan.Filter) (any, error) {
	return nil, nil
}

// Group
func (this *scanIdxCol) VisitInitialGroup(op *plan.InitialGroup) (any, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitIntermediateGroup(op *plan.IntermediateGroup) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitFinalGroup(op *plan.FinalGroup) (any, error) {
	return nil, nil
}

// Window functions
func (this *scanIdxCol) VisitWindowAggregate(op *plan.WindowAggregate) (any, error) {
	return nil, nil
}

// Project
func (this *scanIdxCol) VisitInitialProject(op *plan.InitialProject) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexCountProject(op *plan.IndexCountProject) (any, error) {
	return nil, nil
}

// Distinct
func (this *scanIdxCol) VisitDistinct(op *plan.Distinct) (any, error) {
	return nil, nil
}

// All
func (this *scanIdxCol) VisitAll(op *plan.All) (any, error) {
	return nil, nil
}

// Set operators
func (this *scanIdxCol) VisitUnionAll(op *plan.UnionAll) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIntersectAll(op *plan.IntersectAll) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitExceptAll(op *plan.ExceptAll) (any, error) {
	return nil, nil
}

// Order
func (this *scanIdxCol) VisitOrder(op *plan.Order) (any, error) {
	return nil, nil
}

// Paging
func (this *scanIdxCol) VisitOffset(op *plan.Offset) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitLimit(op *plan.Limit) (any, error) {
	return nil, nil
}

// Insert
func (this *scanIdxCol) VisitSendInsert(op *plan.SendInsert) (any, error) {
	return nil, nil
}

// Upsert
func (this *scanIdxCol) VisitSendUpsert(op *plan.SendUpsert) (any, error) {
	return nil, nil
}

// Delete
func (this *scanIdxCol) VisitSendDelete(op *plan.SendDelete) (any, error) {
	return nil, nil
}

// Update
func (this *scanIdxCol) VisitClone(op *plan.Clone) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSet(op *plan.Set) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitUnset(op *plan.Unset) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSendUpdate(op *plan.SendUpdate) (any, error) {
	return nil, nil
}

// Merge
func (this *scanIdxCol) VisitMerge(op *plan.Merge) (any, error) {
	return nil, nil
}

// Framework
func (this *scanIdxCol) VisitAlias(op *plan.Alias) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAuthorize(op *plan.Authorize) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitParallel(op *plan.Parallel) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSequence(op *plan.Sequence) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDiscard(op *plan.Discard) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitStream(op *plan.Stream) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCollect(op *plan.Collect) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitReceive(op *plan.Receive) (any, error) {
	return nil, nil
}

// Index DDL
func (this *scanIdxCol) VisitCreatePrimaryIndex(op *plan.CreatePrimaryIndex) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCreateIndex(op *plan.CreateIndex) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropIndex(op *plan.DropIndex) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterIndex(op *plan.AlterIndex) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitBuildIndexes(op *plan.BuildIndexes) (any, error) {
	return nil, nil
}

// Bucket DDL
func (this *scanIdxCol) VisitCreateBucket(op *plan.CreateBucket) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterBucket(op *plan.AlterBucket) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropBucket(op *plan.DropBucket) (any, error) {
	return nil, nil
}

// Catalog DDL
func (this *scanIdxCol) VisitCreateCatalog(op *plan.CreateCatalog) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterCatalog(op *plan.AlterCatalog) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropCatalog(op *plan.DropCatalog) (any, error) {
	return nil, nil
}

// Collection DDL
func (this *scanIdxCol) VisitAlterCollection(op *plan.AlterCollection) (any, error) {
	return nil, nil
}

// Scope and Collection DDL
func (this *scanIdxCol) VisitCreateScope(op *plan.CreateScope) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropScope(op *plan.DropScope) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCreateCollection(op *plan.CreateCollection) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropCollection(op *plan.DropCollection) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitFlushCollection(op *plan.FlushCollection) (any, error) {
	return nil, nil
}

// Users
func (this *scanIdxCol) VisitCreateUser(op *plan.CreateUser) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterUser(op *plan.AlterUser) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropUser(op *plan.DropUser) (any, error) {
	return nil, nil
}

// Groups
func (this *scanIdxCol) VisitCreateGroup(op *plan.CreateGroup) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterGroup(op *plan.AlterGroup) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropGroup(op *plan.DropGroup) (any, error) {
	return nil, nil
}

// Roles
func (this *scanIdxCol) VisitGrantRole(op *plan.GrantRole) (any, error) {
	return nil, nil
}
func (this *scanIdxCol) VisitRevokeRole(op *plan.RevokeRole) (any, error) {
	return nil, nil
}

// Explain
func (this *scanIdxCol) VisitExplain(op *plan.Explain) (any, error) {
	return nil, nil
}

// Explain Function
func (this *scanIdxCol) VisitExplainFunction(op *plan.ExplainFunction) (any, error) {
	return nil, nil
}

// Prepare
func (this *scanIdxCol) VisitPrepare(op *plan.Prepare) (any, error) {
	return nil, nil
}

// Infer
func (this *scanIdxCol) VisitInferKeyspace(op *plan.InferKeyspace) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitInferExpression(op *plan.InferExpression) (any, error) {
	return nil, nil
}

// Function statements
func (this *scanIdxCol) VisitCreateFunction(op *plan.CreateFunction) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropFunction(op *plan.DropFunction) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitExecuteFunction(op *plan.ExecuteFunction) (any, error) {
	return nil, nil
}

// IndexFtsSearch
func (this *scanIdxCol) VisitIndexFtsSearch(op *plan.IndexFtsSearch) (any, error) {
	this.addIndexInfo(extractInfo(op.Index(), this.alias, this.keyspace, false, this.validatePhase))
	return nil, nil
}

// Index Advisor
func (this *scanIdxCol) VisitAdvise(op *plan.Advise) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitIndexAdvice(op *plan.IndexAdvice) (any, error) {
	return nil, nil
}

// Update Statistics
func (this *scanIdxCol) VisitUpdateStatistics(op *plan.UpdateStatistics) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitStartTransaction(op *plan.StartTransaction) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCommitTransaction(op *plan.CommitTransaction) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitRollbackTransaction(op *plan.RollbackTransaction) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitTransactionIsolation(op *plan.TransactionIsolation) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitSavepoint(op *plan.Savepoint) (any, error) {
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
			info.SetVectorInfo(vectorInfo, vectorPos, virtualIdx.VectorType())
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
			if vectorPos < 0 && kp.HasAttribute(datastore.IK_VECTORS) {
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

func (this *scanIdxCol) VisitCreateSequence(op *plan.CreateSequence) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCreateKnowledge(op *plan.CreateKnowledge) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropKnowledge(op *plan.DropKnowledge) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropSequence(op *plan.DropSequence) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterSequence(op *plan.AlterSequence) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitCreateCredentialStore(op *plan.CreateCredentialStore) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitAlterCredentialStore(op *plan.AlterCredentialStore) (any, error) {
	return nil, nil
}

func (this *scanIdxCol) VisitDropCredentialStore(op *plan.DropCredentialStore) (any, error) {
	return nil, nil
}
