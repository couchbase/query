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

func (b *keyspaceKeyspace) Count() (int64, errors.Error) {
	count := int64(0)
	namespaceIds, excp := b.namespace.store.actualStore.NamespaceIds()
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.namespace.store.actualStore.NamespaceById(namespaceId)
			if excp == nil {
				keyspaceIds, excp := namespace.KeyspaceIds()
				if excp == nil {
					count += int64(len(keyspaceIds))
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

func (b *keyspaceKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]datastore.AnnotatedPair, 0, len(keys))
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
			item.SetAttachment("meta", map[string]interface{}{
				"id": k,
			})
		}

		rv = append(rv, datastore.AnnotatedPair{
			Key:   k,
			Value: item,
		})
	}

	return rv, errs
}

func (b *keyspaceKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	ids := strings.SplitN(key, "/", 2)

	namespace, err := b.namespace.store.actualStore.NamespaceById(ids[0])
	if namespace != nil {
		keyspace, _ := namespace.KeyspaceById(ids[1])
		if keyspace != nil {
			doc := value.NewAnnotatedValue(map[string]interface{}{
				"id":           keyspace.Id(),
				"name":         keyspace.Name(),
				"namespace_id": namespace.Id(),
				"datastore_id": b.namespace.store.actualStore.Id(),
			})
			return doc, nil
		}
	}
	return nil, err
}

func (b *keyspaceKeyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newKeyspacesKeyspace(p *namespace) (*keyspaceKeyspace, errors.Error) {
	b := new(keyspaceKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_KEYSPACES

	primary := &keyspaceIndex{name: "#primary", keyspace: b}
	b.indexer = &systemIndexer{keyspace: b, indexes: make(map[string]datastore.Index), primary: primary}

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
	return datastore.DEFAULT
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

func (pi *keyspaceIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *keyspaceIndex) Drop() errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *keyspaceIndex) Scan(span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	val := ""

	a := span.Seek[0].Actual()
	switch a := a.(type) {
	case string:
		val = a
	default:
		conn.Error(errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", a, a)))
		return
	}

	ids := strings.SplitN(val, "/", 2)
	if len(ids) != 2 {
		return
	}

	namespace, _ := pi.keyspace.namespace.store.actualStore.NamespaceById(ids[0])
	if namespace == nil {
		return
	}

	keyspace, _ := namespace.KeyspaceById(ids[1])
	if keyspace != nil {
		entry := datastore.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s", namespace.Id(), keyspace.Id())}
		conn.EntryChannel() <- &entry
	}
}

func (pi *keyspaceIndex) ScanEntries(limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	namespaceIds, err := pi.keyspace.namespace.store.actualStore.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.namespace.store.actualStore.NamespaceById(namespaceId)
			if err == nil {
				keyspaceIds, err := namespace.KeyspaceIds()
				if err == nil {
					for i, keyspaceId := range keyspaceIds {
						if limit > 0 && int64(i) > limit {
							break
						}
						entry := datastore.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s", namespaceId, keyspaceId)}
						conn.EntryChannel() <- &entry
					}
				}
			}
		}
	}
}
