//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"strings"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/util"
)

type IndexUpdStatParams struct {
	keyspace datastore.Keyspace
	idxNames []string
}

func newIndexUpdStatParams(keyspace datastore.Keyspace, idxNames []string) *IndexUpdStatParams {
	return &IndexUpdStatParams{
		keyspace: keyspace,
		idxNames: idxNames,
	}
}

func updateStats(names []string, subClass string, keyspace datastore.Keyspace, context *Context) errors.Error {
	allNames := strings.Join(names, ",")
	sessionName, err := util.UUIDV4()
	if err != nil {
		return errors.NewIndexUpdStatsError(allNames, "error getting sessionName", err)
	}

	params := newIndexUpdStatParams(keyspace, names)
	description := keyspace.QualifiedName() + "(" + allNames + ")"
	err = scheduler.ScheduleTask(sessionName, "update_statistics", subClass, time.Second,
		updateIndexStats, nil, params, description, context)
	if err != nil {
		return errors.NewIndexUpdStatsError(allNames, "error scheduling task", err)
	}
	return nil
}

const _MAX_ITERATION = 8

func updateIndexStats(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {
	idxUpdStatParams := parms.(*IndexUpdStatParams)
	if idxUpdStatParams == nil {
		return nil, nil
	}
	keyspace := idxUpdStatParams.keyspace
	idxNames := idxUpdStatParams.idxNames
	if keyspace == nil || len(idxNames) == 0 || algebra.IsSystemId(keyspace.NamespaceId()) {
		return nil, nil
	}

	indexer, err := keyspace.Indexer(datastore.GSI)
	if err != nil {
		return nil, []errors.Error{err}
	}

	var allNames string
	for i, name := range idxNames {
		if i > 0 {
			allNames += ","
		}
		allNames += "`" + name + "`"
		// wait for index to be online
		iteration := 0
		interval := time.Second
		for iteration < _MAX_ITERATION {
			err := indexer.Refresh()
			if err != nil {
				return nil, []errors.Error{err}
			}
			index, err := indexer.IndexByName(name)
			if err != nil {
				return nil, []errors.Error{err}
			}
			state, _, err := index.State()
			if err != nil {
				return nil, []errors.Error{err}
			}
			if state != datastore.ONLINE {
				time.Sleep(interval)
				interval *= 2
			} else {
				break
			}
		}
		// for indexes that goes beyond _MAX_ITERATION, still include the
		// index in the UPDATE STATISTICS command such that distributions
		// for index key expressions can be gathered (does not need index to be online)
		// index statistics will not be gathered by UPDATE STATISTICS if the index
		// is not online; when the index eventually comes online, a query attempting
		// to use that index will then gather index statistics.
	}

	var bucket datastore.Bucket
	var scope datastore.Scope
	var fullName string
	scope = keyspace.Scope()
	if scope != nil {
		bucket = scope.Bucket()
		if bucket != nil {
			fullName += "`" + bucket.Id() + "`."
		}
		fullName += "`" + scope.Id() + "`."
	}
	fullName += "`" + keyspace.Id() + "`"
	query := "UPDATE STATISTICS FOR " + fullName + " INDEX(" + allNames + ")"
	_, _, err1 := context.EvaluateStatement(query, nil, nil, false, true, false, "")
	if err1 != nil {
		return nil, []errors.Error{errors.NewIndexUpdStatsError(allNames, "error running Update Statistics statement", err1)}
	}
	return nil, nil
}
