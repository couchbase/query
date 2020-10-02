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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type dualKeyspace struct {
	keyspaceBase
	di datastore.Indexer
}

func (b *dualKeyspace) Release(close bool) {
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

func (b *dualKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return 1, nil
}

func (b *dualKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *dualKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.di, nil
}

func (b *dualKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.di}, nil
}

func (b *dualKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
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
			item.NewMeta()["keyspace"] = b.fullName
			item.SetId(k)
		}

		keysMap[k] = item
	}

	return
}

func (b *dualKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	return value.NewAnnotatedValue(nil), nil
}

func (b *dualKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemDatastoreError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemDatastoreError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemDatastoreError(nil, "Mutations not allowed on system:dual.")
}

func (b *dualKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewSystemDatastoreError(nil, "Mutations not allowed on system:dual.")
}

func newDualKeyspace(p *namespace) (*dualKeyspace, errors.Error) {
	b := new(dualKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_DUAL)

	primary := &dualIndex{name: "#primary", keyspace: b}
	b.di = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.di)

	return b, nil
}

type dualIndex struct {
	indexBase
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
	return datastore.SYSTEM
}

func (pi *dualIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *dualIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *dualIndex) Condition() expression.Expression {
	return nil
}

func (pi *dualIndex) IsPrimary() bool {
	return true
}

func (pi *dualIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *dualIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *dualIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "For system:dual")
}

func (pi *dualIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	// no fields to compare - we just do a primary scan of one
	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (pi *dualIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	entry := datastore.IndexEntry{PrimaryKey: KEYSPACE_NAME_DUAL}
	sendSystemKey(conn, &entry)
}
