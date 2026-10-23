//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"sort"
	"time"
)

const (
	_LARGE_COUNT            = 10000
	_WT_THRESHOLD           = 10
	_IO_THRESHOLD           = 2
	_AUTH_THRESHOLD         = time.Second
	_IS_THRESHOLD           = 0.9
	_MIN_TIME_FOR_REPORTING = time.Second
)

func AnalyseExecution(start Operator) ([]any, error) {
	a := &execAnalyser{}

	_, err := start.Accept(a)
	if err != nil {
		return nil, err
	}

	if a.ioTime > a.cpuTime*_IO_THRESHOLD && a.ioTime > _MIN_TIME_FOR_REPORTING {
		a.add("High IO time")
	}
	if a.waitTime > (a.cpuTime+a.ioTime)*_WT_THRESHOLD && a.waitTime > _MIN_TIME_FOR_REPORTING {
		a.add("High wait time")
	}
	if a.indexScanTime > time.Duration(float64(a.ioTime)*_IS_THRESHOLD) && a.indexScanTime > _MIN_TIME_FOR_REPORTING {
		a.add("High index scan time")
	}

	if len(a.results) > 1 {
		sort.Slice(a.results, func(i int, j int) bool {
			return a.results[i].(string) < a.results[j].(string)
		})
	}

	return a.results, nil
}

type execAnalyser struct {
	results []any

	cpuTime       time.Duration
	ioTime        time.Duration
	waitTime      time.Duration
	indexScanTime time.Duration
}

func (this *execAnalyser) add(s string) {
	if this.results == nil {
		this.results = make([]any, 0, 10)
	}
	// de-duplicate
	for i := range this.results {
		if this.results[i].(string) == s {
			return
		}
	}
	this.results = append(this.results, s)
}

func (this *execAnalyser) record(op *base) {
	// force accumulation of time
	op.switchPhase(_NOTIME)
	this.cpuTime += op.execTime
	this.ioTime += op.servTime
	this.waitTime += op.kernTime
}

func (this *execAnalyser) VisitPrimaryScan(op *PrimaryScan) (any, error) {
	this.record(op.getBase())
	if op.outDocs > _LARGE_COUNT {
		this.add("High primary scan count")
	}
	return nil, nil
}

func (this *execAnalyser) VisitPrimaryScan3(op *PrimaryScan3) (any, error) {
	this.record(op.getBase())
	if op.outDocs > _LARGE_COUNT {
		this.add("High primary scan count")
	}
	return nil, nil
}

