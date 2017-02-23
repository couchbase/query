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
	"fmt"
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type keyspaceKeyspace struct {
	namespace *namespace
	name      string
	indexer   datastore.Indexer
}

func (b *keyspaceKeyspace) Release() {
}

func (b *keyspaceKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspaceKeyspace) Id() string {
	return b.Name()
}

func (b *keyspaceKeyspace) Name() string {
	return b.name
}

func (b *keyspaceKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	count := int64(0)
	namespaceIds, excp := b.namespace.store.actualStore.NamespaceIds()
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.namespace.store.actualStore.NamespaceById(namespaceId)
			if excp == nil {
				keyspaceIds, excp := namespace.KeyspaceIds()
				if excp == nil {
					for _, keyspaceId := range keyspaceIds {
						if !canRead(context, namespaceId, keyspaceId) {
							continue
						}
						// The list of keyspace ids can include memcached buckets.
						// We do not want to include them in the count of
						// of queryable buckets. Attempting to retrieve the keyspace
						// record of a memcached bucket returns an error,
						// which allows us to distinguish these buckets, and exclude them.
						// See MB-19364 for more info.
						_, err := namespace.KeyspaceByName(keyspaceId)
						if err == nil {
							count++
						}
					}
				} else {
					return 0, errors.NewSystemDatastoreError(excp, "")
				}
			} else {
				return 0, errors.NewSystemDatastoreError(excp, "")
			}
		}
		return count, nil
	}
	return 0, errors.NewSystemDatastoreError(excp, "")
}

func (b *keyspaceKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *keyspaceKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *keyspaceKeyspace) Fetch(keys []string, context datastore.QueryContext) ([]value.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]value.AnnotatedPair, 0, len(keys))
	for _, k := range keys {
		ns, ks := splitId(k)
		if !canRead(context, ns, ks) {
			continue
		}
		item, e := b.fetchOne(ns, ks)

		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.SetAttachment("meta", map[string]interface{}{
				"id": k,
			})
		}

		rv = append(rv, value.AnnotatedPair{
			Name:  k,
			Value: item,
		})
	}

	return rv, errs
}

func (b *keyspaceKeyspace) fetchOne(ns string, ks string) (value.AnnotatedValue, errors.Error) {
	namespace, err := b.namespace.store.actualStore.NamespaceById(ns)
	if namespace != nil {
		keyspace, err := namespace.KeyspaceById(ks)
		if keyspace != nil {
			doc := value.NewAnnotatedValue(map[string]interface{}{
				"id":           keyspace.Id(),
				"name":         keyspace.Name(),
				"namespace_id": namespace.Id(),
				"datastore_id": b.namespace.store.actualStore.Id(),
			})
			return doc, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

func (b *keyspaceKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newKeyspacesKeyspace(p *namespace) (*keyspaceKeyspace, errors.Error) {
	b := new(keyspaceKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_KEYSPACES

	primary := &keyspaceIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)

	return b, nil
}

type keyspaceIndex struct {
	name     string
	keyspace *keyspaceKeyspace
}

func (pi *keyspaceIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *keyspaceIndex) Id() string {
	return pi.Name()
}

func (pi *keyspaceIndex) Name() string {
	return pi.name
}

func (pi *keyspaceIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *keyspaceIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *keyspaceIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *keyspaceIndex) Condition() expression.Expression {
	return nil
}

func (pi *keyspaceIndex) IsPrimary() bool {
	return true
}

func (pi *keyspaceIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *keyspaceIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *keyspaceIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func makeId(n, k string) string {
	return fmt.Sprintf("%s/%s", n, k)
}

func splitId(id string) (string, string) {
	ids := strings.SplitN(id, "/", 2)
	return ids[0], ids[1]
}

func (pi *keyspaceIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer close(conn.EntryChannel())

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}

		var numProduced int64 = 0
		namespaceIds, err := pi.keyspace.namespace.store.actualStore.NamespaceIds()
		if err == nil {

		loop:
			for _, namespaceId := range namespaceIds {
				namespace, err := pi.keyspace.namespace.store.actualStore.NamespaceById(namespaceId)
				if err == nil {
					keyspaceIds, err := namespace.KeyspaceIds()
					if err == nil {
						for _, keyspaceId := range keyspaceIds {
							// The list of keyspace ids can include memcached buckets.
							// We do not want to include them in the list
							// of queryable buckets. Attempting to retrieve the keyspace
							// record of a memcached bucket returns an error,
							// which allows us to distinguish these buckets, and exclude them.
							// See MB-19364 for more info.
							_, err := namespace.KeyspaceByName(keyspaceId)
							if err != nil {
								continue
							}

							id := makeId(namespaceId, keyspaceId)
							if spanEvaluator.evaluate(id) {
								entry := datastore.IndexEntry{PrimaryKey: id}
								conn.EntryChannel() <- &entry
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
	}
}

func (pi *keyspaceIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	var numProduced int64 = 0
	namespaceIds, err := pi.keyspace.namespace.store.actualStore.NamespaceIds()
	if err == nil {

	loop:
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.namespace.store.actualStore.NamespaceById(namespaceId)
			if err == nil {
				keyspaceIds, err := namespace.KeyspaceIds()
				if err == nil {
					for _, keyspaceId := range keyspaceIds {
						// The list of keyspace ids can include memcached buckets.
						// We do not want to include them in the list
						// of queryable buckets. Attempting to retrieve the keyspace
						// record of a memcached bucket returns an error,
						// which allows us to distinguish these buckets, and exclude them.
						// See MB-19364 for more info.
						_, err := namespace.KeyspaceByName(keyspaceId)
						if err != nil {
							continue
						}
						id := makeId(namespaceId, keyspaceId)
						entry := datastore.IndexEntry{PrimaryKey: id}
						conn.EntryChannel() <- &entry
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
