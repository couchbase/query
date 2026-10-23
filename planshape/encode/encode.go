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

func (this *planShape) VisitPrimaryScan(op *execution.PrimaryScan) (any, error) {
	plan := op.PlanOp().(*plan.PrimaryScan)
	this.add(planshape.PRIMARYSCAN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Indexer().KeyspaceId())
	return nil, nil
}

func (this *planShape) VisitPrimaryScan3(op *execution.PrimaryScan3) (any, error) {
	plan := op.PlanOp().(*plan.PrimaryScan3)
	this.add(planshape.PRIMARYSCAN3)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Indexer().KeyspaceId())
	return nil, nil
}

func (this *planShape) VisitIndexScan(op *execution.IndexScan) (any, error) {
	plan := op.PlanOp().(*plan.IndexScan)
	this.add(planshape.INDEXSCAN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexScan2(op *execution.IndexScan2) (any, error) {
	plan := op.PlanOp().(*plan.IndexScan2)
	this.add(planshape.INDEXSCAN2)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexScan3(op *execution.IndexScan3) (any, error) {
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

func (this *planShape) VisitKeyScan(op *execution.KeyScan) (any, error) {
	this.add(planshape.KEYSCAN)
	return nil, nil
}

func (this *planShape) VisitValueScan(op *execution.ValueScan) (any, error) {
	this.add(planshape.VALUESCAN)
	return nil, nil
}

func (this *planShape) VisitDummyScan(op *execution.DummyScan) (any, error) {
	this.add(planshape.DUMMYSCAN)
	return nil, nil
}

func (this *planShape) VisitExternalScan(op *execution.ExternalScan) (any, error) {
	this.add(planshape.EXTERNALSCAN)
	return nil, nil
}

func (this *planShape) VisitCountScan(op *execution.CountScan) (any, error) {
	plan := op.PlanOp().(*plan.CountScan)
	this.add(planshape.COUNTSCAN)
	this.adds(plan.Term().Alias())
	return nil, nil
}

func (this *planShape) VisitIndexCountScan(op *execution.IndexCountScan) (any, error) {
	plan := op.PlanOp().(*plan.IndexCountScan)
	this.add(planshape.INDEXCOUNTSCAN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexCountScan2(op *execution.IndexCountScan2) (any, error) {
	plan := op.PlanOp().(*plan.IndexCountScan2)
	this.add(planshape.INDEXCOUNTSCAN2)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitIndexCountDistinctScan2(op *execution.IndexCountDistinctScan2) (any, error) {
	plan := op.PlanOp().(*plan.IndexCountDistinctScan2)
	this.add(planshape.INDEXCOUNTDISTINCTSCAN2)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitDistinctScan(op *execution.DistinctScan) (any, error) {
	this.add(planshape.DISTINCTSCAN)
	_, e := op.Scan().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitUnionScan(op *execution.UnionScan) (any, error) {
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

func (this *planShape) VisitIntersectScan(op *execution.IntersectScan) (any, error) {
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

func (this *planShape) VisitOrderedIntersectScan(op *execution.OrderedIntersectScan) (any, error) {
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

func (this *planShape) VisitExpressionScan(op *execution.ExpressionScan) (any, error) {
	this.add(planshape.EXPRESSIONSCAN)
	return nil, nil
}

func (this *planShape) VisitIndexFtsSearch(op *execution.IndexFtsSearch) (any, error) {
	this.add(planshape.INDEXFTSSEARCH)
	return nil, nil
}

func (this *planShape) VisitFetch(op *execution.Fetch) (any, error) {
	plan := op.PlanOp().(*plan.Fetch)
	this.add(planshape.FETCH)
	this.adds(plan.Term().Alias())
	return nil, nil
}

func (this *planShape) VisitDummyFetch(op *execution.DummyFetch) (any, error) {
	this.add(planshape.DUMMYFETCH)
	return nil, nil
}

func (this *planShape) VisitJoin(op *execution.Join) (any, error) {
	this.add(planshape.JOIN)
	return nil, nil
}

func (this *planShape) VisitIndexJoin(op *execution.IndexJoin) (any, error) {
	plan := op.PlanOp().(*plan.IndexJoin)
	this.add(planshape.INDEXJOIN)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitNest(op *execution.Nest) (any, error) {
	this.add(planshape.NEST)
	return nil, nil
}

func (this *planShape) VisitIndexNest(op *execution.IndexNest) (any, error) {
	plan := op.PlanOp().(*plan.IndexNest)
	this.add(planshape.INDEXNEST)
	this.adds(plan.Index().Id())
	this.adds(plan.Index().Name())
	return nil, nil
}

func (this *planShape) VisitUnnest(op *execution.Unnest) (any, error) {
	this.add(planshape.UNNEST)
	return nil, nil
}

func (this *planShape) VisitNLJoin(op *execution.NLJoin) (any, error) {
	this.add(planshape.NLJOIN)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitNLNest(op *execution.NLNest) (any, error) {
	this.add(planshape.NLNEST)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitHashJoin(op *execution.HashJoin) (any, error) {
	this.add(planshape.HASHJOIN)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitHashNest(op *execution.HashNest) (any, error) {
	this.add(planshape.HASHNEST)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitLet(op *execution.Let) (any, error) {
	this.add(planshape.LET)
	return nil, nil
}

func (this *planShape) VisitWith(op *execution.With) (any, error) {
	this.add(planshape.WITH)
	_, e := op.Child().Accept(this)
	if e != nil {
		return nil, e
	}
	this.addNUL()
	return nil, nil
}

func (this *planShape) VisitFilter(op *execution.Filter) (any, error) {
	this.add(planshape.FILTER)
	return nil, nil
}

func (this *planShape) VisitInitialGroup(op *execution.InitialGroup) (any, error) {
	this.add(planshape.INITIALGROUP)
	return nil, nil
}

func (this *planShape) VisitIntermediateGroup(op *execution.IntermediateGroup) (any, error) {
	this.add(planshape.INTERMEDIATEGROUP)
	return nil, nil
}

func (this *planShape) VisitFinalGroup(op *execution.FinalGroup) (any, error) {
	this.add(planshape.FINALGROUP)
	return nil, nil
}

func (this *planShape) VisitWindowAggregate(op *execution.WindowAggregate) (any, error) {
	this.add(planshape.WINDOWAGGREGATE)
	return nil, nil
}

func (this *planShape) VisitInitialProject(op *execution.InitialProject) (any, error) {
	this.add(planshape.INITIALPROJECT)
	return nil, nil
}

func (this *planShape) VisitIndexCountProject(op *execution.IndexCountProject) (any, error) {
	this.add(planshape.INDEXCOUNTPROJECT)
	return nil, nil
}

func (this *planShape) VisitDistinct(op *execution.Distinct) (any, error) {
	this.add(planshape.DISTINCT)
	return nil, nil
}

func (this *planShape) VisitAll(op *execution.All) (any, error) {
	this.add(planshape.ALL)
	return nil, nil
}

func (this *planShape) VisitUnionAll(op *execution.UnionAll) (any, error) {
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

func (this *planShape) VisitIntersect(op *execution.Intersect) (any, error) {
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

func (this *planShape) VisitIntersectAll(op *execution.IntersectAll) (any, error) {
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

func (this *planShape) VisitExcept(op *execution.Except) (any, error) {
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

func (this *planShape) VisitExceptAll(op *execution.ExceptAll) (any, error) {
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

func (this *planShape) VisitOrder(op *execution.Order) (any, error) {
	this.add(planshape.ORDER)
	return nil, nil
}

func (this *planShape) VisitOffset(op *execution.Offset) (any, error) {
	this.add(planshape.OFFSET)
	return nil, nil
}

func (this *planShape) VisitLimit(op *execution.Limit) (any, error) {
	this.add(planshape.LIMIT)
	return nil, nil
}

func (this *planShape) VisitSendInsert(op *execution.SendInsert) (any, error) {
	this.add(planshape.SENDINSERT)
	return nil, nil
}

func (this *planShape) VisitSendUpsert(op *execution.SendUpsert) (any, error) {
	this.add(planshape.SENDUPSERT)
	return nil, nil
}

func (this *planShape) VisitSendDelete(op *execution.SendDelete) (any, error) {
	this.add(planshape.SENDDELETE)
	return nil, nil
}

func (this *planShape) VisitClone(op *execution.Clone) (any, error) {
	this.add(planshape.CLONE)
	return nil, nil
}

func (this *planShape) VisitSet(op *execution.Set) (any, error) {
	this.add(planshape.SET)
	return nil, nil
}

func (this *planShape) VisitUnset(op *execution.Unset) (any, error) {
	this.add(planshape.UNSET)
	return nil, nil
}

func (this *planShape) VisitSendUpdate(op *execution.SendUpdate) (any, error) {
	this.add(planshape.SENDUPDATE)
	return nil, nil
}

func (this *planShape) VisitMerge(op *execution.Merge) (any, error) {
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

func (this *planShape) VisitAlias(op *execution.Alias) (any, error) {
	this.add(planshape.ALIAS)
	return nil, nil
}

func (this *planShape) VisitAuthorize(op *execution.Authorize) (any, error) {
	// don't include this operator
	return op.Child().Accept(this)
}

func (this *planShape) VisitParallel(op *execution.Parallel) (any, error) {
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

func (this *planShape) VisitSequence(op *execution.Sequence) (any, error) {
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

func (this *planShape) VisitDiscard(op *execution.Discard) (any, error) {
	this.add(planshape.DISCARD)
	return nil, nil
}

func (this *planShape) VisitStream(op *execution.Stream) (any, error) {
	this.add(planshape.STREAM)
	return nil, nil
}

func (this *planShape) VisitCollect(op *execution.Collect) (any, error) {
	this.add(planshape.COLLECT)
	return nil, nil
}

func (this *planShape) VisitReceive(op *execution.Receive) (any, error) {
	this.add(planshape.RECEIVE)
	return nil, nil
}

func (this *planShape) VisitChannel(op *execution.Channel) (any, error) {
	this.add(planshape.CHANNEL)
	return nil, nil
}

func (this *planShape) VisitCreatePrimaryIndex(op *execution.CreatePrimaryIndex) (any, error) {
	this.add(planshape.CREATEPRIMARYINDEX)
	return nil, nil
}

func (this *planShape) VisitCreateIndex(op *execution.CreateIndex) (any, error) {
	this.add(planshape.CREATEINDEX)
	return nil, nil
}

func (this *planShape) VisitDropIndex(op *execution.DropIndex) (any, error) {
	this.add(planshape.DROPINDEX)
	return nil, nil
}

func (this *planShape) VisitAlterIndex(op *execution.AlterIndex) (any, error) {
	this.add(planshape.ALTERINDEX)
	return nil, nil
}

func (this *planShape) VisitBuildIndexes(op *execution.BuildIndexes) (any, error) {
	this.add(planshape.BUILDINDEXES)
	return nil, nil
}

func (this *planShape) VisitCreateScope(op *execution.CreateScope) (any, error) {
	this.add(planshape.CREATESCOPE)
	return nil, nil
}

func (this *planShape) VisitDropScope(op *execution.DropScope) (any, error) {
	this.add(planshape.DROPSCOPE)
	return nil, nil
}

func (this *planShape) VisitCreateCollection(op *execution.CreateCollection) (any, error) {
	this.add(planshape.CREATECOLLECTION)
	return nil, nil
}

func (this *planShape) VisitAlterCollection(op *execution.AlterCollection) (any, error) {
	this.add(planshape.ALTERCOLLECTION)
	return nil, nil
}

func (this *planShape) VisitDropCollection(op *execution.DropCollection) (any, error) {
	this.add(planshape.DROPCOLLECTION)
	return nil, nil
}

func (this *planShape) VisitFlushCollection(op *execution.FlushCollection) (any, error) {
	this.add(planshape.FLUSHCOLLECTION)
	return nil, nil
}

func (this *planShape) VisitGrantRole(op *execution.GrantRole) (any, error) {
	this.add(planshape.GRANTROLE)
	return nil, nil
}

func (this *planShape) VisitRevokeRole(op *execution.RevokeRole) (any, error) {
	this.add(planshape.REVOKEROLE)
	return nil, nil
}

func (this *planShape) VisitExplain(op *execution.Explain) (any, error) {
	this.add(planshape.EXPLAIN)
	return nil, nil
}

func (this *planShape) VisitExplainFunction(op *execution.ExplainFunction) (any, error) {
	this.add(planshape.EXPLAINFUNCTION)
	return nil, nil
}

func (this *planShape) VisitPrepare(op *execution.Prepare) (any, error) {
	this.add(planshape.PREPARE)
	return nil, nil
}

func (this *planShape) VisitInferKeyspace(op *execution.InferKeyspace) (any, error) {
	this.add(planshape.INFERKEYSPACE)
	return nil, nil
}

func (this *planShape) VisitInferExpression(op *execution.InferExpression) (any, error) {
	this.add(planshape.INFEREXPRESSION)
	return nil, nil
}

func (this *planShape) VisitCreateFunction(op *execution.CreateFunction) (any, error) {
	this.add(planshape.CREATEFUNCTION)
	return nil, nil
}

func (this *planShape) VisitDropFunction(op *execution.DropFunction) (any, error) {
	this.add(planshape.DROPFUNCTION)
	return nil, nil
}

func (this *planShape) VisitExecuteFunction(op *execution.ExecuteFunction) (any, error) {
	this.add(planshape.EXECUTEFUNCTION)
	return nil, nil
}

func (this *planShape) VisitIndexAdvice(op *execution.IndexAdvice) (any, error) {
	this.add(planshape.INDEXADVICE)
	return nil, nil
}

func (this *planShape) VisitAdvise(op *execution.Advise) (any, error) {
	this.add(planshape.ADVISE)
	return nil, nil
}

func (this *planShape) VisitUpdateStatistics(op *execution.UpdateStatistics) (any, error) {
	this.add(planshape.UPDATESTATISTICS)
	return nil, nil
}

func (this *planShape) VisitStartTransaction(op *execution.StartTransaction) (any, error) {
	this.add(planshape.STARTTRANSACTION)
	return nil, nil
}

func (this *planShape) VisitCommitTransaction(op *execution.CommitTransaction) (any, error) {
	this.add(planshape.COMMITTRANSACTION)
	return nil, nil
}

func (this *planShape) VisitRollbackTransaction(op *execution.RollbackTransaction) (any, error) {
	this.add(planshape.ROLLBACKTRANSACTION)
	return nil, nil
}

func (this *planShape) VisitTransactionIsolation(op *execution.TransactionIsolation) (any, error) {
	this.add(planshape.TRANSACTIONISOLATION)
	return nil, nil
}

func (this *planShape) VisitSavepoint(op *execution.Savepoint) (any, error) {
	this.add(planshape.SAVEPOINT)
	return nil, nil
}

func (this *planShape) VisitCreateSequence(op *execution.CreateSequence) (any, error) {
	this.add(planshape.CREATESEQUENCE)
	return nil, nil
}

func (this *planShape) VisitCreateKnowledge(op *execution.CreateKnowledge) (any, error) {
	this.add(planshape.CREATEKNOWLEDGE)
	return nil, nil
}

func (this *planShape) VisitDropKnowledge(op *execution.DropKnowledge) (any, error) {
	this.add(planshape.DROPKNOWLEDGE)
	return nil, nil
}

func (this *planShape) VisitDropSequence(op *execution.DropSequence) (any, error) {
	this.add(planshape.DROPSEQUENCE)
	return nil, nil
}

func (this *planShape) VisitAlterSequence(op *execution.AlterSequence) (any, error) {
	this.add(planshape.ALTERSEQUENCE)
	return nil, nil
}

func (this *planShape) VisitCreateCredentialStore(op *execution.CreateCredentialStore) (any, error) {
	this.add(planshape.CREATECREDENTIALSTORE)
	return nil, nil
}

func (this *planShape) VisitAlterCredentialStore(op *execution.AlterCredentialStore) (any, error) {
	this.add(planshape.ALTERCREDENTIALSTORE)
	return nil, nil
}

func (this *planShape) VisitDropCredentialStore(op *execution.DropCredentialStore) (any, error) {
	this.add(planshape.DROPCREDENTIALSTORE)
	return nil, nil
}

func (this *planShape) VisitCreateBucket(op *execution.CreateBucket) (any, error) {
	this.add(planshape.CREATEBUCKET)
	return nil, nil
}

func (this *planShape) VisitDropBucket(op *execution.DropBucket) (any, error) {
	this.add(planshape.DROPBUCKET)
	return nil, nil
}

func (this *planShape) VisitAlterBucket(op *execution.AlterBucket) (any, error) {
	this.add(planshape.ALTERBUCKET)
	return nil, nil
}

func (this *planShape) VisitCreateCatalog(op *execution.CreateCatalog) (any, error) {
	this.add(planshape.CREATECATALOG)
	return nil, nil
}

func (this *planShape) VisitDropCatalog(op *execution.DropCatalog) (any, error) {
	this.add(planshape.DROPCATALOG)
	return nil, nil
}

func (this *planShape) VisitAlterCatalog(op *execution.AlterCatalog) (any, error) {
	this.add(planshape.ALTERCATALOG)
	return nil, nil
}

func (this *planShape) VisitCreateGroup(op *execution.CreateGroup) (any, error) {
	this.add(planshape.CREATEGROUP)
	return nil, nil
}

func (this *planShape) VisitDropGroup(op *execution.DropGroup) (any, error) {
	this.add(planshape.DROPGROUP)
	return nil, nil
}

func (this *planShape) VisitAlterGroup(op *execution.AlterGroup) (any, error) {
	this.add(planshape.ALTERGROUP)
	return nil, nil
}

func (this *planShape) VisitCreateUser(op *execution.CreateUser) (any, error) {
	this.add(planshape.CREATEUSER)
	return nil, nil
}

func (this *planShape) VisitDropUser(op *execution.DropUser) (any, error) {
	this.add(planshape.DROPUSER)
	return nil, nil
}

func (this *planShape) VisitAlterUser(op *execution.AlterUser) (any, error) {
	this.add(planshape.ALTERUSER)
	return nil, nil
}
