//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package encode

import (
	"encoding/binary"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planshape"
)

func Encode(start execution.Operator, max int) []byte {
	if start == nil || max < 3 {
		return nil
	}
	ps := &planShape{make([]byte, 2, max)}
	binary.BigEndian.PutUint16(ps.o, planshape.MAGIC)
	start.Accept(ps)
	return ps.o
}

type planShape struct {
	o []byte
}

func (this *planShape) addNUL() {
	if len(this.o) < cap(this.o) {
		this.o = append(this.o, planshape.NUL)
	}
}

func (this *planShape) add(b byte) {
	if len(this.o) < cap(this.o) {
		this.o = append(this.o, b)
	}
}

func (this *planShape) adds(s string) {
	if len(this.o)+len(s)+1 <= cap(this.o) {
		this.o = append(append(this.o, []byte(s)...), planshape.NUL)
	}
}

func (this *planShape) VisitPrimaryScan(op *execution.PrimaryScan) (interface{}, error) {
	plan := op.PlanOp().(*plan.PrimaryScan)
	this.add(planshape.PRIMARYSCAN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Indexer().KeyspaceId())
	return nil, nil
}

func (this *planShape) VisitPrimaryScan3(op *execution.PrimaryScan3) (interface{}, error) {
	plan := op.PlanOp().(*plan.PrimaryScan3)
	this.add(planshape.PRIMARYSCAN3)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Indexer().KeyspaceId())
	return nil, nil
}

func (this *planShape) VisitIndexScan(op *execution.IndexScan) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexScan)
	this.add(planshape.INDEXSCAN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitIndexScan2(op *execution.IndexScan2) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexScan2)
	this.add(planshape.INDEXSCAN2)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexScan3(op *execution.IndexScan3) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexScan3)
	this.add(planshape.INDEXSCAN3)
	this.adds(plan.Index().Id())
	if plan.Index().Type() == datastore.SEQ_SCAN {
		this.adds(plan.Index().Indexer().KeyspaceId())
	} else {
		this.adds(plan.Index().Name())
	}
	var flags byte
	if plan.Offset() != nil {
		flags |= planshape.IDX_OFFSET
	}
	if plan.Limit() != nil {
		flags |= planshape.IDX_LIMIT
	}
	if plan.GroupAggs() != nil {
		flags |= planshape.IDX_GROUP
	}
	if plan.Covering() {
		flags |= planshape.IDX_COVER
	}
	if len(plan.OrderTerms()) > 0 {
		flags |= planshape.IDX_ORDER
	}
	this.add(flags)
	return nil, nil
}

func (this *planShape) VisitKeyScan(op *execution.KeyScan) (interface{}, error) {
	this.add(planshape.KEYSCAN)
	return nil, nil
}

func (this *planShape) VisitValueScan(op *execution.ValueScan) (interface{}, error) {
	this.add(planshape.VALUESCAN)
	return nil, nil
}

func (this *planShape) VisitDummyScan(op *execution.DummyScan) (interface{}, error) {
	this.add(planshape.DUMMYSCAN)
	return nil, nil
}

func (this *planShape) VisitCountScan(op *execution.CountScan) (interface{}, error) {
	plan := op.PlanOp().(*plan.CountScan)
	this.add(planshape.COUNTSCAN)
	this.adds(plan.Term().Alias())
	return nil, nil
}

