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

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type indexKeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
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
	namespaceIds, excp := b.namespace.datastore.actualDatastore.NamespaceIds()
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.namespace.datastore.actualDatastore.NamespaceById(namespaceId)
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

func (b *indexKeyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *indexKeyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *indexKeyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *indexKeyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *indexKeyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *indexKeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func (b *indexKeyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
	rv := make([]catalog.Pair, len(keys))
	for i, k := range keys {
		item, e := b.FetchOne(k)
		if e != nil {
			return nil, e
		}

		rv[i].Key = k
		rv[i].Value = item
	}
	return rv, nil
}

func (b *indexKeyspace) FetchOne(key string) (value.Value, errors.Error) {
	ids := strings.SplitN(key, "/", 3)

	namespace, err := b.namespace.datastore.actualDatastore.NamespaceById(ids[0])
	if namespace != nil {
		keyspace, _ := namespace.KeyspaceById(ids[1])
		if keyspace != nil {
			index, _ := keyspace.IndexById(ids[2])
			if index != nil {
				doc := value.NewValue(map[string]interface{}{
					"id":           index.Id(),
					"name":         index.Name(),
					"keyspace_id":  keyspace.Id(),
					"namespace_id": namespace.Id(),
					"datastore_id": b.namespace.datastore.actualDatastore.Id(),
					"index_key":    catalogObjectToJSONSafe(indexKeyToIndexKeyStringArray(index.EqualKey())),
					"index_type":   catalogObjectToJSONSafe(index.Type()),
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

func catalogObjectToJSONSafe(catobj interface{}) interface{} {
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

func (b *indexKeyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
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

func (pi *indexIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *indexIndex) Drop() errors.Error {
	return errors.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *indexIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) Condition() expression.Expression {
	return nil
}

func (pi *indexIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *indexIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	namespaceIds, err := pi.keyspace.namespace.datastore.actualDatastore.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.namespace.datastore.actualDatastore.NamespaceById(namespaceId)
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

									entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", namespaceId, keyspaceId, indexId)}
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

func (pi *indexIndex) Scan(span catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	val := ""

	a := span.Equal[0].Actual()
	switch a := a.(type) {
	case string:
		val = a
	default:
		conn.Error(errors.NewError(nil, fmt.Sprintf("Invalid equality value %v of type %T.", a, a)))
		return
	}

	ids := strings.SplitN(val, "/", 3)
	if len(ids) != 3 {
		return
	}

	namespace, _ := pi.keyspace.namespace.datastore.actualDatastore.NamespaceById(ids[0])
	if namespace == nil {
		return
	}

	keyspace, _ := namespace.KeyspaceById(ids[1])
	if keyspace == nil {
		return
	}

	index, _ := keyspace.IndexById(ids[2])
	if keyspace != nil {
		entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s/%s", namespace.Id(), keyspace.Id(), index.Id())}
		conn.EntryChannel() <- &entry
	}
}
