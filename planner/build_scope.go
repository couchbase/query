//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"sync"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

const _DDL_RETRY_LIMIT = 32
const _DDL_RETRY_WINDOW = 1 * time.Minute

const _DDL_MAX_RETRY = 4
const _DDL_RETRY_DELAY = 100 * time.Millisecond

type ddlRetryTracker struct {
	count     int
	lastError time.Time
	sync.Mutex
}

func (tracker *ddlRetryTracker) allowRetry() bool {
	tracker.Lock()
	defer tracker.Unlock()

	now := time.Now()
	if now.Sub(tracker.lastError) > _DDL_RETRY_WINDOW || tracker.count == 0 {
		tracker.count = 1
		tracker.lastError = now
		return true
	}

	if tracker.count >= _DDL_RETRY_LIMIT {
		if tracker.count == _DDL_RETRY_LIMIT {
			nextWindow := now.Add(_DDL_RETRY_WINDOW)
			logging.Warnf("DDL retry limit reached, no more retries allowed until: %s", nextWindow.Format(util.DEFAULT_FORMAT))
		}
		tracker.count++
		return false
	}

	tracker.count++
	tracker.lastError = now
	return true
}

var ddlTracker *ddlRetryTracker = &ddlRetryTracker{}

func getBucket(credentials *auth.Credentials, retry bool, parts ...string) (datastore.Bucket, error) {

	var bucket datastore.Bucket
	var err errors.Error
	retryInterval := _DDL_RETRY_DELAY
	var callerInfo string
	var bucketName string

	for i := 0; ; i++ {
		bucket, err = datastore.GetBucket(parts...)
		if err == nil || !retry || (err.Code() != errors.E_CB_KEYSPACE_NOT_FOUND) {
			break
		}
		if i == 0 {
			callerInfo = errors.CallerN(1)
			bucketName = parts[1]
		}
		if i >= _DDL_MAX_RETRY {
			logging.Infof("Failed to get bucket: %s - %s", bucketName, callerInfo)
			break
		}
		if !ddlTracker.allowRetry() {
			logging.Infof("Retry limit reached, failed get bucket: %s - %s", bucketName, callerInfo)
			break
		}

		logging.Infof("Retrying to get bucket: %s (remaining retries: %d) - %s",
			bucketName, _DDL_MAX_RETRY-i, callerInfo)
		time.Sleep(retryInterval)
		retryInterval *= 2
	}
	if err != nil {
		err1 := datastore.CheckBucketAccess(credentials, err, parts)

		if err1 != nil {
			return bucket, err1
		}
	}

	return bucket, err
}

func (this *builder) VisitCreateScope(stmt *algebra.CreateScope) (interface{}, error) {
	bucket, err := getBucket(this.context.Credentials(), true, stmt.Scope().Path().Namespace(),
		stmt.Scope().Path().Bucket())
	if err != nil {
		return nil, err
	}

	return plan.NewQueryPlan(plan.NewCreateScope(bucket, stmt)), nil
}

func (this *builder) VisitDropScope(stmt *algebra.DropScope) (interface{}, error) {
	bucket, err := getBucket(this.context.Credentials(), false, stmt.Scope().Path().Namespace(),
		stmt.Scope().Path().Bucket())
	if err != nil {
		return nil, err
	}
	return plan.NewQueryPlan(plan.NewDropScope(bucket, stmt)), nil
}
