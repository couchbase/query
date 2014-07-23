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

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type datastoreKeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
}

func (b *datastoreKeyspace) Release() {
}

func (b *datastoreKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *datastoreKeyspace) Id() string {
	return b.Name()
}

func (b *datastoreKeyspace) Name() string {
	return b.name
}

func (b *datastoreKeyspace) Count() (int64, errors.Error) {
	return 1, nil
}

func (b *datastoreKeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *datastoreKeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *datastoreKeyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *datastoreKeyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *datastoreKeyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *datastoreKeyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *datastoreKeyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *datastoreKeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func (b *datastoreKeyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
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

func (b *datastoreKeyspace) FetchOne(key string) (value.Value, errors.Error) {
	if key == b.namespace.datastore.actualDatastore.Id() {
		doc := value.NewValue(map[string]interface{}{
			"id":  b.namespace.datastore.actualDatastore.Id(),
			"url": b.namespace.datastore.actualDatastore.URL(),
		})
		return doc, nil
	}
	return nil, errors.NewError(nil, "Not Found")
}

func (b *datastoreKeyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *datastoreKeyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *datastoreKeyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *datastoreKeyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
}

func newDatastoresKeyspace(p *namespace) (*datastoreKeyspace, errors.Error) {
	b := new(datastoreKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_DATASTORES

	b.primary = &datastoreIndex{name: "primary", keyspace: b}

	return b, nil
}

type datastoreIndex struct {
	name     string
	keyspace *datastoreKeyspace
}

func (pi *datastoreIndex) KeyspaceId() string {
	return pi.name
}

func (pi *datastoreIndex) Id() string {
	return pi.Name()
}

func (pi *datastoreIndex) Name() string {
	return pi.name
}

func (pi *datastoreIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *datastoreIndex) Drop() errors.Error {
	return errors.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *datastoreIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *datastoreIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *datastoreIndex) Condition() expression.Expression {
	return nil
}

func (pi *datastoreIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *datastoreIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	entry := catalog.IndexEntry{PrimaryKey: pi.keyspace.namespace.datastore.actualDatastore.Id()}
	conn.EntryChannel() <- &entry
}

func (pi *datastoreIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
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

	if strings.EqualFold(val, pi.keyspace.namespace.datastore.actualDatastore.Id()) {
		entry := catalog.IndexEntry{PrimaryKey: pi.keyspace.namespace.datastore.actualDatastore.Id()}
		conn.EntryChannel() <- &entry
	}
}