func (this *planShape) VisitIndexCountScan(op *execution.IndexCountScan) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexCountScan)
	this.add(planshape.INDEXCOUNTSCAN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexCountScan2(op *execution.IndexCountScan2) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexCountScan2)
	this.add(planshape.INDEXCOUNTSCAN2)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexCountDistinctScan2(op *execution.IndexCountDistinctScan2) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexCountDistinctScan2)
	this.add(planshape.INDEXCOUNTDISTINCTSCAN2)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitDistinctScan(op *execution.DistinctScan) (interface{}, error) {
	this.add(planshape.DISTINCTSCAN)
	_, e := op.Scan().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitUnionScan(op *execution.UnionScan) (interface{}, error) {
	this.add(planshape.UNIONSCAN)
	for _, o := range op.Scans() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitIntersectScan(op *execution.IntersectScan) (interface{}, error) {
	this.add(planshape.INTERSECTSCAN)
	for _, o := range op.Scans() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitOrderedIntersectScan(op *execution.OrderedIntersectScan) (interface{}, error) {
	this.add(planshape.ORDEREDINTERSECTSCAN)
	for _, o := range op.Scans() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitExpressionScan(op *execution.ExpressionScan) (interface{}, error) {
	this.add(planshape.EXPRESSIONSCAN)
	return nil, nil
}

func (this *planShape) VisitIndexFtsSearch(op *execution.IndexFtsSearch) (interface{}, error) {
	this.add(planshape.INDEXFTSSEARCH)
	return nil, nil
}

func (this *planShape) VisitFetch(op *execution.Fetch) (interface{}, error) {
	plan := op.PlanOp().(*plan.Fetch)
	this.add(planshape.FETCH)
	this.adds(plan.Term().Alias())
	return nil, nil
}

func (this *planShape) VisitDummyFetch(op *execution.DummyFetch) (interface{}, error) {
	this.add(planshape.DUMMYFETCH)
	return nil, nil
}

func (this *planShape) VisitJoin(op *execution.Join) (interface{}, error) {
	this.add(planshape.JOIN)
	return nil, nil
}

func (this *planShape) VisitIndexJoin(op *execution.IndexJoin) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexJoin)
	this.add(planshape.INDEXJOIN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitNest(op *execution.Nest) (interface{}, error) {
	this.add(planshape.NEST)
	return nil, nil
}

func (this *planShape) VisitIndexNest(op *execution.IndexNest) (interface{}, error) {
	plan := op.PlanOp().(*plan.IndexNest)
	this.add(planshape.INDEXNEST)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitUnnest(op *execution.Unnest) (interface{}, error) {
	this.add(planshape.UNNEST)
	return nil, nil
}

func (this *planShape) VisitNLJoin(op *execution.NLJoin) (interface{}, error) {
	this.add(planshape.NLJOIN)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitNLNest(op *execution.NLNest) (interface{}, error) {
	this.add(planshape.NLNEST)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitHashJoin(op *execution.HashJoin) (interface{}, error) {
	this.add(planshape.HASHJOIN)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitHashNest(op *execution.HashNest) (interface{}, error) {
	this.add(planshape.HASHNEST)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitLet(op *execution.Let) (interface{}, error) {
	this.add(planshape.LET)
	return nil, nil
}

func (this *planShape) VisitWith(op *execution.With) (interface{}, error) {
	this.add(planshape.WITH)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitFilter(op *execution.Filter) (interface{}, error) {
	this.add(planshape.FILTER)
	return nil, nil
}

func (this *planShape) VisitInitialGroup(op *execution.InitialGroup) (interface{}, error) {
	this.add(planshape.INITIALGROUP)
	return nil, nil
}

func (this *planShape) VisitIntermediateGroup(op *execution.IntermediateGroup) (interface{}, error) {
	this.add(planshape.INTERMEDIATEGROUP)
	return nil, nil
}

func (this *planShape) VisitFinalGroup(op *execution.FinalGroup) (interface{}, error) {
	this.add(planshape.FINALGROUP)
	return nil, nil
}

func (this *planShape) VisitWindowAggregate(op *execution.WindowAggregate) (interface{}, error) {
	this.add(planshape.WINDOWAGGREGATE)
	return nil, nil
}

func (this *planShape) VisitInitialProject(op *execution.InitialProject) (interface{}, error) {
	this.add(planshape.INITIALPROJECT)
	return nil, nil
}

func (this *planShape) VisitIndexCountProject(op *execution.IndexCountProject) (interface{}, error) {
	this.add(planshape.INDEXCOUNTPROJECT)
	return nil, nil
}

func (this *planShape) VisitDistinct(op *execution.Distinct) (interface{}, error) {
	this.add(planshape.DISTINCT)
	return nil, nil
}

func (this *planShape) VisitAll(op *execution.All) (interface{}, error) {
	this.add(planshape.ALL)
	return nil, nil
}

func (this *planShape) VisitUnionAll(op *execution.UnionAll) (interface{}, error) {
	this.add(planshape.UNIONALL)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitIntersect(op *execution.Intersect) (interface{}, error) {
	this.add(planshape.INTERSECT)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitIntersectAll(op *execution.IntersectAll) (interface{}, error) {
	this.add(planshape.INTERSECTALL)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitExcept(op *execution.Except) (interface{}, error) {
	this.add(planshape.EXCEPT)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitExceptAll(op *execution.ExceptAll) (interface{}, error) {
	this.add(planshape.EXCEPTALL)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitOrder(op *execution.Order) (interface{}, error) {
	this.add(planshape.ORDER)
	return nil, nil
}

func (this *planShape) VisitOffset(op *execution.Offset) (interface{}, error) {
	this.add(planshape.OFFSET)
	return nil, nil
}

func (this *planShape) VisitLimit(op *execution.Limit) (interface{}, error) {
	this.add(planshape.LIMIT)
	return nil, nil
}

func (this *planShape) VisitSendInsert(op *execution.SendInsert) (interface{}, error) {
	this.add(planshape.SENDINSERT)
	return nil, nil
}

func (this *planShape) VisitSendUpsert(op *execution.SendUpsert) (interface{}, error) {
	this.add(planshape.SENDUPSERT)
	return nil, nil
}

func (this *planShape) VisitSendDelete(op *execution.SendDelete) (interface{}, error) {
	this.add(planshape.SENDDELETE)
	return nil, nil
}

func (this *planShape) VisitClone(op *execution.Clone) (interface{}, error) {
	this.add(planshape.CLONE)
	return nil, nil
}

func (this *planShape) VisitSet(op *execution.Set) (interface{}, error) {
	this.add(planshape.SET)
	return nil, nil
}

func (this *planShape) VisitUnset(op *execution.Unset) (interface{}, error) {
	this.add(planshape.UNSET)
	return nil, nil
}

func (this *planShape) VisitSendUpdate(op *execution.SendUpdate) (interface{}, error) {
	this.add(planshape.SENDUPDATE)
	return nil, nil
}

func (this *planShape) VisitMerge(op *execution.Merge) (interface{}, error) {
	this.add(planshape.MERGE)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitAlias(op *execution.Alias) (interface{}, error) {
	this.add(planshape.ALIAS)
	return nil, nil
}

func (this *planShape) VisitAuthorize(op *execution.Authorize) (interface{}, error) {
	// don't include this operator
	return op.Child().Accept(this)
}

func (this *planShape) VisitParallel(op *execution.Parallel) (interface{}, error) {
	this.add(planshape.PARALLEL)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitSequence(op *execution.Sequence) (interface{}, error) {
	this.add(planshape.SEQUENCE)
	for _, o := range op.Children() {
		_, e := o.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitDiscard(op *execution.Discard) (interface{}, error) {
	this.add(planshape.DISCARD)
	return nil, nil
}

func (this *planShape) VisitStream(op *execution.Stream) (interface{}, error) {
	this.add(planshape.STREAM)
	return nil, nil
}

func (this *planShape) VisitCollect(op *execution.Collect) (interface{}, error) {
	this.add(planshape.COLLECT)
	return nil, nil
}

func (this *planShape) VisitReceive(op *execution.Receive) (interface{}, error) {
	this.add(planshape.RECEIVE)
	return nil, nil
}

func (this *planShape) VisitChannel(op *execution.Channel) (interface{}, error) {
	this.add(planshape.CHANNEL)
	return nil, nil
}

func (this *planShape) VisitCreatePrimaryIndex(op *execution.CreatePrimaryIndex) (interface{}, error) {
	this.add(planshape.CREATEPRIMARYINDEX)
	return nil, nil
}

func (this *planShape) VisitCreateIndex(op *execution.CreateIndex) (interface{}, error) {
	this.add(planshape.CREATEINDEX)
	return nil, nil
}

func (this *planShape) VisitDropIndex(op *execution.DropIndex) (interface{}, error) {
	this.add(planshape.DROPINDEX)
	return nil, nil
}

func (this *planShape) VisitAlterIndex(op *execution.AlterIndex) (interface{}, error) {
	this.add(planshape.ALTERINDEX)
	return nil, nil
}

func (this *planShape) VisitBuildIndexes(op *execution.BuildIndexes) (interface{}, error) {
	this.add(planshape.BUILDINDEXES)
	return nil, nil
}

func (this *planShape) VisitCreateScope(op *execution.CreateScope) (interface{}, error) {
	this.add(planshape.CREATESCOPE)
	return nil, nil
}

func (this *planShape) VisitDropScope(op *execution.DropScope) (interface{}, error) {
	this.add(planshape.DROPSCOPE)
	return nil, nil
}

func (this *planShape) VisitCreateCollection(op *execution.CreateCollection) (interface{}, error) {
	this.add(planshape.CREATECOLLECTION)
	return nil, nil
}

func (this *planShape) VisitDropCollection(op *execution.DropCollection) (interface{}, error) {
	this.add(planshape.DROPCOLLECTION)
	return nil, nil
}

func (this *planShape) VisitFlushCollection(op *execution.FlushCollection) (interface{}, error) {
	this.add(planshape.FLUSHCOLLECTION)
	return nil, nil
}

func (this *planShape) VisitGrantRole(op *execution.GrantRole) (interface{}, error) {
	this.add(planshape.GRANTROLE)
	return nil, nil
}

func (this *planShape) VisitRevokeRole(op *execution.RevokeRole) (interface{}, error) {
	this.add(planshape.REVOKEROLE)
	return nil, nil
}

func (this *planShape) VisitExplain(op *execution.Explain) (interface{}, error) {
	this.add(planshape.EXPLAIN)
	return nil, nil
}

func (this *planShape) VisitExplainFunction(op *execution.ExplainFunction) (interface{}, error) {
	this.add(planshape.EXPLAINFUNCTION)
	return nil, nil
}

func (this *planShape) VisitPrepare(op *execution.Prepare) (interface{}, error) {
	this.add(planshape.PREPARE)
	return nil, nil
}

func (this *planShape) VisitInferKeyspace(op *execution.InferKeyspace) (interface{}, error) {
	this.add(planshape.INFERKEYSPACE)
	return nil, nil
}

func (this *planShape) VisitInferExpression(op *execution.InferExpression) (interface{}, error) {
	this.add(planshape.INFEREXPRESSION)
	return nil, nil
}

func (this *planShape) VisitCreateFunction(op *execution.CreateFunction) (interface{}, error) {
	this.add(planshape.CREATEFUNCTION)
	return nil, nil
}

func (this *planShape) VisitDropFunction(op *execution.DropFunction) (interface{}, error) {
	this.add(planshape.DROPFUNCTION)
	return nil, nil
}

func (this *planShape) VisitExecuteFunction(op *execution.ExecuteFunction) (interface{}, error) {
	this.add(planshape.EXECUTEFUNCTION)
	return nil, nil
}

func (this *planShape) VisitIndexAdvice(op *execution.IndexAdvice) (interface{}, error) {
	this.add(planshape.INDEXADVICE)
	return nil, nil
}

func (this *planShape) VisitAdvise(op *execution.Advise) (interface{}, error) {
	this.add(planshape.ADVISE)
	return nil, nil
}

func (this *planShape) VisitUpdateStatistics(op *execution.UpdateStatistics) (interface{}, error) {
	this.add(planshape.UPDATESTATISTICS)
	return nil, nil
}

func (this *planShape) VisitStartTransaction(op *execution.StartTransaction) (interface{}, error) {
	this.add(planshape.STARTTRANSACTION)
	return nil, nil
}

func (this *planShape) VisitCommitTransaction(op *execution.CommitTransaction) (interface{}, error) {
	this.add(planshape.COMMITTRANSACTION)
	return nil, nil
}

func (this *planShape) VisitRollbackTransaction(op *execution.RollbackTransaction) (interface{}, error) {
	this.add(planshape.ROLLBACKTRANSACTION)
	return nil, nil
}

func (this *planShape) VisitTransactionIsolation(op *execution.TransactionIsolation) (interface{}, error) {
	this.add(planshape.TRANSACTIONISOLATION)
	return nil, nil
}

func (this *planShape) VisitSavepoint(op *execution.Savepoint) (interface{}, error) {
	this.add(planshape.SAVEPOINT)
	return nil, nil
}

func (this *planShape) VisitCreateSequence(op *execution.CreateSequence) (interface{}, error) {
	this.add(planshape.CREATESEQUENCE)
	return nil, nil
}

func (this *planShape) VisitDropSequence(op *execution.DropSequence) (interface{}, error) {
	this.add(planshape.DROPSEQUENCE)
	return nil, nil
}

func (this *planShape) VisitAlterSequence(op *execution.AlterSequence) (interface{}, error) {
	this.add(planshape.ALTERSEQUENCE)
	return nil, nil
}

func (this *planShape) VisitCreateBucket(op *execution.CreateBucket) (interface{}, error) {
	this.add(planshape.CREATEBUCKET)
	return nil, nil
}

func (this *planShape) VisitDropBucket(op *execution.DropBucket) (interface{}, error) {
	this.add(planshape.DROPBUCKET)
	return nil, nil
}

func (this *planShape) VisitAlterBucket(op *execution.AlterBucket) (interface{}, error) {
	this.add(planshape.ALTERBUCKET)
	return nil, nil
}

func (this *planShape) VisitCreateGroup(op *execution.CreateGroup) (interface{}, error) {
	this.add(planshape.CREATEGROUP)
	return nil, nil
}

func (this *planShape) VisitDropGroup(op *execution.DropGroup) (interface{}, error) {
	this.add(planshape.DROPGROUP)
	return nil, nil
}

func (this *planShape) VisitAlterGroup(op *execution.AlterGroup) (interface{}, error) {
	this.add(planshape.ALTERGROUP)
	return nil, nil
}

func (this *planShape) VisitCreateUser(op *execution.CreateUser) (interface{}, error) {
	this.add(planshape.CREATEUSER)
	return nil, nil
}

func (this *planShape) VisitDropUser(op *execution.DropUser) (interface{}, error) {
	this.add(planshape.DROPUSER)
	return nil, nil
}

func (this *planShape) VisitAlterUser(op *execution.AlterUser) (interface{}, error) {
	this.add(planshape.ALTERUSER)
	return nil, nil
}
