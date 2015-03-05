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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
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
									excp = indexer.Refresh()
									if excp != nil {
										return 0, errors.NewSystemDatastoreError(excp, "")
									}

									indexIds, excp := indexer.IndexIds()
									if excp == nil {
										count += int64(len(indexIds))
									} else {
										return 0, errors.NewSystemDatastoreError(excp, "")
									}
								}
							} else {
								return 0, errors.NewSystemDatastoreError(excp, "")
							}
						} else {
							return 0, errors.NewSystemDatastoreError(excp, "")
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

func (b *indexKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *indexKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *indexKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, errors.Error) {
	rv := make([]datastore.AnnotatedPair, 0, len(keys)*2)

	for _, key := range keys {
		pairs, err := b.fetchOne(key)
		if err != nil {
			return nil, err
		}

		rv = append(rv, pairs...)
	}

	return rv, nil
}

func (b *indexKeyspace) fetchOne(key string) ([]datastore.AnnotatedPair, errors.Error) {
	rv := make([]datastore.AnnotatedPair, 0, 2)
	ids := strings.SplitN(key, "/", 3)

	actualStore := b.namespace.store.actualStore
	namespace, err := actualStore.NamespaceById(ids[0])
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceById(ids[1])
	if err != nil {
		return nil, err
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		index, err := indexer.IndexById(ids[2])
		if err != nil {
			continue
		}

		state, msg, err := index.State()
		if err != nil {
			return nil, err
		}

		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":           index.Id(),
			"name":         index.Name(),
			"keyspace_id":  keyspace.Id(),
			"namespace_id": namespace.Id(),
			"datastore_id": actualStore.Id(),
			"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index.RangeKey())),
			"using":        datastoreObjectToJSONSafe(index.Type()),
			"state":        string(state),
		})

		doc.SetAttachment("meta", map[string]interface{}{
			"id": key,
		})

		if msg != "" {
			doc.SetField("message", msg)
		}

		rv = append(rv, datastore.AnnotatedPair{key, doc})
	}

	return rv, nil
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
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *indexKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "Not yet implemented.")
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

func (pi *indexIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *indexIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *indexIndex) Drop() errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *indexIndex) Scan(span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.ScanEntries(limit, cons, vector, conn)
}

func (pi *indexIndex) ScanEntries(limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	// eliminate duplicate keys
	keys := make(map[string]string, 64)

	actualStore := pi.keyspace.namespace.store.actualStore
	namespaceIds, err := actualStore.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := actualStore.NamespaceById(namespaceId)
			if err == nil {
				keyspaceIds, err := namespace.KeyspaceIds()
				if err == nil {
					for _, keyspaceId := range keyspaceIds {
						keyspace, err := namespace.KeyspaceById(keyspaceId)
						if err == nil {
							indexers, err := keyspace.Indexers()
							if err == nil {
								for _, indexer := range indexers {
									err = indexer.Refresh()
									if err != nil {
										conn.Error(errors.NewSystemDatastoreError(err, ""))
										return
									}

									indexIds, err := indexer.IndexIds()
									if err == nil {
										for _, indexId := range indexIds {
											key := fmt.Sprintf("%s/%s/%s", namespaceId, keyspaceId, indexId)
											keys[key] = key
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

	for k, _ := range keys {
		entry := datastore.IndexEntry{PrimaryKey: k}
		conn.EntryChannel() <- &entry
	}
}
