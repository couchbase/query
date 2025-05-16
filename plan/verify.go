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
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

func verifyCovers(covers expression.Covers, keyspace datastore.Keyspace) datastore.Keyspace {
	if covers != nil {
		return keyspace
	}
	return nil
}

func verifyIndex(index datastore.Index, indexer datastore.Indexer, keyspace datastore.Keyspace, prepared *Prepared) errors.Error {
	if indexer == nil {
		if keyspace != nil {
			return errors.NewPlanVerificationError(fmt.Sprintf("failed to load the indexer for the keyspace: %s", keyspace.Name()), nil)
		} else {
			return errors.NewPlanVerificationError("Failed to load the indexer for the keyspace", nil)
		}
	}

	indexer.Refresh()

	state, _, _ := index.State()
	if state != datastore.ONLINE {
		return errors.NewPlanVerificationError(fmt.Sprintf("Index: %s is not online", index.Name()), nil)
	}

	// Checking state is not enough on its own: if the index no longer exists, because we have a
	// stale reference...
	idx, err := indexer.IndexById(index.Id())
	if idx == nil || err != nil {
		return errors.NewPlanVerificationError(fmt.Sprintf("Index: %s does not exist", index.Id()), err)
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		err := prepared.addIndexer(indexer)
		if err == nil && keyspace != nil {
			_, err = verifyKeyspace(keyspace, prepared)
		}
		return err
	}
	return nil
}

func verifyKeyspace(keyspace datastore.Keyspace, prepared *Prepared) (datastore.Keyspace, errors.Error) {
	if keyspace == nil {
		return keyspace, nil
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

	if ks == nil || err != nil {
		return keyspace, errors.NewPlanVerificationError(fmt.Sprintf("Keyspace: %s not found", keyspace.Id()), err)
	}

	if ks.Uid() != keyspace.Uid() {
		return keyspace, errors.NewPlanVerificationError(fmt.Sprintf("Keyspace: %s uuid has changed from %s to %s", keyspace.Id(), keyspace.Uid(), ks.Uid()), nil)
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addKeyspaceMetadata(meta)
	}

	// return newly found keyspace just in case it has been refreshed
	return ks, nil
}

func verifyScope(scope datastore.Scope, prepared *Prepared) (datastore.Scope, errors.Error) {
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
		return scope, errors.NewPlanVerificationError(fmt.Sprintf("Scope: %s not found", scope.Id()), err)
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addKeyspaceMetadata(meta)
	}

	// return newly found keyspace just in case it has been refreshed
	return scp, nil
}

func verifyBucket(bucket datastore.Bucket, prepared *Prepared) (datastore.Bucket, errors.Error) {
	var bkt datastore.Bucket
	var err errors.Error
	var meta datastore.KeyspaceMetadata

	namespace := bucket.Namespace()
	bkt, err = namespace.BucketById(bucket.Id())
	meta = namespace.(datastore.KeyspaceMetadata)

	if bkt == nil || err != nil {
		return bucket, errors.NewPlanVerificationError(fmt.Sprintf("Bucket: %s not found", bucket.Name()), err)
	}

	if bkt.Uid() != bucket.Uid() {
		return bucket, errors.NewPlanVerificationError(fmt.Sprintf("Bucket: %s uuid has changed from %s to %s", bucket.Name(), bucket.Uid(), bkt.Uid()), nil)
	}

	// amend prepared statement version so that next time we avoid checks
	if prepared != nil {
		prepared.addKeyspaceMetadata(meta)
	}

	// return newly found bucket just in case it has been refreshed
	return bkt, nil
}

/*
Not currently used

func verifyKeyspaceName(keyspace string, prepared *Prepared) bool {
	return true
}
*/
