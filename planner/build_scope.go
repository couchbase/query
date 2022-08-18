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
	"github.com/couchbase/query/plan"
)

func getBucket(credentials *auth.Credentials, parts ...string) (datastore.Bucket, error) {

	bucket, err := datastore.GetBucket(parts...)

	if err != nil {
		err1 := datastore.CheckBucketAccess(credentials, err, parts, nil)

		if err1 != nil {
			return bucket, err1
		}
	}

	return bucket, err
}

func (this *builder) VisitCreateScope(stmt *algebra.CreateScope) (interface{}, error) {
	bucket, err := getBucket(this.context.dsContext.Credentials(), stmt.Scope().Path().Namespace(), stmt.Scope().Path().Bucket())
	if err != nil {
		return nil, err
	}

	return plan.NewQueryPlan(plan.NewCreateScope(bucket, stmt)), nil
}

func (this *builder) VisitDropScope(stmt *algebra.DropScope) (interface{}, error) {
	bucket, err := getBucket(this.context.dsContext.Credentials(), stmt.Scope().Path().Namespace(), stmt.Scope().Path().Bucket())
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewDropScope(bucket, stmt)), nil
}
