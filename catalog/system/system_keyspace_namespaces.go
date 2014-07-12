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

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type namespacekeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
}

func (b *namespacekeyspace) Release() {
}

func (b *namespacekeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *namespacekeyspace) Id() string {
	return b.Name()
}

func (b *namespacekeyspace) Name() string {
	return b.name
}

func (b *namespacekeyspace) Count() (int64, errors.Error) {
	namespaceIds, excp := b.namespace.site.actualSite.NamespaceIds()
	if excp == nil {
		return int64(len(namespaceIds)), nil
	}
	return 0, errors.NewError(excp, "")
}

func (b *namespacekeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *namespacekeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *namespacekeyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *namespacekeyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *namespacekeyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *namespacekeyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *namespacekeyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
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

func (b *namespacekeyspace) FetchOne(key string) (value.Value, errors.Error) {
	namespace, excp := b.namespace.site.actualSite.NamespaceById(key)
	if namespace != nil {
		doc := value.NewValue(map[string]interface{}{
			"id":      namespace.Id(),
			"name":    namespace.Name(),
			"site_id": b.namespace.site.actualSite.Id(),
		})
		return doc, nil
	}
	return nil, errors.NewError(excp, "Not Found")
}

func (b *namespacekeyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *namespacekeyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *namespacekeyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *namespacekeyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
}

func (b *namespacekeyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *namespacekeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func newNamespacesKeyspace(p *namespace) (*namespacekeyspace, errors.Error) {
	b := new(namespacekeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_NAMESPACES

	b.primary = &namespaceIndex{name: "primary", keyspace: b}

	return b, nil
}

type namespaceIndex struct {
	name     string
	keyspace *namespacekeyspace
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

func (pi *namespaceIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *namespaceIndex) Drop() errors.Error {
	return errors.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *namespaceIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *namespaceIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *namespaceIndex) Condition() expression.Expression {
	return nil
}

func (pi *namespaceIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *namespaceIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	namespaceIds, err := pi.keyspace.namespace.site.actualSite.NamespaceIds()
	if err == nil {
		for i, namespaceId := range namespaceIds {
			if limit > 0 && int64(i) > limit {
				break
			}

			entry := catalog.IndexEntry{PrimaryKey: namespaceId}
			conn.EntryChannel() <- &entry
		}
	}
}

func (pi *namespaceIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	val := ""

	a := span.Equal[0].Actual()
	switch a := a.(type) {
	case string:
		val = a
	default:
		conn.SendError(errors.NewError(nil, fmt.Sprintf("Invalid equality value %v of type %T.", a, a)))
		return
	}

	namespace, _ := pi.keyspace.namespace.site.actualSite.NamespaceById(val)
	if namespace != nil {
		entry := catalog.IndexEntry{PrimaryKey: namespace.Id()}
		conn.EntryChannel() <- &entry
	}
}
