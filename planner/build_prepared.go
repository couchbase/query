//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
)

func BuildPrepared(stmt algebra.Statement, store, systemstore datastore.Datastore,
	namespace string, subquery, stream, persist bool, context *PrepareContext) (*plan.Prepared, error, map[string]time.Duration) {
	qp, ik, err, subTimes := Build(stmt, store, systemstore, namespace, subquery, stream, false, context)
	if err != nil {
		return nil, err, subTimes
	}

	signature := stmt.Signature()
	var optimHints *algebra.OptimHints
	if stmt.OptimHints() != nil {
		optimHints = stmt.OptimHints().Copy()
	}
	prepared := plan.NewPrepared(qp.PlanOp(), signature, ik, optimHints, persist, false)

	if persist {
		// check and create (if not exists) QUERY_METADATA bucket
		hasMetadata, err := hasQueryMetadata(true, context.RequestId(), true)
		if err == nil && !hasMetadata {
			err = errors.NewMissingQueryMetadataError("SAVE option of PREPARE")
		}
		if err != nil {
			return nil, err, subTimes
		}
		// TODO: save plan
	}

	return prepared, nil, subTimes
}
