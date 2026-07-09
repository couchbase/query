//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

// IndexInfo looks up a single index by bucket/scope/collection/name directly via the datastore
// (namespace -> bucket -> scope -> collection -> indexer -> index), rather than scanning
// system:indexes. It returns a nil value (no error) when any part of the path, or the index
// itself, cannot be found, or when the caller isn't authorized to see it -- matching the
// behavior of querying system:indexes, where unauthorized/non-existent rows are both just absent.
func (this *Context) IndexInfo(bucket, scope, collection, indexName string) (value.Value, error) {
	ds := this.datastore
	if ds == nil {
		return nil, nil
	}

	if !this.authorizedForIndexInfo(bucket, scope, collection) {
		return nil, nil
	}

	// mirrors the not-found handling in datastore/system/system_keyspace_indexes.go: error text
	// varies by backend and level (e.g. the couchbase datastore reports both a missing bucket and
	// a missing collection as "Keyspace not found ..."), so match generically rather than trying
	// to key off a specific object name or error code.
	namespace, err := ds.NamespaceById(this.namespace)
	if err != nil {
		if errors.IsNotFoundError("", err) {
			return nil, nil
		}
		return nil, err
	}
	b, err := namespace.BucketById(bucket)
	if err != nil {
		if errors.IsNotFoundError("", err) {
			return nil, nil
		}
		return nil, err
	}
	sc, err := b.ScopeById(scope)
	if err != nil {
		if errors.IsNotFoundError("", err) {
			return nil, nil
		}
		return nil, err
	}
	ks, err := sc.KeyspaceById(collection)
	if err != nil {
		if errors.IsNotFoundError("", err) {
			return nil, nil
		}
		return nil, err
	}
	if ks.IsExternalCollection() {
		return nil, nil
	}

	indexers, err := ks.Indexers()
	if err != nil {
		return nil, nil
	}
	docs := make([]interface{}, 0, 2)
	for _, indexer := range indexers {
		err = indexer.Refresh()
		if err != nil {
			return nil, err
		}

		index, err := indexer.IndexByName(indexName)
		if err != nil || index == nil {
			continue
		}

		keyspaceId, scopeId, bucketId := ks.Id(), sc.Id(), b.Id()
		if scope == "_default" && collection == "_default" {
			// mirrors the legacy flat-keyspace representation in system:indexes (bucket
			// referenced directly, without a scope/collection path): the default collection's
			// own id is just "_default" and isn't useful as a keyspace_id on its own.
			keyspaceId, scopeId, bucketId = b.Id(), "", ""
		}

		doc, err := datastore.IndexInfoDoc(index, keyspaceId, scopeId, bucketId, namespace.Name(), namespace.Id(), ds.URL())
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return value.NewValue(docs), nil
}

// authorizedForIndexInfo mirrors the privilege checks system:indexes applies per row (read
// access to the keyspace, plus list-index privilege) so that direct datastore access here
// doesn't bypass what the equivalent system:indexes query would have enforced.
func (this *Context) authorizedForIndexInfo(bucket, scope, collection string) bool {
	elems := []string{this.namespace, bucket, scope, collection}
	path := algebra.NewPathFromElements(elems).FullName()

	privs := auth.NewPrivileges()
	if systemStore, ok := this.datastore.(datastore.Systemstore); ok {
		systemStore.PrivilegesFromPath(path, bucket, auth.PRIV_QUERY_SELECT, privs)
		systemStore.PrivilegesFromPath(path, bucket, auth.PRIV_QUERY_LIST_INDEX, privs)
	} else {
		privs.Add(path, auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)
		privs.Add(path, auth.PRIV_QUERY_LIST_INDEX, auth.PRIV_PROPS_NONE)
	}

	// Authorize/AuthorizeInternal rely on each privilege having been precompiled by PreAuthorize
	// first (see plan.NewAuthorize) -- skipping this leaves PrivilegePair.Ready unset and
	// produces a malformed permission check against the cluster.
	this.datastore.PreAuthorize(privs)

	// AuthorizeInternal, like canRead/canListIndexes in system:indexes, avoids logging an audit
	// entry for what's effectively per-row filtering rather than a user-visible access denial.
	return this.datastore.AuthorizeInternal(privs, this.Credentials()) == nil
}
