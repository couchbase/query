//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
)

func validateKnowledgePath(credentials *auth.Credentials, path *algebra.Path) errors.Error {

	parts := path.Parts()
	_, err := datastore.GetScope(parts[0:3]...)
	if err != nil {
		err1 := datastore.CheckBucketAccess(credentials, err, parts)
		if err1 != nil {
			err = err1
		}
	}
	return err
}

func (this *builder) VisitCreateKnowledge(stmt *algebra.CreateKnowledge) (interface{}, error) {
	err := validateKnowledgePath(this.context.Credentials(), stmt.Keyspace())
	if err != nil {
		return nil, err
	}

	// unlike PREPARE's opt_name (deliberately deterministic, so repeated PREPAREs of the same
	// text reuse the same cache slot), an unnamed CREATE KNOWLEDGE should mint a fresh entry on
	// every execution - so the implicit name is generated at execution time (execution/
	// knowledge_create.go), not here: this plan can itself be reused across many executions (a
	// prepared statement re-EXECUTEd, or auto-prepare), and baking a single generated name into
	// the shared algebra node would make every execution after the first collide on that same
	// name instead of minting a fresh one.
	return plan.NewQueryPlan(plan.NewCreateKnowledge(stmt)), nil
}

func (this *builder) VisitDropKnowledge(stmt *algebra.DropKnowledge) (interface{}, error) {
	err := validateKnowledgePath(this.context.Credentials(), stmt.Keyspace())
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewDropKnowledge(stmt)), nil
}
