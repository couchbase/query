//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/plan"
)

func BuildPrepared(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery, stream bool, context *PrepareContext) (*plan.Prepared, error) {
	qp, ik, err := Build(stmt, datastore, systemstore, namespace, subquery, stream, context)
	if err != nil {
		return nil, err
	}

	signature := stmt.Signature()
	var optimHints *algebra.OptimHints
	if stmt.OptimHints() != nil {
		optimHints = stmt.OptimHints().Copy()
	}
	return plan.NewPrepared(qp.PlanOp(), signature, ik, optimHints), nil
}
