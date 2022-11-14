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
	keyspaceBase
	name    string
	indexer datastore.Indexer
}

func (b *requestLogKeyspace) Release(close bool) {
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
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		context.Warning(warn)
	})
	return int64(server.RequestsCount() + count), nil
}

func (b *requestLogKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *requestLogKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *requestLogKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *requestLogKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	creds, authToken := credsFromContext(context)

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != whoAmI {
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
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					context.Warning(warn)
				},
				creds, authToken)
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
					"requestTime":     entry.Time.Format(expression.DEFAULT_FORMAT),
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
				if entry.UseFts {
					item.SetField("useFts", entry.UseFts)
				}
				if entry.PreparedName != "" {
					item.SetField("preparedName", entry.PreparedName)
					item.SetField("preparedText", entry.PreparedText)
				}
				if entry.Mutations != 0 {
					item.SetField("mutations", entry.Mutations)
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
				if entry.Users != "" {
					item.SetField("users", entry.Users)
				}
				if entry.RemoteAddr != "" {
					item.SetField("remoteAddr", entry.RemoteAddr)
				}
				if entry.UserAgent != "" {
					item.SetField("userAgent", entry.UserAgent)
				}
				if entry.Tag != "" {
					item.SetField("~tag", entry.Tag)
				}
				if entry.Errors != nil {
					errors := make([]value.Value, len(entry.Errors))
					for i, e := range entry.Errors {
						errors[i] = value.NewValue(e.Object())
					}
					item.SetField("errors", errors)
				}

				meta := map[string]interface{}{
					"id": key,
				}
				if entry.Timings != nil {
					bytes, _ := json.Marshal(entry.Timings)
					meta["plan"] = bytes
				}
				item.SetAttachment("meta", meta)
				item.SetId(key)
				keysMap[key] = item
			})
		}
	}
	return
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

func (b *requestLogKeyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	var err errors.Error

	creds, authToken := credsFromContext(context)

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, name := range deletes {
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"completed_requests", "DELETE", nil,
				func(warn errors.Error) {
					context.Warning(warn)
				},
				creds, authToken)

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
	setKeyspaceBase(&b.keyspaceBase, p)
	b.name = KEYSPACE_NAME_REQUESTS

	primary := &requestLogIndex{
		name:     "#primary",
		keyspace: b,
		primary:  true,
	}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `node`
	expr, err := parser.Parse(`node`)

	if err == nil {
		key := expression.Expressions{expr}
		nodes := &requestLogIndex{
			name:     "#nodes",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		setIndexBase(&nodes.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type requestLogIndex struct {
	indexBase
	name     string
	keyspace *requestLogKeyspace
	primary  bool
	idxKey   expression.Expressions
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
	return datastore.SYSTEM
}

func (pi *requestLogIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *requestLogIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *requestLogIndex) Condition() expression.Expression {
	return nil
}

func (pi *requestLogIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *requestLogIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
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
		var entry *datastore.IndexEntry
		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		if spanEvaluator.isEquals() {

			// now that the node name can change in flight, use a consistent one across the scan
			whoAmI := distributed.RemoteAccess().WhoAmI()
			if spanEvaluator.key() == distributed.RemoteAccess().WhoAmI() {
				server.RequestsForeach(func(id string, request *server.RequestLogEntry) bool {
					entry = &datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, id),
						EntryKey:   value.Values{value.NewValue(whoAmI)},
					}
					return true
				}, func() bool {
					return sendSystemKey(conn, entry)
				})
			} else {
				nodes := []string{spanEvaluator.key()}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "completed_requests", func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		} else {

			// now that the node name can change in flight, use a consistent one across the scan
			whoAmI := distributed.RemoteAccess().WhoAmI()
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				if spanEvaluator.evaluate(node) {
					if node == whoAmI {
						server.RequestsForeach(func(id string, request *server.RequestLogEntry) bool {
							entry = &datastore.IndexEntry{
								PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, id),
								EntryKey:   value.Values{value.NewValue(distributed.RemoteAccess().WhoAmI())},
							}
							return true
						}, func() bool {
							return sendSystemKey(conn, entry)
						})
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}
			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "completed_requests", func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		}
	}
}

func (pi *requestLogIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry
	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := distributed.RemoteAccess().WhoAmI()
	server.RequestsForeach(func(id string, request *server.RequestLogEntry) bool {
		entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, id)}
		return true
	}, func() bool {
		return sendSystemKey(conn, entry)
	})
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
