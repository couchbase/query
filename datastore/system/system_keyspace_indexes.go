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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/timestamp"
	"github.com/couchbaselabs/query/value"
)

type indexKeyspace struct {
	namespace *namespace
	name      string
	indexer   datastore.Indexer
}

func (b *indexKeyspace) Release() {
}

func (b *indexKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *indexKeyspace) Id() string {
	return b.Name()
}

func (b *indexKeyspace) Name() string {
	return b.name
}

func (b *indexKeyspace) Count() (int64, errors.Error) {
	count := int64(0)
	namespaceIds, excp := b.namespace.store.actualStore.NamespaceIds()
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.namespace.store.actualStore.NamespaceById(namespaceId)
			if excp == nil {
				keyspaceIds, excp := namespace.KeyspaceIds()
				if excp == nil {
					for _, keyspaceId := range keyspaceIds {
						keyspace, excp := namespace.KeyspaceById(keyspaceId)
						if excp == nil {
							indexers, excp := keyspace.Indexers()
							if excp == nil {
								for _, indexer := range indexers {
									indexIds, excp := indexer.IndexIds()
									if excp == nil {
										count += int64(len(indexIds))
									} else {
										return 0, errors.NewError(excp, "")
									}
								}
							} else {
								return 0, errors.NewError(excp, "")
							}
						} else {
							return 0, errors.NewError(excp, "")
						}
					}
				} else {
					return 0, errors.NewError(excp, "")
				}
			} else {
				return 0, errors.NewError(excp, "")
			}
		}
		return count, nil
	}
	return 0, errors.NewError(excp, "")
}

func (b *indexKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *indexKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *indexKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, errors.Error) {
	rv := make([]datastore.AnnotatedPair, len(keys))
	for i, k := range keys {
		item, e := b.fetchOne(k)
		if e != nil {
			return nil, e
		}

		rv[i].Key = k
		rv[i].Value = item
	}
	return rv, nil
}

func (b *indexKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	ids := strings.SplitN(key, "/", 3)

	namespace, err := b.namespace.store.actualStore.NamespaceById(ids[0])
	if namespace != nil {
		keyspace, _ := namespace.KeyspaceById(ids[1])
		if keyspace != nil {
			indexers, _ := keyspace.Indexers()
			index, _ := indexers[0].IndexById(ids[2])
			if index != nil {
				doc := value.NewAnnotatedValue(map[string]interface{}{
					"id":           index.Id(),
					"name":         index.Name(),
					"keyspace_id":  keyspace.Id(),
					"namespace_id": namespace.Id(),
					"datastore_id": b.namespace.store.actualStore.Id(),
					"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index.RangeKey())),
					"index_type":   datastoreObjectToJSONSafe(index.Type()),
				})
				return doc, nil
			}
		}
	}
	return nil, err
}

func indexKeyToIndexKeyStringArray(key expression.Expressions) []string {
	rv := make([]string, len(key))
	for i, kp := range key {
		rv[i] = expression.NewStringer().Visit(kp)
	}
	return rv
}

func datastoreObjectToJSONSafe(catobj interface{}) interface{} {
	var rv interface{}
	bytes, err := json.Marshal(catobj)
	if err == nil {
		json.Unmarshal(bytes, &rv)
	}
	return rv
}

func newIndexesKeyspace(p *namespace) (*indexKeyspace, errors.Error) {
	b := new(indexKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_INDEXES

	primary := &indexIndex{name: "#primary", keyspace: b}
	b.indexer = &systemIndexer{keyspace: b, indexes: make(map[string]datastore.Index), primary: primary}

	return b, nil
}

func (b *indexKeyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

type indexIndex struct {
	name     string
	keyspace *indexKeyspace
}

func (pi *indexIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *indexIndex) Id() string {
	return pi.Name()
}

func (pi *indexIndex) Name() string {
	return pi.name
}

func (pi *indexIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *indexIndex) SeekKey() expression.Expressions {
	return pi.SeekKey()
}

func (pi *indexIndex) RangeKey() expression.Expressions {
	return pi.RangeKey()
}

func (pi *indexIndex) Condition() expression.Expression {
	return pi.Condition()
}

func (pi *indexIndex) State() (datastore.IndexState, errors.Error) {
	return datastore.ONLINE, nil
}

func (pi *indexIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *indexIndex) Drop() errors.Error {
	return errors.NewError(nil, "This primary index cannot be dropped.")
}

func (pi *indexIndex) Scan(span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	val := ""

	a := span.Seek[0].Actual()
	switch a := a.(type) {
	case string:
		val = a
	default:
		conn.Error(errors.NewError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", a, a)))
		return
	}

	ids := strings.SplitN(val, "/", 3)
	if len(ids) != 3 {
		return
	}

	namespace, _ := pi.keyspace.namespace.store.actualStore.NamespaceById(ids[0])
	if namespace == nil {
		return
	}

	keyspace, _ := namespace.KeyspaceById(ids[1])
	if keyspace == nil {
		return
	}

	indexers, _ := keyspace.Indexers()
	index, _ := indexers[0].IndexById(ids[2])
	if keyspace != nil {
		entry := datastore.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", namespace.Id(), keyspace.Id(), index.Id())}
		conn.EntryChannel() <- &entry
	}
}

func (pi *indexIndex) ScanEntries(limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	namespaceIds, err := pi.keyspace.namespace.store.actualStore.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.namespace.store.actualStore.NamespaceById(namespaceId)
			if err == nil {
				keyspaceIds, err := namespace.KeyspaceIds()
				if err == nil {
					for _, keyspaceId := range keyspaceIds {
						keyspace, err := namespace.KeyspaceById(keyspaceId)
						if err == nil {
							indexers, err := keyspace.Indexers()
							if err == nil {
								for _, indexer := range indexers {
									indexIds, err := indexer.IndexIds()
									if err == nil {
										for i, indexId := range indexIds {
											if limit > 0 && int64(i) > limit {
												break
											}

											entry := datastore.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", namespaceId, keyspaceId, indexId)}
											conn.EntryChannel() <- &entry
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
