//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/aus"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type ausSettingsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

// to create an instance of the system:aus_settings keyspace
func newAusSettingsKeyspace(p *namespace) (*ausSettingsKeyspace, errors.Error) {
	b := new(ausSettingsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_AUS_SETTINGS)

	// create a virtual primary index
	primary := &ausSettingsIndex{name: "#primary", keyspace: b}

	// create the indexer for the system keyspace
	b.indexer = newSystemIndexer(b, primary)

	// set the backpointer to the indexer for the primary index
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil

}

func (b *ausSettingsKeyspace) Id() string {
	return b.name
}

func (b *ausSettingsKeyspace) Name() string {
	return b.name
}

func (b *ausSettingsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *ausSettingsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int64
	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())

	for _, b := range buckets {
		err := aus.ScanAusSettings(b, func(path string) error {
			count++
			return nil
		})

		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (b *ausSettingsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *ausSettingsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *ausSettingsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *ausSettingsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context datastore.QueryContext,
	subPath []string, projection []string, useSubDoc bool) errors.Errors {

	var errs errors.Errors

	for _, k := range keys {

		item, errs := b.fetchOne(k)
		if len(errs) > 0 {
			return errs
		} else if item == nil {
			continue
		}

		item.SetMetaField(value.META_ID, k)
		item.SetMetaField(value.META_KEYSPACE, b.fullName)
		keysMap[k] = item
	}
	return errs
}

func (b *ausSettingsKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Errors) {
	item, errs := aus.FetchAusSettings(key)
	if len(errs) > 0 {
		return nil, errs
	}

	if item == nil {
		return nil, nil
	}

	return value.NewAnnotatedValue(item), nil
}

func (b *ausSettingsKeyspace) Insert(inserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {
	return mutationHelper(aus.MOP_INSERT, inserts, context, preserveMutations)
}

func (b *ausSettingsKeyspace) Upsert(upserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {
	return mutationHelper(aus.MOP_UPSERT, upserts, context, preserveMutations)
}

func (b *ausSettingsKeyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {
	return mutationHelper(aus.MOP_UPDATE, updates, context, preserveMutations)
}

func (b *ausSettingsKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {
	return mutationHelper(aus.MOP_DELETE, deletes, context, preserveMutations)
}

func (b *ausSettingsKeyspace) Release(close bool) {
}

type ausSettingsIndex struct {
	indexBase
	name     string
	keyspace *ausSettingsKeyspace
}

func (asi *ausSettingsIndex) KeyspaceId() string {
	return asi.keyspace.Id()
}

func (asi *ausSettingsIndex) Id() string {
	return asi.name
}

func (asi *ausSettingsIndex) Name() string {
	return asi.name
}

func (asi *ausSettingsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (asi *ausSettingsIndex) Indexer() datastore.Indexer {
	return asi.indexer
}

func (asi *ausSettingsIndex) SeekKey() expression.Expressions {
	return nil
}

func (asi *ausSettingsIndex) RangeKey() expression.Expressions {
	return nil
}

func (asi *ausSettingsIndex) Condition() expression.Expression {
	return nil
}

func (asi *ausSettingsIndex) IsPrimary() bool {
	return true
}

func (asi *ausSettingsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (asi *ausSettingsIndex) Statistics(requestId string, span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (asi *ausSettingsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (asi *ausSettingsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	asi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (asi *ausSettingsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	defer conn.Sender().Close()
	context := conn.QueryContext()
	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())

	for _, b := range buckets {
		err := aus.ScanAusSettings(b, func(path string) error {
			entry := datastore.IndexEntry{PrimaryKey: path}
			sendSystemKey(conn, &entry)
			return nil
		})

		if err != nil {
			conn.Error(err)
			return
		}
	}
}

func mutationHelper(op aus.MutateOp, pairs value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {
	var mutationCount int
	var mutatedPairs value.Pairs
	var errs errors.Errors

	if preserveMutations {
		mutatedPairs = make(value.Pairs, 0, len(pairs))
	}

	for _, v := range pairs {
		mCount, mPairs, mErrs := aus.MutateAusSettings(op, v, context, preserveMutations)
		mutationCount += mCount
		if preserveMutations {
			mutatedPairs = append(mutatedPairs, mPairs...)
		}

		if len(mErrs) > 0 {
			errs = append(errs, mErrs...)
			break
		}
	}

	return mutationCount, mutatedPairs, errs
}
