//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
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

type requestLogKeyspace struct {
	keyspaceBase
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
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}
	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			context.Warning(warn)
		}
	}, creds, "")
	if userName == "" {
		return int64(server.RequestsCount() + count), nil
	} else {
		server.RequestsForeach(func(id string, request *server.RequestLogEntry) bool {
			if checkCompleted(request, userName) {
				count++
			}
			return true
		}, nil)
		return int64(count), nil
	}
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
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}
	formData := map[string]interface{}{"duration_style": context.DurationStyle().String()}

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey, "completed_requests", "POST",
				func(doc map[string]interface{}) {
					t, ok := doc["timings"]
					if ok {
						delete(doc, "timings")
					}
					o, ook := doc["optimizerEstimates"]
					if ook {
						delete(doc, "optimizerEstimates")
					}
					remoteValue := value.NewAnnotatedValue(doc)
					meta := remoteValue.NewMeta()
					meta["keyspace"] = b.fullName
					if ok {
						meta["plan"] = value.ApplyDurationStyleToValue(context.DurationStyle(), value.NewValue(t))
					}
					if ook {
						meta["optimizerEstimates"] = value.NewValue(o)
					}
					remoteValue.SetField("node", node)
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				creds, "", formData)
		} else {

			// local entry
			server.RequestDo(localKey, func(entry *server.RequestLogEntry) {
				if userName != "" && !checkCompleted(entry, userName) {
					return
				}
				item := value.NewAnnotatedValue(map[string]interface{}{
					"requestId":       localKey,
					"state":           entry.State,
					"elapsedTime":     context.FormatDuration(entry.ElapsedTime),
					"serviceTime":     context.FormatDuration(entry.ServiceTime),
					"resultCount":     entry.ResultCount,
					"resultSize":      entry.ResultSize,
					"errorCount":      entry.ErrorCount,
					"requestTime":     entry.Time.Format(expression.DEFAULT_FORMAT),
					"scanConsistency": entry.ScanConsistency,
					"n1qlFeatCtrl":    entry.FeatureControls,
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
				if entry.StatementType != "" {
					item.SetField("statementType", entry.StatementType)
				}
				if entry.QueryContext != "" {
					item.SetField("queryContext", entry.QueryContext)
				}
				if entry.UseFts {
					item.SetField("useFts", entry.UseFts)
				}
				if entry.UseCBO {
					item.SetField("useCBO", entry.UseCBO)
				}
				if entry.UseReplica == value.TRUE {
					item.SetField("useReplica", value.TristateToString(entry.UseReplica))
				}
				if entry.TxId != "" {
					item.SetField("txid", entry.TxId)
				}
				if entry.TransactionElapsedTime > 0 {
					item.SetField("transactionElapsedTime", context.FormatDuration(entry.TransactionElapsedTime))
				}
				if entry.TransactionRemainingTime > 0 {
					item.SetField("transactionRemainingTime", context.FormatDuration(entry.TransactionRemainingTime))
				}
				if entry.ThrottleTime > time.Duration(0) {
					item.SetField("throttleTime", context.FormatDuration(entry.ThrottleTime))
				}
				if entry.CpuTime > time.Duration(0) {
					item.SetField("cpuTime", context.FormatDuration(entry.CpuTime))
				}
				if entry.PreparedName != "" {
					item.SetField("preparedName", entry.PreparedName)
					item.SetField("preparedText", entry.PreparedText)
				}
				if entry.Mutations != 0 {
					item.SetField("mutations", entry.Mutations)
				}
				if entry.PhaseTimes != nil {
					// adjust durations to current format
					m := make(map[string]interface{}, len(entry.PhaseTimes))
					for k, v := range entry.PhaseTimes {
						if d, ok := v.(time.Duration); ok {
							m[k] = context.FormatDuration(d)
						} else {
							m[k] = v
						}
					}
					item.SetField("phaseTimes", m)
				}
				if entry.PhaseCounts != nil {
					item.SetField("phaseCounts", entry.PhaseCounts)
				}
				if entry.PhaseOperators != nil {
					item.SetField("phaseOperators", entry.PhaseOperators)
				}
				if entry.UsedMemory != 0 {
					item.SetField("usedMemory", entry.UsedMemory)
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
				if entry.MemoryQuota != 0 {
					item.SetField("memoryQuota", entry.MemoryQuota)
				}
				if entry.Errors != nil {
					errors := make([]value.Value, len(entry.Errors))
					for i, e := range entry.Errors {
						errors[i] = value.NewValue(e)
					}
					item.SetField("errors", errors)
				}
				if entry.Qualifier != "" {
					item.SetField("~qualifier", entry.Qualifier)
				}

				meta := item.NewMeta()
				meta["keyspace"] = b.fullName
				timings := entry.Timings()
				if timings != nil {
					meta["plan"] = value.ApplyDurationStyleToValue(context.DurationStyle(), value.NewValue(timings))
				}
				optEstimates := entry.OptEstimates()
				if optEstimates != nil {
					meta["optimizerEstimates"] = value.NewValue(optEstimates)
				}
				item.SetId(key)
				keysMap[key] = item
			})
		}
	}
	return
}

func (b *requestLogKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var err errors.Error
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey,
				"completed_requests", "DELETE", nil,
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				creds, "", nil)

			// local entry
		} else {
			err = server.RequestDelete(localKey, func(request *server.RequestLogEntry) bool {
				return userName == "" || checkCompleted(request, userName)
			})
		}

		if err != nil {
			errs := errors.Errors{err}
			if preserveMutations {
				// save memory allocations by making a new slice only on errors
				deleted := make([]value.Pair, i)
				if i > 0 {
					copy(deleted, deletes[0:i-1])
				}
				return i, deleted, errs
			} else {
				return i, nil, errs
			}

		}
	}

	if preserveMutations {
		return len(deletes), deletes, nil
	} else {
		return len(deletes), nil, nil
	}
}

