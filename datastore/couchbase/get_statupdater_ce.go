// Copyright (c) 2018 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License. You
// may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

// +build !enterprise

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
	exContext interface{}, internal bool) {
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
