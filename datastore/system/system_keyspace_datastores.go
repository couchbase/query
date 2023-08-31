//  Copyright 2013-Present Couchbase, Inc.
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
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type storeKeyspace struct {
	keyspaceBase
	si datastore.Indexer
}

func (b *storeKeyspace) Release(close bool) {
}

func (b *storeKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *storeKeyspace) Id() string {
	return b.Name()
}

func (b *storeKeyspace) Name() string {
	return b.name
}

func (b *storeKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return 1, nil
}

func (b *storeKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *storeKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *storeKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
}

func (b *storeKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string) (errs errors.Errors) {
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
			item.NewMeta()["keyspace"] = b.fullName
			item.SetId(k)
		}
		keysMap[k] = item
	}

	return
}

func (b *storeKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	if key == b.namespace.store.actualStore.Id() {
		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":  b.namespace.store.actualStore.Id(),
			"url": b.namespace.store.actualStore.URL(),
		})
		return doc, nil
	}
	return nil, errors.NewSystemDatastoreError(nil, "Key Not Found "+key)
}

func newStoresKeyspace(p *namespace) (*storeKeyspace, errors.Error) {
	b := new(storeKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_DATASTORES)

	primary := &storeIndex{name: "#primary", keyspace: b}
	b.si = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.si)

	return b, nil
}

type storeIndex struct {
	indexBase
	name     string
	keyspace *storeKeyspace
}

func (pi *storeIndex) KeyspaceId() string {
	return pi.name
}

func (pi *storeIndex) Id() string {
	return pi.Name()
}

func (pi *storeIndex) Name() string {
	return pi.name
}

func (pi *storeIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *storeIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *storeIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *storeIndex) Condition() expression.Expression {
	return nil
}

func (pi *storeIndex) IsPrimary() bool {
	return true
}

func (pi *storeIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *storeIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *storeIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *storeIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
		} else if spanEvaluator.evaluate(pi.keyspace.namespace.store.actualStore.Id()) {
			entry := datastore.IndexEntry{PrimaryKey: pi.keyspace.namespace.store.actualStore.Id()}
			sendSystemKey(conn, &entry)
		}
		conn.Sender().Close()
	}
}

func (pi *storeIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	entry := datastore.IndexEntry{PrimaryKey: pi.keyspace.namespace.store.actualStore.Id()}
	sendSystemKey(conn, &entry)
}