func newRequestsKeyspace(p *namespace) (*requestLogKeyspace, errors.Error) {
	b := new(requestLogKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_REQUESTS)

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
		var creds distributed.Creds
		var process func(name string, request *server.RequestLogEntry) bool
		var send func() bool
		var doSend bool

		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}

		// now that the node name can change in flight, use a consistent one across the scan
		whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())
		userName := credsFromContext(conn.Context())
		if userName == "" {
			creds = distributed.NO_CREDS
			process = func(name string, request *server.RequestLogEntry) bool {
				entry = &datastore.IndexEntry{
					PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
					EntryKey:   value.Values{value.NewValue(whoAmI)},
				}
				return true
			}
			send = func() bool {
				return sendSystemKey(conn, entry)
			}
		} else {
			creds = distributed.Creds(userName)
			process = func(name string, request *server.RequestLogEntry) bool {
				doSend = checkCompleted(request, userName)
				if doSend {
					entry = &datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
						EntryKey:   value.Values{value.NewValue(whoAmI)},
					}
				}
				return true
			}
			send = func() bool {
				if doSend {
					return sendSystemKey(conn, entry)
				}
				return true
			}
		}
		idx := spanEvaluator.isEquals()
		if idx >= 0 {
			if spanEvaluator.key(idx) == whoAmI {
				server.RequestsForeach(process, send)
			} else {
				nodes := []string{decodeNodeName(spanEvaluator.key(idx))}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "completed_requests", func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						conn.Warning(warn)
					}
				}, creds, "")
			}
		} else {
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				encodedNode := encodeNodeName(node)
				if spanEvaluator.evaluate(encodedNode) {
					if encodedNode == whoAmI {
						server.RequestsForeach(process, send)
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
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						conn.Warning(warn)
					}
				}, creds, "")
			}
		}
	}
}

func (pi *requestLogIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry
	var creds distributed.Creds
	var process func(name string, request *server.RequestLogEntry) bool
	var send func() bool
	var doSend bool

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())

	userName := credsFromContext(conn.Context())
	if userName == "" {
		creds = distributed.NO_CREDS
		process = func(name string, request *server.RequestLogEntry) bool {
			entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
			return true
		}
		send = func() bool {
			return sendSystemKey(conn, entry)
		}
	} else {
		creds = distributed.Creds(userName)
		process = func(name string, request *server.RequestLogEntry) bool {
			doSend = checkCompleted(request, userName)
			if doSend {
				entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
			}
			return true
		}
		send = func() bool {
			if doSend {
				return sendSystemKey(conn, entry)
			}
			return true
		}
	}
	server.RequestsForeach(process, send)
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			conn.Warning(warn)
		}
	}, creds, "")
}

func checkCompleted(request *server.RequestLogEntry, userName string) bool {
	return userName == request.Users
}
