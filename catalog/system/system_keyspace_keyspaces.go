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

type keyspacekeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
}

func (b *keyspacekeyspace) Release() {
}

func (b *keyspacekeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspacekeyspace) Id() string {
	return b.Name()
}

func (b *keyspacekeyspace) Name() string {
	return b.name
}

func (b *keyspacekeyspace) Count() (int64, errors.Error) {
	count := int64(0)
	namespaceIds, excp := b.namespace.site.actualSite.NamespaceIds()
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.namespace.site.actualSite.NamespaceById(namespaceId)
			if excp == nil {
				keyspaceIds, excp := namespace.KeyspaceIds()
				if excp == nil {
					count += int64(len(keyspaceIds))
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

func (b *keyspacekeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *keyspacekeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *keyspacekeyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *keyspacekeyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *keyspacekeyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *keyspacekeyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *keyspacekeyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *keyspacekeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func (b *keyspacekeyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
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

func (b *keyspacekeyspace) FetchOne(key string) (value.Value, errors.Error) {
	ids := strings.SplitN(key, "/", 2)

	namespace, err := b.namespace.site.actualSite.NamespaceById(ids[0])
	if namespace != nil {
		keyspace, _ := namespace.KeyspaceById(ids[1])
		if keyspace != nil {
			doc := value.NewValue(map[string]interface{}{
				"id":           keyspace.Id(),
				"name":         keyspace.Name(),
				"namespace_id": namespace.Id(),
				"site_id":      b.namespace.site.actualSite.Id(),
			})
			return doc, nil
		}
	}
	return nil, err
}

func (b *keyspacekeyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspacekeyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspacekeyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspacekeyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
}

func newKeyspacesKeyspace(p *namespace) (*keyspacekeyspace, errors.Error) {
	b := new(keyspacekeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_KEYSPACES

	b.primary = &keyspaceIndex{name: "primary", keyspace: b}

	return b, nil
}

type keyspaceIndex struct {
	name     string
	keyspace *keyspacekeyspace
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

func (pi *keyspaceIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *keyspaceIndex) Drop() errors.Error {
	return errors.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *keyspaceIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *keyspaceIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *keyspaceIndex) Condition() expression.Expression {
	return nil
}

func (pi *keyspaceIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *keyspaceIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	namespaceIds, err := pi.keyspace.namespace.site.actualSite.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.namespace.site.actualSite.NamespaceById(namespaceId)
			if err == nil {
				keyspaceIds, err := namespace.KeyspaceIds()
				if err == nil {
					for i, keyspaceId := range keyspaceIds {
						if limit > 0 && int64(i) > limit {
							break
						}
						entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s", namespaceId, keyspaceId)}
						conn.EntryChannel() <- &entry
					}
				}
			}
		}
	}
}

func (pi *keyspaceIndex) Scan(span catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
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

	ids := strings.SplitN(val, "/", 2)
	if len(ids) != 2 {
		return
	}

	namespace, _ := pi.keyspace.namespace.site.actualSite.NamespaceById(ids[0])
	if namespace == nil {
		return
	}

	keyspace, _ := namespace.KeyspaceById(ids[1])
	if keyspace != nil {
		entry := catalog.IndexEntry{PrimaryKey: fmt.Sprintf("%s/%s", namespace.Id(), keyspace.Id())}
		conn.EntryChannel() <- &entry
	}
}
