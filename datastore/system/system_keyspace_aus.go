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

const _AUS_DOC_KEY = "global_settings"

type ausKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

// to create an instance of the system:aus keyspace
func newAusKeyspace(p *namespace) (*ausKeyspace, errors.Error) {
	b := new(ausKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_AUS)

	// create a virtual primary index
	primary := &ausIndex{name: PRIMARY_INDEX_NAME, keyspace: b}

	// create the indexer for the system keyspace
	b.indexer = newSystemIndexer(b, primary)

	// set the backpointer to the indexer for the primary index
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil

}

func (b *ausKeyspace) Id() string {
	return b.name
}

func (b *ausKeyspace) Name() string {
	return b.name
}

func (b *ausKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *ausKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return aus.CountAus()
}

func (b *ausKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *ausKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *ausKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *ausKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPath []string,
	projection []string, useSubDoc bool) errors.Errors {

	for _, k := range keys {
		if k != _AUS_DOC_KEY {
			continue
		}

		val, err := aus.FetchAus()
		if err != nil {
			return errors.Errors{err}
		}

		// If the key does not exist in metakv
		if val == nil {
			continue
		}

		av := value.NewAnnotatedValue(val)
		av.SetId(k)
		av.SetMetaField(value.META_KEYSPACE, b.fullName)
		keysMap[k] = av
	}

	return nil
}

// Note - INSERT, UPSERT and DELETE will not be supported on system:aus

func (b *ausKeyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var mPairs value.Pairs
	var mCount int
	var errs errors.Errors
	if preserveMutations {
		mPairs = make(value.Pairs, 0, len(updates))
	}

	for _, pair := range updates {
		if pair.Name != _AUS_DOC_KEY {
			continue
		}

		err, warnings := aus.SetAus(pair.Value, true)
		if warnings != nil {
			for _, w := range warnings {
				context.Warning(w)
			}
		}

		if err != nil {
			errs = append(errs, err)
			break
		}

		mCount++
		if preserveMutations {
			mPairs = append(mPairs, pair)
		}
	}

	return mCount, mPairs, errs
}

func (b *ausKeyspace) Release(close bool) {
}

type ausIndex struct {
	indexBase
	name     string
	keyspace *ausKeyspace
}

func (ai *ausIndex) KeyspaceId() string {
	return ai.keyspace.Id()
}

func (ai *ausIndex) Id() string {
	return ai.name
}

func (ai *ausIndex) Name() string {
	return ai.name
}

func (ai *ausIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (ai *ausIndex) Indexer() datastore.Indexer {
	return ai.indexer
}

func (ai *ausIndex) SeekKey() expression.Expressions {
	return nil
}

func (ai *ausIndex) RangeKey() expression.Expressions {
	return nil
}

func (ai *ausIndex) Condition() expression.Expression {
	return nil
}

func (ai *ausIndex) IsPrimary() bool {
	return true
}

func (ai *ausIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (ai *ausIndex) Statistics(requestId string, span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (ai *ausIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (ai *ausIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	ai.ScanEntries(requestId, limit, cons, vector, conn)
}

func (ai *ausIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
	entry := datastore.IndexEntry{PrimaryKey: _AUS_DOC_KEY}
	sendSystemKey(conn, &entry)
}
