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

func (this *collector) VisitPrimaryScan(plop *plan.PrimaryScan) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitPrimaryScan3(plop *plan.PrimaryScan3) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexScan(plop *plan.IndexScan) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexScan2(plop *plan.IndexScan2) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexScan3(plop *plan.IndexScan3) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexCountScan(plop *plan.IndexCountScan) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexCountScan2(plop *plan.IndexCountScan2) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitIndexCountDistinctScan2(plop *plan.IndexCountDistinctScan2) (any, error) {
	if plop.Index().Type() == datastore.SEQ_SCAN {
		pp := auth.PrivilegePair{
			Target: plop.Term().PathString(),
			Priv:   auth.PRIV_QUERY_SEQ_SCAN,
		}
		this.privs.AddPair(pp)
	}
	return nil, nil
}

func (this *collector) VisitKeyScan(plop *plan.KeyScan) (any, error) {
	return nil, nil
}

func (this *collector) VisitExpressionScan(plop *plan.ExpressionScan) (any, error) {
	if plop.SubqueryPlan() != nil {
		_, err := plop.SubqueryPlan().Accept(this)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (this *collector) VisitValueScan(plop *plan.ValueScan) (any, error) {
	return nil, nil
}

func (this *collector) VisitDummyScan(plop *plan.DummyScan) (any, error) {
	return nil, nil
}

func (this *collector) VisitExternalScan(plop *plan.ExternalScan) (any, error) {
	return nil, nil
}

func (this *collector) VisitCountScan(plop *plan.CountScan) (any, error) {
	return nil, nil
}

func (this *collector) VisitDistinctScan(plop *plan.DistinctScan) (any, error) {
	_, err := plop.Scan().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitUnionScan(plop *plan.UnionScan) (any, error) {
	for _, child := range plop.Scans() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitIntersectScan(plop *plan.IntersectScan) (any, error) {
	for _, child := range plop.Scans() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitOrderedIntersectScan(plop *plan.OrderedIntersectScan) (any, error) {
	for _, child := range plop.Scans() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitFetch(plop *plan.Fetch) (any, error) {
	return nil, nil
}

func (this *collector) VisitDummyFetch(plop *plan.DummyFetch) (any, error) {
	return nil, nil
}

func (this *collector) VisitJoin(plop *plan.Join) (any, error) {
	return nil, nil
}

func (this *collector) VisitIndexJoin(plop *plan.IndexJoin) (any, error) {
	return nil, nil
}

func (this *collector) VisitNLJoin(plop *plan.NLJoin) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitHashJoin(plop *plan.HashJoin) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitNest(plop *plan.Nest) (any, error) {
	return nil, nil
}

func (this *collector) VisitIndexNest(plop *plan.IndexNest) (any, error) {
	return nil, nil
}

func (this *collector) VisitNLNest(plop *plan.NLNest) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitHashNest(plop *plan.HashNest) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitUnnest(plop *plan.Unnest) (any, error) {
	return nil, nil
}

func (this *collector) VisitLet(plop *plan.Let) (any, error) {
	return nil, nil
}

func (this *collector) VisitWith(plop *plan.With) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitFilter(plop *plan.Filter) (any, error) {
	return nil, nil
}

func (this *collector) VisitInitialGroup(plop *plan.InitialGroup) (any, error) {
	return nil, nil
}

func (this *collector) VisitIntermediateGroup(plop *plan.IntermediateGroup) (any, error) {
	return nil, nil
}

func (this *collector) VisitFinalGroup(plop *plan.FinalGroup) (any, error) {
	return nil, nil
}

func (this *collector) VisitWindowAggregate(plop *plan.WindowAggregate) (any, error) {
	return nil, nil
}

func (this *collector) VisitInitialProject(plop *plan.InitialProject) (any, error) {
	return nil, nil
}

func (this *collector) VisitIndexCountProject(plop *plan.IndexCountProject) (any, error) {
	return nil, nil
}

func (this *collector) VisitDistinct(plop *plan.Distinct) (any, error) {
	return nil, nil
}

func (this *collector) VisitAll(plop *plan.All) (any, error) {
	return nil, nil
}

func (this *collector) VisitUnionAll(plop *plan.UnionAll) (any, error) {
	for _, child := range plop.Children() {
		_, e := child.Accept(this)
		if e != nil {
			return nil, e
		}
	}
	return nil, nil
}

func (this *collector) VisitIntersectAll(plop *plan.IntersectAll) (any, error) {
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

func (this *collector) VisitExceptAll(plop *plan.ExceptAll) (any, error) {
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

func (this *collector) VisitOrder(plop *plan.Order) (any, error) {
	return nil, nil
}

func (this *collector) VisitOffset(plop *plan.Offset) (any, error) {
	return nil, nil
}

func (this *collector) VisitLimit(plop *plan.Limit) (any, error) {
	return nil, nil
}

func (this *collector) VisitSendInsert(plop *plan.SendInsert) (any, error) {
	return nil, nil
}

func (this *collector) VisitSendUpsert(plop *plan.SendUpsert) (any, error) {
	return nil, nil
}

func (this *collector) VisitSendDelete(plop *plan.SendDelete) (any, error) {
	return nil, nil
}

func (this *collector) VisitClone(plop *plan.Clone) (any, error) {
	return nil, nil
}

func (this *collector) VisitSet(plop *plan.Set) (any, error) {
	return nil, nil
}

func (this *collector) VisitUnset(plop *plan.Unset) (any, error) {
	return nil, nil
}

func (this *collector) VisitSendUpdate(plop *plan.SendUpdate) (any, error) {
	return nil, nil
}

func (this *collector) VisitMerge(plop *plan.Merge) (any, error) {
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

func (this *collector) VisitAlias(plop *plan.Alias) (any, error) {
	return nil, nil
}

func (this *collector) VisitAuthorize(plop *plan.Authorize) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitParallel(plop *plan.Parallel) (any, error) {
	_, err := plop.Child().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitSequence(plop *plan.Sequence) (any, error) {
	for _, pchild := range plop.Children() {
		_, err := pchild.Accept(this)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (this *collector) VisitDiscard(plop *plan.Discard) (any, error) {
	return nil, nil
}

func (this *collector) VisitStream(plop *plan.Stream) (any, error) {
	return nil, nil
}

func (this *collector) VisitCollect(plop *plan.Collect) (any, error) {
	return nil, nil
}

func (this *collector) VisitReceive(plop *plan.Receive) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreatePrimaryIndex(plop *plan.CreatePrimaryIndex) (any, error) {
	return nil, nil
}

func (this *collector) VisitGrantRole(plop *plan.GrantRole) (any, error) {
	return nil, nil
}

func (this *collector) VisitRevokeRole(plop *plan.RevokeRole) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateIndex(plop *plan.CreateIndex) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropIndex(plop *plan.DropIndex) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterIndex(plop *plan.AlterIndex) (any, error) {
	return nil, nil
}

func (this *collector) VisitBuildIndexes(plop *plan.BuildIndexes) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateScope(plop *plan.CreateScope) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropScope(plop *plan.DropScope) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateCollection(plop *plan.CreateCollection) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropCollection(plop *plan.DropCollection) (any, error) {
	return nil, nil
}

func (this *collector) VisitFlushCollection(plop *plan.FlushCollection) (any, error) {
	return nil, nil
}

func (this *collector) VisitPrepare(plop *plan.Prepare) (any, error) {
	_, err := plop.Plan().Accept(this)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (this *collector) VisitExplain(plop *plan.Explain) (any, error) {
	op := plop.Plan()
	if op != nil {
		return op.Accept(this)
	}
	return nil, nil
}

func (this *collector) VisitExplainFunction(plop *plan.ExplainFunction) (any, error) {
	return nil, nil
}

func (this *collector) VisitInferKeyspace(plop *plan.InferKeyspace) (any, error) {
	return nil, nil
}

func (this *collector) VisitInferExpression(plop *plan.InferExpression) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateFunction(plop *plan.CreateFunction) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropFunction(plop *plan.DropFunction) (any, error) {
	return nil, nil
}

func (this *collector) VisitExecuteFunction(plop *plan.ExecuteFunction) (any, error) {
	return nil, nil
}

func (this *collector) VisitIndexFtsSearch(plop *plan.IndexFtsSearch) (any, error) {
	return nil, nil
}

func (this *collector) VisitIndexAdvice(plop *plan.IndexAdvice) (any, error) {
	return nil, nil
}

func (this *collector) VisitAdvise(plop *plan.Advise) (any, error) {
	return nil, nil
}

func (this *collector) VisitUpdateStatistics(plop *plan.UpdateStatistics) (any, error) {
	return nil, nil
}

func (this *collector) VisitStartTransaction(plop *plan.StartTransaction) (any, error) {
	return nil, nil
}

func (this *collector) VisitCommitTransaction(plop *plan.CommitTransaction) (any, error) {
	return nil, nil
}

func (this *collector) VisitRollbackTransaction(plop *plan.RollbackTransaction) (any, error) {
	return nil, nil
}

func (this *collector) VisitTransactionIsolation(plop *plan.TransactionIsolation) (any, error) {
	return nil, nil
}

func (this *collector) VisitSavepoint(plop *plan.Savepoint) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateSequence(plop *plan.CreateSequence) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateKnowledge(plop *plan.CreateKnowledge) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropKnowledge(plop *plan.DropKnowledge) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropSequence(plop *plan.DropSequence) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterSequence(plop *plan.AlterSequence) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateCredentialStore(plop *plan.CreateCredentialStore) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterCredentialStore(plop *plan.AlterCredentialStore) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropCredentialStore(plop *plan.DropCredentialStore) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateBucket(plop *plan.CreateBucket) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterBucket(plop *plan.AlterBucket) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropBucket(plop *plan.DropBucket) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateCatalog(plop *plan.CreateCatalog) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterCatalog(plop *plan.AlterCatalog) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropCatalog(plop *plan.DropCatalog) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterCollection(plop *plan.AlterCollection) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateUser(plop *plan.CreateUser) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterUser(plop *plan.AlterUser) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropUser(plop *plan.DropUser) (any, error) {
	return nil, nil
}

func (this *collector) VisitCreateGroup(plop *plan.CreateGroup) (any, error) {
	return nil, nil
}

func (this *collector) VisitAlterGroup(plop *plan.AlterGroup) (any, error) {
	return nil, nil
}

func (this *collector) VisitDropGroup(plop *plan.DropGroup) (any, error) {
	return nil, nil
}
