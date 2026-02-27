//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

/*
 * With plan stability ad_hoc mode, determine whether a statement can skip being prepared
 */
func CanSkipPlanStabilityPrepare(stmt Statement) bool {
	switch stmt.(type) {
	case *InferKeyspace, *InferExpression, *Explain, *ExplainFunction, *Prepare, *Execute,
		*UpdateStatistics,
		*CreateIndex, *DropIndex, *BuildIndexes, *AlterIndex, *CreatePrimaryIndex,
		*CreateBucket, *DropBucket, *AlterBucket,
		*CreateScope, *DropScope,
		*CreateCollection, *DropCollection, *FlushCollection,
		*CreateUser, *DropUser, *AlterUser,
		*CreateGroup, *DropGroup, *AlterGroup,
		*GrantRole, *RevokeRole,
		*CreateFunction, *DropFunction, *ExecuteFunction,
		*StartTransaction, *CommitTransaction, *RollbackTransaction, *Savepoint, *TransactionIsolation,
		*CreateSequence, *DropSequence, *AlterSequence:
		return true
	}
	return false
}
