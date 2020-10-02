//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type namespaceKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *namespaceKeyspace) Release(close bool) {
}

func (b *namespaceKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *namespaceKeyspace) Id() string {
	return b.Name()
}

func (b *namespaceKeyspace) Name() string {
	return b.name
}

func (b *namespaceKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	namespaceIds, excp := b.namespace.store.actualStore.NamespaceIds()
	if excp == nil {
		return int64(len(namespaceIds)), nil
	}
	return 0, errors.NewSystemDatastoreError(excp, "")
}

func (b *namespaceKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *namespaceKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *namespaceKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *namespaceKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
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

func (b *namespaceKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	namespace, excp := b.namespace.store.actualStore.NamespaceById(key)
	if namespace != nil {
		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":           namespace.Id(),
			"name":         namespace.Name(),
			"datastore_id": b.namespace.store.actualStore.Id(),
		})
		return doc, nil
	}
	return nil, errors.NewSystemDatastoreError(excp, "Key Not Found "+key)
}

func (b *namespaceKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *namespaceKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *namespaceKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *namespaceKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newNamespacesKeyspace(p *namespace) (*namespaceKeyspace, errors.Error) {
	b := new(namespaceKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_NAMESPACES)

	primary := &namespaceIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type namespaceIndex struct {
	indexBase
	name     string
	keyspace *namespaceKeyspace
}

func (pi *namespaceIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *namespaceIndex) Id() string {
	return pi.Name()
}

func (pi *namespaceIndex) Name() string {
	return pi.name
}

func (pi *namespaceIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *namespaceIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *namespaceIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *namespaceIndex) Condition() expression.Expression {
	return nil
}

func (pi *namespaceIndex) IsPrimary() bool {
	return true
}

func (pi *namespaceIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *namespaceIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *namespaceIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *namespaceIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil || len(span.Seek) == 0 {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer conn.Sender().Close()

		namespaceIds, err := pi.keyspace.namespace.store.actualStore.NamespaceIds()
		if err == nil {
			spanEvaluator, err := compileSpan(span)
			if err != nil {
				conn.Error(err)
			} else {
				var numProduced int64 = 0

			loop:
				for _, namespaceId := range namespaceIds {
					if spanEvaluator.evaluate(namespaceId) {
						entry := datastore.IndexEntry{PrimaryKey: namespaceId}
						if !sendSystemKey(conn, &entry) {
							return
						}
						numProduced++
						if limit > 0 && numProduced >= limit {
							break loop
						}
					}
				}
			}
		}
	}
}

func (pi *namespaceIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	namespaceIds, err := pi.keyspace.namespace.store.actualStore.NamespaceIds()
	if err == nil {
		for i, namespaceId := range namespaceIds {
			if limit > 0 && int64(i) > limit {
				break
			}

			entry := datastore.IndexEntry{PrimaryKey: namespaceId}
			if !sendSystemKey(conn, &entry) {
				return
			}
		}
	}
}
