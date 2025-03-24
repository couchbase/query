//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type StatUpdaterType string

const (
	UPDSTAT_DEFAULT StatUpdaterType = "default"
)

type StatUpdater interface {
	Name() StatUpdaterType
	UpdateStatistics(ks Keyspace, indexes []Index, terms expression.Expressions, with value.Value,
		conn *ValueConnection, exContext interface{}, internal bool, inAus bool) // The StatUpdater should populate the connection.
	DeleteStatistics(ks Keyspace, terms expression.Expressions, conn *ValueConnection, exContext interface{})
}
