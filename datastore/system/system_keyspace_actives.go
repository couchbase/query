//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"strings"
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
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}
	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "active_requests", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			context.Warning(warn)
		}
	}, creds, "")
	if userName == "" {
		c, err := server.ActiveRequestsCount()
		return int64(c + count), err
	} else {
		server.ActiveRequestsForEach(func(name string, request server.Request) bool {
			if checkRequest(request, userName) {
				count++
			}
			return true
		}, nil)
		return int64(count), nil
	}
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
			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey, "active_requests", "POST",
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
					remoteValue.SetField("node", node)
					meta := remoteValue.NewMeta()
					meta["keyspace"] = b.fullName
					if ok {
						meta["plan"] = value.ApplyDurationStyleToValue(context.DurationStyle(), func(s string) bool {
							return strings.HasSuffix(s, "Time")
						}, value.NewValue(t))
					}
					if ook {
						meta["optimizerEstimates"] = value.NewValue(o)
					}
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
			var item value.AnnotatedValue

			// local entry
			err := server.ActiveRequestsGet(localKey, func(request server.Request) {
				if userName != "" && !checkRequest(request, userName) {
					return
				}

				et := context.FormatDuration(0)
				if !request.ServiceTime().IsZero() {
					et = context.FormatDuration(time.Since(request.ServiceTime()))
				}

				item = value.NewAnnotatedValue(map[string]interface{}{
					"requestId":       localKey,
					"requestTime":     request.RequestTime().Format(expression.DEFAULT_FORMAT),
					"elapsedTime":     context.FormatDuration(time.Since(request.RequestTime())),
					"executionTime":   et,
					"state":           request.State().StateName(),
					"scanConsistency": request.ScanConsistency(),
					"n1qlFeatCtrl":    request.FeatureControls(),
				})
				if node != "" {
					item.SetField("node", node)
				}
				cId := request.ClientID().String()
				if cId != "" {
					item.SetField("clientContextID", cId)
				}
				if request.Statement() != "" {
					item.SetField("statement", request.RedactedStatement())
				}
				if request.Type() != "" {
					item.SetField("statementType", request.Type())
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
				if request.UseReplica() == value.TRUE {
					item.SetField("useReplica", value.TristateToString(request.UseReplica()))
				}
				if request.TxId() != "" {
					item.SetField("txid", request.TxId())
				}
				if !request.TransactionStartTime().IsZero() {
					item.SetField("transactionElapsedTime", context.FormatDuration(time.Since(request.TransactionStartTime())))
					remTime := request.TxTimeout() - time.Since(request.TransactionStartTime())
					if remTime > 0 {
						item.SetField("transactionRemainingTime", context.FormatDuration(remTime))
					}
				}
				if request.ThrottleTime() > time.Duration(0) {
					item.SetField("throttleTime", context.FormatDuration(request.ThrottleTime()))
				}
				if request.CpuTime() > time.Duration(0) {
					item.SetField("cpuTime", context.FormatDuration(request.CpuTime()))
				}
				p := request.Output().FmtPhaseCounts()
				if p != nil {
					item.SetField("phaseCounts", p)
				}
				p = request.Output().FmtPhaseOperators()
				if p != nil {
					item.SetField("phaseOperators", p)
				}
				p = request.Output().FmtPhaseTimes(context.DurationStyle())
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
					na := request.RedactedNamedArgs()
					if na != nil {
						item.SetField("namedArgs", na)
					}
					pa := request.RedactedPositionalArgs()
					if pa != nil {
						item.SetField("positionalArgs", pa)
					}
				}

				meta := item.NewMeta()
				meta["keyspace"] = b.fullName

				if !request.Sensitive() {
					timings := request.GetTimings()
					if timings != nil {
						meta["plan"] = value.ApplyDurationStyleToValue(context.DurationStyle(), func(s string) bool {
							return strings.HasSuffix(s, "Time")
						}, value.NewMarshalledValue(timings))
						optEstimates := request.Output().FmtOptimizerEstimates(timings)
						if optEstimates != nil {
							meta["optimizerEstimates"] = value.NewMarshalledValue(optEstimates)
						}
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

func (b *activeRequestsKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (int, value.Pairs, errors.Errors) {
	var done bool
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
				"active_requests", "DELETE", nil,
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				creds, "", nil)
			done = true

			// local entry
		} else {
			done = server.ActiveRequestsDeleteFunc(localKey, func(request server.Request) bool {
				return userName == "" || checkRequest(request, userName)
			})
		}

		if !done {
			err := errors.Errors{errors.NewSystemStmtNotFoundError(nil, name)}
			if preserveMutations {
				// save memory allocations by making a new slice only on errors
				deleted := make([]value.Pair, i)
				if i > 0 {
					copy(deleted, deletes[0:i-1])
				}
				return i, deleted, err
			} else {
				return i, nil, err
			}

		}
	}

	if preserveMutations {
		return len(deletes), deletes, nil
	} else {
		return len(deletes), nil, nil
	}
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
		var creds distributed.Creds
		var process func(id string, request server.Request) bool
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
			process = func(name string, request server.Request) bool {
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
			process = func(name string, request server.Request) bool {
				doSend = checkRequest(request, userName)
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
				server.ActiveRequestsForEach(process, send)
			} else {
				nodes := []string{decodeNodeName(spanEvaluator.key(idx))}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "active_requests", func(id string) bool {
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
						server.ActiveRequestsForEach(process, send)
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
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						conn.Warning(warn)
					}
				}, creds, "")
			}
		}
	}
}

func (pi *activeRequestsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry
	var creds distributed.Creds
	var process func(id string, request server.Request) bool
	var send func() bool
	var doSend bool

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())

	userName := credsFromContext(conn.Context())
	if userName == "" {
		creds = distributed.NO_CREDS
		process = func(name string, request server.Request) bool {
			entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
			return true
		}
		send = func() bool {
			return sendSystemKey(conn, entry)
		}
	} else {
		creds = distributed.Creds(userName)
		process = func(name string, request server.Request) bool {
			doSend = checkRequest(request, userName)
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
	server.ActiveRequestsForEach(process, send)

	distributed.RemoteAccess().GetRemoteKeys([]string{}, "active_requests", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			conn.Warning(warn)
		}
	}, creds, "")
}

func checkRequest(request server.Request, userName string) bool {
	users := datastore.CredsArray(request.Credentials())
	return len(users) > 0 && userName == users[0]
}
