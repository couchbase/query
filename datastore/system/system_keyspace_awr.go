//  Copyright 2024-Present Couchbase, Inc.
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
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

const _DOC_KEY = "global_settings"

type awrKeyspace struct {
	keyspaceBase
	di datastore.Indexer
}

func (b *awrKeyspace) Release(close bool) {
}

func (b *awrKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *awrKeyspace) Id() string {
	return b.Name()
}

func (b *awrKeyspace) Name() string {
	return b.name
}

func (b *awrKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return 1, nil
}

func (b *awrKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *awrKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.di, nil
}

func (b *awrKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.di}, nil
}

func (b *awrKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	for _, k := range keys {
		item, e := b.fetchOne(k)
		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.SetMetaField(value.META_KEYSPACE, b.fullName)
			item.SetId(k)
		}

		keysMap[k] = item
	}

	return
}

func (b *awrKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	v := value.NewValue(server.AwrCB.Config())
	return value.NewAnnotatedValue(v), nil
}

func (b *awrKeyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var errs []errors.Error
	var mutations int
	var preserved value.Pairs
	if !b.namespace.store.enterprise {
		errs = append(errs, errors.NewAwrNotSupportedError())
		return mutations, preserved, errs
	}

	for i := range updates {
		v := updates[i].Value.Actual()
		if m, ok := v.(map[string]interface{}); ok {

			err, warnings := server.AwrCB.SetConfig(m, true)

			if len(warnings) > 0 {
				for _, w := range warnings {
					context.Warning(w)
				}
			}

			if err != nil {
				errs = append(errs, err)
			} else {
				mutations++
				if preserveMutations {
					preserved = append(preserved, updates[i])
				}
			}
		}
	}

	return mutations, preserved, errs
}

func newAWRKeyspace(p *namespace) (*awrKeyspace, errors.Error) {
	b := new(awrKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_AWR)

	primary := &awrIndex{name: PRIMARY_INDEX_NAME, keyspace: b}
	b.di = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.di)

	return b, nil
}

type awrIndex struct {
	indexBase
	name     string
	keyspace *awrKeyspace
}

func (pi *awrIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *awrIndex) Id() string {
	return pi.Name()
}

func (pi *awrIndex) Name() string {
	return pi.name
}

func (pi *awrIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *awrIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *awrIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *awrIndex) Condition() expression.Expression {
	return nil
}

func (pi *awrIndex) IsPrimary() bool {
	return true
}

func (pi *awrIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *awrIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *awrIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "For system:"+KEYSPACE_NAME_AWR)
}

func (pi *awrIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	// no fields to compare - we just do a primary scan of one
	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (pi *awrIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	entry := datastore.IndexEntry{PrimaryKey: _DOC_KEY}
	sendSystemKey(conn, &entry)
}
