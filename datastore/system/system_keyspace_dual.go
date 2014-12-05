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

type dualKeyspace struct {
	namespace *namespace
	name      string
	indexes   map[string]datastore.Index
	primary   datastore.PrimaryIndex
}

func (b *dualKeyspace) Release() {
}

func (b *dualKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *dualKeyspace) Id() string {
	return b.Name()
}

func (b *dualKeyspace) Name() string {
	return b.name
}

func (b *dualKeyspace) Count() (int64, errors.Error) {
	return 1, nil
}

func (b *dualKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *dualKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *dualKeyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *dualKeyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *dualKeyspace) IndexById(id string) (datastore.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *dualKeyspace) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *dualKeyspace) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *dualKeyspace) Indexes() ([]datastore.Index, errors.Error) {
	rv := make([]datastore.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *dualKeyspace) CreatePrimaryIndex(using datastore.IndexType) (datastore.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression, using datastore.IndexType) (datastore.Index, errors.Error) {
	return nil, errors.NewError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, errors.Error) {
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

func (b *dualKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	return value.NewAnnotatedValue(nil), nil
}

func (b *dualKeyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return nil, errors.NewError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return nil, errors.NewError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return nil, errors.NewError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Delete(deletes []string) errors.Error {
	return errors.NewError(nil, "Mutations not allowed on system:dual.")
}

func newDualKeyspace(p *namespace) (*dualKeyspace, errors.Error) {
	b := new(dualKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_DUAL

	b.primary = &dualIndex{name: "#primary", keyspace: b}

	return b, nil
}

type dualIndex struct {
	name     string
	keyspace *dualKeyspace
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

func (pi *dualIndex) Type() datastore.IndexType {
	return datastore.UNSPECIFIED
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

func (pi *dualIndex) State() (datastore.IndexState, errors.Error) {
	return datastore.ONLINE, nil
}

func (pi *dualIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *dualIndex) Drop() errors.Error {
	return errors.NewError(nil, "This primary index cannot be dropped.")
}

func (pi *dualIndex) Scan(span *datastore.Span, distinct bool, limit int64, conn *datastore.IndexConnection) {
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

	if strings.EqualFold(val, KEYSPACE_NAME_DUAL) {
		entry := datastore.IndexEntry{PrimaryKey: KEYSPACE_NAME_DUAL}
		conn.EntryChannel() <- &entry
	}
}

func (pi *dualIndex) ScanEntries(limit int64, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	entry := datastore.IndexEntry{PrimaryKey: KEYSPACE_NAME_DUAL}
	conn.EntryChannel() <- &entry
}
