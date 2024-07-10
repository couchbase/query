//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/plan"
)

func getSeqScanPrivs(plop plan.Operator, privs *auth.Privileges) error {
	collector := &collector{privs}
	_, err := plop.Accept(collector)
	if err != nil {
		return err
	}
	return nil
}

type collector struct {
	privs *auth.Privileges
}

func (this *collector) VisitPrimaryScan(plop *plan.PrimaryScan) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitPrimaryScan3(plop *plan.PrimaryScan3) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexScan(plop *plan.IndexScan) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexScan2(plop *plan.IndexScan2) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexScan3(plop *plan.IndexScan3) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexCountScan(plop *plan.IndexCountScan) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexCountScan2(plop *plan.IndexCountScan2) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexCountDistinctScan2(plop *plan.IndexCountDistinctScan2) (interface{}, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitKeyScan(plop *plan.KeyScan) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitExpressionScan(plop *plan.ExpressionScan) (interface{}, error) {
	if plop.SubqueryPlan() != nil {
		_, err := plop.SubqueryPlan().Accept(this)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (this *collector) VisitValueScan(plop *plan.ValueScan) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDummyScan(plop *plan.DummyScan) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCountScan(plop *plan.CountScan) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDistinctScan(plop *plan.DistinctScan) (interface{}, error) {
	_, err := plop.Scan().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitUnionScan(plop *plan.UnionScan) (interface{}, error) {
	for _, child := range plop.Scans() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitIntersectScan(plop *plan.IntersectScan) (interface{}, error) {
	for _, child := range plop.Scans() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitOrderedIntersectScan(plop *plan.OrderedIntersectScan) (interface{}, error) {
	for _, child := range plop.Scans() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitFetch(plop *plan.Fetch) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDummyFetch(plop *plan.DummyFetch) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitJoin(plop *plan.Join) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitIndexJoin(plop *plan.IndexJoin) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitNLJoin(plop *plan.NLJoin) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitHashJoin(plop *plan.HashJoin) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitNest(plop *plan.Nest) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitIndexNest(plop *plan.IndexNest) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitNLNest(plop *plan.NLNest) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitHashNest(plop *plan.HashNest) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitUnnest(plop *plan.Unnest) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitLet(plop *plan.Let) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitWith(plop *plan.With) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitFilter(plop *plan.Filter) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitInitialGroup(plop *plan.InitialGroup) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitIntermediateGroup(plop *plan.IntermediateGroup) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitFinalGroup(plop *plan.FinalGroup) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitWindowAggregate(plop *plan.WindowAggregate) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitInitialProject(plop *plan.InitialProject) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitFinalProject(plop *plan.FinalProject) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitIndexCountProject(plop *plan.IndexCountProject) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDistinct(plop *plan.Distinct) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitAll(plop *plan.All) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitUnionAll(plop *plan.UnionAll) (interface{}, error) {
	for _, child := range plop.Children() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitIntersectAll(plop *plan.IntersectAll) (interface{}, error) {
	_, e := plop.First().Accept(this)
	if e != nil {
		return nil, e
	}
	_, e = plop.Second().Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *collector) VisitExceptAll(plop *plan.ExceptAll) (interface{}, error) {
	_, e := plop.First().Accept(this)
	if e != nil {
		return nil, e
	}
	_, e = plop.Second().Accept(this)
	if e != nil {
		return nil, e
	}
	return nil, nil
}

func (this *collector) VisitOrder(plop *plan.Order) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitOffset(plop *plan.Offset) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitLimit(plop *plan.Limit) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitSendInsert(plop *plan.SendInsert) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitSendUpsert(plop *plan.SendUpsert) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitSendDelete(plop *plan.SendDelete) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitClone(plop *plan.Clone) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitSet(plop *plan.Set) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitUnset(plop *plan.Unset) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitSendUpdate(plop *plan.SendUpdate) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitMerge(plop *plan.Merge) (interface{}, error) {
	if plop.Update() != nil {
		_, e := plop.Update().Accept(this)
		if e != nil {
			return nil, e
		}
	}
	if plop.Delete() != nil {
		_, e := plop.Delete().Accept(this)
		if e != nil {
			return nil, e
		}
	}
	if plop.Insert() != nil {
		_, e := plop.Insert().Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitAlias(plop *plan.Alias) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitAuthorize(plop *plan.Authorize) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitParallel(plop *plan.Parallel) (interface{}, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitSequence(plop *plan.Sequence) (interface{}, error) {
	for _, pchild := range plop.Children() {
		_, err := pchild.Accept(this)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (this *collector) VisitDiscard(plop *plan.Discard) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitStream(plop *plan.Stream) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCollect(plop *plan.Collect) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitReceive(plop *plan.Receive) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCreatePrimaryIndex(plop *plan.CreatePrimaryIndex) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitGrantRole(plop *plan.GrantRole) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitRevokeRole(plop *plan.RevokeRole) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCreateIndex(plop *plan.CreateIndex) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDropIndex(plop *plan.DropIndex) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitAlterIndex(plop *plan.AlterIndex) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitBuildIndexes(plop *plan.BuildIndexes) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCreateScope(plop *plan.CreateScope) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDropScope(plop *plan.DropScope) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCreateCollection(plop *plan.CreateCollection) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDropCollection(plop *plan.DropCollection) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitFlushCollection(plop *plan.FlushCollection) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitPrepare(plop *plan.Prepare) (interface{}, error) {
	_, err := plop.Plan().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitExplain(plop *plan.Explain) (interface{}, error) {
	op := plop.Plan()
	if op != nil {
		return op.Accept(this)
	}
	return nil, nil
}

func (this *collector) VisitExplainFunction(plop *plan.ExplainFunction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitInferKeyspace(plop *plan.InferKeyspace) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitInferExpression(plop *plan.InferExpression) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCreateFunction(plop *plan.CreateFunction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDropFunction(plop *plan.DropFunction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitExecuteFunction(plop *plan.ExecuteFunction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitIndexFtsSearch(plop *plan.IndexFtsSearch) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitIndexAdvice(plop *plan.IndexAdvice) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitAdvise(plop *plan.Advise) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitUpdateStatistics(plop *plan.UpdateStatistics) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitStartTransaction(plop *plan.StartTransaction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCommitTransaction(plop *plan.CommitTransaction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitRollbackTransaction(plop *plan.RollbackTransaction) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitTransactionIsolation(plop *plan.TransactionIsolation) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitSavepoint(plop *plan.Savepoint) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitCreateSequence(plop *plan.CreateSequence) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitDropSequence(plop *plan.DropSequence) (interface{}, error) {
	return nil, nil
}

func (this *collector) VisitAlterSequence(plop *plan.AlterSequence) (interface{}, error) {
	return nil, nil
}
