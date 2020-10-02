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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type activeRequestsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *activeRequestsKeyspace) Release(close bool) {
}

func (b *activeRequestsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *activeRequestsKeyspace) Id() string {
	return b.Name()
}

func (b *activeRequestsKeyspace) Name() string {
	return b.name
}

func (b *activeRequestsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "active_requests", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		context.Warning(warn)
	})
	c, err := server.ActiveRequestsCount()
	return int64(c + count), err
}

func (b *activeRequestsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *activeRequestsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *activeRequestsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *activeRequestsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"active_requests", "POST",
				func(doc map[string]interface{}) {

					t, ok := doc["timings"]
					if ok {
						delete(doc, "timings")
					}
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					meta := remoteValue.NewMeta()
					meta["keyspace"] = b.fullName
					if ok {
						meta["plan"] = t
					}
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},

				func(warn errors.Error) {
					context.Warning(warn)
				},
				distributed.NO_CREDS, "")
		} else {
			var item value.AnnotatedValue

			// local entry
			err := server.ActiveRequestsGet(localKey, func(request server.Request) {

				item = value.NewAnnotatedValue(map[string]interface{}{
					"requestId":       localKey,
					"requestTime":     request.RequestTime().Format(expression.DEFAULT_FORMAT),
					"elapsedTime":     time.Since(request.RequestTime()).String(),
					"executionTime":   time.Since(request.ServiceTime()).String(),
					"state":           request.State().StateName(),
					"scanConsistency": request.ScanConsistency(),
				})
				if node != "" {
					item.SetField("node", node)
				}
				cId := request.ClientID().String()
				if cId != "" {
					item.SetField("clientContextID", cId)
				}
				if request.Statement() != "" {
					item.SetField("statement", request.Statement())
				}
				if request.QueryContext() != "" {
					item.SetField("queryContext", request.QueryContext())
				}
				if request.UseFts() {
					item.SetField("useFts", request.UseFts())
				}
				if request.UseCBO() {
					item.SetField("useCBO", request.UseCBO())
				}
				if request.TxId() != "" {
					item.SetField("txid", request.TxId())
				}
				p := request.Output().FmtPhaseCounts()
				if p != nil {
					item.SetField("phaseCounts", p)
				}
				p = request.Output().FmtPhaseOperators()
				if p != nil {
					item.SetField("phaseOperators", p)
				}
				p = request.Output().FmtPhaseTimes()
				if p != nil {
					item.SetField("phaseTimes", p)
				}
				usedMemory := request.UsedMemory()
				if usedMemory != 0 {
					item.SetField("usedMemory", usedMemory)
				}

				if request.Prepared() != nil {
					p := request.Prepared()
					item.SetField("preparedName", p.Name())
					item.SetField("preparedText", p.Text())
				}
				prof := request.Profile()
				if prof == server.ProfUnset {
					prof = server.GetProfile()
				}
				credsString := datastore.CredsString(request.Credentials())
				if credsString != "" {
					item.SetField("users", credsString)
				}
				remoteAddr := request.RemoteAddr()
				if remoteAddr != "" {
					item.SetField("remoteAddr", remoteAddr)
				}
				userAgent := request.UserAgent()
				if userAgent != "" {
					item.SetField("userAgent", userAgent)
				}
				memoryQuota := request.MemoryQuota()
				if memoryQuota != 0 {
					item.SetField("memoryQuota", memoryQuota)
				}

				var ctrl bool
				ctr := request.Controls()
				if ctr == value.NONE {
					ctrl = server.GetControls()
				} else {
					ctrl = (ctr == value.TRUE)
				}
				if ctrl {
					na := request.NamedArgs()
					if na != nil {
						item.SetField("namedArgs", na)
					}
					pa := request.PositionalArgs()
					if pa != nil {
						item.SetField("positionalArgs", pa)
					}
				}

				meta := item.NewMeta()
				meta["keyspace"] = b.fullName

				t := request.GetTimings()
				if (prof == server.ProfOn || prof == server.ProfBench) && t != nil {
					bytes, _ := json.Marshal(t)
					meta["plan"] = bytes
					optEstimates := request.Output().FmtOptimizerEstimates(t)
					if optEstimates != nil {
						bytes, _ = json.Marshal(optEstimates)
						meta["optimizerEstimates"] = bytes
					}
				}

				item.SetId(key)
			})
			if err != nil {
				errs = append(errs, err)
			} else if item != nil {
				keysMap[key] = item
			}
		}
	}
	return
}

func (b *activeRequestsKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *activeRequestsKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *activeRequestsKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *activeRequestsKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	var done bool

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"active_requests", "DELETE", nil,
				func(warn errors.Error) {
					context.Warning(warn)
				},
				distributed.NO_CREDS, "")
			done = true

			// local entry
		} else {
			done = server.ActiveRequestsDelete(localKey)
		}

		// save memory allocations by making a new slice only on errors
		if !done {
			deleted := make([]value.Pair, i)
			if i > 0 {
				copy(deleted, deletes[0:i-1])
			}
			return deleted, errors.NewSystemStmtNotFoundError(nil, name)
		}
	}
	return deletes, nil
}

func newActiveRequestsKeyspace(p *namespace) (*activeRequestsKeyspace, errors.Error) {
	b := new(activeRequestsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_ACTIVE)

	primary := &activeRequestsIndex{
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
		nodes := &activeRequestsIndex{
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

type activeRequestsIndex struct {
	indexBase
	name     string
	keyspace *activeRequestsKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *activeRequestsIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *activeRequestsIndex) Id() string {
	return pi.Name()
}

func (pi *activeRequestsIndex) Name() string {
	return pi.name
}

func (pi *activeRequestsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *activeRequestsIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *activeRequestsIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *activeRequestsIndex) Condition() expression.Expression {
	return nil
}

func (pi *activeRequestsIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *activeRequestsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
}

func (pi *activeRequestsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *activeRequestsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *activeRequestsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
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
			if spanEvaluator.key() == whoAmI {
				server.ActiveRequestsForEach(func(id string, request server.Request) bool {
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
				distributed.RemoteAccess().GetRemoteKeys(nodes, "active_requests", func(id string) bool {
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
						server.ActiveRequestsForEach(func(id string, request server.Request) bool {
							entry = &datastore.IndexEntry{
								PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, id),
								EntryKey:   value.Values{value.NewValue(whoAmI)},
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
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "active_requests", func(id string) bool {
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

func (pi *activeRequestsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry
	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := distributed.RemoteAccess().WhoAmI()
	server.ActiveRequestsForEach(func(id string, request server.Request) bool {
		entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, id)}
		return true
	}, func() bool {
		return sendSystemKey(conn, entry)
	})

	distributed.RemoteAccess().GetRemoteKeys([]string{}, "active_requests", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
