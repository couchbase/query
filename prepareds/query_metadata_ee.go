//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package prepareds

import (
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

// initialize cache from persisted entries
func PreparedsFromPersisted() {
	hasQueryMetadata, _ := dictionary.HasQueryMetadata(false, "", false)
	if !hasQueryMetadata {
		return
	}

	var err errors.Error
	var queryMetadata datastore.Keyspace

	store := datastore.GetDatastore()
	if store == nil {
		err = errors.NewNoDatastoreError()
	} else {
		queryMetadata, err = store.GetQueryMetadata()
		if queryMetadata == nil {
			return
		}
		err = dictionary.ForeachPreparedPlan(true, processPreparedPlan)
	}

	if err != nil {
		// TODO: add failure report
		logging.Errorf("Error: %v", err)
	}
}

func processPreparedPlan(name, encoded_plan string) errors.Error {
	// TODO: add reporting
	_, err, _ := DecodePrepared(name, encoded_plan, true, logging.NULL_LOG)

	return err
}
