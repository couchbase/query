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
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type requestLogKeyspace struct {
	namespace *namespace
	name      string
	indexer   datastore.Indexer
}

func (b *requestLogKeyspace) Release() {
}

func (b *requestLogKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *requestLogKeyspace) Id() string {
	return b.Name()
}

func (b *requestLogKeyspace) Name() string {
	return b.name
}

func (b *requestLogKeyspace) Count() (int64, errors.Error) {
	return int64(accounting.RequestsCount()), nil
}

func (b *requestLogKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *requestLogKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *requestLogKeyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]datastore.AnnotatedPair, 0, len(keys))

	accounting.RequestsForeach(func(id string, entry *accounting.RequestLogEntry) {
		item := value.NewAnnotatedValue(map[string]interface{}{
			"RequestId":   id,
			"State":       entry.State,
			"ElapsedTime": entry.ElapsedTime,
			"ServiceTime": entry.ServiceTime,
			"ResultCount": entry.ResultCount,
			"ResultSize":  entry.ResultSize,
			"ErrorCount":  entry.ErrorCount,
			"SortCount":   entry.SortCount,
			"Time":        entry.Time.String(),
		})
		if entry.Statement != "" {
			item.SetField("Statement", entry.Statement)
		}
		if entry.PreparedName != "" {
			item.SetField("PreparedName", entry.PreparedName)
			item.SetField("PreparedText", entry.PreparedText)
		}
		item.SetAttachment("meta", map[string]interface{}{
			"id": id,
		})
		rv = append(rv, datastore.AnnotatedPair{
			Key:   id,
			Value: item,
		})
	})
	return rv, errs
}

func (b *requestLogKeyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *requestLogKeyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *requestLogKeyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *requestLogKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	for i, name := range deletes {
		err := accounting.RequestDelete(name)

		// save memory allocations by making a new slice only on errors
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

func newRequestsKeyspace(p *namespace) (*requestLogKeyspace, errors.Error) {
	b := new(requestLogKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_REQUESTS

	primary := &requestLogIndex{name: "#primary", keyspace: b}
	b.indexer = &systemIndexer{keyspace: b, indexes: make(map[string]datastore.Index), primary: primary}

	return b, nil
}

type requestLogIndex struct {
	name     string
	keyspace *requestLogKeyspace
}

func (pi *requestLogIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *requestLogIndex) Id() string {
	return pi.Name()
}

func (pi *requestLogIndex) Name() string {
	return pi.name
}

func (pi *requestLogIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *requestLogIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *requestLogIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *requestLogIndex) Condition() expression.Expression {
	return nil
}

func (pi *requestLogIndex) IsPrimary() bool {
	return true
}

func (pi *requestLogIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *requestLogIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *requestLogIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *requestLogIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	// NOP
}

func (pi *requestLogIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	accounting.RequestsForeach(func(id string, entry *accounting.RequestLogEntry) {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		conn.EntryChannel() <- &indexEntry
	})
}