func (this *execAnalyser) VisitIndexScan(op *IndexScan) (any, error) {
	this.record(op.getBase())
	this.indexScanTime += op.servTime
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIndexScan2(op *IndexScan2) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexScan3(op *IndexScan3) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitKeyScan(op *KeyScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitValueScan(op *ValueScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDummyScan(op *DummyScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExternalScan(op *ExternalScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCountScan(op *CountScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountScan(op *IndexCountScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountScan2(op *IndexCountScan2) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountDistinctScan2(op *IndexCountDistinctScan2) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDistinctScan(op *DistinctScan) (any, error) {
	this.record(op.getBase())
	_, e := op.scan.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitUnionScan(op *UnionScan) (any, error) {
	this.record(op.getBase())
	for _, o := range op.scans {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIntersectScan(op *IntersectScan) (any, error) {
	this.record(op.getBase())
	for _, o := range op.scans {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitOrderedIntersectScan(op *OrderedIntersectScan) (any, error) {
	this.record(op.getBase())
	for _, o := range op.scans {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitExpressionScan(op *ExpressionScan) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexFtsSearch(op *IndexFtsSearch) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitFetch(op *Fetch) (any, error) {
	this.record(op.getBase())
	if op.outDocs > _LARGE_COUNT {
		this.add("High fetch count")
	}
	return nil, nil
}

func (this *execAnalyser) VisitDummyFetch(op *DummyFetch) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitJoin(op *Join) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		perc := float64(op.outDocs) / float64(op.inDocs)
		if perc < 0.1 {
			this.add("Join eliminating over 90%")
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIndexJoin(op *IndexJoin) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitNest(op *Nest) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexNest(op *IndexNest) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUnnest(op *Unnest) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitNLJoin(op *NLJoin) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		perc := float64(op.outDocs) / float64(op.inDocs)
		if perc < 0.1 {
			this.add("Nested loop join eliminating over 90%")
		}
	}
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitNLNest(op *NLNest) (any, error) {
	this.record(op.getBase())
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitHashJoin(op *HashJoin) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		perc := float64(op.outDocs) / float64(op.inDocs)
		if perc < 0.1 {
			this.add("Hash join eliminating over 90%")
		}
	}
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitHashNest(op *HashNest) (any, error) {
	this.record(op.getBase())
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitLet(op *Let) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitWith(op *With) (any, error) {
	this.record(op.getBase())
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitFilter(op *Filter) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		perc := float64(op.outDocs) / float64(op.inDocs)
		if perc < 0.1 {
			this.add("Filter eliminating over 90%")
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitInitialGroup(op *InitialGroup) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIntermediateGroup(op *IntermediateGroup) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitFinalGroup(op *FinalGroup) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitWindowAggregate(op *WindowAggregate) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitInitialProject(op *InitialProject) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountProject(op *IndexCountProject) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDistinct(op *Distinct) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAll(op *All) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUnionAll(op *UnionAll) (any, error) {
	this.record(op.getBase())
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIntersect(op *Intersect) (any, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIntersectAll(op *IntersectAll) (any, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitExcept(op *Except) (any, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitExceptAll(op *ExceptAll) (any, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitOrder(op *Order) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		this.add("Large sort")
	}
	return nil, nil
}

func (this *execAnalyser) VisitOffset(op *Offset) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitLimit(op *Limit) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendInsert(op *SendInsert) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendUpsert(op *SendUpsert) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendDelete(op *SendDelete) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitClone(op *Clone) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSet(op *Set) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUnset(op *Unset) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendUpdate(op *SendUpdate) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitMerge(op *Merge) (any, error) {
	this.record(op.getBase())
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitAlias(op *Alias) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAuthorize(op *Authorize) (any, error) {
	this.record(op.getBase())
	if op.servTime > _AUTH_THRESHOLD {
		this.add("Slow auth")
	}
	return op.child.Accept(this)
}

func (this *execAnalyser) VisitParallel(op *Parallel) (any, error) {
	this.record(op.getBase())
	if len(op.children) == 1 {
		_, e := op.child.Accept(this)
		if e != nil {
			return nil, e
		}
	} else {
		for _, o := range op.children {
			_, e := o.Accept(this)
			if e != nil {
				return nil, e
			}
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitSequence(op *Sequence) (any, error) {
	this.record(op.getBase())
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitDiscard(op *Discard) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		this.add("Large number of discarded results")
	}
	return nil, nil
}

func (this *execAnalyser) VisitStream(op *Stream) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCollect(op *Collect) (any, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		this.add("Large sub-query result")
	}
	return nil, nil
}

func (this *execAnalyser) VisitReceive(op *Receive) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitChannel(op *Channel) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreatePrimaryIndex(op *CreatePrimaryIndex) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateIndex(op *CreateIndex) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropIndex(op *DropIndex) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterIndex(op *AlterIndex) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitBuildIndexes(op *BuildIndexes) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateScope(op *CreateScope) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropScope(op *DropScope) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateCollection(op *CreateCollection) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterCollection(op *AlterCollection) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropCollection(op *DropCollection) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitFlushCollection(op *FlushCollection) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitGrantRole(op *GrantRole) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitRevokeRole(op *RevokeRole) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExplain(op *Explain) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExplainFunction(op *ExplainFunction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitPrepare(op *Prepare) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitInferKeyspace(op *InferKeyspace) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitInferExpression(op *InferExpression) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateFunction(op *CreateFunction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropFunction(op *DropFunction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExecuteFunction(op *ExecuteFunction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexAdvice(op *IndexAdvice) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAdvise(op *Advise) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUpdateStatistics(op *UpdateStatistics) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitStartTransaction(op *StartTransaction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCommitTransaction(op *CommitTransaction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitRollbackTransaction(op *RollbackTransaction) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitTransactionIsolation(op *TransactionIsolation) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSavepoint(op *Savepoint) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateSequence(op *CreateSequence) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateKnowledge(op *CreateKnowledge) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropKnowledge(op *DropKnowledge) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropSequence(op *DropSequence) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterSequence(op *AlterSequence) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateCredentialStore(op *CreateCredentialStore) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterCredentialStore(op *AlterCredentialStore) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropCredentialStore(op *DropCredentialStore) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterBucket(op *AlterBucket) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateBucket(op *CreateBucket) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropBucket(op *DropBucket) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateCatalog(op *CreateCatalog) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterCatalog(op *AlterCatalog) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropCatalog(op *DropCatalog) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterGroup(op *AlterGroup) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateGroup(op *CreateGroup) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropGroup(op *DropGroup) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterUser(op *AlterUser) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateUser(op *CreateUser) (any, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropUser(op *DropUser) (any, error) {
	this.record(op.getBase())
	return nil, nil
}
