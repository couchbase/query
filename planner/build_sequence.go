//  Copyright 2023-Present Couchbase, Inc.
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

func validateSequencePath(credentials *auth.Credentials, path *algebra.Path) errors.Error {

	if path.Scope() == "" {
		return nil
	}
	parts := path.ScopePath().Parts()
	if len(parts) != 3 {
		return errors.NewDatastoreInvalidScopePartsError(parts...)
	}
	_, err := datastore.GetScope(parts[0:3]...)
	if err != nil {
		err1 := datastore.CheckBucketAccess(credentials, err, parts)
		if err1 != nil {
			err = err1
		}
	}
	return err
}

func (this *builder) VisitCreateSequence(stmt *algebra.CreateSequence) (interface{}, error) {
	err := validateSequencePath(this.context.Credentials(), stmt.Name())
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewCreateSequence(stmt)), nil
}

func (this *builder) VisitDropSequence(stmt *algebra.DropSequence) (interface{}, error) {
	err := validateSequencePath(this.context.Credentials(), stmt.Name())
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewDropSequence(stmt)), nil
}

func (this *builder) VisitAlterSequence(stmt *algebra.AlterSequence) (interface{}, error) {
	err := validateSequencePath(this.context.Credentials(), stmt.Name())
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewAlterSequence(stmt)), nil
}
