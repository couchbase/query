//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// verify implements all the utility functions for autoreprepare

package plan

import (
	"github.com/couchbase/query/datastore"
)

func verifyIndex(index datastore.Index, indexer datastore.Indexer, prepared *Prepared) bool {
	if indexer == nil {
		return false
	}

	indexer.Refresh()

	state, _, _ := index.State()
	if state != datastore.ONLINE {
		return false
	}

	// Checking state is not enough on its own: if the index no longer exists, because we have a
	// stale reference...
	idx, err := indexer.IndexById(index.Id())
	if idx == nil || err != nil {
		return false
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addIndexer(indexer)
	}
	return true
}

func verifyKeyspace(keyspace datastore.Keyspace, prepared *Prepared) (datastore.Keyspace, bool) {
	namespace := keyspace.Namespace()
	ks, err := namespace.KeyspaceById(keyspace.Id())

	if ks == nil || err != nil {
		return keyspace, false
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addNamespace(namespace)
	}

	// return newly found keyspace just in case it has been refreshed
	return ks, true
}

/*
Not currently used

func verifyKeyspaceName(keyspace string, prepared *Prepared) bool {
	return true
}
*/
