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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type preparedsKeyspace struct {
	namespace *namespace
	name      string
	indexer   datastore.Indexer
}

func (b *preparedsKeyspace) Release() {
}

func (b *preparedsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *preparedsKeyspace) Id() string {
	return b.Name()
}

func (b *preparedsKeyspace) Name() string {
	return b.name
}

func (b *preparedsKeyspace) Count() (int64, errors.Error) {
	return int64(plan.CountPrepareds()), nil
}

func (b *preparedsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *preparedsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *preparedsKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]datastore.AnnotatedPair, 0, len(keys))

	for _, key := range keys {
		p := plan.PreparedEntry(key)

		itemMap := map[string]interface{}{
			"name":         key,
			"uses":         p.Uses,
			"statement":    p.Text,
			"encoded_plan": p.Plan,
		}
		if p.Uses > 0 {
			itemMap["lastUse"] = p.LastUse
		}
		item := value.NewAnnotatedValue(itemMap)
		item.SetAttachment("meta", map[string]interface{}{
			"id": key,
		})
		rv = append(rv, datastore.AnnotatedPair{
			Key:   key,
			Value: item,
		})
	}
	return rv, errs
}

func (b *preparedsKeyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *preparedsKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *preparedsKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *preparedsKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	for i, name := range deletes {
		err := plan.DeletePrepared(name)
		if err != nil {
			deleted := make([]string, i)
			if i > 0 {
				copy(deleted, deletes[0:i-1])
			}
			return deleted, err
		}
	}
	return deletes, nil
}

func newPreparedsKeyspace(p *namespace) (*preparedsKeyspace, errors.Error) {
	b := new(preparedsKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_PREPAREDS

	primary := &preparedsIndex{name: "#primary", keyspace: b}
	b.indexer = &systemIndexer{keyspace: b, indexes: make(map[string]datastore.Index), primary: primary}

	return b, nil
}

type preparedsIndex struct {
	name     string
	keyspace *preparedsKeyspace
}

func (pi *preparedsIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *preparedsIndex) Id() string {
	return pi.Name()
}

func (pi *preparedsIndex) Name() string {
	return pi.name
}

func (pi *preparedsIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *preparedsIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *preparedsIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *preparedsIndex) Condition() expression.Expression {
	return nil
}

func (pi *preparedsIndex) IsPrimary() bool {
	return true
}

func (pi *preparedsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *preparedsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *preparedsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *preparedsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	// NOP
}

func (pi *preparedsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	names := plan.NamePrepareds()

	for _, name := range names {
		entry := datastore.IndexEntry{PrimaryKey: name}
		conn.EntryChannel() <- &entry
	}
}
