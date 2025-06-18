//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// verify implements all the utility functions for autoreprepare

package plan

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

func verifyCoversAndSeqScan(covers expression.Covers, keyspace datastore.Keyspace, indexer datastore.Indexer) datastore.Keyspace {
	if (indexer != nil && indexer.Name() == datastore.SEQ_SCAN) || covers != nil {
		return keyspace
	}
	return nil
}

func verifyIndex(index datastore.Index, indexer datastore.Indexer, keyspace datastore.Keyspace, prepared *Prepared) bool {
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
		if keyspace != nil {
			_, rv := verifyKeyspace(keyspace, prepared)
			return rv
		}
	}
	return true
}

func verifyKeyspace(keyspace datastore.Keyspace, prepared *Prepared) (datastore.Keyspace, bool) {
	if keyspace == nil {
		return keyspace, true
	}
	var ks datastore.Keyspace
	var err errors.Error
	var meta datastore.KeyspaceMetadata

	scope := keyspace.Scope()

	// for collections, go all the way up to the namespace and find your way back
	// for buckets, we only need to check the namespace
	if scope != nil {
		bucket := scope.Bucket()
		namespace := bucket.Namespace()
		d, _ := bucket.DefaultKeyspace()

		b, err := namespace.BucketById(bucket.Id())
		if err != nil {
			return keyspace, false
		}

		if b != nil && b.Uid() != bucket.Uid() {
			return keyspace, false
		}
		// if this is the default collection for a bucket, we're done
		if d != nil && d.Name() == keyspace.Name() && d.Id() == keyspace.Id() {
			ks = d
			namespace := keyspace.Namespace()
			meta = namespace.(datastore.KeyspaceMetadata)
		} else {
			b, _ := namespace.BucketById(bucket.Id())
			if b != nil {
				s, _ := b.ScopeById(scope.Id())
				if s != nil {
					ks, err = s.KeyspaceById(keyspace.Id())
					meta = b.(datastore.KeyspaceMetadata)
				}
			}
		}
	} else {
		namespace := keyspace.Namespace()
		ks, err = namespace.KeyspaceById(keyspace.Id())
		meta = namespace.(datastore.KeyspaceMetadata)
	}

	if ks == nil || err != nil || ks.Uid() != keyspace.Uid() {
		return keyspace, false
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addKeyspaceMetadata(meta)
	}

	// return newly found keyspace just in case it has been refreshed
	return ks, true
}

func verifyScope(scope datastore.Scope, prepared *Prepared) (datastore.Scope, bool) {
	var scp datastore.Scope
	var err errors.Error
	var meta datastore.KeyspaceMetadata

	bucket := scope.Bucket()
	namespace := bucket.Namespace()
	b, _ := namespace.BucketById(bucket.Id())
	if b != nil {
		scp, err = b.ScopeById(scope.Id())
		meta = b.(datastore.KeyspaceMetadata)
	}
	if scp == nil || err != nil {
		return scope, false
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addKeyspaceMetadata(meta)
	}

	// return newly found keyspace just in case it has been refreshed
	return scp, true
}

func verifyBucket(bucket datastore.Bucket, prepared *Prepared) (datastore.Bucket, bool) {
	var bkt datastore.Bucket
	var err errors.Error
	var meta datastore.KeyspaceMetadata

	namespace := bucket.Namespace()
	bkt, err = namespace.BucketById(bucket.Id())
	meta = namespace.(datastore.KeyspaceMetadata)

	if bkt == nil || err != nil || bkt.Uid() != bucket.Uid() {
		return bucket, false
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addKeyspaceMetadata(meta)
	}

	// return newly found bucket just in case it has been refreshed
	return bkt, true
}

/*
Not currently used

func verifyKeyspaceName(keyspace string, prepared *Prepared) bool {
	return true
}
*/
