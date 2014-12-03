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
	"github.com/couchbaselabs/query/value"
)

type indexKeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]datastore.Index
	primary   datastore.PrimaryIndex
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
							indexIds, excp := keyspace.IndexIds()
							if excp == nil {
								count += int64(len(indexIds))
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
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *indexKeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *indexKeyspace) IndexById(id string) (datastore.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *indexKeyspace) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *indexKeyspace) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *indexKeyspace) Indexes() ([]datastore.Index, errors.Error) {
	rv := make([]datastore.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *indexKeyspace) Authenticate(credentials datastore.Credentials, requested datastore.Privileges) errors.Error {
	return nil
}

func (b *indexKeyspace) CreatePrimaryIndex(using datastore.IndexType) (datastore.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *indexKeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression, using datastore.IndexType) (datastore.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
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
			index, _ := keyspace.IndexById(ids[2])
			if index != nil {
				doc := value.NewAnnotatedValue(map[string]interface{}{
					"id":           index.Id(),
					"name":         index.Name(),
					"keyspace_id":  keyspace.Id(),
					"namespace_id": namespace.Id(),
					"store_id":     b.namespace.store.actualStore.Id(),
					"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index.SeekKey())),
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
		// TODO: Determine if Expression needs to implement fmt.Stringer per ast.Expression in dp3
		// rv[i] = kp.String()
		rv[i] = fmt.Sprintf("%v", kp)
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

	b.primary = &indexIndex{name: "primary", keyspace: b}

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

func (b *indexKeyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
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
	return datastore.UNSPECIFIED
}

func (pi *indexIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) Condition() expression.Expression {
	return nil
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

func (pi *indexIndex) Scan(span *datastore.Span, distinct bool, limit int64, conn *datastore.IndexConnection) {
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

	index, _ := keyspace.IndexById(ids[2])
	if keyspace != nil {
		entry := datastore.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", namespace.Id(), keyspace.Id(), index.Id())}
		conn.EntryChannel() <- &entry
	}
}

func (pi *indexIndex) ScanEntries(limit int64, conn *datastore.IndexConnection) {
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
							indexIds, err := keyspace.IndexIds()
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
