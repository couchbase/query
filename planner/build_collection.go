//  Copyright 2020-Present Couchbase, Inc.
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

func getScope(credentials *auth.Credentials, parts ...string) (datastore.Scope, errors.Error) {
	if len(parts) != 4 {
		return nil, errors.NewDatastoreInvalidCollectionPartsError(parts...)
	}

	s, err := datastore.GetScope(parts[0:3]...)

	if err != nil {
		err1 := datastore.CheckBucketAccess(credentials, err, parts, nil)

		if err1 != nil {
			return s, err1
		}

	}

	return s, err
}

func (this *builder) VisitCreateCollection(stmt *algebra.CreateCollection) (interface{}, error) {
	scope, err := getScope(this.context.Credentials(), stmt.Keyspace().Path().Parts()...)
	if err != nil {
		return nil, err
	}

	return plan.NewQueryPlan(plan.NewCreateCollection(scope, stmt)), nil
}

func (this *builder) VisitDropCollection(stmt *algebra.DropCollection) (interface{}, error) {
	scope, err := getScope(this.context.Credentials(), stmt.Keyspace().Path().Parts()...)
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewDropCollection(scope, stmt)), nil
}

func (this *builder) VisitFlushCollection(stmt *algebra.FlushCollection) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewFlushCollection(keyspace, stmt)), nil
}
