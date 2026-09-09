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
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/memory"
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

	// MB-73749: A CREATE INDEX/BUILD INDEX statement triggers an UPDATE STATISTICS on the index
	// that is run asynchronously as a task in the scheduler. This statistics update can still be
	// running well after the CREATE INDEX/BUILD INDEX request has finished. When node-quota is enabled, a final release of
	// any memory tracked by the request's memory session is performed when the request completes
	// execution, returning that tracked memory back to the node-wide pool. However, if the
	// scheduled UPDATE STATISTICS statement kept tracking memory against that same,
	// already-released session afterwards, any memory it tracks against it can leak and never be
	// returned back to the node-wide pool, since no final release would ever be called again. To
	// avoid this, give the scheduled task its own Context with an independent memory session. Any
	// memory used by the asynchronous UPDATE STATISTICS execution is tracked against this new
	// session instead, and a final release is performed on it once the execution completes.
	taskCtx := context.Copy()
	if context.memorySession != nil {
		taskCtx.SetMemorySession(memory.Register())
	}
	err = scheduler.ScheduleTask(sessionName, "update_statistics", subClass, time.Second,
		updateIndexStats, nil, params, description, taskCtx)
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

	// MB-73749: Release any tracked memory back to the node-wide quota once the execution completes
	defer context.Release()
	_, _, err1 := context.EvaluateStatement(query, nil, nil, false, true, false, "")
	if err1 != nil {
		// error should already be logged during the scheduled UPDATE STATISTICS statement,
		// no need to repeat the same error, just log the fact that this is an automatic
		// UPDATE STATISTICS statement
		logging.Errorf("Error during automatic UPDATE STATISTICS from index CREATE/BUILD.")
		return nil, []errors.Error{errors.NewIndexUpdStatsError(allNames, "error running Update Statistics statement", err1)}
	}
	return nil, nil
}
