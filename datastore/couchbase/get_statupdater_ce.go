// Copyright 2018-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

//go:build !enterprise

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type NopStatUpdater struct {
}

func (dsu *NopStatUpdater) Name() datastore.StatUpdaterType {
	return ("UpdateStatisticsUnsuppored")
}

func (dsu *NopStatUpdater) UpdateStatistics(ks datastore.Keyspace, indexes []datastore.Index,
	terms expression.Expressions, with value.Value, conn *datastore.ValueConnection,
	exContext interface{}, internal bool, inAus bool) {
	conn.Error(errors.NewOtherNotImplementedError(nil, "UPDATE STATISTICS. This is an Enterprise only feature."))
	close(conn.ValueChannel())
}

func (dsu *NopStatUpdater) DeleteStatistics(ks datastore.Keyspace, terms expression.Expressions,
	conn *datastore.ValueConnection, exContext interface{}) {
	conn.Error(errors.NewOtherNotImplementedError(nil, "UPDATE STATISTICS. This is an Enterprise only feature."))
	close(conn.ValueChannel())
}

func GetDefaultStatUpdater(store datastore.Datastore) (datastore.StatUpdater, errors.Error) {
	statUpdater := new(NopStatUpdater)
	return statUpdater, nil
}
