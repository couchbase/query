//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package datastore

import (
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

type Inferencer interface {
	Name() InferenceType
	// The Inferencer will return data over the connection
	InferKeyspace(context QueryContext, ks Keyspace, with value.Value, conn *ValueConnection)
	InferExpression(context QueryContext, expr expression.Expression, with value.Value, conn *ValueConnection)
}
