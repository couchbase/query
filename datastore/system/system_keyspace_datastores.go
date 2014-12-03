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

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type storeKeyspace struct {
	namespace *namespace
	name      string
	si        datastore.Indexer
}

func (b *storeKeyspace) Release() {
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

func (b *storeKeyspace) Count() (int64, errors.Error) {
	return 1, nil
}

func (b *storeKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *storeKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *storeKeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *storeKeyspace) IndexNames() ([]string, errors.Error) {
	return b.si.IndexNames()
}

func (b *storeKeyspace) IndexById(id string) (datastore.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *storeKeyspace) IndexByName(name string) (datastore.Index, errors.Error) {
	return b.si.IndexByName(name)
}

func (b *storeKeyspace) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	return b.si.IndexByPrimary()
}

func (b *storeKeyspace) Indexes() ([]datastore.Index, errors.Error) {
	return b.si.Indexes()
}

func (b *storeKeyspace) Authenticate(credentials datastore.Credentials, requested datastore.Privileges) errors.Error {
	return nil
}

func (b *storeKeyspace) CreatePrimaryIndex(using datastore.IndexType) (datastore.PrimaryIndex, errors.Error) {
	return b.si.CreatePrimaryIndex()
}

func (b *storeKeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression, using datastore.IndexType) (datastore.Index, errors.Error) {
	return b.si.CreateIndex(name, equalKey, rangeKey, where)
}

func (b *storeKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, errors.Error) {
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

func (b *storeKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	if key == b.namespace.store.actualStore.Id() {
		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":  b.namespace.store.actualStore.Id(),
			"url": b.namespace.store.actualStore.URL(),
		})
		return doc, nil
	}
	return nil, errors.NewError(nil, "Not Found")
}

func (b *storeKeyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *storeKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *storeKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *storeKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func newStoresKeyspace(p *namespace) (*storeKeyspace, errors.Error) {
	b := new(storeKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_DATASTORES

	b.si = newSystemIndexer(b)
	b.si.CreatePrimaryIndex()

	return b, nil
}

type systemIndexer struct {
	keyspace datastore.Keyspace
	indexes  map[string]datastore.Index
	primary  datastore.PrimaryIndex
}

func newSystemIndexer(keyspace datastore.Keyspace) datastore.Indexer {

	return &systemIndexer{
		keyspace: keyspace,
		indexes:  make(map[string]datastore.Index),
	}
}

func (si *systemIndexer) KeyspaceId() string {
	return si.keyspace.Id()
}

func (si *systemIndexer) Name() datastore.IndexType {
	return datastore.UNSPECIFIED
}

func (si *systemIndexer) IndexIds() ([]string, errors.Error) {
	rv := make([]string, 0, len(si.indexes))
	for name, _ := range si.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (si *systemIndexer) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(si.indexes))
	for name, _ := range si.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (si *systemIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return si.IndexByName(id)
}

func (si *systemIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := si.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (si *systemIndexer) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	return si.primary, nil
}

func (si *systemIndexer) Indexes() ([]datastore.Index, errors.Error) {
	rv := make([]datastore.Index, 0, len(si.indexes))
	for _, index := range si.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (si *systemIndexer) CreatePrimaryIndex() (datastore.PrimaryIndex, errors.Error) {
	if si.primary == nil {
		pi := new(storeIndex)
		si.primary = pi
		pi.keyspace = si.keyspace.(*storeKeyspace)
		pi.name = "#primary"
		si.indexes[pi.name] = pi
	}

	return si.primary, nil
}

func (mi *systemIndexer) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression) (datastore.Index, errors.Error) {
	return nil, errors.NewError(nil, "Create index is not supported for file-based datastore.")
}

type storeIndex struct {
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
	return datastore.UNSPECIFIED
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

func (pi *storeIndex) State() (datastore.IndexState, errors.Error) {
	return datastore.ONLINE, nil
}

func (pi *storeIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *storeIndex) Drop() errors.Error {
	return errors.NewError(nil, "This primary index cannot be dropped.")
}

func (pi *storeIndex) Scan(span *datastore.Span, distinct bool, limit int64, conn *datastore.IndexConnection) {
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

	if strings.EqualFold(val, pi.keyspace.namespace.store.actualStore.Id()) {
		entry := datastore.IndexEntry{PrimaryKey: pi.keyspace.namespace.store.actualStore.Id()}
		conn.EntryChannel() <- &entry
	}
}

func (pi *storeIndex) ScanEntries(limit int64, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	entry := datastore.IndexEntry{PrimaryKey: pi.keyspace.namespace.store.actualStore.Id()}
	conn.EntryChannel() <- &entry
}
