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

type dualkeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
}

func (b *dualkeyspace) Release() {
}

func (b *dualkeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *dualkeyspace) Id() string {
	return b.Name()
}

func (b *dualkeyspace) Name() string {
	return b.name
}

func (b *dualkeyspace) Count() (int64, errors.Error) {
	return 1, nil
}

func (b *dualkeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *dualkeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *dualkeyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *dualkeyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *dualkeyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *dualkeyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *dualkeyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *dualkeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func (b *dualkeyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
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

func (b *dualkeyspace) FetchOne(key string) (value.Value, errors.Error) {
	doc := map[string]interface{}{}
	return value.NewValue(doc), nil
}

func (b *dualkeyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *dualkeyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *dualkeyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *dualkeyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
}

func newDualKeyspace(p *namespace) (*dualkeyspace, errors.Error) {
	b := new(dualkeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_DUAL

	b.primary = &dualIndex{name: "primary", keyspace: b}

	return b, nil
}

type dualIndex struct {
	name     string
	keyspace *dualkeyspace
}

func (pi *dualIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *dualIndex) Id() string {
	return pi.Name()
}

func (pi *dualIndex) Name() string {
	return pi.name
}

func (pi *dualIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *dualIndex) Drop() errors.Error {
	return errors.NewError(nil, "Primary index cannot be dropped.")
}

func (pi *dualIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *dualIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *dualIndex) Condition() expression.Expression {
	return nil
}

func (pi *dualIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *dualIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	entry := catalog.IndexEntry{PrimaryKey: KEYSPACE_NAME_DUAL}
	conn.EntryChannel() <- &entry
}

func (pi *dualIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
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

	if strings.EqualFold(val, KEYSPACE_NAME_DUAL) {
		entry := catalog.IndexEntry{PrimaryKey: KEYSPACE_NAME_DUAL}
		conn.EntryChannel() <- &entry
	}
}
