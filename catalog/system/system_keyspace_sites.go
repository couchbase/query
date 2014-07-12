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

type sitekeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
}

func (b *sitekeyspace) Release() {
}

func (b *sitekeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *sitekeyspace) Id() string {
	return b.Name()
}

func (b *sitekeyspace) Name() string {
	return b.name
}

func (b *sitekeyspace) Count() (int64, errors.Error) {
	return 1, nil
}

func (b *sitekeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *sitekeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *sitekeyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *sitekeyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *sitekeyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *sitekeyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *sitekeyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *sitekeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func (b *sitekeyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
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

func (b *sitekeyspace) FetchOne(key string) (value.Value, errors.Error) {
	if key == b.namespace.site.actualSite.Id() {
		doc := value.NewValue(map[string]interface{}{
			"id":  b.namespace.site.actualSite.Id(),
			"url": b.namespace.site.actualSite.URL(),
		})
		return doc, nil
	}
	return nil, errors.NewError(nil, "Not Found")
}

func (b *sitekeyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *sitekeyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *sitekeyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *sitekeyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
}

func newSitesKeyspace(p *namespace) (*sitekeyspace, errors.Error) {
	b := new(sitekeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_SITES

	b.primary = &siteIndex{name: "primary", keyspace: b}

	return b, nil
}

type siteIndex struct {
	name     string
	keyspace *sitekeyspace
}

func (pi *siteIndex) KeyspaceId() string {
	return pi.name
}

func (pi *siteIndex) Id() string {
	return pi.Name()
}

func (pi *siteIndex) Name() string {
	return pi.name
}

func (pi *siteIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *siteIndex) Drop() errors.Error {
	return errors.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *siteIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *siteIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *siteIndex) Condition() expression.Expression {
	return nil
}

func (pi *siteIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *siteIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	entry := catalog.IndexEntry{PrimaryKey: pi.keyspace.namespace.site.actualSite.Id()}
	conn.EntryChannel() <- &entry
}

func (pi *siteIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
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

	if strings.EqualFold(val, pi.keyspace.namespace.site.actualSite.Id()) {
		entry := catalog.IndexEntry{PrimaryKey: pi.keyspace.namespace.site.actualSite.Id()}
		conn.EntryChannel() <- &entry
	}
}
