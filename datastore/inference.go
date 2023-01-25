//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type InferenceType string

const (
	INF_DEFAULT InferenceType = "default"
)

type RandomEntryProvider interface {
	GetRandomEntry(context QueryContext) (string, value.Value, errors.Error)
}

type RandomScanProvider interface {
	StartRandomScan(context QueryContext, sampleSize int, timeout time.Duration, pipelineSize int,
		kvTimeout time.Duration, serverless bool) (interface{}, errors.Error)
	StopKeyScan(scan interface{}) (uint64, errors.Error)
	FetchKeys(scan interface{}, timeout time.Duration) ([]string, errors.Error, bool)
}

type Inferencer interface {
	Name() InferenceType
	// The Inferencer will return data over the connection
	InferKeyspace(context QueryContext, ks Keyspace, with value.Value, conn *ValueConnection)
	InferExpression(context QueryContext, expr expression.Expression, with value.Value, conn *ValueConnection)
}
