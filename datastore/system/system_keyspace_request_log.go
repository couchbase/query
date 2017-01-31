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
	"encoding/json"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/server"
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

func (b *requestLogKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests", func(id string) {
		count++
	}, func(warn errors.Error) {

		// FIXME Count does not handle warnings
	})
	return int64(server.RequestsCount() + count), nil
}

func (b *requestLogKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *requestLogKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *requestLogKeyspace) Fetch(keys []string, context datastore.QueryContext) ([]value.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]value.AnnotatedPair, 0, len(keys))

	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != distributed.RemoteAccess().WhoAmI() {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"completed_requests", "POST",
				func(doc map[string]interface{}) {

					meta := map[string]interface{}{
						"id": key,
					}
					t, ok := doc["timings"]
					if ok {
						meta["plan"] = t
						delete(doc, "timings")
					}
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					remoteValue.SetAttachment("meta", meta)
					rv = append(rv, value.AnnotatedPair{
						Name:  key,
						Value: remoteValue,
					})
				},

				// FIXME Fetch() does not handle warnings
				func(warn errors.Error) {
				})
		} else {

			// local entry
			server.RequestDo(localKey, func(entry *server.RequestLogEntry) {
				item := value.NewAnnotatedValue(map[string]interface{}{
					"requestId":       localKey,
					"state":           entry.State,
					"elapsedTime":     entry.ElapsedTime.String(),
					"serviceTime":     entry.ServiceTime.String(),
					"resultCount":     entry.ResultCount,
					"resultSize":      entry.ResultSize,
					"errorCount":      entry.ErrorCount,
					"time":            entry.Time.String(),
					"scanConsistency": entry.ScanConsistency,
				})
				if node != "" {
					item.SetField("node", node)
				}
				if entry.ClientId != "" {
					item.SetField("clientContextID", entry.ClientId)
				}
				if entry.Statement != "" {
					item.SetField("statement", entry.Statement)
				}
				if entry.PreparedName != "" {
					item.SetField("preparedName", entry.PreparedName)
					item.SetField("preparedText", entry.PreparedText)
				}
				if entry.PhaseTimes != nil {
					item.SetField("phaseTimes", entry.PhaseTimes)
				}
				if entry.PhaseCounts != nil {
					item.SetField("phaseCounts", entry.PhaseCounts)
				}
				if entry.PhaseOperators != nil {
					item.SetField("phaseOperators", entry.PhaseOperators)
				}
				if entry.PositionalArgs != nil {
					item.SetField("positionalArgs", entry.PositionalArgs)
				}
				if entry.NamedArgs != nil {
					item.SetField("namedArgs", entry.NamedArgs)
				}
				meta := map[string]interface{}{
					"id": key,
				}
				if entry.Timings != nil {
					bytes, _ := json.Marshal(entry.Timings)
					meta["plan"] = bytes
				}
				item.SetAttachment("meta", meta)
				rv = append(rv, value.AnnotatedPair{
					Name:  key,
					Value: item,
				})
			})
		}
	}
	return rv, errs
}

func (b *requestLogKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *requestLogKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *requestLogKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *requestLogKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	var err errors.Error

	for i, name := range deletes {
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != distributed.RemoteAccess().WhoAmI() {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"completed_requests", "DELETE",
				nil,

				// FIXME Delete() doesn't do warnings
				func(warn errors.Error) {
				})

			// local entry
		} else {
			err = server.RequestDelete(localKey)
		}

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

	primary := &requestLogIndex{
		name:     "#primary",
		keyspace: b,
		primary:  true,
	}
	b.indexer = newSystemIndexer(b, primary)

	// add a secondary index on `node`
	if distributed.RemoteAccess().WhoAmI() != "" {
		expr, err := parser.Parse(`node`)

		if err == nil {
			key := expression.Expressions{expr}
			nodes := &requestLogIndex{
				name:     "#nodes",
				keyspace: b,
				primary:  false,
				seekKey:  key,
			}
			b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
		} else {
			return nil, errors.NewSystemDatastoreError(err, "")
		}
	}

	return b, nil
}

type requestLogIndex struct {
	name     string
	keyspace *requestLogKeyspace
	primary  bool
	seekKey  expression.Expressions
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
	return pi.seekKey
}

func (pi *requestLogIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *requestLogIndex) Condition() expression.Expression {
	return nil
}

func (pi *requestLogIndex) IsPrimary() bool {
	return pi.primary
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
	if span == nil || pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer close(conn.EntryChannel())

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		if spanEvaluator.isEquals() {
			if spanEvaluator.key() == distributed.RemoteAccess().WhoAmI() {
				server.RequestsForeach(func(id string, entry *server.RequestLogEntry) {
					indexEntry := datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), id)}
					conn.EntryChannel() <- &indexEntry
				})
			} else {
				nodes := []string{spanEvaluator.key()}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "completedctive_requests", func(id string) {
					indexEntry := datastore.IndexEntry{PrimaryKey: id}
					conn.EntryChannel() <- &indexEntry
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		} else {
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				if spanEvaluator.evaluate(node) {
					if spanEvaluator.key() == distributed.RemoteAccess().WhoAmI() {
						server.RequestsForeach(func(id string, entry *server.RequestLogEntry) {
							indexEntry := datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), id)}
							conn.EntryChannel() <- &indexEntry
						})
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}
			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "completed_requests", func(id string) {
					indexEntry := datastore.IndexEntry{PrimaryKey: id}
					conn.EntryChannel() <- &indexEntry
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		}
	}
}

func (pi *requestLogIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	server.RequestsForeach(func(id string, entry *server.RequestLogEntry) {
		indexEntry := datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), id)}
		conn.EntryChannel() <- &indexEntry
	})
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests", func(id string) {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		conn.EntryChannel() <- &indexEntry
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
