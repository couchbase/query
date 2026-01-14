//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/settings"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

const _SETTINGS_DOC_KEY = "global_settings"

type settingsKeyspace struct {
	keyspaceBase
	di datastore.Indexer
}

// to create an instance of the system:settings keyspace
func newSettingsKeyspace(p *namespace) (*settingsKeyspace, errors.Error) {
	b := new(settingsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_SETTINGS)

	// create a virtual primary index
	primary := &settingsIndex{name: PRIMARY_INDEX_NAME, keyspace: b}

	// create the indexer for the system keyspace
	b.di = newSystemIndexer(b, primary)

	// set the backpointer to the indexer for the primary index
	setIndexBase(&primary.indexBase, b.di)

	return b, nil

}

func (b *settingsKeyspace) Id() string {
	return b.Name()
}

func (b *settingsKeyspace) Name() string {
	return b.name
}

func (b *settingsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *settingsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return 1, nil
}

func (b *settingsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *settingsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.di, nil
}

func (b *settingsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.di}, nil
}

func (b *settingsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPath []string,
	projection []string, useSubDoc bool) errors.Errors {

	for _, k := range keys {
		if k != _SETTINGS_DOC_KEY {
			continue
		}

		val, err := settings.FetchSettings()
		if err != nil {
			return errors.Errors{err}
		}

		// If the key does not exist in metakv
		if val == nil {
			continue
		}

		av := value.NewAnnotatedValue(val)
		av.SetId(k)
		keysMap[k] = av
	}

	return nil
}

// Note - INSERT, UPSERT and DELETE will not be supported on system:settings

func (b *settingsKeyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var mPairs value.Pairs
	var mCount int
	var errs errors.Errors
	if preserveMutations {
		mPairs = make(value.Pairs, 0, len(updates))
	}

	for _, pair := range updates {
		if pair.Name != _SETTINGS_DOC_KEY {
			continue
		}

		err, warnings := settings.UpdateSettings(b.namespace.store.enterprise, context.RequestId(), pair.Value)
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

	if mCount > 0 {
		err := settings.PersistSettings()
		if err != nil && len(errs) == 0 {
			errs = append(errs, err)
		}
	}

	return mCount, mPairs, errs
}

func (b *settingsKeyspace) Release(close bool) {
}

type settingsIndex struct {
	indexBase
	name     string
	keyspace *settingsKeyspace
}

func (si *settingsIndex) KeyspaceId() string {
	return si.keyspace.Id()
}

func (si *settingsIndex) Id() string {
	return si.Name()
}

func (si *settingsIndex) Name() string {
	return si.name
}

func (si *settingsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (si *settingsIndex) Indexer() datastore.Indexer {
	return si.indexer
}

func (si *settingsIndex) SeekKey() expression.Expressions {
	return nil
}

func (si *settingsIndex) RangeKey() expression.Expressions {
	return nil
}

func (si *settingsIndex) Condition() expression.Expression {
	return nil
}

func (si *settingsIndex) IsPrimary() bool {
	return true
}

func (si *settingsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (si *settingsIndex) Statistics(requestId string, span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (si *settingsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "For system:"+KEYSPACE_NAME_SETTINGS)
}

func (si *settingsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	si.ScanEntries(requestId, limit, cons, vector, conn)
}

func (si *settingsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
	entry := datastore.IndexEntry{PrimaryKey: _SETTINGS_DOC_KEY}
	sendSystemKey(conn, &entry)
}
