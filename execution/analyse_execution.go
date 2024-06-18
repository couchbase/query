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
	_LARGE_COUNT    = 10000
	_WT_THRESHOLD   = 10
	_IO_THRESHOLD   = 2
	_AUTH_THRESHOLD = time.Second
	_IS_THRESHOLD   = 0.9
)

func AnalyseExecution(start Operator) ([]interface{}, error) {
	a := &execAnalyser{}

	_, err := start.Accept(a)
	if err != nil {
		return nil, err
	}

	if a.ioTime > a.cpuTime*_IO_THRESHOLD {
		a.add("High IO time")
	}
	if a.waitTime > (a.cpuTime+a.ioTime)*_WT_THRESHOLD {
		a.add("High wait time")
	}
	if a.indexScanTime > time.Duration(float64(a.ioTime)*_IS_THRESHOLD) {
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
	results []interface{}

	cpuTime       time.Duration
	ioTime        time.Duration
	waitTime      time.Duration
	indexScanTime time.Duration
}

func (this *execAnalyser) add(s string) {
	if this.results == nil {
		this.results = make([]interface{}, 0, 10)
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
	this.cpuTime += op.execTime
	this.ioTime += op.servTime
	this.waitTime += op.kernTime
}

func (this *execAnalyser) VisitPrimaryScan(op *PrimaryScan) (interface{}, error) {
	this.record(op.getBase())
	if op.outDocs > _LARGE_COUNT {
		this.add("High primary scan count")
	}
	return nil, nil
}

func (this *execAnalyser) VisitPrimaryScan3(op *PrimaryScan3) (interface{}, error) {
	this.record(op.getBase())
	if op.outDocs > _LARGE_COUNT {
		this.add("High primary scan count")
	}
	return nil, nil
}

func (this *execAnalyser) VisitIndexScan(op *IndexScan) (interface{}, error) {
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

func (this *execAnalyser) VisitIndexScan2(op *IndexScan2) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexScan3(op *IndexScan3) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitKeyScan(op *KeyScan) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitValueScan(op *ValueScan) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDummyScan(op *DummyScan) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCountScan(op *CountScan) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountScan(op *IndexCountScan) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountScan2(op *IndexCountScan2) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountDistinctScan2(op *IndexCountDistinctScan2) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDistinctScan(op *DistinctScan) (interface{}, error) {
	this.record(op.getBase())
	_, e := op.scan.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitUnionScan(op *UnionScan) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range op.scans {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIntersectScan(op *IntersectScan) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range op.scans {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitOrderedIntersectScan(op *OrderedIntersectScan) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range op.scans {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitExpressionScan(op *ExpressionScan) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexFtsSearch(op *IndexFtsSearch) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitFetch(op *Fetch) (interface{}, error) {
	this.record(op.getBase())
	if op.outDocs > _LARGE_COUNT {
		this.add("High fetch count")
	}
	return nil, nil
}

func (this *execAnalyser) VisitDummyFetch(op *DummyFetch) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitJoin(op *Join) (interface{}, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		perc := float64(op.outDocs) / float64(op.inDocs)
		if perc < 0.1 {
			this.add("Join eliminating over 90%")
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIndexJoin(op *IndexJoin) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitNest(op *Nest) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexNest(op *IndexNest) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUnnest(op *Unnest) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitNLJoin(op *NLJoin) (interface{}, error) {
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

func (this *execAnalyser) VisitNLNest(op *NLNest) (interface{}, error) {
	this.record(op.getBase())
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitHashJoin(op *HashJoin) (interface{}, error) {
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

func (this *execAnalyser) VisitHashNest(op *HashNest) (interface{}, error) {
	this.record(op.getBase())
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitLet(op *Let) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitWith(op *With) (interface{}, error) {
	this.record(op.getBase())
	_, e := op.child.Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *execAnalyser) VisitFilter(op *Filter) (interface{}, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		perc := float64(op.outDocs) / float64(op.inDocs)
		if perc < 0.1 {
			this.add("Filter eliminating over 90%")
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitInitialGroup(op *InitialGroup) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIntermediateGroup(op *IntermediateGroup) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitFinalGroup(op *FinalGroup) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitWindowAggregate(op *WindowAggregate) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitInitialProject(op *InitialProject) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexCountProject(op *IndexCountProject) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDistinct(op *Distinct) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAll(op *All) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUnionAll(op *UnionAll) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIntersect(op *Intersect) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitIntersectAll(op *IntersectAll) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitExcept(op *Except) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitExceptAll(op *ExceptAll) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range []Operator{op.first, op.second} {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitOrder(op *Order) (interface{}, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		this.add("Large sort")
	}
	return nil, nil
}

func (this *execAnalyser) VisitOffset(op *Offset) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitLimit(op *Limit) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendInsert(op *SendInsert) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendUpsert(op *SendUpsert) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendDelete(op *SendDelete) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitClone(op *Clone) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSet(op *Set) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUnset(op *Unset) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSendUpdate(op *SendUpdate) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitMerge(op *Merge) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitAlias(op *Alias) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAuthorize(op *Authorize) (interface{}, error) {
	this.record(op.getBase())
	if op.servTime > _AUTH_THRESHOLD {
		this.add("Slow auth")
	}
	return op.child.Accept(this)
}

func (this *execAnalyser) VisitParallel(op *Parallel) (interface{}, error) {
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

func (this *execAnalyser) VisitSequence(op *Sequence) (interface{}, error) {
	this.record(op.getBase())
	for _, o := range op.children {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *execAnalyser) VisitDiscard(op *Discard) (interface{}, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		this.add("Large number of discarded results")
	}
	return nil, nil
}

func (this *execAnalyser) VisitStream(op *Stream) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCollect(op *Collect) (interface{}, error) {
	this.record(op.getBase())
	if op.inDocs > _LARGE_COUNT {
		this.add("Large sub-query result")
	}
	return nil, nil
}

func (this *execAnalyser) VisitReceive(op *Receive) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitChannel(op *Channel) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreatePrimaryIndex(op *CreatePrimaryIndex) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateIndex(op *CreateIndex) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropIndex(op *DropIndex) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterIndex(op *AlterIndex) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitBuildIndexes(op *BuildIndexes) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateScope(op *CreateScope) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropScope(op *DropScope) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateCollection(op *CreateCollection) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropCollection(op *DropCollection) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitFlushCollection(op *FlushCollection) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitGrantRole(op *GrantRole) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitRevokeRole(op *RevokeRole) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExplain(op *Explain) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExplainFunction(op *ExplainFunction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitPrepare(op *Prepare) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitInferKeyspace(op *InferKeyspace) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitInferExpression(op *InferExpression) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateFunction(op *CreateFunction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropFunction(op *DropFunction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitExecuteFunction(op *ExecuteFunction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitIndexAdvice(op *IndexAdvice) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAdvise(op *Advise) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitUpdateStatistics(op *UpdateStatistics) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitStartTransaction(op *StartTransaction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCommitTransaction(op *CommitTransaction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitRollbackTransaction(op *RollbackTransaction) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitTransactionIsolation(op *TransactionIsolation) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitSavepoint(op *Savepoint) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateSequence(op *CreateSequence) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropSequence(op *DropSequence) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterSequence(op *AlterSequence) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterBucket(op *AlterBucket) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateBucket(op *CreateBucket) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropBucket(op *DropBucket) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterGroup(op *AlterGroup) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateGroup(op *CreateGroup) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropGroup(op *DropGroup) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitAlterUser(op *AlterUser) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitCreateUser(op *CreateUser) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}

func (this *execAnalyser) VisitDropUser(op *DropUser) (interface{}, error) {
	this.record(op.getBase())
	return nil, nil
}
